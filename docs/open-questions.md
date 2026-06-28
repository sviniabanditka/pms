# Открытые вопросы

Живой реестр. Закрытое → переезжает в ADR, строка помечается ✅ со ссылкой.
Источник большинства — §12 спеки, плюс всплывающее по ходу проработки.

## Из §12 спеки

| # | Вопрос | Статус |
|---|---|---|
| 1 | Local-first сейчас или поэтапно | ✅ Этап А — [ADR-0002](decisions/ADR-0002-local-first-strategy.md) |
| 2 | Семантика вложенных команд (issue в одной команде + rollup) | ✅ подтверждена — [ADR-0004](decisions/ADR-0004-mvp-scope.md) |
| 3 | Глубина custom fields в MVP | ✅ база + relationship — [ADR-0004](decisions/ADR-0004-mvp-scope.md) |
| 4 | Timeline/Gantt в MVP или раннем v2 | ✅ ранний v2 — [ADR-0004](decisions/ADR-0004-mvp-scope.md) |
| 5 | SaaS vs self-hosted | ✅ SaaS-first — [ADR-0001](decisions/ADR-0001-deploy-model.md) |
| 6 | Свой git-хостинг (v3) vs обёртка над Gitea/GitLab | 🔲 отложен (v3) |
| 7 | AI/MCP-стратегия (v6) | 🔲 отложен (v6) |
| 8 | Публичный API: REST / GraphQL / оба (v5) | 🔲 отложен (v5) |
| 9 | CRDT-доки: v2 или позже | ✅ v2+ — [ADR-0004](decisions/ADR-0004-mvp-scope.md) |
| 10 | Командная нумерация без дыр | ✅ дыры допустимы (принято спекой; подтвердим в домене Issues) |

## Стек (пересматриваем выборочно — §7.2)

| # | Вопрос | Статус |
|---|---|---|
| S1 | Контракты | ✅ ConnectRPC — [ADR-0003](decisions/ADR-0003-backend-stack.md) |
| S2 | БД-доступ | ✅ bun — [ADR-0003](decisions/ADR-0003-backend-stack.md) |
| S3 | Очереди | ✅ River — [ADR-0003](decisions/ADR-0003-backend-stack.md) |
| S4 | WS | ✅ coder/websocket — [ADR-0003](decisions/ADR-0003-backend-stack.md) |
| S5 | Фронт-выборы (графики/календарь/DnD/виртуализация/grid) | ✅ зафиксировано — [components](03-frontend/components.md) §3 |

## Всплывшее по ходу проработки

_(пусто — пополняется)_
