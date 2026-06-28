# M1 — план реализации (Issues + List)

Цель: полноценные issues с `KEY-N`, свойствами, inline-edit; ListRenderer, сгруппированный по статусу.
Filter AST/произвольные группировки/сохранённые views — M3 (здесь группировка по статусу хардкодом).

## Срез 1 — Issue-домен (backend) ✅ ГОТОВО
- [x] миграция `issues` (+ `issue_labels`) + RLS + индексы; `cycle_id`/`project_id` — nullable без FK (FK в M4)
- [x] модель `Issue` (json-теги), `IssueLabel`
- [x] proto `issue_service` (Create/Update/State/Assign/Label/Archive) → кодген Go+TS
- [x] `IssueService.CreateIssue`: атомарный `KEY-N` (UPDATE teams RETURNING), дефолтный статус, state→started/completed, change_log
- [x] bootstrap включает `issue`
- **Приёмка:** create issue → корректный `KEY-N`; строка + change_log; RLS; нумерация без коллизий.

## Срез 2 — Мутации issue (backend) ✅ ГОТОВО
- [x] `UpdateIssue` (применяет присутствующие поля title/priority/assignee — merge-семантика; `base_version` принимается, но enforcement по версии отложен)
- [x] `UpdateIssueState` (категория → авто `started_at`/`completed_at`)
- [x] `AssignIssue`, `SetPriority`, `SetTitle`, `ArchiveIssue`
- [ ] `AddLabel`/`RemoveLabel` — **отложено** (нет UI лейблов в M1; Unimplemented)
- [x] unit-тест: переход статуса (категории→метки). [ ] per-field LWW тест — отложен с enforcement
- **Приёмка ✅:** смена статуса ставит/сбрасывает метки (BLD-1: backlog→started→completed проверено); нумерация BLD-1/BLD-2.

## Срез 3 — List-renderer (frontend) ✅ ГОТОВО
- [x] страница команды: issues, сгруппированные по статусу (колонки-секции по категориям/позиции)
- [x] `IssueRow`: identifier, title, status, priority, assignee
- [x] создание issue (inline в группе), inline-edit (статус/приоритет/assignee/title) — оптимистично
- [x] WS-дельта обновляет список; персист
- **Приёмка:** создать issue в UI → `KEY-N` появляется; правка статуса перетаскивает в другую группу; live между вкладками.

## Долги M1 (вперёд)
- IndexedDB уже есть; инкрементальный fetch сущности по дельте (вместо refresh) — кандидат.
- sub-issues/relations/comments/attachments — позже в M1 или M2.
