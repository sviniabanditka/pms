# Mobile App — Architecture & Decisions

Native mobile client for stdtask. Separate from the web app (the web app stays
desktop-only — the responsive-web attempt was reverted, see debt-ledger §B
`MOBILE`).

## Decisions (locked 2026-07-01)

| Fork | Choice | Why |
|---|---|---|
| **Stack** | **React Native + Expo** (TS) | Reuse proto/connect codegen + domain types from web; same language as the stack; EAS cloud builds (iOS without a Mac), expo-notifications (push), expo-sqlite (cache). |
| **Repo** | **Monorepo — `mobile/`** | Atomic `proto + web client + mobile client` commits; shared `buf.gen` and (later) shared core logic. |
| **MVP scope** | **Lean** | Native wins on push + fast reaction ("ping → look → act → close"). Avoid the "everything on a small screen" trap that killed responsive web. Chat/calendar = phase 2. |
| **Data** | **Online-first, SQLite cache** | No delta-sync engine port yet. SQLite (not in-memory) so we can grow into full local-first later without a rewrite. Offline = read last snapshot; no offline write in MVP. |

Rejected: Flutter (zero TS/Go reuse, weak ConnectRPC story), KMP (two UI layers,
overkill for solo), Capacitor/web-wrap (same bad UX we just reverted).

## What already exists on the backend (no change needed)

- **Auth via Bearer**: `Authenticator` accepts `cookie | Bearer | ?token=`
  (`internal/auth/middleware.go`). `POST /login` already returns the session
  token in the JSON body (`internal/auth/http.go:74` →
  `{id,email,name,token}`). Mobile: POST /login → store token in
  `expo-secure-store` → send `Authorization: Bearer <token>` on every RPC.
  `GET /me` validates. `POST /logout` revokes.
- **ConnectRPC + buf**: every service is already generated to TS
  (`frontend/gen`, protoc-gen-es v2). Mobile reuses the same `.proto` + codegen.
- **Realtime**: WS delta stream (`frontend/lib/ws.ts`) — optional in MVP
  (refetch-on-focus is enough; wire WS in phase 2).

## What's net-new on the backend

- **Push infrastructure** (only backend addition for MVP):
  - `device_tokens` table (user_id, workspace_id?, expo_push_token, platform,
    created_at, last_seen) — no RLS or per-user.
  - `POST /api/devices` register / `DELETE /api/devices/{token}` unregister.
  - Send via **Expo Push API** from the notification-creation path — a River
    worker (`PushNotifyWorker`) reading `change_log`/notification inserts, same
    pattern as `WebhookDispatchWorker`. Best-effort, batched.
  - ponytail: use Expo's hosted push service (no direct APNs/FCM certs) until
    volume justifies self-hosting.

## Architecture

```
pms/                          (monorepo)
├── backend/                  Go ConnectRPC (unchanged + push)
├── frontend/                 Nuxt web (unchanged)
├── proto/                    shared .proto (source of truth)
├── buf.gen.yaml              + second es target → mobile/gen
└── mobile/                   ← Expo RN app (TS)
    ├── app/                  expo-router file-based screens
    ├── gen/                  generated connect-es clients (from buf)
    ├── lib/                  connect transport + auth + sqlite cache
    └── components/
```

**Networking.** `@connectrpc/connect` + `@connectrpc/connect-web`
(`createConnectTransport` over `fetch` — works in RN for **unary** RPCs; the MVP
is all unary). Auth interceptor injects `Bearer` from SecureStore; on
`Unauthenticated` → drop token, route to login. Mirror of `frontend/lib/connect.ts`.
> Caveat: RN `fetch` has poor server-streaming support. MVP uses unary only;
> live updates come from the WS channel (phase 2) or refetch-on-focus, not
> connect streams.

**Data layer.** TanStack Query (React Query) for fetch/cache/retry/refetch.
Persist the query cache to **expo-sqlite** (`@tanstack/query-async-storage-persister`
or a thin SQLite persister) → warm start + offline read of the last snapshot.
No custom store, no delta-sync engine in MVP.

**Navigation.** `expo-router` (file-based, deep-link ready — needed for push
tap-through to an issue).

**UI.** RN primitives + **nativewind** (Tailwind syntax) to keep the web mental
model. `@shopify/flash-list` for long issue lists. No heavy UI kit.

**Codegen sharing.** Add a second `protoc-gen-es` target `out: mobile/gen` to
`buf.gen.yaml` (laziest — no refactor). Dedupe into a shared `packages/proto`
workspace later if web+mobile drift becomes annoying.

## MVP feature scope (lean) → existing RPCs

| Screen | RPC / endpoint |
|---|---|
| Login + workspace switch | `POST /login`, `WorkspaceService`, `QueryService.Bootstrap` |
| Inbox / notifications | `NotificationService` (list, mark read) |
| My issues (assigned / created) | `IssueService` / `QueryService` |
| Issue detail | `IssueService.GetIssue`, `CommentService` |
| Quick actions (status / assignee / comment) | `IssueService.UpdateIssue`, `CommentService.Create` |
| Quick create issue | `IssueService.CreateIssue` |
| Push (tap → issue) | **new** `device_tokens` + Expo Push + expo-router deep link |

**Out of MVP:** chat, calendar, time-tracking, docs, meetings, dashboards,
offline-write / full local-first. → phase 2+.

## Open questions

- Session TTL: reuse the web session token (current TTL) or add a long-lived
  user-scoped "device token"? MVP: reuse session token + silent re-login;
  revisit if sessions expire too aggressively on mobile.
- Workspace scoping of push: one device → many workspaces. Register token per
  (user) and include workspace in the payload; filter client-side.

See [roadmap.md](./roadmap.md) for phased delivery.
