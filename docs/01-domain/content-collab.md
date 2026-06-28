# Домен: Content & Collaboration

**Document · Activity**. Наследует [конвенции](00-conventions.md).
Связано: §3.2, §8.1.8. Comment/Attachment/Reaction — в [issues](issues.md) (полиморфны).

---

## Document

Rich-text документ, дерево вложенности; привязан к team или project (или workspace-глобальный).

| Поле | Тип | Заметки |
|---|---|---|
| `id` | uuid v7 PK | |
| `workspace_id` | uuid FK NOT NULL | |
| `parent_id` | uuid FK→documents NULL | дерево доков |
| `scope_kind` | text NOT NULL | `workspace \| team \| project` |
| `team_id` | uuid FK NULL | |
| `project_id` | uuid FK NULL | |
| `title` | text NOT NULL | |
| `body` | jsonb | rich text (TipTap); **LWW по документу в MVP** (ADR-0004 #9) |
| `icon` | text NULL | emoji/иконка |
| `position` | text | порядок среди siblings (fractional) |
| `created_by` | uuid FK→users | |
| стандартные | | `version/archived_at/deleted_at` |

**Инварианты**
- Дерево без циклов; перемещение меняет `parent_id` (+ `position`).
- Видимость наследует scope (team-док — по правам команды; §7.7).
- MVP — одиночное редактирование: конкурентная правка = LWW по `body`/`version`. CRDT (Yjs) — отдельная веха (ADR-0004).
- Встраивание дока как виджета/в issue — через ссылку (`DashboardWidget.type=doc` в v2; в MVP — ссылка-attachment).
- Комментарии к доку — полиморфный Comment (`entity_type=doc`).

---

## Activity (аудит-лог + ленты)

Единый аудит и источник лент активности (team Activities, issue history) и part нотификаций.

| Поле | Тип | Заметки |
|---|---|---|
| `id` | uuid v7 PK | |
| `workspace_id` | uuid FK NOT NULL | |
| `actor_id` | uuid FK→users NULL | NULL = система |
| `entity_type` | text NOT NULL | `issue \| project \| cycle \| doc \| team \| ...` |
| `entity_id` | uuid NOT NULL | |
| `action` | text NOT NULL | `created \| updated \| state_changed \| commented \| assigned \| ...` |
| `diff` | jsonb NULL | изменённые поля (before/after) для update |
| `context` | jsonb NULL | напр. team_id для скоупинга ленты |
| `at` | timestamptz NOT NULL | |

**Инварианты**
- Append-only; не редактируется/не удаляется (кроме GDPR-анонимизации actor).
- Пишется в той же транзакции, что мутация (консистентность с `change_log`; частично пересекается — см. ниже).
- Ленты: team Activities = фильтр по `context.team_id`; issue history = по `(entity_type=issue, entity_id)`.
- Источник для части нотификаций (state_changed/commented/assigned).

**Activity vs change_log:** `change_log` — техническая дельта для синка (что/версия), `Activity` —
человекочитаемое событие для UI/аудита (кто/что/diff). Разные потребители; не сливаем (один машинный, другой продуктовый).

---

## Связи кластера

```
documents   1──* documents (parent_id tree); *──1 teams/projects (scope)
documents   1──* comments (polymorphic, entity_type=doc)
activities  *──1 users(actor) ── polymorphic entity; скоуп по context.team_id
```

## Открытые микро-вопросы

- Версии/история дока (snapshots) — после MVP (Activity фиксирует факт правки).
- Ретеншн Activity (архивация старого) — políтика в `02-architecture` (не блокирует MVP).
