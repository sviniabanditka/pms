<script setup lang="ts">
import type { Issue } from '~/types/entities'
import { useAppStore } from '~/stores/app'

// Строка сабтаска в issue-панели. Молекула: issue пропсом, статус-глиф из стора.
defineProps<{ issue: Issue }>()
const emit = defineEmits<{ (e: 'open', id: string): void }>()
const store = useAppStore()
</script>

<template>
  <div class="flex cursor-pointer items-center gap-2 rounded-md px-2 py-1.5 hover:bg-hover" @click="emit('open', issue.id)">
    <StatusGlyph :category="store.states[issue.stateId]?.category" />
    <span class="text-xs text-accent2">{{ issue.identifier }}</span>
    <span class="truncate" :class="{ 'text-tertiary line-through': issue.completedAt }">{{ issue.title }}</span>
  </div>
</template>
