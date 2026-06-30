import { defineConfig } from 'vitest/config'
import { fileURLToPath } from 'node:url'
import vue from '@vitejs/plugin-vue'

// Алиасы, чтобы импортировать стор/lib вне Nuxt: #imports → стаб, ~ и @ → корень.
// plugin-vue — чтобы монтировать .vue-компоненты (@vue/test-utils + happy-dom).
export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '#imports': fileURLToPath(new URL('./tests/stubs/imports.ts', import.meta.url)),
      '~': fileURLToPath(new URL('.', import.meta.url)),
      '@': fileURLToPath(new URL('.', import.meta.url)),
    },
  },
})
