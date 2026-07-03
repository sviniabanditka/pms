import type { Preview } from '@storybook-vue/nuxt'
import '~/assets/css/main.css'

// Terminal-тёмный фон + отступ для всех стори (Nuxt UI + наш стиль рендерятся 1-в-1).
const surfaceCss = `
.storybook-surface { padding: 1.25rem; min-height: 100%; background: var(--ui-bg, #0b0e14); color: var(--ui-text, #c8d0e0); font-family: 'JetBrains Mono', ui-monospace, monospace; }
`
if (typeof document !== 'undefined' && !document.getElementById('sb-surface')) {
  const el = document.createElement('style')
  el.id = 'sb-surface'
  el.textContent = surfaceCss
  document.head.appendChild(el)
  document.documentElement.classList.add('dark')
}

const preview: Preview = {
  parameters: {
    backgrounds: { disable: true },
    controls: { matchers: { color: /(background|color)$/i, date: /Date$/i } },
  },
}
export default preview
