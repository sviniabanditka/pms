// Тонкий адаптер над local-first стором (lib/store.ts). Сохраняет прежний API
// (stateById/userById/teamById/…), которым пользуются экраны, но данные — из стора.
import { C } from './theme'
import { hydrate, isLoaded, loadSnapshot, S, syncChanges, useStoreVersion } from './store'

export type MetaTeam = { id: string; key: string; name: string }
export type MetaState = { id: string; teamId: string; category: string; name: string; colorToken: string; position: string }
export type MetaUser = { id: string; name: string; email: string }

// status-<x> цвет-токены → hex (Tokyo-Night, паритет с вебом)
const STATUS_HEX: Record<string, string> = {
  'status-backlog': '#565f89',
  'status-unstarted': '#7aa2f7',
  'status-started': '#e0af68',
  'status-completed': '#9ece6a',
  'status-canceled': '#f7768e',
}

// первичная загрузка воркспейса: warm из кэша → snapshot по сети (один раз)
export async function loadMeta(wsId: string): Promise<void> {
  if (isLoaded(wsId)) {
    syncChanges(wsId) // фоновая догонялка дельт
    return
  }
  await hydrate(wsId) // мгновенно из SQLite, если было
  await loadSnapshot(wsId) // свежий полный снапшот
}

export const stateById = (id?: string) => (id ? S.state(id) : undefined)
export const userById = (id?: string) => (id ? S.user(id) : undefined)
export const teamById = (id?: string) => (id ? S.team(id) : undefined)
export const statesForTeam = (teamId: string) => S.statesByTeam(teamId)
export const stateColor = (colorToken?: string) => (colorToken && STATUS_HEX[colorToken]) || C.tertiary
export const allTeams = () => S.teams()
export const allUsers = () => S.users()

// реактивность: перерендер при любом изменении стора
export function useMeta() {
  return useStoreVersion()
}
