# Функц. требования: MVP

Полноценный трекер с представлениями, фильтрами, кастомизацией, календарём; локально-быстрый (§8.1).
Каждая фича ссылается на домен/архитектуру, где детализирована. Здесь — что должно работать.

Критический путь движка: домен + Filter AST (двойной компилятор) + локальный стор/синк +
4 renderer'а + Calendar. Остальное наслаивается (§8.1, замечание в конце).

---

## 1. Управление пространством
- Создание/переключение/настройки воркспейса; приглашения; роли owner/admin/member/guest. → [identity-access](../01-domain/identity-access.md)
- Команды с **вложенностью**; ключи; приватные; per-team `enabled_features`. → [teams-config](../01-domain/teams-config.md), ADR-0004 #2
- Workflow-статусы per-team (категории/цвета/порядок). → teams-config
- Конфиг циклов (длительность/старт/cooldown/авто-старт). → teams-config, [planning](../01-domain/planning.md)
- Лейблы (workspace/team scope, группы). → teams-config
- Оценочная шкала воркспейса (points/fibonacci/t-shirt). → `workspace.settings.estimateScale`

## 2. Issues
- CRUD; `KEY-N` (атомарная нумерация, дыры ок); rich-text описание. → [issues](../01-domain/issues.md)
- Свойства: status, priority, assignee, labels, estimate, due/start/scheduled, project, cycle.
- **Sub-issues** (иерархия, прогресс по детям), **relations** (blocks/blocked-by/related/duplicate, зеркала).
- Комментарии (треды, реакции, упоминания), вложения (файлы/ссылки), подписчики, активность.
- Inline-edit, multi-select + bulk actions, DnD, undo, архивирование. → [components](../03-frontend/components.md) §5
- Базовые хоткеи (create/assign/status/priority/navigation). → components §5

## 3. Cycles
- Current/Upcoming; добавление/перенос issues; авто-роллофер незавершённых (опция). → planning
- Метрики: scope/started/completed, **burndown** (снапшоты), velocity. → planning, [query-engine](../02-architecture/query-engine.md) §5

## 4. Projects
- lead/status/health/target; связь с несколькими командами; milestones; обзор прогресса. → planning

## 5. Views и отображения (сердце MVP)
- Layouts: **Board, List, Table, Calendar** (Timeline — ранний v2, ADR-0004). → [view-renderer](../03-frontend/view-renderer.md)
- Группировка/подгруппировка, сортировки, ручной порядок (fractional). → query-engine §4
- **Filter AST** с полным набором операторов; временные фильтры с шарингом через URL. → query-engine, §4.3
- Сохранённые views (personal/team/project/workspace, shared); дефолтный view команды. → [views-dashboards](../01-domain/views-dashboards.md)
- **Display options**: видимые свойства, плотность, sub-issues/completed, rollup дочерних команд.
- Настройки отображения **для воркспейса** (дефолты), переопределяемые на team/view.

## 6. Calendar (в MVP)
- Месяц/неделя; issues по `scheduled_date`/`due_date`; DnD-перепланирование (`SetDates`). → view-renderer §3
- Фильтры/скоуп (мои/команда/проект); цветовое кодирование по статусу/приоритету/команде.
- (Полная двусторонняя синхронизация с внешними календарями — v2.)

## 7. Кастомизация и реюзабельность
- **Custom fields**: база + relationship (ADR-0004 #3) с фильтрацией и отображением в таблице. → issues, query-engine
- **Дашборды**: холст виджетов; ViewWidget, StatWidget, ChartWidget, NoteWidget (chat/embed — v2). → views-dashboards, §6.4
- Настраиваемый левый рейл (выбор/порядок Sections) per-user. → `WorkspaceMember.sections_config`

## 8. Платформенные базовые
- Inbox/уведомления (assigned/mentioned/state-changed/commented/due-soon) + preferences. → [notifications](../02-architecture/notifications.md), [platform](../01-domain/platform.md)
- Глобальный поиск и quick-switch (issues/views/docs/projects/teams). → SearchService, базовый Cmd-K
- Docs (rich text, дерево; совместное редактирование — позже). → [content-collab](../01-domain/content-collab.md)
- Realtime-синхронизация между клиентами (Этап А); оптимистичный UI; индикатор синка. → [sync-protocol](../02-architecture/sync-protocol.md), [realtime](../02-architecture/realtime.md)
- Аутентификация (opaque-токены/сессии), базовая безопасность, аудит-лог. → [auth](../02-architecture/auth.md), [content-collab](../01-domain/content-collab.md) (Activity)

---

## Definition of Done (MVP, сквозное)
- Все мутации оптимистичны, проходят RBAC, отражаются в `change_log` и синкаются между клиентами.
- Filter AST: три исполнителя дают идентичный результат (контрактные тесты, §10.3).
- Воспринимаемая задержка интерактива < 50мс (§9.1).
- Критические E2E-сценарии (§10.2) зелёные. → [testing](../05-testing.md)
