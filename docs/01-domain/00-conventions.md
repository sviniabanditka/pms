# Доменные конвенции — справочник таблиц

Практическая шпаргалка. Обоснование — [ADR-0005](../decisions/ADR-0005-data-conventions.md).
Каждая сущность в этом разделе наследует приведённое ниже и описывает **только свои** поля.

## Стандартные колонки

Каждая доменная таблица (если не оговорено иное) содержит:

| Колонка | Тип | Назначение |
|---|---|---|
| `id` | `uuid` (v7) PK | внутренний идентификатор, k-sortable |
| `workspace_id` | `uuid` NOT NULL FK | тенант; скоуп всех запросов + RLS (искл.: `users`) |
| `created_at` | `timestamptz` | UTC |
| `updated_at` | `timestamptz` | UTC, бампается при мутации |
| `version` | `bigint` NOT NULL DEFAULT 0 | синк/LWW; ++ при каждой мутации |
| `archived_at` | `timestamptz` NULL | обратимый архив |
| `deleted_at` | `timestamptz` NULL | soft-delete (окно Undo → purge) |

Активная выборка по умолчанию: `archived_at IS NULL AND deleted_at IS NULL` (view может запросить иное).

## Глобальные таблицы (без `workspace_id`)

- `users` — учётка живёт в нескольких воркспейсах; принадлежность через `workspace_members`.

## Синк-инфраструктура

```
change_log (append-only)
  seq          bigserial PK        -- монотонный курсор догона
  workspace_id uuid NOT NULL
  entity_type  text NOT NULL       -- 'issue' | 'view' | ...
  entity_id    uuid NOT NULL
  version      bigint NOT NULL      -- новая версия сущности
  op           text NOT NULL        -- 'create' | 'update' | 'delete' | 'archive'
  changed_fields jsonb              -- дельта полей (для update)
  actor_id     uuid
  at           timestamptz NOT NULL
```

Клиент догоняет запросом `seq > cursor` в рамках своих `workspace_id`/подписанных каналов.

## Соглашения именования

- Таблицы — `snake_case`, множественное число (`issues`, `workflow_states`).
- FK — `<entity>_id` (`team_id`, `parent_id`, `assignee_id`).
- m2m — таблица-связка `<a>_<b>` (`issue_labels`).
- Перечислимые категории — `text` + CHECK или отдельная enum-таблица (решаем по месту; статусы — данные, не enum).

## Порядок (fractional indexing)

- `position text` — ключ между соседями (`keyBetween`). Реордер не трогает соседей.
- Контекст уникальности порядка указывается у каждой сущности (board-колонка, backlog, `parent_id`).

## JSONB

- Rich text → `jsonb` (TipTap JSON). Custom field values → `jsonb` (валидируется схемой поля).
- Индексация JSONB под фильтры — GIN по нужным путям; на масштабе — проекция в OpenSearch (§3.3).

## Кластеры сущностей (план раздела)

1. `identity-access` — User, Workspace, WorkspaceMember
2. `teams-config` — Team (вложенность), TeamMember, WorkflowState, Label
3. `issues` — Issue, sub-issues, IssueRelation, Comment, Attachment, CustomField/Value, Subscription, Reaction
4. `planning` — Project, Milestone, Cycle
5. `views-dashboards` — View, Dashboard, DashboardWidget, Favorite
6. `content-collab` — Document, Activity
7. `platform` — Notification, NotificationPreference

Файлы создаются по мере проработки кластера.
