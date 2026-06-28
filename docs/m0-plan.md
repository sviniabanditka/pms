# M0 — план реализации (фундамент)

Детальный чеклист. 6 вертикальных срезов, каждый — рабочий и проверяемый.
Источники: [roadmap](06-roadmap.md) M0, [ddl-m0](02-architecture/ddl-m0.md), [proto-commands](02-architecture/proto-commands.md),
[ws-protocol](02-architecture/ws-protocol.md), [auth](02-architecture/auth.md), [sync-protocol](02-architecture/sync-protocol.md).

> Версии библиотек **проверять find-docs на момент скаффолда** (Nuxt/Nuxt UI/bun/River/buf/connect быстро меняются).

## Структура монорепо

```
pms/
├── PMS_Specification.md · docs/
├── proto/pms/v1/*.proto          buf.yaml · buf.gen.yaml
├── backend/
│   ├── go.mod · cmd/server/main.go
│   ├── internal/{config,db,httpx,auth,changelog,bus,
│   │             domain/{workspaces,teams},sync,realtime}
│   ├── internal/db/migrations/   (0001…0007 из ddl-m0)
│   └── Makefile
├── frontend/                     (Nuxt 3 + Nuxt UI + Pinia)
├── docker-compose.yml
├── Makefile · .gitignore · README.md
```

---

## Срез 1 — Инфра-бут ✅ ГОТОВО (приёмка пройдена)
- [x] `git init` + `.gitignore` (Go, Node, .env, dist)
- [x] `docker-compose.yml`: postgres:16, redis:7 + volumes + healthchecks
- [x] init-SQL Postgres: роли `pms_app` (RLS enforced), `pms_jobs` (`BYPASSRLS`) + default privileges
- [x] `backend/go.mod`; config-слой (env: DB_URL, MIGRATE_DB_URL, REDIS_URL, PORT, SESSION_TTL)
- [x] bun + pgdriver; коннект под ролью `pms_app`
- [x] golang-migrate (embed): миграции 0001–0007 (из [ddl-m0](02-architecture/ddl-m0.md)) в `internal/db/migrations/`
- [x] chi-сервер: `GET /health` (live), `GET /readyz` (db+redis ping)
- [x] root `Makefile`: `up/down/migrate/run/test/dev`
- **Приёмка ✅:** `compose up` → миграции применены → `/health` 200, `/readyz` 200; 10 таблиц; RLS включён на тенант-таблицах; `pms_app` скоупится (0 строк без контекста).
- **River — перенесён в Срез 4** (первая джоба): не мешать драйверы pgx/bun в фундаменте.

## Срез 2 — Контракты (buf + кодген) ✅ ГОТОВО
- [x] `buf.yaml` (lint BASIC — переиспользуем CommandResult) + `buf.gen.yaml` (Go: protoc-gen-go + connect-go)
- [x] proto-файлы: common/filter/view/workspace_service/team_service/sync_service/query_service (issue → M1, ws → Срез 5)
- [x] `buf generate` → Go-стабы (`backend/gen/`); `make proto` таргет
- [x] connect-хендлеры (заглушки Unimplemented через встроенный `Unimplemented*ServiceHandler`) подключены в chi-mux
- [ ] TS-кодген (connect-es → `frontend/gen/`) — **перенесён в Срез 6** (фронт ещё не заскаффолжен)
- [ ] buf breaking-check в CI — при настройке CI
- **Приёмка ✅:** `buf generate` без ошибок; `POST /pms.v1.WorkspaceService/CreateWorkspace` → `{"code":"unimplemented"}` 501; build exit 0.

## Срез 3 — Auth (сессии в Redis) ✅ ГОТОВО
- [x] `Register` (argon2id, PHC) + `Login` → `session:{token}` в Redis (TTL, скользящее продление)
- [x] httpOnly cookie (SameSite=Lax); `Authenticator` middleware → ctx(user); `RequireAuth`
- [x] `Logout` + `DeleteAllForUser` (отзыв всех, `user_sessions:{id}` set)
- [x] **tx-обёртка** `db.WithWorkspace` через `set_config('app.workspace_id', ?, true)` (RLS) — готова, используется в Срезе 4
- [x] rate-limit на login (Redis INCR, 10/мин)
- [~] CSRF: SameSite=Lax покрывает базовый случай; полноценный CSRF-токен для cross-site — отдельная веха (ponytail)
- **Приёмка ✅:** register→201(UUIDv7)+cookie; me+cookie→200, без→401; дубль→409; неверный пароль→401; logout→204; me после logout→401.

## Срез 4 — Домен: workspaces + teams ✅ ГОТОВО
- [x] `changelog.Record` (в одной tx с мутацией); `version` стартует 0
- [x] `WorkspaceService.CreateWorkspace`: workspace + owner-membership + slug-валидация + 2 change_log
- [x] `TeamService.CreateTeam`: `path` (ltree, неявный каст миграцией 0008), key-валидация, `next_issue_number=0`, **дефолтные `workflow_states`** (5 категорий), lead-членство, change_log на весь бандл
- [x] connect-хендлеры с auth-гейтом (`requireActor`) + маппинг ошибок (InvalidArgument/AlreadyExists/Unauthenticated)
- [x] unit-тесты: ltree-метка, полнота дефолтных статусов, key-regex
- [ ] `TeamService.MoveTeam` (пересчёт path, защита от цикла) — перенос в M1 (нужен при реальной реорганизации)
- [ ] in-process event bus + publish дельт — в Срезе 5 (вместе с realtime)
- **Приёмка ✅:** CreateWorkspace→ws+member+2 log; CreateTeam→team+5 states+member+7 log; вложенность `BLD.UI` (ltree); dup key→already_exists; no-auth→unauthenticated; **RLS доказан** (pms_app без ctx→0, с ctx→2); tx-откат при ошибке целостен.

## Срез 5 — Sync-скелет ✅ ГОТОВО
- [x] `SyncService.Bootstrap`: стрим среза (workspace/team/workflow_state/label/member) + курсор (max seq)
- [x] `SyncService.Changes(since)`: дельты из `change_log` по `seq`, с membership-гейтом
- [x] WS-гейтвей (coder/websocket): upgrade+auth (cookie/Bearer/?token=), `Subscribe/Unsubscribe`, ping/pong, RBAC на канал
- [x] `realtime.Publisher`: publish дельты при мутации (после коммита) → Redis pub/sub → fan-out в WS
- [x] membership-авторизация подписки на `ws:{id}` (через `WithWorkspace`)
- [ ] коалесинг дельт (окно ~50–100мс) — перенос в Срез 7 (доводка realtime); live-delta seq=0 (skeleton)
- **Приёмка ✅:** Bootstrap (connect-стрим) отдаёт 5 чанков (1 ws/3 teams/15 states/0 labels/1 member, cursor=11); Changes→4 дельты; **WS: subscribe→CreateTeam→delta доставлена**; non-member→permission_denied 403.

## Срез 6 — Фронт-скелет ✅ ГОТОВО (компиляция + транспорт верифицированы)
- [x] Nuxt 3.21 + Nuxt UI + Tailwind + токены тёмной темы ([theme-tokens](03-frontend/theme-tokens.md), forced dark)
- [x] TS-кодген connect-es v2 (`buf generate` → `frontend/gen/`, protoc-gen-es)
- [x] Pinia нормализованный стор (StoreGateway/CommandBus в `stores/app.ts`)
- [x] connect-es клиент с Bearer-интерсептором (`lib/connect.ts`), WS-клиент (`lib/ws.ts`)
- [x] страница Login (register/login → token); оболочка: switcher + LeftRail (Sections) + ContextualSidebar (дерево команд по ltree-path)
- [x] bootstrap при входе → стор; WS-дельта → debounce-refresh среза; optimistic add команды
- [x] dev-CORS на бэке + token в теле auth (cross-origin SPA)
- **Приёмка ✅:** `nuxt dev` компилит и отдаёт `/`,`/login` (200); CORS-preflight рефлектит connect-заголовки; cross-origin login→token; connect Changes с Origin+Bearer→11 дельт. Bootstrap/WS-логика на том же транспорте, что проверен Go-клиентом в Срезе 5.
- [ ] IndexedDB-персист стора — перенос в M1 (skeleton держит в памяти)
- _Остаётся ручной click-through в браузере (headless-драйвер тут не поднят) — все зависимости пути верифицированы по отдельности._

---

## Definition of Done — M0
- `compose up` поднимает весь стек; миграции авто-применяются.
- Сквозной путь: register → login → создать workspace → создать team → видно во фронте, синкается во второй клиент.
- RLS активен; забытый скоуп не утекает (тест).
- buf-кодген в CI; базовые unit-тесты (нумерация, переход статуса заготовка, path).
- Скелет realtime есть; optimistic UI работает локально.

## Параллелизуемость
- Срезы 1→2→3→4 последовательны (зависимости). Срез 6 (фронт) можно начинать после 2 (кодген) параллельно с 3–5. Срез 5 зависит от 4.
