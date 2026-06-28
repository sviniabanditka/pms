<script setup lang="ts">
import { computed } from 'vue'
import { useAppStore } from '~/stores/app'

const props = defineProps<{ cycleId: string }>()
const store = useAppStore()

const W = 280, H = 110, PAD = 4
const data = computed(() => store.cycleBurndown(props.cycleId))
const max = computed(() => Math.max(1, ...data.value.map((d) => d.remaining)))
const n = computed(() => data.value.length)
function x(i: number) { return PAD + (n.value <= 1 ? 0 : (i / (n.value - 1)) * (W - 2 * PAD)) }
function y(v: number) { return PAD + (1 - v / max.value) * (H - 2 * PAD) }
const actualPts = computed(() => data.value.map((d, i) => `${x(i)},${y(d.remaining)}`).join(' '))
// идеальная линия: от первого остатка до 0
const idealPts = computed(() => {
  if (n.value < 2) return ''
  return `${x(0)},${y(data.value[0].remaining)} ${x(n.value - 1)},${y(0)}`
})
</script>

<template>
  <div>
    <div v-if="n < 2" class="py-3 text-xs text-tertiary">Burndown is accumulating — one snapshot per day (needs ≥2 days).</div>
    <svg v-else :viewBox="`0 0 ${W} ${H}`" class="w-full">
      <line :x1="PAD" :y1="H - PAD" :x2="W - PAD" :y2="H - PAD" stroke="currentColor" class="text-subtle" stroke-width="0.5" />
      <polyline :points="idealPts" fill="none" stroke="currentColor" class="text-tertiary" stroke-width="1" stroke-dasharray="3 3" />
      <polyline :points="actualPts" fill="none" stroke="#5e6ad2" stroke-width="1.5" />
      <circle v-for="(d, i) in data" :key="i" :cx="x(i)" :cy="y(d.remaining)" r="2" fill="#5e6ad2" />
    </svg>
    <div class="mt-1 flex items-center justify-between text-[11px] text-tertiary">
      <span>{{ data[0]?.day?.slice(5) }}</span>
      <span class="flex gap-3"><span class="text-secondary">— remaining</span><span>┄ ideal</span></span>
      <span>{{ data[n - 1]?.day?.slice(5) }}</span>
    </div>
  </div>
</template>
