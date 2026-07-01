// Data-слой поверх connect. RunView для списков (unary), IssueService/CommentService
// для мутаций. Issue-кэш по id — деталь читает уже загруженную строку (RunView даёт
// полный Issue включая description; отдельного GetIssue RPC нет).
import { clients } from './connect'
import { Operator } from '~/gen/pms/v1/filter_pb'

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

const issueCache = new Map<string, Issue>()
export const getIssue = (id: string) => issueCache.get(id)
function cache(list: Issue[]) {
  for (const i of list) issueCache.set(i.id, i)
}

export async function myIssues(wsId: string, uid: string, scope: IssueScope): Promise<Issue[]> {
  const field = scope === 'assigned' ? 'assignee' : 'creator'
  const res = await clients.query.runView({
    workspaceId: wsId,
    def: {
      dataSource: 'issue',
      filter: {
        node: {
          case: 'predicate',
          value: { field: { ref: { case: 'builtin', value: field } }, op: Operator.EQ, value: { v: { case: 's', value: uid } } },
        },
      },
    },
  })
  const rows = (JSON.parse(new TextDecoder().decode(res.groupedResult)) as Issue[]).sort((a, b) => (a.updatedAt < b.updatedAt ? 1 : -1))
  cache(rows)
  return rows
}

// --- мутации (quick actions). После успеха освежаем version в кэше. ---

export async function updateIssueState(wsId: string, issue: Issue, stateId: string) {
  // base_version бэком не enforce-ится, new_version не возвращается → version не трогаем
  await clients.issues.updateIssueState({ workspaceId: wsId, id: issue.id, baseVersion: BigInt(issue.version), stateId })
  issueCache.set(issue.id, { ...issue, stateId })
}

export async function assignIssue(wsId: string, issue: Issue, assigneeId: string) {
  await clients.issues.assignIssue({ workspaceId: wsId, id: issue.id, baseVersion: BigInt(issue.version), assigneeId })
  issueCache.set(issue.id, { ...issue, assigneeId: assigneeId || undefined })
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
