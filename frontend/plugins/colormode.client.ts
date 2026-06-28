// Приложение dark-only: жёстко фиксируем dark, чтобы Nuxt UI всегда брал dark-палитру.
export default defineNuxtPlugin(() => {
  const cm = useColorMode()
  cm.preference = 'dark'
  cm.value = 'dark'
  if (import.meta.client) document.documentElement.classList.add('dark')
})
