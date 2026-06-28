# DDL: миграции M0 (фундамент)

Code-ready SQL под M0 ([roadmap](../06-roadmap.md)): identity-access + teams-config + change_log + RLS.
Issues и далее — M1+ (отдельные миграции). Конвенции — [ADR-0005](../decisions/ADR-0005-data-conventions.md).

**Формат:** golang-migrate (`NNNN_name.up.sql` / `.down.sql`). Целевой путь при скаффолде бэка —
`backend/internal/db/migrations/`. Здесь — содержимое up (down — симметричный `DROP` в обратном порядке).

**Соглашения, отражённые в DDL:**
- `id uuid PRIMARY KEY` — **приложение генерит UUIDv7** (client-generated, §sync-protocol). Без DB-default → нет plpgsql-зависимости.
- Время — `timestamptz`; `version bigint`; `archived_at`/`deleted_at` где применимо.
- `workspace_id` на всех таблицах кроме `users`/`workspaces`; RLS по `current_setting('app.workspace_id')`.
- Приложение коннектится ролью с **enforced RLS**; джобам с кросс-тенантным доступом (purge) — отдельная роль `BYPASSRLS`.

---

## 0001_extensions.up.sql

```sql
CREATE EXTENSION IF NOT EXISTS citext;   -- case-insensitive email/slug
CREATE EXTENSION IF NOT EXISTS ltree;    -- teams.path (subtree/ancestor)
CREATE EXTENSION IF NOT EXISTS pgcrypto; -- fallback gen_random_uuid(), хэши

-- Роль приложения с принудительным RLS (создаётся вне миграций в проде; здесь для dev/test):
-- CREATE ROLE pms_app LOGIN; ALTER ROLE pms_app SET search_path = public;
-- Роль фоновых джоб с обходом RLS:
-- CREATE ROLE pms_jobs LOGIN BYPASSRLS;
```

---

## 0002_users.up.sql

```sql
CREATE TABLE users (
    id                uuid PRIMARY KEY,
    email             citext NOT NULL UNIQUE,
    email_verified_at timestamptz,
    name              text NOT NULL DEFAULT '',
    avatar_url        text,
    password_hash     text,                 -- argon2id; NULL для OAuth-only (позже)
    preferences       jsonb NOT NULL DEFAULT '{}'::jsonb,
    last_active_at    timestamptz,
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now(),
    version           bigint NOT NULL DEFAULT 0,
    deleted_at        timestamptz           -- GDPR-erasure/анонимизация
);
-- users — глобальная, без workspace_id и без RLS.
```

---

## 0003_workspaces.up.sql

```sql
CREATE TABLE workspaces (
    id          uuid PRIMARY KEY,
    slug        citext NOT NULL UNIQUE,
    name        text NOT NULL,
    logo_url    text,
    settings    jsonb NOT NULL DEFAULT '{}'::jsonb,  -- estimateScale, defaultDisplay, teamNestingPolicy…
    created_by  uuid NOT NULL REFERENCES users(id),
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    version     bigint NOT NULL DEFAULT 0,
    archived_at timestamptz,
    deleted_at  timestamptz,
    CONSTRAINT workspaces_slug_fmt CHECK (slug ~ '^[a-z0-9-]{2,40}$')
);

ALTER TABLE workspaces ENABLE ROW LEVEL SECURITY;
CREATE POLICY workspaces_tenant ON workspaces
    USING (id = current_setting('app.workspace_id', true)::uuid);
```

---

## 0004_workspace_members.up.sql

```sql
CREATE TABLE workspace_members (
    id              uuid PRIMARY KEY,
    workspace_id    uuid NOT NULL REFERENCES workspaces(id),
    user_id         uuid NOT NULL REFERENCES users(id),
    role            text NOT NULL CHECK (role IN ('owner','admin','member','guest')),
    status          text NOT NULL DEFAULT 'active' CHECK (status IN ('active','suspended')),
    sections_config jsonb NOT NULL DEFAULT '{}'::jsonb,   -- конфиг левого рейла (per-user-per-ws)
    joined_at       timestamptz NOT NULL DEFAULT now(),
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    version         bigint NOT NULL DEFAULT 0,
    UNIQUE (workspace_id, user_id)
);
CREATE INDEX workspace_members_user ON workspace_members (user_id);

ALTER TABLE workspace_members ENABLE ROW LEVEL SECURITY;
CREATE POLICY workspace_members_tenant ON workspace_members
    USING (workspace_id = current_setting('app.workspace_id', true)::uuid);

CREATE TABLE invitations (
    id           uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspaces(id),
    email        citext NOT NULL,
    role         text NOT NULL CHECK (role IN ('owner','admin','member','guest')),
    token_hash   text NOT NULL UNIQUE,        -- хэш секрета ссылки
    invited_by   uuid NOT NULL REFERENCES users(id),
    expires_at   timestamptz NOT NULL,
    accepted_at  timestamptz,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    version      bigint NOT NULL DEFAULT 0
);
-- частичный UNIQUE: один активный инвайт на (workspace,email)
CREATE UNIQUE INDEX invitations_active ON invitations (workspace_id, email)
    WHERE accepted_at IS NULL;

ALTER TABLE invitations ENABLE ROW LEVEL SECURITY;
CREATE POLICY invitations_tenant ON invitations
    USING (workspace_id = current_setting('app.workspace_id', true)::uuid);
```

---

## 0005_teams.up.sql

```sql
CREATE TABLE teams (
    id                uuid PRIMARY KEY,
    workspace_id      uuid NOT NULL REFERENCES workspaces(id),
    parent_id         uuid REFERENCES teams(id),
    path              ltree NOT NULL,
    key               text NOT NULL,
    name              text NOT NULL,
    private           boolean NOT NULL DEFAULT false,
    next_issue_number bigint NOT NULL DEFAULT 0,   -- атомарный счётчик нумерации
    enabled_features  jsonb NOT NULL DEFAULT '{}'::jsonb,
    cycle_config      jsonb NOT NULL DEFAULT '{}'::jsonb,
    settings          jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now(),
    version           bigint NOT NULL DEFAULT 0,
    archived_at       timestamptz,
    deleted_at        timestamptz,
    UNIQUE (workspace_id, key),
    CONSTRAINT teams_key_fmt CHECK (key ~ '^[A-Z][A-Z0-9]{0,9}$'),
    CONSTRAINT teams_no_self_parent CHECK (parent_id IS NULL OR parent_id <> id)
);
CREATE INDEX teams_parent ON teams (parent_id);
CREATE INDEX teams_path_gist ON teams USING gist (path);
CREATE INDEX teams_ws ON teams (workspace_id);

ALTER TABLE teams ENABLE ROW LEVEL SECURITY;
CREATE POLICY teams_tenant ON teams
    USING (workspace_id = current_setting('app.workspace_id', true)::uuid);

CREATE TABLE team_members (
    id           uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspaces(id),
    team_id      uuid NOT NULL REFERENCES teams(id),
    user_id      uuid NOT NULL REFERENCES users(id),
    role         text NOT NULL CHECK (role IN ('lead','member')),
    source       text NOT NULL DEFAULT 'direct' CHECK (source IN ('direct','inherited')),
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    version      bigint NOT NULL DEFAULT 0,
    UNIQUE (team_id, user_id)
);
CREATE INDEX team_members_user ON team_members (workspace_id, user_id);

ALTER TABLE team_members ENABLE ROW LEVEL SECURITY;
CREATE POLICY team_members_tenant ON team_members
    USING (workspace_id = current_setting('app.workspace_id', true)::uuid);
```

> `path` поддерживается приложением при insert/move (`parent.path || id_label`). Цикл вложенности
> предотвращается проверкой на move (родитель не в поддереве). `ltree`-метка — текст из `id` без дефисов.

---

## 0006_workflow_states_labels.up.sql

```sql
CREATE TABLE workflow_states (
    id           uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspaces(id),
    team_id      uuid NOT NULL REFERENCES teams(id),
    category     text NOT NULL CHECK (category IN
                   ('backlog','unstarted','started','completed','canceled')),
    name         text NOT NULL,
    color_token  text NOT NULL,           -- ссылка на токен, не hex
    position     text NOT NULL,           -- fractional
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    version      bigint NOT NULL DEFAULT 0
);
CREATE INDEX workflow_states_team ON workflow_states (team_id);

ALTER TABLE workflow_states ENABLE ROW LEVEL SECURITY;
CREATE POLICY workflow_states_tenant ON workflow_states
    USING (workspace_id = current_setting('app.workspace_id', true)::uuid);

CREATE TABLE labels (
    id           uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspaces(id),
    team_id      uuid REFERENCES teams(id),     -- NULL = workspace-scope
    group_id     uuid REFERENCES labels(id),
    is_group     boolean NOT NULL DEFAULT false,
    name         text NOT NULL,
    color_token  text NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    version      bigint NOT NULL DEFAULT 0
);
CREATE INDEX labels_scope ON labels (workspace_id, team_id);

ALTER TABLE labels ENABLE ROW LEVEL SECURITY;
CREATE POLICY labels_tenant ON labels
    USING (workspace_id = current_setting('app.workspace_id', true)::uuid);
```

---

## 0007_change_log.up.sql

```sql
CREATE TABLE change_log (
    seq            bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,  -- монотонный курсор
    workspace_id   uuid NOT NULL REFERENCES workspaces(id),
    entity_type    text NOT NULL,
    entity_id      uuid NOT NULL,
    version        bigint NOT NULL,
    op             text NOT NULL CHECK (op IN ('create','update','delete','archive')),
    changed_fields jsonb,
    actor_id       uuid REFERENCES users(id),
    at             timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX change_log_ws_seq ON change_log (workspace_id, seq);
CREATE INDEX change_log_entity ON change_log (entity_type, entity_id);

ALTER TABLE change_log ENABLE ROW LEVEL SECURITY;
CREATE POLICY change_log_tenant ON change_log
    USING (workspace_id = current_setting('app.workspace_id', true)::uuid);
```

> Догон клиента: `SELECT … FROM change_log WHERE workspace_id = $1 AND seq > $cursor ORDER BY seq`.
> `seq` монотонен per-БД; курсор клиента — последний применённый `seq`.

---

## Применение RLS-контекста (в каждой транзакции приложения)

```sql
-- начало транзакции под ролью pms_app:
SET LOCAL app.workspace_id = '...uuid...';
-- далее любые запросы автоматически скоупятся политиками; забытый WHERE не утечёт.
```

bun-обвязка ставит `SET LOCAL` сразу после `BEGIN`, до доменных запросов (см. [sync-protocol](sync-protocol.md) §2 конвейер).

## Замечания
- `next_issue_number` бампается в транзакции создания issue (M1): `UPDATE teams SET next_issue_number = next_issue_number + 1 WHERE id = $1 RETURNING next_issue_number` — row-lock сериализует.
- River создаёт свои таблицы своей миграцией (`river migrate`), в той же БД.
- Дефолтные `workflow_states` команды (по одному+ на категорию) создаются доменом при `CreateTeam`, не миграцией.
- down-миграции: `DROP TABLE … CASCADE` в обратном порядке + `DROP POLICY`/`DROP EXTENSION` (extensions обычно не дропаем).
