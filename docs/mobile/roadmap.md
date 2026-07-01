# Mobile App — Roadmap

Lean MVP, online-first. Each phase ships something runnable on a real device.
See [README.md](./README.md) for stack/architecture decisions.

## Phase M0 — Scaffold & auth (foundation) ✅ code done

Goal: app boots on iOS+Android sim, logs in, makes one authenticated RPC.

- [x] `mobile/` Expo app (SDK 57, TS, expo-router). Standalone (no root workspace
      yet — ponytail: add only when a shared package appears).
- [x] `buf.gen.yaml`: second `protoc-gen-es` target `out: mobile/gen` (gitignored,
      run `buf generate`). 22 services generated.
- [x] `mobile/lib/connect.ts`: `createConnectTransport` (JSON format for RN-safe
      fetch) + Bearer interceptor (token cached, from `expo-secure-store`) +
      Unauthenticated→`setOnUnauthorized` guard. `lib/auth.ts` (login/me/logout/
      myWorkspaces), `lib/authState.ts` (useSyncExternalStore auth gate).
- [x] Screens: `_layout.tsx` (boot → loadToken → /me → gate), `login.tsx`,
      `index.tsx` (workspaces + a live connect `query.search` self-check).
- [x] `apiBase` via `EXPO_PUBLIC_API_BASE` (`.env.example`); brand: name/scheme `stdtask`.
- [x] **Verified:** `tsc --noEmit` clean; `expo export -p ios` bundles (1236 modules —
      connect-web/protobuf/gen all resolve under RN/Hermes).
- **Pending (needs a device/sim — user):** run on iOS sim + Android, log in against
  the dev Go server (set LAN IP), confirm the connect self-check shows `connect OK`.
  This closes the roadmap "verify connect-web on device early" gate.

## Phase M1 — Navigation shell + My Issues ✅ code done

Goal: the core daily list.

- [x] Tabs nav (expo-router `(tabs)`): Issues · Profile. (Inbox=M3, Create=M2 —
      added later.)
- [x] Workspace switch + persist (`lib/workspace.ts`, useSyncExternalStore +
      storage `pms_ws`). `GET /me/workspaces` (REST) for the list.
- [x] My Issues (assigned / created toggle) via **`QueryService.RunView`** (unary,
      not bootstrap-stream): ad-hoc `ViewDefinition` filter `assignee|creator EQ me`
      → `grouped_result` JSON array of issues. `lib/api.ts`. FlatList (FlashList
      later), pull-to-refresh, loading/error/empty + retry.
- [x] `lib/storage.ts` platform shim (SecureStore/localStorage); `getCurrentUserId`
      from `/auth/me`; tsconfig `~/*` → root alias.
- [x] **Verified:** tsc clean; RunView shape validated e2e (assigned 168 / created
      170 issues on seed); `expo export -p ios` bundles.
- **Deferred to a later phase (need a metadata cache):** status/assignee/team
  *names* — RunView returns only ids, so rows show identifier + title + priority.
  A small unary "workspace metadata" fetch (states/teams/users/labels) comes with
  M2 (issue detail needs it too). Also: TextDecoder assumed present on Hermes
  (SDK 57) — add a polyfill if a device errors.
- **Exit (device — user):** see/scroll my issues, toggle assigned/created,
  switch workspace, pull-to-refresh.

## Phase M2 — Issue detail + quick actions + create ✅ code done

Goal: act on an issue from the phone.

- [x] **Metadata cache** (`lib/meta.ts`) — the M1 gap. New backend unary
      `GET /me/workspaces/{id}/meta` (member-only) → teams/states/users/labels
      (`workspaces.Meta`). Gives names + colors for rows and pickers. Issues list
      now shows state name/color + assignee name.
- [x] Issue detail (`app/issue/[id].tsx`): reads the cached issue (RunView returns
      the full row incl. description — no GetIssue RPC needed), shows identifier/
      title/description + comments (`CommentService.GetComments`).
- [x] Quick actions: status (`UpdateIssueState`), assignee (`AssignIssue`), add
      comment (`AddComment`) — modal pickers, cache updated on success.
- [x] Quick-create (`app/create.tsx`, modal): team + title + priority + optional
      assignee → `CreateIssue` (client-gen uuid).
- [x] **Verified:** tsc clean; full mutation chain e2e (create DATA-430 → state →
      assign → comment → GetComments); `expo export` bundles (1244 modules);
      backend `TestWorkspaces_Meta` green.
- **Notes / deferred:** base_version isn't enforced and CommandResult returns no
  new_version → we don't track version (pass current, ignore). Description rendered
  as plain text (markdown render later). Optimistic UI is "call → update cache";
  no rollback framework (errors surface, refetch on next open). Create returns to
  the list which reloads on pull-refresh (no auto-refresh yet). Labels fetched but
  not shown on detail yet.
- **Exit (device — user):** open an issue, change status/assignee, comment, create
  a new issue.

## Phase M3 — Inbox / notifications ✅ code done

Goal: see what needs attention.

- [x] Backend (net-new, unary — NotificationService has no List, only stream):
      `GET /me/workspaces/{id}/notifications` (member-only) → `notifications.List`
      (recent 50, recipient-scoped). Integration test `TestNotifications_List`.
- [x] Filter engine: added `"id"` builtin column → RunView can fetch a single issue
      by id (needed when a notification/push targets an issue not in the list cache).
- [x] Inbox tab (`app/(tabs)/inbox.tsx`): list, unread dot + count, optimistic mark
      read on open (`NotificationService.MarkRead`), mark all read, pull-to-refresh.
- [x] Tap notification → `issueById` (RunView id filter) caches the issue →
      deep-link `/issue/{entityId}`.
- [x] **Verified:** tsc clean; e2e (list 50 / 27 unread → MarkRead 27→26 →
      MarkAllRead →0; single-issue by id → INFRA-3); `expo export` 1245 modules;
      TestNotifications_List green.
- **Deferred:** notification rows show type + entityType (+ actor name), not the
  issue identifier/title (would need a per-entity lookup); no push yet (M4).
- **Exit (device — user):** open Inbox, see notifications, tap → issue opens, mark
  read / mark all read.

## Phase M4 — Push (the native payoff)

Goal: pings arrive on the lock screen.

- **Backend:**
  - [ ] migration `device_tokens` (user_id, expo_push_token, platform, last_seen).
  - [ ] `POST /api/devices` register / `DELETE /api/devices/{token}`.
  - [ ] `PushNotifyWorker` (River): on notification insert → send via Expo Push API
        to the user's tokens. Best-effort, batched, prune dead tokens (DeviceNotRegistered).
- **Mobile:**
  - [ ] `expo-notifications`: permission prompt, get Expo push token, register on login,
        unregister on logout.
  - [ ] Foreground + background handlers; tap → deep-link to issue (reuse M3 routing).
- **Exit:** assign/comment on web → push on the phone → tap opens the issue.

## Phase M5 — Polish & ship

- [ ] SQLite cache persister (warm start + offline read of last snapshot).
- [ ] Wire WS (`lib/ws.ts` equivalent) for live list updates (optional, refetch-on-focus fallback).
- [ ] App icon, splash, theme (Tokyo-Night to match), error boundary, Sentry (optional).
- [ ] EAS Build profiles (dev/preview/prod), internal TestFlight + Android internal track.
- [ ] Store assets (screenshots, descriptions, privacy).
- **Exit:** installable build in TestFlight / Play internal testing.

## Later (post-MVP, gated by demand)

- Chat (channels/threads), Calendar, Time-tracking, Docs (read), Meetings (join).
- Full local-first: SQLite mirror + delta-sync engine port + offline-write outbox + conflict resolution.
- Biometric unlock, long-lived device tokens, multi-account.

## Risks / watch

- RN `fetch` server-streaming is weak → MVP unary-only; live = WS/refetch.
- connect-web in RN: verify binary/Connect-protocol works on device early (M0 gate).
- Metro + monorepo node_modules: isolate via workspaces; set `watchFolders`/`nodeModulesPaths` if Metro chokes.
- Push tokens are per-install and rotate → always re-register on login + handle DeviceNotRegistered.
