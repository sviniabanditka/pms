// Package filter — серверный компилятор Filter-AST → SQL WHERE для issues (D21).
// Зеркало клиентского match() (frontend/lib/filter.ts): те же поля/операторы, та же
// семантика null (=== ↔ IS NOT DISTINCT FROM). Под текущий масштаб — bun/Postgres ILIKE;
// полнотекст/контент-индекс (OpenSearch) — отдельная веха.
package filter

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/uptrace/bun"
)

// Node — сериализуемое булево дерево (совпадает с JSON-формой клиента: views.filter).
type Node struct {
	Type     string          `json:"type"` // and | or | not | predicate
	Children []Node          `json:"children,omitempty"`
	Child    *Node           `json:"child,omitempty"`
	Field    string          `json:"field,omitempty"`
	Op       string          `json:"op,omitempty"`
	Value    json.RawMessage `json:"value,omitempty"`
}

// builtin поле → SQL-колонка issues. Паритет с FIELD клиента (+ безопасные расширения).
var columns = map[string]string{
	"status":    "state_id",
	"priority":  "priority",
	"assignee":  "assignee_id",
	"title":     "title",
	"project":   "project_id",
	"cycle":     "cycle_id",
	"milestone": "milestone_id",
	"parent":    "parent_id",
	"creator":   "created_by",
}

var textFields = map[string]bool{"title": true}

// Compile превращает дерево в (whereSQL, args) для bun. Пустой/nil узел → ("", nil).
func Compile(n *Node) (string, []any, error) {
	if n == nil {
		return "", nil, nil
	}
	return compile(n)
}

func compile(n *Node) (string, []any, error) {
	switch n.Type {
	case "and":
		return boolGroup(n.Children, "AND", "TRUE")
	case "or":
		// клиент: or с 0 детей → true
		return boolGroup(n.Children, "OR", "TRUE")
	case "not":
		if n.Child == nil {
			return "TRUE", nil, nil
		}
		sql, args, err := compile(n.Child)
		if err != nil {
			return "", nil, err
		}
		return "NOT (" + sql + ")", args, nil
	case "predicate":
		return predicate(n)
	default:
		return "TRUE", nil, nil
	}
}

func boolGroup(children []Node, sep, empty string) (string, []any, error) {
	if len(children) == 0 {
		return empty, nil, nil
	}
	parts := make([]string, 0, len(children))
	var args []any
	for i := range children {
		sql, a, err := compile(&children[i])
		if err != nil {
			return "", nil, err
		}
		parts = append(parts, sql)
		args = append(args, a...)
	}
	return "(" + strings.Join(parts, " "+sep+" ") + ")", args, nil
}

func predicate(n *Node) (string, []any, error) {
	col, ok := columns[n.Field]
	if !ok {
		return "", nil, fmt.Errorf("filter: unknown field %q", n.Field)
	}
	isText := textFields[n.Field]
	switch n.Op {
	case "isSet":
		if isText {
			return "(" + col + " IS NOT NULL AND " + col + " <> '')", nil, nil
		}
		return col + " IS NOT NULL", nil, nil
	case "isNotSet":
		if isText {
			return "(" + col + " IS NULL OR " + col + " = '')", nil, nil
		}
		return col + " IS NULL", nil, nil
	case "eq":
		v := scalar(n.Value)
		return col + "::text IS NOT DISTINCT FROM ?", []any{v}, nil
	case "neq":
		v := scalar(n.Value)
		return col + "::text IS DISTINCT FROM ?", []any{v}, nil
	case "contains":
		// экранируем ILIKE-метасимволы (% _ \), чтобы поиск 'test_case' не матчил 'testXcase'
		v := ilikeEscape(toStr(scalar(n.Value)))
		return col + "::text ILIKE ? ESCAPE '\\'", []any{"%" + v + "%"}, nil
	case "in":
		vals := list(n.Value)
		if len(vals) == 0 {
			return "FALSE", nil, nil // клиент: [].includes(v) === false
		}
		return col + "::text IN (?)", []any{bun.In(vals)}, nil
	case "nin":
		vals := list(n.Value)
		if len(vals) == 0 {
			return "TRUE", nil, nil // клиент: ![].includes(v) === true
		}
		return "(" + col + " IS NULL OR " + col + "::text NOT IN (?))", []any{bun.In(vals)}, nil
	default:
		return "TRUE", nil, nil
	}
}

// ilikeEscape экранирует спецсимволы ILIKE (\ % _) для буквального поиска подстроки.
var ilikeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

func ilikeEscape(s string) string { return ilikeEscaper.Replace(s) }

// scalar декодирует JSON-значение в строку (или nil для null) — колонки сравниваем как text.
func scalar(raw json.RawMessage) any {
	if len(raw) == 0 {
		return nil
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil
	}
	if v == nil {
		return nil
	}
	return toStr(v)
}

func toStr(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	case bool:
		if t {
			return "true"
		}
		return "false"
	default:
		b, _ := json.Marshal(t)
		return string(b)
	}
}

func list(raw json.RawMessage) []string {
	if len(raw) == 0 {
		return nil
	}
	var arr []any
	if err := json.Unmarshal(raw, &arr); err != nil {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, x := range arr {
		if x == nil {
			continue
		}
		out = append(out, toStr(x))
	}
	return out
}
