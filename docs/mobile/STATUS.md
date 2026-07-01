# Mobile — current status (2026-07)

> Supersedes the lean-MVP framing in `roadmap.md`. Direction changed: the user
> wants **full desktop parity** (Linear-style) + a modern **liquid-glass terminal**
> design, not a lean 2-screen client.

## Architecture (the foundation)

- **Local-first store** (`mobile/lib/store.ts`): holds the ENTIRE workspace in
  memory (all 30 entity types — issues, teams, states, labels, cycles, docs,
  channels, dashboards, meetings, calendar events, custom fields, time entries,
  …), mirroring the web Pinia store.
  - Initial load: `GET /me/workspaces/{id}/snapshot` (unary, reuses
    `Sync.Bootstrap`) — the bootstrap stream is RN-hostile, so we made a unary
    equivalent.
  - Deltas: `SyncService.Changes(cursor)` → changed entity types →
    `GET /me/workspaces/{id}/snapshot?only=t1,t2`.
  - Reactive via `useSyncExternalStore`; persisted to SQLite (native) /
    localStorage (web) for warm-start + offline. Selectors under `S.*`.
- `lib/meta.ts` is a thin adapter over the store (keeps `stateById`/`userById`/… API).
- `lib/api.ts`: issues filtered locally from the store; mutations do optimistic
  `patchIssue`; comments/notifications/devices still go over REST/RPC.

## Design system

- Terminal + **liquid-glass**: `lib/glass.tsx` (`Glass` blurred panels with top
  sheen + glowing edge, `Backdrop` color washes, `Avatar`, `Pill`, `GlowDot`),
  `lib/theme.ts` (exact desktop palette + JetBrains Mono), `lib/ui.tsx`
  (`T`/`Bracket`/`Primary`/`Chip`/`Prompt`), `components/Screen.tsx` (shared shell:
  glass header + slide-in `Sidebar` + centralized `sectionRoute`).
- Nav = slide-in **Sidebar** (all sections + teams tree), like the desktop rail.
  Bottom tab bar hidden.

## Screens built (all on the store)

issues (my, assigned/created) · team/[id] (list grouped by status + board columns) ·
issue/[id] (detail + status/assignee pickers + comments) · create · inbox ·
cycles → cycle/[id] · docs → doc/[id] · chat (channel list) · meetings · calendar
(agenda) · dashboards (widgets) · reports (time) · profile/workspace-switch.

## Backend additions this rebuild

`/me/workspaces/{id}/snapshot` (+`?only=`), `/me/workspaces/{id}/meta`,
`/me/workspaces/{id}/notifications`, `/me/devices`, filter builtin `"id"`,
`PushDispatchWorker` + `device_tokens` (M4). All with integration tests.

## Running

- `cd mobile && npx expo start --ios` (Expo Go) — fastest.
- Native build: `brew install cocoapods` then `npx expo run:ios`.
- **babel.config.js is required** (reanimated v4 worklets plugin) — without it the
  native build is a black screen. After babel changes, `expo start -c`.
- Simulator reaches the Go server at `localhost:8080`; real device needs
  `EXPO_PUBLIC_API_BASE=http://<LAN-IP>:8080` in `mobile/.env`.

## Known gaps / next candidates

- Chat: only a channel list — no message history/compose (needs a messages RPC/endpoint).
- Issues: no filters UI yet; board has no drag-drop (tap-to-open only).
- Docs: read-only (no editing). Dashboards: widget list only (no rendered metrics).
- Calendar: agenda list (no month grid). Search (⌘K) not wired.
- Push: needs an EAS `projectId` (`eas init`) to actually deliver.
- Web persist may hit the localStorage quota with 2000+ issues (native SQLite is fine).
