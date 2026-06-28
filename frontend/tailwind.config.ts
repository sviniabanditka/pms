import type { Config } from 'tailwindcss'

// Семантические токены тёмной темы — подобраны под Linear (см. docs/03-frontend/theme-tokens.md).
export default <Partial<Config>>{
  darkMode: 'class',
  theme: {
    extend: {
      colors: {
        base: '#090909', // чёрный фон (рейл/сайдбар/топбар)
        surface: '#141416', // плавающая панель контента
        elevated: '#1e1e22', // карточки/поповеры/инпуты
        hover: '#202024',
        active: '#2a2a30', // выбранный пункт
        subtle: '#26262b', // тонкие бордеры
        secondary: '#b4b8c0', // вторичный текст
        tertiary: '#7c828c', // приглушённый текст/иконки
        accent: '#5e6ad2',
      },
      borderRadius: { sm: '6px', md: '8px', lg: '12px', xl: '16px' },
    },
  },
}
