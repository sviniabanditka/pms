import { defineConfig } from 'vitest/config'
import { fileURLToPath } from 'node:url'

// Алиасы, чтобы импортировать стор/lib вне Nuxt: #imports → стаб, ~ и @ → корень.
export default defineConfig({
  resolve: {
    alias: {
      '#imports': fileURLToPath(new URL('./tests/stubs/imports.ts', import.meta.url)),
      '~': fileURLToPath(new URL('.', import.meta.url)),
      '@': fileURLToPath(new URL('.', import.meta.url)),
    },
  },
})
