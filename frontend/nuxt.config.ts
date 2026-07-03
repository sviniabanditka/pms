export default defineNuxtConfig({
  modules: ['@nuxt/ui', '@pinia/nuxt'],
  // Storybook запускается автономно (.storybook/main.ts, framework @storybook-vue/nuxt).
  // Модуль @nuxtjs/storybook НЕ подключаем — он ломает `nuxt build`/`dev` на холодном
  // старте (падение в storybook core-server resetQueue). Storybook сам поднимает nuxt.
  // local-first логика — клиентская (guarded import.meta.client); SSR оставляем дефолтным.
  devtools: { enabled: false },
  colorMode: { preference: 'dark', fallback: 'dark', storageKey: 'pms-theme' },
  app: {
    head: {
      titleTemplate: '%s · stdtask', // лендинг переопределяет на '%s' (полный маркетинг-тайтл)
      htmlAttrs: { class: 'dark', lang: 'en' },
      meta: [{ name: 'description', content: 'stdtask — local-first project & issue tracker (ClickUp × Linear).' }],
      link: [
        { rel: 'icon', type: 'image/svg+xml', href: '/favicon.svg' },
        { rel: 'preconnect', href: 'https://fonts.googleapis.com' },
        { rel: 'preconnect', href: 'https://fonts.gstatic.com', crossorigin: '' },
        // Не render-blocking: грузим как preload и переключаем в stylesheet по onload.
        // display=swap + mono-фолбэк (Menlo/monospace) → текст виден сразу, без CLS (оба mono).
        { rel: 'preload', as: 'style', href: 'https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;500;600;700&display=swap', onload: "this.onload=null;this.rel='stylesheet'" },
      ],
      noscript: [
        { innerHTML: '<link rel="stylesheet" href="https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;500;600;700&display=swap">' },
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
