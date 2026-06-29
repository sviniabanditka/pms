<script setup lang="ts">
// Кастомный time-picker: инпут "3:00 am" + дропдаун с шагом 15 мин.
// v-model = 'HH:MM' (24ч). Опц. `from` ('HH:MM') → рядом показывает длительность (15m, 1h 00m…).
const props = withDefaults(defineProps<{ modelValue: string; from?: string; placeholder?: string; step?: number }>(), {
  modelValue: '', from: '', placeholder: '--:--', step: 15,
})
const emit = defineEmits<{ (e: 'update:modelValue', v: string): void }>()

const pad = (n: number) => String(n).padStart(2, '0')
const to12 = (hm: string) => {
  if (!hm) return ''
  const [h, m] = hm.split(':').map(Number)
  const ap = h < 12 ? 'am' : 'pm'; const h12 = h % 12 === 0 ? 12 : h % 12
  return `${h12}:${pad(m)} ${ap}`
}
const toMin = (hm: string) => { const [h, m] = hm.split(':').map(Number); return h * 60 + m }
const display = computed(() => to12(props.modelValue))

const opts = computed(() => {
  const fromMin = props.from ? toMin(props.from) : null
  const out = []
  for (let mins = 0; mins < 1440; mins += props.step) {
    const hm = `${pad(Math.floor(mins / 60))}:${pad(mins % 60)}`
    let dur = ''
    if (fromMin != null && mins > fromMin) {
      const d = mins - fromMin
      dur = d < 60 ? `${d}m` : `${Math.floor(d / 60)}h ${pad(d % 60)}m`
    }
    out.push({ hm, label: to12(hm), dur, sel: hm === props.modelValue })
  }
  return out
})
function pick(hm: string, close: () => void) { emit('update:modelValue', hm); close() }
// проскроллить к выбранному при открытии
function scrollToSel(el: any) {
  if (!el) return
  nextTick(() => { const t = el.querySelector('[data-sel="1"]'); if (t) t.scrollIntoView({ block: 'center' }) })
}
</script>

<template>
  <UPopover :popper="{ placement: 'bottom-start' }" :ui="{ width: '' }">
    <button type="button"
      class="flex h-8 w-full items-center gap-1.5 rounded-md border border-subtle bg-elevated px-2.5 text-sm transition-colors hover:border-secondary">
      <span :class="display ? 'text-white' : 'text-tertiary'">{{ display || placeholder }}</span>
    </button>
    <template #panel="{ close }">
      <div :ref="scrollToSel" class="max-h-72 w-44 overflow-y-auto rounded-lg border border-subtle bg-surface py-1 shadow-xl">
        <button v-for="o in opts" :key="o.hm" type="button" :data-sel="o.sel ? 1 : 0"
          class="flex w-full items-center gap-2 px-3 py-1.5 text-left text-sm transition-colors"
          :class="o.sel ? 'bg-active text-white' : 'text-secondary hover:bg-hover'"
          @click="pick(o.hm, close)">
          <span class="flex-1">{{ o.label }}</span>
          <span v-if="o.dur" class="text-xs text-tertiary">{{ o.dur }}</span>
        </button>
      </div>
    </template>
  </UPopover>
</template>
