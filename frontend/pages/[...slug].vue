<script setup lang="ts">
import { isAuthed, clearAuth } from '~/lib/token'
import { fileViewUrl } from '~/lib/upload'
import { useAppStore } from '~/stores/app'
import { PRIO, labelOpts, fmtD, d10, localDate, localHM, colorForUser } from '~/lib/ui'
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
const showLabelModal = ref(false)
const showRelModal = ref(false)
// описание/AI-подсказки/комменты/активность — в PanelsIssueDetailPanel (issue-id проп)
function openIssue(id: string) {
  selectedIssueId.value = id; store.loadComments(id); store.loadActivity(id)
}
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

// burndown/breakdown/edit-форма цикла — в PanelsCycleInsightsPanel; здесь только сброс фильтров при закрытии
function clearBreakdownFilters() {
  for (const fld of ['assignee', 'priority', 'milestone', 'team']) store.setFieldPredicate(fld, null)
  store.labelFilter = []
}

// --- Dashboards (M6 ч.2) ---
const selectedDashboardId = ref('')
const selectedDashboard = computed(() => store.dashboards[selectedDashboardId.value])
const newDashboard = ref('')
function openDashboard(id: string) { selectedDashboardId.value = id }
const createDash = () => wrap(async () => { if (newDashboard.value.trim()) { const id = await store.createDashboard(newDashboard.value); newDashboard.value = ''; openDashboard(id) } })
// add-widget form (грид виджетов — в SectionsDashboardSection)
const showWidget = ref(false)
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
// copy-id/duplicate/archive-меню задачи — в PanelsIssueDetailPanel

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
// Сетка (месяц/неделя/день, drag-create/move/resize) — в SectionsCalendarSection.
// Здесь — роутовый calAnchor, выбор календарей (сайдбар) и модалка события.
const calAnchor = ref('') // фокус-дата 'YYYY-MM-DD'
if (!calAnchor.value) calAnchor.value = fmtD(new Date())
// какие календари показывать (userId). По умолчанию — только мой; чужие включаешь тогглом.
const calUsers = ref<string[]>([])
watch(() => store.currentUserId, (id) => { if (id && !calUsers.value.length) calUsers.value = [id] }, { immediate: true })
function toggleCalUser(id: string) {
  calUsers.value = calUsers.value.includes(id) ? calUsers.value.filter((x) => x !== id) : [...calUsers.value, id]
}
// drag-create в тайм-сетке → секция отдаёт готовый диапазон
function onCalCreateAt(p: { date: string; start: string; end: string }) {
  Object.assign(calForm, { id: '', title: '', description: '', kind: 'event', participants: [], date: p.date, start: p.start, end: p.end, allDay: false, recurrence: 'none', recUntil: '' })
  showCalEvent.value = true
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
// Стейт/загрузка — в useReport (синглтон): сайдбар здесь, таблица — в SectionsReportsSection.
const { repGroup, repUser, repPreset, repFrom, repTo, setRange } = useReport()
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
// ресайз правых панелей — внутри самих панелей (PanelsIssueDetailPanel/PanelsCycleInsightsPanel/SectionsThreadPanel)
const joinScheduled = (mid: string) => wrap(() => meeting.join(mid))

// --- Chats ---
// композер/шапка/скролл/хадлы живут в SectionsChatSection, тред — в SectionsThreadPanel.
// Здесь — открытие каналов (зовётся из сайдбара/палитры/роутера/уведомлений) + unread-разделитель.
const currentChannel = computed(() => store.channels[store.activeChannelId])
const chatSection = ref<{ scrollBottom: () => void; scrollToUnread: () => void } | null>(null)
// inline @/# пикеры (юзеры + сущности); desc/comment-пикеры — в PanelsIssueDetailPanel
const docMention = useMention(docDraft, docEl)
const docFileEl = ref<HTMLInputElement | null>(null)
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
// reactions на комменты задачи — в PanelsIssueDetailPanel; чат-сообщения рендерит ChatMessage
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

      <!-- CALENDAR: месяц-сетка + week/day тайм-сетка -->
      <SectionsCalendarSection v-else-if="activeSection === 'calendar'"
        v-model:anchor="calAnchor" :cal-users="calUsers"
        @create="openCalCreate" @create-at="onCalCreateAt" @edit-event="openCalEdit"
        @open-issue="openIssue" @open-meetings="activeSection = 'meetings'" @ical="openIcal" />

      <!-- REPORTS: учёт времени -->
      <SectionsReportsSection v-else-if="activeSection === 'reports'" />

      <!-- DASHBOARD: грид виджетов -->
      <SectionsDashboardSection v-else-if="activeSection === 'dashboard'"
        :dashboard-id="selectedDashboardId" @add-widget="showWidget = true" />

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
    <PanelsCycleInsightsPanel v-if="store.cycles[selectedCycleId] && !selectedIssue" :cycle-id="selectedCycleId" @close="closeCycle" />

    <!-- Issue detail panel (M4) -->
    <PanelsIssueDetailPanel v-if="selectedIssue" :issue-id="selectedIssueId" :link-copied="linkCopied"
      @open-issue="openIssue" @copy-link="copyLink" @close="selectedIssueId = ''"
      @add-label="showLabelModal = true" @add-relation="showRelModal = true" @log-time="openTimeEntry" />

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
