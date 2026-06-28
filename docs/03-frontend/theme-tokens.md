# Тёмная тема: токены и палитра (code-ready)

Конкретные значения под [design-system](design-system.md). Только тёмная тема в MVP,
но структура токенов допускает светлую без переделок (§5.2). Статусы/лейблы хранят `color_token`
(имя), не hex — здесь имена резолвятся в значения.

Стек: Tailwind theme + Nuxt UI `app.config.ts`. Эстетика — Linear-дарк (спокойный, низкий контраст фонов, функциональный цвет).

---

## 1. Семантические токены (hex)

| Токен | Hex | Назначение |
|---|---|---|
| `bg-base` | `#0b0d0f` | основной фон приложения |
| `bg-surface` | `#111316` | панели/сайдбары |
| `bg-elevated` | `#181b1f` | поповеры/модалки/карточки |
| `bg-hover` | `#1f2329` | hover-состояние строк |
| `bg-active` | `#262b32` | выбранная строка/таб |
| `border-subtle` | `#23272e` | тонкие разделители |
| `border-strong` | `#333a42` | акцентные границы/инпуты |
| `text-primary` | `#e6e9ed` | основной текст |
| `text-secondary` | `#9aa3ad` | вторичный |
| `text-tertiary` | `#6b747e` | плейсхолдеры/метаданные |
| `accent` | `#5e6ad2` | акцент (Linear-индиго) |
| `accent-hover` | `#6e79e0` | hover акцента |
| `accent-muted` | `#2a2f57` | фон акцентных бэйджей |
| `focus-ring` | `#5e6ad2` | кольцо фокуса (a11y) |
| `danger` | `#e5484d` | деструктив/ошибки |
| `success` | `#3fb950` | успех |
| `warning` | `#d29922` | предупреждение |

---

## 2. Статусы (по категориям, §workflow-states)

`color_token` статуса ссылается сюда; категория задаёт дефолт, имя статуса — произвольно.

| Категория | token | Hex | Иконка (тип) |
|---|---|---|---|
| backlog | `status-backlog` | `#6b747e` | пунктирный круг |
| unstarted | `status-unstarted` | `#9aa3ad` | пустой круг |
| started | `status-started` | `#d29922` | круг-прогресс |
| completed | `status-completed` | `#5e6ad2` | заполненный круг ✓ |
| canceled | `status-canceled` | `#6b747e` | круг ✕ |

---

## 3. Приоритеты

| Приоритет | token | Hex |
|---|---|---|
| urgent (4) | `priority-urgent` | `#e5484d` |
| high (3) | `priority-high` | `#e08c3f` |
| medium (2) | `priority-medium` | `#d29922` |
| low (1) | `priority-low` | `#6b747e` |
| none (0) | `priority-none` | `#4a525b` |

---

## 4. Палитра лейблов (`label-*`, выбор при создании)

| token | Hex | token | Hex |
|---|---|---|---|
| `label-gray` | `#8b949e` | `label-green` | `#3fb950` |
| `label-red` | `#e5484d` | `label-teal` | `#2dd4bf` |
| `label-orange` | `#e08c3f` | `label-blue` | `#4493f8` |
| `label-yellow` | `#d29922` | `label-indigo` | `#5e6ad2` |
| `label-lime` | `#a5d64c` | `label-purple` | `#a371f7` |
| `label-pink` | `#db61a2` | `label-brown` | `#a18072` |

Бэйдж лейбла: текст `label-*`, фон — тот же hex при ~12–16% alpha.

---

## 5. Типографика

| Шаг | px / line-height | Использование |
|---|---|---|
| xs | 12 / 16 | метаданные, бэйджи |
| sm | 13 / 18 | строки списков (дефолт-плотность) |
| base | 14 / 20 | основной текст |
| md | 16 / 24 | заголовки секций |
| lg | 20 / 28 | заголовки страниц |
| xl | 24 / 32 | крупные заголовки |

Шрифт: `Inter, ui-sans-serif, system-ui, …`. Числа — `font-variant-numeric: tabular-nums` в таблицах.

## 6. Spacing / радиусы

- Spacing — шкала 4-pt: `0,4,8,12,16,20,24,32,40,48`.
- Радиусы: `sm 4px`, `md 6px`, `lg 8px` (карточки/модалки). Минимализм — без больших скруглений.
- Тени: только `overlay` (`0 8px 24px rgba(0,0,0,.4)`) для поповеров/модалок; в плоском UI больше нет.

## 7. Плотность (строки списков)

| density | высота строки |
|---|---|
| `comfortable` | 40px |
| `compact` | 32px |

---

## 8. Реализация — Tailwind + Nuxt UI

`tailwind.config.ts` (фрагмент):
```ts
export default {
  theme: { extend: { colors: {
    base: '#0b0d0f', surface: '#111316', elevated: '#181b1f',
    hover: '#1f2329', active: '#262b32',
    'border-subtle': '#23272e', 'border-strong': '#333a42',
    'text-primary': '#e6e9ed', 'text-secondary': '#9aa3ad', 'text-tertiary': '#6b747e',
    accent: { DEFAULT: '#5e6ad2', hover: '#6e79e0', muted: '#2a2f57' },
    danger: '#e5484d', success: '#3fb950', warning: '#d29922',
    // status-*, priority-*, label-* по таблицам выше
  },
  borderRadius: { sm: '4px', md: '6px', lg: '8px' },
  fontFamily: { sans: ['Inter', 'ui-sans-serif', 'system-ui'] },
}}}
```

`app.config.ts` (Nuxt UI):
```ts
export default defineAppConfig({
  ui: {
    primary: 'accent',          // маппинг на акцентный токен
    gray: 'neutral',
    // тёмная тема принудительно; переключатель отключён в MVP
  },
})
```
`nuxt.config.ts`: `colorMode: { preference: 'dark', fallback: 'dark' }` (без UI-переключателя).

## Замечания
- Точные hex — стартовая палитра; финальная калибровка контраста (WCAG AA) — на M8 visual QA (§roadmap).
- Светлая тема: добавить второй набор значений под тем же именам токенов — компоненты не трогаются.
- `color_token` в БД (статусы/лейблы) = ключи из §2/§4; рендер резолвит в hex по этой таблице.
