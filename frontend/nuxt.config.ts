export default defineNuxtConfig({
  modules: ['@nuxt/ui', '@pinia/nuxt'],
  // local-first логика — клиентская (guarded import.meta.client); SSR оставляем дефолтным.
  devtools: { enabled: false },
  colorMode: { preference: 'dark', fallback: 'dark', storageKey: 'pms-theme' },
  app: {
    head: {
      titleTemplate: '%s · stdtask', // лендинг переопределяет на '%s' (полный маркетинг-тайтл)
      htmlAttrs: { class: 'dark' },
      // viewport: без него телефон рендерит desktop-вёрстку зумом. viewport-fit=cover — под вырезы.
      meta: [
        { name: 'viewport', content: 'width=device-width, initial-scale=1, viewport-fit=cover' },
        { name: 'theme-color', content: '#0b0c12' },
        { name: 'mobile-web-app-capable', content: 'yes' },
        { name: 'apple-mobile-web-app-status-bar-style', content: 'black-translucent' },
      ],
      link: [
        { rel: 'icon', type: 'image/svg+xml', href: '/favicon.svg' },
        { rel: 'manifest', href: '/manifest.webmanifest' },
        { rel: 'apple-touch-icon', href: '/favicon.svg' },
        { rel: 'preconnect', href: 'https://fonts.googleapis.com' },
        { rel: 'preconnect', href: 'https://fonts.gstatic.com', crossorigin: '' },
        { rel: 'stylesheet', href: 'https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;500;600;700&display=swap' },
      ],
    },
  },
  css: ['~/assets/css/main.css'],
  runtimeConfig: {
    public: {
      apiBase: 'http://localhost:8080',
      wsBase: 'ws://localhost:8080',
    },
  },
  compatibilityDate: '2025-01-01',
})
