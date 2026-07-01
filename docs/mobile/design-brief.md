# stdtask — mobile design brief (terminal / Tokyo-Night)

Paste this into Claude Design as the style context. Goal: design **mobile app
screens** that reuse the web feature set, in the desktop terminal aesthetic.
Everything is **monospace (JetBrains Mono)**. Dark only. Feels like a terminal
UI: prompts (`~/`, `#`, `$`), bracket buttons `[ label ]`, ASCII glyphs.

## Palette (exact)

| token     | hex       | use                                   |
|-----------|-----------|---------------------------------------|
| base      | `#0b0c12` | app background                        |
| surface   | `#0f1117` | floating content panel                |
| elevated  | `#16171f` | cards / inputs / popovers             |
| hover     | `#1b1d27` | hover                                 |
| active    | `#232634` | selected row                          |
| subtle    | `#20222e` | thin borders                          |
| text      | `#c5cae0` | body text                             |
| secondary | `#aab0c7` | secondary text                        |
| tertiary  | `#6b7089` | muted text / icons / placeholders     |
| accent    | `#7aa2f7` | primary accent (blue)                 |
| accent2   | `#bb9af7` | purple (keywords, KEY-N)              |
| ok        | `#9ece6a` | success / string (green)              |
| info      | `#7dcfff` | cyan                                  |
| warn      | `#ff9e64` | orange                                |
| danger    | `#f7768e` | red / errors                          |
| bright    | `#eef0fa` | headings / emphasised values          |

## Type

- Family: **JetBrains Mono** everywhere (weights: 400 / 500 / 600 / 700).
- No sans-serif. Sizes: body 14, small 13, labels 13, screen title ~30.
- Border radius: 8–10px. Borders 1px `subtle`.

## Primitive vocabulary (build screens from these)

- **T (text)** — mono text. Color variants: body, dim(`secondary`),
  faint(`tertiary`), bright, accent, ok, danger, warn, info.
- **Prompt** — colored prefix glyph before labels: `~/` (green), `#`, `$`, `>`.
  Used in titles and input rows, e.g. `~/stdtask`, `> email`.
- **Bracket button** — text button rendered as `[ label ]` in accent color.
  This is the default action button (no fill).
- **Primary CTA** — filled accent (`#7aa2f7`) button, base-color bold text,
  radius 8, padding ~13. Only for login / create / send.
- **Chip** — 1px `subtle` border, `elevated` bg, radius 6, small padding.
  Status / assignee / filter. Active chip: border `accent`, bg `active`.
- **Input row** — `elevated` bg, `subtle` border, radius 8, a colored Prompt
  glyph on the left (`>` in green), mono text, `tertiary` placeholder.
- **Spinner** — accent ActivityIndicator.

## Style rules

- Terminal metaphor: prefer glyphs (`[ ]`, `>`, `#`, `~/`, `✓`, `▸`) over icons.
- ASCII checkboxes `[x]` / `[ ]`, ASCII progress bars where lists show progress.
- Flat, dense, left-aligned. Thin borders separate regions, not shadows.
- Color is semantic only (ok/warn/danger/accent), never decorative.

## Existing mobile screens (match these)

Tab bar app. Screens already built — design new ones consistent with them:
- **login** — centered, `~/stdtask` title, `// sign in to your workspace`
  comment line, two prompt input rows (`>`), filled Primary "sign in".
- **(tabs)**: index (my issues), inbox (notifications), profile.
- **create** — quick-create issue modal.
- **issue/[id]** — issue detail + quick actions (status/assignee/priority
  chips, comments).

## Feature set to cover (from web)

Issues (list / detail / quick-create / status·assignee·priority·labels),
inbox / notifications, comments + mentions, my-issues grouped by deadline,
workspaces, activity feed. Design mobile-first flows for these.
