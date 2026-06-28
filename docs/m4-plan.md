# M4 — план (Planning)

## Часть 1: Sub-issues + Relations ✅ ГОТОВО
(см. историю; срезы 1–2 закрыты)

## Часть 2: Projects + Cycles

### Срез 1 — backend ✅ ГОТОВО
- [x] миграция `projects` (+FK `issues.project_id`) и `cycles` (+FK `issues.cycle_id`) + RLS
- [x] модели `Project`, `Cycle`; proto `project_service`, `cycle_service` + `IssueService.SetProject/SetCycle` → кодген
- [x] `ProjectService.CreateProject/UpdateProject`; `CycleService.CreateCycle/UpdateCycle` (number авто per-team)
- [x] `IssueService.SetProject/SetCycle` (привязка/снятие)
- [x] bootstrap включает `project`, `cycle`
- **Приёмка:** создать проект/цикл; привязать issue; FK целостны; RLS.

### Срез 2 — frontend ✅ ГОТОВО
- [x] detail-панель: селекты Project и Cycle (привязка issue)
- [x] сайдбар: создать проект (workspace) + список; создать цикл (для выбранной команды) + список Current/Upcoming (по датам)
- [x] клик по проекту → main показывает его issues (по project_id, по статусам); клик по циклу → issues цикла
- **Приёмка:** привязал issue к проекту → виден в проекте; цикл current/upcoming по датам.

## Часть 3: burndown ✅ ГОТОВО
- [x] метрики цикла (scope/started/completed) live из стора
- [x] burndown SVG (ideal пунктир + actual из completedAt), полоса над списком
- [x] клик цикла → openCycle (фильтр + burndown); close убирает
- Снапшоты/velocity (River) — отложены (D29)
