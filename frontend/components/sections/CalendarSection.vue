<script setup lang="ts">
import { useAppStore } from '~/stores/app'
import { d10, fmtD, parseD, localDate, localHM, localMin, colorForUser } from '~/lib/ui'

// Персональный календарь: месяц-сетка + week/day тайм-сетка с drag-create/move/resize.
// Роутовый стейт (calAnchor) и модалки (событие / iCal) живут на странице:
// сетка отдаёт наверх чистые payload'ы через эмиты.
const props = defineProps<{ calUsers: string[] }>()
const emit = defineEmits<{
  (e: 'create', ds: string): void
  (e: 'create-at', p: { date: string; start: string; end: string }): void
  (e: 'edit-event', ev: any): void
  (e: 'open-issue', id: string): void
  (e: 'open-meetings'): void
  (e: 'ical'): void
}>()
const anchor = defineModel<string>('anchor', { required: true })

const store = useAppStore()
const { wrap } = useActionWrap()

const calView = ref<'month' | 'week' | 'day'>('month') // вид сетки
if (import.meta.client) { const v = localStorage.getItem('calView'); if (v === 'week' || v === 'day' || v === 'month') calView.value = v }
watch(calView, (v) => { if (import.meta.client) localStorage.setItem('calView', v) })
const HOUR_PX = 44 // высота часа в тайм-сетке (week/day)
const HOURS = Array.from({ length: 24 }, (_, h) => h)
// текущее время для «now»-линии (тикает раз в минуту)
const nowTick = ref(0)
let nowTimer: any = null
onMounted(() => { nowTimer = setInterval(() => nowTick.value++, 60000) })
onBeforeUnmount(() => clearInterval(nowTimer))
const nowMin = computed(() => { void nowTick.value; const d = new Date(); return d.getHours() * 60 + d.getMinutes() })
// листание зависит от вида: месяц ±1мес, неделя ±7д, день ±1д
function shiftRange(n: number) {
  const d = parseD(anchor.value)
  if (calView.value === 'month') d.setMonth(d.getMonth() + n)
  else d.setDate(d.getDate() + n * (calView.value === 'week' ? 7 : 1))
  anchor.value = fmtD(d)
}
function calToday() { anchor.value = fmtD(new Date()) }
const calTodayStr = computed(() => fmtD(new Date()))
const calHeaderLabel = computed(() => {
  const d = parseD(anchor.value)
  if (calView.value === 'month') return d.toLocaleString('en', { month: 'long', year: 'numeric' })
  if (calView.value === 'day') return d.toLocaleString('en', { weekday: 'short', month: 'short', day: 'numeric', year: 'numeric' })
  const w = weekDays.value
  return `${parseD(w[0]).toLocaleString('en', { month: 'short', day: 'numeric' })} – ${parseD(w[6]).toLocaleString('en', { month: 'short', day: 'numeric' })}`
})
// дни недели якоря (Mon..Sun)
const weekDays = computed(() => {
  const d = parseD(anchor.value); d.setDate(d.getDate() - ((d.getDay() + 6) % 7))
  return Array.from({ length: 7 }, (_, i) => { const x = new Date(d); x.setDate(d.getDate() + i); return fmtD(x) })
})
// дни активного диапазона тайм-сетки: week→7, day→1
const gridDays = computed(() => (calView.value === 'day' ? [anchor.value] : weekDays.value))
// 6×7 сетка месяца
const calGrid = computed(() => {
  const a = parseD(anchor.value); const m = a.getMonth()
  const first = new Date(a.getFullYear(), m, 1)
  const start = new Date(first); start.setDate(1 - ((first.getDay() + 6) % 7))
  return Array.from({ length: 42 }, (_, i) => {
    const d = new Date(start); d.setDate(start.getDate() + i); const ds = fmtD(d)
    return { ds, day: d.getDate(), inMonth: d.getMonth() === m, isToday: ds === calTodayStr.value }
  })
})
const weekdayLabel = (ds: string) => parseD(ds).toLocaleString('en', { weekday: 'short' })
// агрегатор на день (месяц-чипы): события + дедлайны + митинги, плоско
function calItemsForDay(ds: string) {
  const b = calDayBuckets(ds)
  return [...b.allDay, ...b.timed].sort((a, c) => (a.allDay === c.allDay ? 0 : a.allDay ? -1 : 1) || (a.sMin ?? -1) - (c.sMin ?? -1))
}
// повторяется ли событие в день ds (sd = дата первого вхождения)
function recursOn(e: any, ds: string, sd: string) {
  if (ds < sd) return false
  if (e.recurrenceUntil && ds > localDate(e.recurrenceUntil)) return false
  const a = parseD(sd), b = parseD(ds)
  if (e.recurrence === 'daily') return true
  if (e.recurrence === 'weekly') return a.getDay() === b.getDay()
  if (e.recurrence === 'monthly') return a.getDate() === b.getDate()
  return false
}
// разделяем на all-day и timed (с минутами) для тайм-сетки
function calDayBuckets(ds: string) {
  const allDay: any[] = [], timed: any[] = []
  for (const e of store.calendarEventsList) {
    if (!props.calUsers.includes(e.userId)) continue
    const sd = localDate(e.startsAt), ed = localDate(e.endsAt)
    const rec = e.recurrence && e.recurrence !== 'none'
    const onDay = rec ? recursOn(e, ds, sd) : (sd <= ds && ds <= ed)
    if (!onDay) continue
    const base = { id: e.id, title: e.title || '(no title)', color: colorForUser(e.userId), ev: e, kind: e.kind === 'dayoff' ? 'dayoff' : 'event', allDay: !!e.allDay }
    if (e.allDay) { allDay.push(base) } else if (rec) {
      // повтор: один день, время как у оригинала, длительность та же
      const sMin = localMin(e.startsAt), dur = Math.max(15, localMin(e.endsAt) - localMin(e.startsAt))
      timed.push({ ...base, sMin, eMin: Math.min(1440, sMin + dur), time: localHM(e.startsAt) })
    } else {
      const sMin = sd === ds ? localMin(e.startsAt) : 0
      const eMin = ed === ds ? localMin(e.endsAt) : 1440
      timed.push({ ...base, sMin, eMin: Math.max(eMin, sMin + 15), time: localHM(e.startsAt) })
    }
  }
  for (const m of store.scheduledMeetings) {
    if (m.scheduledAt && localDate(m.scheduledAt) === ds) { const sMin = localMin(m.scheduledAt); const eMin = m.endsAt && localDate(m.endsAt) === ds ? localMin(m.endsAt) : Math.min(1440, sMin + 30); timed.push({ kind: 'meeting', id: m.id, title: m.title || 'Meeting', color: '#1f9cf0', sMin, eMin: Math.max(eMin, sMin + 15), time: localHM(m.scheduledAt) }) }
  }
  for (const i of Object.values(store.issues)) {
    if (i.archivedAt || i.deletedAt) continue
    if (i.assigneeId && !props.calUsers.includes(i.assigneeId)) continue
    if ((d10(i.scheduledDate) || d10(i.dueDate)) === ds) allDay.push({ kind: 'issue', id: i.id, title: i.title, color: '#7c828c', identifier: i.identifier, allDay: true })
  }
  // overlap-раскладка: пересекающиеся события делят ширину на колонки (lane/cols)
  timed.sort((a, b) => a.sMin - b.sMin || a.eMin - b.eMin)
  let cluster: any[] = [], clusterEnd = -1, lanes: number[] = []
  const flush = () => { if (cluster.length) { const cols = Math.max(...cluster.map((e) => e.lane)) + 1; cluster.forEach((e) => (e.cols = cols)) } cluster = []; lanes = [] }
  for (const e of timed) {
    if (e.sMin >= clusterEnd) flush()
    let l = lanes.findIndex((end) => end <= e.sMin)
    if (l === -1) { l = lanes.length; lanes.push(e.eMin) } else lanes[l] = e.eMin
    e.lane = l; cluster.push(e); clusterEnd = Math.max(clusterEnd, e.eMin)
  }
  flush()
  return { allDay, timed }
}
// --- drag-create / drag-move в тайм-сетке ---
const SNAP = 15 // минут
const min2hm = (m: number) => `${String(Math.floor(m / 60)).padStart(2, '0')}:${String(m % 60).padStart(2, '0')}`
const isoFromLocalMin = (ds: string, min: number) => new Date(`${ds}T${min2hm(Math.max(0, Math.min(1439, min)))}:00`).toISOString()
const calDrag = reactive({ active: false, mode: '', ds: '', id: '', startMin: 0, curMin: 0 })
let calDragMoved = false
function snapY(el: HTMLElement, clientY: number) {
  const r = el.getBoundingClientRect()
  return Math.max(0, Math.min(1440, Math.round((clientY - r.top) / HOUR_PX * 60 / SNAP) * SNAP))
}
// pointerdown на пустом месте столбца → тянем диапазон → создаём событие
function dragCreate(ds: string, e: PointerEvent) {
  if (e.button !== 0) return
  const col = e.currentTarget as HTMLElement
  calDragMoved = false
  Object.assign(calDrag, { active: true, mode: 'create', ds, id: '', startMin: snapY(col, e.clientY) })
  calDrag.curMin = calDrag.startMin + SNAP
  const move = (ev: PointerEvent) => { calDrag.curMin = snapY(col, ev.clientY); calDragMoved = true }
  const up = () => {
    window.removeEventListener('pointermove', move); window.removeEventListener('pointerup', up)
    const a = Math.min(calDrag.startMin, calDrag.curMin), b = Math.max(calDrag.startMin, calDrag.curMin)
    const dur = b - a < SNAP ? 60 : b - a // клик (без протяжки) → 1 час
    calDrag.active = false
    emit('create-at', { date: ds, start: min2hm(a), end: min2hm(Math.min(1440, a + dur)) })
  }
  window.addEventListener('pointermove', move); window.addEventListener('pointerup', up)
}
// pointerdown на своём событии → тянем по вертикали → меняем время (тот же день)
function dragMove(it: any, ds: string, e: PointerEvent) {
  if (e.button !== 0 || (it.kind !== 'event' && it.kind !== 'dayoff')) return
  e.stopPropagation()
  const dur = it.eMin - it.sMin, origStart = it.sMin, startY = e.clientY
  Object.assign(calDrag, { active: true, mode: 'move', ds, id: it.id, startMin: origStart, curMin: origStart })
  calDragMoved = false
  const move = (ev: PointerEvent) => {
    const delta = Math.round((ev.clientY - startY) / HOUR_PX * 60 / SNAP) * SNAP
    if (delta !== 0) calDragMoved = true
    calDrag.curMin = Math.max(0, Math.min(1440 - dur, origStart + delta))
  }
  const up = async () => {
    window.removeEventListener('pointermove', move); window.removeEventListener('pointerup', up)
    const ns = calDrag.curMin; calDrag.active = false
    if (calDragMoved && ns !== origStart) {
      const ev = store.calendarEvents[it.id]
      if (ev) await wrap(() => store.updateCalendarEvent(it.id, { title: ev.title, description: ev.description, kind: ev.kind, startsAt: isoFromLocalMin(ds, ns), endsAt: isoFromLocalMin(ds, ns + dur), allDay: false, recurrence: ev.recurrence, recurrenceUntil: ev.recurrenceUntil }))
    }
  }
  window.addEventListener('pointermove', move); window.addEventListener('pointerup', up)
}
// pointerdown на нижней ручке события → тянем низ → меняем длительность (старт фиксирован)
function dragResize(it: any, ds: string, e: PointerEvent) {
  if (e.button !== 0 || (it.kind !== 'event' && it.kind !== 'dayoff')) return
  e.stopPropagation()
  const origStart = it.sMin, startY = e.clientY, origEnd = it.eMin
  Object.assign(calDrag, { active: true, mode: 'resize', ds, id: it.id, startMin: origStart, curMin: origEnd })
  calDragMoved = false
  const move = (ev: PointerEvent) => {
    const delta = Math.round((ev.clientY - startY) / HOUR_PX * 60 / SNAP) * SNAP
    if (delta !== 0) calDragMoved = true
    calDrag.curMin = Math.max(origStart + SNAP, Math.min(1440, origEnd + delta))
  }
  const up = async () => {
    window.removeEventListener('pointermove', move); window.removeEventListener('pointerup', up)
    const ne = calDrag.curMin; calDrag.active = false
    if (calDragMoved && ne !== origEnd) {
      const ev = store.calendarEvents[it.id]
      if (ev) await wrap(() => store.updateCalendarEvent(it.id, { title: ev.title, description: ev.description, kind: ev.kind, startsAt: isoFromLocalMin(ds, origStart), endsAt: isoFromLocalMin(ds, ne), allDay: false, recurrence: ev.recurrence, recurrenceUntil: ev.recurrenceUntil }))
    }
  }
  window.addEventListener('pointermove', move); window.addEventListener('pointerup', up)
}
function calClickItem(it: any) {
  if (calDragMoved) { calDragMoved = false; return } // был drag — не открывать
  if (it.kind === 'event' || it.kind === 'dayoff') emit('edit-event', it.ev)
  else if (it.kind === 'issue') emit('open-issue', it.id)
  else emit('open-meetings')
}
</script>

<template>
  <ClientOnly>
  <div class="sticky top-0 z-10 flex items-center gap-2 border-b border-subtle bg-surface px-4 py-2.5 text-sm">
    <UIcon name="i-heroicons-calendar-days" class="text-tertiary" />
    <span class="font-medium">{{ calHeaderLabel }}</span>
    <div class="ml-2 flex items-center gap-1">
      <UButton color="gray" variant="ghost" size="xs" square @click="shiftRange(-1)" />
      <UButton color="gray" variant="ghost" size="xs" @click="calToday">[ today ]</UButton>
      <UButton icon="i-heroicons-chevron-right" color="gray" variant="ghost" size="xs" square @click="shiftRange(1)" />
    </div>
    <div class="ml-2 flex items-center gap-0.5 rounded-md bg-elevated p-0.5">
      <button v-for="v in ['month','week','day']" :key="v"
        class="rounded px-2 py-0.5 text-xs capitalize transition-colors"
        :class="calView === v ? 'bg-active text-white' : 'text-tertiary hover:text-secondary'"
        @click="calView = v as any">{{ v }}</button>
    </div>
    <UButton class="ml-auto" size="xs" color="gray" variant="ghost" @click="$emit('ical')">[ subscribe ]</UButton>
    <UButton size="xs" color="gray" variant="ghost" @click="$emit('create', anchor)">[ new event ]</UButton>
  </div>

  <!-- MONTH -->
  <template v-if="calView === 'month'">
    <div class="grid grid-cols-7 border-b border-subtle text-center text-xs font-medium uppercase tracking-wider text-tertiary">
      <div v-for="wd in ['Mon','Tue','Wed','Thu','Fri','Sat','Sun']" :key="wd" class="py-1.5">{{ wd }}</div>
    </div>
    <div class="grid grid-cols-7" style="grid-auto-rows: minmax(96px, 1fr)">
      <div v-for="cell in calGrid" :key="cell.ds"
        class="group/cell relative cursor-pointer border-b border-r border-subtle/50 p-1 text-left align-top transition hover:bg-hover/40"
        :class="cell.inMonth ? '' : 'bg-elevated/30'"
        @click="$emit('create', cell.ds)">
        <div class="flex items-center justify-between px-0.5">
          <span class="text-xs" :class="cell.isToday ? 'flex h-5 w-5 items-center justify-center rounded bg-accent font-semibold text-base' : cell.inMonth ? 'text-secondary' : 'text-tertiary/50'">{{ cell.day }}</span>
        </div>
        <div class="mt-0.5 space-y-0.5">
          <div v-for="it in calItemsForDay(cell.ds).slice(0, 4)" :key="it.kind + it.id"
            class="flex cursor-pointer items-center gap-1 truncate rounded px-1 py-0.5 text-[11px] leading-tight transition hover:brightness-125"
            :class="it.kind === 'issue' || it.kind === 'meeting' ? 'bg-hover text-secondary hover:bg-active' : 'text-white'"
            :style="it.kind === 'event' || it.kind === 'dayoff' ? { background: it.color + '33', borderLeft: `2px solid ${it.color}` } : {}"
            @click.stop="calClickItem(it)">
            <UIcon v-if="it.kind === 'meeting'" name="i-heroicons-video-camera" class="shrink-0 text-[10px]" />
            <UIcon v-else-if="it.kind === 'issue'" name="i-heroicons-flag" class="shrink-0 text-[10px]" />
            <UIcon v-else-if="it.kind === 'dayoff'" name="i-heroicons-sun" class="shrink-0 text-[10px]" />
            <span v-if="it.time" class="shrink-0 tabular-nums opacity-70">{{ it.time }}</span>
            <span class="truncate">{{ it.identifier ? it.identifier + ' ' : '' }}{{ it.title }}</span>
          </div>
          <div v-if="calItemsForDay(cell.ds).length > 4" class="px-1 text-[10px] text-tertiary">+{{ calItemsForDay(cell.ds).length - 4 }} more</div>
        </div>
      </div>
    </div>
  </template>

  <!-- WEEK / DAY — тайм-сетка -->
  <template v-else>
    <!-- шапка дней -->
    <div class="grid border-b border-subtle" :style="{ gridTemplateColumns: `52px repeat(${gridDays.length}, minmax(0,1fr))` }">
      <div />
      <div v-for="ds in gridDays" :key="ds" class="cursor-pointer border-l border-subtle/50 py-1.5 text-center hover:bg-hover" @click="anchor = ds; calView = 'day'">
        <div class="text-[11px] uppercase tracking-wider text-tertiary">{{ weekdayLabel(ds) }}</div>
        <div class="text-sm" :class="ds === calTodayStr ? 'mx-auto flex h-6 w-6 items-center justify-center rounded bg-accent font-semibold text-base' : 'text-secondary'">{{ parseD(ds).getDate() }}</div>
      </div>
    </div>
    <!-- all-day полоса -->
    <div class="grid border-b border-subtle" :style="{ gridTemplateColumns: `52px repeat(${gridDays.length}, minmax(0,1fr))` }">
      <div class="py-1 pr-1 text-right text-[10px] uppercase text-tertiary">all-day</div>
      <div v-for="ds in gridDays" :key="ds" class="min-h-[26px] space-y-0.5 border-l border-subtle/50 p-0.5">
        <div v-for="it in calDayBuckets(ds).allDay" :key="it.kind + it.id"
          class="flex cursor-pointer items-center gap-1 truncate rounded px-1 py-0.5 text-[11px] leading-tight transition hover:brightness-125"
          :class="it.kind === 'issue' ? 'bg-hover text-secondary hover:bg-active' : 'text-white'"
          :style="it.kind === 'event' || it.kind === 'dayoff' ? { background: it.color + '33', borderLeft: `2px solid ${it.color}` } : {}"
          @click.stop="calClickItem(it)">
          <UIcon v-if="it.kind === 'issue'" name="i-heroicons-flag" class="shrink-0 text-[10px]" />
          <UIcon v-else-if="it.kind === 'dayoff'" name="i-heroicons-sun" class="shrink-0 text-[10px]" />
          <span class="truncate">{{ it.identifier ? it.identifier + ' ' : '' }}{{ it.title }}</span>
        </div>
      </div>
    </div>
    <!-- часовая сетка -->
    <div class="grid" :style="{ gridTemplateColumns: `52px repeat(${gridDays.length}, minmax(0,1fr))`, height: HOUR_PX * 24 + 'px' }">
      <div class="relative">
        <div v-for="h in HOURS" :key="h" class="absolute right-1.5 -translate-y-1/2 text-[10px] tabular-nums text-tertiary" :style="{ top: h * HOUR_PX + 'px' }">{{ h > 0 ? String(h).padStart(2, '0') + ':00' : '' }}</div>
      </div>
      <div v-for="ds in gridDays" :key="ds" class="relative select-none border-l border-subtle/50" style="cursor: cell" @pointerdown.self="dragCreate(ds, $event)">
        <div v-for="h in HOURS" :key="h" class="pointer-events-none absolute inset-x-0 border-t border-subtle/25" :style="{ top: h * HOUR_PX + 'px' }" />
        <!-- now-линия -->
        <div v-if="ds === calTodayStr" class="pointer-events-none absolute inset-x-0 z-20 border-t border-danger" :style="{ top: nowMin / 60 * HOUR_PX + 'px' }">
          <span class="absolute -left-1 -top-1 h-2 w-2 rounded-full bg-danger" />
        </div>
        <!-- drag-create превью -->
        <div v-if="calDrag.active && calDrag.mode === 'create' && calDrag.ds === ds" class="pointer-events-none absolute inset-x-0.5 z-10 rounded bg-accent/30 ring-1 ring-accent"
          :style="{ top: Math.min(calDrag.startMin, calDrag.curMin) / 60 * HOUR_PX + 'px', height: Math.max(4, Math.abs(calDrag.curMin - calDrag.startMin) / 60 * HOUR_PX) + 'px' }" />
        <div v-for="it in calDayBuckets(ds).timed" :key="it.kind + it.id"
          class="group/ev absolute overflow-hidden rounded px-1 text-[11px] leading-tight ring-1 ring-inset ring-white/10 transition hover:z-10 hover:brightness-125 hover:ring-white/40"
          :class="(it.kind === 'event' || it.kind === 'dayoff') ? 'cursor-grab active:cursor-grabbing' : 'cursor-pointer'"
          :style="{ top: ((calDrag.active && calDrag.mode === 'move' && calDrag.id === it.id ? calDrag.curMin : it.sMin) / 60 * HOUR_PX) + 'px', height: Math.max(16, ((calDrag.active && calDrag.mode === 'resize' && calDrag.id === it.id ? calDrag.curMin - it.sMin : it.eMin - it.sMin)) / 60 * HOUR_PX) + 'px', left: `calc(${(it.lane || 0) / (it.cols || 1) * 100}% + 1px)`, width: `calc(${100 / (it.cols || 1)}% - 2px)`, background: it.color + '33', borderLeft: `2px solid ${it.color}` }"
          @pointerdown="dragMove(it, ds, $event)" @click.stop="calClickItem(it)">
          <span class="font-medium tabular-nums">{{ it.time }}</span>
          <span class="ml-1">{{ it.kind === 'meeting' ? '📹 ' : '' }}{{ it.title }}</span>
          <div v-if="it.kind === 'event' || it.kind === 'dayoff'" class="absolute inset-x-0 bottom-0 h-1.5 cursor-ns-resize opacity-0 group-hover/ev:opacity-100"
            @pointerdown.stop="dragResize(it, ds, $event)" />
        </div>
      </div>
    </div>
  </template>
  </ClientOnly>
</template>
