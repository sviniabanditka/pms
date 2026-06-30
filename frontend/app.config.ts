export default defineAppConfig({
  ui: {
    primary: 'blue', // терминальный акцент (Tokyo-Night blue ≈ #7aa2f7)
    gray: 'zinc', // нейтральный тёмный
    // все кнопки бордерлес/плоские (как «mark all read»): по умолчанию ghost + gray
    button: { default: { variant: 'ghost', color: 'gray', size: 'xs' } },
    // все плавающие поверхности = фон основного контента (surface).
    // dark:-вариант ОБЯЗАТЕЛЕН: дефолт `dark:bg-gray-900` иначе побеждает в twMerge при .dark.
    modal: { background: 'dark:bg-surface' },
    slideover: { background: 'dark:bg-surface' },
    card: { background: 'dark:bg-surface' },
    popover: { background: 'dark:bg-surface' },
    dropdown: { background: 'dark:bg-surface', item: { active: 'bg-hover text-bright', inactive: 'text-secondary' } },
    select: { background: 'dark:bg-surface' },
    selectMenu: { background: 'dark:bg-surface', option: { active: 'bg-hover', selected: 'bg-active' } },
    commandPalette: { background: 'dark:bg-surface' },
    contextMenu: { background: 'dark:bg-surface' },
  },
})
