// Регистрация offline-SW (только клиент). Кэширует shell+ассеты для оффлайна;
// данные приложения уже в IndexedDB (persist.client → store.hydrate).
export default defineNuxtPlugin(() => {
  if (!import.meta.client || !('serviceWorker' in navigator)) return
  if (import.meta.dev) return // в dev SW мешает HMR (stale-кэш) — только прод
  window.addEventListener('load', () => {
    navigator.serviceWorker.register('/sw.js').catch(() => {})
  })
})
