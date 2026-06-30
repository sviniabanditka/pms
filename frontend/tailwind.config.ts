import type { Config } from 'tailwindcss'

// Семантические токены тёмной темы — терминальный стиль (Tokyo-Night), синк с лендингом.
export default <Partial<Config>>{
  darkMode: 'class',
  theme: {
    extend: {
      colors: {
        base: '#0b0c12', // фон (рейл/сайдбар/топбар)
        surface: '#0f1117', // плавающая панель контента
        elevated: '#16171f', // карточки/поповеры/инпуты
        hover: '#1b1d27',
        active: '#232634', // выбранный пункт
        subtle: '#20222e', // тонкие бордеры
        secondary: '#aab0c7', // вторичный текст
        tertiary: '#6b7089', // приглушённый текст/иконки
        accent: '#7aa2f7', // основной акцент (blue)
        accent2: '#bb9af7', // фиолетовый (KEY-N, keyword)
        ok: '#9ece6a', // success/string (green)
        info: '#7dcfff', // cyan
        warn: '#ff9e64', // orange
        danger: '#f7768e', // red
        bright: '#eef0fa', // яркий текст (заголовки/значения)
      },
      fontFamily: {
        mono: ['JetBrains Mono', 'ui-monospace', 'SFMono-Regular', 'Menlo', 'monospace'],
      },
      borderRadius: { sm: '6px', md: '8px', lg: '12px', xl: '16px' },
    },
  },
}
