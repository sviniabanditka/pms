# M5 — Calendar (task-календарь: issues по датам) ✅ ГОТОВО

> Это **task-календарь** из §8.1.6 (issues по scheduled/due датам) — оставляем, возможно допилим.
> **Персональный календарь событий** (митинги/события/релизы, `CalendarEvent`) — отдельный Section,
> вынесен в **v2** (детали в [later-versions](04-features/later-versions.md) → v2). Митингов пока нет.

- [x] backend: даты в `UpdateIssue` (SetScheduledDate/SetDueDate, ""→NULL, ?::date)
- [x] store: layout `calendar`, `setIssueSchedule`, getter `calendarIssues(team, date)`
- [x] CalendarRenderer: месячная сетка (Mon-start), сегодня обведено, навигация ‹/›/today
- [x] issues по `scheduledDate` (fallback `dueDate`); DnD на день → смена даты; фильтр применяется
- [x] detail-панель: поле Date (scheduledDate)
- **Приёмка:** поставил дату → issue в ячейке; перетащил на другой день → дата сменилась; reload сохраняет.

Долги (в реестр): только месяц (нет week-view, §8.1.6) — D31; нативный DnD — D19.
