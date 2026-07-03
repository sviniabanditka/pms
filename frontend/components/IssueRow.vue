<script setup lang="ts">
import type { Issue } from '~/types/entities'
import { useAppStore } from '~/stores/app'

// Строка issue в списке (Linear-стиль). Молекула: issue + флаги выбора приходят
// пропсами, derived-данные (state/labels/assignee) читаются из стора.
const props = defineProps<{
  issue: Issue
  selected?: boolean       // отмечена в мультивыборе
  selectionActive?: boolean // есть активный мультивыбор (показывать чекбоксы)
  active?: boolean          // открыта в detail-панели
}>()
const emit = defineEmits<{ (e: 'open', id: string): void; (e: 'toggle', id: string): void }>()
const store = useAppStore()
</script>

<template>
  <div class="group flex cursor-pointer items-center gap-3 border-b border-subtle/40 px-4 py-2.5 hover:bg-hover"
    :class="{ 'bg-active': active || selected }"
    @click="emit('open', issue.id)">
    <TermCheck v-if="store.canWrite" :model-value="!!selected" class="shrink-0 transition-opacity"
      :class="selectionActive || selected ? 'opacity-100' : 'opacity-0 group-hover:opacity-100'"
      @click.stop @update:model-value="emit('toggle', issue.id)" />
    <PriorityGlyph v-if="issue.priority" :priority="issue.priority" />
    <UIcon v-else name="i-heroicons-minus" class="shrink-0 text-tertiary/50" />
    <span class="min-w-14 shrink-0 whitespace-nowrap text-sm text-accent2">{{ issue.identifier }}</span>
    <StatusGlyph :category="store.states[issue.stateId]?.category" />
    <span class="truncate">{{ issue.title }}</span>
    <span class="flex flex-1 flex-wrap items-center gap-1">
      <LabelChips :labels="store.labelsForIssue(issue.id)" />
    </span>
    <span v-if="issue.dueDate || issue.scheduledDate" class="shrink-0 text-sm tabular-nums text-tertiary">{{ String(issue.scheduledDate || issue.dueDate).slice(5, 10) }}</span>
    <span v-if="issue.assigneeId" class="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-elevated text-[10px]" :title="store.userName(issue.assigneeId)">{{ (store.userName(issue.assigneeId) || '?')[0].toUpperCase() }}</span>
    <UButton v-if="store.canWrite" icon="i-heroicons-trash" color="gray" variant="ghost" size="xs" square
      class="shrink-0 opacity-0 hover:opacity-100" title="Archive" @click.stop="store.archiveIssue(issue.id)" aria-label="Archive" />
  </div>
</template>
