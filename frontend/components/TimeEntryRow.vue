<script setup lang="ts">
import { fmtMin } from '~/lib/ui'

// Запись учёта времени под свойством Time. Молекула из IssueDetailPanel:
// entry пропсом, удаление — эмитом (панель зовёт time.deleteTimeEntry).
defineProps<{ entry: any }>()
const emit = defineEmits<{ (e: 'delete', id: string): void }>()

function dur(te: any) {
  return fmtMin(Math.max(0, Math.round((new Date(te.endedAt).getTime() - new Date(te.startedAt).getTime()) / 60000)))
}
</script>

<template>
  <div class="group ml-24 flex items-center gap-2 pl-3 text-xs text-tertiary">
    <UIcon :name="entry.endedAt ? 'i-heroicons-check-circle' : 'i-heroicons-play-circle'" :class="entry.endedAt ? '' : 'text-ok'" class="shrink-0" />
    <span class="tabular-nums">{{ new Date(entry.startedAt).toLocaleDateString([], { month: 'short', day: 'numeric' }) }} {{ new Date(entry.startedAt).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }) }}</span>
    <span class="flex-1 truncate">{{ entry.description }}</span>
    <span v-if="entry.endedAt" class="tabular-nums">{{ dur(entry) }}</span>
    <span v-else class="text-ok">running</span>
    <UButton class="opacity-0 group-hover:opacity-100" icon="i-heroicons-x-mark" color="gray" variant="ghost" size="2xs" square @click="emit('delete', entry.id)" />
  </div>
</template>
