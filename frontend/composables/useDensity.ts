import { ref } from 'vue'

// Плотность интерфейса (клиентское, localStorage) — модульный синглтон:
// страница применяет сохранённое значение в onMounted, SettingsModal переключает.
const density = ref('default')
const DENSITY_PX: Record<string, string> = { compact: '15px', default: '16px', comfortable: '18px' }

export function useDensity() {
  function applyDensity(d: string) {
    density.value = d
    if (import.meta.client) { document.documentElement.style.fontSize = DENSITY_PX[d] || '16px'; localStorage.setItem('pms-density', d) }
  }
  return { density, applyDensity }
}
