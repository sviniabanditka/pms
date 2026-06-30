<script setup lang="ts">
// Терминальный чекбокс: [x] / [ ].
const props = defineProps<{ modelValue?: boolean; label?: string }>()
const emit = defineEmits<{ (e: 'update:modelValue', v: boolean): void }>()
</script>

<template>
  <button type="button" class="inline-flex select-none items-center gap-2 text-sm" @click.stop="emit('update:modelValue', !modelValue)">
    <span class="font-mono text-tertiary">[<span class="text-ok">{{ modelValue ? 'x' : ' ' }}</span>]</span>
    <span v-if="label" :class="modelValue ? 'text-secondary' : 'text-tertiary'">{{ label }}</span>
  </button>
</template>
