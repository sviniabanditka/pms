<script setup lang="ts">
import { isAuthed, clearAuth } from '~/lib/token'
import { fileViewUrl } from '~/lib/upload'
import { useAppStore } from '~/stores/app'
import { PRIO, cap, labelOpts, propSelUi, modalUi, STATE_CATS, REL_TYPES, ROLES, notifTime } from '~/lib/ui'
import { useVirtualizer } from '@tanstack/vue-virtual'

const store = useAppStore()
// Разделы левого рейла (расширяется: calendar, chats…)
const SECTIONS = [
  { key: 'dashboard', label: 'Dashboard', icon: 'i-heroicons-squares-2x2' },
  { key: 'inbox', label: 'Inbox', icon: 'i-heroicons-inbox' },
  { key: 'tasks', label: 'Tasks', icon: 'i-heroicons-queue-list' },
  { key: 'calendar', label: 'Calendar', icon: 'i-heroicons-calendar-days' },
  { key: 'reports', label: 'Reports', icon: 'i-heroicons-chart-bar' },
  { key: 'chats', label: 'Chats', icon: 'i-heroicons-chat-bubble-left-right' },
  { key: 'meetings', label: 'Meetings', icon: 'i-heroicons-video-camera' },
]
// начальная секция из URL (SSR-safe, чтобы тайтл вкладки совпадал с роутом сразу;
// полную привязку к сущностям делает applyRoute клиентски в onMounted)
const SECTION_BY_SEG: Record<string, 'dashboard' | 'tasks' | 'inbox' | 'chats' | 'meetings' | 'calendar' | 'reports'> =
  { inbox: 'inbox', chats: 'chats', chat: 'chats', meetings: 'meetings', calendar: 'calendar', reports: 'reports', dashboard: 'dashboard' }
const activeSection = ref<'dashboard' | 'tasks' | 'inbox' | 'chats' | 'meetings' | 'calendar' | 'reports'>(
  SECTION_BY_SEG[useRoute().path.split('/').filter(Boolean)[0] || ''] || 'tasks')
// видимые разделы по конфигу пользователя (порядок + видимость); новые — в конец
const railSections = computed(() => {
  const saved = (store.mySectionsConfig?.sections || []) as { key: string; visible?: boolean }[]
  const byKey = Object.fromEntries(SECTIONS.map((s) => [s.key, s]))
  if (!saved.length) return SECTIONS
  const out = saved.filter((c) => byKey[c.key] && c.visible !== false).map((c) => byKey[c.key])
  const inCfg = new Set(saved.map((c) => c.key))
  for (const s of SECTIONS) if (!inCfg.has(s.key)) out.push(s)
  return out
})
// панель настройки рейла
const showRailCfg = ref(false)
const cfgList = ref<{ key: string; label: string; icon: string; visible: boolean }[]>([])
function openRailCfg() {
  const saved = (store.mySectionsConfig?.sections || []) as { key: string; visible?: boolean }[]
  const byKey = Object.fromEntries(SECTIONS.map((s) => [s.key, s]))
  const order = saved.length
    ? saved.filter((c) => byKey[c.key]).map((c) => ({ ...byKey[c.key], visible: c.visible !== false }))
    : SECTIONS.map((s) => ({ ...s, visible: true }))
  const inOrder = new Set(order.map((o) => o.key))
  for (const s of SECTIONS) if (!inOrder.has(s.key)) order.push({ ...s, visible: true })
  cfgList.value = order
  showRailCfg.value = true
}
function saveRailCfg() {
  store.updateSectionsConfig({ sections: cfgList.value.map((c) => ({ key: c.key, visible: c.visible })) })
}
function moveSection(i: number, dir: number) {
  const a = cfgList.value; const j = i + dir
  if (j < 0 || j >= a.length) return
  ;[a[i], a[j]] = [a[j], a[i]]
  saveRailCfg()
}
function toggleSection(i: number) { cfgList.value[i].visible = !cfgList.value[i].visible; saveRailCfg() }

function switchSection(key: 'dashboard' | 'tasks' | 'inbox' | 'chats' | 'meetings' | 'calendar' | 'reports') {
  activeSection.value = key
  if (key === 'dashboard' && !selectedDashboardId.value && store.workspaceDashboards.length) {
    selectedDashboardId.value = store.workspaceDashboards[0].id
  }
}

const slug = ref('')
const name = ref('')
const teamKey = ref('')
const teamName = ref('')
const teamParent = ref('')
const selectedTeamId = ref('')
const newIssue = reactive<Record<string, string>>({}) // stateId -> draft title
const { err, wrap } = useActionWrap()

const { statusColor, statusGlyph, prioColor, prioGlyph } = useColors()
const selectedTeam = computed(() => store.teams[selectedTeamId.value])
const draggedId = ref('')

// --- Issue detail panel (M4) ---
const selectedIssueId = ref('')
const selectedIssue = computed(() => store.issues[selectedIssueId.value])
const newSub = ref('')
const showLabelModal = ref(false)
const showRelModal = ref(false)
// issue description (markdown, inline edit + md toolbar)
const descDraft = ref('')
const descEditing = ref(false)
const descEl = ref<HTMLTextAreaElement | null>(null)
function startDescEdit() { descDraft.value = selectedIssue.value?.description?.md || ''; descEditing.value = true; nextTick(() => descEl.value?.focus()) }
function saveDesc() { descEditing.value = false; if (selectedIssue.value) store.setIssueDescription(selectedIssue.value.id, descDraft.value) }
function descWrap(pre: string, post = pre) {
  const el = descEl.value; if (!el) { descDraft.value += pre + post; return }
  const s = el.selectionStart, e = el.selectionEnd, v = descDraft.value
  descDraft.value = v.slice(0, s) + pre + v.slice(s, e) + post + v.slice(e)
  nextTick(() => { el.focus(); el.selectionStart = s + pre.length; el.selectionEnd = e + pre.length })
}
// AI-резюме описания (Фаза 4)
const descSummary = ref(''); const descSummaryLoading = ref(false)
async function summarizeDesc() {
  if (!selectedIssue.value) return
  descSummaryLoading.value = true
  descSummary.value = await store.summarizeIssue(selectedIssue.value.id)
  descSummaryLoading.value = false
}
watch(selectedIssueId, () => { descSummary.value = '' })
const aiSuggest = ref<any>(null)
const simIssues = computed(() => (aiSuggest.value?.similar || []).filter((s: any) => s.id !== selectedIssueId.value))
function openIssue(id: string) {
  selectedIssueId.value = id; descEditing.value = false; store.loadComments(id); store.loadActivity(id)
  aiSuggest.value = null
  store.suggestForIssue(store.issues[id]).then((r) => { if (selectedIssueId.value === id) aiSuggest.value = r })
}
// подсказки, которых у issue ещё нет
const suggestLabels = computed(() => {
  const iss = selectedIssue.value; if (!iss || !aiSuggest.value) return []
  const have = store.labelsByIssue.get(iss.id) || new Set()
  return (aiSuggest.value.labelIds || []).filter((lid: string) => !have.has(lid) && store.labels[lid]).slice(0, 3)
})
const suggestAssignee = computed(() => {
  const iss = selectedIssue.value; if (!iss || !aiSuggest.value?.assigneeId) return null
  return (!iss.assigneeId && aiSuggest.value.assigneeId !== iss.assigneeId) ? aiSuggest.value.assigneeId : null
})
const PRI_LABELS = ['none', 'urgent', 'high', 'medium', 'low']
const suggestPriority = computed(() => {
  const iss = selectedIssue.value; const p = aiSuggest.value?.priority
  if (!iss || p == null || p === iss.priority) return null
  return p
})
const suggestEstimate = computed(() => {
  const iss = selectedIssue.value; const e = aiSuggest.value?.estimate
  if (!iss || e == null || iss.estimate != null) return null
  return Math.max(1, Math.round(e))
})
const slipWarn = computed(() => {
  const iss = selectedIssue.value
  return !!(aiSuggest.value?.slipRisk && iss?.dueDate && !iss.completedAt)
})
// bulk selection (List)
const selectedIds = ref<Set<string>>(new Set())
const selCount = computed(() => selectedIds.value.size)
function toggleSel(id: string) { const s = new Set(selectedIds.value); s.has(id) ? s.delete(id) : s.add(id); selectedIds.value = s }
function clearSel() { selectedIds.value = new Set() }
const bulkDo = (fn: (id: string) => Promise<any>) => wrap(async () => { for (const id of Array.from(selectedIds.value)) await fn(id); clearSel() })
const bulkStatus = (sid: any) => sid && bulkDo((id) => store.setIssueState(id, sid))
const bulkAssignee = (uid: any) => bulkDo((id) => store.setIssueAssignee(id, uid || ''))
const bulkLabel = (lid: any) => lid && bulkDo((id) => store.addLabel(id, lid))
const bulkArchive = () => bulkDo((id) => store.archiveIssue(id))

// milestones
const selectedMsId = ref('')
const selectedMs = computed(() => store.milestones[selectedMsId.value])
function openMilestone(id: string) { selectedMsId.value = id; selectedIssueId.value = '' }
const todayISO = new Date().toISOString().slice(0, 10)
function msOverdue(m: any) { return m.targetDate && String(m.targetDate).slice(0, 10) < todayISO && m.status !== 'done' }
// workflow states manager
const showStates = ref(false)
// comments
const commentDraft = ref('')
const commentEl = ref<HTMLTextAreaElement | null>(null)
const commentPending = ref<File[]>([])
const commentFileEl = ref<HTMLInputElement | null>(null)
const onCommentFilePick = (e: Event) => {
  const files = (e.target as HTMLInputElement).files; if (!files?.length) return
  for (const f of Array.from(files)) commentPending.value = [...commentPending.value, f]
  if (commentFileEl.value) commentFileEl.value.value = ''
}
const onCommentPaste = (e: ClipboardEvent) => {
  const items = e.clipboardData?.items; if (!items) return
  for (const it of Array.from(items)) if (it.type.startsWith('image/')) { e.preventDefault(); const f = it.getAsFile(); if (f) commentPending.value = [...commentPending.value, f] }
}
const editCommentId = ref('')
const editCommentDraft = ref('')
const addComment = () => wrap(async () => {
  const b = commentDraft.value.trim()
  if ((!b && !commentPending.value.length) || !selectedIssue.value) return
  const files = commentPending.value.slice()
  commentDraft.value = ''; commentPending.value = []
  await store.addCommentAttached(selectedIssue.value.id, b, files)
})
function startEditComment(c: any) { editCommentId.value = c.id; editCommentDraft.value = c.body }
const saveComment = () => wrap(async () => { if (editCommentDraft.value.trim()) await store.editComment(editCommentId.value, selectedIssue.value.id, editCommentDraft.value); editCommentId.value = '' })
const addSub = () => wrap(async () => { if (newSub.value.trim()) { await store.createSubIssue(selectedIssueId.value, newSub.value); newSub.value = '' } })

// --- Cycles (M4 ч.2) ---
const cycleStart = ref('')
const cycleEnd = ref('')
const addCycle = () => wrap(async () => { if (cycleStart.value && cycleEnd.value) { await store.createCycle(selectedTeamId.value, cycleStart.value, cycleEnd.value); cycleStart.value = ''; cycleEnd.value = '' } })
function filterBy(field: string, value: string) {
  store.setFieldPredicate(field, value ? { type: 'predicate', field, op: 'eq', value } : null)
}
// --- Calendar (M5) ---
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

const selectedCycleId = ref('')
const activeCycleOf = (tid: string) => store.teamCycles(tid).find((c: any) => c.status === 'active')
const upcomingCycleOf = (tid: string) => store.teamCycles(tid).find((c: any) => c.status === 'upcoming') || store.teamCycles(tid).find((c: any) => c.status !== 'active' && c.status !== 'completed')
function openCycle(id: string) { selectedCycleId.value = id; filterBy('cycle', id) }
function openCycleFor(tid: string, c: any) {
  if (!c) return
  selectedTeamId.value = tid; selectedDocId.value = ''; selectedIssueId.value = ''
  teamSection.value = ''; personalView.value = ''; activeSection.value = 'tasks'
  openCycle(c.id)
}
function closeCycle() {
  selectedCycleId.value = ''; store.setFieldPredicate('cycle', null); clearBreakdownFilters()
}

// burndown: live-расчёт из completedAt issues цикла (scope константа = текущее число).
const cycleBurndown = computed(() => {
  const c = store.cycles[selectedCycleId.value]
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
  const c = store.cycles[selectedCycleId.value]
  if (!c) return ''
  const f = (d: string) => new Date(d).toLocaleDateString('en', { month: 'short', day: 'numeric' })
  return `${f(c.startsOn)} → ${f(c.endsOn)}`
})
// edit cycle (name + dates) через UpdateCycle
const cycleEdit = ref(false)
const cycleForm = reactive({ name: '', startsOn: '', endsOn: '' })
const d10 = (s: string) => (s ? String(s).slice(0, 10) : '') // ISO → YYYY-MM-DD для input[type=date]
function startCycleEdit() {
  const c = store.cycles[selectedCycleId.value]; if (!c) return
  cycleForm.name = c.name || `Cycle ${c.number}`; cycleForm.startsOn = d10(c.startsOn); cycleForm.endsOn = d10(c.endsOn)
  cycleEdit.value = true
}
const saveCycle = () => wrap(async () => {
  await store.updateCycle(selectedCycleId.value, { name: cycleForm.name, startsOn: cycleForm.startsOn, endsOn: cycleForm.endsOn })
  cycleEdit.value = false
})
const cycleStatus = computed(() => {
  const c = store.cycles[selectedCycleId.value]; if (!c) return 'Cycle'
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
function clearBreakdownFilters() {
  for (const fld of ['assignee', 'priority', 'milestone', 'team']) store.setFieldPredicate(fld, null)
  store.labelFilter = []
}
const cycleBreakdown = computed(() => {
  const iss = store.issuesInCycle(selectedCycleId.value)
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

// --- Dashboards (M6 ч.2) ---
const selectedDashboardId = ref('')
const selectedDashboard = computed(() => store.dashboards[selectedDashboardId.value])
const newDashboard = ref('')
function openDashboard(id: string) { selectedDashboardId.value = id }
const createDash = () => wrap(async () => { if (newDashboard.value.trim()) { const id = await store.createDashboard(newDashboard.value); newDashboard.value = ''; openDashboard(id) } })
// add-widget form
const showWidget = ref(false)
const draggedWidget = ref('')
function statValue(w: any) {
  const p = w.config?.priority
  const filter = p === '' || p == null ? null : { type: 'predicate' as const, field: 'priority', op: 'eq' as const, value: Number(p) }
  return store.issueCount(filter as any)
}
// --- Custom fields (M6) ---
const CF_TYPES = ['text', 'number', 'date', 'checkbox', 'select', 'multiselect', 'url', 'people', 'relationship']
const cfName = ref('')
const cfType = ref('text')
const cfOptions = ref('')
const addField = () => wrap(async () => {
  if (!cfName.value.trim()) return
  const opts = cfOptions.value.split(',').map((s) => s.trim()).filter(Boolean)
  await store.createCustomField(selectedTeamId.value, cfName.value, cfType.value, opts)
  cfName.value = ''; cfOptions.value = ''
})
function cfMultiToggle(fieldId: string, opt: string) {
  const cur = (store.cfValue(selectedIssueId.value, fieldId) as string[]) || []
  store.setCustomFieldValue(selectedIssueId.value, fieldId, cur.includes(opt) ? cur.filter((x) => x !== opt) : [...cur, opt])
}

// --- FilterBar (M3) ---
const search = computed({
  get: () => (store.getFieldPredicate('title') as any)?.value ?? '',
  set: (v: string) =>
    store.setFieldPredicate('title', v ? { type: 'predicate', field: 'title', op: 'contains', value: v } : null),
})
function prioFilter(): number[] {
  return ((store.getFieldPredicate('priority') as any)?.value as number[]) ?? []
}
function togglePrio(p: number) {
  const cur = new Set(prioFilter())
  cur.has(p) ? cur.delete(p) : cur.add(p)
  const arr = [...cur]
  store.setFieldPredicate('priority', arr.length ? { type: 'predicate', field: 'priority', op: 'in', value: arr } : null)
}
const isPrio = (p: number) => prioFilter().includes(p)

async function saveView() {
  const name = prompt('View name')
  if (name) await wrap(() => store.createView(name, selectedTeamId.value))
}
function openView(id: string) {
  const v = store.applyView(id)
  if (v) selectedTeamId.value = v.teamId
  teamSection.value = ''; selectedDocId.value = ''; selectedCycleId.value = ''; personalView.value = ''
}
async function renameView(v: any) {
  const name = prompt('View name', v.name)
  if (name && name !== v.name) await wrap(() => store.renameView(v.id, name))
}
async function deleteView(v: any) {
  if (confirm(`Delete view "${v.name}"?`)) await wrap(() => store.deleteView(v.id))
}

async function addIssue(stateId: string) {
  const title = (newIssue[stateId] || '').trim()
  if (!title) return
  await wrap(() => store.createIssue(selectedTeamId.value, title, stateId))
  newIssue[stateId] = ''
}

// DnD: вставить перетащенную карточку перед targetId в колонке stateId.
function onDropOnCard(stateId: string, targetId: string) {
  if (!draggedId.value || draggedId.value === targetId) { draggedId.value = ''; return }
  const list = store.issuesFor(selectedTeamId.value, stateId).filter((x) => x.id !== draggedId.value)
  const idx = list.findIndex((x) => x.id === targetId)
  const before = idx > 0 ? list[idx - 1].id : ''
  wrap(() => store.reorderIssue(draggedId.value, stateId, before, targetId))
  draggedId.value = ''
}
// DnD: бросок в пустую часть колонки → в конец.
// Клавиатурный реордер борда (a11y, D19): Alt+↑/↓ — внутри колонки, Alt+←/→ — между колонками.
function boardKey(e: KeyboardEvent, stateId: string, id: string) {
  if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); openIssue(id); return }
  if (!e.altKey) return
  const states = store.teamStates(selectedTeamId.value)
  const colIdx = states.findIndex((s: any) => s.id === stateId)
  if (e.key === 'ArrowUp' || e.key === 'ArrowDown') {
    e.preventDefault()
    const list = store.issuesFor(selectedTeamId.value, stateId)
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
  const list = store.issuesFor(selectedTeamId.value, stateId).filter((x) => x.id !== draggedId.value)
  const before = list.length ? list[list.length - 1].id : ''
  wrap(() => store.reorderIssue(draggedId.value, stateId, before, ''))
  draggedId.value = ''
}

onMounted(async () => {
  if (!isAuthed()) return navigateTo('/login')
  await store.loadUser()
  await store.fetchMyWorkspaces()
  // активный ws валиден только если юзер всё ещё его участник (после seed/смены аккаунта
  // персистнутый id может указывать на удалённый/чужой ws → bootstrap бы упал permission_denied).
  const wsList = store.myWorkspaces || []
  const valid = !!store.activeWorkspaceId && wsList.some((w: any) => w.id === store.activeWorkspaceId)
  const target = valid ? store.activeWorkspaceId : wsList[0]?.id
  if (!valid) store.activeWorkspaceId = '' // сброс стейла, иначе bootstrap не очистит бакеты
  // гидратация подняла кэш → показываем сразу (без скелетона), освежаем в фоне.
  if (valid && Object.keys(store.teams).length) store.loaded = true
  if (target) {
    try {
      await store.bootstrap(target)
    } catch (e) {
      // недоступный ws → фолбэк на первый доступный
      store.activeWorkspaceId = ''
      const fallback = wsList.find((w: any) => w.id !== target)
      if (fallback) await store.bootstrap(fallback.id).catch(() => {})
    }
  }
  applyRoute(route.path) // deep-link при загрузке
})

const showCreate = ref(false)
const wsMenu = ref(false)
const wsInitial = computed(() => (store.activeWorkspace?.name || 'P')[0].toUpperCase())
const createWs = () => wrap(async () => {
  await store.createWorkspace(slug.value, name.value)
  slug.value = ''; name.value = ''; showCreate.value = false
})
function switchWorkspace(id: string) {
  wsMenu.value = false
  if (id && id !== store.activeWorkspaceId) { selectedTeamId.value = ''; store.bootstrap(id) }
}
function newWorkspace() { wsMenu.value = false; showCreate.value = true }
const addTeam = () => wrap(async () => { await store.createTeam(teamKey.value, teamName.value, teamParent.value); teamKey.value = ''; teamName.value = '' })
async function logout() {
  const base = useRuntimeConfig().public.apiBase as string
  try { await fetch(`${base}/auth/logout`, { method: 'POST', credentials: 'include' }) } catch {}
  clearAuth(); navigateTo('/login')
}

// --- Маршрутизация: путь ↔ состояние (шаренные ссылки) ---
const route = useRoute()
let routeSyncing = false
let urlSyncing = false

function applyRoute(path: string) {
  routeSyncing = true
  const seg = path.split('/').filter(Boolean)
  // сброс выбора
  selectedIssueId.value = ''; selectedDocId.value = ''; selectedCycleId.value = ''
  teamSection.value = ''; personalView.value = ''
  switch (seg[0]) {
    case 'inbox': activeSection.value = 'inbox'; break
    case 'chats': activeSection.value = 'chats'; break
    case 'chat': activeSection.value = 'chats'; if (seg[1]) openChannel(seg[1]); break
    case 'meetings': activeSection.value = 'meetings'; break
    case 'calendar': activeSection.value = 'calendar'; if (seg[1]) calAnchor.value = seg[1]; break
    case 'reports': activeSection.value = 'reports'; break
    case 'dashboard': activeSection.value = 'dashboard'; if (seg[1]) selectedDashboardId.value = seg[1]; break
    case 'my': activeSection.value = 'tasks'; personalView.value = (seg[1] as any) || 'assigned'; break
    case 'issue': { const i = store.issueByIdentifier(seg[1]); if (i) { activeSection.value = 'tasks'; selectedTeamId.value = i.teamId; openIssue(i.id) } break }
    case 'doc': { activeSection.value = 'tasks'; const d = store.documents[seg[1]]; if (d) selectedTeamId.value = d.teamId; openDoc(seg[1]); break }
    case 'cycle': { activeSection.value = 'tasks'; const c = store.cycles[seg[1]]; if (c) { selectedTeamId.value = c.teamId; openCycle(seg[1]) } break }
    case 'team': { const t = store.teamByKey(seg[1]); if (t) { activeSection.value = 'tasks'; selectedTeamId.value = t.id; if (seg[2]) teamSection.value = seg[2] as any } break }
    default: activeSection.value = 'tasks'; if (!seg.length) personalView.value = 'assigned' // дефолт-лендинг — My issues
  }
  nextTick(() => { routeSyncing = false })
}

function currentPath(): string {
  if (activeSection.value === 'inbox') return '/inbox'
  if (activeSection.value === 'chats') return store.activeChannelId ? `/chat/${store.activeChannelId}` : '/chats'
  if (activeSection.value === 'meetings') return '/meetings'
  if (activeSection.value === 'calendar') return `/calendar/${calAnchor.value}`
  if (activeSection.value === 'reports') return '/reports'
  if (activeSection.value === 'dashboard') return selectedDashboardId.value ? `/dashboard/${selectedDashboardId.value}` : '/dashboard'
  if (personalView.value) return `/my/${personalView.value}`
  if (selectedIssueId.value && store.issues[selectedIssueId.value]) return `/issue/${store.issues[selectedIssueId.value].identifier}`
  if (selectedDocId.value) return `/doc/${selectedDocId.value}`
  if (selectedCycleId.value) return `/cycle/${selectedCycleId.value}`
  if (teamSection.value && selectedTeam.value) return `/team/${selectedTeam.value.key}/${teamSection.value}`
  if (selectedTeam.value) return `/team/${selectedTeam.value.key}`
  return '/'
}

const linkCopied = ref(false)
function copyLink() {
  if (!import.meta.client) return
  navigator.clipboard?.writeText(location.origin + currentPath())
  linkCopied.value = true; setTimeout(() => (linkCopied.value = false), 1500)
}
function copyText(t: string) { if (import.meta.client) navigator.clipboard?.writeText(t) }
const duplicateIssue = (i: any) => wrap(async () => {
  const id = await store.createIssue(i.teamId, i.title + ' (copy)', i.stateId)
  if (i.priority) await store.setIssuePriority(id, i.priority)
  if (i.assigneeId) await store.setIssueAssignee(id, i.assigneeId)
  openIssue(id)
})
// completed-category state for "Mark complete"
function completedStateId(teamId: string) {
  return store.teamStates(teamId).find((s: any) => s.category === 'completed')?.id || ''
}
const issueActions = computed(() => {
  const i = selectedIssue.value
  if (!i) return []
  const doneId = completedStateId(i.teamId)
  return [
    [
      { label: 'Copy link', icon: 'i-heroicons-link', click: copyLink },
      { label: 'Copy ID', icon: 'i-heroicons-hashtag', click: () => copyText(i.identifier) },
    ],
    ...(store.canWrite ? [
      [
        ...(doneId && i.stateId !== doneId ? [{ label: 'Mark complete', icon: 'i-heroicons-check-circle', click: () => store.setIssueState(i.id, doneId) }] : []),
        { label: 'Duplicate', icon: 'i-heroicons-document-duplicate', click: () => duplicateIssue(i) },
      ],
      [
        { label: 'Archive', icon: 'i-heroicons-archive-box', click: () => { store.archiveIssue(i.id); selectedIssueId.value = '' } },
      ],
    ] : []),
  ]
})

// --- Team tree expand state ---
const expandTeam = reactive<Record<string, boolean>>({})
const expandCycles = reactive<Record<string, boolean>>({})
const expandDocs = reactive<Record<string, boolean>>({})
const expandViews = reactive<Record<string, boolean>>({})
const collapsedStatus = reactive<Record<string, boolean>>({}) // свёрнутые группы статусов в списке
const showFilters = ref(false)

// --- List virtualization (плоский поток: header-строки + issue-строки) ---
type ListRow = { type: 'header'; key: string; g: any } | { type: 'issue'; key: string; i: any; g: any }
const listRows = computed<ListRow[]>(() => {
  const rows: ListRow[] = []
  if (!selectedTeamId.value || store.layout !== 'list') return rows
  for (const g of store.issueGroups(selectedTeamId.value)) {
    rows.push({ type: 'header', key: 'h:' + g.key, g })
    if (!collapsedStatus[g.key]) for (const i of g.items) rows.push({ type: 'issue', key: i.id, i, g })
  }
  return rows
})
const mainEl = ref<HTMLElement | null>(null)
const listEl = ref<HTMLElement | null>(null)
const listScrollMargin = ref(0)
const measureScrollMargin = () => {
  if (!mainEl.value || !listEl.value) return
  listScrollMargin.value = listEl.value.getBoundingClientRect().top - mainEl.value.getBoundingClientRect().top + mainEl.value.scrollTop
}
const rowVirtualizer = useVirtualizer(computed(() => ({
  count: listRows.value.length,
  getScrollElement: () => mainEl.value,
  estimateSize: () => 42,
  overscan: 10,
  scrollMargin: listScrollMargin.value,
  getItemKey: (idx: number) => listRows.value[idx]?.key ?? idx,
})))
const measureRow = (el: Element | null) => { if (el) rowVirtualizer.value.measureElement(el) }
// пересчёт оффсета списка при изменениях, влияющих на высоту sticky-хедера/раскладку
watch([showFilters, () => store.layout, selectedTeamId], () => nextTick(measureScrollMargin))
onMounted(() => nextTick(measureScrollMargin))

// --- Глобальное создание через модалку (Linear-стиль) ---
// поля/сабмит живут в ModalsCreateModal; здесь только тип + open (нужны хоткею и сайдбару)
const createOpen = ref(false)
const createType = ref<'issue' | 'doc' | 'cycle' | 'team' | 'workspace'>('issue')
function openCreate(type: any = 'issue') {
  createType.value = type
  createOpen.value = true
}
const createMenu = [
  [
    { label: 'New issue', icon: 'i-heroicons-pencil-square', click: () => openCreate('issue') },
    { label: 'New doc', icon: 'i-heroicons-document-text', click: () => openCreate('doc') },
    { label: 'New cycle', icon: 'i-heroicons-arrow-path-rounded-square', click: () => openCreate('cycle') },
    { label: 'New milestone', icon: 'i-heroicons-flag', click: () => openCreate('milestone') },
    { label: 'New meeting', icon: 'i-heroicons-video-camera', click: () => openCreate('meeting') },
    { label: 'New dashboard', icon: 'i-heroicons-squares-2x2', click: () => openCreate('dashboard') },
    { label: 'New channel', icon: 'i-heroicons-hashtag', click: () => openCreate('channel') },
  ],
  [
    { label: 'New team', icon: 'i-heroicons-user-group', click: () => openCreate('team') },
    { label: 'New workspace', icon: 'i-heroicons-building-office-2', click: () => openCreate('workspace') },
  ],
]
// навигация после создания (сама модалка не знает про selectedTeamId/activeSection)
function onCreated(p: { type: string; id?: string; teamId?: string }) {
  if (p.type === 'doc') { selectedTeamId.value = p.teamId || ''; openDoc(p.id!) }
  else if (p.type === 'dashboard') { activeSection.value = 'dashboard'; selectedDashboardId.value = p.id! }
  else if (p.type === 'channel') { activeSection.value = 'chats'; store.loadMessages(p.id!) }
  else if (p.type === 'meeting') { activeSection.value = 'meetings' }
}
function selectTeamIssues(t: any) {
  selectedTeamId.value = t.id; selectedDocId.value = ''; selectedDashboardId.value = ''
  selectedCycleId.value = ''; store.setFieldPredicate('cycle', null); personalView.value = ''; teamSection.value = ''
  activeSection.value = 'tasks'
}

// персональные срезы (над списком команд)
const personalView = ref<'' | 'assigned' | 'created' | 'active'>('')
const PERSONAL = [
  { key: 'assigned', label: 'My issues', icon: 'i-heroicons-user-circle' },
  { key: 'active', label: 'Active', icon: 'i-heroicons-bolt' },
  { key: 'created', label: 'Created by me', icon: 'i-heroicons-pencil' },
]
function openPersonal(mode: any) {
  personalView.value = mode; selectedIssueId.value = ''; selectedDocId.value = ''; selectedCycleId.value = ''
  teamSection.value = ''; activeSection.value = 'tasks'
}

// My issues сгруппированы по дедлайну (effective date = scheduled ?? due).
function isoAddDays(base: string, n: number) {
  const t = new Date(base + 'T00:00:00'); t.setDate(t.getDate() + n); return t.toISOString().slice(0, 10)
}
const personalGroups = computed(() => {
  const iss = store.personalIssues(personalView.value)
  const today = d10(new Date().toISOString())
  const weekEnd = isoAddDays(today, 6)
  const buckets: Record<string, any[]> = { overdue: [], today: [], week: [], later: [], none: [] }
  for (const i of iss) {
    const d = d10(i.scheduledDate) || d10(i.dueDate)
    if (!d) buckets.none.push(i)
    else if (d < today) buckets.overdue.push(i)
    else if (d === today) buckets.today.push(i)
    else if (d <= weekEnd) buckets.week.push(i)
    else buckets.later.push(i)
  }
  const eff = (i: any) => d10(i.scheduledDate) || d10(i.dueDate) || '9999'
  for (const k of Object.keys(buckets)) buckets[k].sort((a, b) => eff(a).localeCompare(eff(b)) || (b.priority || 0) - (a.priority || 0))
  return [
    { key: 'overdue', label: 'Overdue', color: '#f85149', items: buckets.overdue },
    { key: 'today', label: 'Today', color: '#d29922', items: buckets.today },
    { key: 'week', label: 'This week', color: '#3fb950', items: buckets.week },
    { key: 'later', label: 'Later', color: '#7c828c', items: buckets.later },
    { key: 'none', label: 'No date', color: '#7c828c', items: buckets.none },
  ].filter((g) => g.items.length)
})

// --- Personal Calendar (top-level раздел) ---
const fmtD = (d: Date) => `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
const parseD = (ds: string) => new Date(ds + 'T00:00:00')
// ISO (UTC) → локальные дата/время/минуты. События хранятся в UTC, показываем в таймзоне юзера.
const localDate = (iso: string) => fmtD(new Date(iso))
const localHM = (iso: string) => { const d = new Date(iso); return `${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}` }
const localMin = (iso: string) => { const d = new Date(iso); return d.getHours() * 60 + d.getMinutes() }
const calView = ref<'month' | 'week' | 'day'>('month') // вид сетки
const calAnchor = ref('') // фокус-дата 'YYYY-MM-DD'
if (!calAnchor.value) calAnchor.value = fmtD(new Date())
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
// какие календари показывать (userId). По умолчанию — только мой; чужие включаешь тогглом.
const calUsers = ref<string[]>([])
watch(() => store.currentUserId, (id) => { if (id && !calUsers.value.length) calUsers.value = [id] }, { immediate: true })
function toggleCalUser(id: string) {
  calUsers.value = calUsers.value.includes(id) ? calUsers.value.filter((x) => x !== id) : [...calUsers.value, id]
}
const CAL_PALETTE = ['#5e6ad2', '#3fb950', '#d29922', '#f85149', '#a371f7', '#1f9cf0', '#db61a2', '#3bb2a8']
function colorForUser(id: string) { let h = 0; for (const c of id) h = (h * 31 + c.charCodeAt(0)) >>> 0; return CAL_PALETTE[h % CAL_PALETTE.length] }
// листание зависит от вида: месяц ±1мес, неделя ±7д, день ±1д
function shiftRange(n: number) {
  const d = parseD(calAnchor.value)
  if (calView.value === 'month') d.setMonth(d.getMonth() + n)
  else d.setDate(d.getDate() + n * (calView.value === 'week' ? 7 : 1))
  calAnchor.value = fmtD(d)
}
function calToday() { calAnchor.value = fmtD(new Date()) }
const calTodayStr = computed(() => fmtD(new Date()))
const calHeaderLabel = computed(() => {
  const d = parseD(calAnchor.value)
  if (calView.value === 'month') return d.toLocaleString('en', { month: 'long', year: 'numeric' })
  if (calView.value === 'day') return d.toLocaleString('en', { weekday: 'short', month: 'short', day: 'numeric', year: 'numeric' })
  const w = weekDays.value
  return `${parseD(w[0]).toLocaleString('en', { month: 'short', day: 'numeric' })} – ${parseD(w[6]).toLocaleString('en', { month: 'short', day: 'numeric' })}`
})
// дни недели якоря (Mon..Sun)
const weekDays = computed(() => {
  const d = parseD(calAnchor.value); d.setDate(d.getDate() - ((d.getDay() + 6) % 7))
  return Array.from({ length: 7 }, (_, i) => { const x = new Date(d); x.setDate(d.getDate() + i); return fmtD(x) })
})
// дни активного диапазона тайм-сетки: week→7, day→1
const gridDays = computed(() => (calView.value === 'day' ? [calAnchor.value] : weekDays.value))
// 6×7 сетка месяца
const calGrid = computed(() => {
  const a = parseD(calAnchor.value); const m = a.getMonth()
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
    if (!calUsers.value.includes(e.userId)) continue
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
    if (i.assigneeId && !calUsers.value.includes(i.assigneeId)) continue
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
// минуты → "1h 15m" / "45m"
const fmtMin = (m: number) => m < 60 ? `${m}m` : `${Math.floor(m / 60)}h ${String(m % 60).padStart(2, '0')}m`
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
    Object.assign(calForm, { id: '', title: '', description: '', kind: 'event', participants: [], date: ds, start: min2hm(a), end: min2hm(Math.min(1440, a + dur)), allDay: false, recurrence: 'none', recUntil: '' })
    showCalEvent.value = true
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
  if (it.kind === 'event' || it.kind === 'dayoff') openCalEdit(it.ev)
  else if (it.kind === 'issue') openIssue(it.id)
  else activeSection.value = 'meetings'
}
// модалка создания/правки события
const showCalEvent = ref(false)
const calForm = reactive({ id: '', title: '', description: '', kind: 'event', participants: [] as string[], date: '', start: '09:00', end: '10:00', allDay: false, recurrence: 'none', recUntil: '' })
function openCalCreate(ds: string) {
  Object.assign(calForm, { id: '', title: '', description: '', kind: 'event', participants: [], date: ds, start: '09:00', end: '10:00', allDay: false, recurrence: 'none', recUntil: '' })
  showCalEvent.value = true
}
function openCalEdit(ev: any) {
  Object.assign(calForm, {
    id: ev.id, title: ev.title, description: ev.description || '', kind: ev.kind || 'event', participants: [],
    date: localDate(ev.startsAt), start: localHM(ev.startsAt), end: localHM(ev.endsAt), allDay: !!ev.allDay,
    recurrence: ev.recurrence || 'none', recUntil: ev.recurrenceUntil ? localDate(ev.recurrenceUntil) : '',
  })
  showCalEvent.value = true
}
const calIsoBounds = () => {
  // локальные дата+время → ISO. allDay → весь день.
  const mkLocal = (t: string) => new Date(`${calForm.date}T${t}:00`).toISOString()
  if (calForm.allDay) return { startsAt: new Date(`${calForm.date}T00:00:00`).toISOString(), endsAt: new Date(`${calForm.date}T23:59:00`).toISOString() }
  return { startsAt: mkLocal(calForm.start), endsAt: mkLocal(calForm.end) }
}
const saveCalEvent = () => wrap(async () => {
  if (!calForm.title.trim() || !calForm.date) return
  const { startsAt, endsAt } = calIsoBounds()
  // kind=meeting → создаём scheduled-митинг (с участниками + концом), не персональное событие
  if (!calForm.id && calForm.kind === 'meeting') {
    await store.createMeeting(calForm.title, startsAt, endsAt, [...calForm.participants])
    showCalEvent.value = false
    return
  }
  const recUntil = calForm.recurrence !== 'none' && calForm.recUntil ? new Date(`${calForm.recUntil}T23:59:00`).toISOString() : ''
  const p = { title: calForm.title, description: calForm.description, kind: calForm.kind, startsAt, endsAt, allDay: calForm.allDay, recurrence: calForm.recurrence, recurrenceUntil: recUntil }
  if (calForm.id) await store.updateCalendarEvent(calForm.id, p)
  else await store.createCalendarEvent(p)
  showCalEvent.value = false
})
const deleteCalEvent = () => wrap(async () => { if (calForm.id) { await store.deleteCalendarEvent(calForm.id); showCalEvent.value = false } })

// Automations + issue templates живут в ModalsSettingsModal

// --- Ручная запись времени (задним числом) ---
const showTimeEntry = ref(false)
const teForm = reactive({ issueId: '', date: '', start: '09:00', end: '10:00', description: '' })
function openTimeEntry(issueId: string) {
  Object.assign(teForm, { issueId, date: fmtD(new Date()), start: '09:00', end: '10:00', description: '' })
  showTimeEntry.value = true
}
const saveTimeEntry = () => wrap(async () => {
  if (!teForm.date) return
  const s = new Date(`${teForm.date}T${teForm.start}:00`).toISOString()
  const e = new Date(`${teForm.date}T${teForm.end}:00`).toISOString()
  if (new Date(e) <= new Date(s)) { err.value = 'End must be after start'; return }
  await store.addTimeEntry(teForm.issueId, teForm.description, s, e)
  showTimeEntry.value = false
})

// --- Time reports ---
const repGroup = ref<'user' | 'issue' | 'day'>('user')
const repUser = ref('') // фильтр по участнику (userId или '')
const repPreset = ref('month')
const repFrom = ref(''); const repTo = ref('')
function setRange(preset: string) {
  repPreset.value = preset
  const n = new Date(); const to = fmtD(n)
  if (preset === 'today') { repFrom.value = to; repTo.value = to }
  else if (preset === 'week') { const d = new Date(n); d.setDate(n.getDate() - ((n.getDay() + 6) % 7)); repFrom.value = fmtD(d); repTo.value = to }
  else if (preset === 'month') { repFrom.value = fmtD(new Date(n.getFullYear(), n.getMonth(), 1)); repTo.value = to }
  else if (preset === 'last7') { const d = new Date(n); d.setDate(n.getDate() - 6); repFrom.value = fmtD(d); repTo.value = to }
  else if (preset === 'last30') { const d = new Date(n); d.setDate(n.getDate() - 29); repFrom.value = fmtD(d); repTo.value = to }
  // custom — оставляем как есть
}
if (!repFrom.value) setRange('month')
const repRows = ref<any[]>([]); const repTotal = ref(0); const repDaily = ref<any[]>([]); const repLoading = ref(false)
async function loadReport() {
  if (!store.activeWorkspaceId) return
  repLoading.value = true
  try {
    const fromIso = new Date(`${repFrom.value}T00:00:00`).toISOString()
    const toD = new Date(`${repTo.value}T00:00:00`); toD.setDate(toD.getDate() + 1) // конец дня включительно
    const toIso = toD.toISOString()
    const uid = repUser.value || undefined
    const grouped = await store.timeReport({ from: fromIso, to: toIso, groupBy: repGroup.value, userId: uid })
    repRows.value = grouped.rows; repTotal.value = grouped.total
    const daily = repGroup.value === 'day' ? grouped : await store.timeReport({ from: fromIso, to: toIso, groupBy: 'day', userId: uid })
    repDaily.value = [...daily.rows].sort((a, b) => String(a.key).localeCompare(String(b.key)))
  } catch (e: any) { err.value = e?.message || String(e) } finally { repLoading.value = false }
}
function repLabel(key: string) {
  if (repGroup.value === 'user') return store.userName(key) || key
  if (repGroup.value === 'issue') return key ? (store.issues[key] ? `${store.issues[key].identifier} ${store.issues[key].title}` : '(unknown)') : 'No task'
  return key
}
const repMax = computed(() => Math.max(1, ...repRows.value.map((r) => r.minutes)))
const repDailyMax = computed(() => Math.max(1, ...repDaily.value.map((r) => r.minutes)))
const repEntries = computed(() => repRows.value.reduce((a, r) => a + r.count, 0))
const repActiveDays = computed(() => repDaily.value.length)
const repAvgPerDay = computed(() => repActiveDays.value ? Math.round(repTotal.value / repActiveDays.value) : 0)
function exportReportCsv() {
  const rows = [['Group', 'Minutes', 'Time', 'Entries'], ...repRows.value.map((r) => [repLabel(r.key), r.minutes, fmtMin(r.minutes), r.count])]
  const csv = rows.map((row) => row.map((c) => `"${String(c).replace(/"/g, '""')}"`).join(',')).join('\n')
  const blob = new Blob([csv], { type: 'text/csv' }); const url = URL.createObjectURL(blob)
  const a = document.createElement('a'); a.href = url; a.download = `time-report-${repFrom.value}_${repTo.value}.csv`; a.click(); URL.revokeObjectURL(url)
}
const dayShort = (ds: string) => parseD(ds).toLocaleDateString('en', { month: 'short', day: 'numeric' })
watch([() => activeSection.value, repFrom, repTo, repGroup, repUser], () => { if (activeSection.value === 'reports') loadReport() }, { immediate: false })
// AI-инсайты (Фаза 4): темы + аномалии + настроение
const aiThemes = ref<any[]>([]); const aiOutliers = ref<any[]>([]); const aiMood = ref<any>(null)
const aiLoaded = ref(false)
watch(() => activeSection.value, (s) => {
  if (s === 'reports' && !aiLoaded.value) {
    aiLoaded.value = true
    store.themes().then((t) => { aiThemes.value = t })
    store.anomalies().then((o) => { aiOutliers.value = o })
    store.mood().then((m) => { aiMood.value = m })
  }
})
const moodPct = computed(() => aiMood.value?.n ? Math.round((aiMood.value.avg + 1) / 2 * 100) : 0)
// iCal-подписка
const showIcal = ref(false)
const icalUrl = ref('')
const openIcal = () => wrap(async () => { icalUrl.value = await store.icalFeedUrl(false); showIcal.value = true })

// страницы-разделы команды (cycles / docs / views)
const teamSection = ref<'' | 'cycles' | 'docs' | 'views'>('')
function openTeamSection(tid: string, section: any) {
  selectedTeamId.value = tid; teamSection.value = section
  personalView.value = ''; selectedDocId.value = ''; selectedCycleId.value = ''; selectedIssueId.value = ''; selectedMsId.value = ''
  store.setFieldPredicate('cycle', null); activeSection.value = 'tasks'
}

// --- Docs (M7 ч.3) ---
const selectedDocId = ref('')
const selectedDoc = computed(() => store.documents[selectedDocId.value])
const newDoc = ref('')
function openDoc(id: string) { selectedDocId.value = id; selectedIssueId.value = ''; teamSection.value = '' }
const addDoc = () => wrap(async () => { if (newDoc.value.trim()) { const id = await store.createDoc(selectedTeamId.value, newDoc.value); newDoc.value = ''; openDoc(id) } })
// rich doc editor (md тулбар + live preview)
const docMode = ref<'edit' | 'split' | 'preview'>('split')
const docEl = ref<HTMLTextAreaElement | null>(null)
const docDraft = ref('')
watch(selectedDocId, (id) => { docDraft.value = id ? (store.documents[id]?.body?.md || '') : '' }, { immediate: true })
let docSaveTimer: any = null
function onDocInput() {
  if (!selectedDocId.value) return
  clearTimeout(docSaveTimer)
  docSaveTimer = setTimeout(() => store.updateDoc(selectedDocId.value, { md: docDraft.value }), 600)
}
function docWrap(pre: string, post = pre) {
  const el = docEl.value; if (!el) { docDraft.value += pre + post; onDocInput(); return }
  const s = el.selectionStart, e = el.selectionEnd, v = docDraft.value
  docDraft.value = v.slice(0, s) + pre + v.slice(s, e) + post + v.slice(e)
  onDocInput(); nextTick(() => { el.focus(); el.selectionStart = s + pre.length; el.selectionEnd = e + pre.length })
}
const addChildDoc = (teamId: string, parentId: string) => wrap(async () => {
  selectedTeamId.value = teamId; expandDocs[teamId] = true
  const id = await store.createDoc(teamId, 'Untitled', parentId)
  openDoc(id)
})

// --- Notifications (M7 ч.2) ---
const notifOpen = ref(false)
function openNotif(n: any) {
  notifOpen.value = false
  store.markNotificationRead(n.id)
  if (n.entityType === 'channel') { openChannel(n.entityId); return }
  const iss = store.issues[n.entityId]
  if (iss) { activeSection.value = 'tasks'; personalView.value = ''; teamSection.value = ''; selectedTeamId.value = iss.teamId; openIssue(iss.id) }
}
// notifMeta/inboxList живут в SectionsInboxSection
// Inbox-фильтры (сайдбар)
const inboxFilter = ref<'all' | 'unread' | 'assigned' | 'state_changed'>('all')
const INBOX_FILTERS = [
  { key: 'all', label: 'All', icon: 'i-heroicons-inbox' },
  { key: 'unread', label: 'Unread', icon: 'i-heroicons-envelope' },
  { key: 'assigned', label: 'Assigned to me', icon: 'i-heroicons-user-plus' },
  { key: 'state_changed', label: 'Status changes', icon: 'i-heroicons-arrow-path' },
  { key: 'commented', label: 'Comments', icon: 'i-heroicons-chat-bubble-left-ellipsis' },
]
function inboxCount(key: string) {
  if (key === 'all') return store.myNotifications.length
  if (key === 'unread') return store.unreadCount
  return store.myNotifications.filter((n: any) => n.type === key).length
}

// --- Meetings ---
const meeting = useMeeting()
// ресайз правых панелей (таска/цикл; тред — в SectionsThreadPanel)
const detailPanel = useResizable('detail', 384)
const cyclePanel = useResizable('cycle', 320)
const joinScheduled = (mid: string) => wrap(() => meeting.join(mid))

// --- Chats ---
// композер/шапка/скролл/хадлы живут в SectionsChatSection, тред — в SectionsThreadPanel.
// Здесь — открытие каналов (зовётся из сайдбара/палитры/роутера/уведомлений) + unread-разделитель.
const currentChannel = computed(() => store.channels[store.activeChannelId])
const chatSection = ref<{ scrollBottom: () => void; scrollToUnread: () => void } | null>(null)
// inline @/# пикеры (юзеры + сущности)
const descMention = useMention(descDraft, descEl)
const docMention = useMention(docDraft, docEl)
const commentMention = useMention(commentDraft, commentEl)
function onCommentKeydown(e: KeyboardEvent) {
  if (commentMention.onKeydown(e)) return
  if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); addComment() }
}
// вложения в описании/доках — chat-модель: paste/пик → attachment (тумба/плашка), не inline md
const descFileEl = ref<HTMLInputElement | null>(null)
const docFileEl = ref<HTMLInputElement | null>(null)
const attachFiles = (entityType: string, entityId: string, files?: FileList | File[] | null) => wrap(async () => {
  if (!files || !entityId) return
  for (const f of Array.from(files)) await store.addAttachment(entityType, entityId, f)
})
function onAttachPaste(e: ClipboardEvent, entityType: string, entityId: string) {
  const items = e.clipboardData?.items; if (!items) return
  const fs: File[] = []
  for (const it of Array.from(items)) if (it.kind === 'file') { const f = it.getAsFile(); if (f) fs.push(f) }
  if (fs.length) { e.preventDefault(); attachFiles(entityType, entityId, fs) }
}
const onDescFilePick = (e: Event) => { attachFiles('issue', selectedIssueId.value, (e.target as HTMLInputElement).files); (e.target as HTMLInputElement).value = '' }
// доки: вставка прямо в тело (markdown), рендер как обычно
function docInsertAtCaret(text: string) {
  const el = docEl.value
  const pos = el ? (el.selectionStart ?? docDraft.value.length) : docDraft.value.length
  docDraft.value = docDraft.value.slice(0, pos) + text + docDraft.value.slice(pos)
  onDocInput()
  nextTick(() => { if (el) { el.focus(); const c = pos + text.length; el.selectionStart = el.selectionEnd = c } })
}
const docInsertFile = (file: File) => wrap(async () => {
  const up = await store.uploadAndGetKey(file)
  const url = fileViewUrl(up.key)
  docInsertAtCaret(file.type.startsWith('image/') ? `\n![${file.name || 'image'}](${url})\n` : `[📎 ${file.name}](${url})`)
})
function onDocPaste(e: ClipboardEvent) {
  const items = e.clipboardData?.items; if (!items) return
  const fs: File[] = []
  for (const it of Array.from(items)) if (it.kind === 'file') { const f = it.getAsFile(); if (f) fs.push(f) }
  if (fs.length) { e.preventDefault(); fs.forEach(docInsertFile) }
}
const onDocFilePick = (e: Event) => { const files = (e.target as HTMLInputElement).files; if (files) Array.from(files).forEach(docInsertFile); (e.target as HTMLInputElement).value = '' }
function onDescKeydown(e: KeyboardEvent) {
  if (descMention.onKeydown(e)) return
  if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') saveDesc()
}
function onDocKeydown(e: KeyboardEvent) {
  if (docMention.onKeydown(e)) return
}
const lightbox = useLightbox()
// клик по картинке/#-чипу в отрендеренном markdown
function onRefClick(e: MouseEvent) {
  const t = e.target as HTMLElement
  if (t.tagName === 'IMG' && t.classList.contains('md-img')) { lightbox.open((t as HTMLImageElement).src, (t as HTMLImageElement).alt); return }
  const el = t.closest?.('.ref-chip') as HTMLElement | null
  if (!el) return
  const iss = el.dataset.issue
  if (iss) { const i = store.issueByIdentifier(iss); if (i) { activeSection.value = 'tasks'; selectedTeamId.value = i.teamId; openIssue(i.id) } return }
  const type = el.dataset.type, id = el.dataset.id || ''
  if (type === 'doc') { const d = store.documents[id]; if (d) { activeSection.value = 'tasks'; selectedTeamId.value = d.teamId; openDoc(id) } }
  else if (type === 'channel') { openChannel(id) }
  else if (type === 'meeting') { activeSection.value = 'meetings' }
}
// разделитель «new messages»: метка времени, до которой прочитано (зафиксирована при открытии)
const unreadDividerTs = ref('')
const firstUnreadMsgId = computed(() => {
  if (!unreadDividerTs.value || !currentChannel.value) return ''
  const m = store.channelMessages(currentChannel.value.id).find((x: any) => x.authorId !== store.currentUserId && String(x.createdAt) > unreadDividerTs.value)
  return m?.id || ''
})
async function openChannel(id: string) {
  activeSection.value = 'chats'; store.closeThread()
  unreadDividerTs.value = store.channelUnreadCount(id) > 0 ? (store.channelLastReadAt(id) || '1970-01-01') : ''
  await store.loadMessages(id)
  await nextTick()
  if (firstUnreadMsgId.value) chatSection.value?.scrollToUnread()
  else chatSection.value?.scrollBottom()
  store.markChannelRead(id)
}
const openDM = (userId: string) => wrap(async () => { activeSection.value = 'chats'; store.closeThread(); await store.createDM(userId); chatSection.value?.scrollBottom() })
// reactions (комменты задачи; чат-сообщения рендерит ChatMessage)
const EMOJIS = ['👍', '❤️', '😄', '🎉', '👀', '🚀', '✅', '🔥']
function reactionGroups(m: any) {
  const g: Record<string, { emoji: string; count: number; mine: boolean }> = {}
  for (const r of m.reactions || []) {
    ;(g[r.emoji] ||= { emoji: r.emoji, count: 0, mine: false }).count++
    if (r.userId === store.currentUserId) g[r.emoji].mine = true
  }
  return Object.values(g)
}
const cReactPickerId = ref('')
const commentReact = (c: any, emoji: string) => { cReactPickerId.value = ''; if (selectedIssue.value) wrap(() => store.toggleCommentReaction(c.id, selectedIssue.value!.id, emoji)) }
// members
const showMembers = ref(false)
// invite-стейт — синглтон-композабл: им пользуются MembersModal (через страницу) и SettingsModal
const { inviteEmail, inviteRole, inviteLink, doInvite, copyInviteLink } = useInvites()


// --- Settings-модалка (workspace) ---
// содержимое (табы, автоматизации, шаблоны, токены, вебхуки…) живёт в ModalsSettingsModal;
// здесь только open/tab (открывается из рейла и wsMenu) и то, что нужно остальной странице
const showSettings = ref(false)
const settingsTab = ref('general')
// appearance — синглтон-композабл (страница применяет сохранённое в onMounted)
const { applyDensity } = useDensity()
// notifications prefs (клиентский фильтр типов в Inbox — используется в inboxList)
const notifPrefs = ref<Record<string, boolean>>({ assigned: true, state_changed: true, commented: true, mention: true })
function toggleNotifPref(k: string) {
  notifPrefs.value[k] = !notifPrefs.value[k]
  if (import.meta.client) localStorage.setItem('pms-notif-prefs', JSON.stringify(notifPrefs.value))
}
function openSettings() {
  openRailCfg() // подготовить cfgList для вкладки Sidebar
  settingsTab.value = 'general'; showSettings.value = true
}
function confirmDeleteTeam(t: any) {
  if (!import.meta.client || !confirm(`Delete team "${t.name}"? Sub-teams included. Must have no active issues.`)) return
  wrap(async () => { await store.deleteTeam(t.id); if (selectedTeamId.value === t.id) { selectedTeamId.value = ''; teamSection.value = '' } })
}


// --- Command palette / поиск (M7) ---
const paletteOpen = ref(false)
const paletteQ = ref('')
const paletteIdx = ref(0)
const paletteInput = ref<HTMLInputElement | null>(null)
// Серверные FTS-результаты (тело задач/доков, опечатки, сообщения) — дебаунс.
const srvResults = ref<any[]>([])
let srvSearchTimer: any = null
watch(paletteQ, (q) => {
  clearTimeout(srvSearchTimer)
  if (!q.trim()) { srvResults.value = []; return }
  srvSearchTimer = setTimeout(async () => {
    try { srvResults.value = await store.serverSearch(q) } catch { srvResults.value = [] }
  }, 200)
})
// Клиентский searchAll — мгновенно по заголовкам/локальному стору; серверный добавляет
// совпадения по телу/опечаткам. Дедуп по type:id (клиент первым — у него стабильный teamId).
const paletteResults = computed(() => {
  const seen = new Set<string>()
  const out: any[] = []
  for (const r of [...store.searchAll(paletteQ.value), ...srvResults.value]) {
    const k = `${r.type}:${r.id}`
    if (seen.has(k)) continue
    seen.add(k); out.push(r)
  }
  return out
})
function openPalette() {
  paletteOpen.value = true; paletteQ.value = ''; paletteIdx.value = 0; srvResults.value = []
  nextTick(() => paletteInput.value?.focus())
}
function paletteMove(d: number) {
  const n = paletteResults.value.length
  if (n) paletteIdx.value = (paletteIdx.value + d + n) % n
}
function runResult(r: any) {
  paletteOpen.value = false
  if (!r) return
  if (r.type === 'issue') { activeSection.value = 'tasks'; selectedTeamId.value = r.teamId; openIssue(r.id) }
  else if (r.type === 'team') { activeSection.value = 'tasks'; selectTeamIssues({ id: r.id }) }
  else if (r.type === 'dashboard') { activeSection.value = 'dashboard'; selectedDashboardId.value = r.id }
  else if (r.type === 'view') { activeSection.value = 'tasks'; openView(r.id) }
  else if (r.type === 'doc') { selectedTeamId.value = r.teamId; openDoc(r.id) }
  else if (r.type === 'channel' || r.type === 'message') { openChannel(r.id) }
}
function onGlobalKeydown(e: KeyboardEvent) {
  if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') { e.preventDefault(); openPalette() }
  else if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'n') { e.preventDefault(); openCreate('issue') }
  else if (e.key === 'Escape') { paletteOpen.value = false; createOpen.value = false }
}
onBeforeUnmount(() => {
  window.removeEventListener('keydown', onGlobalKeydown)
  window.removeEventListener('popstate', onPop)
})
onMounted(() => {
  window.addEventListener('keydown', onGlobalKeydown)
  window.addEventListener('popstate', onPop)
  // appearance + notif prefs
  const d = localStorage.getItem('pms-density'); if (d) applyDensity(d)
  try { const np = JSON.parse(localStorage.getItem('pms-notif-prefs') || 'null'); if (np) notifPrefs.value = np } catch {}
  // персист развёрнутости дерева
  try {
    const s = JSON.parse(localStorage.getItem('pms-expand') || '{}')
    Object.assign(expandTeam, s.team || {}); Object.assign(expandCycles, s.cycles || {}); Object.assign(expandDocs, s.docs || {}); Object.assign(expandViews, s.views || {})
  } catch {}
  watch([expandTeam, expandCycles, expandDocs, expandViews], () => {
    localStorage.setItem('pms-expand', JSON.stringify({ team: expandTeam, cycles: expandCycles, docs: expandDocs, views: expandViews }))
  }, { deep: true })
})

// глубина отступа дерева по числу сегментов ltree-path
const depth = (p: string) => Math.max(0, String(p || '').split('.').length - 1)

// роут-синк (в конце — после объявления всех ref, иначе TDZ)
// состояние → URL (history.replace, без перерендера и без эха в applyRoute)
watch(() => [activeSection.value, selectedTeamId.value, selectedIssueId.value, selectedDocId.value, selectedCycleId.value, teamSection.value, personalView.value, store.activeChannelId, selectedDashboardId.value, calAnchor.value],
  () => { if (routeSyncing) return; const p = currentPath(); if (import.meta.client && p !== location.pathname) history.replaceState(history.state, '', p) })
// URL → состояние только на back/forward (popstate) и при старте; наш replaceState popstate не вызывает
function onPop() { if (store.activeWorkspaceId) applyRoute(location.pathname) }

// --- Тайтл вкладки (динамический по секции/выбранной сущности; глобальный titleTemplate добавит " · stdtask") ---
const SECTION_TITLES: Record<string, string> = { dashboard: 'Dashboards', inbox: 'Inbox', chats: 'Chat', meetings: 'Meetings', calendar: 'Calendar', reports: 'Reports', tasks: 'Tasks' }
const PERSONAL_TITLES: Record<string, string> = { assigned: 'My issues', created: 'Created by me', active: 'Active' }
const pageTitle = computed(() => {
  if (selectedIssue.value) return `${selectedIssue.value.identifier} ${selectedIssue.value.title}`
  if (selectedDoc.value) return selectedDoc.value.title || 'Untitled doc'
  if (selectedMs.value) return selectedMs.value.name
  const sec = activeSection.value
  if (sec === 'chats') return currentChannel.value ? `#${currentChannel.value.name}` : 'Chat'
  if (sec === 'dashboard') return selectedDashboard.value?.name || 'Dashboards'
  if (sec === 'tasks') {
    if (personalView.value) return PERSONAL_TITLES[personalView.value] || 'Tasks'
    const cy = selectedCycleId.value && store.cycles[selectedCycleId.value]
    if (cy) return `Cycle ${cy.number}${selectedTeam.value ? ' · ' + selectedTeam.value.name : ''}`
    return selectedTeam.value?.name || 'Tasks'
  }
  return SECTION_TITLES[sec] || 'stdtask'
})
useHead({ title: () => pageTitle.value })
</script>

<template>
  <div class="flex h-screen flex-col">
    <!-- ⓪ Top bar -->
    <header class="flex h-14 shrink-0 items-center gap-3 bg-base px-3 text-sm">
      <!-- workspace switcher -->
      <div class="relative">
        <UButton color="gray" variant="ghost" trailing-icon="i-heroicons-chevron-down" @click="wsMenu = !wsMenu">
          <Avatar :name="store.activeWorkspace?.name" :avatar-key="(store.activeWorkspace as any)?.logoKey" :size="20" />
          <span class="font-medium"><span class="text-ok">~/</span>{{ store.activeWorkspace?.name || 'workspace' }}</span>
        </UButton>
        <div v-if="wsMenu" class="absolute left-0 top-10 z-50 w-56 rounded-md border border-subtle bg-elevated p-1 shadow-lg">
          <div class="px-2 py-1 text-xs uppercase text-tertiary"><span class="text-ok"># </span>Workspaces</div>
          <button v-for="w in store.myWorkspaces" :key="w.id"
            class="flex w-full items-center gap-2 rounded-sm px-2 py-1.5 text-left hover:bg-hover"
            :class="{ 'text-accent': w.id === store.activeWorkspaceId }" @click="switchWorkspace(w.id)">
            <span class="flex h-5 w-5 items-center justify-center rounded bg-surface text-xs">{{ (w.name||'?')[0].toUpperCase() }}</span>{{ w.name }}
          </button>
          <div class="my-1 border-t border-subtle" />
          <button class="w-full rounded-sm px-2 py-1.5 text-left text-tertiary hover:bg-hover" @click="wsMenu = false; openSettings(); settingsTab = 'members'">Manage members</button>
          <button v-if="store.canWrite" class="w-full rounded-sm px-2 py-1.5 text-left text-tertiary hover:bg-hover" @click="openCreate('workspace')">＋ New workspace</button>
        </div>
      </div>

      <!-- глобальное создание -->
      <UDropdown :items="createMenu" :popper="{ placement: 'bottom-start' }">
        <UButton icon="i-heroicons-plus" color="gray" variant="ghost" square title="Create… (⌘N)" />
      </UDropdown>

      <!-- search -->
      <UButton color="gray" variant="ghost" class="mx-auto w-96 ring-1 ring-subtle" @click="openPalette">
        <span class="flex-1 text-left text-tertiary"><span class="text-ok">$</span> search…</span>
        <template #trailing><UKbd>⌘K</UKbd></template>
      </UButton>

      <!-- notifications -->
      <div class="relative">
        <UButton icon="i-heroicons-bell" color="gray" variant="ghost" square @click="notifOpen = !notifOpen" />
        <UBadge v-if="store.unreadCount" color="primary" size="xs" class="pointer-events-none absolute -right-1 -top-1 rounded">{{ store.unreadCount }}</UBadge>
        <div v-if="notifOpen" class="absolute right-0 top-10 z-50 w-80 rounded-md border border-subtle bg-elevated shadow-lg">
          <div class="flex items-center justify-between border-b border-subtle px-3 py-2 text-xs uppercase text-tertiary">
            <span><span class="text-ok"># </span>Notifications</span>
            <UButton v-if="store.unreadCount" size="2xs" color="gray" variant="ghost" @click="store.markAllNotificationsRead()">[ mark all read ]</UButton>
          </div>
          <div class="max-h-96 overflow-y-auto">
            <div v-for="n in store.myNotifications.slice(0, 15)" :key="n.id"
              class="cursor-pointer border-b border-subtle px-3 py-2 text-xs hover:bg-hover"
              :class="{ 'opacity-50': n.readAt }" @click="openNotif(n)">
              <span class="text-tertiary">{{ n.type }}</span> · {{ store.issueById(n.entityId)?.identifier || n.entityType }}
              <span v-if="!n.readAt" class="ml-1 inline-block h-1.5 w-1.5 rounded-full bg-accent align-middle" />
            </div>
            <div v-if="!store.myNotifications.length" class="px-3 py-3 text-xs text-tertiary">// no notifications</div>
          </div>
        </div>
      </div>
    </header>

    <div class="flex flex-1 overflow-hidden bg-base" @click="onRefClick">
    <!-- ② Left rail -->
    <aside class="flex w-16 flex-col items-center gap-2 bg-base py-3">
      <div v-for="s in railSections" :key="s.key" class="relative">
        <UButton :icon="s.icon" :title="s.label" square
          :color="activeSection === s.key ? 'primary' : 'gray'"
          variant="ghost"
          @click="switchSection(s.key as any)" />
        <span v-if="s.key === 'chats' && store.totalUnread" class="absolute -right-0.5 -top-0.5 flex min-w-[16px] items-center justify-center rounded bg-accent px-1 text-[10px] font-semibold text-white">{{ store.totalUnread }}</span>
      </div>

      <!-- настройки воркспейса -->
      <UButton class="mt-auto" icon="i-heroicons-cog-6-tooth" color="gray" variant="ghost" square title="Workspace settings" @click="openSettings" />
      <UButton icon="i-heroicons-arrow-right-on-rectangle" color="gray" variant="ghost" square title="Logout" @click="logout" />
    </aside>

    <!-- ③ Contextual sidebar -->
    <aside class="w-60 overflow-y-auto bg-base px-2 py-3 text-sm">
      <!-- скелетон сайдбара -->
      <div v-if="!store.loaded" class="space-y-2 px-1 pt-1">
        <div class="skeleton h-3 w-20" />
        <div v-for="n in 6" :key="n" class="skeleton h-7" :style="{ width: 55 + (n * 7) % 40 + '%' }" />
      </div>

      <!-- create workspace (по кнопке ＋ или когда воркспейсов нет) -->
      <div v-if="showCreate || (store.loaded && !store.activeWorkspaceId && !store.myWorkspaces.length)"
        class="mb-3 space-y-1.5 rounded-md border border-subtle p-2">
        <div class="text-xs text-tertiary">New workspace</div>
        <UInput v-model="slug" placeholder="slug" size="xs" />
        <UInput v-model="name" placeholder="name" size="xs" />
        <UButton block size="xs" @click="createWs">[ create ]</UButton>
      </div>

      <!-- DASHBOARD section: список дашбордов -->
      <template v-if="store.activeWorkspaceId && activeSection === 'dashboard'">
        <div class="mb-1 flex items-center px-2 text-xs font-medium uppercase tracking-wider text-tertiary">
          <span><span class="text-ok">#</span> Dashboards</span>
          <UButton v-if="store.canWrite" class="ml-auto" icon="i-heroicons-plus" color="gray" variant="ghost" size="2xs" square @click="openCreate('dashboard')" />
        </div>
        <div v-for="d in store.workspaceDashboards" :key="d.id"
          class="flex h-8 cursor-pointer items-center gap-2 rounded px-2 hover:bg-hover"
          :class="selectedDashboardId === d.id ? 'bg-active text-white' : 'text-secondary'"
          @click="selectedDashboardId = d.id"><UIcon name="i-heroicons-squares-2x2" class="text-tertiary" /> {{ d.name }}</div>
        <div v-if="!store.workspaceDashboards.length" class="px-2 py-1 text-xs text-tertiary/60">// no dashboards yet</div>
      </template>

      <!-- INBOX section: фильтры -->
      <template v-if="store.activeWorkspaceId && activeSection === 'inbox'">
        <div class="mb-1 px-2 text-xs font-medium uppercase tracking-wider text-tertiary"><span class="text-ok"># </span>Filters</div>
        <div v-for="f in INBOX_FILTERS" :key="f.key"
          class="flex h-8 cursor-pointer items-center gap-2 rounded px-2 hover:bg-hover"
          :class="inboxFilter === f.key ? 'bg-active text-white' : 'text-secondary'"
          @click="inboxFilter = f.key as any">
          <UIcon :name="f.icon" class="text-tertiary" />
          <span class="flex-1">{{ f.label }}</span>
          <span class="text-xs text-tertiary">{{ inboxCount(f.key) }}</span>
        </div>
      </template>

      <!-- CALENDAR section: чьи календари показывать -->
      <template v-if="store.activeWorkspaceId && activeSection === 'calendar'">
        <UButton block color="gray" variant="ghost" size="sm" class="mb-4" @click="openCalCreate(calAnchor)">[ new event ]</UButton>
        <div class="mb-1 px-2 text-xs font-medium uppercase tracking-wider text-tertiary"><span class="text-ok"># </span>Calendars</div>
        <div v-for="u in [store.users[store.currentUserId], ...store.wsUsers].filter(Boolean)" :key="u.id"
          class="flex h-8 cursor-pointer items-center gap-2 rounded px-2 hover:bg-hover"
          @click="toggleCalUser(u.id)">
          <span class="h-3 w-3 shrink-0 rounded-sm ring-1 ring-white/20" :style="{ background: calUsers.includes(u.id) ? colorForUser(u.id) : 'transparent' }" />
          <span class="flex-1 truncate" :class="calUsers.includes(u.id) ? 'text-white' : 'text-tertiary'">{{ u.id === store.currentUserId ? 'Me' : (u.name || u.email) }}</span>
        </div>
      </template>

      <!-- REPORTS section sidebar: диапазон / группировка / участник -->
      <template v-if="store.activeWorkspaceId && activeSection === 'reports'">
        <div class="mb-1 px-2 text-xs font-medium uppercase tracking-wider text-tertiary"><span class="text-ok"># </span>Date range</div>
        <div v-for="p in [{ k: 'today', l: 'Today' }, { k: 'week', l: 'This week' }, { k: 'month', l: 'This month' }, { k: 'last7', l: 'Last 7 days' }, { k: 'last30', l: 'Last 30 days' }, { k: 'custom', l: 'Custom' }]" :key="p.k"
          class="flex h-8 cursor-pointer items-center rounded px-2 text-sm hover:bg-hover"
          :class="repPreset === p.k ? 'bg-active text-white' : 'text-secondary'"
          @click="setRange(p.k)">{{ p.l }}</div>
        <div v-if="repPreset === 'custom'" class="mt-1 space-y-1.5 px-2">
          <DatePicker v-model="repFrom" block />
          <DatePicker v-model="repTo" block />
        </div>

        <div class="mb-1 mt-5 px-2 text-xs font-medium uppercase tracking-wider text-tertiary"><span class="text-ok"># </span>Group by</div>
        <div v-for="g in [{ k: 'user', l: 'Person', i: 'i-heroicons-user' }, { k: 'issue', l: 'Task', i: 'i-heroicons-rectangle-stack' }, { k: 'day', l: 'Day', i: 'i-heroicons-calendar' }]" :key="g.k"
          class="flex h-8 cursor-pointer items-center gap-2 rounded px-2 text-sm hover:bg-hover"
          :class="repGroup === g.k ? 'bg-active text-white' : 'text-secondary'"
          @click="repGroup = g.k as any"><UIcon :name="g.i" class="text-tertiary" /> {{ g.l }}</div>

        <div class="mb-1 mt-5 px-2 text-xs font-medium uppercase tracking-wider text-tertiary"><span class="text-ok"># </span>Person</div>
        <div class="flex h-8 cursor-pointer items-center gap-2 rounded px-2 text-sm hover:bg-hover"
          :class="!repUser ? 'bg-active text-white' : 'text-secondary'" @click="repUser = ''"><UIcon name="i-heroicons-users" class="text-tertiary" /> Everyone</div>
        <div v-for="u in [store.users[store.currentUserId], ...store.wsUsers].filter(Boolean)" :key="u.id"
          class="flex h-8 cursor-pointer items-center gap-2 rounded px-2 text-sm hover:bg-hover"
          :class="repUser === u.id ? 'bg-active text-white' : 'text-secondary'" @click="repUser = u.id">
          <Avatar :name="u.name || u.email" :avatar-key="u.avatarKey" :size="18" />
          <span class="truncate">{{ u.id === store.currentUserId ? 'Me' : (u.name || u.email) }}</span>
        </div>
      </template>

      <!-- CHATS section: каналы + личные -->
      <template v-if="store.activeWorkspaceId && activeSection === 'chats'">
        <div class="mb-1 flex items-center px-2 text-xs font-medium uppercase tracking-wider text-tertiary">
          <span><span class="text-ok">#</span> Channels</span>
          <UButton v-if="store.canWrite" class="ml-auto" icon="i-heroicons-plus" color="gray" variant="ghost" size="2xs" square @click="openCreate('channel')" />
        </div>
        <div v-for="c in store.channelList" :key="c.id"
          class="flex h-8 cursor-pointer items-center gap-2 rounded px-2 hover:bg-hover"
          :class="store.activeChannelId === c.id ? 'bg-active text-white' : store.channelUnreadCount(c.id) ? 'font-semibold text-white' : 'text-secondary'"
          @click="openChannel(c.id)"><UIcon :name="c.private ? 'i-heroicons-lock-closed' : 'i-heroicons-hashtag'" class="text-tertiary" /> <span class="truncate">{{ c.name }}</span>
          <span v-if="store.channelUnreadCount(c.id) && store.activeChannelId !== c.id" class="ml-auto rounded bg-accent px-1.5 text-[11px] font-medium text-white">{{ store.channelUnreadCount(c.id) }}</span></div>
        <div v-if="!store.channelList.length" class="px-2 py-1 text-xs text-tertiary/60">// no channels</div>

        <div class="mb-1 mt-4 px-2 text-xs font-medium uppercase tracking-wider text-tertiary"><span class="text-ok"># </span>Direct messages</div>
        <div v-for="u in store.wsUsers" :key="u.id"
          class="flex h-8 cursor-pointer items-center gap-2 rounded px-2 hover:bg-hover"
          :class="currentChannel?.kind === 'dm' && store.dmOtherId(currentChannel.id) === u.id ? 'bg-active text-white' : store.channelUnreadCount(store.dmChannelWith(u.id)) ? 'font-semibold text-white' : 'text-secondary'"
          @click="openDM(u.id)">
          <Avatar :name="u.name || u.email" :avatar-key="u.avatarKey" :size="20" />
          <span class="truncate">{{ u.name || u.email }}</span>
          <span v-if="store.channelUnreadCount(store.dmChannelWith(u.id))" class="ml-auto rounded bg-accent px-1.5 text-[11px] font-medium text-white">{{ store.channelUnreadCount(store.dmChannelWith(u.id)) }}</span>
        </div>
        <div v-if="!store.wsUsers.length" class="px-2 py-1 text-xs text-tertiary/60">// no other users</div>
      </template>

      <!-- MEETINGS section -->
      <template v-if="store.activeWorkspaceId && activeSection === 'meetings'">
        <UButton v-if="store.canWrite" block color="gray" size="sm" class="mb-4" @click="openCreate('meeting')">[ new meeting ]</UButton>
        <div v-if="store.liveMeetings.length" class="mb-1 px-2 text-xs font-medium uppercase tracking-wider text-tertiary"><span class="text-ok"># </span>Live now</div>
        <div v-for="m in store.liveMeetings" :key="m.id"
          class="flex h-8 cursor-pointer items-center gap-2 rounded px-2 text-secondary hover:bg-hover" @click="joinScheduled(m.id)">
          <span class="flex h-2 w-2 shrink-0 rounded-full bg-danger" />
          <span class="truncate">{{ m.kind === 'huddle' ? 'Huddle' : m.title }}</span>
          <span class="ml-auto text-xs text-tertiary">{{ store.meetingJoined(m.id).length }}</span>
        </div>
        <div class="mb-1 mt-4 px-2 text-xs font-medium uppercase tracking-wider text-tertiary"><span class="text-ok"># </span>Upcoming</div>
        <div v-for="m in store.scheduledMeetings" :key="m.id"
          class="flex h-8 cursor-pointer items-center gap-2 rounded px-2 text-secondary hover:bg-hover" @click="joinScheduled(m.id)">
          <UIcon name="i-heroicons-calendar" class="shrink-0 text-tertiary" />
          <span class="truncate">{{ m.title || 'Meeting' }}</span>
        </div>
        <div v-if="!store.scheduledMeetings.length && !store.liveMeetings.length" class="px-2 py-1 text-xs text-tertiary/60">// no meetings</div>
      </template>

      <!-- TASKS section: команды/проекты/циклы/поля -->
      <template v-if="store.activeWorkspaceId && activeSection === 'tasks'">
        <!-- персональные срезы -->
        <div class="mb-5">
          <div v-for="p in PERSONAL" :key="p.key"
            class="flex h-8 cursor-pointer items-center gap-2 rounded px-2 hover:bg-hover"
            :class="personalView === p.key ? 'bg-active text-white' : 'text-secondary'"
            @click="openPersonal(p.key)"><UIcon :name="p.icon" class="text-tertiary" /> {{ p.label }}</div>
        </div>

        <div class="mb-0.5 px-2 text-xs font-medium uppercase tracking-wider text-tertiary"><span class="text-ok"># </span>Your teams</div>
        <template v-for="t in store.teamList" :key="t.id">
          <div class="group flex h-8 cursor-pointer items-center gap-1.5 rounded px-1.5 hover:bg-hover"
            :style="{ paddingLeft: 6 + depth(t.path) * 12 + 'px' }" @click="expandTeam[t.id] = !expandTeam[t.id]">
            <span class="shrink-0 text-tertiary">{{ expandTeam[t.id] ? '▾' : '▸' }}</span>
            <UIcon name="i-heroicons-cube" class="shrink-0 text-tertiary" />
            <span class="truncate font-medium text-secondary">{{ t.name }}</span>
            <span class="text-xs text-tertiary">{{ t.key }}</span>
          </div>
          <div v-if="expandTeam[t.id]" :style="{ paddingLeft: 12 + depth(t.path) * 12 + 'px' }">
            <!-- Issues -->
            <div class="flex h-8 cursor-pointer items-center gap-2 rounded px-2 text-sm hover:bg-hover"
              :class="selectedTeamId === t.id && !selectedDocId && !selectedCycleId && !teamSection && !personalView ? 'bg-active text-white' : 'text-tertiary'"
              @click="selectTeamIssues(t)"><UIcon name="i-heroicons-queue-list" /> Issues</div>
            <!-- Cycles -->
            <div class="flex h-8 cursor-pointer items-center gap-1 rounded px-2 text-sm hover:bg-hover"
              :class="selectedTeamId === t.id && teamSection === 'cycles' ? 'bg-active text-white' : 'text-tertiary'"
              @click="openTeamSection(t.id, 'cycles')">
              <span class="cursor-pointer rounded text-tertiary hover:bg-active" @click.stop="expandCycles[t.id] = !expandCycles[t.id]">{{ expandCycles[t.id] ? '▾' : '▸' }}</span> Cycles
            </div>
            <div v-if="expandCycles[t.id]" class="pl-6">
              <div class="flex h-8 cursor-pointer items-center rounded px-2 text-sm hover:bg-hover"
                :class="selectedCycleId && selectedCycleId === activeCycleOf(t.id)?.id ? 'bg-active text-white' : 'text-tertiary'"
                @click="openCycleFor(t.id, activeCycleOf(t.id))">Current</div>
              <div class="flex h-8 cursor-pointer items-center rounded px-2 text-sm hover:bg-hover"
                :class="selectedCycleId && selectedCycleId === upcomingCycleOf(t.id)?.id ? 'bg-active text-white' : 'text-tertiary'"
                @click="openCycleFor(t.id, upcomingCycleOf(t.id))">Upcoming</div>
            </div>
            <!-- Milestones -->
            <div class="flex h-8 cursor-pointer items-center gap-2 rounded px-2 text-sm hover:bg-hover"
              :class="selectedTeamId === t.id && teamSection === 'milestones' ? 'bg-active text-white' : 'text-tertiary'"
              @click="openTeamSection(t.id, 'milestones')"><span class="w-3.5" /><UIcon name="i-heroicons-flag" /> Milestones</div>
            <!-- Docs (дерево) -->
            <div class="flex h-8 cursor-pointer items-center gap-1 rounded px-2 text-sm hover:bg-hover"
              :class="selectedTeamId === t.id && teamSection === 'docs' ? 'bg-active text-white' : 'text-tertiary'"
              @click="openTeamSection(t.id, 'docs')">
              <span class="cursor-pointer rounded text-tertiary hover:bg-active" @click.stop="expandDocs[t.id] = !expandDocs[t.id]">{{ expandDocs[t.id] ? '▾' : '▸' }}</span> Docs
            </div>
            <div v-if="expandDocs[t.id]" class="pl-6">
              <div v-for="nd in store.teamDocTree(t.id)" :key="nd.doc.id"
                class="flex h-8 cursor-pointer items-center gap-1.5 rounded px-2 text-sm hover:bg-hover"
                :class="selectedDocId === nd.doc.id ? 'bg-active text-white' : 'text-tertiary'" :style="{ paddingLeft: 8 + nd.depth * 12 + 'px' }"
                @click="selectedTeamId = t.id; openDoc(nd.doc.id)"><UIcon name="i-heroicons-document" class="shrink-0" /> <span class="truncate">{{ nd.doc.title }}</span></div>
            </div>
            <!-- Views -->
            <div class="flex h-8 cursor-pointer items-center gap-1 rounded px-2 text-sm hover:bg-hover"
              :class="selectedTeamId === t.id && teamSection === 'views' ? 'bg-active text-white' : 'text-tertiary'"
              @click="openTeamSection(t.id, 'views')">
              <span class="cursor-pointer rounded text-tertiary hover:bg-active" @click.stop="expandViews[t.id] = !expandViews[t.id]">{{ expandViews[t.id] ? '▾' : '▸' }}</span> Views
            </div>
            <div v-if="expandViews[t.id]" class="pl-6">
              <div v-for="v in store.teamViews(t.id)" :key="v.id"
                class="flex h-8 cursor-pointer items-center gap-1.5 rounded px-2 text-sm text-tertiary hover:bg-hover"
                @click="selectedTeamId = t.id; selectedDocId = ''; openView(v.id)"><UIcon name="i-heroicons-star" /> {{ v.name }}</div>
              <div v-if="!store.teamViews(t.id).length" class="px-2 py-0.5 text-xs text-tertiary/50">no views</div>
            </div>
          </div>
        </template>
      </template>

      <p v-if="err" class="mt-3 text-xs text-danger">{{ err }}</p>
    </aside>

    <!-- Контент-область: отдельные плавающие панели -->
    <div class="flex flex-1 gap-2 overflow-hidden p-2">
    <!-- ④ Main content -->
    <main ref="mainEl" class="flex-1 overflow-y-auto rounded-xl border border-subtle bg-surface">
      <!-- скелетон main (холодный старт) -->
      <template v-if="!store.loaded">
        <div class="border-b border-subtle px-4 py-3"><div class="skeleton h-4 w-40" /></div>
        <div class="space-y-2 px-4 py-3">
          <div v-for="n in 8" :key="n" class="flex items-center gap-3">
            <div class="skeleton h-4 w-4 rounded-full" /><div class="skeleton h-3 w-12" /><div class="skeleton h-3 flex-1" :style="{ maxWidth: 40 + (n * 11) % 45 + '%' }" />
          </div>
        </div>
      </template>

      <!-- DASHBOARD section -->
      <!-- CALENDAR: месяц-сетка -->
      <template v-else-if="activeSection === 'calendar'">
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
          <UButton class="ml-auto" size="xs" color="gray" variant="ghost" @click="openIcal">[ subscribe ]</UButton>
          <UButton size="xs" color="gray" variant="ghost" @click="openCalCreate(calAnchor)">[ new event ]</UButton>
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
              @click="openCalCreate(cell.ds)">
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
            <div v-for="ds in gridDays" :key="ds" class="cursor-pointer border-l border-subtle/50 py-1.5 text-center hover:bg-hover" @click="calAnchor = ds; calView = 'day'">
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

      <!-- REPORTS: учёт времени -->
      <template v-else-if="activeSection === 'reports'">
        <ClientOnly>
        <div class="mx-auto w-full max-w-4xl px-6 py-6">
          <div class="mb-4 flex items-center gap-2">
            <h1 class="text-lg font-semibold text-white">Time report</h1>
            <span class="text-sm text-tertiary">{{ dayShort(repFrom) }} – {{ dayShort(repTo) }}{{ repUser ? ' · ' + (repUser === store.currentUserId ? 'Me' : store.userName(repUser)) : '' }}</span>
            <UButton class="ml-auto" color="gray" size="xs" :disabled="!repRows.length" @click="exportReportCsv">[ export csv ]</UButton>
          </div>

          <!-- сводные карточки -->
          <div class="mb-5 grid grid-cols-2 gap-3 sm:grid-cols-4">
            <div class="rounded-lg border border-subtle bg-elevated p-3">
              <div class="text-xs text-tertiary">Total</div><div class="text-2xl font-semibold text-white">{{ fmtMin(repTotal) }}</div>
            </div>
            <div class="rounded-lg border border-subtle bg-elevated p-3">
              <div class="text-xs text-tertiary">Entries</div><div class="text-2xl font-semibold text-white">{{ repEntries }}</div>
            </div>
            <div class="rounded-lg border border-subtle bg-elevated p-3">
              <div class="text-xs text-tertiary">Active days</div><div class="text-2xl font-semibold text-white">{{ repActiveDays }}</div>
            </div>
            <div class="rounded-lg border border-subtle bg-elevated p-3">
              <div class="text-xs text-tertiary">Avg / active day</div><div class="text-2xl font-semibold text-white">{{ fmtMin(repAvgPerDay) }}</div>
            </div>
          </div>

          <div v-if="repLoading" class="py-10 text-center text-sm text-tertiary">Loading…</div>
          <div v-else-if="!repRows.length" class="py-16 text-center text-sm text-tertiary">No time tracked in this range.</div>
          <template v-else>
            <!-- дневной график -->
            <div class="mb-6">
              <div class="mb-2 text-xs font-medium uppercase tracking-wider text-tertiary"><span class="text-ok"># </span>Daily</div>
              <div class="flex items-end gap-1" style="height: 120px">
                <div v-for="d in repDaily" :key="d.key" class="group/bar relative flex flex-1 flex-col items-center justify-end" :title="`${dayShort(d.key)}: ${fmtMin(d.minutes)}`">
                  <div class="w-full rounded-t bg-accent/50 transition-colors group-hover/bar:bg-accent" :style="{ height: (d.minutes / repDailyMax * 96) + 'px' }" />
                </div>
              </div>
              <div class="mt-1 flex gap-1 text-[10px] text-tertiary">
                <div v-for="d in repDaily" :key="d.key" class="flex-1 truncate text-center">{{ parseD(d.key).getDate() }}</div>
              </div>
            </div>

            <!-- разбивка по группе -->
            <div class="mb-2 text-xs font-medium uppercase tracking-wider text-tertiary"><span class="text-ok"># </span>By {{ repGroup === 'user' ? 'person' : repGroup === 'issue' ? 'task' : 'day' }}</div>
            <div class="space-y-1.5">
              <div v-for="r in repRows" :key="r.key" class="flex items-center gap-3">
                <div class="w-56 shrink-0 truncate text-sm" :title="repLabel(r.key)">{{ repLabel(r.key) }}</div>
                <div class="flex-1"><TermBar :width="28" :pct="repMax ? r.minutes / repMax * 100 : 0" hide-label /></div>
                <div class="w-12 shrink-0 text-right text-xs text-tertiary">{{ repTotal ? Math.round(r.minutes / repTotal * 100) : 0 }}%</div>
                <div class="w-20 shrink-0 text-right text-sm tabular-nums text-white">{{ fmtMin(r.minutes) }}</div>
                <div class="w-10 shrink-0 text-right text-xs text-tertiary">{{ r.count }}</div>
              </div>
            </div>
          </template>

          <!-- AI insights (Фаза 4): темы + аномалии + настроение -->
          <div v-if="aiThemes.length || aiOutliers.length || (aiMood && aiMood.n)" class="mt-8 border-t border-subtle pt-6">
            <!-- настроение команды -->
            <div v-if="aiMood && aiMood.n" class="mb-6">
              <div class="mb-2 text-xs font-medium uppercase tracking-wider text-tertiary"><span class="text-accent2">≈ </span>Team mood <span class="text-tertiary/60">({{ aiMood.n }} comments)</span></div>
              <div class="flex items-center gap-3">
                <div class="h-2 flex-1 overflow-hidden rounded-full bg-elevated">
                  <div class="h-full rounded-full transition-all" :style="{ width: moodPct + '%', background: aiMood.avg >= 0 ? '#9ece6a' : '#f7768e' }" />
                </div>
                <span class="w-24 shrink-0 text-right text-xs text-tertiary">{{ aiMood.pos }}+ / {{ aiMood.neg }}− / {{ aiMood.neu }}·</span>
              </div>
            </div>

            <!-- топик-кластеры -->
            <div v-if="aiThemes.length" class="mb-6">
              <div class="mb-2 text-xs font-medium uppercase tracking-wider text-tertiary"><span class="text-accent2">≈ </span>AI themes</div>
              <div class="space-y-1.5">
                <div v-for="t in aiThemes" :key="t.label" class="flex items-center gap-3 rounded-lg border border-subtle bg-elevated px-3 py-2">
                  <span class="w-10 shrink-0 text-right text-sm tabular-nums text-white">{{ t.size }}</span>
                  <span class="min-w-0 flex-1 truncate text-sm text-secondary" :title="t.label">{{ t.label }}</span>
                  <span class="shrink-0 text-xs text-tertiary">{{ (t.sample || []).join(' · ') }}</span>
                </div>
              </div>
            </div>

            <!-- аномалии (выбивающиеся задачи) -->
            <div v-if="aiOutliers.length">
              <div class="mb-2 text-xs font-medium uppercase tracking-wider text-tertiary"><span class="text-accent2">≈ </span>Outliers <span class="text-tertiary/60">(unusual vs the rest)</span></div>
              <div class="space-y-1.5">
                <div v-for="o in aiOutliers" :key="o.identifier" class="flex items-center gap-3 rounded-lg border border-subtle bg-elevated px-3 py-2">
                  <span class="shrink-0 text-xs font-medium text-accent2">{{ o.identifier }}</span>
                  <span class="min-w-0 flex-1 truncate text-sm text-secondary" :title="o.title">{{ o.title }}</span>
                  <span class="shrink-0 text-xs text-tertiary tabular-nums">{{ Math.round(o.score * 100) }}%</span>
                </div>
              </div>
            </div>
          </div>
        </div>
        </ClientOnly>
      </template>

      <template v-else-if="activeSection === 'dashboard'">
      <template v-if="selectedDashboard">
        <div class="sticky top-0 z-10 flex items-center gap-2 border-b border-subtle bg-surface px-4 py-3 text-sm">
          <UIcon name="i-heroicons-squares-2x2" class="text-tertiary" />
          <span class="font-medium">{{ selectedDashboard.name }}</span>
          <UButton class="ml-auto" size="xs" color="gray" variant="ghost" @click="showWidget = true">[ add widget ]</UButton>
        </div>
        <div class="grid grid-cols-1 gap-3 p-4 sm:grid-cols-2 xl:grid-cols-3">
          <div v-for="w in store.widgetsOf(selectedDashboardId)" :key="w.id"
            class="group relative rounded-lg border border-subtle bg-elevated p-4 transition-shadow"
            :class="[(w.w || 1) >= 2 ? 'sm:col-span-2' : '', draggedWidget === w.id ? 'opacity-40' : '', 'cursor-grab']"
            draggable="true" @dragstart="draggedWidget = w.id" @dragend="draggedWidget = ''"
            @dragover.prevent @drop="draggedWidget && draggedWidget !== w.id && store.moveWidget(selectedDashboardId, draggedWidget, w.id)">
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
          <div v-if="!store.widgetsOf(selectedDashboardId).length"
            class="flex h-32 cursor-pointer items-center justify-center rounded-lg border border-dashed border-subtle text-sm text-tertiary hover:bg-hover"
            @click="showWidget = true"><span class="text-ok">$</span>&nbsp;add your first widget</div>
        </div>
      </template>
      <div v-else class="p-8 text-sm text-tertiary">Pick or create a dashboard →</div>
      </template>

      <!-- INBOX section -->
      <SectionsInboxSection v-else-if="activeSection === 'inbox'" :filter="inboxFilter" :notif-prefs="notifPrefs" @open="openNotif" />

      <!-- CHATS section -->
      <SectionsChatSection v-else-if="activeSection === 'chats'" ref="chatSection"
        :link-copied="linkCopied" :first-unread-msg-id="firstUnreadMsgId" @copy-link="copyLink" />

      <!-- MEETINGS section -->
      <SectionsMeetingsSection v-else-if="activeSection === 'meetings'" @create="openCreate('meeting')" @join="joinScheduled" />

      <!-- TASKS section -->
      <template v-else>
      <!-- Personal view (My issues / Active / Created) -->
      <template v-if="personalView">
        <div class="sticky top-0 z-10 flex items-center gap-2 border-b border-subtle bg-surface px-4 py-3 text-sm">
          <UIcon :name="PERSONAL.find((p) => p.key === personalView)?.icon" class="text-tertiary" />
          <span class="font-medium">{{ PERSONAL.find((p) => p.key === personalView)?.label }}</span>
          <span class="text-tertiary">{{ store.personalIssues(personalView).length }}</span>
        </div>
        <template v-for="g in personalGroups" :key="g.key">
          <div class="sticky top-[49px] z-[5] flex items-center gap-2 border-b border-subtle/60 bg-elevated/80 px-4 py-1.5 text-xs font-medium backdrop-blur">
            <span class="h-2 w-2 rounded-full" :style="{ background: g.color }" />
            <span class="uppercase tracking-wider" :style="{ color: g.color }">{{ g.label }}</span>
            <span class="text-tertiary">{{ g.items.length }}</span>
          </div>
          <div v-for="i in g.items" :key="i.id"
            class="flex cursor-pointer items-center gap-3 border-b border-subtle/40 px-4 py-2.5 hover:bg-hover"
            :class="{ 'bg-active': selectedIssueId === i.id }" @click="openIssue(i.id)">
            <span v-if="i.priority" class="inline-block shrink-0 w-4 text-center leading-none" :style="{ color: prioColor(i.priority) }">{{ prioGlyph(i.priority) }}</span>
            <UIcon v-else name="i-heroicons-minus" class="shrink-0 text-tertiary/50" />
            <span class="min-w-16 shrink-0 whitespace-nowrap text-sm text-accent2">{{ i.identifier }}</span>
            <span class="inline-block shrink-0 w-4 text-center leading-none" :style="{ color: statusColor(store.states[i.stateId]?.category) }">{{ statusGlyph(store.states[i.stateId]?.category) }}</span>
            <span class="flex-1 truncate">{{ i.title }}</span>
            <span v-if="d10(i.scheduledDate) || d10(i.dueDate)" class="shrink-0 text-xs"
              :class="g.key === 'overdue' ? 'text-danger' : 'text-tertiary'">{{ d10(i.scheduledDate) || d10(i.dueDate) }}</span>
            <span class="shrink-0 text-xs text-tertiary">{{ store.teams[i.teamId]?.key }}</span>
          </div>
        </template>
        <div v-if="!store.personalIssues(personalView).length" class="px-4 py-8 text-center text-sm text-tertiary">// no issues</div>
      </template>

      <!-- CYCLES page -->
      <template v-else-if="teamSection === 'cycles' && selectedTeam">
        <div class="sticky top-0 z-10 flex items-center gap-1.5 border-b border-subtle bg-surface px-4 py-3 text-sm">
          <span class="text-tertiary">{{ store.activeWorkspace?.name }}</span>
          <span class="text-tertiary">/</span>
          <button class="text-tertiary hover:text-secondary" @click="selectTeamIssues(selectedTeam)">{{ selectedTeam.name }}</button>
          <span class="text-tertiary">/</span>          <span class="font-medium">cycles</span>
          <span class="text-tertiary">{{ store.teamCycles(selectedTeamId).length }}</span>
          <UButton v-if="store.canManageTeam(selectedTeamId)" class="ml-auto" size="xs" color="gray" variant="ghost" @click="openCreate('cycle')">[ new cycle ]</UButton>
        </div>
        <div class="grid grid-cols-1 gap-3 p-4 sm:grid-cols-2 xl:grid-cols-3">
          <div v-for="c in store.teamCycles(selectedTeamId)" :key="c.id"
            class="cursor-pointer rounded-lg border border-subtle bg-elevated p-4 hover:bg-hover"
            @click="openCycleFor(selectedTeamId, c)">
            <div class="mb-1 flex items-center gap-2">
              <UIcon name="i-heroicons-play-circle" :class="c.status === 'active' ? 'text-accent' : 'text-tertiary'" />
              <span class="font-medium">Cycle {{ c.number }}</span>
              <span class="ml-auto rounded bg-surface px-1.5 text-xs text-tertiary">{{ c.status }}</span>
            </div>
            <div class="text-xs text-tertiary">{{ String(c.startsOn).slice(0,10) }} → {{ String(c.endsOn).slice(0,10) }}</div>
            <div class="mt-2 text-xs text-tertiary">{{ store.cycleStats(c.id).completed }}/{{ store.cycleStats(c.id).scope }} done</div>
          </div>
          <div v-if="store.canManageTeam(selectedTeamId) && !store.teamCycles(selectedTeamId).length"
            class="flex h-28 cursor-pointer items-center justify-center rounded-lg border border-dashed border-subtle text-sm text-tertiary hover:bg-hover"
            @click="openCreate('cycle')">+ Create first cycle</div>
        </div>
      </template>

      <!-- MILESTONES page -->
      <template v-else-if="teamSection === 'milestones' && selectedTeam">
        <!-- детали выбранной вехи -->
        <template v-if="selectedMs">
          <div class="sticky top-0 z-10 border-b border-subtle bg-surface px-4 py-3">
            <div class="flex items-center gap-1.5 text-sm">
              <span class="text-tertiary">{{ store.activeWorkspace?.name }}</span>
              <span class="text-tertiary">/</span>
              <button class="text-tertiary hover:text-secondary" @click="selectTeamIssues(selectedTeam)">{{ selectedTeam.name }}</button>
              <span class="text-tertiary">/</span>
              <button class="text-tertiary hover:text-secondary" @click="selectedMsId = ''">Milestones</button>
              <span class="text-tertiary">/</span>
              <UIcon name="i-heroicons-flag" :class="selectedMs.status === 'done' ? 'text-accent' : selectedMs.status === 'active' ? 'text-warn' : 'text-tertiary'" />
              <span class="font-medium">{{ selectedMs.name }}</span>
              <USelectMenu :model-value="selectedMs.status" :options="labelOpts(['planned', 'active', 'done'])" value-attribute="value" option-attribute="label" size="xs" class="w-28" @update:model-value="(st) => store.updateMilestone(selectedMs.id, { status: st })" />
              <span v-if="selectedMs.targetDate" class="text-xs" :class="msOverdue(selectedMs) ? 'text-danger' : 'text-tertiary'">{{ String(selectedMs.targetDate).slice(0,10) }}{{ msOverdue(selectedMs) ? ' · overdue' : '' }}</span>
              <span class="ml-auto text-xs text-tertiary">{{ store.milestoneProgress(selectedMs.id).done }}/{{ store.milestoneProgress(selectedMs.id).total }}</span>
            </div>
            <TermBar class="mt-2 block" :width="20" :done="store.milestoneProgress(selectedMs.id).done" :total="store.milestoneProgress(selectedMs.id).total" />
          </div>
          <div v-for="i in store.milestoneIssues(selectedMs.id)" :key="i.id"
            class="flex cursor-pointer items-center gap-3 border-b border-subtle/40 px-4 py-2.5 hover:bg-hover" @click="openIssue(i.id)">
            <span class="inline-block shrink-0 w-4 text-center leading-none" :style="{ color: statusColor(store.states[i.stateId]?.category) }">{{ statusGlyph(store.states[i.stateId]?.category) }}</span>
            <span class="min-w-14 shrink-0 whitespace-nowrap text-sm text-accent2">{{ i.identifier }}</span>
            <span class="flex-1 truncate">{{ i.title }}</span>
            <span v-if="i.assigneeId" class="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-elevated text-[10px]">{{ (store.userName(i.assigneeId) || '?')[0].toUpperCase() }}</span>
          </div>
          <div v-if="!store.milestoneIssues(selectedMs.id).length" class="py-10 text-center text-sm text-tertiary">// no issues — assign via Milestone field</div>
        </template>

        <!-- список вех -->
        <template v-else>
          <div class="sticky top-0 z-10 flex items-center gap-1.5 border-b border-subtle bg-surface px-4 py-3 text-sm">
            <span class="text-tertiary">{{ store.activeWorkspace?.name }}</span>
            <span class="text-tertiary">/</span>
            <button class="text-tertiary hover:text-secondary" @click="selectTeamIssues(selectedTeam)">{{ selectedTeam.name }}</button>
            <span class="text-tertiary">/</span>            <span class="font-medium">milestones</span>
            <span class="text-tertiary">{{ store.teamMilestones(selectedTeamId).length }}</span>
            <UButton v-if="store.canManageTeam(selectedTeamId)" class="ml-auto" size="xs" color="gray" variant="ghost" @click="openCreate('milestone')">[ new milestone ]</UButton>
          </div>
          <div class="p-4">
            <div class="space-y-2">
              <div v-for="m in store.teamMilestones(selectedTeamId)" :key="m.id"
                class="group cursor-pointer rounded-lg border border-subtle bg-elevated p-3 hover:border-strong" @click="openMilestone(m.id)">
                <div class="flex items-center gap-2">
                  <UIcon name="i-heroicons-flag" :class="m.status === 'done' ? 'text-accent' : m.status === 'active' ? 'text-warn' : 'text-tertiary'" />
                  <span class="flex-1 truncate font-medium">{{ m.name }}</span>
                  <span class="rounded bg-surface px-1.5 text-xs text-tertiary">{{ m.status }}</span>
                  <span v-if="m.targetDate" class="text-xs" :class="msOverdue(m) ? 'text-danger' : 'text-tertiary'">{{ String(m.targetDate).slice(5,10) }}</span>
                  <UButton icon="i-heroicons-trash" color="gray" variant="ghost" size="2xs" square class="opacity-0 group-hover:opacity-100" @click.stop="store.deleteMilestone(m.id)" />
                </div>
                <div class="mt-2 flex items-center gap-2 text-xs text-tertiary">
                  <TermBar :width="14" :done="store.milestoneProgress(m.id).done" :total="store.milestoneProgress(m.id).total" />
                  <span>{{ store.milestoneProgress(m.id).done }}/{{ store.milestoneProgress(m.id).total }}</span>
                </div>
              </div>
              <div v-if="!store.teamMilestones(selectedTeamId).length" class="py-10 text-center text-sm text-tertiary">// no milestones</div>
            </div>
          </div>
        </template>
      </template>

      <!-- DOCS home -->
      <template v-else-if="teamSection === 'docs' && selectedTeam">
        <div class="sticky top-0 z-10 flex items-center gap-1.5 border-b border-subtle bg-surface px-4 py-3 text-sm">
          <span class="text-tertiary">{{ store.activeWorkspace?.name }}</span>
          <span class="text-tertiary">/</span>
          <button class="text-tertiary hover:text-secondary" @click="selectTeamIssues(selectedTeam)">{{ selectedTeam.name }}</button>
          <span class="text-tertiary">/</span>
          <UIcon name="i-heroicons-document-text" class="text-tertiary" />
          <span class="font-medium">docs</span>
          <span class="text-tertiary">{{ store.teamDocs(selectedTeamId).length }}</span>
          <UButton v-if="store.canWrite" class="ml-auto" size="xs" color="gray" variant="ghost" @click="openCreate('doc')">[ new doc ]</UButton>
        </div>
        <div class="p-4">
          <div class="mb-2 text-xs font-medium uppercase tracking-wider text-tertiary"><span class="text-ok"># </span>Recent</div>
          <div class="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-3">
            <div v-for="d in store.teamDocs(selectedTeamId)" :key="d.id"
              class="cursor-pointer rounded-lg border border-subtle bg-elevated p-4 hover:bg-hover"
              @click="openDoc(d.id)">
              <div class="flex items-center gap-2"><UIcon name="i-heroicons-document" class="text-tertiary" /><span class="truncate font-medium">{{ d.title }}</span></div>
              <div class="mt-1 line-clamp-2 text-xs text-tertiary">{{ d.body?.md || 'Empty document' }}</div>
            </div>
            <div v-if="store.canWrite && !store.teamDocs(selectedTeamId).length"
              class="flex h-28 cursor-pointer items-center justify-center rounded-lg border border-dashed border-subtle text-sm text-tertiary hover:bg-hover"
              @click="openCreate('doc')">+ Create first doc</div>
          </div>
        </div>
      </template>

      <!-- VIEWS page -->
      <template v-else-if="teamSection === 'views' && selectedTeam">
        <div class="sticky top-0 z-10 flex items-center gap-1.5 border-b border-subtle bg-surface px-4 py-3 text-sm">
          <span class="text-tertiary">{{ store.activeWorkspace?.name }}</span>
          <span class="text-tertiary">/</span>
          <button class="text-tertiary hover:text-secondary" @click="selectTeamIssues(selectedTeam)">{{ selectedTeam.name }}</button>
          <span class="text-tertiary">/</span>
          <UIcon name="i-heroicons-square-3-stack-3d" class="text-tertiary" />
          <span class="font-medium">views</span>
          <span class="text-tertiary">{{ store.teamViews(selectedTeamId).length }}</span>
        </div>
        <div class="p-4">
          <div class="divide-y divide-subtle/40 overflow-hidden rounded-lg border border-subtle">
            <div v-for="v in store.teamViews(selectedTeamId)" :key="v.id"
              class="group flex cursor-pointer items-center gap-2 bg-elevated px-4 py-2.5 hover:bg-hover"
              @click="openView(v.id)">
              <UIcon name="i-heroicons-star" class="text-tertiary" />
              <span class="flex-1 truncate">{{ v.name }}</span>
              <button class="hidden text-xs text-tertiary hover:text-secondary group-hover:inline" title="rename"
                @click.stop="renameView(v)">✎</button>
              <button class="hidden text-xs text-tertiary hover:text-red-400 group-hover:inline" title="delete"
                @click.stop="deleteView(v)">×</button>
              <UIcon name="i-heroicons-chevron-right" class="text-tertiary" />
            </div>
            <div v-if="!store.teamViews(selectedTeamId).length" class="bg-elevated px-4 py-8 text-center text-sm text-tertiary">No views — save one from a filtered Issues list</div>
          </div>
        </div>
      </template>

      <!-- DOC editor -->
      <template v-else-if="selectedDoc">
        <div class="sticky top-0 z-10 flex items-center gap-1.5 border-b border-subtle bg-surface px-4 py-3 text-sm">
          <template v-if="store.teams[selectedDoc.teamId]">
            <button class="text-tertiary hover:text-secondary" @click="selectTeamIssues(store.teams[selectedDoc.teamId])">{{ store.teams[selectedDoc.teamId].name }}</button>
            <span class="text-tertiary">/</span>
            <button class="text-tertiary hover:text-secondary" @click="openTeamSection(selectedDoc.teamId, 'docs')">Docs</button>
            <span class="text-tertiary">/</span>
          </template>
          <UIcon name="i-heroicons-document-text" class="text-tertiary" />
          <UInput :model-value="selectedDoc.title" variant="none" :padded="false" class="flex-1 font-medium"
            @change="(v:any) => store.updateDoc(selectedDoc.id, { title: v })" />
          <div class="flex gap-0.5">
            <UButton size="2xs" square icon="i-heroicons-pencil" :color="docMode==='edit'?'primary':'gray'" :variant="docMode==='edit'?'soft':'ghost'" title="Edit" @click="docMode='edit'" />
            <UButton size="2xs" square icon="i-heroicons-view-columns" :color="docMode==='split'?'primary':'gray'" :variant="docMode==='split'?'soft':'ghost'" title="Split" @click="docMode='split'" />
            <UButton size="2xs" square icon="i-heroicons-eye" :color="docMode==='preview'?'primary':'gray'" :variant="docMode==='preview'?'soft':'ghost'" title="Preview" @click="docMode='preview'" />
          </div>
          <UButton class="ml-1" :icon="linkCopied ? 'i-heroicons-check' : 'i-heroicons-link'" size="xs" color="gray" variant="ghost" square :title="linkCopied?'Copied!':'Copy link'" @click="copyLink" />
          <UButton size="xs" color="red" variant="ghost" square icon="i-heroicons-trash" @click="store.deleteDoc(selectedDoc.id); selectedDocId = ''" />
          <UButton size="xs" color="gray" variant="ghost" square icon="i-heroicons-x-mark" @click="selectedDocId = ''" />
        </div>
        <!-- toolbar -->
        <div v-if="docMode !== 'preview'" class="flex items-center gap-1 border-b border-subtle px-3 py-1">
          <UButton size="2xs" square color="gray" variant="ghost" class="font-bold" title="Bold" @click="docWrap('**')">B</UButton>
          <UButton size="2xs" square color="gray" variant="ghost" class="italic" title="Italic" @click="docWrap('_')">I</UButton>
          <UButton size="2xs" square color="gray" variant="ghost" class="font-bold" title="Heading" @click="docWrap('\n# ', '')">H</UButton>
          <UButton size="2xs" square color="gray" variant="ghost" icon="i-heroicons-list-bullet" title="List" @click="docWrap('\n- ', '')" />
          <UButton size="2xs" square color="gray" variant="ghost" icon="i-heroicons-code-bracket" title="Code" @click="docWrap('`')" />
          <UButton size="2xs" square color="gray" variant="ghost" icon="i-heroicons-link" title="Link" @click="docWrap('[', '](url)')" />
          <UButton size="2xs" square color="gray" variant="ghost" icon="i-heroicons-paper-clip" title="Attach file/image" @click="docFileEl?.click()" />
          <input ref="docFileEl" type="file" multiple class="hidden" @change="onDocFilePick" />
        </div>
        <div class="flex h-[calc(100vh-150px)]">
          <div v-if="docMode !== 'preview'" class="relative" :class="docMode === 'split' ? 'w-1/2 border-r border-subtle' : 'w-full'">
            <MentionMenu :m="docMention" />
            <textarea ref="docEl" v-model="docDraft" placeholder="Write… (markdown, @mention, #ref, paste images/files)"
              class="h-full w-full resize-none bg-transparent p-4 text-sm leading-relaxed outline-none"
              @paste="onDocPaste" @input="onDocInput(); docMention.detect()" @keydown="onDocKeydown" />
          </div>
          <div v-if="docMode !== 'edit'" :class="docMode === 'split' ? 'w-1/2' : 'w-full'" class="prose-chat overflow-y-auto p-4 text-sm leading-relaxed" v-html="renderMarkdown(docDraft) || '<span class=&quot;text-tertiary&quot;>// nothing to preview</span>'" />
        </div>
      </template>

      <template v-else-if="selectedTeam">
        <div class="sticky top-0 z-10 border-b border-subtle bg-surface">
          <div class="flex items-center gap-1.5 px-4 py-2.5 text-sm">
            <!-- breadcrumbs -->
            <span class="text-tertiary">{{ store.activeWorkspace?.name }}</span>
            <span class="text-tertiary">/</span>
            <button class="font-medium hover:text-secondary" @click="selectTeamIssues(selectedTeam)">{{ selectedTeam.name }}</button>
            <template v-if="selectedCycleId">
              <span class="text-tertiary">/</span>
              <span>Cycle {{ store.cycles[selectedCycleId]?.number }}</span>
            </template>
            <div class="ml-auto flex items-center gap-1">
              <USelectMenu v-if="store.layout === 'list'" v-model="store.groupBy" :options="labelOpts(['status', 'priority', 'assignee', 'milestone'])" value-attribute="value" option-attribute="label" size="xs" class="w-32" />
              <UButton size="xs" color="gray" variant="ghost" icon="i-heroicons-adjustments-horizontal" title="Manage statuses" @click="showStates = true" />
              <UButton size="xs" :color="showFilters || store.activeFilter ? 'primary' : 'gray'" variant="ghost" icon="i-heroicons-funnel" @click="showFilters = !showFilters" />
              <span class="mx-1 h-4 w-px bg-subtle" />
              <UButton size="xs" square icon="i-heroicons-list-bullet" :color="store.layout==='list' ? 'primary' : 'gray'" variant="ghost" @click="store.layout='list'" />
              <UButton size="xs" square icon="i-heroicons-view-columns" :color="store.layout==='board' ? 'primary' : 'gray'" variant="ghost" @click="store.layout='board'" />
              <UButton size="xs" square icon="i-heroicons-calendar-days" :color="store.layout==='calendar' ? 'primary' : 'gray'" variant="ghost" @click="store.layout='calendar'" />
            </div>
          </div>
          <!-- FilterBar (скрыт по умолчанию) -->
          <div v-if="showFilters" class="flex flex-wrap items-center gap-2 px-4 pb-2.5">
            <UInput v-model="search" placeholder="Search…" icon="i-heroicons-magnifying-glass" size="xs" class="w-48" />
            <UIcon name="i-heroicons-flag" class="text-tertiary" />
            <button v-for="(p, idx) in PRIO" :key="idx"
              class="rounded-md border px-2 py-1 text-xs"
              :class="isPrio(idx) ? 'border-subtle bg-active text-white' : 'border-subtle bg-elevated text-tertiary hover:bg-hover'"
              @click="togglePrio(idx)">{{ p }}</button>
            <button v-if="store.activeFilter" class="px-1 text-xs text-tertiary hover:text-secondary" @click="store.clearFilter()">clear</button>
            <template v-if="store.teamLabels(selectedTeamId).length">
              <span class="mx-1 text-subtle">|</span>
              <button v-for="l in store.teamLabels(selectedTeamId)" :key="l.id"
                class="flex items-center gap-1 rounded-md border px-2 py-1 text-xs"
                :class="store.labelFilter.includes(l.id) ? 'border-subtle bg-active text-white' : 'border-subtle bg-elevated text-tertiary hover:bg-hover'"
                @click="store.labelFilter.includes(l.id) ? store.labelFilter = store.labelFilter.filter((x) => x !== l.id) : store.labelFilter.push(l.id)">
                <span :style="{ color: l.colorToken }">#{{ l.name }}</span>
              </button>
            </template>
            <span class="mx-1 text-subtle">|</span>
            <button v-for="v in store.teamViews(selectedTeamId)" :key="v.id"
              class="flex items-center gap-1 rounded-md border border-subtle bg-elevated px-2 py-1 text-xs text-tertiary hover:bg-hover" @click="openView(v.id)">
              <UIcon name="i-heroicons-star" />{{ v.name }}
            </button>
            <button class="ml-auto flex items-center gap-1 rounded-md border border-subtle bg-elevated px-2 py-1 text-xs text-tertiary hover:bg-hover" @click="saveView">
              <UIcon name="i-heroicons-bookmark" />Save view
            </button>
          </div>
        </div>

        <!-- BOARD -->
        <div v-if="store.layout==='board'" class="flex h-[calc(100%-56px)] gap-4 overflow-x-auto px-4 py-3">
          <div v-for="st in store.teamStates(selectedTeamId)" :key="st.id"
            class="group/col flex w-72 shrink-0 flex-col rounded-xl border border-subtle bg-base p-2"
            @dragover.prevent @drop="onDropOnColumn(st.id)">
            <!-- column header -->
            <div class="mb-2 flex items-center gap-2 px-1.5 py-1 text-sm">
              <span class="inline-block shrink-0 w-4 text-center leading-none" :style="{ color: statusColor(st.category) }">{{ statusGlyph(st.category) }}</span>
              <span class="font-medium">{{ st.name }}</span>
              <span class="text-tertiary">{{ store.issuesFor(selectedTeamId, st.id).length }}</span>
              <UButton v-if="store.canWrite" class="ml-auto opacity-0 group-hover/col:opacity-100" icon="i-heroicons-plus" color="gray" variant="ghost" size="2xs" square @click="openCreate('issue')" />
            </div>
            <div class="flex-1 space-y-2 overflow-y-auto">
              <div v-for="i in store.issuesFor(selectedTeamId, st.id)" :key="i.id"
                draggable="true" @dragstart="draggedId = i.id"
                @dragover.prevent @drop.stop="onDropOnCard(st.id, i.id)"
                @click="openIssue(i.id)"
                tabindex="0" role="button"
                :aria-label="`${i.identifier} ${i.title}. Alt+arrows to move`"
                @keydown="boardKey($event, st.id, i.id)"
                class="cursor-pointer rounded-lg border border-subtle bg-elevated p-3 hover:bg-hover focus:outline-none focus:ring-2 focus:ring-accent"
                :class="{ '!border-accent': selectedIssueId === i.id }">
                <div class="mb-1.5 text-xs text-accent2">{{ i.identifier }}</div>
                <div class="line-clamp-3 text-sm font-medium leading-snug">{{ i.title }}</div>
                <div v-if="store.labelsForIssue(i.id).length" class="mt-1.5 flex flex-wrap gap-1">
                  <span v-for="l in store.labelsForIssue(i.id)" :key="l.id" class="flex items-center gap-1 rounded border border-subtle px-1.5 py-0.5 text-xs text-tertiary">
                    <span :style="{ color: l.colorToken }">#{{ l.name }}</span>
                  </span>
                </div>
                <div class="mt-2.5 flex items-center gap-2 text-xs text-tertiary">
                  <span v-if="i.priority" class="inline-block shrink-0 w-4 text-center leading-none" :style="{ color: prioColor(i.priority) }">{{ prioGlyph(i.priority) }}</span>
                  <span v-if="store.childProgress(i.id).total" class="flex items-center gap-1"><UIcon name="i-heroicons-queue-list" /> {{ store.childProgress(i.id).done }}/{{ store.childProgress(i.id).total }}</span>
                  <span v-if="i.assigneeId" class="ml-auto flex h-5 w-5 items-center justify-center rounded-full bg-surface text-[10px]" :title="store.userName(i.assigneeId)">{{ (store.userName(i.assigneeId) || '?')[0].toUpperCase() }}</span>
                  <span v-if="i.createdAt" :class="i.assigneeId ? '' : 'ml-auto'">{{ new Date(i.createdAt).toLocaleDateString('en', { month: 'short', day: 'numeric' }) }}</span>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- CALENDAR -->
        <div v-else-if="store.layout==='calendar'" class="p-4">
          <div class="mb-2 flex items-center gap-2 text-sm">
            <UButton icon="i-heroicons-chevron-left" color="gray" variant="ghost" size="xs" square @click="monthOffset--" />
            <span class="font-medium">{{ calRangeLabel }}</span>
            <UButton color="gray" variant="ghost" size="xs" square @click="monthOffset++" />
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
              <div v-for="i in store.calendarIssues(selectedTeamId, c.date)" :key="i.id"
                draggable="true" @dragstart="draggedId = i.id" @click="openIssue(i.id)"
                class="mb-0.5 cursor-pointer truncate rounded-sm bg-elevated px-1 py-0.5 hover:bg-hover"
                :title="i.identifier + ' ' + i.title">
                <span class="text-accent2">{{ i.identifier }}</span> {{ i.title }}
              </div>
            </div>
          </div>
        </div>

        <!-- LIST (Linear-стиль: плотные строки, edit в detail; виртуализованный плоский поток) -->
        <template v-else>
        <div ref="listEl" :style="{ height: rowVirtualizer.getTotalSize() + 'px', position: 'relative', width: '100%' }">
          <div v-for="vRow in rowVirtualizer.getVirtualItems()" :key="vRow.key"
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
              @click="openIssue(listRows[vRow.index].i.id)">
              <TermCheck v-if="store.canWrite" :model-value="selectedIds.has(listRows[vRow.index].i.id)" class="shrink-0 transition-opacity" :class="selCount || selectedIds.has(listRows[vRow.index].i.id) ? 'opacity-100' : 'opacity-0 group-hover:opacity-100'" @click.stop @update:model-value="toggleSel(listRows[vRow.index].i.id)" />
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
      </template>

      <div v-else class="p-6 text-sm text-tertiary">
        {{ store.activeWorkspace ? `${store.teamList.length} teams · select a team →` : 'Pick or create a workspace →' }}
      </div>
      </template>
    </main>

    <!-- Cycle insights panel (когда выбран цикл и не открыт issue) -->
    <ResizeHandle v-if="cycleBurndown && !selectedIssue" @down="cyclePanel.onResizeStart" />
    <aside v-if="cycleBurndown && !selectedIssue" :style="{ width: cyclePanel.w.value + 'px' }" class="shrink-0 overflow-y-auto rounded-xl border border-subtle bg-surface px-4 py-3 text-sm">
      <!-- header: Current pill + range -->
      <div class="mb-3 flex items-center gap-2 text-sm">
        <span class="rounded-md px-2 py-0.5 text-xs font-medium"
          :class="cycleStatus === 'Current' ? 'bg-accent/20 text-accent' : 'bg-elevated text-secondary'">{{ cycleStatus }}</span>
        <span class="text-tertiary">{{ cycleRange }}</span>
        <UButton class="ml-auto" size="2xs" color="gray" variant="ghost" square icon="i-heroicons-x-mark" @click="closeCycle" />
      </div>
      <!-- title -->
      <div class="mb-4 flex items-center gap-2">
        <UIcon name="i-heroicons-play-circle" class="text-accent" />
        <span class="text-base font-semibold text-white">{{ store.cycles[selectedCycleId]?.name || `Cycle ${store.cycles[selectedCycleId]?.number}` }}</span>
        <UButton class="ml-auto" icon="i-heroicons-pencil-square" color="gray" variant="ghost" size="xs" square title="Edit cycle" @click="startCycleEdit" />
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
        <span class="text-tertiary">Velocity <span class="text-secondary">{{ store.teamVelocity(store.cycles[selectedCycleId]?.teamId) }}</span> pts/cycle</span>
      </div>

      <div class="my-4 h-px bg-subtle" />

      <!-- burndown chart (реальные ежедневные слепки, River) -->
      <div class="mb-1 text-xs font-medium uppercase tracking-wider text-tertiary"><span class="text-ok"># </span>Burndown</div>
      <Burndown :cycle-id="selectedCycleId" />

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
      <p class="mt-1 px-2 text-[11px] text-tertiary/70">Click a row to filter the list · click again to clear.</p>
    </aside>

    <!-- Issue detail panel (M4) -->
    <ResizeHandle v-if="selectedIssue" @down="detailPanel.onResizeStart" />
    <aside v-if="selectedIssue" :style="{ width: detailPanel.w.value + 'px' }" class="flex shrink-0 flex-col overflow-y-auto rounded-xl border border-subtle bg-surface text-sm">
      <!-- top bar -->
      <div class="flex items-center gap-2 border-b border-subtle px-4 py-2.5">
        <span class="inline-block shrink-0 w-4 text-center leading-none" :style="{ color: statusColor(store.states[selectedIssue.stateId]?.category) }">{{ statusGlyph(store.states[selectedIssue.stateId]?.category) }}</span>
        <span class="text-xs text-accent2">{{ selectedIssue.identifier }}</span>
        <UButton class="ml-auto" :icon="linkCopied ? 'i-heroicons-check' : 'i-heroicons-link'" color="gray" variant="ghost" size="xs" square :title="linkCopied ? 'Copied!' : 'Copy link'" @click="copyLink" />
        <UDropdown :items="issueActions" :popper="{ placement: 'bottom-end' }">
          <UButton icon="i-heroicons-ellipsis-horizontal" color="gray" variant="ghost" size="xs" square />
        </UDropdown>
        <UButton icon="i-heroicons-x-mark" color="gray" variant="ghost" size="xs" square @click="selectedIssueId = ''" />
      </div>

      <div class="px-4 py-4">
        <!-- title -->
        <UInput :model-value="selectedIssue.title" variant="none" size="xl" class="mb-3 w-full font-semibold" :padded="false"
          :ui="{ base: 'text-white' }" :disabled="!store.canWrite"
          @change="(v:any) => store.setIssueTitle(selectedIssue.id, v)" />

        <!-- description -->
        <div class="mb-6">
          <div v-if="descEditing" class="rounded-lg border border-subtle bg-elevated">
            <div class="flex items-center gap-0.5 border-b border-subtle px-1.5 py-1">
              <UButton size="2xs" square color="gray" variant="ghost" class="font-bold" title="Bold" @click="descWrap('**')">B</UButton>
              <UButton size="2xs" square color="gray" variant="ghost" class="italic" title="Italic" @click="descWrap('_')">I</UButton>
              <UButton size="2xs" square color="gray" variant="ghost" class="font-bold" title="Heading" @click="descWrap('\n# ', '')">H</UButton>
              <UButton size="2xs" square color="gray" variant="ghost" icon="i-heroicons-list-bullet" title="List" @click="descWrap('\n- ', '')" />
              <UButton size="2xs" square color="gray" variant="ghost" icon="i-heroicons-code-bracket" title="Code" @click="descWrap('`')" />
              <UButton size="2xs" square color="gray" variant="ghost" icon="i-heroicons-link" title="Link" @click="descWrap('[', '](url)')" />
              <UButton size="2xs" square color="gray" variant="ghost" icon="i-heroicons-paper-clip" title="Attach file/image" @click="descFileEl?.click()" />
              <input ref="descFileEl" type="file" multiple class="hidden" @change="onDescFilePick" />
              <UButton class="ml-auto" size="2xs" color="gray" variant="ghost" title="Save (⌘↵)" @click="saveDesc">[ done ]</UButton>
            </div>
            <div class="relative">
              <MentionMenu :m="descMention" />
              <textarea ref="descEl" v-model="descDraft" rows="5"
                class="block w-full resize-y bg-transparent px-3 py-2 text-sm leading-relaxed text-secondary outline-none placeholder:text-tertiary"
                placeholder="Add description… (markdown, @mention, #ref)" @paste="onAttachPaste($event, 'issue', selectedIssue.id)" @input="descMention.detect()" @keydown="onDescKeydown"></textarea>
            </div>
          </div>
          <template v-else-if="selectedIssue.description?.md">
            <div
              class="prose-invert -mx-2 cursor-text rounded-md px-2 py-1.5 text-sm leading-relaxed text-secondary hover:bg-elevated [&_a]:text-accent [&_code]:rounded [&_code]:bg-elevated [&_code]:px-1 [&_h1]:mb-1 [&_h1]:mt-2 [&_h1]:text-base [&_h1]:font-semibold [&_h1]:text-white [&_h2]:mb-1 [&_h2]:mt-2 [&_h2]:font-semibold [&_h2]:text-white [&_li]:ml-4 [&_li]:list-disc [&_p]:mb-1.5 [&_strong]:text-white"
              @click="store.canWrite && startDescEdit()" v-html="renderMarkdown(selectedIssue.description.md)" />
            <!-- AI-резюме для длинных описаний -->
            <div v-if="(selectedIssue.description.md || '').length > 280" class="mt-1.5">
              <button v-if="!descSummary && !descSummaryLoading" class="text-xs text-tertiary hover:text-accent2" @click="summarizeDesc">≈ summarize</button>
              <span v-else-if="descSummaryLoading" class="text-xs text-tertiary">≈ summarizing…</span>
              <div v-else class="rounded-md border border-subtle bg-elevated px-3 py-2 text-xs leading-relaxed text-secondary">
                <span class="text-accent2">≈ TL;DR </span>{{ descSummary }}
              </div>
            </div>
          </template>
          <button v-else-if="store.canWrite" class="-mx-2 flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-sm text-tertiary hover:bg-elevated" @click="startDescEdit">
            <UIcon name="i-heroicons-bars-3-bottom-left" /> Add description…
          </button>
        </div>

        <!-- Properties (guest = read-only: inert отключает все контролы, значения видны) -->
        <div class="mb-2 text-xs font-medium uppercase tracking-wider text-tertiary"><span class="text-ok"># </span>Properties</div>
        <div class="mb-6 space-y-1" :inert="!store.canWrite" :class="{ 'opacity-70': !store.canWrite }">
          <div class="flex min-h-8 items-center gap-3">
            <span class="flex w-24 shrink-0 items-center gap-2 text-tertiary"><span class="inline-block shrink-0 w-4 text-center leading-none" :style="{ color: statusColor(store.states[selectedIssue.stateId]?.category) }">{{ statusGlyph(store.states[selectedIssue.stateId]?.category) }}</span> Status</span>
            <USelectMenu class="flex-1 rounded-md transition-colors hover:bg-elevated" variant="none" :ui="propSelUi" :model-value="selectedIssue.stateId" value-attribute="value" option-attribute="label"
              :options="store.teamStates(selectedIssue.teamId).map((s2) => ({ label: s2.name, value: s2.id }))"
              @update:model-value="store.setIssueState(selectedIssue.id, $event)" />
          </div>
          <div class="flex min-h-8 items-center gap-3">
            <span class="flex w-24 shrink-0 items-center gap-2 text-tertiary"><span class="inline-block shrink-0 w-4 text-center leading-none" :style="{ color: prioColor(selectedIssue.priority) }">{{ prioGlyph(selectedIssue.priority) }}</span> Priority</span>
            <USelectMenu class="flex-1 rounded-md transition-colors hover:bg-elevated" variant="none" :ui="propSelUi" :model-value="selectedIssue.priority" value-attribute="value" option-attribute="label"
              :options="PRIO.map((p, idx) => ({ label: p, value: idx }))"
              @update:model-value="store.setIssuePriority(selectedIssue.id, Number($event))" />
          </div>
          <div class="flex min-h-8 items-center gap-3">
            <span class="flex w-24 shrink-0 items-center gap-2 text-tertiary"><UIcon name="i-heroicons-user-circle" /> Assignee</span>
            <USelectMenu class="flex-1 rounded-md transition-colors hover:bg-elevated" variant="none" :ui="propSelUi" placeholder="unassigned" :model-value="selectedIssue.assigneeId || undefined" value-attribute="value" option-attribute="label"
              :options="[{ label: 'unassigned', value: '' }, ...store.wsUsers.map((u) => ({ label: u.name || u.email, value: u.id })), ...(store.users[store.currentUserId] ? [{ label: (store.users[store.currentUserId].name || store.users[store.currentUserId].email) + ' (me)', value: store.currentUserId }] : [])]"
              @update:model-value="store.setIssueAssignee(selectedIssue.id, $event)" />
          </div>
          <div class="flex min-h-8 items-center gap-3">
            <span class="flex w-24 shrink-0 items-center gap-2 text-tertiary"><UIcon name="i-heroicons-arrow-path-rounded-square" /> Cycle</span>
            <USelectMenu class="flex-1 rounded-md transition-colors hover:bg-elevated" variant="none" :ui="propSelUi" placeholder="no cycle" :model-value="selectedIssue.cycleId || undefined" value-attribute="value" option-attribute="label"
              :options="[{ label: 'no cycle', value: '' }, ...store.teamCycles(selectedIssue.teamId).map((c) => ({ label: `Cycle ${c.number} (${c.status})`, value: c.id }))]"
              @update:model-value="store.setIssueCycle(selectedIssue.id, $event)" />
          </div>
          <div class="flex min-h-8 items-center gap-3">
            <span class="flex w-24 shrink-0 items-center gap-2 text-tertiary"><UIcon name="i-heroicons-flag" /> Milestone</span>
            <USelectMenu class="flex-1 rounded-md transition-colors hover:bg-elevated" variant="none" :ui="propSelUi" placeholder="no milestone" :model-value="selectedIssue.milestoneId || undefined" value-attribute="value" option-attribute="label"
              :options="[{ label: 'no milestone', value: '' }, ...store.teamMilestones(selectedIssue.teamId).map((m) => ({ label: m.name, value: m.id }))]"
              @update:model-value="store.setIssueMilestone(selectedIssue.id, $event)" />
          </div>
          <div class="flex min-h-8 items-center gap-3">
            <span class="flex w-24 shrink-0 items-center gap-2 text-tertiary"><UIcon name="i-heroicons-calendar" /> Due date</span>
            <DatePicker class="flex-1" placeholder="No due date" :model-value="selectedIssue.scheduledDate ? String(selectedIssue.scheduledDate).slice(0,10) : ''"
              @update:model-value="(v:any) => store.setIssueSchedule(selectedIssue.id, v)" />
          </div>
          <div class="flex min-h-8 items-center gap-3">
            <span class="flex w-24 shrink-0 items-center gap-2 text-tertiary"><UIcon name="i-heroicons-scale" /> Estimate</span>
            <UInput class="flex-1 rounded-md transition-colors hover:bg-elevated" variant="none" :ui="propSelUi" type="number" min="0" :model-value="selectedIssue.estimate ?? ''" placeholder="no estimate"
              @change="(v:any) => store.setIssueEstimate(selectedIssue.id, Number(v))" />
          </div>
          <!-- Time tracking -->
          <div class="flex min-h-8 items-center gap-3">
            <span class="flex w-24 shrink-0 items-center gap-2 text-tertiary"><UIcon name="i-heroicons-clock" /> Time</span>
            <span class="flex-1 text-sm" :class="store.issueTrackedMin(selectedIssue.id) ? 'text-white' : 'text-tertiary'">{{ store.issueTrackedMin(selectedIssue.id) ? fmtMin(store.issueTrackedMin(selectedIssue.id)) : 'Not tracked' }}</span>
            <UButton v-if="store.runningTimer?.issueId === selectedIssue.id" color="red" size="2xs" @click="wrap(() => store.stopTimer())">Stop</UButton>
            <UButton v-else color="gray" size="2xs" @click="wrap(() => store.startTimer(selectedIssue.id))">Start</UButton>
            <UButton color="gray" variant="ghost" size="2xs" square icon="i-heroicons-plus" title="Add time manually" @click="openTimeEntry(selectedIssue.id)" />
          </div>
          <!-- записи времени -->
          <div v-for="te in store.timeEntriesForIssue(selectedIssue.id)" :key="te.id" class="group ml-24 flex items-center gap-2 pl-3 text-xs text-tertiary">
            <UIcon :name="te.endedAt ? 'i-heroicons-check-circle' : 'i-heroicons-play-circle'" :class="te.endedAt ? '' : 'text-ok'" class="shrink-0" />
            <span class="tabular-nums">{{ new Date(te.startedAt).toLocaleDateString([], { month: 'short', day: 'numeric' }) }} {{ new Date(te.startedAt).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }) }}</span>
            <span class="flex-1 truncate">{{ te.description }}</span>
            <span v-if="te.endedAt" class="tabular-nums">{{ fmtMin(Math.max(0, Math.round((new Date(te.endedAt).getTime() - new Date(te.startedAt).getTime()) / 60000))) }}</span>
            <span v-else class="text-ok">running</span>
            <UButton class="opacity-0 group-hover:opacity-100" icon="i-heroicons-x-mark" color="gray" variant="ghost" size="2xs" square @click="wrap(() => store.deleteTimeEntry(te.id))" />
          </div>
          <!-- Custom fields -->
          <div v-for="f in store.fieldsForTeam(selectedIssue.teamId)" :key="f.id" class="flex min-h-8 items-center gap-3">
            <span class="w-24 shrink-0 truncate text-tertiary">{{ f.name }}</span>
            <div v-if="f.type==='checkbox'" class="flex-1 pl-2"><TermCheck :model-value="!!store.cfValue(selectedIssue.id, f.id)"
              @update:model-value="store.setCustomFieldValue(selectedIssue.id, f.id, $event)" /></div>
            <USelectMenu v-else-if="f.type==='select'" class="flex-1 rounded-md transition-colors hover:bg-elevated" variant="none" :ui="propSelUi" placeholder="—" value-attribute="value" option-attribute="label"
              :model-value="store.cfValue(selectedIssue.id, f.id) || undefined" :options="[{ label: '—', value: '' }, ...(f.config?.options || []).map((o) => ({ label: o, value: o }))]"
              @update:model-value="store.setCustomFieldValue(selectedIssue.id, f.id, $event)" />
            <div v-else-if="f.type==='multiselect'" class="flex flex-1 flex-wrap gap-1 py-1">
              <button v-for="o in (f.config?.options || [])" :key="o"
                class="rounded-md px-2 py-0.5 text-xs"
                :class="(store.cfValue(selectedIssue.id, f.id) || []).includes(o) ? 'bg-active text-white' : 'bg-elevated text-tertiary hover:bg-hover'"
                @click="cfMultiToggle(f.id, o)">{{ o }}</button>
            </div>
            <USelectMenu v-else-if="f.type==='people'" class="flex-1 rounded-md transition-colors hover:bg-elevated" variant="none" :ui="propSelUi" placeholder="—"
              :model-value="store.cfValue(selectedIssue.id, f.id) || undefined" value-attribute="value" option-attribute="label"
              :options="[{ label: '—', value: '' }, ...store.wsUsers.map((u) => ({ label: u.name || u.email, value: u.id }))]"
              @update:model-value="store.setCustomFieldValue(selectedIssue.id, f.id, $event)" />
            <USelectMenu v-else-if="f.type==='relationship'" class="flex-1 rounded-md transition-colors hover:bg-elevated" variant="none" :ui="propSelUi" placeholder="—"
              :model-value="store.cfValue(selectedIssue.id, f.id) || undefined" value-attribute="value" option-attribute="label"
              :options="[{ label: '—', value: '' }, ...store.issuesForTeamAll(selectedIssue.teamId).map((i) => ({ label: `${i.identifier} ${i.title}`, value: i.id }))]"
              @update:model-value="store.setCustomFieldValue(selectedIssue.id, f.id, $event)" />
            <UInput v-else class="flex-1 rounded-md transition-colors hover:bg-elevated" variant="none" :ui="propSelUi" :type="f.type==='number' ? 'number' : f.type==='date' ? 'date' : 'text'"
              :model-value="store.cfValue(selectedIssue.id, f.id) ?? ''" placeholder="—"
              @change="(v:any) => store.setCustomFieldValue(selectedIssue.id, f.id, f.type==='number' ? Number(v) : v)" />
          </div>
        </div>

        <!-- Labels -->
        <div class="mb-5">
          <div class="mb-1.5 text-xs font-medium uppercase tracking-wider text-tertiary"><span class="text-ok"># </span>Labels</div>
          <div class="flex flex-wrap items-center gap-1.5" :inert="!store.canWrite" :class="{ 'opacity-70': !store.canWrite }">
            <span v-for="l in store.labelsForIssue(selectedIssue.id)" :key="l.id" class="group/lbl flex items-center gap-1 rounded-full border border-subtle px-2 py-0.5 text-xs">
              <span :style="{ color: l.colorToken }">#{{ l.name }}</span>
              <UIcon name="i-heroicons-x-mark" class="cursor-pointer text-tertiary hover:text-danger" @click="store.removeLabel(selectedIssue.id, l.id)" />
            </span>
            <button class="flex items-center gap-1 rounded-full border border-dashed border-subtle px-2 py-0.5 text-xs text-tertiary hover:border-strong hover:text-secondary" @click="showLabelModal = true">
              <UIcon name="i-heroicons-plus" /> Add label
            </button>
          </div>
        </div>

        <!-- Sub-issues -->
        <div class="mb-5">
          <div class="mb-1.5 flex items-center gap-2 text-xs font-medium uppercase tracking-wider text-tertiary">
            <span class="text-ok"># </span>Sub-issues
            <span v-if="store.childProgress(selectedIssue.id).total" class="rounded bg-elevated px-1.5 text-tertiary">{{ store.childProgress(selectedIssue.id).done }}/{{ store.childProgress(selectedIssue.id).total }}</span>
          </div>
          <div v-for="c in store.subIssuesOf(selectedIssue.id)" :key="c.id"
            class="flex cursor-pointer items-center gap-2 rounded-md px-2 py-1.5 hover:bg-hover" @click="openIssue(c.id)">
            <span class="inline-block shrink-0 w-4 text-center leading-none" :style="{ color: statusColor(store.states[c.stateId]?.category) }">{{ statusGlyph(store.states[c.stateId]?.category) }}</span>
            <span class="text-xs text-accent2">{{ c.identifier }}</span>
            <span class="truncate" :class="{ 'text-tertiary line-through': c.completedAt }">{{ c.title }}</span>
          </div>
          <UInput v-if="store.canWrite" v-model="newSub" placeholder="+ Add sub-issue" size="xs" variant="none" class="w-full" @keyup.enter="addSub" />
        </div>

        <!-- Relations -->
        <div class="mb-4">
          <div class="mb-1.5 text-xs font-medium uppercase tracking-wider text-tertiary"><span class="text-ok"># </span>Relations</div>
          <div v-for="r in store.relationsForIssue(selectedIssue.id)" :key="r.id"
            class="group flex items-center gap-2 rounded-md px-2 py-1.5 hover:bg-hover">
            <span class="text-tertiary">{{ r.label }}</span>
            <span class="cursor-pointer hover:text-accent" @click="openIssue(r.otherId)">{{ store.issueById(r.otherId)?.identifier || '—' }}</span>
            <UButton v-if="store.canWrite" class="ml-auto opacity-0 group-hover:opacity-100" icon="i-heroicons-x-mark" color="gray" variant="ghost" size="2xs" square @click="store.removeRelation(r.id)" />
          </div>
          <button v-if="store.canWrite" class="mt-1 flex items-center gap-1 rounded-md px-2 py-1.5 text-sm text-tertiary hover:bg-hover hover:text-secondary" @click="showRelModal = true">
            <UIcon name="i-heroicons-plus" /> Add relation
          </button>
        </div>

        <!-- Attachments -->
        <div class="mb-5">
          <div class="mb-1.5 text-xs font-medium uppercase tracking-wider text-tertiary"><span class="text-ok"># </span>Attachments</div>
          <Attachments entity-type="issue" :entity-id="selectedIssue.id" />
        </div>

        <!-- AI: suggestions + similar (Фаза 1–2) -->
        <div v-if="simIssues.length || suggestLabels.length || suggestAssignee || suggestPriority !== null || suggestEstimate !== null || slipWarn" class="mb-4">
          <div class="mb-2 flex items-center gap-2 text-xs font-medium uppercase tracking-wider text-tertiary">
            <span class="text-accent2">≈ </span>AI
          </div>
          <div v-if="slipWarn" class="mb-2 flex items-center gap-1 text-xs text-danger">
            ⚠ likely to miss its due date
          </div>
          <!-- подсказки в один клик -->
          <div v-if="suggestLabels.length || suggestAssignee || suggestPriority !== null || suggestEstimate !== null" class="mb-2 flex flex-wrap gap-1.5">
            <button v-if="suggestPriority !== null" class="flex items-center gap-1 rounded border border-subtle px-2 py-0.5 text-xs text-secondary hover:border-warn hover:text-warn"
              @click="store.setIssuePriority(selectedIssue.id, suggestPriority); store.aiFeedback('priority','applied',selectedIssue.id,String(suggestPriority))">
              ⚑ priority {{ PRI_LABELS[suggestPriority] }}
            </button>
            <button v-if="suggestEstimate !== null" class="flex items-center gap-1 rounded border border-subtle px-2 py-0.5 text-xs text-secondary hover:border-accent hover:text-accent"
              @click="store.setIssueEstimate(selectedIssue.id, suggestEstimate); store.aiFeedback('estimate','applied',selectedIssue.id,String(suggestEstimate))">
              ≈ est {{ suggestEstimate }} pts
            </button>
            <button v-if="suggestAssignee" class="flex items-center gap-1 rounded border border-subtle px-2 py-0.5 text-xs text-secondary hover:border-accent hover:text-accent"
              @click="store.setIssueAssignee(selectedIssue.id, suggestAssignee); store.aiFeedback('assignee','applied',selectedIssue.id,suggestAssignee)">
              ＋ assign {{ store.userName(suggestAssignee) }}
            </button>
            <button v-for="lid in suggestLabels" :key="lid" class="flex items-center gap-1 rounded border border-subtle px-2 py-0.5 text-xs hover:border-accent"
              :style="{ color: store.labels[lid]?.colorToken ? undefined : undefined }" @click="store.addLabel(selectedIssue.id, lid); store.aiFeedback('label','applied',selectedIssue.id,lid)">
              ＋ {{ store.labels[lid]?.name }}
            </button>
          </div>
          <!-- похожие / дубли -->
          <button v-for="s in simIssues" :key="s.id" class="flex w-full items-center gap-2 rounded-sm px-2 py-1 text-left text-sm hover:bg-hover" @click="openIssue(s.id)">
            <span class="shrink-0 text-xs text-accent2">{{ s.identifier }}</span>
            <span class="min-w-0 flex-1 truncate text-secondary">{{ s.title }}</span>
          </button>
        </div>

        <!-- Comments -->
        <div class="mb-4">
          <div class="mb-2 flex items-center gap-2 text-xs font-medium uppercase tracking-wider text-tertiary">
            <span class="text-ok"># </span>Comments <span class="text-tertiary">{{ store.issueComments(selectedIssue.id).length }}</span>
          </div>
          <div v-for="c in store.issueComments(selectedIssue.id)" :key="c.id" class="group mb-2 flex gap-2">
            <Avatar class="mt-0.5" :name="store.userName(c.authorId)" :avatar-key="store.users[c.authorId]?.avatarKey" :size="28" />
            <div class="min-w-0 flex-1">
              <div class="flex items-baseline gap-2">
                <span class="text-sm font-medium">{{ store.userName(c.authorId) }}</span>
                <span class="text-xs text-tertiary">{{ notifTime(c.createdAt) }}</span>
                <span v-if="c.editedAt" class="text-xs text-tertiary/60">(edited)</span>
                <div class="relative ml-auto" :class="c.authorId === store.currentUserId ? '' : 'ml-auto'">
                  <UButton v-if="!c.deletedAt" class="opacity-0 group-hover:opacity-100" icon="i-heroicons-face-smile" color="gray" variant="ghost" size="2xs" square @click="cReactPickerId = cReactPickerId === c.id ? '' : c.id" />
                  <div v-if="cReactPickerId === c.id" class="absolute right-0 z-20 mt-1 flex gap-0.5 rounded-md border border-subtle bg-elevated p-1 shadow-lg">
                    <button v-for="e in EMOJIS" :key="e" class="rounded p-1 text-base hover:bg-hover" @click="commentReact(c, e)">{{ e }}</button>
                  </div>
                </div>
                <template v-if="c.authorId === store.currentUserId && !c.deletedAt && editCommentId !== c.id">
                  <UButton class="opacity-0 group-hover:opacity-100" icon="i-heroicons-pencil" color="gray" variant="ghost" size="2xs" square @click="startEditComment(c)" />
                  <UButton class="opacity-0 group-hover:opacity-100" icon="i-heroicons-trash" color="gray" variant="ghost" size="2xs" square @click="store.deleteComment(c.id, selectedIssue.id)" />
                </template>
              </div>
              <div v-if="editCommentId === c.id" class="mt-1">
                <textarea v-model="editCommentDraft" rows="2" class="w-full rounded-md border border-subtle bg-elevated px-2 py-1 text-sm outline-none" @keydown.enter.exact.prevent="saveComment" @keydown.esc="editCommentId = ''" />
                <div class="mt-1 flex gap-2"><UButton size="2xs" @click="saveComment">[ save ]</UButton><UButton size="2xs" color="gray" variant="ghost" @click="editCommentId = ''">[ cancel ]</UButton></div>
              </div>
              <div v-else-if="c.deletedAt" class="text-sm italic text-tertiary">(deleted)</div>
              <div v-else class="prose-chat text-sm leading-relaxed" v-html="renderMarkdown(c.body)" />
              <Attachments v-if="store.attachmentsFor('comment', c.id).length" class="mt-1.5" entity-type="comment" :entity-id="c.id" readonly />
              <div v-if="reactionGroups(c).length" class="mt-1 flex flex-wrap gap-1">
                <button v-for="g in reactionGroups(c)" :key="g.emoji"
                  class="rounded border px-1.5 py-0.5 text-xs"
                  :class="g.mine ? 'border-accent bg-accent/15 text-accent' : 'border-subtle text-secondary hover:bg-hover'"
                  @click="commentReact(c, g.emoji)">{{ g.emoji }} {{ g.count }}</button>
              </div>
            </div>
          </div>
          <PendingAttachments v-if="store.canWrite && commentPending.length" class="mt-2" :files="commentPending" @remove="(i) => commentPending = commentPending.filter((_, j) => j !== i)" />
          <div v-if="store.canWrite" class="mt-2 flex items-end gap-2">
            <div class="relative flex flex-1 gap-2 rounded-md border border-subtle bg-elevated px-2.5 py-1.5">
              <MentionMenu :m="commentMention" />
              <span class="select-none pt-px text-ok">&gt;</span>
              <textarea ref="commentEl" v-model="commentDraft" rows="1" placeholder="comment… (markdown, @mention, #ref)"
                class="block w-full resize-none bg-transparent text-sm outline-none placeholder:text-tertiary"
                @paste="onCommentPaste" @input="commentMention.detect()" @keydown="onCommentKeydown" />
            </div>
            <UButton icon="i-heroicons-paper-clip" color="gray" variant="ghost" size="sm" square title="Attach" @click="commentFileEl?.click()" />
            <input ref="commentFileEl" type="file" multiple class="hidden" @change="onCommentFilePick" />
            <UButton icon="i-heroicons-paper-airplane" size="sm" :disabled="!commentDraft.trim() && !commentPending.length" @click="addComment" />
          </div>
        </div>

        <!-- Activity -->
        <div class="mb-4">
          <div class="mb-1.5 text-xs font-medium uppercase tracking-wider text-tertiary"><span class="text-ok"># </span>Activity</div>
          <div v-for="(a, idx) in store.issueActivity(selectedIssue.id)" :key="idx" class="flex items-center gap-2 py-1 text-xs text-tertiary">
            <UIcon :name="a.op === 'create' ? 'i-heroicons-plus-circle' : a.op === 'delete' ? 'i-heroicons-trash' : 'i-heroicons-pencil'" />
            <span class="text-secondary">{{ a.actorId ? store.userName(a.actorId) : 'Someone' }}</span>
            <span>{{ a.op === 'create' ? 'created' : a.op === 'delete' ? 'deleted' : 'updated' }}</span>
            <span class="ml-auto">{{ notifTime(a.at) }}</span>
          </div>
          <div v-if="!store.issueActivity(selectedIssue.id).length" class="text-xs text-tertiary/60">No activity</div>
        </div>

        <p v-if="err" class="text-xs text-danger">{{ err }}</p>
      </div>
    </aside>

    <!-- Thread panel (плавающая) -->
    <SectionsThreadPanel v-if="store.threadParentId && activeSection === 'chats'" />
    </div><!-- /content panel -->
    </div><!-- /flex flex-1 -->

    <!-- Bulk-бар (мультивыбор в List) -->
    <div v-if="selCount && selectedTeam && store.canWrite" class="fixed bottom-6 left-1/2 z-[90] flex -translate-x-1/2 items-center gap-2 rounded-xl border border-subtle bg-elevated px-3 py-2 text-sm shadow-xl">
      <span class="px-1 font-medium">{{ selCount }} selected</span>
      <span class="h-4 w-px bg-subtle" />
      <USelectMenu placeholder="Status" size="xs" class="w-28" value-attribute="value" option-attribute="label"
        :options="store.teamStates(selectedTeamId).map((s2) => ({ label: s2.name, value: s2.id }))" @update:model-value="bulkStatus" />
      <USelectMenu placeholder="Assignee" size="xs" class="w-32" value-attribute="value" option-attribute="label"
        :options="[{ label: 'No assignee', value: '' }, ...store.wsUsers.map((u) => ({ label: u.name || u.email, value: u.id }))]" @update:model-value="bulkAssignee" />
      <USelectMenu v-if="store.teamLabels(selectedTeamId).length" placeholder="+ Label" size="xs" class="w-28" value-attribute="value" option-attribute="label"
        :options="store.teamLabels(selectedTeamId).map((l) => ({ label: l.name, value: l.id }))" @update:model-value="bulkLabel" />
      <UButton size="xs" color="gray" variant="ghost" @click="bulkArchive">[ archive ]</UButton>
      <UButton size="xs" color="gray" variant="ghost" icon="i-heroicons-x-mark" square @click="clearSel" />
    </div>

    <!-- Create-модалка (поля под выбранную сущность) -->
    <ModalsCreateModal v-model="createOpen" :type="createType" :default-team="selectedTeamId" @created="onCreated" />

    <!-- Add-widget модалка -->
    <ModalsWidgetModal v-model="showWidget" :dashboard-id="selectedDashboardId" />

    <!-- Calendar event модалка -->
    <ModalsCalendarEventModal v-model="showCalEvent" :form="calForm" @save="saveCalEvent" @delete="deleteCalEvent" />

    <!-- Ручная запись времени -->
    <ModalsTimeEntryModal v-model="showTimeEntry" :form="teForm" @save="saveTimeEntry" />

    <!-- iCal subscribe модалка -->
    <ModalsIcalModal v-model="showIcal" v-model:url="icalUrl" />

    <!-- Add label модалка -->
    <ModalsAddLabelModal v-model="showLabelModal" :issue="selectedIssue" />

    <!-- Add relation модалка -->
    <ModalsAddRelationModal v-model="showRelModal" :issue="selectedIssue" />

    <!-- Statuses модалка -->
    <ModalsWorkflowStatesModal v-model="showStates" :team-id="selectedTeamId" :team-key="selectedTeam?.key" />

    <!-- Workspace Settings (большая модалка с сайдбаром) -->
    <ModalsSettingsModal v-model="showSettings" v-model:tab="settingsTab" :selected-team-id="selectedTeamId"
      :cfg-list="cfgList" :notif-prefs="notifPrefs"
      @new-team="openCreate('team')" @open-states="(id) => { selectedTeamId = id; showStates = true }"
      @delete-team="confirmDeleteTeam" @open-issue="openIssue" @logout="logout"
      @toggle-pref="toggleNotifPref" @toggle-section="toggleSection" @move-section="moveSection" />

    <!-- Members модалка -->
    <ModalsMembersModal v-model="showMembers" v-model:invite-email="inviteEmail" v-model:invite-role="inviteRole"
      :invite-link="inviteLink" @invite="doInvite" @copy-link="copyInviteLink" />

    <!-- Command palette (Cmd/Ctrl+K) -->
    <div v-if="paletteOpen" class="fixed inset-0 z-[100] flex items-start justify-center bg-black/40 pt-32" @click.self="paletteOpen = false">
      <div class="w-[32rem] rounded-md border border-subtle bg-elevated shadow-xl">
        <div class="flex items-center gap-2 border-b border-subtle px-4">
          <span class="text-ok">$</span>
          <input ref="paletteInput" v-model="paletteQ" placeholder="search issues, teams, dashboards, views…"
            class="w-full bg-transparent py-3 text-sm outline-none focus:outline-none focus-visible:outline-none"
            @keydown.down.prevent="paletteMove(1)" @keydown.up.prevent="paletteMove(-1)"
            @keydown.enter.prevent="runResult(paletteResults[paletteIdx])" />
        </div>
        <div class="max-h-80 overflow-y-auto p-1">
          <div v-for="(r, i) in paletteResults" :key="r.type + r.id"
            class="flex cursor-pointer items-center gap-3 rounded-sm px-3 py-2 text-sm"
            :class="i === paletteIdx ? 'bg-active' : 'hover:bg-hover'"
            @mouseenter="paletteIdx = i" @click="runResult(r)">
            <span class="w-16 shrink-0 text-[10px] uppercase tracking-wide text-tertiary">{{ r.type }}</span>
            <span class="min-w-0 flex-1 truncate">{{ r.label }}</span>
            <span v-if="r.semantic" class="shrink-0 text-[10px] text-accent2" title="semantic match">≈ ai</span>
          </div>
          <div v-if="paletteQ && !paletteResults.length" class="px-3 py-2 text-sm text-tertiary">No results</div>
          <div v-if="!paletteQ" class="px-3 py-2 text-xs text-tertiary">Type to search · ⌘K / Ctrl+K</div>
        </div>
      </div>
    </div>

    <!-- активный звонок (плавающая панель, сама прячется) -->
    <MeetingCall />
    <!-- лайтбокс картинок -->
    <ImageLightbox />
  </div>
</template>
