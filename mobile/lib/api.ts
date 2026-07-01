// Data-слой поверх connect. RunView для списков (unary), IssueService/CommentService
// для мутаций. Issue-кэш по id — деталь читает уже загруженную строку (RunView даёт
// полный Issue включая description; отдельного GetIssue RPC нет).
import { clients } from './connect'
import { API_BASE } from './config'
import { getCachedToken, getCurrentUserId } from './auth'
import { cacheGet, cacheSet } from './cache'
import { isLoaded, loadSnapshot, patchIssue, S, syncChanges } from './store'

const authHeader = () => ({ Authorization: `Bearer ${getCachedToken() ?? ''}` })

export type Issue = {
  id: string
  teamId: string
  number: number
  identifier: string
  title: string
  description?: { md?: string } | null
  stateId: string
  priority: number
  assigneeId?: string
  version: number
  updatedAt: string
}

export type Comment = {
  id: string
  issueId: string
  authorId: string
  body: string
  createdAt: string
  editedAt?: string
}

export type IssueScope = 'assigned' | 'created'

// client-gen id для команд (сервер ждёт валидный uuid). ponytail: v4 (не v7-ordered) — id-уникальности хватает.
export function uuid(): string {
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, (c) => {
    const r = (Math.random() * 16) | 0
    return (c === 'x' ? r : (r & 0x3) | 0x8).toString(16)
  })
}

// issues из local-first стора (все загружены снапшотом) — фильтр локальный
export const getIssue = (id: string): Issue | undefined => S.issue(id)

export async function myIssues(_wsId: string, uid: string, scope: IssueScope): Promise<Issue[]> {
  const rows = (scope === 'assigned' ? S.issuesAssigned(uid) : S.issuesCreated(uid)) as Issue[]
  return rows.slice().sort((a, b) => (a.updatedAt < b.updatedAt ? 1 : -1))
}

// offline тоже покрыт стором (hydrate из кэша) → тот же источник (uid из auth)
export async function cachedIssues(wsId: string, scope: IssueScope): Promise<Issue[]> {
  return myIssues(wsId, getCurrentUserId() ?? '', scope)
}

// --- мутации (quick actions): RPC + optimistic-патч стора ---

export async function updateIssueState(wsId: string, issue: Issue, stateId: string) {
  await clients.issues.updateIssueState({ workspaceId: wsId, id: issue.id, baseVersion: BigInt(issue.version), stateId })
  patchIssue(issue.id, { stateId })
}

export async function assignIssue(wsId: string, issue: Issue, assigneeId: string) {
  await clients.issues.assignIssue({ workspaceId: wsId, id: issue.id, baseVersion: BigInt(issue.version), assigneeId })
  patchIssue(issue.id, { assigneeId: assigneeId || undefined })
}

export async function createIssue(wsId: string, p: { teamId: string; title: string; stateId?: string; priority?: number; assigneeId?: string }): Promise<string> {
  const id = uuid()
  await clients.issues.createIssue({
    workspaceId: wsId,
    id,
    teamId: p.teamId,
    title: p.title,
    stateId: p.stateId ?? '',
    priority: p.priority ?? 0,
    assigneeId: p.assigneeId ?? '',
  })
  syncChanges(wsId) // подтянуть новый issue в стор
  return id
}

// --- комментарии ---

export async function getComments(wsId: string, issueId: string): Promise<Comment[]> {
  const res = await clients.comments.getComments({ workspaceId: wsId, issueId })
  const arr = JSON.parse(new TextDecoder().decode(res.comments)) as Comment[]
  return arr.sort((a, b) => (a.createdAt < b.createdAt ? -1 : 1))
}

export async function addComment(wsId: string, issueId: string, body: string) {
  await clients.comments.addComment({ workspaceId: wsId, id: uuid(), issueId, body })
}

// одиночный issue по id (для tap уведомления/пуша) — из стора; если стор не загружен,
// подтягиваем снапшот.
export async function issueById(wsId: string, id: string): Promise<Issue | undefined> {
  let iss = S.issue(id) as Issue | undefined
  if (!iss && !isLoaded(wsId)) {
    await loadSnapshot(wsId).catch(() => {})
    iss = S.issue(id)
  }
  return iss
}

// --- уведомления (inbox) ---

export type Notification = {
  id: string
  type: string
  entityType: string
  entityId: string
  actorId?: string
  readAt?: string
  at: string
}

export async function listNotifications(wsId: string): Promise<Notification[]> {
  const res = await fetch(`${API_BASE}/me/workspaces/${wsId}/notifications`, { headers: authHeader() })
  if (!res.ok) throw new Error(`notifications ${res.status}`)
  const list = ((await res.json()) as Notification[]) ?? []
  cacheSet(`notifications:${wsId}`, list)
  return list
}

export async function cachedNotifications(wsId: string): Promise<Notification[]> {
  return (await cacheGet<Notification[]>(`notifications:${wsId}`)) ?? []
}

export async function markRead(wsId: string, id: string) {
  await clients.notifications.markRead({ workspaceId: wsId, id })
}

export async function markAllRead(wsId: string) {
  await clients.notifications.markAllRead({ workspaceId: wsId })
}

// --- push device registration (REST) ---

export async function registerDevice(token: string, platform: string) {
  await fetch(`${API_BASE}/me/devices`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...authHeader() },
    body: JSON.stringify({ token, platform }),
  })
}

export async function unregisterDevice(token: string) {
  await fetch(`${API_BASE}/me/devices/${encodeURIComponent(token)}`, { method: 'DELETE', headers: authHeader() })
}
