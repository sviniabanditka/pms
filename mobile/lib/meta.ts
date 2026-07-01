// Метаданные воркспейса (teams/states/users/labels) — unary REST-снапшот
// (GET /me/workspaces/{id}/meta) вместо bootstrap-стрима. Даёт имена/цвета для
// issue-строк и пикеры для quick-actions/create. Кэш per-ws в памяти + reactive.
import { useSyncExternalStore } from 'react'
import { API_BASE } from './config'
import { getCachedToken } from './auth'
import { cacheGet, cacheSet } from './cache'

export type MetaTeam = { id: string; key: string; name: string }
export type MetaState = { id: string; teamId: string; category: string; name: string; colorToken: string; position: string }
export type MetaUser = { id: string; name: string; email: string }
export type MetaLabel = { id: string; teamId?: string; name: string; colorToken: string }
export type Meta = { teams: MetaTeam[]; states: MetaState[]; users: MetaUser[]; labels: MetaLabel[] }

const EMPTY: Meta = { teams: [], states: [], users: [], labels: [] }
const cache = new Map<string, Meta>()
let currentWs: string | null = null
const subs = new Set<() => void>()
const emit = () => subs.forEach((f) => f())

// status-<x> цвет-токены → hex (Tokyo-Night-ish, паритет с вебом)
const STATUS_HEX: Record<string, string> = {
  'status-backlog': '#565f89',
  'status-unstarted': '#7aa2f7',
  'status-started': '#e0af68',
  'status-completed': '#9ece6a',
  'status-canceled': '#f7768e',
}

const fetched = new Set<string>()

export async function loadMeta(wsId: string, force = false): Promise<void> {
  currentWs = wsId
  // warm-start: мгновенно поднять из SQLite, если в памяти пусто
  if (!cache.has(wsId)) {
    const c = await cacheGet<Meta>(`meta:${wsId}`)
    if (c) {
      cache.set(wsId, c)
      emit()
    }
  }
  if (fetched.has(wsId) && !force) {
    emit()
    return
  }
  const tok = getCachedToken()
  if (!tok) return
  try {
    const res = await fetch(`${API_BASE}/me/workspaces/${wsId}/meta`, { headers: { Authorization: `Bearer ${tok}` } })
    if (!res.ok) return
    const data = (await res.json()) as Meta
    cache.set(wsId, data)
    fetched.add(wsId)
    emit()
    cacheSet(`meta:${wsId}`, data)
  } catch {
    /* offline: остаёмся на SQLite-снапшоте */
  }
}

function meta(): Meta {
  return (currentWs && cache.get(currentWs)) || EMPTY
}

export const stateById = (id?: string) => (id ? meta().states.find((s) => s.id === id) : undefined)
export const userById = (id?: string) => (id ? meta().users.find((u) => u.id === id) : undefined)
export const teamById = (id?: string) => (id ? meta().teams.find((t) => t.id === id) : undefined)
export const statesForTeam = (teamId: string) => meta().states.filter((s) => s.teamId === teamId).sort((a, b) => (a.position < b.position ? -1 : 1))
export const stateColor = (colorToken?: string) => (colorToken && STATUS_HEX[colorToken]) || '#565f89'
export const allTeams = () => meta().teams
export const allUsers = () => meta().users

// reactive-хук: перерендер при загрузке меты (снапшот = ссылка на Meta текущего ws)
export function useMeta(): Meta {
  return useSyncExternalStore(
    (cb) => {
      subs.add(cb)
      return () => subs.delete(cb)
    },
    meta,
    meta,
  )
}
