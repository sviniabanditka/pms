import { defineStore } from 'pinia'
import { get, set } from 'idb-keyval'
import { useRuntimeConfig } from '#imports'
import { useClients } from '~/lib/connect'
import { connectWs } from '~/lib/ws'
import { uploadFile } from '~/lib/upload'
import { getUser, handleUnauthorized } from '~/lib/token'
import { match, type FilterNode } from '~/lib/filter'
import { applyChunk as applyChunkCore, emptyBuckets } from '~/lib/sync'
import type { Entity } from '~/types/entities'

// Метки направления связи (прямое / зеркальное).
const REL_FWD: Record<string, string> = { blocks: 'blocks', blocked_by: 'blocked by', related: 'related', duplicate: 'duplicate', duplicate_of: 'duplicate of' }
const REL_INV: Record<string, string> = { blocks: 'blocked by', blocked_by: 'blocks', related: 'related', duplicate: 'duplicate of', duplicate_of: 'duplicate' }

const IDB_KEY = 'pms-store'
let ws: { close: () => void } | null = null
let refreshTimer: any = null
const pendingTypes = new Set<string>() // entity_type'ы, изменённые WS-дельтами в debounce-окне
let bootstrapEpoch = 0 // конкурентные bootstrap'ы: чанки устаревшего стрима отбрасываются

export const useAppStore = defineStore('app', {
  state: () => ({
    // 31 сущностный бакет (Record<id, T>), типизированы через Entities — см. types/entities.ts и lib/sync.ts
    ...emptyBuckets(),
    messages: {} as Record<string, any[]>, // channelId -> messages (DTO)
    hasMoreMessages: {} as Record<string, boolean>,
    activeChannelId: '' as string,
    threadParentId: '' as string,
    threadMessages: [] as any[],
    typing: {} as Record<string, Record<string, number>>, // channelId -> userId -> ts
    comments: {} as Record<string, any[]>, // issueId -> comments
    activity: {} as Record<string, any[]>, // issueId -> change_log rows
    labelFilter: [] as string[], // активный фильтр по лейблам (issuesFor)
    groupBy: 'status' as 'status' | 'priority' | 'assignee' | 'milestone', // группировка списка
    currentUserId: '' as string,
    loaded: false as boolean, // первый bootstrap завершён
    myWorkspaces: [] as Entity[],
    cursor: 0,
    activeWorkspaceId: '',
    syncing: false,
    // M3: Filter AST + layout (применяются к List/Board)
    activeFilter: null as FilterNode | null,
    layout: 'list' as 'list' | 'board' | 'calendar',
  }),

  getters: {
    activeWorkspace: (s) => s.workspaces[s.activeWorkspaceId],
    // конфиг разделов рейла текущего пользователя
    mySectionsConfig: (s) => {
      const m = Object.values(s.members).find((x) => x.userId === s.currentUserId)
      return (m?.sectionsConfig as any) || {}
    },
    // --- Роли/доступы текущего пользователя (UX-гейтинг; бэк энфорсит независимо) ---
    myRole: (s): string => {
      const m = Object.values(s.members).find((x) => x.userId === s.currentUserId)
      return (m?.role as string) || ''
    },
    isWsAdmin(): boolean { return this.myRole === 'owner' || this.myRole === 'admin' },
    isWsOwner(): boolean { return this.myRole === 'owner' },
    isGuest(): boolean { return this.myRole === 'guest' },
    canWrite(): boolean { return this.myRole !== '' && this.myRole !== 'guest' },
    teamMembersOf: (s) => (teamId: string) =>
      Object.values(s.teamMembers).filter((tm: any) => tm.teamId === teamId),
    // управлять командой может lead команды-или-ancestor (по ltree-path) либо ws-admin
    canManageTeam(): (teamId: string) => boolean {
      return (teamId: string) => {
        if (this.isWsAdmin) return true
        const team: any = this.teams[teamId]
        if (!team) return false
        const path = String(team.path || '')
        return Object.values(this.teamMembers).some((tm: any) => {
          if (tm.userId !== this.currentUserId || tm.role !== 'lead') return false
          const anc: any = this.teams[tm.teamId]
          if (!anc) return false
          const ap = String(anc.path || '')
          return path === ap || path.startsWith(ap + '.')
        })
      }
    },
    teamList: (s) =>
      Object.values(s.teams).filter((t) => !t.deletedAt).sort((a, b) => String(a.path || '').localeCompare(String(b.path || ''))),
    teamStates: (s) => (teamId: string) =>
      Object.values(s.states)
        .filter((x) => x.teamId === teamId)
        .sort((a, b) => String(a.position).localeCompare(String(b.position))),
    // --- Кэш-индексы (немемоизируемые геттеры → Pinia/Vue их кэширует,
    // пересчёт только при изменении issues/issueLabels). Параметрик-геттеры читают
    // из них O(1) вместо Object.values(all).filter() на каждый вызов. ---
    activeIssuesByTeam(state): Map<string, any[]> {
      const m = new Map<string, any[]>()
      for (const i of Object.values(state.issues) as any[]) {
        if (i.archivedAt || i.deletedAt) continue
        let a = m.get(i.teamId); if (!a) { a = []; m.set(i.teamId, a) }
        a.push(i)
      }
      return m
    },
    labelsByIssue(state): Map<string, Set<string>> {
      const m = new Map<string, Set<string>>()
      for (const il of Object.values(state.issueLabels) as any[]) {
        let set = m.get(il.issueId); if (!set) { set = new Set(); m.set(il.issueId, set) }
        set.add(il.labelId)
      }
      return m
    },
    issuesByMilestone(state): Map<string, any[]> {
      const m = new Map<string, any[]>()
      for (const i of Object.values(state.issues) as any[]) {
        if (i.archivedAt || !i.milestoneId) continue
        let a = m.get(i.milestoneId); if (!a) { a = []; m.set(i.milestoneId, a) }
        a.push(i)
      }
      return m
    },
    issuesByParent(state): Map<string, any[]> {
      const m = new Map<string, any[]>()
      for (const i of Object.values(state.issues) as any[]) {
        if (i.archivedAt || !i.parentId) continue
        let a = m.get(i.parentId); if (!a) { a = []; m.set(i.parentId, a) }
        a.push(i)
      }
      return m
    },
    cfValuesByIssue(state): Map<string, Map<string, any>> {
      const m = new Map<string, Map<string, any>>()
      for (const v of Object.values(state.customFieldValues) as any[]) {
        let f = m.get(v.issueId); if (!f) { f = new Map(); m.set(v.issueId, f) }
        f.set(v.fieldId, v.value)
      }
      return m
    },
    relationsByIssue(state): Map<string, { id: string; label: string; otherId: string }[]> {
      const m = new Map<string, { id: string; label: string; otherId: string }[]>()
      const add = (k: string, e: { id: string; label: string; otherId: string }) => {
        let a = m.get(k); if (!a) { a = []; m.set(k, a) }
        a.push(e)
      }
      for (const r of Object.values(state.relations) as any[]) {
        add(r.sourceId, { id: r.id, label: REL_FWD[r.type] || r.type, otherId: r.targetId })
        add(r.targetId, { id: r.id, label: REL_INV[r.type] || r.type, otherId: r.sourceId })
      }
      return m
    },
    cyclesByTeam(state): Map<string, any[]> {
      const m = new Map<string, any[]>()
      for (const c of Object.values(state.cycles) as any[]) {
        let a = m.get(c.teamId); if (!a) { a = []; m.set(c.teamId, a) }
        a.push(c)
      }
      for (const a of m.values()) a.sort((x, y) => String(x.startsOn).localeCompare(String(y.startsOn)))
      return m
    },
    // активные issues в статусе, отфильтрованные Filter AST, по board_position
    issuesFor(): (teamId: string, stateId: string) => any[] {
      return (teamId: string, stateId: string) => {
        const lf = this.labelFilter, lbi = this.labelsByIssue
        const byLabel = (x: any) => { if (!lf.length) return true; const ids = lbi.get(x.id); return !!ids && lf.some((id) => ids.has(id)) }
        return (this.activeIssuesByTeam.get(teamId) || [])
          .filter((x) => x.stateId === stateId && match(this.activeFilter, x) && byLabel(x))
          .sort((a, b) => String(a.boardPosition || '').localeCompare(String(b.boardPosition || '')))
      }
    },
    teamViews: (s) => (teamId: string) =>
      Object.values(s.views).filter((v) => v.teamId === teamId && !v.deletedAt),
    // группы списка по выбранному измерению (status/priority/assignee)
    issueGroups(): (teamId: string) => any[] {
      return (teamId: string) => {
        const PL = ['None', 'Low', 'Med', 'High', 'Urgent']
        const lf = this.labelFilter, lbi = this.labelsByIssue
        const byLabel = (x: any) => { if (!lf.length) return true; const ids = lbi.get(x.id); return !!ids && lf.some((id) => ids.has(id)) }
        const all = (this.activeIssuesByTeam.get(teamId) || [])
          .filter((x) => match(this.activeFilter, x) && byLabel(x))
          .sort((a, b) => String(a.boardPosition || '').localeCompare(String(b.boardPosition || '')))
        if (this.groupBy === 'priority') {
          const by = [0, 1, 2, 3, 4].map(() => [] as any[])
          for (const i of all) by[i.priority]?.push(i)
          return [4, 3, 2, 1, 0].map((p) => ({ key: 'p' + p, kind: 'priority', label: PL[p], prio: p, items: by[p] }))
        }
        if (this.groupBy === 'assignee') {
          const by = new Map<string, any[]>()
          for (const i of all) { const k = i.assigneeId || 'none'; let a = by.get(k); if (!a) { a = []; by.set(k, a) } a.push(i) }
          const g = Array.from(by.keys()).filter((k) => k !== 'none').map((u) => ({ key: u, kind: 'assignee', label: this.users[u]?.name || this.users[u]?.email || '—', items: by.get(u)! }))
          g.push({ key: 'none', kind: 'assignee', label: 'No assignee', items: by.get('none') || [] })
          return g
        }
        if (this.groupBy === 'milestone') {
          const by = new Map<string, any[]>()
          for (const i of all) { const k = i.milestoneId || 'none'; let a = by.get(k); if (!a) { a = []; by.set(k, a) } a.push(i) }
          const ms = Object.values(this.milestones).filter((m: any) => m.teamId === teamId && !m.deletedAt) as any[]
          const g = ms.map((m) => ({ key: m.id, kind: 'milestone', label: m.name, items: by.get(m.id) || [] }))
          g.push({ key: 'none', kind: 'milestone', label: 'No milestone', items: by.get('none') || [] })
          return g.filter((x) => x.items.length)
        }
        // status — группировка одним проходом (раньше O(states×issues))
        const states = Object.values(this.states).filter((st: any) => st.teamId === teamId)
          .sort((a: any, b: any) => String(a.position).localeCompare(String(b.position))) as any[]
        const by = new Map<string, any[]>(states.map((st) => [st.id, [] as any[]]))
        for (const i of all) { const a = by.get(i.stateId); if (a) a.push(i) }
        return states.map((st) => ({ key: st.id, kind: 'status', label: st.name, category: st.category, stateId: st.id, items: by.get(st.id)! }))
      }
    },
    issueById: (s) => (id: string) => s.issues[id],
    issueByIdentifier: (s) => (ident: string) => Object.values(s.issues).find((i) => i.identifier === ident),
    teamByKey: (s) => (key: string) => Object.values(s.teams).find((t) => t.key === key),
    issuesForTeamAll: (s) => (teamId: string) =>
      Object.values(s.issues).filter((i) => i.teamId === teamId && !i.archivedAt),
    // персональные срезы по всем командам
    personalIssues: (s) => (mode: string) => {
      const all = Object.values(s.issues).filter((i) => !i.archivedAt && !i.deletedAt)
      if (mode === 'assigned') return all.filter((i) => i.assigneeId === s.currentUserId)
      if (mode === 'created') return all.filter((i) => i.createdBy === s.currentUserId)
      if (mode === 'active') return all.filter((i) => { const c = s.states[i.stateId]?.category; return c !== 'completed' && c !== 'canceled' })
      return all
    },
    stateCategory: (s) => (stateId: string) => s.states[stateId]?.category as string | undefined,
    issuesInCycle: (s) => (cycleId: string) =>
      Object.values(s.issues).filter((i) => i.cycleId === cycleId && !i.archivedAt),
    // метрики цикла (live-расчёт из стора)
    cycleStats: (s) => (cycleId: string) => {
      const iss = Object.values(s.issues).filter((i) => i.cycleId === cycleId && !i.archivedAt)
      const cat = (id: string) => s.states[id]?.category
      const pts = (arr: any[]) => arr.reduce((n, i) => n + (Number(i.estimate) || 0), 0)
      const done = iss.filter((i) => cat(i.stateId) === 'completed')
      return {
        scope: iss.length,
        started: iss.filter((i) => cat(i.stateId) === 'started').length,
        completed: done.length,
        scopePoints: pts(iss),
        donePoints: pts(done),
      }
    },
    // velocity: средние завершённые points по прошедшим (completed) циклам команды
    teamVelocity: (s) => (teamId: string) => {
      const cat = (id: string) => s.states[id]?.category
      const cyc = Object.values(s.cycles).filter((c) => c.teamId === teamId && c.status === 'completed')
      if (!cyc.length) return 0
      let total = 0
      for (const c of cyc) {
        total += Object.values(s.issues)
          .filter((i) => i.cycleId === c.id && !i.archivedAt && cat(i.stateId) === 'completed')
          .reduce((n, i) => n + (Number(i.estimate) || 0), 0)
      }
      return Math.round(total / cyc.length)
    },
    // issues команды на конкретную дату (scheduled, иначе due), с учётом фильтра
    calendarIssues: (s) => (teamId: string, ds: string) =>
      Object.values(s.issues).filter((i) => {
        if (i.teamId !== teamId || i.archivedAt || !match(s.activeFilter, i)) return false
        const d = i.scheduledDate ? String(i.scheduledDate).slice(0, 10) : i.dueDate ? String(i.dueDate).slice(0, 10) : ''
        return d === ds
      }),
    // поля команды: workspace-scope (teamId пуст) + team-scope, не архивные
    fieldsForTeam: (s) => (teamId: string) =>
      Object.values(s.customFields).filter((f) => !f.archivedAt && (!f.teamId || f.teamId === teamId)),
    cfValue(): (issueId: string, fieldId: string) => any {
      const idx = this.cfValuesByIssue
      return (issueId, fieldId) => idx.get(issueId)?.get(fieldId)
    },
    workspaceDashboards: (s) => Object.values(s.dashboards).filter((d) => !d.deletedAt),
    widgetsOf: (s) => (dashboardId: string) =>
      Object.values(s.dashboardWidgets).filter((w) => w.dashboardId === dashboardId)
        .sort((a, b) => (a.y ?? 0) - (b.y ?? 0) || (a.x ?? 0) - (b.x ?? 0)),
    // count issues по фильтру (для StatWidget) — клиентский match
    issueCount: (s) => (filter: FilterNode | null) =>
      Object.values(s.issues).filter((i) => !i.archivedAt && match(filter, i)).length,
    // issues сохранённого view (для ViewWidget): команда view + его фильтр
    viewIssues: (s) => (viewId: string) => {
      const v = s.views[viewId]
      if (!v) return []
      const f = v.filter && (v.filter as any).type ? (v.filter as FilterNode) : null
      return Object.values(s.issues)
        .filter((i) => i.teamId === v.teamId && !i.archivedAt && match(f, i))
        .slice(0, 25)
    },
    // глобальный поиск над локальным стором (quick-switch)
    searchAll: (s) => (q: string) => {
      const t = q.trim().toLowerCase()
      if (!t) return [] as any[]
      const out: any[] = []
      const hit = (str: string) => str && str.toLowerCase().includes(t)
      for (const i of Object.values(s.issues)) {
        if (i.archivedAt) continue
        if (hit(i.identifier) || hit(i.title)) out.push({ type: 'issue', id: i.id, label: `${i.identifier} ${i.title}`, teamId: i.teamId })
        if (out.length > 30) break
      }
      for (const tm of Object.values(s.teams)) if (hit(tm.key) || hit(tm.name)) out.push({ type: 'team', id: tm.id, label: `${tm.key} · ${tm.name}` })
      for (const d of Object.values(s.documents)) if (!d.deletedAt && hit(d.title)) out.push({ type: 'doc', id: d.id, label: `Doc · ${d.title}`, teamId: d.teamId })
      for (const c of Object.values(s.channels)) if (!c.deletedAt && c.kind === 'channel' && hit(c.name)) out.push({ type: 'channel', id: c.id, label: `Channel · ${c.name}` })
      for (const d of Object.values(s.dashboards)) if (!d.deletedAt && hit(d.name)) out.push({ type: 'dashboard', id: d.id, label: `Dashboard · ${d.name}` })
      for (const v of Object.values(s.views)) if (!v.deletedAt && hit(v.name)) out.push({ type: 'view', id: v.id, label: `View · ${v.name}`, teamId: v.teamId })
      return out.slice(0, 30)
    },
    myNotifications: (s) => Object.values(s.notifications).sort((a, b) => String(b.at).localeCompare(String(a.at))),
    unreadCount: (s) => Object.values(s.notifications).filter((n) => !n.readAt).length,
    // --- Чаты ---
    channelList: (s) => Object.values(s.channels).filter((c) => c.kind === 'channel' && !c.deletedAt && !c.archivedAt).sort((a, b) => String(a.name).localeCompare(String(b.name))),
    myDMs: (s) => Object.values(s.channels).filter((c) => c.kind === 'dm' && !c.deletedAt),
    userName: (s) => (id: string) => s.users[id]?.name || s.users[id]?.email || 'Unknown',
    wsUsers: (s) => Object.values(s.users).filter((u) => u.id !== s.currentUserId),
    // участники канала (userId[]) и проверка членства текущего юзера (право управлять)
    channelMembersOf: (s) => (channelId: string) =>
      Object.values(s.channelMembers).filter((m) => m.channelId === channelId).map((m) => m.userId),
    isChannelMember: (s) => (channelId: string) =>
      Object.values(s.channelMembers).some((m) => m.channelId === channelId && m.userId === s.currentUserId),
    // участники для отображения: публичный канал = весь воркспейс (видят все), приватный = явные member'ы
    channelParticipants: (s) => (channelId: string) => {
      const ch = s.channels[channelId]
      if (ch && ch.kind === 'channel' && !ch.private) return Object.keys(s.users)
      return Object.values(s.channelMembers).filter((m) => m.channelId === channelId).map((m) => m.userId)
    },
    dmOtherId: (s) => (channelId: string) => {
      const m = Object.values(s.channelMembers).find((x) => x.channelId === channelId && x.userId !== s.currentUserId)
      return m?.userId
    },
    // id DM-канала с конкретным юзером (для бейджа непрочитанных в списке DM)
    dmChannelWith: (s) => (userId: string) => {
      const dmIds = new Set(Object.values(s.channels).filter((c) => c.kind === 'dm' && !c.deletedAt).map((c) => c.id))
      const mine = new Set(Object.values(s.channelMembers).filter((m) => m.userId === s.currentUserId && dmIds.has(m.channelId)).map((m) => m.channelId))
      const hit = Object.values(s.channelMembers).find((m) => m.userId === userId && mine.has(m.channelId))
      return hit?.channelId || ''
    },
    channelMessages: (s) => (channelId: string) => s.messages[channelId] || [],
    issueComments: (s) => (issueId: string) => s.comments[issueId] || [],
    teamLabels: (s) => (teamId: string) => Object.values(s.labels).filter((l) => !l.teamId || l.teamId === teamId),
    issuesByCategory: (s) => {
      const cats = [
        { cat: 'backlog', color: '#6b747e' }, { cat: 'unstarted', color: '#9aa3ad' },
        { cat: 'started', color: '#d29922' }, { cat: 'completed', color: '#5e6ad2' }, { cat: 'canceled', color: '#6b747e' },
      ]
      const all = Object.values(s.issues).filter((i) => !i.archivedAt && !i.deletedAt)
      return cats.map((c) => ({ ...c, count: all.filter((i) => s.states[i.stateId]?.category === c.cat).length }))
    },
    workspaceMembers: (s) => Object.values(s.members).map((m) => ({ ...m, name: s.users[m.userId]?.name || s.users[m.userId]?.email || '—' })),
    pendingInvites: (s) => Object.values(s.invitations).filter((i) => !i.acceptedAt),
    issueActivity: (s) => (issueId: string) => s.activity[issueId] || [],
    teamMilestones: (s) => (teamId: string) => Object.values(s.milestones).filter((m) => m.teamId === teamId && !m.deletedAt).sort((a, b) => String(a.targetDate || '9999').localeCompare(String(b.targetDate || '9999'))),
    milestoneProgress(state): (msId: string) => { done: number; total: number } {
      const idx = this.issuesByMilestone
      return (msId) => {
        const iss = idx.get(msId) || []
        const done = iss.filter((i) => state.states[i.stateId]?.category === 'completed').length
        return { done, total: iss.length }
      }
    },
    milestoneIssues(): (msId: string) => any[] {
      const idx = this.issuesByMilestone
      return (msId) => [...(idx.get(msId) || [])]
        .sort((a, b) => String(a.boardPosition || '').localeCompare(String(b.boardPosition || '')))
    },
    labelsForIssue(state): (issueId: string) => any[] {
      const idx = this.labelsByIssue
      return (issueId) => [...(idx.get(issueId) || [])].map((id) => state.labels[id]).filter(Boolean)
    },
    // --- Meetings ---
    scheduledMeetings: (s) => Object.values(s.meetings).filter((m) => !m.deletedAt && m.status === 'scheduled')
      .sort((a, b) => String(a.scheduledAt || '9999').localeCompare(String(b.scheduledAt || '9999'))),
    // предстоящие: без даты или дата в будущем
    upcomingMeetings: (s) => {
      const now = Date.now()
      return Object.values(s.meetings).filter((m) => !m.deletedAt && m.status === 'scheduled' && (!m.scheduledAt || new Date(m.scheduledAt).getTime() >= now))
        .sort((a, b) => String(a.scheduledAt || '9999').localeCompare(String(b.scheduledAt || '9999')))
    },
    // прошедшие: дата в прошлом (звонок завершён, но встреча остаётся в истории)
    pastMeetings: (s) => {
      const now = Date.now()
      return Object.values(s.meetings).filter((m) => !m.deletedAt && m.status === 'scheduled' && m.scheduledAt && new Date(m.scheduledAt).getTime() < now)
        .sort((a, b) => String(b.scheduledAt || '').localeCompare(String(a.scheduledAt || '')))
    },
    liveMeetings: (s) => Object.values(s.meetings).filter((m) => !m.deletedAt && m.status === 'live'),
    meetingById: (s) => (id: string) => s.meetings[id],
    meetingParts: (s) => (meetingId: string) => Object.values(s.meetingParticipants).filter((p) => p.meetingId === meetingId),
    meetingJoined: (s) => (meetingId: string) =>
      Object.values(s.meetingParticipants).filter((p) => p.meetingId === meetingId && p.state === 'joined'),
    liveHuddleForChannel: (s) => (channelId: string) =>
      Object.values(s.meetings).find((m) => m.channelId === channelId && m.kind === 'huddle' && m.status === 'live' && !m.deletedAt),
    // --- Calendar ---
    calendarEventsList: (s) => Object.values(s.calendarEvents).filter((e) => !e.deletedAt),
    // --- Automations ---
    automationsList: (s) => Object.values(s.automations).filter((a) => !a.deletedAt),
    templatesList: (s) => Object.values(s.issueTemplates).filter((t) => !t.deletedAt)
      .sort((a, b) => String(a.name).localeCompare(String(b.name))),
    // --- Time tracking ---
    timeEntriesForIssue: (s) => (issueId: string) => Object.values(s.timeEntries)
      .filter((e) => !e.deletedAt && e.issueId === issueId)
      .sort((a, b) => String(b.startedAt).localeCompare(String(a.startedAt))),
    runningTimer: (s) => Object.values(s.timeEntries).find((e) => !e.deletedAt && !e.endedAt),
    // суммарные минуты по задаче (бегущая запись считается до now)
    issueTrackedMin: (s) => (issueId: string) => {
      const now = Date.now()
      return Object.values(s.timeEntries).filter((e) => !e.deletedAt && e.issueId === issueId).reduce((acc, e) => {
        const end = e.endedAt ? new Date(e.endedAt).getTime() : now
        return acc + Math.max(0, Math.round((end - new Date(e.startedAt).getTime()) / 60000))
      }, 0)
    },
    // --- Burndown ---
    cycleBurndown: (s) => (cycleId: string) => {
      const snaps = Object.values(s.cycleSnapshots).filter((x: any) => x.cycleId === cycleId)
        .sort((a: any, b: any) => String(a.day).localeCompare(String(b.day)))
      // если у задач есть очки — по очкам, иначе по количеству
      const byPoints = snaps.some((x: any) => Number(x.totalPoints) > 0)
      return snaps.map((x: any) => ({ day: x.day, remaining: byPoints ? Number(x.remainingPoints) : Number(x.remainingCount) }))
    },
    // --- Chat unread ---
    channelUnreadCount: (s) => (channelId: string) => Number(s.channelUnread[channelId]?.count || 0),
    channelLastReadAt: (s) => (channelId: string) => s.channelUnread[channelId]?.lastReadAt || null,
    totalUnread: (s) => Object.values(s.channelUnread).reduce((n, u: any) => n + Number(u.count || 0), 0),
    // --- Attachments ---
    attachmentsFor: (s) => (entityType: string, entityId: string) =>
      Object.values(s.attachments).filter((a) => a.entityType === entityType && a.entityId === entityId && !a.deletedAt)
        .sort((a, b) => String(a.createdAt).localeCompare(String(b.createdAt))),
    typingNames: (s) => (channelId: string) => {
      const t = s.typing[channelId] || {}
      return Object.keys(t)
        .filter((uid) => t[uid] && uid !== s.currentUserId)
        .map((uid) => s.users[uid]?.name || s.users[uid]?.email || 'Someone')
    },

    teamDocs: (s) => (teamId: string) =>
      Object.values(s.documents).filter((d) => d.teamId === teamId && !d.deletedAt && !d.archivedAt),
    // плоский список доков команды с глубиной (DFS по parentId) — для дерева
    teamDocTree: (s) => (teamId: string) => {
      const all = Object.values(s.documents).filter((d) => d.teamId === teamId && !d.deletedAt && !d.archivedAt)
      const byParent: Record<string, any[]> = {}
      for (const d of all) {
        const k = d.parentId || '_root'
        ;(byParent[k] ||= []).push(d)
      }
      const out: { doc: any; depth: number }[] = []
      const walk = (parent: string, depth: number) => {
        for (const d of (byParent[parent] || []).sort((a, b) => String(a.title).localeCompare(String(b.title)))) {
          out.push({ doc: d, depth })
          walk(d.id, depth + 1)
        }
      }
      walk('_root', 0)
      return out
    },
    // циклы команды + производный статус по датам (ISO); список из индекса cyclesByTeam
    teamCycles(): (teamId: string) => any[] {
      const idx = this.cyclesByTeam
      const today = new Date().toISOString().slice(0, 10)
      return (teamId) => (idx.get(teamId) || []).map((c) => ({
        ...c,
        status: today < String(c.startsOn).slice(0, 10) ? 'upcoming' : today > String(c.endsOn).slice(0, 10) ? 'completed' : 'active',
      }))
    },
    // дочерние issue
    subIssuesOf(): (parentId: string) => any[] {
      const idx = this.issuesByParent
      return (parentId) => idx.get(parentId) || []
    },
    // прогресс по детям: completed по категории статуса
    childProgress(state): (parentId: string) => { done: number; total: number } {
      const idx = this.issuesByParent
      return (parentId) => {
        const kids = idx.get(parentId) || []
        const done = kids.filter((k) => state.states[k.stateId]?.category === 'completed').length
        return { done, total: kids.length }
      }
    },
    // связи issue: прямые (source) + зеркальные (target), с человекочитаемой меткой
    relationsForIssue(): (issueId: string) => { id: string; label: string; otherId: string }[] {
      const idx = this.relationsByIssue
      return (issueId) => idx.get(issueId) || []
    },
  },

  actions: {
    // Влить пачку сущностей одного типа (camelCase из bootstrap). Ядро — чистая
    // функция lib/sync.applyChunk (unit-тестируется отдельно); стор передаёт свой
    // state-контейнер, D7-merge + prune делаются там.
    applyChunk(entityType: string, arr: Entity[]) {
      applyChunkCore(this as any, entityType, arr)
    },

    async bootstrap(wsId: string) {
      const epoch = ++bootstrapEpoch // конкурентный bootstrap (switchWorkspace + refetch) не должен затирать более поздний
      this.syncing = true
      try {
        // смена воркспейса = перезагрузка среза: чистим бакеты прошлого тенанта.
        if (this.activeWorkspaceId !== wsId) {
          this.teams = {}; this.states = {}; this.issues = {}; this.views = {}; this.relations = {}
          this.cycles = {}; this.customFields = {}; this.customFieldValues = {}
          this.dashboards = {}; this.dashboardWidgets = {}; this.members = {}; this.teamMembers = {}; this.notifications = {}; this.documents = {}
          this.channels = {}; this.channelMembers = {}; this.users = {}; this.messages = {}; this.activeChannelId = ''
          this.hasMoreMessages = {}; this.threadParentId = ''; this.threadMessages = []; this.typing = {}; this.comments = {}
          this.labels = {}; this.issueLabels = {}; this.invitations = {}; this.milestones = {}; this.activity = {}; this.labelFilter = []; this.channelUnread = {}; this.cycleSnapshots = {}
          this.activeFilter = null
        }
        // подписка ДО стрима: мутации, случившиеся во время снапшота, копятся
        // в pendingTypes и догоняются catchUp'ом сразу после (см. flush ниже).
        this.subscribe(wsId)
        const { sync } = useClients()
        for await (const chunk of sync.bootstrap({ workspaceId: wsId })) {
          if (epoch !== bootstrapEpoch) return // стартовал более новый bootstrap — наши чанки устарели
          const json = new TextDecoder().decode(chunk.entities) || '[]'
          this.applyChunk(chunk.entityType, JSON.parse(json))
          this.cursor = Number(chunk.cursor)
        }
        if (epoch !== bootstrapEpoch) return
        this.activeWorkspaceId = wsId
        this.loaded = true
        this.persist()
        // дельты, прилетевшие во время стрима (classic snapshot+journal)
        if (pendingTypes.size) {
          const types = [...pendingTypes]
          pendingTypes.clear()
          await this.catchUp(wsId, types)
        }
      } finally {
        if (epoch === bootstrapEpoch) this.syncing = false
      }
    },

    // resync — догон после WS-реконнекта: за разрыв дельты потеряны, спрашиваем
    // change_log по курсору и рефетчим только изменившиеся типы. Фолбэк — полный bootstrap.
    async resync(wsId: string) {
      if (this.activeWorkspaceId !== wsId) return
      try {
        const { sync } = useClients()
        const res = await sync.changes({ workspaceId: wsId, sinceCursor: BigInt(this.cursor || 0) })
        const types = new Set<string>()
        for (const d of res.deltas || []) if (d.entityType) types.add(d.entityType)
        if (types.size) await this.catchUp(wsId, [...types])
        if (Number(res.cursor)) { this.cursor = Number(res.cursor); this.persist() }
      } catch {
        await this.bootstrap(wsId)
      }
    },

    // --- Docs (M7 ч.3) ---
    async createDoc(teamId: string, title: string, parentId = ''): Promise<string> {
      const { docs } = useClients()
      const res = await docs.createDoc({ workspaceId: this.activeWorkspaceId, teamId, title, parentId })
      this.documents[res.entityId] = { id: res.entityId, teamId, title, body: {}, parentId: parentId || null }
      return res.entityId
    },
    async updateDoc(id: string, patch: { title?: string; md?: string }) {
      const { docs } = useClients()
      const d = this.documents[id]
      if (d) { if (patch.title != null) d.title = patch.title; if (patch.md != null) d.body = { md: patch.md } }
      await docs.updateDoc({
        workspaceId: this.activeWorkspaceId, id,
        title: patch.title,
        body: patch.md != null ? new TextEncoder().encode(JSON.stringify({ md: patch.md })) : undefined,
      })
    },
    async deleteDoc(id: string) {
      const { docs } = useClients()
      if (this.documents[id]) delete this.documents[id]
      await docs.deleteDoc({ workspaceId: this.activeWorkspaceId, id })
    },

    async markNotificationRead(id: string) {
      const { notifications } = useClients()
      if (this.notifications[id]) this.notifications[id].readAt = new Date().toISOString()
      await notifications.markRead({ workspaceId: this.activeWorkspaceId, id })
    },
    async markAllNotificationsRead() {
      const { notifications } = useClients()
      for (const n of Object.values(this.notifications)) if (!n.readAt) n.readAt = new Date().toISOString()
      await notifications.markAllRead({ workspaceId: this.activeWorkspaceId })
    },

    // currentUserId: из localStorage, иначе /auth/me.
    async loadUser() {
      this.currentUserId = getUser()
      if (!this.currentUserId) {
        const base = useRuntimeConfig().public.apiBase as string
        const res = await fetch(`${base}/auth/me`, { credentials: 'include' })
        if (res.status === 401) return handleUnauthorized()
        if (res.ok) this.currentUserId = (await res.json()).user_id || ''
      }
    },
    async updateSectionsConfig(config: any) {
      const { workspaces } = useClients()
      // optimistic
      const m = Object.values(this.members).find((x) => x.userId === this.currentUserId)
      if (m) m.sectionsConfig = config
      await workspaces.updateSectionsConfig({ workspaceId: this.activeWorkspaceId, config: new TextEncoder().encode(JSON.stringify(config)) })
    },

    // Список «мои воркспейсы» для свитчера.
    async fetchMyWorkspaces() {
      const base = useRuntimeConfig().public.apiBase as string
      const res = await fetch(`${base}/me/workspaces`, { credentials: 'include' })
      if (res.status === 401) return handleUnauthorized()
      if (res.ok) this.myWorkspaces = (await res.json()) || [] // API может отдать null
    },

    async createWorkspace(slug: string, name: string): Promise<string> {
      const { workspaces } = useClients()
      const res = await workspaces.createWorkspace({ slug, name })
      await this.fetchMyWorkspaces()
      await this.bootstrap(res.entityId)
      return res.entityId
    },

    async createTeam(key: string, name: string, parentId = '') {
      const { teams } = useClients()
      const res = await teams.createTeam({ workspaceId: this.activeWorkspaceId, key, name, parentId })
      // optimistic: показать сразу; реальный path придёт с delta-refresh.
      const parentPath = parentId ? String(this.teams[parentId]?.path || '') + '.' : ''
      this.teams[res.entityId] = { id: res.entityId, key, name, path: parentPath + 'zzz_' + res.entityId }
    },

    // --- Issues (M1) ---
    async createIssue(teamId: string, title: string, stateId = '') {
      const { issues } = useClients()
      const res = await issues.createIssue({ workspaceId: this.activeWorkspaceId, teamId, title, stateId })
      this.issues[res.entityId] = {
        id: res.entityId, teamId, title, stateId, priority: 0, identifier: '…', archivedAt: null,
      }
      return res.entityId
    },
    async setIssueState(id: string, stateId: string) {
      const { issues } = useClients()
      if (this.issues[id]) this.issues[id].stateId = stateId // optimistic
      await issues.updateIssueState({ workspaceId: this.activeWorkspaceId, id, stateId })
    },
    // --- Milestones ---
    async createMilestone(teamId: string, name: string, targetDate = '') {
      const { milestones } = useClients()
      const res = await milestones.createMilestone({ workspaceId: this.activeWorkspaceId, teamId, name, targetDate })
      this.milestones[res.entityId] = { id: res.entityId, teamId, name, targetDate: targetDate || null, status: 'planned' }
      return res.entityId
    },
    async updateMilestone(id: string, patch: { name?: string; targetDate?: string; status?: string }) {
      const { milestones } = useClients()
      const m = this.milestones[id]
      if (m) Object.assign(m, patch.name != null ? { name: patch.name } : {}, patch.status != null ? { status: patch.status } : {}, patch.targetDate != null ? { targetDate: patch.targetDate } : {})
      await milestones.updateMilestone({ workspaceId: this.activeWorkspaceId, id, ...patch })
    },
    async deleteMilestone(id: string) {
      const { milestones } = useClients()
      if (this.milestones[id]) delete this.milestones[id]
      await milestones.deleteMilestone({ workspaceId: this.activeWorkspaceId, id })
    },
    async setIssueMilestone(id: string, milestoneId: string) {
      const { issues } = useClients()
      if (this.issues[id]) this.issues[id].milestoneId = milestoneId || null
      await issues.setMilestone({ workspaceId: this.activeWorkspaceId, id, milestoneId: milestoneId || '' })
    },
    // --- Estimate ---
    async setIssueEstimate(id: string, estimate: number) {
      const { issues } = useClients()
      if (this.issues[id]) this.issues[id].estimate = estimate || null
      await issues.setEstimate({ workspaceId: this.activeWorkspaceId, id, estimate: estimate || 0 })
    },
    // --- Workflow states ---
    async createState(teamId: string, name: string, category: string) {
      const { teams } = useClients()
      await teams.createWorkflowState({ workspaceId: this.activeWorkspaceId, teamId, name, category, colorToken: 'status-' + category })
    },
    async updateState(id: string, patch: { name?: string; category?: string; colorToken?: string }) {
      const { teams } = useClients()
      const s = this.states[id]
      if (s) { if (patch.name != null) s.name = patch.name; if (patch.category != null) s.category = patch.category }
      await teams.updateWorkflowState({ workspaceId: this.activeWorkspaceId, id, ...patch })
    },
    async deleteState(id: string) {
      const { teams } = useClients()
      await teams.deleteWorkflowState({ workspaceId: this.activeWorkspaceId, id })
    },
    // --- Workspace settings ---
    async updateTeam(id: string, patch: { name?: string; private?: boolean }) {
      const { teams } = useClients()
      const t = this.teams[id]
      if (t) Object.assign(t, patch)
      await teams.updateTeam({ workspaceId: this.activeWorkspaceId, id, ...patch })
    },
    async moveTeam(id: string, newParentId: string) {
      const { teams } = useClients()
      await teams.moveTeam({ workspaceId: this.activeWorkspaceId, id, newParentId })
      await this.catchUp(this.activeWorkspaceId, ['team']) // path-пересчёт у потомков
    },
    async deleteTeam(id: string) {
      const { teams } = useClients()
      await teams.deleteTeam({ workspaceId: this.activeWorkspaceId, id })
      delete this.teams[id]
    },
    async deleteWorkspace() {
      const { workspaces } = useClients()
      const id = this.activeWorkspaceId
      await workspaces.deleteWorkspace({ workspaceId: id, id })
      delete this.workspaces[id]
      this.myWorkspaces = this.myWorkspaces.filter((w) => w.id !== id)
      this.activeWorkspaceId = ''
      const next = this.myWorkspaces[0]?.id
      if (next) await this.bootstrap(next)
    },
    async renameWorkspace(name: string) {
      const { workspaces } = useClients()
      if (this.workspaces[this.activeWorkspaceId]) this.workspaces[this.activeWorkspaceId].name = name
      await workspaces.updateWorkspaceSettings({ workspaceId: this.activeWorkspaceId, name })
    },
    // --- Members / invites ---
    // inviteMember возвращает invite-токен (непустой только для нового, ещё не зарегистрированного юзера)
    async inviteMember(email: string, role = 'member'): Promise<string> {
      const { workspaces } = useClients()
      const res = await workspaces.inviteMember({ workspaceId: this.activeWorkspaceId, email, role })
      await this.catchUp(this.activeWorkspaceId, ['workspace_member', 'invitation'])
      return res.token || ''
    },
    // acceptInvitation — принять приглашение по токену; возвращает wsId присоединённого воркспейса
    async acceptInvitation(token: string): Promise<string> {
      const { workspaces } = useClients()
      const res = await workspaces.acceptInvitation({ token })
      return res.entityId
    },
    async updateMemberRole(userId: string, role: string) {
      const { workspaces } = useClients()
      const m = Object.values(this.members).find((x) => x.userId === userId)
      if (m) m.role = role
      await workspaces.updateMemberRole({ workspaceId: this.activeWorkspaceId, userId, role })
    },
    async revokeInvitation(id: string) {
      const { workspaces } = useClients()
      delete this.invitations[id]
      await workspaces.revokeInvitation({ workspaceId: this.activeWorkspaceId, id })
    },
    // --- Team membership (private-team access management) ---
    async addTeamMember(teamId: string, userId: string, role = 'member') {
      const { teams } = useClients()
      await teams.addTeamMember({ workspaceId: this.activeWorkspaceId, teamId, userId, role })
      await this.catchUp(this.activeWorkspaceId, ['team_member'])
    },
    async removeTeamMember(teamId: string, userId: string) {
      const { teams } = useClients()
      for (const k of Object.keys(this.teamMembers)) {
        const tm: any = this.teamMembers[k]
        if (tm.teamId === teamId && tm.userId === userId) delete this.teamMembers[k]
      }
      await teams.removeTeamMember({ workspaceId: this.activeWorkspaceId, teamId, userId })
    },
    async leaveTeam(teamId: string) {
      const { teams } = useClients()
      for (const k of Object.keys(this.teamMembers)) {
        const tm: any = this.teamMembers[k]
        if (tm.teamId === teamId && tm.userId === this.currentUserId) delete this.teamMembers[k]
      }
      await teams.leaveTeam({ workspaceId: this.activeWorkspaceId, teamId })
    },
    // --- Activity ---
    async loadActivity(issueId: string) {
      const { issues } = useClients()
      const res = await issues.getActivity({ workspaceId: this.activeWorkspaceId, entityId: issueId })
      this.activity[issueId] = this._decode(res.activity)
      // description приходит лениво (bootstrap его не шлёт) → мёржим в стор-issue
      const iss = this.issues[issueId]
      if (iss) iss.description = res.description ? { md: res.description } : (iss.description || {})
    },
    // --- Labels ---
    async createLabel(teamId: string, name: string, colorToken = '#6b747e'): Promise<string> {
      const { teams } = useClients()
      const res = await teams.createLabel({ workspaceId: this.activeWorkspaceId, teamId, name, colorToken })
      this.labels[res.entityId] = { id: res.entityId, teamId, name, colorToken }
      return res.entityId
    },
    async updateLabel(id: string, patch: { name?: string; colorToken?: string }) {
      const { teams } = useClients()
      const l = this.labels[id]
      if (l) Object.assign(l, patch.name != null ? { name: patch.name } : {}, patch.colorToken != null ? { colorToken: patch.colorToken } : {})
      await teams.updateLabel({ workspaceId: this.activeWorkspaceId, id, ...patch })
    },
    async deleteLabel(id: string) {
      const { teams } = useClients()
      delete this.labels[id]
      for (const k of Object.keys(this.issueLabels)) if (this.issueLabels[k].labelId === id) delete this.issueLabels[k]
      await teams.deleteLabel({ workspaceId: this.activeWorkspaceId, id })
    },
    async removeMember(userId: string) {
      const { workspaces } = useClients()
      const m = Object.values(this.members).find((x) => x.userId === userId)
      if (m) delete this.members[m.id]
      await workspaces.removeMember({ workspaceId: this.activeWorkspaceId, userId })
    },
    async addLabel(issueId: string, labelId: string) {
      const { issues } = useClients()
      this.issueLabels[`${issueId}:${labelId}`] = { issueId, labelId }
      await issues.addLabel({ workspaceId: this.activeWorkspaceId, issueId, labelId })
    },
    async removeLabel(issueId: string, labelId: string) {
      const { issues } = useClients()
      delete this.issueLabels[`${issueId}:${labelId}`]
      await issues.removeLabel({ workspaceId: this.activeWorkspaceId, issueId, labelId })
    },
    async setIssueAssignee(id: string, userId: string) {
      const { issues } = useClients()
      if (this.issues[id]) this.issues[id].assigneeId = userId || null // optimistic
      await issues.assignIssue({ workspaceId: this.activeWorkspaceId, id, assigneeId: userId || '' })
    },
    async setIssuePriority(id: string, priority: number) {
      const { issues } = useClients()
      if (this.issues[id]) this.issues[id].priority = priority
      await issues.setPriority({ workspaceId: this.activeWorkspaceId, id, priority })
    },
    async setIssueTitle(id: string, title: string) {
      const { issues } = useClients()
      if (this.issues[id]) this.issues[id].title = title
      await issues.updateIssue({ workspaceId: this.activeWorkspaceId, id, title })
    },
    // --- Чаты ---
    _decode(bytes: Uint8Array) { return JSON.parse(new TextDecoder().decode(bytes) || '[]') },
    async loadMessages(channelId: string) {
      const { chats } = useClients()
      this.activeChannelId = channelId
      const res = await chats.getMessages({ workspaceId: this.activeWorkspaceId, channelId, limit: 50 })
      this.messages[channelId] = this._decode(res.messages)
      this.hasMoreMessages[channelId] = res.hasMore
    },
    // подгрузка истории вверх (бесконечный скролл)
    async loadOlder(channelId: string) {
      const cur = this.messages[channelId] || []
      if (!cur.length || !this.hasMoreMessages[channelId]) return
      const { chats } = useClients()
      const res = await chats.getMessages({ workspaceId: this.activeWorkspaceId, channelId, before: cur[0].id, limit: 50 })
      const older = this._decode(res.messages)
      this.messages[channelId] = [...older, ...cur]
      this.hasMoreMessages[channelId] = res.hasMore
    },
    // фоновый рефетч активного канала (по WS-дельте) без сброса скролла
    async refetchActive() {
      if (!this.activeChannelId) return
      const { chats } = useClients()
      const n = (this.messages[this.activeChannelId] || []).length || 50
      const res = await chats.getMessages({ workspaceId: this.activeWorkspaceId, channelId: this.activeChannelId, limit: Math.min(100, Math.max(50, n)) })
      this.messages[this.activeChannelId] = this._decode(res.messages)
      this.hasMoreMessages[this.activeChannelId] = res.hasMore
      if (this.threadParentId) this.loadThread(this.threadParentId)
    },
    async sendMessage(channelId: string, body: string, parentId = '') {
      const { chats } = useClients()
      const res = await chats.sendMessage({ workspaceId: this.activeWorkspaceId, channelId, body, parentId })
      if (parentId) { await this.loadThread(parentId); await this.refetchActive() }
      else await this.refetchActive()
      return res.entityId
    },
    async sendMessageAttached(channelId: string, body: string, files: File[], parentId = '') {
      const { chats, attachments } = useClients()
      const ups = [] as { key: string; filename: string; contentType: string; size: number }[]
      for (const f of files) ups.push(await this.uploadAndGetKey(f))
      const res = await chats.sendMessage({ workspaceId: this.activeWorkspaceId, channelId, body, parentId })
      for (const up of ups) {
        await attachments.createAttachment({
          workspaceId: this.activeWorkspaceId, entityType: 'message', entityId: res.entityId,
          filename: up.filename, contentType: up.contentType, size: BigInt(up.size), s3Key: up.key,
        })
      }
      if (ups.length) await this.catchUp(this.activeWorkspaceId, ['attachment'])
      if (parentId) await this.loadThread(parentId)
      await this.refetchActive()
    },
    async editMessage(id: string, body: string) {
      const { chats } = useClients()
      await chats.editMessage({ workspaceId: this.activeWorkspaceId, id, body })
      await this.refetchActive()
      if (this.threadParentId) await this.loadThread(this.threadParentId)
    },
    async deleteMessage(id: string) {
      const { chats } = useClients()
      await chats.deleteMessage({ workspaceId: this.activeWorkspaceId, id })
      await this.refetchActive()
      if (this.threadParentId) await this.loadThread(this.threadParentId)
    },
    async toggleReaction(messageId: string, emoji: string) {
      const { chats } = useClients()
      await chats.toggleReaction({ workspaceId: this.activeWorkspaceId, messageId, emoji })
      await this.refetchActive()
      if (this.threadParentId) await this.loadThread(this.threadParentId)
    },
    async sendTyping(channelId: string) {
      const { chats } = useClients()
      try { await chats.sendTyping({ workspaceId: this.activeWorkspaceId, channelId }) } catch {}
    },
    async loadThread(parentId: string) {
      this.threadParentId = parentId
      const { chats } = useClients()
      const res = await chats.getMessages({ workspaceId: this.activeWorkspaceId, channelId: this.activeChannelId, parentId, limit: 100 })
      this.threadMessages = this._decode(res.messages)
    },
    closeThread() { this.threadParentId = ''; this.threadMessages = [] },
    // --- Комментарии к задачам ---
    // Серверный поиск (FTS): тело задач/доков + ранжирование + опечатки (trgm) + сообщения.
    // Возвращает в форме палитры; клиентский searchAll покрывает только заголовки/локальный стор.
    async serverSearch(q: string): Promise<any[]> {
      if (!q.trim() || !this.activeWorkspaceId) return []
      const { query } = useClients()
      const [kwRes, sem] = await Promise.all([
        query.search({ workspaceId: this.activeWorkspaceId, query: q }),
        this.semanticSearch(q),
      ])
      const kw = (this._decode(kwRes.results) as any[]).map((r) => {
        if (r.type === 'issue') return { type: 'issue', id: r.id, label: `${r.subtitle} ${r.title}`, teamId: r.teamId }
        if (r.type === 'doc') return { type: 'doc', id: r.id, label: `Doc · ${r.title}`, teamId: r.teamId }
        if (r.type === 'channel') return { type: 'channel', id: r.id, label: `Channel · ${r.title}` }
        if (r.type === 'message') return { type: 'message', id: r.id, label: `${r.subtitle} · ${r.title}` }
        return { type: r.type, id: r.id, label: r.title, teamId: r.teamId }
      })
      // семантика — после точных совпадений, без дублей
      const seen = new Set(kw.map((r) => r.id))
      return [...kw, ...sem.filter((r) => !seen.has(r.id))]
    },

    // Семантический поиск issue (AI Фаза 1). Пусто при ошибке/недоступности эмбеддингов.
    async semanticSearch(q: string): Promise<any[]> {
      if (!q.trim() || !this.activeWorkspaceId) return []
      const { query } = useClients()
      try {
        const res = await query.semanticSearch({ workspaceId: this.activeWorkspaceId, query: q, limit: 6 })
        // теперь cross-entity Result[] {type,id,title,teamId} (issue/doc; comment→issue)
        const rs = this._decode(res.results) as any[]
        return rs.map((r) => ({ type: r.type, id: r.id, label: r.title, teamId: r.teamId, semantic: true }))
      } catch {
        return []
      }
    },

    // Похожие issue по вектору (для панели «similar / возможный дубль»).
    async similarIssues(issueId: string): Promise<any[]> {
      if (!this.activeWorkspaceId) return []
      const { query } = useClients()
      try {
        const res = await query.similarIssues({ workspaceId: this.activeWorkspaceId, issueId, limit: 6 })
        return this._decode(res.results) as any[]
      } catch {
        return []
      }
    },

    // AI петля фидбэка: лог реакции на подсказку (applied|dismissed) — сигнал для ретрейна.
    aiFeedback(kind: string, action: string, entityId: string, suggested: string) {
      if (!this.activeWorkspaceId) return
      const base = useRuntimeConfig().public.apiBase as string
      fetch(`${base}/me/workspaces/${this.activeWorkspaceId}/ai/feedback`, {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ kind, action, entityId, suggested }),
      }).catch(() => {})
    },

    // AI-подсказки для issue (Фаза 2): kNN-агрегация → {assigneeId, labelIds, similar}.
    async suggestForIssue(issue: any): Promise<any> {
      if (!this.activeWorkspaceId || !issue) return null
      const { query } = useClients()
      try {
        const text = `${issue.title || ''} ${issue.description?.md || ''}`.trim()
        const res = await query.suggest({ workspaceId: this.activeWorkspaceId, text, teamId: issue.teamId || '', limit: 6 })
        return this._decode(res.result) as any
      } catch {
        return null
      }
    },
    async themes(): Promise<any[]> {
      if (!this.activeWorkspaceId) return []
      const { query } = useClients()
      try {
        const res = await query.themes({ workspaceId: this.activeWorkspaceId })
        return (this._decode(res.result) as any[]) || []
      } catch {
        return []
      }
    },
    async anomalies(): Promise<any[]> {
      if (!this.activeWorkspaceId) return []
      const { query } = useClients()
      try {
        const res = await query.anomalies({ workspaceId: this.activeWorkspaceId, limit: 8 })
        return (this._decode(res.result) as any[]) || []
      } catch {
        return []
      }
    },
    async mood(): Promise<any> {
      if (!this.activeWorkspaceId) return null
      const { query } = useClients()
      try {
        const res = await query.mood({ workspaceId: this.activeWorkspaceId })
        return this._decode(res.result) as any
      } catch {
        return null
      }
    },
    async summarizeIssue(issueId: string): Promise<string> {
      if (!this.activeWorkspaceId) return ''
      const { query } = useClients()
      try {
        const res = await query.summarize({ workspaceId: this.activeWorkspaceId, issueId, maxSentences: 3 })
        return ((this._decode(res.result) as any)?.summary as string) || ''
      } catch {
        return ''
      }
    },
    async loadComments(issueId: string) {
      const { comments } = useClients()
      const res = await comments.getComments({ workspaceId: this.activeWorkspaceId, issueId })
      this.comments[issueId] = this._decode(res.comments)
    },
    async addComment(issueId: string, body: string) {
      const { comments } = useClients()
      await comments.addComment({ workspaceId: this.activeWorkspaceId, issueId, body })
      await this.loadComments(issueId)
    },
    async addCommentAttached(issueId: string, body: string, files: File[]) {
      const { comments, attachments } = useClients()
      const ups = [] as { key: string; filename: string; contentType: string; size: number }[]
      for (const f of files) ups.push(await this.uploadAndGetKey(f))
      const res = await comments.addComment({ workspaceId: this.activeWorkspaceId, issueId, body })
      for (const up of ups) {
        await attachments.createAttachment({
          workspaceId: this.activeWorkspaceId, entityType: 'comment', entityId: res.entityId,
          filename: up.filename, contentType: up.contentType, size: BigInt(up.size), s3Key: up.key,
        })
      }
      if (ups.length) await this.catchUp(this.activeWorkspaceId, ['attachment'])
      await this.loadComments(issueId)
    },
    async editComment(id: string, issueId: string, body: string) {
      const { comments } = useClients()
      await comments.editComment({ workspaceId: this.activeWorkspaceId, id, body })
      await this.loadComments(issueId)
    },
    async deleteComment(id: string, issueId: string) {
      const { comments } = useClients()
      await comments.deleteComment({ workspaceId: this.activeWorkspaceId, id })
      await this.loadComments(issueId)
    },
    async toggleCommentReaction(id: string, issueId: string, emoji: string) {
      const { comments } = useClients()
      await comments.toggleReaction({ workspaceId: this.activeWorkspaceId, commentId: id, emoji })
      await this.loadComments(issueId)
    },
    markTyping(channelId: string, userId: string) {
      ;(this.typing[channelId] ||= {})[userId] = Date.now()
      setTimeout(() => { if (this.typing[channelId]) { delete this.typing[channelId][userId] } }, 4000)
    },
    async createChannel(name: string, isPrivate = false): Promise<string> {
      const { chats } = useClients()
      const res = await chats.createChannel({ workspaceId: this.activeWorkspaceId, name, private: isPrivate })
      this.channels[res.entityId] = { id: res.entityId, kind: 'channel', name, private: isPrivate }
      await this.catchUp(this.activeWorkspaceId, ['channel', 'channel_member']) // подтянуть channel_member (себя)
      return res.entityId
    },
    async addChannelMember(channelId: string, userId: string) {
      const { chats } = useClients()
      await chats.addChannelMember({ workspaceId: this.activeWorkspaceId, channelId, userId })
      await this.catchUp(this.activeWorkspaceId, ['channel', 'channel_member'])
    },
    async removeChannelMember(channelId: string, userId: string) {
      const { chats } = useClients()
      await chats.removeChannelMember({ workspaceId: this.activeWorkspaceId, channelId, userId })
      await this.catchUp(this.activeWorkspaceId, ['channel', 'channel_member'])
    },
    async leaveChannel(channelId: string) {
      const { chats } = useClients()
      await chats.removeChannelMember({ workspaceId: this.activeWorkspaceId, channelId, userId: this.currentUserId })
      if (this.activeChannelId === channelId) this.activeChannelId = ''
      await this.catchUp(this.activeWorkspaceId, ['channel', 'channel_member'])
    },
    async archiveChannel(channelId: string) {
      const { chats } = useClients()
      await chats.archiveChannel({ workspaceId: this.activeWorkspaceId, id: channelId })
      if (this.activeChannelId === channelId) this.activeChannelId = ''
      if (this.channels[channelId]) this.channels[channelId].archivedAt = new Date().toISOString()
    },
    async createDM(userId: string): Promise<string> {
      const { chats } = useClients()
      const res = await chats.createDM({ workspaceId: this.activeWorkspaceId, otherUserId: userId })
      if (!this.channels[res.entityId]) this.channels[res.entityId] = { id: res.entityId, kind: 'dm' }
      await this.loadMessages(res.entityId)
      return res.entityId
    },

    // --- Cycles (M4 ч.2) ---
    async createCycle(teamId: string, startsOn: string, endsOn: string, name = '') {
      const { cycles } = useClients()
      const res = await cycles.createCycle({ workspaceId: this.activeWorkspaceId, teamId, name, startsOn, endsOn })
      this.cycles[res.entityId] = { id: res.entityId, teamId, name, startsOn, endsOn }
    },
    async updateCycle(id: string, patch: { name?: string; startsOn?: string; endsOn?: string }) {
      const { cycles } = useClients()
      const c = this.cycles[id]
      await cycles.updateCycle({ workspaceId: this.activeWorkspaceId, id, baseVersion: BigInt(c?.version ?? 0), ...patch })
      if (c) Object.assign(c, patch)
      await this.catchUp(this.activeWorkspaceId, ['cycle'])
    },
    async setIssueCycle(id: string, cycleId: string) {
      const { issues } = useClients()
      if (this.issues[id]) this.issues[id].cycleId = cycleId || null
      await issues.setCycle({ workspaceId: this.activeWorkspaceId, id, cycleId })
    },

    async createSubIssue(parentId: string, title: string) {
      const parent = this.issues[parentId]
      if (!parent) return
      const { issues } = useClients()
      const res = await issues.createIssue({ workspaceId: this.activeWorkspaceId, teamId: parent.teamId, title, parentId })
      this.issues[res.entityId] = { id: res.entityId, teamId: parent.teamId, parentId, title, identifier: '…', priority: 0, archivedAt: null }
    },
    async addRelation(sourceId: string, targetIdentifier: string, type: string) {
      const target = Object.values(this.issues).find((i) => i.identifier === targetIdentifier.trim())
      if (!target) throw new Error('target issue not found: ' + targetIdentifier)
      const { issues } = useClients()
      await issues.addRelation({ workspaceId: this.activeWorkspaceId, sourceId, targetId: target.id, type })
    },
    async removeRelation(relId: string) {
      const { issues } = useClients()
      if (this.relations[relId]) delete this.relations[relId]
      await issues.removeRelation({ workspaceId: this.activeWorkspaceId, id: relId })
    },
    // --- Dashboards / Widgets (M6 ч.2) ---
    async createDashboard(name: string) {
      const { dashboards } = useClients()
      const res = await dashboards.createDashboard({ workspaceId: this.activeWorkspaceId, name })
      this.dashboards[res.entityId] = { id: res.entityId, name }
      return res.entityId
    },
    async addWidget(dashboardId: string, type: string, title: string, config: any) {
      const { dashboards } = useClients()
      const res = await dashboards.addWidget({
        workspaceId: this.activeWorkspaceId, dashboardId, type, title,
        config: new TextEncoder().encode(JSON.stringify(config || {})),
      })
      this.dashboardWidgets[res.entityId] = { id: res.entityId, dashboardId, type, title, config }
    },
    async removeWidget(id: string) {
      const { dashboards } = useClients()
      if (this.dashboardWidgets[id]) delete this.dashboardWidgets[id]
      await dashboards.removeWidget({ workspaceId: this.activeWorkspaceId, id })
    },
    // перестановка виджета draggedId перед targetId (нормализуем y по текущему порядку)
    async moveWidget(dashboardId: string, draggedId: string, targetId: string) {
      const { dashboards } = useClients()
      const list = Object.values(this.dashboardWidgets)
        .filter((w) => w.dashboardId === dashboardId)
        .sort((a, b) => (a.y ?? 0) - (b.y ?? 0) || (a.x ?? 0) - (b.x ?? 0))
      const from = list.findIndex((w) => w.id === draggedId)
      const to = list.findIndex((w) => w.id === targetId)
      if (from < 0 || to < 0 || from === to) return
      const [moved] = list.splice(from, 1); list.splice(to, 0, moved)
      await Promise.all(list.map((w, i) => {
        if (w.y === i) return null
        w.y = i
        return dashboards.updateWidget({ workspaceId: this.activeWorkspaceId, id: w.id, y: i })
      }).filter(Boolean) as any)
    },
    async resizeWidget(id: string, w: number, h: number) {
      const { dashboards } = useClients()
      const wd = this.dashboardWidgets[id]; if (wd) { wd.w = w; wd.h = h }
      await dashboards.updateWidget({ workspaceId: this.activeWorkspaceId, id, w, h })
    },

    async createCustomField(teamId: string, name: string, type: string, options: string[] = []) {
      const { customFields } = useClients()
      const config = ['select', 'multiselect'].includes(type) ? { options } : {}
      const res = await customFields.createCustomField({
        workspaceId: this.activeWorkspaceId, teamId, name, type,
        config: new TextEncoder().encode(JSON.stringify(config)),
      })
      this.customFields[res.entityId] = { id: res.entityId, teamId, name, type, config }
    },
    async setCustomFieldValue(issueId: string, fieldId: string, value: any) {
      const { issues } = useClients()
      const empty = value === '' || value == null || (Array.isArray(value) && !value.length)
      const bytes = empty ? new Uint8Array() : new TextEncoder().encode(JSON.stringify(value))
      // optimistic
      const ex = Object.values(this.customFieldValues).find((x) => x.issueId === issueId && x.fieldId === fieldId)
      if (empty) { if (ex) delete this.customFieldValues[ex.id] }
      else if (ex) ex.value = value
      else { const k = 'tmp-' + fieldId + '-' + issueId; this.customFieldValues[k] = { id: k, issueId, fieldId, value } }
      await issues.setCustomFieldValue({ workspaceId: this.activeWorkspaceId, issueId, fieldId, value: bytes })
    },
    async setIssueSchedule(id: string, date: string) {
      const { issues } = useClients()
      if (this.issues[id]) this.issues[id].scheduledDate = date || null
      await issues.updateIssue({ workspaceId: this.activeWorkspaceId, id, scheduledDate: date })
    },
    async setIssueDescription(id: string, md: string) {
      const { issues } = useClients()
      if (this.issues[id]) this.issues[id].description = { md }
      await issues.updateIssue({ workspaceId: this.activeWorkspaceId, id, description: md })
    },

    // --- Meetings ---
    async createMeeting(title: string, scheduledAt: string, endsAt: string, participantIds: string[], channelId = '') {
      const { meetings } = useClients()
      const res = await meetings.createMeeting({ workspaceId: this.activeWorkspaceId, title, scheduledAt, endsAt, participantIds, channelId })
      await this.catchUp(this.activeWorkspaceId, ['meeting', 'meeting_participant'])
      return res.entityId
    },
    async startHuddle(channelId: string) {
      const { meetings } = useClients()
      const res = await meetings.startHuddle({ workspaceId: this.activeWorkspaceId, channelId })
      await this.catchUp(this.activeWorkspaceId, ['meeting', 'meeting_participant'])
      return res.entityId
    },
    async joinMeeting(meetingId: string): Promise<{ url: string; token: string }> {
      const { meetings } = useClients()
      const res = await meetings.joinMeeting({ workspaceId: this.activeWorkspaceId, meetingId })
      return { url: res.livekitUrl, token: res.token }
    },
    async leaveMeeting(meetingId: string) {
      const { meetings } = useClients()
      await meetings.leaveMeeting({ workspaceId: this.activeWorkspaceId, id: meetingId })
    },
    async endMeeting(meetingId: string) {
      const { meetings } = useClients()
      await meetings.endMeeting({ workspaceId: this.activeWorkspaceId, id: meetingId })
    },

    // --- Calendar ---
    async createCalendarEvent(p: { title: string; description?: string; kind?: string; startsAt: string; endsAt: string; allDay?: boolean; recurrence?: string; recurrenceUntil?: string }) {
      const { calendar } = useClients()
      const res = await calendar.createEvent({ workspaceId: this.activeWorkspaceId, title: p.title, description: p.description || '', kind: p.kind || 'event', startsAt: p.startsAt, endsAt: p.endsAt, allDay: !!p.allDay, recurrence: p.recurrence || 'none', recurrenceUntil: p.recurrenceUntil || '' })
      return res.entityId
    },
    async updateCalendarEvent(id: string, p: { title: string; description?: string; kind?: string; startsAt: string; endsAt: string; allDay?: boolean; recurrence?: string; recurrenceUntil?: string }) {
      const { calendar } = useClients()
      await calendar.updateEvent({ workspaceId: this.activeWorkspaceId, id, title: p.title, description: p.description || '', kind: p.kind || 'event', startsAt: p.startsAt, endsAt: p.endsAt, allDay: !!p.allDay, recurrence: p.recurrence || 'none', recurrenceUntil: p.recurrenceUntil || '' })
    },
    async deleteCalendarEvent(id: string) {
      const { calendar } = useClients()
      await calendar.deleteEvent({ workspaceId: this.activeWorkspaceId, id })
    },
    // --- Time tracking ---
    async startTimer(issueId = '', description = '') {
      const { time } = useClients()
      await time.startTimer({ workspaceId: this.activeWorkspaceId, issueId, description })
    },
    async stopTimer() {
      const { time } = useClients()
      await time.stopTimer({ workspaceId: this.activeWorkspaceId })
      await this.catchUp(this.activeWorkspaceId, ['time_entry']) // ended_at проставлен на нескольких — перечитать
    },
    async addTimeEntry(issueId: string, description: string, startedAt: string, endedAt: string) {
      const { time } = useClients()
      await time.addEntry({ workspaceId: this.activeWorkspaceId, issueId, description, startedAt, endedAt })
    },
    async deleteTimeEntry(id: string) {
      const { time } = useClients()
      await time.deleteEntry({ workspaceId: this.activeWorkspaceId, id })
    },
    async timeReport(p: { from: string; to: string; groupBy: string; issueId?: string; userId?: string }) {
      const { time } = useClients()
      const r = await time.report({ workspaceId: this.activeWorkspaceId, from: p.from, to: p.to, groupBy: p.groupBy, issueId: p.issueId || '', userId: p.userId || '' })
      return { rows: r.rows.map((x: any) => ({ key: x.key, minutes: Number(x.minutes), count: Number(x.count) })), total: Number(r.totalMinutes) }
    },
    // --- Automations ---
    async createAutomation(p: { teamId?: string; name: string; enabled: boolean; trigger: any; actions: any }) {
      const { automations } = useClients()
      await automations.createAutomation({ workspaceId: this.activeWorkspaceId, teamId: p.teamId || '', name: p.name, enabled: p.enabled, trigger: JSON.stringify(p.trigger), conditions: '', actions: JSON.stringify(p.actions) })
    },
    async updateAutomation(id: string, p: { teamId?: string; name: string; enabled: boolean; trigger: any; actions: any }) {
      const { automations } = useClients()
      await automations.updateAutomation({ workspaceId: this.activeWorkspaceId, id, teamId: p.teamId || '', name: p.name, enabled: p.enabled, trigger: JSON.stringify(p.trigger), conditions: '', actions: JSON.stringify(p.actions) })
    },
    async toggleAutomation(id: string, enabled: boolean) {
      const { automations } = useClients()
      await automations.setAutomationEnabled({ workspaceId: this.activeWorkspaceId, id, enabled })
    },
    async deleteAutomation(id: string) {
      const { automations } = useClients()
      await automations.deleteAutomation({ workspaceId: this.activeWorkspaceId, id })
    },
    // --- Issue templates + recurring ---
    _tmplInput(p: any) {
      return {
        teamId: p.teamId || '', name: p.name || '', title: p.title || '', descriptionMd: p.descriptionMd || '',
        priority: p.priority || 0, assigneeId: p.assigneeId || '', estimate: p.estimate || 0,
        labelIds: p.labelIds || [], recurrence: p.recurrence || 'none',
      }
    },
    async createTemplate(p: any) {
      const { templates } = useClients()
      await templates.createTemplate({ workspaceId: this.activeWorkspaceId, input: this._tmplInput(p) })
    },
    async updateTemplate(id: string, p: any) {
      const { templates } = useClients()
      await templates.updateTemplate({ workspaceId: this.activeWorkspaceId, id, input: this._tmplInput(p) })
    },
    async deleteTemplate(id: string) {
      const { templates } = useClients()
      await templates.deleteTemplate({ workspaceId: this.activeWorkspaceId, id })
    },
    async spawnTemplate(id: string): Promise<string> {
      const { templates } = useClients()
      const res = await templates.spawnTemplate({ workspaceId: this.activeWorkspaceId, id })
      await this.catchUp(this.activeWorkspaceId, ['issue', 'issue_template']) // подтянуть новый issue
      return res.issueId
    },
    // iCal-фид: вернуть полный URL подписки (создаёт токен при первом вызове).
    async icalFeedUrl(rotate = false): Promise<string> {
      const base = useRuntimeConfig().public.apiBase as string
      const res = await fetch(`${base}/ical/token${rotate ? '/rotate' : ''}`, { method: rotate ? 'POST' : 'GET', credentials: 'include' })
      if (res.status === 401) { handleUnauthorized(); return '' }
      const { url } = await res.json()
      return `${base}${url}`
    },

    // --- Attachments / files ---
    async addAttachment(entityType: string, entityId: string, file: File) {
      const up = await uploadFile(this.activeWorkspaceId, file)
      const { attachments } = useClients()
      await attachments.createAttachment({
        workspaceId: this.activeWorkspaceId, entityType, entityId,
        filename: up.filename, contentType: up.contentType, size: BigInt(up.size), s3Key: up.key,
      })
      await this.catchUp(this.activeWorkspaceId, ['attachment'])
    },
    async deleteAttachment(id: string) {
      const { attachments } = useClients()
      if (this.attachments[id]) delete this.attachments[id]
      await attachments.deleteAttachment({ workspaceId: this.activeWorkspaceId, id })
    },
    async fileUrl(s3Key: string): Promise<string> {
      const { attachments } = useClients()
      const res = await attachments.getAttachmentURL({ workspaceId: this.activeWorkspaceId, s3Key })
      return res.url
    },
    async uploadAndGetKey(file: File) { return uploadFile(this.activeWorkspaceId, file) },

    // --- Chat unread ---
    async markChannelRead(channelId: string) {
      if (this.channelUnread[channelId]) this.channelUnread[channelId] = { ...this.channelUnread[channelId], count: 0, lastReadAt: new Date().toISOString() }
      else this.channelUnread[channelId] = { channelId, count: 0, lastReadAt: new Date().toISOString() }
      const { chats } = useClients()
      try { await chats.markChannelRead({ workspaceId: this.activeWorkspaceId, channelId }) } catch {}
    },
    bumpUnread(channelId: string) {
      const cur = this.channelUnread[channelId]
      this.channelUnread[channelId] = { channelId, lastReadAt: cur?.lastReadAt || null, count: Number(cur?.count || 0) + 1 }
    },
    async setAvatar(s3Key: string) {
      const { attachments } = useClients()
      if (this.users[this.currentUserId]) this.users[this.currentUserId].avatarKey = s3Key || null
      await attachments.setUserAvatar({ workspaceId: this.activeWorkspaceId, s3Key })
    },
    async setLogo(s3Key: string) {
      const { attachments } = useClients()
      if (this.activeWorkspace) (this.activeWorkspace as any).logoKey = s3Key || null
      await attachments.setWorkspaceLogo({ workspaceId: this.activeWorkspaceId, s3Key })
    },
    async reorderIssue(id: string, stateId: string, beforeId = '', afterId = '') {
      const { issues } = useClients()
      if (this.issues[id]) this.issues[id].stateId = stateId // optimistic (позиция — после refresh)
      await issues.reorderIssue({ workspaceId: this.activeWorkspaceId, id, stateId, beforeId, afterId })
    },
    async archiveIssue(id: string) {
      const { issues } = useClients()
      if (this.issues[id]) this.issues[id].archivedAt = new Date().toISOString()
      await issues.archiveIssue({ workspaceId: this.activeWorkspaceId, id })
    },

    // --- Filter AST (M3) ---
    getFieldPredicate(field: string): FilterNode | undefined {
      const f = this.activeFilter
      if (!f || f.type !== 'and') return undefined
      return f.children.find((c) => c.type === 'predicate' && (c as any).field === field)
    },
    setFieldPredicate(field: string, pred: FilterNode | null) {
      const cur = this.activeFilter && this.activeFilter.type === 'and' ? this.activeFilter.children : []
      const next = cur.filter((c) => !(c.type === 'predicate' && (c as any).field === field))
      if (pred) next.push(pred)
      this.activeFilter = next.length ? { type: 'and', children: next } : null
    },
    clearFilter() {
      this.activeFilter = null
    },
    toggleLabelFilter(id: string) {
      this.labelFilter = this.labelFilter.includes(id) ? this.labelFilter.filter((x) => x !== id) : [...this.labelFilter, id]
    },

    // --- Views (M3) ---
    async createView(name: string, teamId: string) {
      const { views } = useClients()
      const filterBytes = new TextEncoder().encode(JSON.stringify(this.activeFilter || {}))
      const res = await views.createView({
        workspaceId: this.activeWorkspaceId, id: '', name, dataSource: 'issue',
        scopeKind: 'team', teamId, filter: filterBytes, layout: this.layout,
      })
      this.views[res.entityId] = {
        id: res.entityId, name, teamId, layout: this.layout, filter: this.activeFilter || {},
      }
    },
    async renameView(id: string, name: string) {
      const { views } = useClients()
      const v = this.views[id]
      if (v) v.name = name
      await views.updateView({ workspaceId: this.activeWorkspaceId, id, name })
    },
    async deleteView(id: string) {
      const { views } = useClients()
      delete this.views[id]
      await views.deleteView({ workspaceId: this.activeWorkspaceId, id })
    },
    applyView(viewId: string): Entity | undefined {
      const v = this.views[viewId]
      if (!v) return
      this.activeFilter = v.filter && (v.filter as any).type ? (v.filter as FilterNode) : null
      this.layout = v.layout === 'board' ? 'board' : 'list'
      return v
    },

    subscribe(wsId: string) {
      ws?.close()
      ws = connectWs(wsId, (d: any) => {
        const type = d?.entity_type || d?.entityType
        const eid = d?.entity_id || d?.entityId
        // typing: op = channelId, actor = userId
        if (type === 'typing') {
          if (d.op && d.actor) this.markTyping(d.op, d.actor)
          return
        }
        // сообщение в активном канале — точечный рефетч + mark read; в фоновом — bump непрочитанных
        if (type === 'message') {
          if (eid === this.activeChannelId) { this.refetchActive(); this.markChannelRead(eid) }
          else if (d.actor !== this.currentUserId) this.bumpUnread(eid)
          return
        }
        // комментарий — рефетч если эта задача открыта/загружена
        if (type === 'comment') {
          if (this.comments[eid] !== undefined) this.loadComments(eid)
          return
        }
        // прочее: инкрементальный catch-up — копим изменённые типы и тянем только их chunk'и
        // (а не весь срез). Неизвестный тип → копим как есть; сервер вернёт пустой chunk, если нет builder'а.
        if (type) pendingTypes.add(type)
        clearTimeout(refreshTimer)
        refreshTimer = setTimeout(() => {
          // bootstrap этого ws ещё идёт (subscribe-first) → не сбрасываем накопленное,
          // ждём: bootstrap сам сфлашит pendingTypes по завершении
          if (this.activeWorkspaceId !== wsId) return
          const types = [...pendingTypes]
          pendingTypes.clear()
          this.catchUp(wsId, types)
        }, 150)
      }, () => this.resync(wsId)) // WS-реконнект: дельты за разрыв потеряны → догон по курсору
    },

    // catchUp подтягивает текущее состояние только указанных entity_type (инкрементальный синк
    // по WS-дельте). applyChunk + prune те же, что в bootstrap → удаления/архив исчезают штатно.
    async catchUp(wsId: string, types: string[]) {
      if (!types.length || this.activeWorkspaceId !== wsId) return
      const { sync } = useClients()
      for await (const chunk of sync.bootstrap({ workspaceId: wsId, types })) {
        const json = new TextDecoder().decode(chunk.entities) || '[]'
        this.applyChunk(chunk.entityType, JSON.parse(json))
        if (Number(chunk.cursor)) this.cursor = Number(chunk.cursor)
      }
      this.persist()
    },

    // --- IndexedDB-персист (переживает перезагрузку, §7.9/9.7) ---
    persist() {
      set(IDB_KEY, {
        workspaces: this.workspaces,
        teams: this.teams,
        states: this.states,
        cursor: this.cursor,
        activeWorkspaceId: this.activeWorkspaceId,
      }).catch(() => {})
    },
    async hydrate() {
      const saved = await get(IDB_KEY).catch(() => null)
      if (saved) this.$patch(saved as any)
    },
  },
})
