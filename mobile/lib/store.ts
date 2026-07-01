// Local-first стор: держит ВЕСЬ воркспейс в памяти (как Pinia веба). Первичная
// загрузка — /snapshot (unary), дельты — SyncService.Changes → дозагрузка изменённых
// типов через /snapshot?only=. Реактивность через useSyncExternalStore. Персист в
// SQLite/localStorage (warm-start + offline).
import { useSyncExternalStore } from 'react'
import { API_BASE } from './config'
import { getCachedToken } from './auth'
import { clients } from './connect'
import { cacheGet, cacheSet } from './cache'

type Dict<T = any> = Record<string, T>

// entityType (snapshot) → ключ стора + хранить ли как map по id (иначе массив)
const MAP: Record<string, { key: string; map: boolean }> = {
  workspace: { key: 'workspace', map: true },
  team: { key: 'team', map: true },
  team_member: { key: 'teamMember', map: false },
  workflow_state: { key: 'state', map: true },
  label: { key: 'label', map: true },
  workspace_member: { key: 'member', map: false },
  issue: { key: 'issue', map: true },
  issue_label: { key: 'issueLabel', map: false },
  view: { key: 'view', map: true },
  issue_relation: { key: 'relation', map: false },
  cycle: { key: 'cycle', map: true },
  custom_field: { key: 'customField', map: true },
  custom_field_value: { key: 'cfValue', map: false },
  dashboard: { key: 'dashboard', map: true },
  dashboard_widget: { key: 'widget', map: false },
  notification: { key: 'notification', map: true },
  document: { key: 'document', map: true },
  channel: { key: 'channel', map: true },
  channel_member: { key: 'channelMember', map: false },
  channel_unread: { key: 'channelUnread', map: false },
  user: { key: 'user', map: true },
  milestone: { key: 'milestone', map: true },
  meeting: { key: 'meeting', map: true },
  meeting_participant: { key: 'meetingParticipant', map: false },
  calendar_event: { key: 'calendarEvent', map: true },
  time_entry: { key: 'timeEntry', map: false },
  automation: { key: 'automation', map: true },
  cycle_snapshot: { key: 'cycleSnapshot', map: false },
}

function empty(): Dict {
  const d: Dict = {}
  for (const { key, map } of Object.values(MAP)) d[key] = map ? {} : []
  return d
}

export let db: Dict = empty()
let cursor = 0
let loadedWs: string | null = null
const subs = new Set<() => void>()
let ver = 0
const bump = () => {
  ver++
  subs.forEach((f) => f())
}

type Chunk = { entityType: string; entities: any[] | null }
function apply(chunks: Chunk[], merge: boolean) {
  for (const c of chunks) {
    const m = MAP[c.entityType]
    if (!m) continue
    const arr = c.entities || []
    if (m.map) {
      if (merge) for (const e of arr) db[m.key][e.id] = e
      else {
        const o: Dict = {}
        for (const e of arr) o[e.id] = e
        db[m.key] = o
      }
    } else {
      db[m.key] = arr // массивы заменяем целиком (тип перезагружен)
    }
  }
}

export function isLoaded(wsId: string) {
  return loadedWs === wsId
}

export async function hydrate(wsId: string): Promise<boolean> {
  const c = await cacheGet<{ cursor: number; db: Dict }>(`snap:${wsId}`)
  if (c?.db) {
    db = { ...empty(), ...c.db }
    cursor = c.cursor
    loadedWs = wsId
    bump()
    return true
  }
  return false
}

export async function loadSnapshot(wsId: string, only?: string[]): Promise<void> {
  const tok = getCachedToken()
  if (!tok) return
  const q = only?.length ? `?only=${only.join(',')}` : ''
  const res = await fetch(`${API_BASE}/me/workspaces/${wsId}/snapshot${q}`, { headers: { Authorization: `Bearer ${tok}` } })
  if (!res.ok) throw new Error(`snapshot ${res.status}`)
  const data = (await res.json()) as { cursor: number; chunks: Chunk[] }
  if (!only) {
    db = empty()
    loadedWs = wsId
  }
  apply(data.chunks, !!only)
  cursor = data.cursor
  bump()
  cacheSet(`snap:${wsId}`, { cursor, db })
}

// Дельта-синк: Changes(since cursor) → изменённые типы → дозагрузка ?only=.
export async function syncChanges(wsId: string): Promise<void> {
  if (loadedWs !== wsId) return
  try {
    const res = await clients.sync.changes({ workspaceId: wsId, sinceCursor: BigInt(cursor) })
    if (!res.deltas.length) {
      cursor = Number(res.cursor)
      return
    }
    const types = new Set<string>()
    for (const d of res.deltas) if (MAP[d.entityType]) types.add(d.entityType)
    if (types.size) await loadSnapshot(wsId, [...types])
    cursor = Number(res.cursor)
  } catch {
    /* offline — молча */
  }
}

// optimistic-патч issue в сторе (после мутации)
export function patchIssue(id: string, fields: Dict) {
  if (db.issue[id]) {
    db.issue[id] = { ...db.issue[id], ...fields }
    bump()
  }
}

export function useStoreVersion() {
  return useSyncExternalStore(
    (cb) => {
      subs.add(cb)
      return () => subs.delete(cb)
    },
    () => ver,
    () => ver,
  )
}

// --- селекторы ---
export const S = {
  issue: (id: string) => db.issue[id] as any,
  issues: () => Object.values(db.issue) as any[],
  issuesByTeam: (teamId: string) => (Object.values(db.issue) as any[]).filter((i) => i.teamId === teamId),
  issuesByCycle: (cycleId: string) => (Object.values(db.issue) as any[]).filter((i) => i.cycleId === cycleId),
  issuesAssigned: (uid: string) => (Object.values(db.issue) as any[]).filter((i) => i.assigneeId === uid),
  issuesCreated: (uid: string) => (Object.values(db.issue) as any[]).filter((i) => i.createdBy === uid),
  team: (id: string) => db.team[id] as any,
  teams: () => (Object.values(db.team) as any[]).sort((a, b) => (a.key < b.key ? -1 : 1)),
  state: (id: string) => db.state[id] as any,
  statesByTeam: (teamId: string) => (Object.values(db.state) as any[]).filter((s) => s.teamId === teamId).sort((a, b) => (a.position < b.position ? -1 : 1)),
  user: (id: string) => db.user[id] as any,
  users: () => Object.values(db.user) as any[],
  label: (id: string) => db.label[id] as any,
  labels: () => Object.values(db.label) as any[],
  cycle: (id: string) => db.cycle[id] as any,
  cyclesByTeam: (teamId: string) => (Object.values(db.cycle) as any[]).filter((c) => c.teamId === teamId).sort((a, b) => b.number - a.number),
  milestonesByTeam: (teamId: string) => (Object.values(db.milestone) as any[]).filter((m) => m.teamId === teamId),
  docs: () => Object.values(db.document) as any[],
  doc: (id: string) => db.document[id] as any,
  channels: () => (Object.values(db.channel) as any[]).sort((a, b) => (a.name < b.name ? -1 : 1)),
  channel: (id: string) => db.channel[id] as any,
  unread: (channelId: string) => (db.channelUnread as any[]).find((u) => u.channelId === channelId)?.count ?? 0,
  dashboards: () => Object.values(db.dashboard) as any[],
  widgetsByDashboard: (dashId: string) => (db.widget as any[]).filter((w) => w.dashboardId === dashId).sort((a, b) => a.y - b.y || a.x - b.x),
  meetings: () => (Object.values(db.meeting) as any[]).sort((a, b) => (a.scheduledAt < b.scheduledAt ? 1 : -1)),
  calendarEvents: () => Object.values(db.calendarEvent) as any[],
  timeEntries: () => db.timeEntry as any[],
  automations: () => Object.values(db.automation) as any[],
  labelsForIssue: (issueId: string) => (db.issueLabel as any[]).filter((il) => il.issueId === issueId).map((il) => db.label[il.labelId]).filter(Boolean),
}
