export default defineNuxtConfig({
  modules: ['@nuxt/ui', '@pinia/nuxt'],
  // local-first логика — клиентская (guarded import.meta.client); SSR оставляем дефолтным.
  devtools: { enabled: false },
  colorMode: { preference: 'dark', fallback: 'dark', storageKey: 'pms-theme' },
  app: { head: { htmlAttrs: { class: 'dark' } } },
  css: ['~/assets/css/main.css'],
  runtimeConfig: {
    public: {
      apiBase: 'http://localhost:8080',
      wsBase: 'ws://localhost:8080',
    },
  },
  compatibilityDate: '2025-01-01',
})
