import { ref } from 'vue'

// Глобальный лайтбокс для картинок (клик по картинке → модалка во весь экран).
const src = ref('')
const alt = ref('')

export function useLightbox() {
  return {
    src,
    alt,
    open: (s: string, a = '') => { src.value = s; alt.value = a },
    close: () => { src.value = '' },
  }
}
