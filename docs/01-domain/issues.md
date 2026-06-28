# Домен: Issues

Ядро рабочих элементов: **Issue · IssueRelation · Comment · Attachment ·
CustomField/CustomFieldValue · Subscription · Reaction · IssueLabel**.
Наследует [конвенции](00-conventions.md). Связано: §3.2, §3.3, §8.1.2.

---

## Issue

Основная единица работы; человекочитаемый `KEY-N`; иерархия sub-issues.

| Поле | Тип | Заметки |
|---|---|---|
| `id` | uuid v7 PK | client-generated для оптимистичных мутаций |
| `workspace_id` | uuid FK NOT NULL | |
| `team_id` | uuid FK NOT NULL | владелец (ровно один, ADR-0004 #2) |
| `number` | bigint NOT NULL | из `teams.next_issue_number` |
| `identifier` | text GENERATED | `team.key || '-' || number` (или собираем в приложении) |
| `parent_id` | uuid FK→issues NULL | sub-issue иерархия |
| `title` | text NOT NULL | |
| `description` | jsonb | rich text (TipTap); LWW по полю в MVP (ADR-0004 #9) |
| `state_id` | uuid FK→workflow_states NOT NULL | статус |
| `priority` | smallint NOT NULL DEFAULT 0 | `0 none,1 low,2 medium,3 high,4 urgent` |
| `assignee_id` | uuid FK→users NULL | |
| `estimate` | numeric NULL | трактуется по `estimateScale` воркспейса |
| `project_id` | uuid FK→projects NULL | |
| `cycle_id` | uuid FK→cycles NULL | |
| `due_date` | date NULL | |
| `start_date` | date NULL | |
| `scheduled_date` | date NULL | для календаря (§8.1.6) |
| `started_at` | timestamptz NULL | авто по категории статуса |
| `completed_at` | timestamptz NULL | авто по категории статуса |
| `board_position` | text | порядок в колонке board (fractional, контекст = группа) |
| `sub_position` | text | порядок среди siblings под `parent_id` |
| `created_by` | uuid FK→users | |
| стандартные | | `version/archived_at/deleted_at` |

**Инварианты**
- `(team_id, number)` UNIQUE; `number` берётся атомарно (teams-config).
- Смена `state_id`: если новая категория `started` и `started_at IS NULL` → ставим; `completed`/`canceled` → `completed_at`; возврат в более раннюю категорию → сброс соответствующих меток. Чистая функция перехода — кандидат на TDD (§10.1).
- `parent_id` — без циклов; sub-issue в той же команде, что родитель (упрощение MVP; cross-team sub-issues — нет).
- Прогресс родителя = агрегат по детям (completed/total) — вычисляемо, не хранимо.
- archive/soft-delete каскадит на sub-issues, comments, attachments пометкой (ADR-0005).
- `scheduled_date` vs `due_date` для календаря: renderer кладёт issue по `scheduled_date`, fallback `due_date` (конфиг View).

**Метки (m2m): `issue_labels`** — `(issue_id, label_id)`, `workspace_id`. Без своих стандартных колонок (чистая связка), но в `change_log` отражается как изменение issue.

---

## IssueRelation

Типизированные связи между issues.

| Поле | Тип | Заметки |
|---|---|---|
| `id` | uuid v7 PK | |
| `workspace_id` | uuid FK NOT NULL | |
| `source_id` | uuid FK→issues NOT NULL | |
| `target_id` | uuid FK→issues NOT NULL | |
| `type` | text NOT NULL | `blocks \| blocked_by \| related \| duplicate \| duplicate_of` |
| стандартные | | `version` |

**Инварианты**
- **Зеркала**: `blocks`↔`blocked_by`, `duplicate`↔`duplicate_of`, `related` симметричен. Храним одну строку, зеркало выводим — ИЛИ храним обе и синхронизируем. **Решение MVP:** храним каноническую пару (одна запись), зеркальное направление выводится в чтении. Зеркальная консистентность — кандидат на TDD (§10.1).
- UNIQUE `(source_id, target_id, type)`; запрет self-relation.
- Relations между разными командами разрешены (в отличие от sub-issues).

---

## Comment (полиморфный)

Комментарии с тредами и реакциями на issue/doc/project.

| Поле | Тип | Заметки |
|---|---|---|
| `id` | uuid v7 PK | |
| `workspace_id` | uuid FK NOT NULL | |
| `entity_type` | text NOT NULL | `issue \| doc \| project` |
| `entity_id` | uuid NOT NULL | полиморфная ссылка |
| `parent_id` | uuid FK→comments NULL | тред (одноуровневый в MVP) |
| `author_id` | uuid FK→users NOT NULL | |
| `body` | jsonb | rich text + mentions |
| `mentions` | uuid[] | денорм. для нотификаций |
| стандартные | | `version/deleted_at` (edit history — позже) |

**Инварианты**
- `(entity_type, entity_id)` индексируется; полиморфизм без FK — целостность на уровне приложения.
- Mention → нотификация (Notifications-модуль).
- Треды одноуровневые в MVP (reply на reply сплющивается).

---

## Attachment (полиморфный)

Файлы и внешние ссылки (PR/Figma) с метаданными.

| Поле | Тип | Заметки |
|---|---|---|
| `id` | uuid v7 PK | |
| `workspace_id` | uuid FK NOT NULL | |
| `entity_type` | text NOT NULL | `issue \| doc \| comment` |
| `entity_id` | uuid NOT NULL | |
| `kind` | text NOT NULL | `file \| link` |
| `url` | text NOT NULL | S3-ключ или внешний URL |
| `title` | text | |
| `meta` | jsonb | size/mime/preview; для link — тип (PR/Figma), live-статус (v3) |
| `uploaded_by` | uuid FK→users | |
| стандартные | | `version/deleted_at` |

---

## CustomField / CustomFieldValue

Пользовательские поля и значения (ADR-0004 #3: база + relationship).

**CustomField** (определение):

| Поле | Тип | Заметки |
|---|---|---|
| `id` | uuid v7 PK | |
| `workspace_id` | uuid FK NOT NULL | |
| `team_id` | uuid FK NULL | NULL = workspace-scope |
| `entity_type` | text NOT NULL | `issue \| project` (MVP) |
| `key` | text NOT NULL | стабильный ключ для Filter AST (`{kind:'custom',fieldId}`) |
| `name` | text NOT NULL | |
| `type` | text NOT NULL | `text\|number\|date\|checkbox\|select\|multiselect\|url\|people\|relationship` |
| `config` | jsonb | опции select, target-тип relationship и т.п. |
| `position` | text | порядок в таблице/панели |
| стандартные | | `version` |

**CustomFieldValue** (значение):

| Поле | Тип | Заметки |
|---|---|---|
| `id` | uuid v7 PK | |
| `workspace_id` | uuid FK NOT NULL | |
| `field_id` | uuid FK→custom_fields NOT NULL | |
| `issue_id` | uuid FK→issues NOT NULL | (или project_id — полиморфно по entity_type поля) |
| `value` | jsonb | типизировано схемой `field.type` |
| стандартные | | `version` |

**Инварианты**
- UNIQUE `(field_id, issue_id)`.
- `value` валидируется по `field.type`/`config` доменом.
- Фильтрация: GIN-индекс по `value`; на масштабе — проекция в OpenSearch (§3.3).
- `formula`/`rollup` НЕ поддерживаются в MVP (ADR-0004); `relationship` → предикаты eq/in/isSet по target-id, без вычислений.

---

## Subscription / Reaction

| Сущность | Поля | Заметки |
|---|---|---|
| **Subscription** | `id, workspace_id, issue_id, user_id` | подписчики issue; авто-подписка на assign/comment; источник нотификаций |
| **Reaction** | `id, workspace_id, entity_type(issue\|comment), entity_id, user_id, emoji` | UNIQUE `(entity,user,emoji)` |

---

## Связи кластера

```
issues *──1 teams / workflow_states / projects(opt) / cycles(opt) / users(assignee)
issues 1──* issues (parent_id, sub-issues)
issues *──* labels (issue_labels)
issues 1──* issue_relations (source/target, mirrored)
issues 1──* comments / attachments / custom_field_values / subscriptions / reactions
```

## Открытые микро-вопросы

- `identifier` — GENERATED колонка vs сборка в приложении (зависит от иммутабельности `team.key` — она иммутабельна, значит GENERATED безопасно).
- Многоуровневые треды комментариев — после MVP.
- Edit-history комментариев/issue — отдельно от Activity? (склоняюсь: Activity покрывает).
