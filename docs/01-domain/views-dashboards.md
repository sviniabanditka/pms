# Домен: Views & Dashboards

Сердце реюзабельности: **View · Dashboard · DashboardWidget · Favorite**.
Наследует [конвенции](00-conventions.md). Связано: §6 целиком, §8.1.5, §8.1.7.

> Семантика исполнения (Filter AST, query-движок, renderer-контракт) — в
> `02-architecture/query-engine.md` и `03-frontend/view-renderer.md`. Здесь — хранение.

---

## View (сохранённое представление)

Хранит `ViewDefinition` (§6.1). «Что и как показать»: источник + Filter AST + группировка/сортировка + layout.

| Поле | Тип | Заметки |
|---|---|---|
| `id` | uuid v7 PK | |
| `workspace_id` | uuid FK NOT NULL | |
| `name` | text NOT NULL | |
| `data_source` | text NOT NULL | `issue \| doc \| project \| cycle \| activity` (расширяемо) |
| `scope_kind` | text NOT NULL | `workspace \| team \| project \| personal` |
| `team_id` | uuid FK NULL | при scope=team |
| `project_id` | uuid FK NULL | при scope=project |
| `owner_id` | uuid FK→users NULL | при scope=personal |
| `shared` | boolean DEFAULT false | расшарен ли personal/team view |
| `filter` | jsonb | **Filter AST** (§6.2) |
| `grouping` | jsonb NULL | GroupSpec |
| `sub_grouping` | jsonb NULL | swimlanes |
| `ordering` | jsonb | OrderSpec[] (+ ручной порядок) |
| `layout` | text NOT NULL | `board \| list \| table \| calendar` (timeline — v2, ADR-0004) |
| `layout_options` | jsonb NULL | колонки board, видимые поля |
| `display_properties` | jsonb | какие свойства показывать |
| `options` | jsonb NULL | rollupDescendants, showSubIssues, showCompleted, density |
| `is_default` | boolean DEFAULT false | дефолтный view команды |
| `position` | text NULL | порядок в списке Views |
| стандартные | | `version/archived_at/deleted_at` |

**Инварианты**
- `filter` валидируется по `data_source` (какие поля/операторы допустимы, §6.2) перед сохранением — модуль Views.
- Дефолтная страница «Issues» команды = `View` с `data_source=issue, scope=team, grouping=status` — спец-кода нет (§6.1). Один `is_default=true` на команду.
- Временные (несохранённые) views не хранятся — кодируются в URL query (§4.3); сохранение = INSERT.
- Personal view виден только owner, пока `shared=false`.

---

## Dashboard

Холст из виджетов (§6.4).

| Поле | Тип | Заметки |
|---|---|---|
| `id` | uuid v7 PK | |
| `workspace_id` | uuid FK NOT NULL | |
| `name` | text NOT NULL | |
| `scope_kind` | text NOT NULL | `personal \| team \| workspace` |
| `team_id` | uuid FK NULL | |
| `owner_id` | uuid FK→users NULL | |
| `shared` | boolean DEFAULT false | |
| `grid_config` | jsonb NULL | глобальные настройки сетки (cols, rowHeight) |
| `position` | text NULL | порядок в списке Dashboards |
| стандартные | | `version/archived_at/deleted_at` |

---

## DashboardWidget

Виджет на дашборде (§6.4). Ключевой тип `view` встраивает любой `ViewDefinition`.

| Поле | Тип | Заметки |
|---|---|---|
| `id` | uuid v7 PK | |
| `workspace_id` | uuid FK NOT NULL | |
| `dashboard_id` | uuid FK NOT NULL | |
| `type` | text NOT NULL | `view \| stat \| chart \| note` (MVP); `chat/doc/embed` — v2 |
| `x`,`y`,`w`,`h` | int | позиция в grid |
| `title` | text NULL | |
| `config` | jsonb | по типу (см. ниже) |
| `position` | — | (порядок задаётся x/y) |
| стандартные | | `version` |

**`config` jsonb по типу:**
- `view`: `{ viewId?: uuid, inline?: ViewDefinition }` — ссылка на сохранённый View **или** инлайн-определение.
- `stat`: `{ metric: MetricSpec }` — одно число (count/sum поверх Filter AST).
- `chart`: `{ chart: ChartSpec }` — burndown/velocity/агрегация.
- `note`: `{ markdown: string }`.

**Инварианты**
- `ViewWidget` рендерит те же renderer'ы, что полноэкранный View (§6.4) — нет дублирующего кода (фронт-контракт).
- `inline`-определение валидируется так же, как сохранённый View.
- MetricSpec/ChartSpec агрегируют поверх того же Filter AST → консистентность с фильтрами (§6.4).

---

## Favorite

«Звёздочки» на любые сущности, per-user.

| Поле | Тип | Заметки |
|---|---|---|
| `id` | uuid v7 PK | |
| `workspace_id` | uuid FK NOT NULL | |
| `user_id` | uuid FK→users NOT NULL | |
| `entity_type` | text NOT NULL | `issue \| view \| dashboard \| doc \| project \| team \| cycle` |
| `entity_id` | uuid NOT NULL | |
| `position` | text | порядок в избранном (fractional) |
| стандартные | | `version` |

**Инварианты** — UNIQUE `(user_id, entity_type, entity_id)`.

---

## Связи кластера

```
views         *──1 teams/projects/users (по scope)
dashboards    1──* dashboard_widgets ──(config.viewId)──> views
favorites     *──1 users ── polymorphic entity
```

## Открытые микро-вопросы

- Хранить ViewDefinition «разнесённо по колонкам» (как выше) vs одним `definition jsonb`. Выбрано разнесённо для индексации/частичных апдейтов; полностью-jsonb проще, но хуже для частичных дельт. Подтвердить при M3.
- Версионирование/история изменений View — после MVP.
- StatWidget/ChartWidget точный набор метрик — в `02-architecture` (Query/метрики).
