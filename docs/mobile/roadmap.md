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

## Phase M2 — Issue detail + quick actions + create

Goal: act on an issue from the phone.

- [ ] Issue detail: title, description (markdown render), status, assignee, labels, comments.
- [ ] Quick actions: change status, reassign, add comment (`IssueService.UpdateIssue`, `CommentService.Create`).
- [ ] Quick-create issue (team picker + title + optional assignee) → `IssueService.CreateIssue`.
- [ ] Optimistic UI via React Query mutations; rollback on error.
- **Exit:** full "look → act → close" loop works.

## Phase M3 — Inbox / notifications

Goal: see what needs attention.

- [ ] Inbox screen (`NotificationService`): list, unread badge, mark read / mark all.
- [ ] Tap notification → deep-link to the issue (expo-router).
- **Exit:** inbox mirrors web, tap-through navigates.

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
