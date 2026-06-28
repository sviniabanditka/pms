# M3 — план реализации (Views + Filter AST)

Цель: Filter AST как данные → **клиентский исполнитель** над локальным стором (мгновенная фильтрация,
local-first), группировки/сортировка, сохранённые Views, FilterBar. Применяется к List и Board.

> **Scope-решение:** в M3 — только клиентский исполнитель (он и даёт UX). Серверный SQL/OpenSearch
> компиляторы + property-based контрактные тесты (§6.2, §10.3) — отдельная веха (нужны для масштаба/
> экспорта, не для local-first UI). Вынесено в debt-ledger (D21).

## Срез 1 — Filter AST + исполнитель (frontend) ✅ ГОТОВО
- [x] TS-тип `FilterNode` (and/or/not/predicate) + `match(node, issue, ctx)` над стором
- [x] операторы: eq/neq/in/nin/isSet/isNotSet/contains (для M3-полей: status/priority/assignee/title/label)
- [ ] self-check на match — **отложено** (нет фронт-тест-раннера, D23)
- [~] группировка в M3 — по статусу (как есть); произвольная (priority/assignee) — отложено (рендереры надо обобщать), см. D22
- **Приёмка:** фильтр мгновенно меняет список/борд без сервера.

## Срез 2 — Saved Views (backend) ✅ ГОТОВО
- [x] миграция `views` (filter/grouping/ordering jsonb, scope, layout) + RLS
- [x] модель `View`; proto `view_service` (Create/Update/Delete) → кодген
- [x] `ViewService.CreateView/UpdateView/DeleteView` + change_log + publish
- [x] bootstrap включает `view`
- **Приёмка:** создать view → строка + bootstrap отдаёт; RLS.

## Срез 3 — FilterBar + Views UI (frontend) ✅ ГОТОВО
- [x] FilterBar: поиск по title (contains) + приоритет (in) chips + clear. (assignee=me/группировка-селектор — отложено, D22/D11)
- [x] применять фильтр к текущей команде (List и Board) — клиентский исполнитель
- [x] «Save view» (имя → CreateView с текущим filter+layout, scope=team)
- [x] Views команды чипами в FilterBar → открыть (грузит filter+layout)
- **Приёмка:** настроил фильтр/группу → сохранил view → переоткрыл → состояние то же.

## Долги вехи (в реестр)
- D21: серверный Filter AST компилятор (SQL/OpenSearch) + контрактные тесты — отдельная веха.
- Cmd-K (базовый) — перенос на M4 (роадмап допускает M3–M4).
