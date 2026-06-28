<script setup lang="ts">
// Ресайз-хендл МЕЖДУ панелями (отдельный flex-элемент, не внутри overflow-панели):
// вертикальная пилюля-островок, скрыта, проявляется на ховере.
defineEmits<{ (e: 'down', ev: PointerEvent): void }>()
</script>

<template>
  <div
    class="group/rsz relative w-2 shrink-0 cursor-col-resize self-stretch"
    title="Drag to resize"
    @pointerdown="$emit('down', $event)">
    <div class="absolute left-1/2 top-1/2 h-10 w-1 -translate-x-1/2 -translate-y-1/2 rounded-full bg-secondary/60 opacity-0 transition-opacity duration-150 group-hover/rsz:opacity-100" />
  </div>
</template>
