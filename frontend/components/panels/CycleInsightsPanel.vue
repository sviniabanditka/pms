<script setup lang="ts">
import { useAppStore } from '~/stores/app'
import { PRIO, d10 } from '~/lib/ui'

// Правая панель инсайтов цикла: прогресс, burndown, breakdown с фильтрами.
// Роутовое состояние (selectedCycleId) живёт на странице → сюда приходит cycle-id.
const props = defineProps<{ cycleId: string }>()
const emit = defineEmits<{ (e: 'close'): void }>()

const store = useAppStore()
const { wrap } = useActionWrap()
const { prioColor } = useColors()
const cyclePanel = useResizable('cycle', 320)

// burndown: live-расчёт из completedAt issues цикла (scope константа = текущее число).
const cycleBurndown = computed(() => {
  const c = store.cycles[props.cycleId]
  if (!c) return null
  const iss = store.issuesInCycle(c.id)
  const scope = iss.length
  const start = String(c.startsOn).slice(0, 10)
  const end = String(c.endsOn).slice(0, 10)
  const days: string[] = []
  let d = new Date(start + 'T00:00:00Z')
  const e = new Date(end + 'T00:00:00Z')
  while (d <= e && days.length < 120) { days.push(d.toISOString().slice(0, 10)); d = new Date(d.getTime() + 86400000) }
  const stats = store.cycleStats(c.id)
  if (days.length < 2) return { ...stats, idealPts: '', actualPts: '' }
  const completed = iss.filter((i) => i.completedAt).map((i) => String(i.completedAt).slice(0, 10))
  const W = 280, H = 70
  const pt = (idx: number, val: number) => `${(idx / (days.length - 1)) * W},${(H - (scope ? (val / scope) * H : 0)).toFixed(1)}`
  const idealPts = days.map((_, i) => pt(i, scope * (1 - i / (days.length - 1)))).join(' ')
  const actualPts = days.map((day, i) => pt(i, scope - completed.filter((cd) => cd <= day).length)).join(' ')
  const areaPts = `0,${H} ${actualPts} ${W},${H}`
  const fmt = (s: string) => new Date(s + 'T00:00:00Z').toLocaleDateString('en', { month: 'short', day: 'numeric' })
  return { ...stats, idealPts, actualPts, areaPts, startLabel: fmt(start), midLabel: fmt(days[Math.floor(days.length / 2)]), endLabel: fmt(end) }
})
const insightTab = ref<'assignees' | 'priority' | 'labels' | 'milestones' | 'teams'>('priority')
if (import.meta.client) {
  const s = localStorage.getItem('cycleInsightTab')
  if (s) insightTab.value = s as any
}
watch(insightTab, (v) => { if (import.meta.client) localStorage.setItem('cycleInsightTab', v) })
const INSIGHT_TABS = ['Assignees', 'Labels', 'Priority', 'Milestones', 'Teams']
const cycleRange = computed(() => {
  const c = store.cycles[props.cycleId]
  if (!c) return ''
  const f = (d: string) => new Date(d).toLocaleDateString('en', { month: 'short', day: 'numeric' })
  return `${f(c.startsOn)} → ${f(c.endsOn)}`
})
// edit cycle (name + dates) через UpdateCycle
const cycleEdit = ref(false)
const cycleForm = reactive({ name: '', startsOn: '', endsOn: '' })
function startCycleEdit() {
  const c = store.cycles[props.cycleId]; if (!c) return
  cycleForm.name = c.name || `Cycle ${c.number}`; cycleForm.startsOn = d10(c.startsOn); cycleForm.endsOn = d10(c.endsOn)
  cycleEdit.value = true
}
const saveCycle = () => wrap(async () => {
  await store.updateCycle(props.cycleId, { name: cycleForm.name, startsOn: cycleForm.startsOn, endsOn: cycleForm.endsOn })
  cycleEdit.value = false
})
const cycleStatus = computed(() => {
  const c = store.cycles[props.cycleId]; if (!c) return 'Cycle'
  const today = new Date().toISOString().slice(0, 10)
  if (today < c.startsOn) return 'Upcoming'
  if (today > c.endsOn) return 'Past'
  return 'Current'
})
const BREAKDOWN_PALETTE = ['#5e6ad2', '#26a69a', '#d29922', '#e5484d', '#9c5cf6', '#0ea5e9', '#f76808', '#46a758']
const NONE = '__none__' // сентинел «без значения» (isNotSet)
// фильтр по любой breakdown-строке: priority/assignee/milestone/team — predicate; label — labelFilter.
function activeValueFor(field: string) {
  const f = store.activeFilter
  const kids = f && f.type === 'and' ? f.children : f ? [f] : []
  const p: any = kids.find((c: any) => c.type === 'predicate' && c.field === field && (c.op === 'eq' || c.op === 'isNotSet'))
  if (!p) return null
  return p.op === 'isNotSet' ? NONE : p.value
}
function rowActive(r: any) {
  if (r.kind === 'label') return store.labelFilter.includes(r.value)
  return activeValueFor(r.field) === r.value
}
function toggleRow(r: any) {
  if (r.kind === 'label') { store.toggleLabelFilter(r.value); return }
  const next = activeValueFor(r.field) === r.value ? null : r.value
  store.setFieldPredicate(r.field, next === null ? null
    : next === NONE ? { type: 'predicate', field: r.field, op: 'isNotSet' }
    : { type: 'predicate', field: r.field, op: 'eq', value: next })
}
const cycleBreakdown = computed(() => {
  const iss = store.issuesInCycle(props.cycleId)
  const total = iss.length || 1
  const pct = (n: number) => Math.round((n / total) * 100)
  if (insightTab.value === 'priority') {
    return PRIO.map((p, idx) => ({ field: 'priority', value: idx, label: p, color: prioColor(idx), count: iss.filter((i) => i.priority === idx).length }))
      .filter((r) => r.count).map((r) => ({ ...r, pct: pct(r.count) }))
  }
  if (insightTab.value === 'assignees') {
    const ids = Array.from(new Set(iss.map((i) => i.assigneeId).filter(Boolean)))
    const rows = ids.map((uid, n) => ({
      field: 'assignee', value: uid as string, label: store.userName(uid as string),
      color: BREAKDOWN_PALETTE[n % BREAKDOWN_PALETTE.length],
      count: iss.filter((i) => i.assigneeId === uid).length,
    })).map((r) => ({ ...r, pct: pct(r.count) }))
    const none = iss.filter((i) => !i.assigneeId).length
    if (none) rows.push({ field: 'assignee', value: NONE, label: 'No assignee', color: '#7c828c', count: none, pct: pct(none) })
    return rows
  }
  if (insightTab.value === 'labels') {
    // один проход по issues цикла через кэш-индекс labelsByIssue (раньше O(labels×issues×issueLabels))
    const lbi = store.labelsByIssue
    const counts = new Map<string, number>()
    for (const i of iss) { const ids = lbi.get(i.id); if (ids) for (const lid of ids) counts.set(lid, (counts.get(lid) || 0) + 1) }
    return Array.from(counts.entries()).map(([lid, count], n) => {
      const l: any = store.labels[lid]
      return { kind: 'label', value: lid, label: l?.name || '?', color: l?.colorToken || BREAKDOWN_PALETTE[n % BREAKDOWN_PALETTE.length], count, pct: pct(count) }
    }).filter((r) => r.count)
  }
  if (insightTab.value === 'milestones') {
    // один проход (раньше O(milestones×issues))
    const counts = new Map<string, number>()
    let none = 0
    for (const i of iss) { if (i.milestoneId) counts.set(i.milestoneId, (counts.get(i.milestoneId) || 0) + 1); else none++ }
    const rows = Array.from(counts.entries()).map(([mid, count], n) => ({
      field: 'milestone', value: mid, label: (store.milestones[mid] as any)?.name || '?',
      color: BREAKDOWN_PALETTE[n % BREAKDOWN_PALETTE.length], count, pct: pct(count),
    }))
    if (none) rows.push({ field: 'milestone', value: NONE, label: 'No milestone', color: '#7c828c', count: none, pct: pct(none) })
    return rows
  }
  if (insightTab.value === 'teams') {
    return Object.values(store.teams).map((t: any, n: number) => ({
      field: 'team', value: t.id, label: `${t.key} · ${t.name}`, color: BREAKDOWN_PALETTE[n % BREAKDOWN_PALETTE.length],
      count: iss.filter((i) => i.teamId === t.id).length,
    })).filter((r) => r.count).map((r) => ({ ...r, pct: pct(r.count) }))
  }
  return []
})
</script>

<template>
  <ResizeHandle @down="cyclePanel.onResizeStart" />
  <aside v-if="cycleBurndown" :style="{ width: cyclePanel.w.value + 'px' }" class="shrink-0 overflow-y-auto rounded-xl border border-subtle bg-surface px-4 py-3 text-sm">
    <!-- header: Current pill + range -->
    <div class="mb-3 flex items-center gap-2 text-sm">
      <span class="rounded-md px-2 py-0.5 text-xs font-medium"
        :class="cycleStatus === 'Current' ? 'bg-accent/20 text-accent' : 'bg-elevated text-secondary'">{{ cycleStatus }}</span>
      <span class="text-tertiary">{{ cycleRange }}</span>
      <UButton class="ml-auto" size="2xs" color="gray" variant="ghost" square icon="i-heroicons-x-mark" @click="emit('close')" aria-label="Close" />
    </div>
    <!-- title -->
    <div class="mb-4 flex items-center gap-2">
      <UIcon name="i-heroicons-play-circle" class="text-accent" />
      <span class="text-base font-semibold text-white">{{ store.cycles[cycleId]?.name || `Cycle ${store.cycles[cycleId]?.number}` }}</span>
      <UButton class="ml-auto" icon="i-heroicons-pencil-square" color="gray" variant="ghost" size="xs" square title="Edit cycle" @click="startCycleEdit" aria-label="Edit cycle" />
    </div>

    <!-- edit form -->
    <div v-if="cycleEdit" class="mb-4 space-y-2 rounded-lg border border-subtle bg-elevated p-3">
      <UInput v-model="cycleForm.name" size="xs" placeholder="Cycle name (optional)" />
      <div class="flex gap-2">
        <div class="flex-1"><label class="mb-1 block text-[11px] text-tertiary">Start</label><DatePicker v-model="cycleForm.startsOn" block /></div>
        <div class="flex-1"><label class="mb-1 block text-[11px] text-tertiary">End</label><DatePicker v-model="cycleForm.endsOn" block /></div>
      </div>
      <div class="flex justify-end gap-2"><UButton size="2xs" color="gray" variant="ghost" @click="cycleEdit = false">[ cancel ]</UButton><UButton size="2xs" @click="saveCycle">[ save ]</UButton></div>
    </div>

    <!-- Progress -->
    <div class="mb-2 text-xs font-medium uppercase tracking-wider text-tertiary"><span class="text-ok"># </span>Progress</div>
    <div class="mb-3 grid grid-cols-3 gap-2 text-xs">
      <div>
        <div class="flex items-center gap-1.5 text-tertiary"><span class="h-2 w-2 rounded-sm bg-tertiary" />Scope</div>
        <div class="mt-1 font-medium">{{ cycleBurndown.scope }}</div>
      </div>
      <div>
        <div class="flex items-center gap-1.5 text-tertiary"><span class="h-2 w-2 rounded-sm" style="background:#d29922" />Started</div>
        <div class="mt-1 font-medium">{{ cycleBurndown.started }} <span class="text-tertiary">· {{ cycleBurndown.scope ? Math.round(cycleBurndown.started / cycleBurndown.scope * 100) : 0 }}%</span></div>
      </div>
      <div>
        <div class="flex items-center gap-1.5 text-tertiary"><span class="h-2 w-2 rounded-sm bg-accent" />Completed</div>
        <div class="mt-1 font-medium">{{ cycleBurndown.completed }} <span class="text-tertiary">· {{ cycleBurndown.scope ? Math.round(cycleBurndown.completed / cycleBurndown.scope * 100) : 0 }}%</span></div>
      </div>
    </div>
    <div class="flex items-center gap-4 text-xs">
      <span class="text-tertiary">Points <span class="text-secondary">{{ cycleBurndown.donePoints }}/{{ cycleBurndown.scopePoints }}</span></span>
      <span class="text-tertiary">Velocity <span class="text-secondary">{{ store.teamVelocity(store.cycles[cycleId]?.teamId) }}</span> pts/cycle</span>
    </div>

    <div class="my-4 h-px bg-subtle" />

    <!-- burndown chart (реальные ежедневные слепки, River) -->
    <div class="mb-1 text-xs font-medium uppercase tracking-wider text-tertiary"><span class="text-ok"># </span>Burndown</div>
    <Burndown :cycle-id="cycleId" />

    <div class="my-4 h-px bg-subtle" />

    <!-- breakdown -->
    <div class="mb-2 text-xs font-medium uppercase tracking-wider text-tertiary"><span class="text-ok"># </span>Breakdown</div>
    <!-- tabs -->
    <div class="mb-3 flex flex-wrap gap-1 text-xs">
      <button v-for="t in INSIGHT_TABS" :key="t"
        class="rounded-md px-2 py-1"
        :class="insightTab === t.toLowerCase() ? 'bg-elevated font-medium' : 'text-tertiary hover:bg-hover'"
        @click="insightTab = t.toLowerCase() as any">{{ t }}</button>
    </div>
    <!-- breakdown -->
    <div class="space-y-1">
      <div v-for="r in cycleBreakdown" :key="r.label"
        class="flex cursor-pointer items-center gap-2 rounded-md px-2 py-1.5"
        :class="rowActive(r) ? 'bg-accent/15 ring-1 ring-accent/40' : 'hover:bg-hover'"
        @click="toggleRow(r)">
        <span class="h-2.5 w-2.5 shrink-0 rounded-full" :style="{ background: r.color }" />
        <span class="flex-1 truncate">{{ r.label }}</span>
        <UIcon v-if="rowActive(r)" name="i-heroicons-funnel" class="text-accent" />
        <span class="text-tertiary">{{ r.pct }}% of {{ r.count }}</span>
      </div>
      <div v-if="!cycleBreakdown.length" class="px-2 py-3 text-xs text-tertiary">No data for {{ insightTab }}</div>
    </div>
    <p class="mt-1 px-2 text-[11px] text-tertiary">Click a row to filter the list · click again to clear.</p>
  </aside>
</template>
