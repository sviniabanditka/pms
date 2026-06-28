package filter

import (
	"encoding/json"
	"testing"
)

func pred(field, op string, val any) Node {
	raw, _ := json.Marshal(val)
	if val == nil {
		raw = nil
	}
	return Node{Type: "predicate", Field: field, Op: op, Value: raw}
}

func TestCompile(t *testing.T) {
	cases := []struct {
		name    string
		node    *Node
		wantSQL string
		nargs   int
	}{
		{"nil → no filter", nil, "", 0},
		{"eq", &Node{Type: "predicate", Field: "status", Op: "eq", Value: json.RawMessage(`"s1"`)}, "state_id::text IS NOT DISTINCT FROM ?", 1},
		{"neq null-safe", pred("assignee", "neq", "u1").ptr(), "assignee_id::text IS DISTINCT FROM ?", 1},
		{"in", pred("priority", "in", []int{1, 2}).ptr(), "priority::text IN (?)", 1},
		{"in empty → FALSE", pred("priority", "in", []int{}).ptr(), "FALSE", 0},
		{"nin empty → TRUE", pred("priority", "nin", []int{}).ptr(), "TRUE", 0},
		{"nin null-safe", pred("cycle", "nin", []string{"c1"}).ptr(), "(cycle_id IS NULL OR cycle_id::text NOT IN (?))", 1},
		{"contains text", pred("title", "contains", "bug").ptr(), "title::text ILIKE ?", 1},
		{"isSet text", pred("title", "isSet", nil).ptr(), "(title IS NOT NULL AND title <> '')", 0},
		{"isNotSet ref", pred("assignee", "isNotSet", nil).ptr(), "assignee_id IS NULL", 0},
		{"and", &Node{Type: "and", Children: []Node{pred("status", "eq", "s1"), pred("priority", "eq", 2)}},
			"(state_id::text IS NOT DISTINCT FROM ? AND priority::text IS NOT DISTINCT FROM ?)", 2},
		{"or empty → TRUE", &Node{Type: "or", Children: nil}, "TRUE", 0},
		{"not", &Node{Type: "not", Child: pred("status", "eq", "s1").ptr()}, "NOT (state_id::text IS NOT DISTINCT FROM ?)", 1},
	}
	for _, c := range cases {
		sql, args, err := Compile(c.node)
		if err != nil {
			t.Fatalf("%s: err %v", c.name, err)
		}
		if sql != c.wantSQL {
			t.Errorf("%s: sql = %q want %q", c.name, sql, c.wantSQL)
		}
		if len(args) != c.nargs {
			t.Errorf("%s: nargs = %d want %d", c.name, len(args), c.nargs)
		}
	}
}

func TestCompileUnknownField(t *testing.T) {
	n := pred("bogus", "eq", "x")
	if _, _, err := Compile(&n); err == nil {
		t.Fatal("expected error on unknown field")
	}
}

func (n Node) ptr() *Node { return &n }
