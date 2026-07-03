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
</script>

<template>
  <template v-if="selectedDashboard">
    <div class="sticky top-0 z-10 flex items-center gap-2 border-b border-subtle bg-surface px-4 py-3 text-sm">
      <UIcon name="i-heroicons-squares-2x2" class="text-tertiary" />
      <span class="font-medium">{{ selectedDashboard.name }}</span>
      <UButton class="ml-auto" size="xs" color="gray" variant="ghost" @click="$emit('add-widget')">[ add widget ]</UButton>
    </div>
    <div class="grid grid-cols-1 gap-3 p-4 sm:grid-cols-2 xl:grid-cols-3">
      <WidgetCard v-for="w in store.widgetsOf(dashboardId)" :key="w.id" :widget="w" :dashboard-id="dashboardId"
        :dragged="draggedWidget === w.id"
        @dragstart="draggedWidget = w.id" @dragend="draggedWidget = ''"
        @drop="draggedWidget && draggedWidget !== w.id && store.moveWidget(dashboardId, draggedWidget, w.id)" />
      <div v-if="!store.widgetsOf(dashboardId).length"
        class="flex h-32 cursor-pointer items-center justify-center rounded-lg border border-dashed border-subtle text-sm text-tertiary hover:bg-hover"
        @click="$emit('add-widget')"><span class="text-ok">$</span>&nbsp;add your first widget</div>
    </div>
  </template>
  <div v-else class="p-8 text-sm text-tertiary">Pick or create a dashboard →</div>
</template>
