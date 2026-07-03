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
          <StatusGlyph v-if="listRows[vRow.index].g.kind === 'status'" :category="listRows[vRow.index].g.category" />
          <PriorityGlyph v-else-if="listRows[vRow.index].g.kind === 'priority'" :priority="listRows[vRow.index].g.prio" />
          <UIcon v-else-if="listRows[vRow.index].g.kind === 'milestone'" name="i-heroicons-flag" class="text-tertiary" />
          <span v-else class="flex h-4 w-4 items-center justify-center rounded-full bg-elevated text-[9px]">{{ (listRows[vRow.index].g.label || '?')[0].toUpperCase() }}</span>
          <span>{{ listRows[vRow.index].g.label }}</span>
          <span class="font-normal text-tertiary">{{ listRows[vRow.index].g.items.length }}</span>
        </template>
      </div>
      <!-- issue row -->
      <IssueRow v-else-if="listRows[vRow.index]" :issue="listRows[vRow.index].i"
        :selected="selectedIds.has(listRows[vRow.index].i.id)"
        :selection-active="!!selCount"
        :active="selectedIssueId === listRows[vRow.index].i.id"
        @open="emit('open-issue', $event)" @toggle="emit('toggle-sel', $event)" />
    </div>
  </div>
</template>
