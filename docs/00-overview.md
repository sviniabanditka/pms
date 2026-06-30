# stdtask — Документация проекта

Источник правды по реализации. Спека-карта: [`../PMS_Specification.md`](../PMS_Specification.md) (draft 0.1).
Эти `docs/` разворачивают карту в имплементируемую деталь и фиксируют каждое решение.

## Как устроена документация

| Раздел | Содержание |
|---|---|
| `00-overview.md` | этот файл — точка входа, индекс, статус |
| `decisions/` | ADR: одно решение = один файл. Контекст → решение → почему → последствия |
| `open-questions.md` | живой реестр нерешённого; убывает по мере работы |
| `01-domain/` | сущности: поля, инварианты, связи, схема таблиц |
| `02-architecture/` | sync-протокол, realtime, command/query API, RBAC, query-engine |
| `03-frontend/` | стор/StoreGateway, View↔renderer, дизайн-токены, компоненты |
| `04-features/` | функц. требования по версиям (MVP детально → v6 направлениями) |

Файлы доменов/архитектуры создаются по мере проработки, а не пустыми заглушками.

## Принятые ключевые решения (быстрый индекс ADR)

| ADR | Решение | Статус |
|---|---|---|
| [0001](decisions/ADR-0001-deploy-model.md) | Деплой-модель: **SaaS-first** | accepted |
| [0002](decisions/ADR-0002-local-first-strategy.md) | Local-first: **Этап А (optimistic + delta-sync)** в MVP | accepted |
| [0003](decisions/ADR-0003-backend-stack.md) | Бэк-стек: **ConnectRPC · bun · River · coder/websocket** | accepted |
| [0004](decisions/ADR-0004-mvp-scope.md) | Скоуп MVP: teams-семантика, custom fields (база+relationship), 4 renderer'а, доки LWW | accepted |
| [0005](decisions/ADR-0005-data-conventions.md) | Конвенции данных: UUIDv7, app-level+RLS, archive+soft-delete, version/change_log, fractional | accepted |
| [0006](decisions/ADR-0006-auth.md) | Auth: **opaque-токены в Redis** | accepted |

## Доменная модель (`01-domain/`)

| Док | Сущности |
|---|---|
| [00-conventions](01-domain/00-conventions.md) | стандартные колонки, change_log, fractional indexing, JSONB |
| [identity-access](01-domain/identity-access.md) | User, Workspace, WorkspaceMember, Invitation |
| [teams-config](01-domain/teams-config.md) | Team (вложенность), TeamMember, WorkflowState, Label |
| [issues](01-domain/issues.md) | Issue, IssueRelation, Comment, Attachment, CustomField/Value, Subscription, Reaction |
| [planning](01-domain/planning.md) | Project, Milestone, Cycle, CycleProgressSnapshot |
| [views-dashboards](01-domain/views-dashboards.md) | View, Dashboard, DashboardWidget, Favorite |
| [content-collab](01-domain/content-collab.md) | Document, Activity |
| [platform](01-domain/platform.md) | Notification, NotificationPreference |

## Архитектура (`02-architecture/`)

| Док | Содержание |
|---|---|
| [query-engine](02-architecture/query-engine.md) | Filter AST → 3 исполнителя (client/SQL/OpenSearch), валидация, метрики |
| [sync-protocol](02-architecture/sync-protocol.md) | команды, дельты, догон, per-field LWW, апгрейд на Этап Б |
| [realtime](02-architecture/realtime.md) | WS-каналы, fan-out через Redis, presence, батчинг |
| [rbac](02-architecture/rbac.md) | policy-слой, наследование при вложенности, permission-карта |
| [commands](02-architecture/commands.md) | каталог команд + ConnectRPC-сервисы, конвейер, ошибки |
| [notifications](02-architecture/notifications.md) | генерация из событий, дедуп, доставка |
| [modules](02-architecture/modules.md) | границы Go-пакетов, event bus, путь к экстракции |
| [auth](02-architecture/auth.md) | opaque-токены, сессии в Redis, WS-auth, tenancy-связь |
| [ddl-m0](02-architecture/ddl-m0.md) | code-ready миграции фундамента M0 + RLS + индексы |
| [proto-commands](02-architecture/proto-commands.md) | code-ready .proto: команды, Filter AST, View, sync |
| [ws-protocol](02-architecture/ws-protocol.md) | code-ready WS-фреймы: subscribe/delta/presence, close-коды |

## Фронтенд (`03-frontend/`)

| Док | Содержание |
|---|---|
| [store](03-frontend/store.md) | нормализованный стор, StoreGateway, command bus, оптимистичный поток |
| [view-renderer](03-frontend/view-renderer.md) | контракт View↔renderer, 4 renderer'а MVP, один компонент — много мест |
| [design-system](03-frontend/design-system.md) | токены, тёмная тема, типографика, плотность, a11y |
| [components](03-frontend/components.md) | примитивы + доменные компоненты, S5-библиотеки (зафиксированы) |
| [theme-tokens](03-frontend/theme-tokens.md) | code-ready тёмная палитра (hex), status/priority/label, Tailwind+Nuxt UI конфиг |

## Функц. требования, тестирование, roadmap

| Док | Содержание |
|---|---|
| [04-features/mvp](04-features/mvp.md) | детальные требования MVP, кросс-ссылки на домен/архитектуру, DoD |
| [04-features/later-versions](04-features/later-versions.md) | v2–v6 направления, точки интеграции, матрица версий |
| [05-testing](05-testing.md) | пирамида, контрактные тесты Filter AST, E2E-сценарии, TDD-узлы |
| [06-roadmap](06-roadmap.md) | вехи M0–M8 с критериями приёмки |

## Статус

**Первый полный проход завершён.** Покрыты: §12-развилки (ADR), доменная модель (8 доков),
архитектура (8 доков), фронтенд (4 дока), функц. требования, тестирование, roadmap.
Открытые микро-вопросы — в [open-questions](open-questions.md) (отложенные на v2–v6 или на этап реализации).

Готово к декомпозиции и старту M0. Любой раздел углубляется по запросу
(напр. полный каталог команд proto, точные DDL-миграции, токены тёмной темы).
