# M6 — план (Customization)

## Часть 1: Custom fields
Базовые типы (ADR-0004): text/number/date/checkbox/select/multiselect/url.
(people/relationship — отложу, нужны пикеры; formula/rollup — вне MVP.)

### Срез 1 — backend ✅ ГОТОВО
- [x] миграции `custom_fields` (def) + `custom_field_values` (value jsonb) + RLS
- [x] модели; proto `custom_field_service` (Create/Archive) + `IssueService.SetCustomFieldValue` → кодген
- [x] сервис: CreateCustomField/ArchiveCustomField; issues.SetCustomFieldValue (upsert)
- [x] bootstrap: `custom_field` + `custom_field_value`
- **Приёмка:** создать поле; задать значение на issue; RLS.

### Срез 2 — frontend ✅ ГОТОВО
- [x] создать поле (имя+тип, для select — опции) — форма в сайдбаре
- [x] detail-панель: рендер каждого поля по типу + правка значения (оптимистично)
- [x] показ значений; (фильтр по custom field — отложу, D)
- **Приёмка:** создал поле → в панели редактируется → значение держится.

## Часть 2: Dashboards + Widgets ✅ ГОТОВО
> IA-правка: Dashboard — **отдельный раздел в левом рейле** (как Tasks), персональный, со своими виджетами.
> Рейл = свитчер воркспейса → разделитель → разделы (Dashboard, Tasks; далее Calendar/Chats); раздел переключает весь контекст.
- [x] backend: dashboards/dashboard_widgets + DashboardService (Create/AddWidget/UpdateWidget/RemoveWidget) + bootstrap
- [x] frontend: секция Dashboards, режим дашборда, виджеты note/stat/view (chart — заглушка)
- [x] StatWidget = count по Filter AST; ViewWidget = issues сохранённого view
- Grid drag/resize (grid-layout-plus) — отложен (D35); chart — D36

## Часть 3: Настраиваемый левый рейл ✅ ГОТОВО
- [x] backend: WorkspaceService.UpdateSectionsConfig (per-user sections_config) + publish
- [x] frontend: currentUserId (+members bucket), railSections по конфигу, ⚙-панель (видимость + ↑↓ порядок)
- [x] SectionsConfig []byte→json.RawMessage (фикс base64 для jsonb)
- **Приёмка:** спрятать/переставить раздел → применяется, переживает reload.
