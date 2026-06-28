<script setup lang="ts">
// Ресайз-хендл между панелями: вертикальная пилюля, скрыта, проявляется на ховере.
defineEmits<{ (e: 'down', ev: PointerEvent): void }>()
</script>

<template>
  <div
    class="group/rsz absolute inset-y-0 left-0 z-20 flex w-3 cursor-col-resize items-center justify-center"
    title="Drag to resize"
    @pointerdown="$emit('down', $event)">
    <div class="h-10 w-1 rounded-full bg-secondary/50 opacity-0 transition-opacity duration-150 group-hover/rsz:opacity-100" />
  </div>
</template>
