<script setup lang="ts">
import { useAppStore } from '~/stores/app'

// Kanban-борд (layout=board): колонки по статусам, DnD карточек (reorderIssue),
// клавиатурный реордер (a11y). Инлайн-«+» в колонке открывает общую Create-модалку.
const props = defineProps<{ teamId: string; selectedIssueId: string }>()
const emit = defineEmits<{
  (e: 'open-issue', id: string): void
  (e: 'create-issue'): void
}>()

const store = useAppStore()
const { wrap } = useActionWrap()
const draggedId = ref('')

// DnD: вставить перетащенную карточку перед targetId в колонке stateId.
function onDropOnCard(stateId: string, targetId: string) {
  if (!draggedId.value || draggedId.value === targetId) { draggedId.value = ''; return }
  const list = store.issuesFor(props.teamId, stateId).filter((x) => x.id !== draggedId.value)
  const idx = list.findIndex((x) => x.id === targetId)
  const before = idx > 0 ? list[idx - 1].id : ''
  wrap(() => store.reorderIssue(draggedId.value, stateId, before, targetId))
  draggedId.value = ''
}
// DnD: бросок в пустую часть колонки → в конец.
// Клавиатурный реордер борда (a11y, D19): Alt+↑/↓ — внутри колонки, Alt+←/→ — между колонками.
function boardKey(e: KeyboardEvent, stateId: string, id: string) {
  if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); emit('open-issue', id); return }
  if (!e.altKey) return
  const states = store.teamStates(props.teamId)
  const colIdx = states.findIndex((s: any) => s.id === stateId)
  if (e.key === 'ArrowUp' || e.key === 'ArrowDown') {
    e.preventDefault()
    const list = store.issuesFor(props.teamId, stateId)
    const idx = list.findIndex((x: any) => x.id === id)
    if (e.key === 'ArrowUp' && idx > 0) {
      const before = idx >= 2 ? list[idx - 2].id : ''
      wrap(() => store.reorderIssue(id, stateId, before, list[idx - 1].id))
    } else if (e.key === 'ArrowDown' && idx >= 0 && idx < list.length - 1) {
      const after = idx + 2 < list.length ? list[idx + 2].id : ''
      wrap(() => store.reorderIssue(id, stateId, list[idx + 1].id, after))
    }
  } else if (e.key === 'ArrowLeft' && colIdx > 0) {
    e.preventDefault()
    wrap(() => store.reorderIssue(id, states[colIdx - 1].id, '', ''))
  } else if (e.key === 'ArrowRight' && colIdx >= 0 && colIdx < states.length - 1) {
    e.preventDefault()
    wrap(() => store.reorderIssue(id, states[colIdx + 1].id, '', ''))
  }
}
function onDropOnColumn(stateId: string) {
  if (!draggedId.value) return
  const list = store.issuesFor(props.teamId, stateId).filter((x) => x.id !== draggedId.value)
  const before = list.length ? list[list.length - 1].id : ''
  wrap(() => store.reorderIssue(draggedId.value, stateId, before, ''))
  draggedId.value = ''
}
</script>

<template>
  <div class="flex h-[calc(100%-56px)] gap-4 overflow-x-auto px-4 py-3">
    <div v-for="st in store.teamStates(teamId)" :key="st.id"
      class="group/col flex w-72 shrink-0 flex-col rounded-xl border border-subtle bg-base p-2"
      @dragover.prevent @drop="onDropOnColumn(st.id)">
      <!-- column header -->
      <div class="mb-2 flex items-center gap-2 px-1.5 py-1 text-sm">
        <StatusGlyph :category="st.category" />
        <span class="font-medium">{{ st.name }}</span>
        <span class="text-tertiary">{{ store.issuesFor(teamId, st.id).length }}</span>
        <UButton v-if="store.canWrite" class="ml-auto opacity-0 group-hover/col:opacity-100" icon="i-heroicons-plus" color="gray" variant="ghost" size="2xs" square @click="emit('create-issue')" aria-label="Add" />
      </div>
      <div class="flex-1 space-y-2 overflow-y-auto">
        <div v-for="i in store.issuesFor(teamId, st.id)" :key="i.id"
          draggable="true" @dragstart="draggedId = i.id"
          @dragover.prevent @drop.stop="onDropOnCard(st.id, i.id)"
          @click="emit('open-issue', i.id)"
          tabindex="0" role="button"
          :aria-label="`${i.identifier} ${i.title}. Alt+arrows to move`"
          @keydown="boardKey($event, st.id, i.id)"
          class="cursor-pointer rounded-lg border border-subtle bg-elevated p-3 hover:bg-hover focus:outline-none focus:ring-2 focus:ring-accent"
          :class="{ '!border-accent': selectedIssueId === i.id }">
          <div class="mb-1.5 text-xs text-accent2">{{ i.identifier }}</div>
          <div class="line-clamp-3 text-sm font-medium leading-snug">{{ i.title }}</div>
          <div v-if="store.labelsForIssue(i.id).length" class="mt-1.5 flex flex-wrap gap-1">
            <LabelChips :labels="store.labelsForIssue(i.id)" />
          </div>
          <div class="mt-2.5 flex items-center gap-2 text-xs text-tertiary">
            <PriorityGlyph v-if="i.priority" :priority="i.priority" />
            <span v-if="store.childProgress(i.id).total" class="flex items-center gap-1"><UIcon name="i-heroicons-queue-list" /> {{ store.childProgress(i.id).done }}/{{ store.childProgress(i.id).total }}</span>
            <span v-if="i.assigneeId" class="ml-auto flex h-5 w-5 items-center justify-center rounded-full bg-surface text-[10px]" :title="store.userName(i.assigneeId)">{{ (store.userName(i.assigneeId) || '?')[0].toUpperCase() }}</span>
            <span v-if="i.createdAt" :class="i.assigneeId ? '' : 'ml-auto'">{{ new Date(i.createdAt).toLocaleDateString('en', { month: 'short', day: 'numeric' }) }}</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
