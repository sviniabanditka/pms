# Архитектура: Команды и контракты (ConnectRPC)

Write-path: типизированные команды (не CRUD), идемпотентные, оптимистичные (§7.6).
Контракты в protobuf → Go-типы + TS-клиент (ADR-0003 S1). Read/stream — отдельно.
Связано: §7.6, [sync-protocol](sync-protocol.md), [rbac](rbac.md).

---

## 1. Структура сервисов

| Сервис (proto) | Назначение |
|---|---|
| `IssueService` | команды над issues/sub-issues/relations/labels/comments |
| `TeamService` | команды над teams/states/labels-config |
| `PlanningService` | projects/milestones/cycles |
| `ViewService` | views/dashboards/widgets/favorites |
| `WorkspaceService` | workspace/members/invitations/settings |
| `DocService` | документы |
| `QueryService` | чтение: исполнение View/Filter AST, метрики (unary) |
| `SyncService` | bootstrap-срез + поток дельт (server-streaming) |
| `SearchService` | глобальный поиск / quick-switch |

**Типизированные методы**, не один generic `Execute` с `Any`: ConnectRPC силён именно типобезопасностью.
Каждый write-метод принимает свой message и возвращает `CommandResult`.

```proto
message CommandResult {
  string entity_id = 1;
  int64  new_version = 2;
}
```

Идемпотентность — заголовок `Idempotency-Key` = `command.id` (client UUIDv7); сервер дедупит в окне.

---

## 2. Каталог команд MVP (репрезентативно, не исчерпывающе)

**Issues** — `CreateIssue`, `UpdateIssue` (частичный набор полей + `baseVersion`), `UpdateIssueState`,
`AssignIssue`, `SetPriority`, `AddLabel`/`RemoveLabel`, `SetEstimate`, `SetDates`, `MoveToProject`,
`MoveToCycle`, `ReorderIssue` (board/backlog/sub), `CreateSubIssue`, `AddRelation`/`RemoveRelation`,
`ArchiveIssue`/`DeleteIssue`/`RestoreIssue`, `SetCustomFieldValue`, `Subscribe`/`Unsubscribe`,
`AddComment`/`EditComment`/`DeleteComment`, `AddReaction`/`RemoveReaction`.

**Teams** — `CreateTeam`, `UpdateTeam`, `MoveTeam` (смена parent), `CreateWorkflowState`,
`UpdateWorkflowState`, `ReorderWorkflowState`, `CreateLabel`/`UpdateLabel`.

**Planning** — `CreateProject`/`UpdateProject`, `LinkProjectTeam`, `CreateMilestone`,
`CreateCycle`/`UpdateCycle`, `AddIssuesToCycle`.

**Views/Dashboards** — `CreateView`/`UpdateView`/`DeleteView`, `SetDefaultView`,
`CreateDashboard`/`UpdateDashboard`, `AddWidget`/`UpdateWidget`/`RemoveWidget`,
`AddFavorite`/`RemoveFavorite`.

**Workspace** — `CreateWorkspace`, `UpdateWorkspaceSettings`, `InviteMember`,
`AcceptInvitation`, `UpdateMemberRole`, `UpdateSectionsConfig`.

**Custom fields** — `CreateCustomField`/`UpdateCustomField`/`ArchiveCustomField`.

**Notifications** — `MarkRead`/`MarkAllRead`, `UpdateNotificationPreference`.

---

## 3. Конвейер обработки команды

```
receive → dedup(command.id) → decode → validate(payload+FilterAST) →
authorize(RBAC) → tx{ apply → version++ → change_log → enqueue River jobs } →
publish delta(Redis) → return CommandResult
```

- Всё в одной транзакции (River транзакционен, ADR-0003 S3) — нет dual-write.
- `validate` использует реестр полей/схемы команд; невалидное отклоняется до tx.
- `authorize` — единственная точка прав ([rbac](rbac.md)); отказ → `permission_denied`.

---

## 4. Модель ошибок

| Код (Connect) | Когда |
|---|---|
| `invalid_argument` | провал валидации payload/AST |
| `permission_denied` | RBAC отказ |
| `failed_precondition` | конфликт инварианта (напр. удаление статуса с issues) |
| `aborted` | конфликт версий, требующий ретрая клиента |
| `already_exists` | дубль по идемпотентности (возвращаем прежний результат) |

Клиент по ошибке откатывает оптимистичное изменение + toast (§sync-protocol).

## Открытые микро-вопросы

- `UpdateIssue` generic (набор полей) vs гранулярные команды на каждое поле. Выбрано: generic `UpdateIssue` для свойств + отдельные команды там, где есть побочные эффекты (`UpdateIssueState` ставит started/completed). Подтвердить на M1–M2.
- Батч-команды (bulk actions, multi-select) — `BulkUpdateIssues{ids[], patch}` одной транзакцией; в каталоге MVP, детализировать.
- Версионирование proto / эволюция контрактов — buf + правила обратной совместимости.
