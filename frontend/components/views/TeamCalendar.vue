<script setup lang="ts">
import { useAppStore } from '~/stores/app'

// Командный календарь (layout=calendar в Tasks): месяц/неделя-сетка по due/scheduled,
// DnD задач на день → setIssueSchedule. Личный календарь — SectionsCalendarSection.
const props = defineProps<{ teamId: string }>()
const emit = defineEmits<{ (e: 'open-issue', id: string): void }>()

const store = useAppStore()
const { wrap } = useActionWrap()
const draggedId = ref('')

const WEEKDAYS = ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun']
const monthOffset = ref(0)
const calMonth = computed(() => { const n = new Date(); return new Date(n.getFullYear(), n.getMonth() + monthOffset.value, 1) })
const calLabel = computed(() => calMonth.value.toLocaleDateString('en', { month: 'long', year: 'numeric' }))
function fmtDate(d: Date) { return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}` }
const todayStr = fmtDate(new Date())
const calendarCells = computed(() => {
  const first = calMonth.value
  const month = first.getMonth()
  const start = new Date(first)
  start.setDate(1 - ((first.getDay() + 6) % 7)) // понедельник недели с 1-м числом
  const cells: { date: string; inMonth: boolean; day: number }[] = []
  for (let i = 0; i < 42; i++) {
    const d = new Date(start); d.setDate(start.getDate() + i)
    cells.push({ date: fmtDate(d), inMonth: d.getMonth() === month, day: d.getDate() })
  }
  return cells
})
function dropOnDay(ds: string) {
  if (draggedId.value) { wrap(() => store.setIssueSchedule(draggedId.value, ds)); draggedId.value = '' }
}
// week-view
const calMode = ref<'month' | 'week'>('month')
const weekCells = computed(() => {
  const n = new Date(); const base = new Date(n.getFullYear(), n.getMonth(), n.getDate() + monthOffset.value * 7)
  base.setDate(base.getDate() - ((base.getDay() + 6) % 7)) // понедельник
  const cells: { date: string; inMonth: boolean; day: number }[] = []
  for (let i = 0; i < 7; i++) { const d = new Date(base); d.setDate(base.getDate() + i); cells.push({ date: fmtDate(d), inMonth: true, day: d.getDate() }) }
  return cells
})
const calCells = computed(() => (calMode.value === 'week' ? weekCells.value : calendarCells.value))
const calRangeLabel = computed(() => {
  if (calMode.value === 'month') return calLabel.value
  const w = weekCells.value
  return `${w[0].date.slice(5)} – ${w[6].date.slice(5)}`
})
</script>

<template>
  <div class="p-4">
    <div class="mb-2 flex items-center gap-2 text-sm">
      <UButton icon="i-heroicons-chevron-left" color="gray" variant="ghost" size="xs" square @click="monthOffset--" aria-label="Previous" />
      <span class="font-medium">{{ calRangeLabel }}</span>
      <UButton color="gray" variant="ghost" size="xs" square @click="monthOffset++" aria-label="Next" />
      <UButton class="ml-2" color="gray" variant="ghost" size="xs" @click="monthOffset = 0">[ today ]</UButton>
      <div class="ml-auto flex gap-1">
        <UButton size="2xs" :color="calMode === 'month' ? 'primary' : 'gray'" variant="ghost" @click="calMode = 'month'; monthOffset = 0">[ month ]</UButton>
        <UButton size="2xs" :color="calMode === 'week' ? 'primary' : 'gray'" variant="ghost" @click="calMode = 'week'; monthOffset = 0">[ week ]</UButton>
      </div>
    </div>
    <div class="grid grid-cols-7 gap-px text-xs">
      <div v-for="w in WEEKDAYS" :key="w" class="px-1 py-1 text-tertiary">{{ w }}</div>
      <div v-for="c in calCells" :key="c.date"
        :class="[calMode === 'week' ? 'min-h-[360px]' : 'min-h-[88px]', 'rounded-sm border p-1', c.inMonth ? 'border-subtle bg-surface' : 'border-transparent bg-base opacity-40', c.date === todayStr ? '!border-accent' : '']"
        @dragover.prevent @drop="dropOnDay(c.date)">
        <div class="mb-1 text-tertiary">{{ c.day }}</div>
        <div v-for="i in store.calendarIssues(teamId, c.date)" :key="i.id"
          draggable="true" @dragstart="draggedId = i.id" @click="emit('open-issue', i.id)"
          class="mb-0.5 cursor-pointer truncate rounded-sm bg-elevated px-1 py-0.5 hover:bg-hover"
          :title="i.identifier + ' ' + i.title">
          <span class="text-accent2">{{ i.identifier }}</span> {{ i.title }}
        </div>
      </div>
    </div>
  </div>
</template>
