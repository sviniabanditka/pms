<script setup lang="ts">
// Моно-глиф статуса (терминальный стиль): ○◔◐●⊘ в цвете категории.
// Атом дизайн-системы — заменяет inline `<span :style>{{ statusGlyph }}</span>`.
const props = defineProps<{ category?: string }>()
const { statusColor, statusGlyph } = useColors()
</script>

<template>
  <span class="inline-block shrink-0 w-4 text-center leading-none" :style="{ color: statusColor(props.category || '') }">{{ statusGlyph(props.category || '') }}</span>
</template>
