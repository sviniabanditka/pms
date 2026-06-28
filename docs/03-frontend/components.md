# Фронтенд: Компонентная база и библиотеки (S5)

Nuxt UI (Reka UI + Tailwind) как примитивы; поверх — доменные компоненты, переиспользуемые
повсеместно (§5.3). Здесь же фиксируем S5 — выбор фронт-библиотек (открытый из ADR-0003).
Связано: §5.3, §7.2, [view-renderer](view-renderer.md).

---

## 1. Примитивы (Nuxt UI)

Button, Dropdown/Menu, Popover, Tooltip, Modal, Command (палитра), Combobox, Select, Checkbox,
Switch, Avatar, Badge, Tabs, Toast, Skeleton — доступные из коробки (§5.3).

## 2. Доменные компоненты (переиспользуемые)

`IssueRow`, `IssueCard`, `StatusSelect`, `PrioritySelect`, `AssigneePicker`, `LabelPicker`,
`EstimatePicker`, `DatePicker`, `CycleSelect`, `FilterBar`, `DisplayOptionsMenu`, `BurndownChart`,
`ProgressBar`, `PropertyPanel`, `RichTextEditor` (TipTap), `MentionInput`.

Плюс инфраструктурные: `ViewHost` (рендер View, [view-renderer](view-renderer.md)), `LeftRail`,
`ContextualSidebar`, `WorkspaceSwitcher`, `CommandPalette`, `InboxList`.

## 3. S5 — выбор библиотек (ЗАФИКСИРОВАНО)

То, что Nuxt UI не покрывает (§5.3):

| Потребность | Выбор | Почему |
|---|---|---|
| **DnD** | **pragmatic-drag-and-drop** | framework-agnostic, производительный, без vue-специфики |
| **Виртуализация** | **@tanstack/vue-virtual** | списки на тысячи issues (§9.1) |
| **Grid дашбордов** | **grid-layout-plus** | drag/resize виджетов, сохранение в `Dashboard.grid_config` (§6.4) |
| **Графики** | **@unovis/vue** | легче и чище под Linear-минимализм, чем echarts (burndown/velocity/агрегации) |
| **Календарь** | **кастомный** | под минимал-стиль; renderer над тем же query-движком (§6.3) |
| **Таблица (DataGrid)** | **кастомный** | ресайз/реордер/сортировка колонок; Nuxt UI не покрывает |
| **Дерево (Tree)** | **кастомный** (на Reka UI primitives) | дерево команд/доков с рекурсией |
| **Rich text** | **TipTap** (Vue-биндинги) | описания/доки/комментарии; JSON в JSONB |
| **Иконки** | Lucide (монохром, currentColor) | один набор (§5.2) |
| **Роутинг** | файловый роутинг Nuxt | §7.2 |

> Календарь и Timeline — **renderer'ы** (контракт §6.3), не отдельные страницы: потребляют `GroupedResult`.

## 4. Композиция оболочки (§4.1)

Четыре зоны: `WorkspaceSwitcher` (верх рейла) → `LeftRail` (настраиваемые Sections, drag-reorder,
конфиг в `WorkspaceMember.sections_config`) → `ContextualSidebar` (зависит от Section) → `ViewHost`/Doc/Dashboard
(+ верхний тулбар breadcrumbs/Filter/Display/layout, + правый PropertyPanel).

## 5. UX-паттерны (§5.4) — на уровне компонентов

- Optimistic UI (все мутации через command bus).
- Inline-edit прямо в `IssueRow`/`IssueCard`.
- Multi-select + bulk actions (shift/cmd-клик → `BulkUpdateIssues`).
- Keyboard-first: хоткеи (C/A/S/P, J/K) + базовый Cmd-K (quick-switch/create) в MVP, полный — v2.
- Undo на деструктив (toast «Undo», окно перед purge, §ADR-0005).
- Empty states с CTA; скелетоны при первичной загрузке.

## Открытые микро-вопросы

- Точная versions-матрица библиотек — на старте M0 (через find-docs проверить актуальное API).
- Кастомный DataGrid: своя виртуализация колонок vs только строк — решаем при M3 (Table).
- CommandPalette: базовый в MVP (quick-switch+create) — объём команд в палитре уточнить на M3–M4.
