<script setup lang="ts">
import { ref, watch, onMounted } from 'vue'

const props = defineProps<{ stream: MediaStream | null; muted?: boolean; label?: string; mutedIcon?: boolean; camOff?: boolean }>()
const el = ref<HTMLVideoElement | null>(null)

function attach() { if (el.value && props.stream && el.value.srcObject !== props.stream) el.value.srcObject = props.stream }
watch(() => props.stream, attach)
onMounted(attach)
</script>

<template>
  <div class="relative aspect-video overflow-hidden rounded-lg bg-black">
    <video ref="el" autoplay playsinline :muted="muted" class="h-full w-full object-cover" :class="{ 'opacity-0': camOff }" />
    <div v-if="camOff" class="absolute inset-0 flex items-center justify-center bg-elevated text-2xl font-semibold text-tertiary">
      {{ (label || '?').charAt(0).toUpperCase() }}
    </div>
    <div class="absolute bottom-1.5 left-1.5 flex items-center gap-1 rounded bg-black/60 px-1.5 py-0.5 text-xs text-white">
      <UIcon v-if="mutedIcon" name="i-heroicons-microphone-slash" class="text-red-400" />
      {{ label }}
    </div>
  </div>
</template>
