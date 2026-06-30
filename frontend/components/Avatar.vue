<script setup lang="ts">
import { computed } from 'vue'
import { fileViewUrl } from '~/lib/upload'

const props = withDefaults(defineProps<{ name?: string; avatarKey?: string | null; size?: number }>(), { size: 24 })
const src = computed(() => fileViewUrl(props.avatarKey))
const letter = computed(() => (props.name || '?').charAt(0).toUpperCase())
</script>

<template>
  <span class="inline-flex shrink-0 items-center justify-center overflow-hidden rounded-md bg-elevated font-medium text-tertiary"
    :style="{ width: size + 'px', height: size + 'px', fontSize: Math.round(size * 0.42) + 'px', borderRadius: Math.max(4, Math.round(size * 0.28)) + 'px' }">
    <img v-if="src" :src="src" class="h-full w-full object-cover" alt="" />
    <template v-else>{{ letter }}</template>
  </span>
</template>
