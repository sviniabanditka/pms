// Точная палитра десктопа (frontend/tailwind.config.ts) + JetBrains Mono —
// терминальный вид как в вебе.
export const C = {
  base: '#0b0c12', // фон
  surface: '#0f1117', // плавающая панель
  elevated: '#16171f', // карточки/инпуты/поповеры
  hover: '#1b1d27',
  active: '#232634', // выбранный пункт
  subtle: '#20222e', // тонкие бордеры
  text: '#c5cae0', // основной текст
  secondary: '#aab0c7',
  tertiary: '#6b7089', // приглушённый
  accent: '#7aa2f7', // blue
  accent2: '#bb9af7', // purple
  ok: '#9ece6a', // green
  info: '#7dcfff', // cyan
  warn: '#ff9e64', // orange
  danger: '#f7768e', // red
  bright: '#eef0fa', // заголовки/значения
}

// семейства JetBrains Mono (fontWeight не переключает файл — нужен явный family)
export const F = {
  r: 'JetBrainsMono_400Regular',
  m: 'JetBrainsMono_500Medium',
  sb: 'JetBrainsMono_600SemiBold',
  b: 'JetBrainsMono_700Bold',
}

export const RADIUS = 10
