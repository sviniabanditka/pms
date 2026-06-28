# Архитектура: Доменные модули (границы монолита)

Модульный монолит на Go: `internal/<domain>`. Общение между модулями — через сервис-интерфейсы
и события, **без прямого доступа к чужим таблицам** (§7.3). Это удешевляет будущую экстракцию
сервисов (git/CI/IDE/AI — поздние версии) и self-host.

Связано: §7.3, ADR-0001 (последствия для границ).

---

## 1. Модули

| Модуль | Ответственность |
|---|---|
| `identity` | users, аутентификация, сессии, 2FA (позже) |
| `workspaces` | воркспейсы, members, роли, настройки, invitations, billing (позже) |
| `teams` | команды (вложенность/path), team-members, workflow states, cycle config, labels |
| `issues` | issues, sub-issues, relations, estimates, нумерация, порядок, comments, attachments, custom fields, subscriptions, reactions |
| `projects` | проекты, milestones, project-team связи, cycles |
| `views` | сохранённые views, dashboards, widgets, валидация Filter AST |
| `query` | исполнение Filter AST: компиляция в SQL/OpenSearch, агрегации/метрики |
| `docs` | документы, дерево |
| `activity` | аудит-лог, ленты активности |
| `notifications` | генерация, Inbox, preferences, доставка |
| `realtime` | presence, broadcast дельт, каналы подписки |
| `search` | индексация в OpenSearch, глобальный поиск, quick-switch |
| `files` | загрузка/выдача вложений, превью |

---

## 2. Правила границ

- Каждый модуль владеет своими таблицами; чужие читает **только** через публичный интерфейс модуля-владельца.
- Кросс-модульная связь:
  - **синхронно** — вызов сервис-интерфейса (напр. `issues` спрашивает `teams.Visibility()`).
  - **асинхронно** — доменные события через **in-process event bus** (MVP) → подписчики ставят River-джобы (нотификации, индексация, activity).
- **RabbitMQ** включается, когда появляются внешние интеграции (v3/v5) — событийная шина наружу; в MVP не нужен (§7.2).

---

## 3. Shared kernel

Общий минимум, от которого зависят все модули (без бизнес-логики):
- ID (UUIDv7), ошибки/коды, `workspace_id`-скоуп/RLS-обвязка, `change_log`/`version` примитивы, command bus, event bus, время/таймзоны.

---

## 4. Направление зависимостей

```
shared kernel ← (все модули)
identity ← workspaces ← teams ← {issues, projects, docs}
{issues, projects, docs} → query (исполнение фильтров)
{issues, projects, docs, comments} → activity → notifications
все мутации → realtime (broadcast), search (index) через event bus
```

- Циклы запрещены; пересечения разрешаются через события, а не взаимные вызовы.
- `query` зависит от схемы данных модулей через реестр полей (§query-engine), не от их внутренностей.

---

## 5. Путь к экстракции сервисов (поздние версии)

- Границы уже сервисные → git-хостинг (v3), CI-раннеры (v3), web IDE (v4), AI/MCP (v6) выносятся как отдельные сервисы за теми же интерфейсами/событиями.
- Первые кандидаты на вынос — git-хостинг и CI-раннеры (§8.3): изолированы, ресурсоёмки.

## Открытые микро-вопросы

- `issues` — крупный модуль; возможно дробление на под-пакеты (issues-core / comments / custom-fields) внутри границы. Решаем при росте.
- In-process event bus: синхронные listeners (в tx) vs асинхронные (после commit). Решение: транзакционные эффекты — через River (после commit, но транзакционно поставлены); чистые in-tx инварианты — прямой вызов.
