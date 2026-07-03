<script setup lang="ts">
import type { ComponentPublicInstance } from 'vue'
import { useAppStore } from '~/stores/app'
import { useVirtualizer } from '@tanstack/vue-virtual'

// Список задач (layout=list, Linear-стиль): виртуализованный плоский поток
// header-строк групп + issue-строк. Скроллится <main> страницы → scroll-el проп.
// Мультивыбор (selected-ids) и bulk-бар живут на странице: выбор переживает смену layout.
const props = defineProps<{
  teamId: string
  selectedIssueId: string
  selectedIds: Set<string>
  filtersShown: boolean
  scrollEl: HTMLElement | null
}>()
const emit = defineEmits<{
  (e: 'open-issue', id: string): void
  (e: 'toggle-sel', id: string): void
}>()

const store = useAppStore()
const { statusColor, statusGlyph, prioColor, prioGlyph } = useColors()
const selCount = computed(() => props.selectedIds.size)
const collapsedStatus = reactive<Record<string, boolean>>({}) // свёрнутые группы статусов в списке

// --- List virtualization (плоский поток: header-строки + issue-строки) ---
// header|issue в одном плоском потоке; g/i — any, поэтому union без дискриминации
// (индексный доступ listRows[idx] в шаблоне не нарезался бы по type).
type ListRow = { type: 'header' | 'issue'; key: string; g: any; i?: any }
const listRows = computed<ListRow[]>(() => {
  const rows: ListRow[] = []
  if (!props.teamId || store.layout !== 'list') return rows
  for (const g of store.issueGroups(props.teamId)) {
    rows.push({ type: 'header', key: 'h:' + g.key, g })
    if (!collapsedStatus[g.key]) for (const i of g.items) rows.push({ type: 'issue', key: i.id, i, g })
  }
  return rows
})
const listEl = ref<HTMLElement | null>(null)
const listScrollMargin = ref(0)
const measureScrollMargin = () => {
  if (!props.scrollEl || !listEl.value) return
  listScrollMargin.value = listEl.value.getBoundingClientRect().top - props.scrollEl.getBoundingClientRect().top + props.scrollEl.scrollTop
}
const rowVirtualizer = useVirtualizer(computed(() => {
  const scrollEl = props.scrollEl // явная зависимость: виртуализатор подхватит поздний mount <main>
  return {
    count: listRows.value.length,
    getScrollElement: () => scrollEl,
    estimateSize: () => 42,
    overscan: 10,
    scrollMargin: listScrollMargin.value,
    getItemKey: (idx: number) => listRows.value[idx]?.key ?? idx,
  }
}))
const measureRow = (el: Element | ComponentPublicInstance | null) => { if (el instanceof Element) rowVirtualizer.value.measureElement(el) }
// пересчёт оффсета списка при изменениях, влияющих на высоту sticky-хедера/раскладку
watch([() => props.filtersShown, () => store.layout, () => props.teamId, () => props.scrollEl], () => nextTick(measureScrollMargin))
onMounted(() => nextTick(measureScrollMargin))
</script>

<template>
  <div ref="listEl" :style="{ height: rowVirtualizer.getTotalSize() + 'px', position: 'relative', width: '100%' }">
    <div v-for="vRow in rowVirtualizer.getVirtualItems()" :key="String(vRow.key)"
      :ref="measureRow" :data-index="vRow.index"
      :style="{ position: 'absolute', top: 0, left: 0, width: '100%', transform: `translateY(${vRow.start - listScrollMargin}px)` }">
      <!-- group header bar -->
      <div v-if="listRows[vRow.index]?.type === 'header'"
        class="flex cursor-pointer items-center gap-2 bg-elevated/50 px-4 py-2.5 text-sm font-semibold hover:bg-elevated"
        @click="collapsedStatus[listRows[vRow.index].g.key] = !collapsedStatus[listRows[vRow.index].g.key]">
        <template v-if="listRows[vRow.index]" >
          <UIcon :name="collapsedStatus[listRows[vRow.index].g.key] ? 'i-heroicons-chevron-right' : 'i-heroicons-chevron-down'" class="text-tertiary" />
          <span v-if="listRows[vRow.index].g.kind === 'status'" class="inline-block shrink-0 w-4 text-center leading-none" :style="{ color: statusColor(listRows[vRow.index].g.category) }">{{ statusGlyph(listRows[vRow.index].g.category) }}</span>
          <span v-else-if="listRows[vRow.index].g.kind === 'priority'" class="inline-block shrink-0 w-4 text-center leading-none" :style="{ color: prioColor(listRows[vRow.index].g.prio) }">{{ prioGlyph(listRows[vRow.index].g.prio) }}</span>
          <UIcon v-else-if="listRows[vRow.index].g.kind === 'milestone'" name="i-heroicons-flag" class="text-tertiary" />
          <span v-else class="flex h-4 w-4 items-center justify-center rounded-full bg-elevated text-[9px]">{{ (listRows[vRow.index].g.label || '?')[0].toUpperCase() }}</span>
          <span>{{ listRows[vRow.index].g.label }}</span>
          <span class="font-normal text-tertiary">{{ listRows[vRow.index].g.items.length }}</span>
        </template>
      </div>
      <!-- issue row -->
      <div v-else-if="listRows[vRow.index]"
        class="group flex cursor-pointer items-center gap-3 border-b border-subtle/40 px-4 py-2.5 hover:bg-hover"
        :class="{ 'bg-active': selectedIssueId === listRows[vRow.index].i.id || selectedIds.has(listRows[vRow.index].i.id) }"
        @click="emit('open-issue', listRows[vRow.index].i.id)">
        <TermCheck v-if="store.canWrite" :model-value="selectedIds.has(listRows[vRow.index].i.id)" class="shrink-0 transition-opacity" :class="selCount || selectedIds.has(listRows[vRow.index].i.id) ? 'opacity-100' : 'opacity-0 group-hover:opacity-100'" @click.stop @update:model-value="emit('toggle-sel', listRows[vRow.index].i.id)" />
        <span v-if="listRows[vRow.index].i.priority" class="inline-block shrink-0 w-4 text-center leading-none" :style="{ color: prioColor(listRows[vRow.index].i.priority) }">{{ prioGlyph(listRows[vRow.index].i.priority) }}</span>
        <UIcon v-else name="i-heroicons-minus" class="shrink-0 text-tertiary/50" />
        <span class="min-w-14 shrink-0 whitespace-nowrap text-sm text-accent2">{{ listRows[vRow.index].i.identifier }}</span>
        <span class="inline-block shrink-0 w-4 text-center leading-none" :style="{ color: statusColor(store.states[listRows[vRow.index].i.stateId]?.category) }">{{ statusGlyph(store.states[listRows[vRow.index].i.stateId]?.category) }}</span>
        <span class="truncate">{{ listRows[vRow.index].i.title }}</span>
        <span class="flex flex-1 flex-wrap items-center gap-1">
          <span v-for="l in store.labelsForIssue(listRows[vRow.index].i.id)" :key="l.id" class="flex items-center gap-1 rounded border border-subtle px-1.5 py-0.5 text-xs text-tertiary">
            <span :style="{ color: l.colorToken }">#{{ l.name }}</span>
          </span>
        </span>
        <span v-if="listRows[vRow.index].i.dueDate || listRows[vRow.index].i.scheduledDate" class="shrink-0 text-sm tabular-nums text-tertiary">{{ String(listRows[vRow.index].i.scheduledDate || listRows[vRow.index].i.dueDate).slice(5, 10) }}</span>
        <span v-if="listRows[vRow.index].i.assigneeId" class="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-elevated text-[10px]" :title="store.userName(listRows[vRow.index].i.assigneeId)">{{ (store.userName(listRows[vRow.index].i.assigneeId) || '?')[0].toUpperCase() }}</span>
        <UButton v-if="store.canWrite" icon="i-heroicons-trash" color="gray" variant="ghost" size="xs" square
          class="shrink-0 opacity-0 hover:opacity-100" title="Archive" @click.stop="store.archiveIssue(listRows[vRow.index].i.id)" />
      </div>
    </div>
  </div>
</template>
