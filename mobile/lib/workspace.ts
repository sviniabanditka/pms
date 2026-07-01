// Активный воркспейс: persisted + reactive (useSyncExternalStore). Список грузится
// один раз на boot; активный — из storage или первый.
import { useSyncExternalStore } from 'react'
import type { Workspace } from './auth'
import { storage } from './storage'

const KEY = 'pms_ws'
let active: string | null = null
let list: Workspace[] = []
const subs = new Set<() => void>()
const emit = () => subs.forEach((f) => f())

export async function loadActiveWs(workspaces: Workspace[]) {
  list = workspaces
  const saved = await storage.get(KEY)
  active = saved && workspaces.some((w) => w.id === saved) ? saved : (workspaces[0]?.id ?? null)
  if (active) await storage.set(KEY, active)
  emit()
}

export function setActiveWs(id: string) {
  active = id
  storage.set(KEY, id)
  emit()
}

export const getActiveWs = () => active
export const getWorkspaces = () => list

// hook возвращает id (примитив — стабильный снапшот для useSyncExternalStore)
export function useActiveWs() {
  return useSyncExternalStore(
    (cb) => {
      subs.add(cb)
      return () => subs.delete(cb)
    },
    getActiveWs,
    getActiveWs,
  )
}
