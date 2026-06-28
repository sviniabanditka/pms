import { ref } from 'vue'

// Горизонтальный ресайз правой панели (тащим левый край). Ширина в px, сохраняется в localStorage.
export function useResizable(key: string, def: number, min = 300, max = 820) {
  const w = ref(def)
  if (import.meta.client) {
    const s = Number(localStorage.getItem('panelw:' + key))
    if (s) w.value = Math.min(max, Math.max(min, s))
  }
  function onResizeStart(e: PointerEvent) {
    e.preventDefault()
    const startX = e.clientX
    const startW = w.value
    const move = (ev: PointerEvent) => {
      // панель справа → тянем левый край влево = шире
      w.value = Math.min(max, Math.max(min, startW + (startX - ev.clientX)))
    }
    const up = () => {
      window.removeEventListener('pointermove', move)
      window.removeEventListener('pointerup', up)
      document.body.style.userSelect = ''
      if (import.meta.client) localStorage.setItem('panelw:' + key, String(Math.round(w.value)))
    }
    document.body.style.userSelect = 'none'
    window.addEventListener('pointermove', move)
    window.addEventListener('pointerup', up)
  }
  return { w, onResizeStart }
}
