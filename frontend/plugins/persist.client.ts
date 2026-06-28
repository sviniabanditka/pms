import { useAppStore } from '~/stores/app'

// Гидратация стора из IndexedDB при старте (мгновенный рендер из кэша до bootstrap).
export default defineNuxtPlugin(async () => {
  const store = useAppStore()
  await store.hydrate()
})
