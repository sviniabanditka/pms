<script setup lang="ts">
import { computed } from 'vue'
// ASCII прогресс-бар: [████░░░░] 60%
const props = withDefaults(defineProps<{ done?: number; total?: number; pct?: number; width?: number; hideLabel?: boolean }>(), { width: 12 })
const ratio = computed(() => props.pct != null ? props.pct / 100 : (props.total ? (props.done || 0) / props.total : 0))
const filled = computed(() => Math.round(Math.max(0, Math.min(1, ratio.value)) * props.width))
const label = computed(() => Math.round(Math.max(0, Math.min(1, ratio.value)) * 100) + '%')
</script>

<template>
  <span class="font-mono text-xs whitespace-nowrap">
    <span class="text-tertiary">[</span><span class="text-ok">{{ '█'.repeat(filled) }}</span><span class="text-tertiary/40">{{ '░'.repeat(width - filled) }}</span><span class="text-tertiary">]</span>
    <span v-if="!hideLabel" class="ml-1 text-tertiary">{{ label }}</span>
  </span>
</template>
