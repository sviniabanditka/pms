<script setup lang="ts">
// Моно-глиф приоритета: ·▂▄▆█ (none→urgent) в цвете приоритета.
const props = defineProps<{ priority?: number }>()
const { prioColor, prioGlyph } = useColors()
</script>

<template>
  <span class="inline-block shrink-0 w-4 text-center leading-none" :style="{ color: prioColor(props.priority ?? 0) }">{{ prioGlyph(props.priority ?? 0) }}</span>
</template>
