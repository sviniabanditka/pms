# Домен: Identity & Access

Кластер идентичности и доступа: **User · Workspace · WorkspaceMember · Invitation**
+ per-user-per-workspace настройки. Наследует [конвенции](00-conventions.md).

Связано: §3.2, §7.7 (RBAC). Механика аутентификации (токены/сессии) — в `02-architecture`,
здесь только доменные сущности.

---

## User (глобальная, без `workspace_id`)

Учётная запись; живёт в нескольких воркспейсах.

| Поле | Тип | Заметки |
|---|---|---|
| `id` | uuid v7 PK | |
| `email` | citext UNIQUE NOT NULL | логин; case-insensitive |
| `email_verified_at` | timestamptz NULL | |
| `name` | text | отображаемое имя |
| `avatar_url` | text NULL | (файл — через Files-модуль) |
| `password_hash` | text NULL | argon2id; NULL если только OAuth (OAuth — позже) |
| `preferences` | jsonb | глобальные предпочтения (тема зашита тёмная, но место есть: локаль, density-дефолт) |
| `last_active_at` | timestamptz NULL | |
| стандартные | | `created_at/updated_at/version`; archive/delete — см. ниже |

**Инварианты**
- `email` уникален глобально, нормализуется (lowercase/trim).
- Удаление пользователя = GDPR-erasure (§9.4): анонимизация в Activity/Comments, отвязка членств. Не каскадный hard-delete контента команды.
- `version`/`change_log` — да (профиль синкается как справочник участников).

---

## Workspace (tenant)

Верхнеуровневый арендатор, граница изоляции (ADR-0001, ADR-0005).

| Поле | Тип | Заметки |
|---|---|---|
| `id` | uuid v7 PK | сам себе `workspace_id` для единообразия RLS не нужен — это корень |
| `slug` | citext UNIQUE NOT NULL | в URL `/w/:slug`; иммутабелен после создания (смена — отдельная операция с редиректом) |
| `name` | text NOT NULL | |
| `logo_url` | text NULL | |
| `settings` | jsonb | дефолты воркспейса (см. ниже) |
| `created_by` | uuid FK→users | первый owner |
| стандартные | | `created_at/updated_at/version/archived_at/deleted_at` |

**`settings` jsonb (workspace-level дефолты):**
- `estimateScale`: `'points' | 'fibonacci' | 'tshirt'` (§8.1.1).
- `defaultDisplay`: дефолтные display-options (плотность, видимые свойства) — переопределяемы на team/view (§8.1.5).
- `teamNestingPolicy`: политика наследования членства для вложенных команд (§7.7) — `'inherit-public' | 'strict'`.
- `undoWindowMs`, прочие — по мере появления.

**Инварианты**
- `slug` — глобально уникален, `^[a-z0-9-]{2,40}$`.
- Удаление воркспейса — только `owner`; тяжёлая операция (async purge всего тенанта).

---

## WorkspaceMember (членство + per-workspace настройки пользователя)

Связь User ↔ Workspace + роль + персональные настройки в этом воркспейсе.
Грейн «per-user-per-workspace» — здесь же живёт конфиг левого рейла (§4.2).

| Поле | Тип | Заметки |
|---|---|---|
| `id` | uuid v7 PK | |
| `workspace_id` | uuid FK NOT NULL | |
| `user_id` | uuid FK→users NOT NULL | |
| `role` | text NOT NULL | `owner \| admin \| member \| guest` (§7.7) |
| `status` | text NOT NULL | `active \| suspended` |
| `sections_config` | jsonb | выбор и порядок Sections левого рейла (§4.2), per-user-per-workspace |
| `favorites_cache` | — | (Favorite — отдельная сущность, см. views-dashboards) |
| `joined_at` | timestamptz | |
| стандартные | | `version` (роль/настройки синкаются) |

**Инварианты**
- UNIQUE `(workspace_id, user_id)`.
- Всегда ≥1 `owner` на воркспейс (нельзя снять/удалить последнего owner).
- `guest` (§7.7): видит только явно расшаренное/назначенное; не видит дерево команд целиком.
- Роли — фикс-набор MVP (кастомные роли — позже).

**Матрица прав (workspace-уровень, §7.7):**

| Действие | owner | admin | member | guest |
|---|:-:|:-:|:-:|:-:|
| Биллинг, удаление воркспейса | ✅ | | | |
| Управление членами/настройками/командами | ✅ | ✅ | | |
| Обычная работа (issues/views в доступных командах) | ✅ | ✅ | ✅ | огранич. |
| Доступ только к расшаренному/назначенному | — | — | — | ✅ |

(Team- и resource-уровень прав — в `02-architecture/rbac.md`.)

---

## Invitation

Приглашение в воркспейс (§8.1.1).

| Поле | Тип | Заметки |
|---|---|---|
| `id` | uuid v7 PK | |
| `workspace_id` | uuid FK NOT NULL | |
| `email` | citext NOT NULL | приглашаемый |
| `role` | text NOT NULL | роль при принятии |
| `token` | text UNIQUE NOT NULL | секрет ссылки (хранить хэш) |
| `invited_by` | uuid FK→users | |
| `expires_at` | timestamptz NOT NULL | |
| `accepted_at` | timestamptz NULL | |
| стандартные | | без archive/delete; истёкшие чистит джоба |

**Инварианты**
- UNIQUE активного инвайта на `(workspace_id, email)`.
- Принятие: если `users.email` существует — привязка; иначе — флоу регистрации, затем `WorkspaceMember`.
- `token` хранится хэшом (как пароль-ресет).

---

## Связи кластера

```
users (global) 1───* workspace_members *───1 workspaces
users           1───* invitations(by)
workspaces      1───* invitations
workspaces      1───* teams (см. teams-config)
```

## Открытые микро-вопросы (вынесены)

- Метод аутентификации: opaque-токены в Redis vs JWT+сессии → `02-architecture` (влияет на `sessions`, не на эти таблицы).
- Может ли один email иметь несколько `users`? — нет, глобально уникален (принято выше).
