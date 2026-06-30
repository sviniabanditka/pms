<script setup lang="ts">
import { isAuthed, clearAuth } from '~/lib/token'
import { fileViewUrl } from '~/lib/upload'
import { useAppStore } from '~/stores/app'

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
const activeSection = ref<'dashboard' | 'tasks' | 'inbox' | 'chats' | 'meetings' | 'calendar' | 'reports'>('tasks')
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
const err = ref('')

const PRIO = ['None', 'Low', 'Med', 'High', 'Urgent']
const cap = (s: string) => (s ? s.charAt(0).toUpperCase() + s.slice(1).replace(/_/g, ' ') : s)
const labelOpts = (arr: string[]) => arr.map((v) => ({ label: cap(v), value: v }))
// borderless ghost look for issue-detail property controls
const propSelUi = { padding: { sm: 'px-2 py-1' }, base: 'cursor-pointer' }
const { statusColor, statusGlyph, prioColor, prioGlyph } = useColors()
const selectedTeam = computed(() => store.teams[selectedTeamId.value])
const draggedId = ref('')

// --- Issue detail panel (M4) ---
const selectedIssueId = ref('')
const selectedIssue = computed(() => store.issues[selectedIssueId.value])
const newSub = ref('')
const relType = ref('blocks')
const relTarget = ref('')
const REL_TYPES = ['blocks', 'blocked_by', 'related', 'duplicate', 'duplicate_of']
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
function openIssue(id: string) { selectedIssueId.value = id; descEditing.value = false; store.loadComments(id); store.loadActivity(id) }
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
const STATE_CATS = ['backlog', 'unstarted', 'started', 'completed', 'canceled']
const newStateName = ref('')
const newStateCat = ref('unstarted')
const addState = () => wrap(async () => { if (newStateName.value.trim()) { await store.createState(selectedTeamId.value, newStateName.value, newStateCat.value); newStateName.value = '' } })
const delState = (id: string) => wrap(() => store.deleteState(id))
// labels
const newLabelName = ref('')
const createLabelInIssue = () => wrap(async () => {
  if (!newLabelName.value.trim() || !selectedIssue.value) return
  const id = await store.createLabel(selectedIssue.value.teamId, newLabelName.value)
  newLabelName.value = ''
  await store.addLabel(selectedIssue.value.id, id)
})
const addLabelFromModal = (lid: any) => lid && wrap(async () => { await store.addLabel(selectedIssueId.value, lid) })
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
    return Object.values(store.labels).map((l: any, n: number) => ({
      kind: 'label', value: l.id, label: l.name, color: l.colorToken || BREAKDOWN_PALETTE[n % BREAKDOWN_PALETTE.length],
      count: iss.filter((i) => store.labelsForIssue(i.id).some((x: any) => x.id === l.id)).length,
    })).filter((r) => r.count).map((r) => ({ ...r, pct: pct(r.count) }))
  }
  if (insightTab.value === 'milestones') {
    const rows = Object.values(store.milestones).map((m: any, n: number) => ({
      field: 'milestone', value: m.id, label: m.name, color: BREAKDOWN_PALETTE[n % BREAKDOWN_PALETTE.length],
      count: iss.filter((i) => i.milestoneId === m.id).length,
    })).filter((r) => r.count).map((r) => ({ ...r, pct: pct(r.count) }))
    const none = iss.filter((i) => !i.milestoneId).length
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
const addRel = () => wrap(async () => { if (relTarget.value.trim()) { await store.addRelation(selectedIssueId.value, relTarget.value, relType.value); relTarget.value = ''; showRelModal.value = false } })

// --- Dashboards (M6 ч.2) ---
const selectedDashboardId = ref('')
const selectedDashboard = computed(() => store.dashboards[selectedDashboardId.value])
const newDashboard = ref('')
function openDashboard(id: string) { selectedDashboardId.value = id }
const createDash = () => wrap(async () => { if (newDashboard.value.trim()) { const id = await store.createDashboard(newDashboard.value); newDashboard.value = ''; openDashboard(id) } })
// add-widget form
const showWidget = ref(false)
const draggedWidget = ref('')
const wType = ref('note')
const wTitle = ref('')
const wNote = ref('')
const wStatPrio = ref('')
const wViewId = ref('')
const addWidget = () => wrap(async () => {
  let config: any = {}
  if (wType.value === 'note') config = { text: wNote.value }
  else if (wType.value === 'stat') config = { priority: wStatPrio.value }
  else if (wType.value === 'view') config = { viewId: wViewId.value }
  await store.addWidget(selectedDashboardId.value, wType.value, wTitle.value, config)
  wTitle.value = ''; wNote.value = ''
})
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
  // гидратация подняла кэш → показываем сразу (без скелетона), освежаем в фоне.
  if (store.activeWorkspaceId && Object.keys(store.teams).length) store.loaded = true
  if (store.activeWorkspaceId) await store.bootstrap(store.activeWorkspaceId)
  else if (store.myWorkspaces.length) await store.bootstrap(store.myWorkspaces[0].id)
  applyRoute(route.path) // deep-link при загрузке
})

async function wrap(fn: () => Promise<any>) {
  err.value = ''
  try { await fn() } catch (e: any) { err.value = e?.message || String(e) }
}
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
const router = useRouter()
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
    [
      ...(doneId && i.stateId !== doneId ? [{ label: 'Mark complete', icon: 'i-heroicons-check-circle', click: () => store.setIssueState(i.id, doneId) }] : []),
      { label: 'Duplicate', icon: 'i-heroicons-document-duplicate', click: () => duplicateIssue(i) },
    ],
    [
      { label: 'Archive', icon: 'i-heroicons-archive-box', click: () => { store.archiveIssue(i.id); selectedIssueId.value = '' } },
    ],
  ]
})

// --- Team tree expand state ---
const expandTeam = reactive<Record<string, boolean>>({})
const expandCycles = reactive<Record<string, boolean>>({})
const expandDocs = reactive<Record<string, boolean>>({})
const expandViews = reactive<Record<string, boolean>>({})
const collapsedStatus = reactive<Record<string, boolean>>({}) // свёрнутые группы статусов в списке
const showFilters = ref(false)

// --- Глобальное создание через модалку (Linear-стиль) ---
const createOpen = ref(false)
const createType = ref<'issue' | 'doc' | 'cycle' | 'team' | 'workspace'>('issue')
const CREATE_TYPES = [
  { label: 'Issue', value: 'issue' },
  { label: 'Doc', value: 'doc' },
  { label: 'Cycle', value: 'cycle' },
  { label: 'Milestone', value: 'milestone' },
  { label: 'Meeting', value: 'meeting' },
  { label: 'Dashboard', value: 'dashboard' },
  { label: 'Channel', value: 'channel' },
  { label: 'Team', value: 'team' },
  { label: 'Workspace', value: 'workspace' },
]
const cTeam = ref(''); const cTitle = ref('')
const cStart = ref(''); const cEnd = ref(''); const cMsDate = ref(''); const cPrivate = ref(false)
const cKey = ref(''); const cName = ref(''); const cParent = ref(''); const cSlug = ref('')
const cMeetDate = ref(''); const cMeetStart = ref('09:00'); const cMeetEnd = ref('09:30'); const cMeetParts = ref<string[]>([])
function openCreate(type: any = 'issue') {
  createType.value = type
  cTeam.value = selectedTeamId.value || store.teamList[0]?.id || ''
  createOpen.value = true
}
const createTypeLabel = computed(() => CREATE_TYPES.find((c) => c.value === createType.value)?.label || 'Item')
const CREATE_ICON: Record<string, string> = {
  issue: 'i-heroicons-pencil-square', doc: 'i-heroicons-document-text', cycle: 'i-heroicons-arrow-path-rounded-square',
  dashboard: 'i-heroicons-squares-2x2', channel: 'i-heroicons-hashtag', milestone: 'i-heroicons-flag', meeting: 'i-heroicons-video-camera', team: 'i-heroicons-user-group', workspace: 'i-heroicons-building-office-2',
}
const createIcon = computed(() => CREATE_ICON[createType.value] || 'i-heroicons-plus')
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
const submitCreate = () => wrap(async () => {
  const t = createType.value
  if (t === 'issue' && cTeam.value && cTitle.value.trim()) await store.createIssue(cTeam.value, cTitle.value)
  else if (t === 'doc' && cTeam.value && cTitle.value.trim()) { const id = await store.createDoc(cTeam.value, cTitle.value); selectedTeamId.value = cTeam.value; openDoc(id) }
  else if (t === 'dashboard' && cTitle.value.trim()) { const id = await store.createDashboard(cTitle.value); activeSection.value = 'dashboard'; selectedDashboardId.value = id }
  else if (t === 'channel' && cTitle.value.trim()) { const id = await store.createChannel(cTitle.value, cPrivate.value); cPrivate.value = false; activeSection.value = 'chats'; store.loadMessages(id) }
  else if (t === 'cycle' && cTeam.value && cStart.value && cEnd.value) await store.createCycle(cTeam.value, cStart.value, cEnd.value)
  else if (t === 'milestone' && cTeam.value && cTitle.value.trim()) await store.createMilestone(cTeam.value, cTitle.value, cMsDate.value)
  else if (t === 'meeting' && cTitle.value.trim()) {
    const at = cMeetDate.value && cMeetStart.value ? new Date(`${cMeetDate.value}T${cMeetStart.value}:00`).toISOString() : ''
    const end = at && cMeetEnd.value ? new Date(`${cMeetDate.value}T${cMeetEnd.value}:00`).toISOString() : ''
    await store.createMeeting(cTitle.value, at, end, [...cMeetParts.value]); activeSection.value = 'meetings'
  }
  else if (t === 'team' && cKey.value && cName.value) await store.createTeam(cKey.value, cName.value, cParent.value)
  else if (t === 'workspace' && cSlug.value && cName.value) await store.createWorkspace(cSlug.value, cName.value)
  else return
  cTitle.value = cStart.value = cEnd.value = cMsDate.value = cKey.value = cName.value = cSlug.value = cMeetDate.value = ''
  cMeetStart.value = '09:00'; cMeetEnd.value = '09:30'; cMeetParts.value = []
  createOpen.value = false
})
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
// текущее время для «now»-линии (тикает раз в минуту; ponytail: без cleanup — компонент живёт всю сессию)
const nowTick = ref(0)
onMounted(() => { setInterval(() => nowTick.value++, 60000) })
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

// --- Automations (rule builder) ---
const showAuto = ref(false)
const autoForm = reactive({ id: '', name: '', teamId: '', enabled: true, event: 'status_changed', statusTarget: 'cat:completed', actions: [] as any[] })
const AUTO_EVENTS = [
  { value: 'created', label: 'Issue created' },
  { value: 'status_changed', label: 'Status changed' },
  { value: 'assignee_changed', label: 'Assignee changed' },
  { value: 'priority_changed', label: 'Priority changed' },
]
const ACTION_TYPES = [
  { value: 'set_status', label: 'Set status' },
  { value: 'set_assignee', label: 'Set assignee' },
  { value: 'set_priority', label: 'Set priority' },
  { value: 'add_label', label: 'Add label' },
  { value: 'post_message', label: 'Post to channel' },
]
const stateOpts = computed(() => Object.values(store.states).filter((s: any) => !s.deletedAt).map((s: any) => ({ value: s.id, label: `${store.teams[s.teamId]?.key || '?'} · ${s.name}` })))
const statusTargetOpts = computed(() => [
  { value: '', label: 'Any status' },
  { value: 'cat:started', label: 'Category: Started' },
  { value: 'cat:completed', label: 'Category: Completed' },
  { value: 'cat:canceled', label: 'Category: Canceled' },
  ...stateOpts.value.map((s) => ({ value: 'state:' + s.value, label: s.label })),
])
const assigneeOpts = computed(() => [{ value: 'me', label: 'Me (trigger actor)' }, { value: '', label: 'Unassign' }, ...store.wsUsers.map((u: any) => ({ value: u.id, label: u.name || u.email }))])
const labelOptsAuto = computed(() => Object.values(store.labels).filter((l: any) => !l.deletedAt).map((l: any) => ({ value: l.id, label: l.name })))
const channelOptsAuto = computed(() => store.channelList.map((c: any) => ({ value: c.id, label: c.name })))
function defaultAction(type: string) {
  if (type === 'set_status') return { type, stateId: '' }
  if (type === 'set_assignee') return { type, userId: 'me' }
  if (type === 'set_priority') return { type, priority: 2 }
  if (type === 'add_label') return { type, labelId: '' }
  return { type: 'post_message', channelId: '', text: '{issue} updated' }
}
function openAutoCreate() {
  Object.assign(autoForm, { id: '', name: '', teamId: '', enabled: true, event: 'status_changed', statusTarget: 'cat:completed', actions: [defaultAction('set_assignee')] })
  showAuto.value = true
}
function openAutoEdit(a: any) {
  const t = a.trigger || {}
  let st = ''
  if (t.toStateId) st = 'state:' + t.toStateId
  else if (t.toCategory) st = 'cat:' + t.toCategory
  Object.assign(autoForm, { id: a.id, name: a.name, teamId: a.teamId || '', enabled: a.enabled, event: t.event || 'status_changed', statusTarget: st, actions: JSON.parse(JSON.stringify(a.actions || [])) })
  showAuto.value = true
}
function addAutoAction() { autoForm.actions.push(defaultAction('set_status')) }
function onActionType(i: number) { autoForm.actions[i] = defaultAction(autoForm.actions[i].type) }
const saveAuto = () => wrap(async () => {
  if (!autoForm.name.trim()) { err.value = 'Name required'; return }
  const trigger: any = { event: autoForm.event }
  if (autoForm.event === 'status_changed' && autoForm.statusTarget) {
    if (autoForm.statusTarget.startsWith('cat:')) trigger.toCategory = autoForm.statusTarget.slice(4)
    else if (autoForm.statusTarget.startsWith('state:')) trigger.toStateId = autoForm.statusTarget.slice(6)
  }
  const p = { teamId: autoForm.teamId, name: autoForm.name, enabled: autoForm.enabled, trigger, actions: autoForm.actions }
  if (autoForm.id) await store.updateAutomation(autoForm.id, p)
  else await store.createAutomation(p)
  showAuto.value = false
})
// человекочитаемое описание правила для списка
function autoSummary(a: any) {
  const t = a.trigger || {}
  const ev = AUTO_EVENTS.find((e) => e.value === t.event)?.label || t.event
  let cond = ''
  if (t.event === 'status_changed') {
    if (t.toCategory) cond = ` → ${t.toCategory}`
    else if (t.toStateId) cond = ` → ${store.states[t.toStateId]?.name || '?'}`
  }
  const acts = (a.actions || []).map((x: any) => ACTION_TYPES.find((y) => y.value === x.type)?.label || x.type).join(', ')
  return `${ev}${cond} ⇒ ${acts}`
}

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
// iCal-подписка
const showIcal = ref(false)
const icalUrl = ref('')
const icalCopied = ref(false)
const openIcal = () => wrap(async () => { icalUrl.value = await store.icalFeedUrl(false); showIcal.value = true })
const rotateIcal = () => wrap(async () => { icalUrl.value = await store.icalFeedUrl(true) })
function copyIcal() { navigator.clipboard?.writeText(icalUrl.value); icalCopied.value = true; setTimeout(() => (icalCopied.value = false), 1500) }

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
const NOTIF_META: Record<string, { icon: string; verb: string; color: string }> = {
  assigned: { icon: 'i-heroicons-user-plus', verb: 'Assigned to you', color: '#5e6ad2' },
  state_changed: { icon: 'i-heroicons-arrow-path', verb: 'Status changed', color: '#d29922' },
  commented: { icon: 'i-heroicons-chat-bubble-left-ellipsis', verb: 'New comment', color: '#3fb950' },
  mention: { icon: 'i-heroicons-at-symbol', verb: 'Mentioned you', color: '#5e6ad2' },
}
function notifMeta(t: string) { return NOTIF_META[t] || { icon: 'i-heroicons-bell', verb: t, color: '#7c828c' } }
// Inbox-фильтры (сайдбар)
const inboxFilter = ref<'all' | 'unread' | 'assigned' | 'state_changed'>('all')
const INBOX_FILTERS = [
  { key: 'all', label: 'All', icon: 'i-heroicons-inbox' },
  { key: 'unread', label: 'Unread', icon: 'i-heroicons-envelope' },
  { key: 'assigned', label: 'Assigned to me', icon: 'i-heroicons-user-plus' },
  { key: 'state_changed', label: 'Status changes', icon: 'i-heroicons-arrow-path' },
  { key: 'commented', label: 'Comments', icon: 'i-heroicons-chat-bubble-left-ellipsis' },
]
const inboxList = computed(() => {
  // скрываем типы, выключенные в настройках уведомлений
  const all = store.myNotifications.filter((n: any) => notifPrefs.value[n.type] !== false)
  if (inboxFilter.value === 'all') return all
  if (inboxFilter.value === 'unread') return all.filter((n: any) => !n.readAt)
  return all.filter((n: any) => n.type === inboxFilter.value)
})
function inboxCount(key: string) {
  if (key === 'all') return store.myNotifications.length
  if (key === 'unread') return store.unreadCount
  return store.myNotifications.filter((n: any) => n.type === key).length
}
function notifTime(at: any) {
  const s = String(at); const d = new Date(s)
  if (isNaN(+d)) return s.slice(5, 10)
  return d.toLocaleDateString('en', { month: 'short', day: 'numeric' })
}

// --- Meetings ---
const meeting = useMeeting()
// ресайз правых панелей (таска/цикл/тред)
const detailPanel = useResizable('detail', 384)
const threadPanel = useResizable('thread', 384)
const cyclePanel = useResizable('cycle', 320)
const startOrJoinHuddle = (channelId: string) => wrap(async () => {
  const existing = store.liveHuddleForChannel(channelId)
  if (existing && meeting.activeMeetingId.value === existing.id) return // уже в нём
  const mid = existing ? existing.id : await store.startHuddle(channelId)
  await meeting.join(mid)
})
const joinScheduled = (mid: string) => wrap(() => meeting.join(mid))

// --- Chats ---
const msgDraft = ref('')
const composerEl = ref<HTMLTextAreaElement | null>(null)
const showPreview = ref(false)
const currentChannel = computed(() => store.channels[store.activeChannelId])
const threadParent = computed(() => store.channelMessages(store.activeChannelId).find((m: any) => m.id === store.threadParentId))
const membersOpen = ref(false)
// управлять списком можно только в приватном канале (в публичном участники = весь воркспейс)
const canManageMembers = computed(() => !!currentChannel.value && currentChannel.value.kind === 'channel' && currentChannel.value.private && store.isChannelMember(currentChannel.value.id))
const channelActions = computed(() => {
  const c = currentChannel.value
  if (!c || c.kind === 'dm') return []
  const items: any[] = []
  if (c.private && store.isChannelMember(c.id)) items.push({ label: 'Leave channel', icon: 'i-heroicons-arrow-right-on-rectangle', click: () => wrap(() => store.leaveChannel(c.id)) })
  if (store.isChannelMember(c.id) || c.createdBy === store.currentUserId) items.push({ label: 'Archive channel', icon: 'i-heroicons-archive-box', click: () => wrap(() => store.archiveChannel(c.id)) })
  return items.length ? [items] : []
})
const channelAddable = computed(() => {
  if (!currentChannel.value) return [] as any[]
  const inCh = new Set(store.channelMembersOf(currentChannel.value.id))
  return Object.values(store.users).filter((u: any) => !inCh.has(u.id)).map((u: any) => ({ label: store.userName(u.id), value: u.id }))
})
const addMember = (uid: string) => { if (uid && currentChannel.value) wrap(() => store.addChannelMember(currentChannel.value!.id, uid)) }
watch(() => store.activeChannelId, () => { membersOpen.value = false })
function channelTitle(c: any) {
  if (!c) return ''
  return c.kind === 'dm' ? store.userName(store.dmOtherId(c.id)) : c.name
}
// inline @/# пикеры (юзеры + сущности)
const chatMention = useMention(msgDraft, composerEl)
const descMention = useMention(descDraft, descEl)
const docMention = useMention(docDraft, docEl)
const commentMention = useMention(commentDraft, commentEl)
function onCommentKeydown(e: KeyboardEvent) {
  if (commentMention.onKeydown(e)) return
  if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); addComment() }
}
function onComposerKeydown(e: KeyboardEvent) {
  if (chatMention.onKeydown(e)) return
  if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); sendMsg() }
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
const chatFileEl = ref<HTMLInputElement | null>(null)
const onChatFilePick = (e: Event) => {
  const files = (e.target as HTMLInputElement).files
  if (!files?.length) return
  for (const f of Array.from(files)) pendingFiles.value = [...pendingFiles.value, f] // всё (картинки+файлы) → превью в композере, шлём прикреплённым
  if (chatFileEl.value) chatFileEl.value.value = ''
}
const msgScrollEl = ref<HTMLElement | null>(null)
function scrollMessagesBottom() { nextTick(() => { const el = msgScrollEl.value; if (el) el.scrollTop = el.scrollHeight }) }
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
  if (firstUnreadMsgId.value && msgScrollEl.value) {
    const el = msgScrollEl.value.querySelector('[data-unread-divider]') as HTMLElement | null
    if (el) el.scrollIntoView({ block: 'center' }); else scrollMessagesBottom()
  } else scrollMessagesBottom()
  store.markChannelRead(id)
}
const openDM = (userId: string) => wrap(async () => { activeSection.value = 'chats'; store.closeThread(); await store.createDM(userId); scrollMessagesBottom() })
const pendingFiles = ref<File[]>([])
const sendMsg = () => wrap(async () => {
  const b = msgDraft.value.trim()
  if ((!b && !pendingFiles.value.length) || !store.activeChannelId) return
  const files = pendingFiles.value.slice()
  msgDraft.value = ''; pendingFiles.value = []; showPreview.value = false
  await store.sendMessageAttached(store.activeChannelId, b, files)
  scrollMessagesBottom()
})
const onChatPaste = (e: ClipboardEvent) => {
  const items = e.clipboardData?.items; if (!items) return
  for (const it of Array.from(items)) {
    if (it.type.startsWith('image/')) { e.preventDefault(); const f = it.getAsFile(); if (f) pendingFiles.value = [...pendingFiles.value, f] }
  }
}
function wrapSel(pre: string, post = pre) {
  const el = composerEl.value
  if (!el) { msgDraft.value += pre + post; return }
  const s = el.selectionStart, e = el.selectionEnd, v = msgDraft.value
  msgDraft.value = v.slice(0, s) + pre + v.slice(s, e) + post + v.slice(e)
  nextTick(() => { el.focus(); el.selectionStart = s + pre.length; el.selectionEnd = e + pre.length })
}
// reactions (комменты задачи; чат-сообщения рендерит ChatMessage)
const EMOJIS = ['👍', '❤️', '😄', '🎉', '👀', '🚀', '✅', '🔥']
const threadDraft = ref('')
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
const threadPending = ref<File[]>([])
const threadFileEl = ref<HTMLInputElement | null>(null)
const onThreadFilePick = (e: Event) => {
  const files = (e.target as HTMLInputElement).files; if (!files) return
  for (const f of Array.from(files)) threadPending.value = [...threadPending.value, f]
  ;(e.target as HTMLInputElement).value = ''
}
const onThreadPaste = (e: ClipboardEvent) => {
  const items = e.clipboardData?.items; if (!items) return
  for (const it of Array.from(items)) if (it.type.startsWith('image/')) { e.preventDefault(); const f = it.getAsFile(); if (f) threadPending.value = [...threadPending.value, f] }
}
const sendThread = () => wrap(async () => {
  const b = threadDraft.value.trim()
  if ((!b && !threadPending.value.length) || !store.threadParentId) return
  const files = threadPending.value.slice()
  threadDraft.value = ''; threadPending.value = []
  await store.sendMessageAttached(store.activeChannelId, b, files, store.threadParentId)
})
// mentions
function insertToken(text: string) {
  const el = composerEl.value
  if (!el) { msgDraft.value += text; return }
  const s = el.selectionStart, v = msgDraft.value
  msgDraft.value = v.slice(0, s) + text + v.slice(s)
  nextTick(() => { el.focus(); el.selectionStart = el.selectionEnd = s + text.length })
}
const mentionItems = computed(() => [[...store.wsUsers, store.users[store.currentUserId]].filter(Boolean).map((u: any) => ({
  label: u.name || u.email, avatar: undefined, click: () => insertToken(`@[${u.name || u.email}](${u.id}) `),
}))])
// members
const showMembers = ref(false)
const inviteEmail = ref('')
const inviteRole = ref('member')
const ROLES = ['member', 'admin', 'guest']
const inviteLink = ref('')
const doInvite = () => wrap(async () => {
  if (!inviteEmail.value.trim()) return
  const token = await store.inviteMember(inviteEmail.value.trim(), inviteRole.value)
  inviteEmail.value = ''
  // новый юзер ещё не зарегистрирован → даём ссылку-приглашение для отправки вручную
  inviteLink.value = token ? `${location.origin}/invite/${token}` : ''
})
const copyInviteLink = () => { if (inviteLink.value) navigator.clipboard?.writeText(inviteLink.value) }

// Фон модалок = основной контент (surface). ВАЖНО: нужен dark:-вариант, иначе дефолтный
// `dark:bg-gray-900` побеждает в twMerge при .dark теме (см. mergeConfig=defuTwMerge).
const modalUi = { background: 'dark:bg-surface', ring: 'ring-1 ring-subtle' }

// --- Settings-модалка (workspace) ---
const showSettings = ref(false)
const settingsTab = ref<'general' | 'members' | 'teams' | 'labels' | 'automations' | 'sections'>('general')
const SETTINGS_TABS = [
  { key: 'general', label: 'General', icon: 'i-heroicons-cog-6-tooth', group: 'Workspace' },
  { key: 'members', label: 'Members', icon: 'i-heroicons-user-group', group: 'Workspace' },
  { key: 'teams', label: 'Teams', icon: 'i-heroicons-rectangle-stack', group: 'Workspace' },
  { key: 'labels', label: 'Labels', icon: 'i-heroicons-tag', group: 'Workspace' },
  { key: 'automations', label: 'Automations', icon: 'i-heroicons-bolt', group: 'Workspace' },
  { key: 'account', label: 'Account', icon: 'i-heroicons-user-circle', group: 'Personal' },
  { key: 'appearance', label: 'Appearance', icon: 'i-heroicons-swatch', group: 'Personal' },
  { key: 'notifications', label: 'Notifications', icon: 'i-heroicons-bell', group: 'Personal' },
  { key: 'sections', label: 'Sidebar', icon: 'i-heroicons-bars-3', group: 'Personal' },
]
const settingsGroups = computed(() => {
  const g: Record<string, any[]> = {}
  for (const t of SETTINGS_TABS) (g[t.group] ||= []).push(t)
  return g
})
// appearance (клиентское, localStorage)
const density = ref('default')
const DENSITY_PX: Record<string, string> = { compact: '15px', default: '16px', comfortable: '18px' }
function applyDensity(d: string) {
  density.value = d
  if (import.meta.client) { document.documentElement.style.fontSize = DENSITY_PX[d] || '16px'; localStorage.setItem('pms-density', d) }
}
// notifications prefs (клиентский фильтр типов в Inbox)
const notifPrefs = ref<Record<string, boolean>>({ assigned: true, state_changed: true, commented: true, mention: true })
function toggleNotifPref(k: string) {
  notifPrefs.value[k] = !notifPrefs.value[k]
  if (import.meta.client) localStorage.setItem('pms-notif-prefs', JSON.stringify(notifPrefs.value))
}
const NOTIF_PREF_LIST = [
  { key: 'assigned', label: 'Assigned to me' },
  { key: 'state_changed', label: 'Status changes' },
  { key: 'commented', label: 'New comments' },
  { key: 'mention', label: 'Mentions' },
]
const wsNameDraft = ref('')
function openSettings() {
  wsNameDraft.value = store.activeWorkspace?.name || ''
  openRailCfg() // подготовить cfgList для вкладки Sidebar
  settingsTab.value = 'general'; showSettings.value = true; showAuto.value = false
}
const saveWsName = () => wrap(() => store.renameWorkspace(wsNameDraft.value.trim()))
// avatar / logo upload
const avatarEl = ref<HTMLInputElement | null>(null)
const logoEl = ref<HTMLInputElement | null>(null)
const onAvatarPick = (e: Event) => wrap(async () => {
  const f = (e.target as HTMLInputElement).files?.[0]; if (!f) return
  const up = await store.uploadAndGetKey(f); await store.setAvatar(up.key)
  if (avatarEl.value) avatarEl.value.value = ''
})
const onLogoPick = (e: Event) => wrap(async () => {
  const f = (e.target as HTMLInputElement).files?.[0]; if (!f) return
  const up = await store.uploadAndGetKey(f); await store.setLogo(up.key)
  if (logoEl.value) logoEl.value.value = ''
})
function confirmDeleteTeam(t: any) {
  if (!import.meta.client || !confirm(`Delete team "${t.name}"? Sub-teams included. Must have no active issues.`)) return
  wrap(async () => { await store.deleteTeam(t.id); if (selectedTeamId.value === t.id) { selectedTeamId.value = ''; teamSection.value = '' } })
}
function confirmDeleteWs() {
  if (!import.meta.client || !confirm(`Delete workspace "${store.activeWorkspace?.name}"? This hides it and all its data.`)) return
  wrap(async () => { await store.deleteWorkspace(); showSettings.value = false })
}
// labels (workspace-wide管理)
const allLabels = computed(() => Object.values(store.labels))
const newWsLabelName = ref('')
const newWsLabelColor = ref('#5e6ad2')
const LABEL_COLORS = ['#5e6ad2', '#e5484d', '#d29922', '#3fb950', '#1f9cf0', '#a371f7', '#6b747e']
const addWsLabel = () => wrap(async () => {
  const tid = selectedTeamId.value || store.teamList[0]?.id
  if (!newWsLabelName.value.trim() || !tid) return
  await store.createLabel(tid, newWsLabelName.value, newWsLabelColor.value); newWsLabelName.value = ''
})

// typing throttle
let typingTs = 0
function onComposerInput() {
  const now = Date.now()
  if (store.activeChannelId && now - typingTs > 2500) { typingTs = now; store.sendTyping(store.activeChannelId) }
}
function onMessagesScroll(e: Event) {
  const el = e.target as HTMLElement
  if (el.scrollTop < 40 && currentChannel.value) {
    const prevH = el.scrollHeight
    store.loadOlder(currentChannel.value.id).then(() => nextTick(() => { el.scrollTop = el.scrollHeight - prevH }))
  }
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
onMounted(() => {
  window.addEventListener('keydown', (e) => {
    if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') { e.preventDefault(); openPalette() }
    else if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'n') { e.preventDefault(); openCreate('issue') }
    else if (e.key === 'Escape') { paletteOpen.value = false; createOpen.value = false }
  })
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
          <button class="w-full rounded-sm px-2 py-1.5 text-left text-tertiary hover:bg-hover" @click="openCreate('workspace')">＋ New workspace</button>
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
        <UBadge v-if="store.unreadCount" color="primary" size="xs" class="pointer-events-none absolute -right-1 -top-1 rounded-full">{{ store.unreadCount }}</UBadge>
        <div v-if="notifOpen" class="absolute right-0 top-10 z-50 w-80 rounded-md border border-subtle bg-elevated shadow-lg">
          <div class="flex items-center justify-between border-b border-subtle px-3 py-2 text-xs uppercase text-tertiary">
            <span><span class="text-ok"># </span>Notifications</span>
            <UButton v-if="store.unreadCount" size="2xs" color="gray" variant="ghost" @click="store.markAllNotificationsRead()">Mark all read</UButton>
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
          :variant="activeSection === s.key ? 'soft' : 'ghost'"
          @click="switchSection(s.key as any)" />
        <span v-if="s.key === 'chats' && store.totalUnread" class="absolute -right-0.5 -top-0.5 flex min-w-[16px] items-center justify-center rounded-full bg-accent px-1 text-[10px] font-semibold text-white">{{ store.totalUnread }}</span>
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
          <UButton class="ml-auto" icon="i-heroicons-plus" color="gray" variant="ghost" size="2xs" square @click="openCreate('dashboard')" />
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
        <UButton block icon="i-heroicons-plus" color="gray" variant="ghost" size="sm" class="mb-4" @click="openCalCreate(calAnchor)">New event</UButton>
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
          <UButton class="ml-auto" icon="i-heroicons-plus" color="gray" variant="ghost" size="2xs" square @click="openCreate('channel')" />
        </div>
        <div v-for="c in store.channelList" :key="c.id"
          class="flex h-8 cursor-pointer items-center gap-2 rounded px-2 hover:bg-hover"
          :class="store.activeChannelId === c.id ? 'bg-active text-white' : store.channelUnreadCount(c.id) ? 'font-semibold text-white' : 'text-secondary'"
          @click="openChannel(c.id)"><UIcon :name="c.private ? 'i-heroicons-lock-closed' : 'i-heroicons-hashtag'" class="text-tertiary" /> <span class="truncate">{{ c.name }}</span>
          <span v-if="store.channelUnreadCount(c.id) && store.activeChannelId !== c.id" class="ml-auto rounded-full bg-accent px-1.5 text-[11px] font-medium text-white">{{ store.channelUnreadCount(c.id) }}</span></div>
        <div v-if="!store.channelList.length" class="px-2 py-1 text-xs text-tertiary/60">// no channels</div>

        <div class="mb-1 mt-4 px-2 text-xs font-medium uppercase tracking-wider text-tertiary"><span class="text-ok"># </span>Direct messages</div>
        <div v-for="u in store.wsUsers" :key="u.id"
          class="flex h-8 cursor-pointer items-center gap-2 rounded px-2 hover:bg-hover"
          :class="currentChannel?.kind === 'dm' && store.dmOtherId(currentChannel.id) === u.id ? 'bg-active text-white' : store.channelUnreadCount(store.dmChannelWith(u.id)) ? 'font-semibold text-white' : 'text-secondary'"
          @click="openDM(u.id)">
          <Avatar :name="u.name || u.email" :avatar-key="u.avatarKey" :size="20" />
          <span class="truncate">{{ u.name || u.email }}</span>
          <span v-if="store.channelUnreadCount(store.dmChannelWith(u.id))" class="ml-auto rounded-full bg-accent px-1.5 text-[11px] font-medium text-white">{{ store.channelUnreadCount(store.dmChannelWith(u.id)) }}</span>
        </div>
        <div v-if="!store.wsUsers.length" class="px-2 py-1 text-xs text-tertiary/60">// no other users</div>
      </template>

      <!-- MEETINGS section -->
      <template v-if="store.activeWorkspaceId && activeSection === 'meetings'">
        <UButton block icon="i-heroicons-plus" color="gray" size="sm" class="mb-4" @click="openCreate('meeting')">New meeting</UButton>
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
            <UIcon :name="expandTeam[t.id] ? 'i-heroicons-chevron-down' : 'i-heroicons-chevron-right'" class="shrink-0 text-sm text-tertiary" />
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
              <UIcon :name="expandCycles[t.id] ? 'i-heroicons-chevron-down' : 'i-heroicons-chevron-right'" class="rounded text-sm hover:bg-active" @click.stop="expandCycles[t.id] = !expandCycles[t.id]" /><UIcon name="i-heroicons-arrow-path-rounded-square" /> Cycles
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
              <UIcon :name="expandDocs[t.id] ? 'i-heroicons-chevron-down' : 'i-heroicons-chevron-right'" class="rounded text-sm hover:bg-active" @click.stop="expandDocs[t.id] = !expandDocs[t.id]" /><UIcon name="i-heroicons-document-text" /> Docs
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
              <UIcon :name="expandViews[t.id] ? 'i-heroicons-chevron-down' : 'i-heroicons-chevron-right'" class="rounded text-sm hover:bg-active" @click.stop="expandViews[t.id] = !expandViews[t.id]" /><UIcon name="i-heroicons-square-3-stack-3d" /> Views
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
    <main class="flex-1 overflow-y-auto rounded-xl border border-subtle bg-surface">
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
            <UButton icon="i-heroicons-chevron-left" color="gray" variant="ghost" size="xs" square @click="shiftRange(-1)" />
            <UButton color="gray" variant="ghost" size="xs" @click="calToday">Today</UButton>
            <UButton icon="i-heroicons-chevron-right" color="gray" variant="ghost" size="xs" square @click="shiftRange(1)" />
          </div>
          <div class="ml-2 flex items-center gap-0.5 rounded-md bg-elevated p-0.5">
            <button v-for="v in ['month','week','day']" :key="v"
              class="rounded px-2 py-0.5 text-xs capitalize transition-colors"
              :class="calView === v ? 'bg-active text-white' : 'text-tertiary hover:text-secondary'"
              @click="calView = v as any">{{ v }}</button>
          </div>
          <UButton class="ml-auto" icon="i-heroicons-arrow-top-right-on-square" size="xs" color="gray" variant="ghost" @click="openIcal">Subscribe</UButton>
          <UButton icon="i-heroicons-plus" size="xs" color="gray" variant="ghost" @click="openCalCreate(calAnchor)">New event</UButton>
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
            <UButton class="ml-auto" icon="i-heroicons-arrow-down-tray" color="gray" size="xs" :disabled="!repRows.length" @click="exportReportCsv">Export CSV</UButton>
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
                <div class="relative h-6 flex-1 overflow-hidden rounded bg-elevated">
                  <div class="absolute inset-y-0 left-0 rounded bg-accent/40" :style="{ width: (r.minutes / repMax * 100) + '%' }" />
                </div>
                <div class="w-12 shrink-0 text-right text-xs text-tertiary">{{ repTotal ? Math.round(r.minutes / repTotal * 100) : 0 }}%</div>
                <div class="w-20 shrink-0 text-right text-sm tabular-nums text-white">{{ fmtMin(r.minutes) }}</div>
                <div class="w-10 shrink-0 text-right text-xs text-tertiary">{{ r.count }}</div>
              </div>
            </div>
          </template>
        </div>
        </ClientOnly>
      </template>

      <template v-else-if="activeSection === 'dashboard'">
      <template v-if="selectedDashboard">
        <div class="sticky top-0 z-10 flex items-center gap-2 border-b border-subtle bg-surface px-4 py-3 text-sm">
          <UIcon name="i-heroicons-squares-2x2" class="text-tertiary" />
          <span class="font-medium">{{ selectedDashboard.name }}</span>
          <UButton class="ml-auto" icon="i-heroicons-plus" size="xs" color="gray" variant="ghost" @click="showWidget = true">Add widget</UButton>
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
            @click="showWidget = true">+ Add your first widget</div>
        </div>
      </template>
      <div v-else class="p-8 text-sm text-tertiary">Pick or create a dashboard →</div>
      </template>

      <!-- INBOX section -->
      <template v-else-if="activeSection === 'inbox'">
        <div class="sticky top-0 z-10 flex items-center gap-2 border-b border-subtle bg-surface px-4 py-3 text-sm">
          <UIcon name="i-heroicons-inbox" class="text-tertiary" />
          <span class="font-medium">Inbox</span>
          <span v-if="store.unreadCount" class="rounded bg-accent/20 px-1.5 text-xs text-accent">{{ store.unreadCount }}</span>
          <UButton v-if="store.unreadCount" class="ml-auto" icon="i-heroicons-check" size="xs" color="gray" variant="ghost" @click="store.markAllNotificationsRead()">Mark all read</UButton>
        </div>
        <div v-for="n in inboxList" :key="n.id"
          class="group flex cursor-pointer items-start gap-3 border-b border-subtle/40 px-4 py-3 hover:bg-hover"
          @click="openNotif(n)">
          <span class="mt-1.5 h-2 w-2 shrink-0 rounded-full" :class="n.readAt ? 'bg-transparent' : 'bg-accent'" />
          <span class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-elevated">
            <UIcon :name="notifMeta(n.type).icon" :style="{ color: notifMeta(n.type).color }" />
          </span>
          <div class="min-w-0 flex-1">
            <div class="flex items-center gap-2">
              <span :class="n.readAt ? 'text-secondary' : 'font-medium'">{{ notifMeta(n.type).verb }}</span>
              <span class="text-xs text-tertiary">{{ store.issueById(n.entityId)?.identifier }}</span>
              <span class="ml-auto shrink-0 text-xs text-tertiary">{{ notifTime(n.at) }}</span>
            </div>
            <div class="truncate text-sm text-tertiary">{{ store.issueById(n.entityId)?.title || n.entityType }}</div>
          </div>
          <UButton v-if="!n.readAt" class="mt-0.5 opacity-0 group-hover:opacity-100" icon="i-heroicons-check" color="gray" variant="ghost" size="2xs" square title="Mark read" @click.stop="store.markNotificationRead(n.id)" />
        </div>
        <div v-if="!inboxList.length" class="flex flex-col items-center justify-center gap-2 py-20 text-tertiary">
          <UIcon name="i-heroicons-inbox" class="text-3xl" />
          <span class="text-sm">{{ inboxFilter === 'all' ? "// inbox zero — you're all caught up" : '// nothing here' }}</span>
        </div>
      </template>

      <!-- CHATS section -->
      <template v-else-if="activeSection === 'chats'">
        <div v-if="currentChannel" class="flex h-full flex-col">
          <div class="flex shrink-0 items-center gap-2 border-b border-subtle px-4 py-3">
            <UIcon :name="currentChannel.kind === 'dm' ? 'i-heroicons-user' : currentChannel.private ? 'i-heroicons-lock-closed' : 'i-heroicons-hashtag'" class="text-tertiary" />
            <span class="font-medium">{{ channelTitle(currentChannel) }}</span>

            <!-- channel members (count + management dropdown) -->
            <div v-if="currentChannel.kind !== 'dm'" class="relative ml-auto">
              <UButton color="gray" variant="ghost" size="xs" icon="i-heroicons-users" @click="membersOpen = !membersOpen">
                {{ store.channelParticipants(currentChannel.id).length }}
              </UButton>
              <div v-if="membersOpen" class="absolute right-0 top-9 z-30 w-64 rounded-lg border border-subtle bg-elevated p-2 shadow-xl">
                <div class="mb-1 px-1 text-xs font-medium uppercase tracking-wide text-tertiary"><span class="text-ok"># </span>Members{{ !currentChannel.private ? ' · entire workspace' : '' }}</div>
                <div class="max-h-56 overflow-y-auto">
                  <div v-for="uid in store.channelParticipants(currentChannel.id)" :key="uid" class="group flex items-center gap-2 rounded px-1.5 py-1 hover:bg-hover">
                    <Avatar :name="store.userName(uid)" :avatar-key="store.users[uid]?.avatarKey" :size="22" />
                    <span class="min-w-0 flex-1 truncate text-sm">{{ store.userName(uid) }}{{ uid === store.currentUserId ? ' (you)' : '' }}</span>
                    <UButton v-if="canManageMembers && uid !== store.currentUserId" icon="i-heroicons-x-mark" color="gray" variant="ghost" size="2xs" square title="Remove" @click="store.removeChannelMember(currentChannel.id, uid)" />
                  </div>
                </div>
                <template v-if="canManageMembers">
                  <div class="my-1.5 h-px bg-subtle" />
                  <USelectMenu :options="channelAddable" value-attribute="value" option-attribute="label" searchable placeholder="Add member…" @update:model-value="addMember" />
                </template>
              </div>
            </div>

            <template v-if="store.liveHuddleForChannel(currentChannel.id)">
              <UButton class="ml-auto" :color="meeting.activeMeetingId.value === store.liveHuddleForChannel(currentChannel.id)?.id ? 'gray' : 'green'" variant="ghost" size="xs" icon="i-heroicons-signal"
                @click="startOrJoinHuddle(currentChannel.id)">
                {{ meeting.activeMeetingId.value === store.liveHuddleForChannel(currentChannel.id)?.id ? 'In huddle' : 'Join huddle' }}
                · {{ store.meetingJoined(store.liveHuddleForChannel(currentChannel.id)!.id).length }}
              </UButton>
            </template>
            <UButton v-else class="ml-auto" color="gray" variant="ghost" size="xs" icon="i-heroicons-signal" title="Start huddle" @click="startOrJoinHuddle(currentChannel.id)">Huddle</UButton>
            <UButton :icon="linkCopied ? 'i-heroicons-check' : 'i-heroicons-link'" color="gray" variant="ghost" size="xs" square :title="linkCopied ? 'Copied!' : 'Copy link'" @click="copyLink" />
            <UDropdown v-if="channelActions.length" :items="channelActions" :popper="{ placement: 'bottom-end' }">
              <UButton icon="i-heroicons-ellipsis-horizontal" color="gray" variant="ghost" size="xs" square title="Channel actions" />
            </UDropdown>
          </div>
          <!-- messages -->
          <div ref="msgScrollEl" class="flex flex-1 flex-col gap-1 overflow-y-auto px-2 py-3" @scroll="onMessagesScroll">
            <div v-if="store.hasMoreMessages[currentChannel.id]" class="py-1 text-center text-xs text-tertiary">loading older…</div>
            <template v-for="m in store.channelMessages(currentChannel.id)" :key="m.id">
            <div v-if="m.id === firstUnreadMsgId" data-unread-divider class="my-1 flex items-center gap-2 px-2 text-xs font-medium text-danger">
              <span class="h-px flex-1 bg-danger/40" /> New messages <span class="h-px flex-1 bg-danger/40" />
            </div>
            <ChatMessage :m="m" />
            </template>
            <div v-if="store.messages[currentChannel.id] === undefined" class="space-y-3 py-2">
              <div v-for="n in 6" :key="n" class="flex gap-3"><div class="skeleton h-8 w-8 rounded-full" /><div class="flex-1 space-y-1"><div class="skeleton h-3 w-24" /><div class="skeleton h-3" :style="{ width: 30 + (n * 13) % 55 + '%' }" /></div></div>
            </div>
            <div v-else-if="!store.channelMessages(currentChannel.id).length" class="py-16 text-center text-sm text-tertiary">No messages yet — say hi 👋</div>
          </div>
          <!-- typing -->
          <div class="h-5 px-4 text-xs text-tertiary">
            <span v-if="store.typingNames(currentChannel.id).length">{{ store.typingNames(currentChannel.id).join(', ') }} typing…</span>
          </div>
          <!-- composer -->
          <div class="shrink-0 px-3 pb-3">
            <PendingAttachments v-if="pendingFiles.length" class="mb-1.5" :files="pendingFiles" @remove="(i) => pendingFiles = pendingFiles.filter((_, j) => j !== i)" />
            <div class="rounded-lg border border-subtle bg-elevated">
              <div class="flex items-center gap-1 border-b border-subtle px-2 py-1">
                <UButton color="gray" variant="ghost" size="2xs" square title="Bold" class="font-bold" @click="wrapSel('**')">B</UButton>
                <UButton color="gray" variant="ghost" size="2xs" square title="Italic" class="italic" @click="wrapSel('_')">I</UButton>
                <UButton icon="i-heroicons-code-bracket" color="gray" variant="ghost" size="2xs" square title="Code" @click="wrapSel('`')" />
                <UButton icon="i-heroicons-link" color="gray" variant="ghost" size="2xs" square title="Link" @click="wrapSel('[', '](url)')" />
                <UButton icon="i-heroicons-list-bullet" color="gray" variant="ghost" size="2xs" square title="List" @click="wrapSel('\n- ', '')" />
                <UDropdown :items="mentionItems" :popper="{ placement: 'top-start' }">
                  <UButton icon="i-heroicons-at-symbol" color="gray" variant="ghost" size="2xs" square title="Mention" />
                </UDropdown>
                <UButton icon="i-heroicons-paper-clip" color="gray" variant="ghost" size="2xs" square title="Attach file" @click="chatFileEl?.click()" />
                <input ref="chatFileEl" type="file" multiple class="hidden" @change="onChatFilePick" />
                <UButton class="ml-auto" :icon="showPreview ? 'i-heroicons-pencil' : 'i-heroicons-eye'" color="gray" variant="ghost" size="2xs" square @click="showPreview = !showPreview" />
              </div>
              <div v-if="showPreview" class="prose-chat min-h-[60px] px-3 py-2 text-sm" v-html="renderMarkdown(msgDraft) || '<span class=&quot;text-tertiary&quot;>Nothing to preview</span>'" />
              <div v-else class="relative">
                <MentionMenu :m="chatMention" />
                <textarea ref="composerEl" v-model="msgDraft" rows="3" placeholder="Message… (markdown, @mention, #ref, Enter to send)"
                  class="w-full resize-none bg-transparent px-3 py-2 text-sm outline-none placeholder:text-tertiary"
                  @paste="onChatPaste" @input="onComposerInput(); chatMention.detect()" @keydown="onComposerKeydown" />
              </div>
              <div class="flex items-center justify-between px-3 py-1.5">
                <span class="text-xs text-tertiary">**bold** _italic_ `code` — Enter send, Shift+Enter newline</span>
                <UButton icon="i-heroicons-paper-airplane" size="xs" :disabled="!msgDraft.trim() && !pendingFiles.length" @click="sendMsg">[ send ]</UButton>
              </div>
            </div>
          </div>
        </div>
        <div v-else class="flex h-full flex-col items-center justify-center gap-2 text-tertiary">
          <UIcon name="i-heroicons-chat-bubble-left-right" class="text-3xl" />
          <span class="text-sm"><span class="text-ok">$</span> open a channel or DM</span>
        </div>
      </template>

      <!-- MEETINGS section -->
      <template v-else-if="activeSection === 'meetings'">
        <div class="mx-auto max-w-3xl px-6 py-6">
          <div class="mb-5 flex items-center gap-2">
            <h1 class="text-lg font-semibold text-white">Meetings</h1>
            <UButton class="ml-auto" icon="i-heroicons-plus" color="gray" size="sm" @click="openCreate('meeting')">New meeting</UButton>
          </div>

          <div v-if="store.liveMeetings.length" class="mb-2 text-xs font-medium uppercase tracking-wider text-tertiary"><span class="text-ok"># </span>Live now</div>
          <div v-for="m in store.liveMeetings" :key="m.id" class="mb-2 flex items-center gap-3 rounded-lg border border-subtle bg-elevated p-3">
            <span class="flex h-2.5 w-2.5 shrink-0 rounded-full bg-danger" />
            <div class="min-w-0 flex-1">
              <div class="truncate font-medium text-white">{{ m.kind === 'huddle' ? 'Huddle' : (m.title || 'Meeting') }}</div>
              <div class="text-xs text-tertiary">{{ store.meetingJoined(m.id).length }} in call · {{ store.meetingParts(m.id).length }} invited</div>
            </div>
            <div class="flex -space-x-1.5">
              <span v-for="p in store.meetingJoined(m.id).slice(0, 4)" :key="p.userId"
                class="flex h-6 w-6 items-center justify-center rounded-full border border-surface bg-active text-[10px] text-white" :title="store.userName(p.userId)">
                {{ store.userName(p.userId).charAt(0).toUpperCase() }}
              </span>
            </div>
            <UButton v-if="meeting.activeMeetingId.value !== m.id" color="green" size="sm" icon="i-heroicons-signal" @click="joinScheduled(m.id)">Join</UButton>
            <UButton v-else color="gray" size="sm" variant="ghost" disabled>In call</UButton>
          </div>

          <div v-if="store.upcomingMeetings.length" class="mb-2 mt-6 text-xs font-medium uppercase tracking-wider text-tertiary"><span class="text-ok"># </span>Upcoming</div>
          <div v-for="m in store.upcomingMeetings" :key="m.id" class="group mb-2 flex items-center gap-3 rounded-lg border border-subtle bg-elevated p-3">
            <UIcon name="i-heroicons-calendar" class="shrink-0 text-tertiary" />
            <div class="min-w-0 flex-1">
              <div class="truncate font-medium text-white">{{ m.title || 'Meeting' }}</div>
              <div class="text-xs text-tertiary">
                <span v-if="m.scheduledAt">{{ new Date(m.scheduledAt).toLocaleString() }}<span v-if="m.endsAt">–{{ new Date(m.endsAt).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" }) }}</span> · </span>{{ store.meetingParts(m.id).length }} invited
              </div>
            </div>
            <div class="flex -space-x-1.5">
              <span v-for="p in store.meetingParts(m.id).slice(0, 4)" :key="p.userId"
                class="flex h-6 w-6 items-center justify-center rounded-full border border-surface bg-elevated text-[10px] text-tertiary" :title="store.userName(p.userId)">
                {{ store.userName(p.userId).charAt(0).toUpperCase() }}
              </span>
            </div>
            <UButton color="gray" size="sm" icon="i-heroicons-video-camera" @click="joinScheduled(m.id)">Start</UButton>
            <UButton class="opacity-0 group-hover:opacity-100" icon="i-heroicons-trash" color="gray" variant="ghost" size="xs" square title="Delete" @click="store.endMeeting(m.id)" />
          </div>

          <div v-if="store.pastMeetings.length" class="mb-2 mt-6 text-xs font-medium uppercase tracking-wider text-tertiary"><span class="text-ok"># </span>Past</div>
          <div v-for="m in store.pastMeetings" :key="m.id" class="group mb-2 flex items-center gap-3 rounded-lg border border-subtle/60 bg-elevated/40 p-3">
            <UIcon name="i-heroicons-clock" class="shrink-0 text-tertiary/60" />
            <div class="min-w-0 flex-1">
              <div class="truncate font-medium text-secondary">{{ m.title || 'Meeting' }}</div>
              <div class="text-xs text-tertiary">
                <span v-if="m.scheduledAt">{{ new Date(m.scheduledAt).toLocaleString() }}<span v-if="m.endsAt">–{{ new Date(m.endsAt).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" }) }}</span> · </span>{{ store.meetingParts(m.id).length }} invited
              </div>
            </div>
            <UButton color="gray" size="sm" icon="i-heroicons-arrow-path" title="Start again" @click="joinScheduled(m.id)">Restart</UButton>
            <UButton class="opacity-0 group-hover:opacity-100" icon="i-heroicons-trash" color="gray" variant="ghost" size="xs" square title="Delete" @click="store.endMeeting(m.id)" />
          </div>

          <div v-if="!store.scheduledMeetings.length && !store.liveMeetings.length" class="py-16 text-center text-sm text-tertiary">
            No meetings yet. Create one or start a huddle from any chat.
          </div>
        </div>
      </template>

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
            <span class="w-16 shrink-0 text-sm text-accent2">{{ i.identifier }}</span>
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
          <UIcon name="i-heroicons-chevron-right" class="text-xs text-tertiary" />
          <button class="text-tertiary hover:text-secondary" @click="selectTeamIssues(selectedTeam)">{{ selectedTeam.name }}</button>
          <UIcon name="i-heroicons-chevron-right" class="text-xs text-tertiary" />
          <UIcon name="i-heroicons-arrow-path-rounded-square" class="text-tertiary" />
          <span class="font-medium">Cycles</span>
          <span class="text-tertiary">{{ store.teamCycles(selectedTeamId).length }}</span>
          <UButton class="ml-auto" icon="i-heroicons-plus" size="xs" color="gray" variant="ghost" @click="openCreate('cycle')">New cycle</UButton>
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
          <div v-if="!store.teamCycles(selectedTeamId).length"
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
              <UIcon name="i-heroicons-chevron-right" class="text-xs text-tertiary" />
              <button class="text-tertiary hover:text-secondary" @click="selectTeamIssues(selectedTeam)">{{ selectedTeam.name }}</button>
              <UIcon name="i-heroicons-chevron-right" class="text-xs text-tertiary" />
              <button class="text-tertiary hover:text-secondary" @click="selectedMsId = ''">Milestones</button>
              <UIcon name="i-heroicons-chevron-right" class="text-xs text-tertiary" />
              <UIcon name="i-heroicons-flag" :class="selectedMs.status === 'done' ? 'text-accent' : selectedMs.status === 'active' ? 'text-warn' : 'text-tertiary'" />
              <span class="font-medium">{{ selectedMs.name }}</span>
              <USelectMenu :model-value="selectedMs.status" :options="labelOpts(['planned', 'active', 'done'])" value-attribute="value" option-attribute="label" size="xs" class="w-28" @update:model-value="(st) => store.updateMilestone(selectedMs.id, { status: st })" />
              <span v-if="selectedMs.targetDate" class="text-xs" :class="msOverdue(selectedMs) ? 'text-danger' : 'text-tertiary'">{{ String(selectedMs.targetDate).slice(0,10) }}{{ msOverdue(selectedMs) ? ' · overdue' : '' }}</span>
              <span class="ml-auto text-xs text-tertiary">{{ store.milestoneProgress(selectedMs.id).done }}/{{ store.milestoneProgress(selectedMs.id).total }}</span>
            </div>
            <div class="mt-2 h-1.5 overflow-hidden rounded-full bg-elevated">
              <div class="h-full rounded-full bg-accent" :style="{ width: (store.milestoneProgress(selectedMs.id).total ? store.milestoneProgress(selectedMs.id).done / store.milestoneProgress(selectedMs.id).total * 100 : 0) + '%' }" />
            </div>
          </div>
          <div v-for="i in store.milestoneIssues(selectedMs.id)" :key="i.id"
            class="flex cursor-pointer items-center gap-3 border-b border-subtle/40 px-4 py-2.5 hover:bg-hover" @click="openIssue(i.id)">
            <span class="inline-block shrink-0 w-4 text-center leading-none" :style="{ color: statusColor(store.states[i.stateId]?.category) }">{{ statusGlyph(store.states[i.stateId]?.category) }}</span>
            <span class="w-14 shrink-0 text-sm text-accent2">{{ i.identifier }}</span>
            <span class="flex-1 truncate">{{ i.title }}</span>
            <span v-if="i.assigneeId" class="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-elevated text-[10px]">{{ (store.userName(i.assigneeId) || '?')[0].toUpperCase() }}</span>
          </div>
          <div v-if="!store.milestoneIssues(selectedMs.id).length" class="py-10 text-center text-sm text-tertiary">// no issues — assign via Milestone field</div>
        </template>

        <!-- список вех -->
        <template v-else>
          <div class="sticky top-0 z-10 flex items-center gap-1.5 border-b border-subtle bg-surface px-4 py-3 text-sm">
            <span class="text-tertiary">{{ store.activeWorkspace?.name }}</span>
            <UIcon name="i-heroicons-chevron-right" class="text-xs text-tertiary" />
            <button class="text-tertiary hover:text-secondary" @click="selectTeamIssues(selectedTeam)">{{ selectedTeam.name }}</button>
            <UIcon name="i-heroicons-chevron-right" class="text-xs text-tertiary" />
            <UIcon name="i-heroicons-flag" class="text-tertiary" />
            <span class="font-medium">Milestones</span>
            <span class="text-tertiary">{{ store.teamMilestones(selectedTeamId).length }}</span>
            <UButton class="ml-auto" icon="i-heroicons-plus" size="xs" color="gray" variant="ghost" @click="openCreate('milestone')">New milestone</UButton>
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
                  <div class="h-1.5 flex-1 overflow-hidden rounded-full bg-surface">
                    <div class="h-full rounded-full bg-accent" :style="{ width: (store.milestoneProgress(m.id).total ? store.milestoneProgress(m.id).done / store.milestoneProgress(m.id).total * 100 : 0) + '%' }" />
                  </div>
                  <span>{{ store.milestoneProgress(m.id).done }}/{{ store.milestoneProgress(m.id).total }}</span>
                </div>
              </div>
              <div v-if="!store.teamMilestones(selectedTeamId).length" class="py-10 text-center text-sm text-tertiary">No milestones yet</div>
            </div>
          </div>
        </template>
      </template>

      <!-- DOCS home -->
      <template v-else-if="teamSection === 'docs' && selectedTeam">
        <div class="sticky top-0 z-10 flex items-center gap-1.5 border-b border-subtle bg-surface px-4 py-3 text-sm">
          <span class="text-tertiary">{{ store.activeWorkspace?.name }}</span>
          <UIcon name="i-heroicons-chevron-right" class="text-xs text-tertiary" />
          <button class="text-tertiary hover:text-secondary" @click="selectTeamIssues(selectedTeam)">{{ selectedTeam.name }}</button>
          <UIcon name="i-heroicons-chevron-right" class="text-xs text-tertiary" />
          <UIcon name="i-heroicons-document-text" class="text-tertiary" />
          <span class="font-medium">Docs</span>
          <span class="text-tertiary">{{ store.teamDocs(selectedTeamId).length }}</span>
          <UButton class="ml-auto" icon="i-heroicons-plus" size="xs" color="gray" variant="ghost" @click="openCreate('doc')">New doc</UButton>
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
            <div v-if="!store.teamDocs(selectedTeamId).length"
              class="flex h-28 cursor-pointer items-center justify-center rounded-lg border border-dashed border-subtle text-sm text-tertiary hover:bg-hover"
              @click="openCreate('doc')">+ Create first doc</div>
          </div>
        </div>
      </template>

      <!-- VIEWS page -->
      <template v-else-if="teamSection === 'views' && selectedTeam">
        <div class="sticky top-0 z-10 flex items-center gap-1.5 border-b border-subtle bg-surface px-4 py-3 text-sm">
          <span class="text-tertiary">{{ store.activeWorkspace?.name }}</span>
          <UIcon name="i-heroicons-chevron-right" class="text-xs text-tertiary" />
          <button class="text-tertiary hover:text-secondary" @click="selectTeamIssues(selectedTeam)">{{ selectedTeam.name }}</button>
          <UIcon name="i-heroicons-chevron-right" class="text-xs text-tertiary" />
          <UIcon name="i-heroicons-square-3-stack-3d" class="text-tertiary" />
          <span class="font-medium">Views</span>
          <span class="text-tertiary">{{ store.teamViews(selectedTeamId).length }}</span>
        </div>
        <div class="p-4">
          <div class="divide-y divide-subtle/40 overflow-hidden rounded-lg border border-subtle">
            <div v-for="v in store.teamViews(selectedTeamId)" :key="v.id"
              class="group flex cursor-pointer items-center gap-2 bg-elevated px-4 py-2.5 hover:bg-hover"
              @click="openView(v.id)">
              <UIcon name="i-heroicons-star" class="text-tertiary" />
              <span class="flex-1 truncate">{{ v.name }}</span>
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
            <UIcon name="i-heroicons-chevron-right" class="text-xs text-tertiary" />
            <button class="text-tertiary hover:text-secondary" @click="openTeamSection(selectedDoc.teamId, 'docs')">Docs</button>
            <UIcon name="i-heroicons-chevron-right" class="text-xs text-tertiary" />
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
          <div v-if="docMode !== 'edit'" :class="docMode === 'split' ? 'w-1/2' : 'w-full'" class="prose-chat overflow-y-auto p-4 text-sm leading-relaxed" v-html="renderMarkdown(docDraft) || '<span class=&quot;text-tertiary&quot;>Nothing to preview</span>'" />
        </div>
      </template>

      <template v-else-if="selectedTeam">
        <div class="sticky top-0 z-10 border-b border-subtle bg-surface">
          <div class="flex items-center gap-1.5 px-4 py-2.5 text-sm">
            <!-- breadcrumbs -->
            <span class="text-tertiary">{{ store.activeWorkspace?.name }}</span>
            <UIcon name="i-heroicons-chevron-right" class="text-xs text-tertiary" />
            <button class="font-medium hover:text-secondary" @click="selectTeamIssues(selectedTeam)">{{ selectedTeam.name }}</button>
            <template v-if="selectedCycleId">
              <UIcon name="i-heroicons-chevron-right" class="text-xs text-tertiary" />
              <span>Cycle {{ store.cycles[selectedCycleId]?.number }}</span>
            </template>
            <div class="ml-auto flex items-center gap-1">
              <USelectMenu v-if="store.layout === 'list'" v-model="store.groupBy" :options="labelOpts(['status', 'priority', 'assignee', 'milestone'])" value-attribute="value" option-attribute="label" size="xs" class="w-32" />
              <UButton size="xs" color="gray" variant="ghost" icon="i-heroicons-adjustments-horizontal" title="Manage statuses" @click="showStates = true" />
              <UButton size="xs" :color="showFilters || store.activeFilter ? 'primary' : 'gray'" variant="ghost" icon="i-heroicons-funnel" @click="showFilters = !showFilters" />
              <span class="mx-1 h-4 w-px bg-subtle" />
              <UButton size="xs" square icon="i-heroicons-list-bullet" :color="store.layout==='list' ? 'primary' : 'gray'" :variant="store.layout==='list' ? 'soft' : 'ghost'" @click="store.layout='list'" />
              <UButton size="xs" square icon="i-heroicons-view-columns" :color="store.layout==='board' ? 'primary' : 'gray'" :variant="store.layout==='board' ? 'soft' : 'ghost'" @click="store.layout='board'" />
              <UButton size="xs" square icon="i-heroicons-calendar-days" :color="store.layout==='calendar' ? 'primary' : 'gray'" :variant="store.layout==='calendar' ? 'soft' : 'ghost'" @click="store.layout='calendar'" />
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
                <span class="h-2 w-2 rounded-full" :style="{ background: l.colorToken }" />{{ l.name }}
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
              <UButton class="ml-auto opacity-0 group-hover/col:opacity-100" icon="i-heroicons-plus" color="gray" variant="ghost" size="2xs" square @click="openCreate('issue')" />
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
                  <span v-for="l in store.labelsForIssue(i.id)" :key="l.id" class="flex items-center gap-1 rounded-full border border-subtle px-1.5 py-0.5 text-xs text-tertiary">
                    <span class="h-2 w-2 rounded-full" :style="{ background: l.colorToken }" />{{ l.name }}
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
            <UButton icon="i-heroicons-chevron-right" color="gray" variant="ghost" size="xs" square @click="monthOffset++" />
            <UButton class="ml-2" color="gray" variant="ghost" size="xs" @click="monthOffset = 0">today</UButton>
            <div class="ml-auto flex gap-1">
              <UButton size="2xs" :color="calMode === 'month' ? 'primary' : 'gray'" :variant="calMode === 'month' ? 'soft' : 'ghost'" @click="calMode = 'month'; monthOffset = 0">Month</UButton>
              <UButton size="2xs" :color="calMode === 'week' ? 'primary' : 'gray'" :variant="calMode === 'week' ? 'soft' : 'ghost'" @click="calMode = 'week'; monthOffset = 0">Week</UButton>
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

        <!-- LIST (Linear-стиль: плотные строки, edit в detail) -->
        <template v-else>
        <div v-for="g in store.issueGroups(selectedTeamId)" :key="g.key" class="group/sec">
          <!-- group header bar -->
          <div class="flex cursor-pointer items-center gap-2 bg-elevated/50 px-4 py-2.5 text-sm font-semibold hover:bg-elevated"
            @click="collapsedStatus[g.key] = !collapsedStatus[g.key]">
            <UIcon :name="collapsedStatus[g.key] ? 'i-heroicons-chevron-right' : 'i-heroicons-chevron-down'" class="text-tertiary" />
            <span v-if="g.kind === 'status'" class="inline-block shrink-0 w-4 text-center leading-none" :style="{ color: statusColor(g.category) }">{{ statusGlyph(g.category) }}</span>
            <span v-else-if="g.kind === 'priority'" class="inline-block shrink-0 w-4 text-center leading-none" :style="{ color: prioColor(g.prio) }">{{ prioGlyph(g.prio) }}</span>
            <UIcon v-else-if="g.kind === 'milestone'" name="i-heroicons-flag" class="text-tertiary" />
            <span v-else class="flex h-4 w-4 items-center justify-center rounded-full bg-elevated text-[9px]">{{ (g.label || '?')[0].toUpperCase() }}</span>
            <span>{{ g.label }}</span>
            <span class="font-normal text-tertiary">{{ g.items.length }}</span>
          </div>

          <template v-if="!collapsedStatus[g.key]">
            <div v-for="i in g.items" :key="i.id"
              class="group flex cursor-pointer items-center gap-3 border-b border-subtle/40 px-4 py-2.5 hover:bg-hover"
              :class="{ 'bg-active': selectedIssueId === i.id || selectedIds.has(i.id) }"
              @click="openIssue(i.id)">
              <UCheckbox :model-value="selectedIds.has(i.id)" class="shrink-0 transition-opacity" :class="selCount || selectedIds.has(i.id) ? 'opacity-100' : 'opacity-0 group-hover:opacity-100'" @click.stop @update:model-value="toggleSel(i.id)" />
              <span v-if="i.priority" class="inline-block shrink-0 w-4 text-center leading-none" :style="{ color: prioColor(i.priority) }">{{ prioGlyph(i.priority) }}</span>
              <UIcon v-else name="i-heroicons-minus" class="shrink-0 text-tertiary/50" />
              <span class="w-14 shrink-0 text-sm text-accent2">{{ i.identifier }}</span>
              <span class="inline-block shrink-0 w-4 text-center leading-none" :style="{ color: statusColor(store.states[i.stateId]?.category) }">{{ statusGlyph(store.states[i.stateId]?.category) }}</span>
              <span class="truncate">{{ i.title }}</span>
              <span class="flex flex-1 flex-wrap items-center gap-1">
                <span v-for="l in store.labelsForIssue(i.id)" :key="l.id" class="flex items-center gap-1 rounded-full border border-subtle px-1.5 py-0.5 text-xs text-tertiary">
                  <span class="h-2 w-2 rounded-full" :style="{ background: l.colorToken }" />{{ l.name }}
                </span>
              </span>
              <span v-if="i.dueDate || i.scheduledDate" class="shrink-0 text-sm tabular-nums text-tertiary">{{ String(i.scheduledDate || i.dueDate).slice(5, 10) }}</span>
              <span v-if="i.assigneeId" class="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-elevated text-[10px]" :title="store.userName(i.assigneeId)">{{ (store.userName(i.assigneeId) || '?')[0].toUpperCase() }}</span>
              <UButton icon="i-heroicons-trash" color="gray" variant="ghost" size="xs" square
                class="shrink-0 opacity-0 hover:opacity-100" title="Archive" @click.stop="store.archiveIssue(i.id)" />
            </div>
          </template>
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
        <div class="flex justify-end gap-2"><UButton size="2xs" color="gray" variant="ghost" @click="cycleEdit = false">Cancel</UButton><UButton size="2xs" @click="saveCycle">[ save ]</UButton></div>
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
          :ui="{ base: 'text-white' }"
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
              <UButton class="ml-auto" size="2xs" color="gray" variant="ghost" title="Save (⌘↵)" @click="saveDesc">Done</UButton>
            </div>
            <div class="relative">
              <MentionMenu :m="descMention" />
              <textarea ref="descEl" v-model="descDraft" rows="5"
                class="block w-full resize-y bg-transparent px-3 py-2 text-sm leading-relaxed text-secondary outline-none placeholder:text-tertiary"
                placeholder="Add description… (markdown, @mention, #ref)" @paste="onAttachPaste($event, 'issue', selectedIssue.id)" @input="descMention.detect()" @keydown="onDescKeydown"></textarea>
            </div>
          </div>
          <div v-else-if="selectedIssue.description?.md"
            class="prose-invert -mx-2 cursor-text rounded-md px-2 py-1.5 text-sm leading-relaxed text-secondary hover:bg-elevated [&_a]:text-accent [&_code]:rounded [&_code]:bg-elevated [&_code]:px-1 [&_h1]:mb-1 [&_h1]:mt-2 [&_h1]:text-base [&_h1]:font-semibold [&_h1]:text-white [&_h2]:mb-1 [&_h2]:mt-2 [&_h2]:font-semibold [&_h2]:text-white [&_li]:ml-4 [&_li]:list-disc [&_p]:mb-1.5 [&_strong]:text-white"
            @click="startDescEdit" v-html="renderMarkdown(selectedIssue.description.md)" />
          <button v-else class="-mx-2 flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-sm text-tertiary hover:bg-elevated" @click="startDescEdit">
            <UIcon name="i-heroicons-bars-3-bottom-left" /> Add description…
          </button>
        </div>

        <!-- Properties -->
        <div class="mb-2 text-xs font-medium uppercase tracking-wider text-tertiary"><span class="text-ok"># </span>Properties</div>
        <div class="mb-6 space-y-1">
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
            <USelectMenu class="flex-1 rounded-md transition-colors hover:bg-elevated" variant="none" :ui="propSelUi" placeholder="Unassigned" :model-value="selectedIssue.assigneeId || undefined" value-attribute="value" option-attribute="label"
              :options="[{ label: 'Unassigned', value: '' }, ...store.wsUsers.map((u) => ({ label: u.name || u.email, value: u.id })), ...(store.users[store.currentUserId] ? [{ label: (store.users[store.currentUserId].name || store.users[store.currentUserId].email) + ' (me)', value: store.currentUserId }] : [])]"
              @update:model-value="store.setIssueAssignee(selectedIssue.id, $event)" />
          </div>
          <div class="flex min-h-8 items-center gap-3">
            <span class="flex w-24 shrink-0 items-center gap-2 text-tertiary"><UIcon name="i-heroicons-arrow-path-rounded-square" /> Cycle</span>
            <USelectMenu class="flex-1 rounded-md transition-colors hover:bg-elevated" variant="none" :ui="propSelUi" placeholder="No cycle" :model-value="selectedIssue.cycleId || undefined" value-attribute="value" option-attribute="label"
              :options="[{ label: 'No cycle', value: '' }, ...store.teamCycles(selectedIssue.teamId).map((c) => ({ label: `Cycle ${c.number} (${c.status})`, value: c.id }))]"
              @update:model-value="store.setIssueCycle(selectedIssue.id, $event)" />
          </div>
          <div class="flex min-h-8 items-center gap-3">
            <span class="flex w-24 shrink-0 items-center gap-2 text-tertiary"><UIcon name="i-heroicons-flag" /> Milestone</span>
            <USelectMenu class="flex-1 rounded-md transition-colors hover:bg-elevated" variant="none" :ui="propSelUi" placeholder="No milestone" :model-value="selectedIssue.milestoneId || undefined" value-attribute="value" option-attribute="label"
              :options="[{ label: 'No milestone', value: '' }, ...store.teamMilestones(selectedIssue.teamId).map((m) => ({ label: m.name, value: m.id }))]"
              @update:model-value="store.setIssueMilestone(selectedIssue.id, $event)" />
          </div>
          <div class="flex min-h-8 items-center gap-3">
            <span class="flex w-24 shrink-0 items-center gap-2 text-tertiary"><UIcon name="i-heroicons-calendar" /> Due date</span>
            <DatePicker class="flex-1" placeholder="No due date" :model-value="selectedIssue.scheduledDate ? String(selectedIssue.scheduledDate).slice(0,10) : ''"
              @update:model-value="(v:any) => store.setIssueSchedule(selectedIssue.id, v)" />
          </div>
          <div class="flex min-h-8 items-center gap-3">
            <span class="flex w-24 shrink-0 items-center gap-2 text-tertiary"><UIcon name="i-heroicons-scale" /> Estimate</span>
            <UInput class="flex-1 rounded-md transition-colors hover:bg-elevated" variant="none" :ui="propSelUi" type="number" min="0" :model-value="selectedIssue.estimate ?? ''" placeholder="No estimate"
              @change="(v:any) => store.setIssueEstimate(selectedIssue.id, Number(v))" />
          </div>
          <!-- Time tracking -->
          <div class="flex min-h-8 items-center gap-3">
            <span class="flex w-24 shrink-0 items-center gap-2 text-tertiary"><UIcon name="i-heroicons-clock" /> Time</span>
            <span class="flex-1 text-sm" :class="store.issueTrackedMin(selectedIssue.id) ? 'text-white' : 'text-tertiary'">{{ store.issueTrackedMin(selectedIssue.id) ? fmtMin(store.issueTrackedMin(selectedIssue.id)) : 'Not tracked' }}</span>
            <UButton v-if="store.runningTimer?.issueId === selectedIssue.id" color="red" size="2xs" icon="i-heroicons-stop" @click="wrap(() => store.stopTimer())">Stop</UButton>
            <UButton v-else color="gray" size="2xs" icon="i-heroicons-play" @click="wrap(() => store.startTimer(selectedIssue.id))">Start</UButton>
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
            <div v-if="f.type==='checkbox'" class="flex-1 pl-2"><UCheckbox :model-value="!!store.cfValue(selectedIssue.id, f.id)"
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
          <div class="flex flex-wrap items-center gap-1.5">
            <span v-for="l in store.labelsForIssue(selectedIssue.id)" :key="l.id" class="group/lbl flex items-center gap-1 rounded-full border border-subtle px-2 py-0.5 text-xs">
              <span class="h-2 w-2 rounded-full" :style="{ background: l.colorToken }" />{{ l.name }}
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
          <UInput v-model="newSub" placeholder="+ Add sub-issue" size="xs" variant="none" class="w-full" @keyup.enter="addSub" />
        </div>

        <!-- Relations -->
        <div class="mb-4">
          <div class="mb-1.5 text-xs font-medium uppercase tracking-wider text-tertiary"><span class="text-ok"># </span>Relations</div>
          <div v-for="r in store.relationsForIssue(selectedIssue.id)" :key="r.id"
            class="group flex items-center gap-2 rounded-md px-2 py-1.5 hover:bg-hover">
            <span class="text-tertiary">{{ r.label }}</span>
            <span class="cursor-pointer hover:text-accent" @click="openIssue(r.otherId)">{{ store.issueById(r.otherId)?.identifier || '—' }}</span>
            <UButton class="ml-auto opacity-0 group-hover:opacity-100" icon="i-heroicons-x-mark" color="gray" variant="ghost" size="2xs" square @click="store.removeRelation(r.id)" />
          </div>
          <button class="mt-1 flex items-center gap-1 rounded-md px-2 py-1.5 text-sm text-tertiary hover:bg-hover hover:text-secondary" @click="showRelModal = true">
            <UIcon name="i-heroicons-plus" /> Add relation
          </button>
        </div>

        <!-- Attachments -->
        <div class="mb-5">
          <div class="mb-1.5 text-xs font-medium uppercase tracking-wider text-tertiary"><span class="text-ok"># </span>Attachments</div>
          <Attachments entity-type="issue" :entity-id="selectedIssue.id" />
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
                <div class="mt-1 flex gap-2"><UButton size="2xs" @click="saveComment">[ save ]</UButton><UButton size="2xs" color="gray" variant="ghost" @click="editCommentId = ''">Cancel</UButton></div>
              </div>
              <div v-else-if="c.deletedAt" class="text-sm italic text-tertiary">(deleted)</div>
              <div v-else class="prose-chat text-sm leading-relaxed" v-html="renderMarkdown(c.body)" />
              <Attachments v-if="store.attachmentsFor('comment', c.id).length" class="mt-1.5" entity-type="comment" :entity-id="c.id" readonly />
              <div v-if="reactionGroups(c).length" class="mt-1 flex flex-wrap gap-1">
                <button v-for="g in reactionGroups(c)" :key="g.emoji"
                  class="rounded-full border px-1.5 py-0.5 text-xs"
                  :class="g.mine ? 'border-accent bg-accent/15 text-accent' : 'border-subtle text-secondary hover:bg-hover'"
                  @click="commentReact(c, g.emoji)">{{ g.emoji }} {{ g.count }}</button>
              </div>
            </div>
          </div>
          <PendingAttachments v-if="commentPending.length" class="mt-2" :files="commentPending" @remove="(i) => commentPending = commentPending.filter((_, j) => j !== i)" />
          <div class="mt-2 flex items-end gap-2">
            <div class="relative flex-1">
              <MentionMenu :m="commentMention" />
              <textarea ref="commentEl" v-model="commentDraft" rows="1" placeholder="Add a comment… (markdown, @mention, #ref)"
                class="block w-full resize-none rounded-md border border-subtle bg-elevated px-2.5 py-1.5 text-sm outline-none placeholder:text-tertiary"
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
    <ResizeHandle v-if="store.threadParentId && activeSection === 'chats'" @down="threadPanel.onResizeStart" />
    <aside v-if="store.threadParentId && activeSection === 'chats'" :style="{ width: threadPanel.w.value + 'px' }" class="flex shrink-0 flex-col overflow-hidden rounded-xl border border-subtle bg-surface text-sm">
      <div class="flex shrink-0 items-center gap-2 border-b border-subtle px-4 py-3">
        <UIcon name="i-heroicons-chat-bubble-left-right" class="text-tertiary" />
        <span class="font-medium">Thread</span>
        <UButton class="ml-auto" icon="i-heroicons-x-mark" color="gray" variant="ghost" size="xs" square @click="store.closeThread()" />
      </div>
      <div class="flex-1 overflow-y-auto px-2 py-3">
        <!-- parent (full message: reactions/attachments, без reply-в-тред) -->
        <div v-if="threadParent" class="mb-1 border-b border-subtle pb-1">
          <ChatMessage :m="threadParent" :can-reply="false" />
        </div>
        <div class="mb-1 px-2 text-xs text-tertiary">{{ store.threadMessages.length }} {{ store.threadMessages.length === 1 ? 'reply' : 'replies' }}</div>
        <ChatMessage v-for="m in store.threadMessages" :key="m.id" :m="m" :can-reply="false" />
      </div>
      <div class="shrink-0 border-t border-subtle p-2">
        <PendingAttachments v-if="threadPending.length" class="mb-2" :files="threadPending" @remove="(i) => threadPending = threadPending.filter((_, j) => j !== i)" />
        <textarea v-model="threadDraft" rows="2" placeholder="Reply…" class="w-full resize-none rounded-md border border-subtle bg-elevated px-2 py-1.5 text-sm outline-none" @paste="onThreadPaste" @keydown.enter.exact.prevent="sendThread" />
        <div class="mt-1 flex items-center justify-end gap-2">
          <UButton icon="i-heroicons-paper-clip" color="gray" variant="ghost" size="2xs" square title="Attach" @click="threadFileEl?.click()" />
          <input ref="threadFileEl" type="file" multiple class="hidden" @change="onThreadFilePick" />
          <UButton size="2xs" icon="i-heroicons-paper-airplane" :disabled="!threadDraft.trim() && !threadPending.length" @click="sendThread">Reply</UButton>
        </div>
      </div>
    </aside>
    </div><!-- /content panel -->
    </div><!-- /flex flex-1 -->

    <!-- Bulk-бар (мультивыбор в List) -->
    <div v-if="selCount && selectedTeam" class="fixed bottom-6 left-1/2 z-[90] flex -translate-x-1/2 items-center gap-2 rounded-xl border border-subtle bg-elevated px-3 py-2 text-sm shadow-xl">
      <span class="px-1 font-medium">{{ selCount }} selected</span>
      <span class="h-4 w-px bg-subtle" />
      <USelectMenu placeholder="Status" size="xs" class="w-28" value-attribute="value" option-attribute="label"
        :options="store.teamStates(selectedTeamId).map((s2) => ({ label: s2.name, value: s2.id }))" @update:model-value="bulkStatus" />
      <USelectMenu placeholder="Assignee" size="xs" class="w-32" value-attribute="value" option-attribute="label"
        :options="[{ label: 'No assignee', value: '' }, ...store.wsUsers.map((u) => ({ label: u.name || u.email, value: u.id }))]" @update:model-value="bulkAssignee" />
      <USelectMenu v-if="store.teamLabels(selectedTeamId).length" placeholder="+ Label" size="xs" class="w-28" value-attribute="value" option-attribute="label"
        :options="store.teamLabels(selectedTeamId).map((l) => ({ label: l.name, value: l.id }))" @update:model-value="bulkLabel" />
      <UButton size="xs" color="gray" variant="ghost" icon="i-heroicons-archive-box" @click="bulkArchive">Archive</UButton>
      <UButton size="xs" color="gray" variant="ghost" icon="i-heroicons-x-mark" square @click="clearSel" />
    </div>

    <!-- Create-модалка (поля под выбранную сущность) -->
    <UModal v-model="createOpen" :ui="modalUi">
      <div class="p-4">
        <div class="mb-4 flex items-center gap-2">
          <span class="flex h-7 w-7 items-center justify-center rounded-md bg-elevated"><UIcon :name="createIcon" class="text-secondary" /></span>
          <span class="text-base font-semibold text-white"><span class="text-ok">$</span> New {{ createTypeLabel.toLowerCase() }}</span>
          <UButton class="ml-auto" icon="i-heroicons-x-mark" color="gray" variant="ghost" size="xs" square @click="createOpen = false" />
        </div>

        <!-- ISSUE -->
        <div v-if="createType === 'issue'" class="space-y-3">
          <div><label class="mb-1 block text-xs text-tertiary">Team</label>
            <USelectMenu v-model="cTeam" class="w-full" value-attribute="value" option-attribute="label"
              :options="store.teamList.map((t) => ({ label: `${t.key} · ${t.name}`, value: t.id }))" /></div>
          <div><label class="mb-1 block text-xs text-tertiary">Title</label>
            <UInput v-model="cTitle" placeholder="Issue title" autofocus @keyup.enter="submitCreate" /></div>
        </div>

        <!-- DOC -->
        <div v-else-if="createType === 'doc'" class="space-y-3">
          <div><label class="mb-1 block text-xs text-tertiary">Team</label>
            <USelectMenu v-model="cTeam" class="w-full" value-attribute="value" option-attribute="label"
              :options="store.teamList.map((t) => ({ label: `${t.key} · ${t.name}`, value: t.id }))" /></div>
          <div><label class="mb-1 block text-xs text-tertiary">Title</label>
            <UInput v-model="cTitle" placeholder="Document title" autofocus @keyup.enter="submitCreate" /></div>
        </div>

        <!-- CYCLE -->
        <div v-else-if="createType === 'cycle'" class="space-y-3">
          <div><label class="mb-1 block text-xs text-tertiary">Team</label>
            <USelectMenu v-model="cTeam" class="w-full" value-attribute="value" option-attribute="label"
              :options="store.teamList.map((t) => ({ label: `${t.key} · ${t.name}`, value: t.id }))" /></div>
          <div class="flex gap-2">
            <div class="flex-1"><label class="mb-1 block text-xs text-tertiary">Start</label><DatePicker v-model="cStart" block /></div>
            <div class="flex-1"><label class="mb-1 block text-xs text-tertiary">End</label><DatePicker v-model="cEnd" block /></div>
          </div>
        </div>

        <!-- MILESTONE -->
        <div v-else-if="createType === 'milestone'" class="space-y-3">
          <div><label class="mb-1 block text-xs text-tertiary">Team</label>
            <USelectMenu v-model="cTeam" class="w-full" value-attribute="value" option-attribute="label"
              :options="store.teamList.map((t) => ({ label: `${t.key} · ${t.name}`, value: t.id }))" /></div>
          <div><label class="mb-1 block text-xs text-tertiary">Name</label>
            <UInput v-model="cTitle" placeholder="Milestone name" autofocus @keyup.enter="submitCreate" /></div>
          <div><label class="mb-1 block text-xs text-tertiary">Target date</label>
            <DatePicker v-model="cMsDate" block /></div>
        </div>

        <!-- MEETING -->
        <div v-else-if="createType === 'meeting'" class="space-y-3">
          <div><label class="mb-1 block text-xs text-tertiary">Title</label>
            <UInput v-model="cTitle" placeholder="Meeting title" autofocus @keyup.enter="submitCreate" /></div>
          <div><label class="mb-1 block text-xs text-tertiary">When (optional)</label>
            <div class="flex items-center gap-2">
              <DatePicker v-model="cMeetDate" block class="flex-1" placeholder="Pick a date" />
              <TimePicker v-model="cMeetStart" class="w-28" />
              <span class="text-tertiary">–</span>
              <TimePicker v-model="cMeetEnd" :from="cMeetStart" class="w-28" />
            </div>
          </div>
          <div><label class="mb-1 block text-xs text-tertiary">Participants</label>
            <USelectMenu v-model="cMeetParts" multiple searchable searchable-placeholder="Search people…" placeholder="Select participants" class="w-full"
              value-attribute="value" option-attribute="label"
              :options="store.wsUsers.map((u) => ({ label: u.name || u.email, value: u.id }))" />
            <div class="mt-1 text-xs text-tertiary">You're the host; others are invited. You can start right away.</div>
          </div>
        </div>

        <!-- DASHBOARD -->
        <div v-else-if="createType === 'dashboard'" class="space-y-3">
          <div><label class="mb-1 block text-xs text-tertiary">Name</label>
            <UInput v-model="cTitle" placeholder="Dashboard name" autofocus @keyup.enter="submitCreate" /></div>
        </div>

        <!-- CHANNEL -->
        <div v-else-if="createType === 'channel'" class="space-y-3">
          <div><label class="mb-1 block text-xs text-tertiary">Channel name</label>
            <UInput v-model="cTitle" placeholder="general" autofocus @keyup.enter="submitCreate" /></div>
          <label class="flex cursor-pointer items-center gap-2 text-sm">
            <UToggle v-model="cPrivate" />
            <span class="flex items-center gap-1"><UIcon name="i-heroicons-lock-closed" class="text-tertiary" /> Private — invite-only</span>
          </label>
        </div>

        <!-- TEAM -->
        <div v-else-if="createType === 'team'" class="space-y-3">
          <div><label class="mb-1 block text-xs text-tertiary">Parent</label>
            <USelectMenu v-model="cParent" class="w-full" value-attribute="value" option-attribute="label"
              :options="[{ label: '— root team —', value: '' }, ...store.teamList.map((t) => ({ label: `under ${t.key}`, value: t.id }))]" /></div>
          <div class="flex gap-2">
            <div class="w-24"><label class="mb-1 block text-xs text-tertiary">Key</label><UInput v-model="cKey" placeholder="KEY" class="w-full" /></div>
            <div class="flex-1"><label class="mb-1 block text-xs text-tertiary">Name</label><UInput v-model="cName" placeholder="Team name" class="w-full" /></div>
          </div>
        </div>

        <!-- WORKSPACE -->
        <div v-else-if="createType === 'workspace'" class="space-y-3">
          <div><label class="mb-1 block text-xs text-tertiary">Slug</label><UInput v-model="cSlug" placeholder="acme" /></div>
          <div><label class="mb-1 block text-xs text-tertiary">Name</label><UInput v-model="cName" placeholder="Acme Inc." /></div>
        </div>

        <div class="mt-4 flex justify-end gap-2">
          <UButton color="gray" variant="ghost" @click="createOpen = false">Cancel</UButton>
          <UButton @click="submitCreate">[ create {{ createTypeLabel.toLowerCase() }} ]</UButton>
        </div>
      </div>
    </UModal>

    <!-- Add-widget модалка -->
    <UModal v-model="showWidget" :ui="modalUi">
      <div class="space-y-3 p-4">
        <span class="text-sm font-semibold text-white"><span class="text-ok">$</span> Add widget</span>
        <USelectMenu v-model="wType" :options="labelOpts(['note', 'stat', 'view', 'chart'])" value-attribute="value" option-attribute="label" />
        <UInput v-model="wTitle" placeholder="Title (optional)" />
        <UTextarea v-if="wType==='note'" v-model="wNote" placeholder="Text…" :rows="3" />
        <USelectMenu v-else-if="wType==='stat'" v-model="wStatPrio" value-attribute="value" option-attribute="label"
          :options="[{ label: 'all issues', value: '' }, ...PRIO.map((p, idx) => ({ label: `priority: ${p}`, value: String(idx) }))]" />
        <USelectMenu v-else-if="wType==='view'" v-model="wViewId" value-attribute="value" option-attribute="label"
          :options="[{ label: 'pick a view…', value: '' }, ...Object.values(store.views).filter((x) => !x.deletedAt).map((v) => ({ label: v.name, value: v.id }))]" />
        <div class="flex justify-end gap-2 pt-1">
          <UButton color="gray" variant="ghost" @click="showWidget = false">Cancel</UButton>
          <UButton icon="i-heroicons-plus" @click="addWidget(); showWidget = false">[ add ]</UButton>
        </div>
      </div>
    </UModal>

    <!-- Calendar event модалка -->
    <UModal v-model="showCalEvent" :ui="modalUi">
      <div class="space-y-3 p-4">
        <span class="text-sm font-semibold text-white"><span class="text-ok">$</span> {{ calForm.id ? 'Edit event' : calForm.kind === 'meeting' ? 'New meeting' : 'New event' }}</span>
        <UInput v-model="calForm.title" placeholder="Title" autofocus @keydown.enter="saveCalEvent" />
        <div class="flex items-center gap-2">
          <USelectMenu v-model="calForm.kind" class="w-32" :disabled="!!calForm.id" :options="calForm.id ? [{ label: 'Event', value: 'event' }, { label: 'Day off', value: 'dayoff' }] : [{ label: 'Event', value: 'event' }, { label: 'Day off', value: 'dayoff' }, { label: 'Meeting', value: 'meeting' }]" value-attribute="value" option-attribute="label" />
          <DatePicker v-model="calForm.date" block class="flex-1" />
        </div>
        <div v-if="!calForm.allDay || calForm.kind === 'meeting'" class="flex items-center gap-2">
          <TimePicker v-model="calForm.start" class="flex-1" />
          <span class="text-tertiary">–</span>
          <TimePicker v-model="calForm.end" :from="calForm.start" class="flex-1" />
        </div>
        <UCheckbox v-if="calForm.kind !== 'meeting'" v-model="calForm.allDay" label="All day" />
        <!-- повтор (не для митинга) -->
        <div v-if="calForm.kind !== 'meeting'" class="flex items-center gap-2">
          <USelectMenu v-model="calForm.recurrence" class="w-40"
            :options="[{ label: 'Does not repeat', value: 'none' }, { label: 'Daily', value: 'daily' }, { label: 'Weekly', value: 'weekly' }, { label: 'Monthly', value: 'monthly' }]"
            value-attribute="value" option-attribute="label" />
          <template v-if="calForm.recurrence !== 'none'">
            <span class="text-xs text-tertiary">until</span>
            <DatePicker v-model="calForm.recUntil" class="flex-1" placeholder="forever" />
          </template>
        </div>
        <!-- участники митинга (free/busy: видно занятость в сетке через тоггл календарей слева) -->
        <USelectMenu v-if="calForm.kind === 'meeting'" v-model="calForm.participants" multiple searchable placeholder="Invite participants…"
          :options="store.wsUsers.map((u) => ({ label: u.name || u.email, value: u.id }))" value-attribute="value" option-attribute="label" />
        <UTextarea v-if="calForm.kind !== 'meeting'" v-model="calForm.description" placeholder="Notes (optional)" :rows="2" />
        <div class="flex items-center gap-2 pt-1">
          <UButton v-if="calForm.id" color="red" variant="ghost" icon="i-heroicons-trash" @click="deleteCalEvent">Delete</UButton>
          <UButton class="ml-auto" color="gray" variant="ghost" @click="showCalEvent = false">Cancel</UButton>
          <UButton icon="i-heroicons-check" @click="saveCalEvent">[ {{ calForm.id ? 'save' : 'create' }} ]</UButton>
        </div>
      </div>
    </UModal>

    <!-- Ручная запись времени -->
    <UModal v-model="showTimeEntry" :ui="modalUi">
      <div class="space-y-3 p-4">
        <span class="text-sm font-semibold text-white"><span class="text-ok">$</span> Log time</span>
        <div class="flex items-center gap-2">
          <DatePicker v-model="teForm.date" block class="flex-1" />
          <TimePicker v-model="teForm.start" class="w-28" />
          <span class="text-tertiary">–</span>
          <TimePicker v-model="teForm.end" :from="teForm.start" class="w-28" />
        </div>
        <UInput v-model="teForm.description" placeholder="What did you work on? (optional)" @keydown.enter="saveTimeEntry" />
        <div class="flex justify-end gap-2 pt-1">
          <UButton color="gray" variant="ghost" @click="showTimeEntry = false">Cancel</UButton>
          <UButton color="gray" icon="i-heroicons-check" @click="saveTimeEntry">[ log ]</UButton>
        </div>
      </div>
    </UModal>

    <!-- iCal subscribe модалка -->
    <UModal v-model="showIcal" :ui="modalUi">
      <div class="space-y-3 p-4">
        <span class="text-sm font-semibold text-white"><span class="text-ok">$</span> Subscribe to your calendar</span>
        <p class="text-xs text-tertiary">Add this URL in Google Calendar / Apple Calendar (subscribe by URL). Read-only: your events, assigned deadlines and meetings. Keep it private — anyone with the link can view.</p>
        <div class="flex items-center gap-2">
          <UInput :model-value="icalUrl" readonly class="flex-1" :ui="{ base: 'font-mono text-xs' }" @focus="(e: any) => e.target.select()" />
          <UButton :icon="icalCopied ? 'i-heroicons-check' : 'i-heroicons-clipboard-document'" color="gray" @click="copyIcal">{{ icalCopied ? 'Copied' : 'Copy' }}</UButton>
        </div>
        <div class="flex items-center gap-2 pt-1">
          <UButton color="gray" variant="ghost" icon="i-heroicons-arrow-path" @click="rotateIcal">Regenerate</UButton>
          <span class="text-xs text-tertiary">Regenerating invalidates the old link.</span>
          <UButton class="ml-auto" color="gray" variant="ghost" @click="showIcal = false">Close</UButton>
        </div>
      </div>
    </UModal>

    <!-- Add label модалка -->
    <UModal v-model="showLabelModal" :ui="modalUi">
      <div v-if="selectedIssue" class="space-y-4 p-4">
        <div class="flex items-center gap-2">
          <UIcon name="i-heroicons-tag" class="text-tertiary" />
          <span class="text-base font-semibold text-white"><span class="text-ok">$</span> Add label</span>
          <UButton class="ml-auto" icon="i-heroicons-x-mark" color="gray" variant="ghost" size="xs" square @click="showLabelModal = false" />
        </div>
        <div>
          <div class="mb-1.5 text-xs font-medium uppercase tracking-wider text-tertiary"><span class="text-ok"># </span>Existing</div>
          <div class="flex flex-wrap gap-1.5">
            <button v-for="l in store.teamLabels(selectedIssue.teamId).filter((l) => !store.labelsForIssue(selectedIssue.id).some((x) => x.id === l.id))" :key="l.id"
              class="flex items-center gap-1 rounded-full border border-subtle px-2.5 py-1 text-sm hover:border-strong hover:bg-hover"
              @click="addLabelFromModal(l.id); showLabelModal = false">
              <span class="h-2 w-2 rounded-full" :style="{ background: l.colorToken }" />{{ l.name }}
            </button>
            <span v-if="!store.teamLabels(selectedIssue.teamId).filter((l) => !store.labelsForIssue(selectedIssue.id).some((x) => x.id === l.id)).length" class="text-sm text-tertiary">All labels added</span>
          </div>
        </div>
        <div class="border-t border-subtle pt-3">
          <div class="mb-1.5 text-xs font-medium uppercase tracking-wider text-tertiary"><span class="text-ok"># </span>Create new</div>
          <div class="flex gap-2">
            <UInput v-model="newLabelName" size="sm" placeholder="Label name" class="flex-1" @keyup.enter="createLabelInIssue(); showLabelModal = false" />
            <UButton icon="i-heroicons-plus" @click="createLabelInIssue(); showLabelModal = false">[ create ]</UButton>
          </div>
        </div>
      </div>
    </UModal>

    <!-- Add relation модалка -->
    <UModal v-model="showRelModal" :ui="modalUi">
      <div v-if="selectedIssue" class="space-y-4 p-4">
        <div class="flex items-center gap-2">
          <UIcon name="i-heroicons-link" class="text-tertiary" />
          <span class="text-base font-semibold text-white"><span class="text-ok">$</span> Add relation</span>
          <UButton class="ml-auto" icon="i-heroicons-x-mark" color="gray" variant="ghost" size="xs" square @click="showRelModal = false" />
        </div>
        <div>
          <label class="mb-1 block text-xs text-tertiary">Type</label>
          <USelectMenu v-model="relType" :options="labelOpts(REL_TYPES)" value-attribute="value" option-attribute="label" class="w-full" />
        </div>
        <div>
          <label class="mb-1 block text-xs text-tertiary">Target issue</label>
          <USelectMenu v-model="relTarget" searchable searchable-placeholder="Search issue…" placeholder="Select issue" value-attribute="value" option-attribute="label"
            :options="store.issuesForTeamAll(selectedIssue.teamId).filter((i) => i.id !== selectedIssue.id).map((i) => ({ label: `${i.identifier} · ${i.title}`, value: i.identifier }))" class="w-full" />
        </div>
        <div class="flex justify-end gap-2 pt-1">
          <UButton color="gray" variant="ghost" @click="showRelModal = false">Cancel</UButton>
          <UButton icon="i-heroicons-plus" :disabled="!relTarget" @click="addRel">[ add ]</UButton>
        </div>
      </div>
    </UModal>

    <!-- Statuses модалка -->
    <UModal v-model="showStates" :ui="modalUi">
      <div class="space-y-3 p-4">
        <div class="flex items-center gap-2">
          <UIcon name="i-heroicons-adjustments-horizontal" class="text-tertiary" />
          <span class="text-base font-semibold text-white"><span class="text-ok">$</span> Workflow statuses · {{ selectedTeam?.key }}</span>
          <UButton class="ml-auto" icon="i-heroicons-x-mark" color="gray" variant="ghost" size="xs" square @click="showStates = false" />
        </div>
        <div v-for="st in store.teamStates(selectedTeamId)" :key="st.id" class="flex items-center gap-2">
          <span class="inline-block shrink-0 w-4 text-center leading-none" :style="{ color: statusColor(st.category) }">{{ statusGlyph(st.category) }}</span>
          <UInput :model-value="st.name" size="sm" class="flex-1" @change="(v:any) => store.updateState(st.id, { name: v })" />
          <USelectMenu :model-value="st.category" :options="labelOpts(STATE_CATS)" value-attribute="value" option-attribute="label" size="sm" class="w-36" @update:model-value="(c) => store.updateState(st.id, { category: c })" />
          <UButton icon="i-heroicons-trash" color="gray" variant="ghost" size="2xs" square title="Delete (must be empty)" @click="delState(st.id)" />
        </div>
        <div class="flex items-center gap-2 border-t border-subtle pt-3">
          <UInput v-model="newStateName" placeholder="New status name" size="sm" class="flex-1" @keyup.enter="addState" />
          <USelectMenu v-model="newStateCat" :options="labelOpts(STATE_CATS)" value-attribute="value" option-attribute="label" size="sm" class="w-36" />
          <UButton icon="i-heroicons-plus" size="xs" square @click="addState" />
        </div>
        <p v-if="err" class="text-xs text-danger">{{ err }}</p>
      </div>
    </UModal>

    <!-- Workspace Settings (большая модалка с сайдбаром) -->
    <UModal v-model="showSettings" :prevent-close="showAuto" :ui="{ ...modalUi, width: 'sm:max-w-3xl', rounded: 'rounded-xl' }">
      <div class="flex h-[34rem] overflow-hidden rounded-xl">
        <!-- сайдбар табов -->
        <div class="w-48 shrink-0 overflow-y-auto border-r border-subtle bg-base p-2">
          <template v-for="(tabs, grp) in settingsGroups" :key="grp">
            <div class="mb-1 mt-2 px-2 text-xs font-medium uppercase tracking-wider text-tertiary"><span class="text-ok"># </span>{{ grp }}</div>
            <div v-for="t in tabs" :key="t.key"
              class="flex h-8 cursor-pointer items-center gap-2 rounded px-2 text-sm hover:bg-hover"
              :class="settingsTab === t.key ? 'bg-active text-white' : 'text-secondary'"
              @click="settingsTab = t.key as any"><UIcon :name="t.icon" class="text-tertiary" /> {{ t.label }}</div>
          </template>
        </div>
        <!-- контент -->
        <div class="flex-1 overflow-y-auto p-5">
          <!-- GENERAL -->
          <template v-if="settingsTab === 'general'">
            <div class="mb-4 flex items-center justify-between">
              <h2 class="text-base font-semibold text-white"><span class="text-ok">$</span> General</h2>
              <UButton icon="i-heroicons-x-mark" color="gray" variant="ghost" size="xs" square @click="showSettings = false" />
            </div>
            <div class="space-y-4 text-sm">
              <div>
                <label class="mb-1 block text-xs text-tertiary">Logo</label>
                <div class="flex items-center gap-3">
                  <Avatar :name="store.activeWorkspace?.name" :avatar-key="(store.activeWorkspace as any)?.logoKey" :size="48" />
                  <UButton color="gray" variant="ghost" size="xs" icon="i-heroicons-arrow-up-tray" @click="logoEl?.click()">Upload</UButton>
                  <UButton v-if="(store.activeWorkspace as any)?.logoKey" color="gray" variant="ghost" size="xs" @click="store.setLogo('')">Remove</UButton>
                  <input ref="logoEl" type="file" accept="image/*" class="hidden" @change="onLogoPick" />
                </div>
              </div>
              <div>
                <label class="mb-1 block text-xs text-tertiary">Workspace name</label>
                <div class="flex gap-2"><UInput v-model="wsNameDraft" class="flex-1" /><UButton :disabled="!wsNameDraft.trim() || wsNameDraft === store.activeWorkspace?.name" @click="saveWsName">[ save ]</UButton></div>
              </div>
              <div>
                <label class="mb-1 block text-xs text-tertiary">Slug</label>
                <UInput :model-value="store.activeWorkspace?.slug" disabled />
              </div>
              <div class="text-xs text-tertiary">ID: {{ store.activeWorkspaceId }}</div>
              <div class="mt-6 rounded-lg border border-danger/30 p-3">
                <div class="mb-1 text-sm font-medium text-danger">Danger zone</div>
                <p class="mb-2 text-xs text-tertiary">Deleting the workspace hides it and all its data. This action is irreversible.</p>
                <UButton color="red" variant="ghost" icon="i-heroicons-trash" @click="confirmDeleteWs">Delete workspace</UButton>
              </div>
            </div>
          </template>

          <!-- MEMBERS -->
          <template v-else-if="settingsTab === 'members'">
            <div class="mb-4 flex items-center justify-between">
              <h2 class="text-base font-semibold text-white"><span class="text-ok">$</span> Members</h2>
              <UButton icon="i-heroicons-x-mark" color="gray" variant="ghost" size="xs" square @click="showSettings = false" />
            </div>
            <div class="mb-4 flex gap-2">
              <UInput v-model="inviteEmail" placeholder="email to invite" class="flex-1" @keyup.enter="doInvite" />
              <USelectMenu v-model="inviteRole" :options="labelOpts(ROLES)" value-attribute="value" option-attribute="label" class="w-32" />
              <UButton icon="i-heroicons-paper-airplane" :disabled="!inviteEmail.trim()" @click="doInvite">[ invite ]</UButton>
            </div>
            <div v-if="inviteLink" class="mb-4 flex items-center gap-2 rounded-md bg-elevated px-2 py-1.5 text-xs">
              <UIcon name="i-heroicons-link" class="shrink-0 text-tertiary" />
              <span class="flex-1 truncate font-mono text-tertiary">{{ inviteLink }}</span>
              <UButton size="2xs" color="gray" @click="copyInviteLink">Copy</UButton>
            </div>
            <div v-for="m in store.workspaceMembers" :key="m.id" class="group flex items-center gap-2 rounded-md px-2 py-1.5 hover:bg-hover">
              <span class="flex h-7 w-7 items-center justify-center rounded-full bg-elevated text-xs">{{ (m.name || '?')[0].toUpperCase() }}</span>
              <span class="flex-1 truncate text-sm">{{ m.name }}<span v-if="m.userId === store.currentUserId" class="ml-1 text-xs text-tertiary">(you)</span></span>
              <USelectMenu :model-value="m.role" :options="labelOpts(ROLES)" value-attribute="value" option-attribute="label" size="xs" class="w-28" @update:model-value="(r) => store.updateMemberRole(m.userId, r)" />
              <UButton v-if="m.userId !== store.currentUserId" icon="i-heroicons-trash" color="red" variant="ghost" size="2xs" square class="opacity-0 group-hover:opacity-100" title="Remove" @click="store.removeMember(m.userId)" />
            </div>
            <template v-if="store.pendingInvites.length">
              <div class="mb-1 mt-3 text-xs font-medium uppercase tracking-wider text-tertiary"><span class="text-ok"># </span>Pending</div>
              <div v-for="iv in store.pendingInvites" :key="iv.id" class="flex items-center gap-2 px-2 py-1.5 text-sm text-tertiary">
                <UIcon name="i-heroicons-envelope" /> <span class="flex-1 truncate">{{ iv.email }}</span><span class="text-xs">{{ iv.role }}</span>
              </div>
            </template>
          </template>

          <!-- TEAMS -->
          <template v-else-if="settingsTab === 'teams'">
            <div class="mb-4 flex items-center justify-between">
              <h2 class="text-base font-semibold text-white"><span class="text-ok">$</span> Teams</h2>
              <UButton icon="i-heroicons-plus" size="xs" @click="showSettings = false; openCreate('team')">New team</UButton>
            </div>
            <div v-for="t in store.teamList" :key="t.id" class="group flex items-center gap-2 rounded-md px-2 py-2 hover:bg-hover">
              <UIcon name="i-heroicons-cube" class="text-tertiary" />
              <UInput :model-value="t.name" variant="none" :padded="false" class="flex-1 font-medium" @change="(v:any) => store.updateTeam(t.id, { name: v })" />
              <span class="text-xs text-tertiary">{{ t.key }}</span>
              <USelectMenu :model-value="t.parentId || ''" size="2xs" class="w-32" value-attribute="value" option-attribute="label"
                :options="[{ label: '— root —', value: '' }, ...store.teamList.filter((x) => x.id !== t.id).map((x) => ({ label: x.key, value: x.id }))]"
                @update:model-value="(p) => store.moveTeam(t.id, p)" />
              <UButton icon="i-heroicons-adjustments-horizontal" color="gray" variant="ghost" size="2xs" square title="Statuses" @click="selectedTeamId = t.id; showSettings = false; showStates = true" />
              <UButton icon="i-heroicons-trash" color="red" variant="ghost" size="2xs" square class="opacity-0 group-hover:opacity-100" title="Delete team" @click="confirmDeleteTeam(t)" />
            </div>
          </template>

          <!-- LABELS -->
          <template v-else-if="settingsTab === 'labels'">
            <div class="mb-4 flex items-center justify-between">
              <h2 class="text-base font-semibold text-white"><span class="text-ok">$</span> Labels</h2>
              <UButton icon="i-heroicons-x-mark" color="gray" variant="ghost" size="xs" square @click="showSettings = false" />
            </div>
            <div class="mb-4 flex items-center gap-2">
              <div class="flex gap-1">
                <button v-for="c in LABEL_COLORS" :key="c" class="h-5 w-5 rounded-full ring-offset-1 ring-offset-base" :class="newWsLabelColor === c ? 'ring-2 ring-white' : ''" :style="{ background: c }" @click="newWsLabelColor = c" />
              </div>
              <UInput v-model="newWsLabelName" placeholder="Label name" class="flex-1" @keyup.enter="addWsLabel" />
              <UButton icon="i-heroicons-plus" :disabled="!newWsLabelName.trim()" @click="addWsLabel">[ add ]</UButton>
            </div>
            <div v-for="l in allLabels" :key="l.id" class="group flex items-center gap-2 rounded-md px-2 py-1.5 hover:bg-hover">
              <div class="flex gap-0.5">
                <button v-for="c in LABEL_COLORS" :key="c" class="h-3.5 w-3.5 rounded-full" :class="l.colorToken === c ? 'ring-1 ring-white ring-offset-1 ring-offset-base' : ''" :style="{ background: c }" @click="store.updateLabel(l.id, { colorToken: c })" />
              </div>
              <UInput :model-value="l.name" variant="none" :padded="false" class="flex-1 text-sm" @change="(v:any) => store.updateLabel(l.id, { name: v })" />
              <span class="text-xs text-tertiary">{{ store.teams[l.teamId]?.key || 'workspace' }}</span>
              <UButton icon="i-heroicons-trash" color="red" variant="ghost" size="2xs" square class="opacity-0 group-hover:opacity-100" @click="store.deleteLabel(l.id)" />
            </div>
            <div v-if="!allLabels.length" class="py-6 text-center text-sm text-tertiary">No labels yet</div>
          </template>

          <!-- AUTOMATIONS -->
          <template v-else-if="settingsTab === 'automations'">
            <!-- редактор правила (inline, без вложенной модалки) -->
            <template v-if="showAuto">
              <div class="mb-4 flex items-center gap-2">
                <UButton icon="i-heroicons-arrow-left" color="gray" variant="ghost" size="xs" square @click="showAuto = false" />
                <h2 class="text-base font-semibold text-white"><span class="text-ok">$</span> {{ autoForm.id ? 'Edit rule' : 'New rule' }}</h2>
              </div>
              <div class="space-y-3">
                <UInput v-model="autoForm.name" placeholder="Rule name" autofocus />
                <div class="flex items-center gap-2">
                  <span class="w-16 shrink-0 text-xs text-tertiary">Team</span>
                  <USelectMenu v-model="autoForm.teamId" class="flex-1" :options="[{ value: '', label: 'All teams' }, ...store.teamList.map((t:any) => ({ value: t.id, label: t.name }))]" value-attribute="value" option-attribute="label" />
                </div>
                <div class="rounded-md border border-subtle p-2.5">
                  <div class="mb-1.5 text-xs font-medium uppercase tracking-wider text-tertiary"><span class="text-ok"># </span>When</div>
                  <div class="flex items-center gap-2">
                    <USelectMenu v-model="autoForm.event" class="flex-1" :options="AUTO_EVENTS" value-attribute="value" option-attribute="label" />
                    <USelectMenu v-if="autoForm.event === 'status_changed'" v-model="autoForm.statusTarget" class="flex-1" :options="statusTargetOpts" value-attribute="value" option-attribute="label" searchable />
                  </div>
                </div>
                <div class="rounded-md border border-subtle p-2.5">
                  <div class="mb-1.5 flex items-center">
                    <span class="text-xs font-medium uppercase tracking-wider text-tertiary"><span class="text-ok"># </span>Then</span>
                    <UButton class="ml-auto" icon="i-heroicons-plus" color="gray" variant="ghost" size="2xs" @click="addAutoAction">Add action</UButton>
                  </div>
                  <div v-for="(ac, i) in autoForm.actions" :key="i" class="mb-1.5 flex items-center gap-2">
                    <USelectMenu v-model="ac.type" class="w-36 shrink-0" :options="ACTION_TYPES" value-attribute="value" option-attribute="label" @update:model-value="onActionType(i)" />
                    <USelectMenu v-if="ac.type === 'set_status'" v-model="ac.stateId" class="flex-1" :options="stateOpts" value-attribute="value" option-attribute="label" searchable placeholder="state" />
                    <USelectMenu v-else-if="ac.type === 'set_assignee'" v-model="ac.userId" class="flex-1" :options="assigneeOpts" value-attribute="value" option-attribute="label" />
                    <USelectMenu v-else-if="ac.type === 'set_priority'" v-model="ac.priority" class="flex-1" :options="[0,1,2,3,4].map((p) => ({ value: p, label: PRIO[p] }))" value-attribute="value" option-attribute="label" />
                    <USelectMenu v-else-if="ac.type === 'add_label'" v-model="ac.labelId" class="flex-1" :options="labelOptsAuto" value-attribute="value" option-attribute="label" searchable placeholder="label" />
                    <template v-else-if="ac.type === 'post_message'">
                      <USelectMenu v-model="ac.channelId" class="w-32 shrink-0" :options="channelOptsAuto" value-attribute="value" option-attribute="label" placeholder="channel" />
                      <UInput v-model="ac.text" class="flex-1" placeholder="Message ({issue})" />
                    </template>
                    <UButton icon="i-heroicons-x-mark" color="gray" variant="ghost" size="2xs" square @click="autoForm.actions.splice(i, 1)" />
                  </div>
                  <div v-if="!autoForm.actions.length" class="py-2 text-center text-xs text-tertiary">No actions — add one.</div>
                </div>
                <div class="flex items-center gap-2 pt-1">
                  <UCheckbox v-model="autoForm.enabled" label="Enabled" />
                  <UButton class="ml-auto" color="gray" variant="ghost" @click="showAuto = false">Cancel</UButton>
                  <UButton color="gray" icon="i-heroicons-check" @click="saveAuto">[ {{ autoForm.id ? 'save' : 'create' }} ]</UButton>
                </div>
              </div>
            </template>

            <!-- список правил -->
            <template v-else>
              <div class="mb-4 flex items-center justify-between">
                <h2 class="text-base font-semibold text-white"><span class="text-ok">$</span> Automations</h2>
                <div class="flex items-center gap-2">
                  <UButton icon="i-heroicons-plus" color="gray" size="xs" @click="openAutoCreate">New rule</UButton>
                  <UButton icon="i-heroicons-x-mark" color="gray" variant="ghost" size="xs" square @click="showSettings = false" />
                </div>
              </div>
              <p class="mb-3 text-xs text-tertiary">When something happens to an issue, run actions automatically.</p>
              <div v-for="a in store.automationsList" :key="a.id" class="group mb-1.5 flex items-center gap-3 rounded-md border border-subtle px-3 py-2">
                <UToggle :model-value="a.enabled" @update:model-value="(v:any) => store.toggleAutomation(a.id, v)" />
                <div class="min-w-0 flex-1 cursor-pointer" @click="openAutoEdit(a)">
                  <div class="truncate text-sm font-medium" :class="a.enabled ? 'text-white' : 'text-tertiary'">{{ a.name }}</div>
                  <div class="truncate text-xs text-tertiary">{{ autoSummary(a) }}<span v-if="a.teamId"> · {{ store.teams[a.teamId]?.key }}</span></div>
                </div>
                <UButton icon="i-heroicons-pencil" color="gray" variant="ghost" size="2xs" square @click="openAutoEdit(a)" />
                <UButton icon="i-heroicons-trash" color="red" variant="ghost" size="2xs" square @click="store.deleteAutomation(a.id)" />
              </div>
              <div v-if="!store.automationsList.length" class="py-6 text-center text-sm text-tertiary">No automations yet. Create your first rule.</div>
            </template>
          </template>

          <!-- ACCOUNT -->
          <template v-else-if="settingsTab === 'account'">
            <div class="mb-4 flex items-center justify-between">
              <h2 class="text-base font-semibold text-white"><span class="text-ok">$</span> Account</h2>
              <UButton icon="i-heroicons-x-mark" color="gray" variant="ghost" size="xs" square @click="showSettings = false" />
            </div>
            <div class="mb-4 flex items-center gap-3">
              <Avatar :name="store.userName(store.currentUserId)" :avatar-key="store.users[store.currentUserId]?.avatarKey" :size="48" />
              <div class="flex-1">
                <div class="font-medium">{{ store.users[store.currentUserId]?.name || '—' }}</div>
                <div class="text-xs text-tertiary">{{ store.users[store.currentUserId]?.email }}</div>
              </div>
              <UButton color="gray" variant="ghost" size="xs" icon="i-heroicons-arrow-up-tray" @click="avatarEl?.click()">Upload</UButton>
              <UButton v-if="store.users[store.currentUserId]?.avatarKey" color="gray" variant="ghost" size="xs" @click="store.setAvatar('')">Remove</UButton>
              <input ref="avatarEl" type="file" accept="image/*" class="hidden" @change="onAvatarPick" />
            </div>
            <UButton color="red" variant="ghost" icon="i-heroicons-arrow-right-on-rectangle" @click="logout">Log out</UButton>
          </template>

          <!-- APPEARANCE -->
          <template v-else-if="settingsTab === 'appearance'">
            <div class="mb-4 flex items-center justify-between">
              <h2 class="text-base font-semibold text-white"><span class="text-ok">$</span> Appearance</h2>
              <UButton icon="i-heroicons-x-mark" color="gray" variant="ghost" size="xs" square @click="showSettings = false" />
            </div>
            <div class="mb-1 text-xs text-tertiary">Density</div>
            <div class="flex gap-2">
              <button v-for="d in ['compact', 'default', 'comfortable']" :key="d"
                class="rounded-md border px-3 py-2 text-sm capitalize"
                :class="density === d ? 'border-accent bg-active text-white' : 'border-subtle bg-elevated text-tertiary hover:bg-hover'"
                @click="applyDensity(d)">{{ d }}</button>
            </div>
            <p class="mt-2 text-xs text-tertiary">Scales the whole interface.</p>
          </template>

          <!-- NOTIFICATIONS -->
          <template v-else-if="settingsTab === 'notifications'">
            <div class="mb-4 flex items-center justify-between">
              <h2 class="text-base font-semibold text-white"><span class="text-ok">$</span> Notifications</h2>
              <UButton icon="i-heroicons-x-mark" color="gray" variant="ghost" size="xs" square @click="showSettings = false" />
            </div>
            <p class="mb-3 text-xs text-tertiary">Which notifications to show in the Inbox.</p>
            <div v-for="p in NOTIF_PREF_LIST" :key="p.key" class="flex items-center gap-2 rounded-md px-2 py-2 hover:bg-hover">
              <UCheckbox :model-value="notifPrefs[p.key] !== false" @update:model-value="toggleNotifPref(p.key)" />
              <span class="text-sm">{{ p.label }}</span>
            </div>
          </template>

          <!-- SECTIONS (sidebar) -->
          <template v-else-if="settingsTab === 'sections'">
            <div class="mb-4 flex items-center justify-between">
              <h2 class="text-base font-semibold text-white"><span class="text-ok">$</span> Sidebar sections</h2>
              <UButton icon="i-heroicons-x-mark" color="gray" variant="ghost" size="xs" square @click="showSettings = false" />
            </div>
            <p class="mb-3 text-xs text-tertiary">Toggle rail sections and set their order.</p>
            <div v-for="(s, i) in cfgList" :key="s.key" class="flex items-center gap-2 rounded-md px-2 py-1.5 hover:bg-hover">
              <UCheckbox :model-value="s.visible" @update:model-value="toggleSection(i)" />
              <UIcon :name="s.icon" class="text-tertiary" />
              <span class="flex-1 text-sm">{{ s.label }}</span>
              <UButton icon="i-heroicons-chevron-up" color="gray" variant="ghost" size="2xs" square :disabled="i === 0" @click="moveSection(i, -1)" />
              <UButton icon="i-heroicons-chevron-down" color="gray" variant="ghost" size="2xs" square :disabled="i === cfgList.length - 1" @click="moveSection(i, 1)" />
            </div>
          </template>
        </div>
      </div>
    </UModal>

    <!-- Members модалка -->
    <UModal v-model="showMembers" :ui="modalUi">
      <div class="space-y-3 p-4">
        <div class="flex items-center gap-2">
          <UIcon name="i-heroicons-user-group" class="text-tertiary" />
          <span class="text-base font-semibold text-white"><span class="text-ok">$</span> Members</span>
          <UButton class="ml-auto" icon="i-heroicons-x-mark" color="gray" variant="ghost" size="xs" square @click="showMembers = false" />
        </div>
        <!-- invite -->
        <div class="flex gap-2">
          <UInput v-model="inviteEmail" placeholder="email to invite" class="flex-1" @keyup.enter="doInvite" />
          <USelectMenu v-model="inviteRole" :options="labelOpts(ROLES)" value-attribute="value" option-attribute="label" class="w-32" />
          <UButton icon="i-heroicons-paper-airplane" :disabled="!inviteEmail.trim()" @click="doInvite">[ invite ]</UButton>
        </div>
        <div v-if="inviteLink" class="flex items-center gap-2 rounded-md bg-elevated px-2 py-1.5 text-xs">
          <UIcon name="i-heroicons-link" class="shrink-0 text-tertiary" />
          <span class="flex-1 truncate font-mono text-tertiary">{{ inviteLink }}</span>
          <UButton size="2xs" color="gray" @click="copyInviteLink">Copy</UButton>
        </div>
        <!-- members list -->
        <div class="max-h-72 overflow-y-auto">
          <div class="mb-1 text-xs font-medium uppercase tracking-wider text-tertiary"><span class="text-ok"># </span>Members ({{ store.workspaceMembers.length }})</div>
          <div v-for="m in store.workspaceMembers" :key="m.id" class="flex items-center gap-2 rounded-md px-2 py-1.5 hover:bg-hover">
            <span class="flex h-7 w-7 items-center justify-center rounded-full bg-elevated text-xs">{{ (m.name || '?')[0].toUpperCase() }}</span>
            <span class="flex-1 truncate text-sm">{{ m.name }}</span>
            <USelectMenu :model-value="m.role" :options="labelOpts(ROLES)" value-attribute="value" option-attribute="label" size="xs" class="w-28" @update:model-value="(r) => store.updateMemberRole(m.userId, r)" />
          </div>
          <template v-if="store.pendingInvites.length">
            <div class="mb-1 mt-3 text-xs font-medium uppercase tracking-wider text-tertiary"><span class="text-ok"># </span>Pending invites</div>
            <div v-for="iv in store.pendingInvites" :key="iv.id" class="flex items-center gap-2 px-2 py-1.5 text-sm text-tertiary">
              <UIcon name="i-heroicons-envelope" /> <span class="flex-1 truncate">{{ iv.email }}</span><span class="text-xs">{{ iv.role }}</span>
            </div>
          </template>
        </div>
      </div>
    </UModal>

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
