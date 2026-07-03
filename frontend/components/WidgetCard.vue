<script setup lang="ts">
import { useAppStore } from '~/stores/app'

// Карточка виджета дашборда (note/stat/view/chart). Молекула из DashboardSection:
// widget пропсом, данные (issueCount/viewIssues/issuesByCategory) из стора.
// drag-стейт координируется секцией: `dragged` проп + emit dragstart/dragend/drop.
const props = defineProps<{ widget: any; dashboardId: string; dragged?: boolean }>()
const emit = defineEmits<{ (e: 'dragstart'): void; (e: 'dragend'): void; (e: 'drop'): void }>()
const store = useAppStore()

function statValue(w: any) {
  const p = w.config?.priority
  const filter = p === '' || p == null ? null : { type: 'predicate' as const, field: 'priority', op: 'eq' as const, value: Number(p) }
  return store.issueCount(filter as any)
}
</script>

<template>
  <div class="group relative rounded-lg border border-subtle bg-elevated p-4 transition-shadow"
    :class="[(widget.w || 1) >= 2 ? 'sm:col-span-2' : '', dragged ? 'opacity-40' : '', 'cursor-grab']"
    draggable="true" @dragstart="emit('dragstart')" @dragend="emit('dragend')" @dragover.prevent @drop="emit('drop')">
    <div class="absolute right-2 top-2 flex gap-0.5 opacity-0 group-hover:opacity-100">
      <UButton :icon="(widget.w || 1) >= 2 ? 'i-heroicons-arrows-pointing-in' : 'i-heroicons-arrows-pointing-out'" color="gray" variant="ghost" size="2xs" square :title="(widget.w||1)>=2 ? 'Shrink' : 'Widen'" @click="store.resizeWidget(widget.id, (widget.w||1) >= 2 ? 1 : 2, widget.h || 1)" />
      <UButton icon="i-heroicons-x-mark" color="gray" variant="ghost" size="2xs" square @click="store.removeWidget(widget.id)" aria-label="Close" />
    </div>
    <div class="mb-2 text-xs font-medium uppercase tracking-wider text-tertiary"><span class="text-ok"># </span>{{ widget.title || widget.type }}</div>
    <div v-if="widget.type==='note'" class="whitespace-pre-wrap text-sm">{{ widget.config?.text }}</div>
    <div v-else-if="widget.type==='stat'" class="text-4xl font-semibold">{{ statValue(widget) }}</div>
    <div v-else-if="widget.type==='view'" class="space-y-1 text-sm">
      <div v-for="i in store.viewIssues(widget.config?.viewId)" :key="i.id" class="flex gap-2">
        <span class="text-accent2">{{ i.identifier }}</span><span class="truncate">{{ i.title }}</span>
      </div>
      <div v-if="!store.viewIssues(widget.config?.viewId).length" class="text-tertiary">// empty · pick a view</div>
    </div>
    <div v-else class="flex h-28 items-end gap-2 pt-2">
      <div v-for="c in store.issuesByCategory" :key="c.cat" class="flex flex-1 flex-col items-center gap-1">
        <div class="flex w-full flex-1 items-end">
          <div class="w-full rounded-t" :style="{ height: Math.max(4, (c.count / Math.max(1, Math.max(...store.issuesByCategory.map((x) => x.count)))) * 80) + 'px', background: c.color }" />
        </div>
        <span class="text-[10px] text-tertiary">{{ c.count }}</span>
      </div>
    </div>
  </div>
</template>
