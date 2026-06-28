# Домен: Teams & Config

Команды (с вложенностью) и их конфигурация: **Team · TeamMember · WorkflowState · Label**.
Наследует [конвенции](00-conventions.md). Связано: §3.2, §3.3, §8.1.1, ADR-0004 (#2).

---

## Team (вложенная орг-единица)

Владеет issues/cycles/views/docs; ключ для идентификаторов `KEY-N`.

| Поле | Тип | Заметки |
|---|---|---|
| `id` | uuid v7 PK | |
| `workspace_id` | uuid FK NOT NULL | |
| `parent_id` | uuid FK→teams NULL | вложенность; NULL = корневая команда |
| `path` | ltree | материализованный путь для subtree/ancestor-запросов (rollup) |
| `key` | text NOT NULL | напр. `BLD`; используется в `KEY-N` |
| `name` | text NOT NULL | |
| `private` | boolean NOT NULL DEFAULT false | приватность (§7.7) |
| `next_issue_number` | bigint NOT NULL DEFAULT 0 | атомарный счётчик нумерации |
| `enabled_features` | jsonb | per-team фичи (§8.1.1) |
| `cycle_config` | jsonb | длительность/старт/cooldown/авто-старт циклов (§8.1.1) |
| `settings` | jsonb | переопределения display-дефолтов воркспейса |
| стандартные | | `version/archived_at/deleted_at` |

**Инварианты**
- `key` UNIQUE в пределах `(workspace_id)`; иммутабелен после создания (issues уже несут его в идентификаторе).
- `path` поддерживается триггером/приложением при смене `parent_id`; цикл вложенности запрещён.
- Нумерация: создание issue делает `UPDATE teams SET next_issue_number = next_issue_number + 1 ... RETURNING` в той же транзакции → row-lock сериализует конкурентные создания, коллизий нет (§10.3). Дыры допустимы (rollback транзакции «съедает» номер — это ок, ADR-0004 #10).
- Видимость: `private` команда не видна не-членам; наследование членства для public-дочерних — по `workspace.settings.teamNestingPolicy` (§7.7).
- Rollup issues дочерних команд — через `path` + флаг View `rollupDescendants` (ADR-0004 #2).

**Почему ltree, а не только recursive CTE:** rollup и проверка видимости бьют по поддереву часто;
`ltree` + GiST-индекс дают subtree/ancestor за один индексный скан. `parent_id` остаётся источником
истины, `path` — производное (поддерживается при move). Альтернатива — closure table (тяжелее на запись).

---

## TeamMember

Членство и роль в команде.

| Поле | Тип | Заметки |
|---|---|---|
| `id` | uuid v7 PK | |
| `workspace_id` | uuid FK NOT NULL | |
| `team_id` | uuid FK NOT NULL | |
| `user_id` | uuid FK→users NOT NULL | |
| `role` | text NOT NULL | `lead \| member` |
| `source` | text NOT NULL | `direct \| inherited` (унаследовано от родителя для public) |
| стандартные | | `version` |

**Инварианты**
- UNIQUE `(team_id, user_id)`.
- `inherited` членства не хранятся материализованно для всех — вычисляются по `path` + политике; материализуем только при необходимости перфоманса (открытый вопрос ниже).

---

## WorkflowState (статус по категории, per-team)

| Поле | Тип | Заметки |
|---|---|---|
| `id` | uuid v7 PK | |
| `workspace_id` | uuid FK NOT NULL | |
| `team_id` | uuid FK NOT NULL | |
| `category` | text NOT NULL | `backlog \| unstarted \| started \| completed \| canceled` |
| `name` | text NOT NULL | произвольное имя статуса внутри категории |
| `color_token` | text NOT NULL | **ссылка на токен**, не hex (§5.2) |
| `position` | text | порядок отображения (fractional) |
| стандартные | | `version` |

**Инварианты**
- `category` управляет авто-простановкой на issue: переход в `started`-категорию ставит `started_at`; в `completed` — `completed_at`; уход обратно — сбрасывает (логика в домене Issues).
- У команды дефолтный набор статусов создаётся при создании команды (по одному+ на категорию).
- Нельзя удалить статус, если на нём висят issues без замены (reassign-on-delete).

---

## Label

Метка классификации; scope workspace или team; поддерживает группы.

| Поле | Тип | Заметки |
|---|---|---|
| `id` | uuid v7 PK | |
| `workspace_id` | uuid FK NOT NULL | |
| `team_id` | uuid FK NULL | NULL = workspace-scope; иначе team-scope |
| `group_id` | uuid FK→labels NULL | группа лейблов (родитель); группа = label с `is_group` |
| `is_group` | boolean DEFAULT false | |
| `name` | text NOT NULL | |
| `color_token` | text NOT NULL | ссылка на токен |
| стандартные | | `version` |

**Инварианты**
- Имя уникально в своём scope/группе.
- Группа лейблов может вести себя как «однозначный выбор» (mutually exclusive) — флаг на группе (открытый вопрос: нужно ли в MVP).
- Связь с issue — m2m `issue_labels` (см. issues).

---

## Связи кластера

```
workspaces 1──* teams (self-nesting via parent_id/path)
teams      1──* team_members *──1 users
teams      1──* workflow_states
workspaces 1──* labels (team_id NULL=ws-scope)  ── group_id self-FK
```

## Открытые микро-вопросы

- Материализовать ли inherited-членства (перф vs свежесть) — решаем в `02-architecture/rbac.md`.
- Mutually-exclusive группы лейблов в MVP — да/нет.
- `ltree` vs closure table окончательно — подтвердить при реализации M0 (по умолчанию ltree).
