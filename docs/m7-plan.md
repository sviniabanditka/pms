# M7 — план (Платформа)

## Часть 1: Cmd-K + глобальный поиск ✅ ГОТОВО
Клиентский поиск над стором (local-first, мгновенно), палитра по Cmd/Ctrl+K.
- [x] store.searchAll(q): issues (identifier/title), teams, projects, dashboards, views
- [x] CommandPalette (modal): Cmd/Ctrl+K открывает, Esc закрывает, стрелки/Enter
- [x] переход по результату: issue→detail, team→tasks, project/cycle-фильтр, dashboard→раздел, view→применить
- **Приёмка:** Cmd-K → ввод → выбор → нужный контекст открывается.

## Часть 2: Inbox / нотификации ✅ ГОТОВО
- [x] backend: notifications table + notifications.Record (в tx) + MarkRead/MarkAllRead + bootstrap (per-user)
- [x] генерация: AssignIssue→assigned, UpdateIssueState→state_changed (получатели ≠ actor)
- [x] frontend: **топбар** (свитчер ws + Search⌘K + ✉ дропдаун с бейджем), раздел **Inbox** в рейле, mark read
- Comments/mentions/due_soon — нет (комментов ещё нет); River-доставка email — D1

## Часть 3: Docs ✅ ГОТОВО
- [x] backend: documents (tree-ready parent_id) + DocService Create/Update/Delete + bootstrap
- [x] frontend: секция Docs внутри команды (уровень Cycles), редактор (title + markdown textarea, LWW)
- Дерево доков (parent_id есть в БД) — UI плоский список (D); rich-редактор TipTap — позже (D)

## realtime-доводка
D7 (инкрементальное применение дельт вместо полного re-bootstrap) — кандидат сюда.
