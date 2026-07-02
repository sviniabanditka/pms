<script setup lang="ts">
import { useAppStore } from '~/stores/app'

// Дашборд: грид виджетов (note/stat/view/chart) с drag-reorder и resize.
// selectedDashboardId — роутовый стейт страницы → приходит пропом;
// модалка add-widget живёт на странице → эмит.
const props = defineProps<{ dashboardId: string }>()
defineEmits<{ (e: 'add-widget'): void }>()

const store = useAppStore()
const selectedDashboard = computed(() => store.dashboards[props.dashboardId])
const draggedWidget = ref('')
function statValue(w: any) {
  const p = w.config?.priority
  const filter = p === '' || p == null ? null : { type: 'predicate' as const, field: 'priority', op: 'eq' as const, value: Number(p) }
  return store.issueCount(filter as any)
}
</script>

<template>
  <template v-if="selectedDashboard">
    <div class="sticky top-0 z-10 flex items-center gap-2 border-b border-subtle bg-surface px-4 py-3 text-sm">
      <UIcon name="i-heroicons-squares-2x2" class="text-tertiary" />
      <span class="font-medium">{{ selectedDashboard.name }}</span>
      <UButton class="ml-auto" size="xs" color="gray" variant="ghost" @click="$emit('add-widget')">[ add widget ]</UButton>
    </div>
    <div class="grid grid-cols-1 gap-3 p-4 sm:grid-cols-2 xl:grid-cols-3">
      <div v-for="w in store.widgetsOf(dashboardId)" :key="w.id"
        class="group relative rounded-lg border border-subtle bg-elevated p-4 transition-shadow"
        :class="[(w.w || 1) >= 2 ? 'sm:col-span-2' : '', draggedWidget === w.id ? 'opacity-40' : '', 'cursor-grab']"
        draggable="true" @dragstart="draggedWidget = w.id" @dragend="draggedWidget = ''"
        @dragover.prevent @drop="draggedWidget && draggedWidget !== w.id && store.moveWidget(dashboardId, draggedWidget, w.id)">
        <div class="absolute right-2 top-2 flex gap-0.5 opacity-0 group-hover:opacity-100">
          <UButton :icon="(w.w || 1) >= 2 ? 'i-heroicons-arrows-pointing-in' : 'i-heroicons-arrows-pointing-out'" color="gray" variant="ghost" size="2xs" square :title="(w.w||1)>=2 ? 'Shrink' : 'Widen'" @click="store.resizeWidget(w.id, (w.w||1) >= 2 ? 1 : 2, w.h || 1)" />
          <UButton icon="i-heroicons-x-mark" color="gray" variant="ghost" size="2xs" square @click="store.removeWidget(w.id)" />
        </div>
        <div class="mb-2 text-xs font-medium uppercase tracking-wider text-tertiary"><span class="text-ok"># </span>{{ w.title || w.type }}</div>
        <div v-if="w.type==='note'" class="whitespace-pre-wrap text-sm">{{ w.config?.text }}</div>
        <div v-else-if="w.type==='stat'" class="text-4xl font-semibold">{{ statValue(w) }}</div>
        <div v-else-if="w.type==='view'" class="space-y-1 text-sm">
          <div v-for="i in store.viewIssues(w.config?.viewId)" :key="i.id" class="flex gap-2">
            <span class="text-accent2">{{ i.identifier }}</span><span class="truncate">{{ i.title }}</span>
          </div>
          <div v-if="!store.viewIssues(w.config?.viewId).length" class="text-tertiary">// empty · pick a view</div>
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
      <div v-if="!store.widgetsOf(dashboardId).length"
        class="flex h-32 cursor-pointer items-center justify-center rounded-lg border border-dashed border-subtle text-sm text-tertiary hover:bg-hover"
        @click="$emit('add-widget')"><span class="text-ok">$</span>&nbsp;add your first widget</div>
    </div>
  </template>
  <div v-else class="p-8 text-sm text-tertiary">Pick or create a dashboard →</div>
</template>
