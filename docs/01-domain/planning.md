# Домен: Planning

Планирование: **Project · Milestone · Cycle** (+ снапшоты прогресса).
Наследует [конвенции](00-conventions.md). Связано: §3.2, §8.1.3, §8.1.4.

---

## Project

Группировка issues вокруг цели; **охватывает несколько команд**.

| Поле | Тип | Заметки |
|---|---|---|
| `id` | uuid v7 PK | |
| `workspace_id` | uuid FK NOT NULL | |
| `lead_id` | uuid FK→users NULL | |
| `name` | text NOT NULL | |
| `description` | jsonb | rich text |
| `status` | text NOT NULL | `backlog \| planned \| in_progress \| paused \| completed \| canceled` |
| `health` | text NULL | `on_track \| at_risk \| off_track` (ручной/вычисляемый) |
| `start_date` | date NULL | |
| `target_date` | date NULL | |
| `icon` / `color_token` | text NULL | визуал |
| стандартные | | `version/archived_at/deleted_at` |

**Связь с командами (m2m): `project_teams`** — `(project_id, team_id, workspace_id)`.
Проект охватывает несколько команд; issue ссылается на проект через `issues.project_id` (один проект на issue).

**Инварианты**
- Прогресс проекта = агрегат по связанным issues (scope/started/completed) — вычисляемо.
- `health` в MVP — ручной выбор; авто-эвристика (по burndown/датам) — позже.
- Удаление проекта не удаляет issues (отвязка `project_id := NULL`).

---

## Milestone

Веха внутри проекта.

| Поле | Тип | Заметки |
|---|---|---|
| `id` | uuid v7 PK | |
| `workspace_id` | uuid FK NOT NULL | |
| `project_id` | uuid FK NOT NULL | |
| `name` | text NOT NULL | |
| `target_date` | date NULL | |
| `position` | text | порядок (fractional) |
| стандартные | | `version` |

**Инварианты**
- Issue может ссылаться на milestone (`issues.milestone_id` NULL FK) — опционально; решаем нужно ли в MVP (по умолчанию да, лёгкое поле).
- Прогресс вехи = агрегат по её issues.

---

## Cycle

Тайм-боксированная итерация **одной команды** (current/upcoming); источник burndown/velocity.

| Поле | Тип | Заметки |
|---|---|---|
| `id` | uuid v7 PK | |
| `workspace_id` | uuid FK NOT NULL | |
| `team_id` | uuid FK NOT NULL | владелец (cycle принадлежит одной команде) |
| `number` | int NOT NULL | порядковый в команде (Cycle 1, 2, …) |
| `name` | text NULL | необязательное имя |
| `starts_on` | date NOT NULL | |
| `ends_on` | date NOT NULL | |
| `status` | text | `upcoming \| active \| completed` — выводится из дат + завершения |
| `cooldown_days` | int | из `team.cycle_config` |
| стандартные | | `version` |

**Связь с issues** — `issues.cycle_id` (issue в одном цикле за раз).

**Инварианты**
- `(team_id, number)` UNIQUE.
- `active` = `today ∈ [starts_on, ends_on]`; в каждый момент у команды максимум один active (по конфигу).
- Авто-роллофер незавершённых issues в следующий цикл — опция `cycle_config` (§8.1.3).
- Период не пересекается с другим циклом той же команды (для автогенерируемых).

---

## CycleProgressSnapshot (burndown)

Инкрементальные снапшоты для burndown/velocity без тяжёлого пересчёта (§7.9).

| Поле | Тип | Заметки |
|---|---|---|
| `id` | uuid v7 PK | |
| `workspace_id` | uuid FK NOT NULL | |
| `cycle_id` | uuid FK NOT NULL | |
| `at` | date NOT NULL | день снапшота |
| `scope_count` / `scope_estimate` | numeric | всего в цикле |
| `started_count` / `started_estimate` | numeric | |
| `completed_count` / `completed_estimate` | numeric | |

**Инварианты**
- По одному снапшоту на `(cycle_id, at)` (UNIQUE); пишет джоба (River) раз в день + при значимых изменениях.
- Burndown-график (правый панель, §4.1) строится по этим снапшотам; «сегодня» — live-агрегат.

---

## Связи кластера

```
projects *──* teams (project_teams)
projects 1──* milestones
projects 1──* issues (issues.project_id)
teams    1──* cycles 1──* issues (issues.cycle_id)
cycles   1──* cycle_progress_snapshots
```

## Открытые микро-вопросы

- `issues.milestone_id` в MVP — да (по умолчанию).
- `health` авто-вычисление — после MVP.
- Метрика velocity (скользящее среднее N циклов) — формула фиксируется в `02-architecture` (Query/метрики).
