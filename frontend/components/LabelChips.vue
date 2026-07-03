<script setup lang="ts">
import type { Label } from '~/types/entities'
// Чипы лейблов issue: #name в цвете лейбла. Props-pure — принимает готовый список.
defineProps<{ labels: Label[] }>()
</script>

<template>
  <span v-for="l in labels" :key="l.id" class="flex items-center gap-1 rounded border border-subtle px-1.5 py-0.5 text-xs text-tertiary">
    <span :style="{ color: l.colorToken }">#{{ l.name }}</span>
  </span>
</template>
