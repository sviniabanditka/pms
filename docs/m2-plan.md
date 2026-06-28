# M2 — план реализации (Board + Workflow)

Цель: Board-renderer (канбан), DnD между колонками статусов + реордеринг внутри колонки,
порядок через fractional-позицию. Тот же домен, что M1 — новый renderer + команда Reorder.

## Срез 1 — Reorder (backend) ✅ ГОТОВО
- [x] позиция issue — `double precision` (миграция 0010; конверт из text); скоуп порядка = (team, state)
- [x] `CreateIssue`: дописывать в конец колонки (`MAX(board_position)+1`)
- [x] `IssueService.ReorderIssue(issueId, stateId, beforeId, afterId)` → смена статуса (если надо) + позиция между соседями (midpoint); proto + кодген
- [x] unit-тест: midpoint между соседями / края / пусто
- **Приёмка:** drag в другую колонку меняет статус (+метки времени) и позицию; порядок стабилен.

## Срез 2 — Board-renderer (frontend) ✅ ГОТОВО
- [x] переключатель layout List/Board на странице команды
- [x] Board: колонки = статусы (по position), карточки = issues (по board_position)
- [x] DnD (нативный HTML5 — скелет; pragmatic-drag-and-drop позже, см. debt) между/внутри колонок → ReorderIssue
- [x] оптимистично + WS-синк
- **Приёмка:** перетащил карточку между колонками → статус сменился, осталась на месте после refresh; реордер внутри колонки сохраняется.

## Долги вехи (в реестр)
- Нативный DnD вместо pragmatic-drag-and-drop (touch/a11y) — заменить позже.
- Float-midpoint позиции: исчерпание точности после ~50 вставок в одну точку → заменить на fractional-index строки.
