// Типизированные фикстуры для seeded-store декоратора. Форма под typecheck-гейтом
// (types/entities.ts) — не дрейфует. Переиспользуемо и в vitest.
import type { Workspace, Team, WorkflowState, Issue, User, WorkspaceMember, Label } from '~/types/entities'

export const wsId = 'ws_1'
export const teamId = 'team_1'
export const userId = 'user_1'

export const workspace: Workspace = { id: wsId, slug: 'acme', name: 'Acme' }

export const users: Record<string, User> = {
  user_1: { id: 'user_1', name: 'Ada Lovelace', email: 'ada@acme.dev' },
  user_2: { id: 'user_2', name: 'Alan Turing', email: 'alan@acme.dev' },
}

export const members: Record<string, WorkspaceMember> = {
  m1: { id: 'm1', userId: 'user_1', role: 'owner', status: 'active' },
  m2: { id: 'm2', userId: 'user_2', role: 'member', status: 'active' },
}

export const teams: Record<string, Team> = {
  team_1: { id: teamId, workspaceId: wsId, key: 'ENG', name: 'Engineering', path: 'eng', private: false },
}

export const states: Record<string, WorkflowState> = {
  s_backlog: { id: 's_backlog', teamId, category: 'backlog', name: 'Backlog', colorToken: 'status-backlog', position: 'a' },
  s_started: { id: 's_started', teamId, category: 'started', name: 'In Progress', colorToken: 'status-started', position: 'b' },
  s_done: { id: 's_done', teamId, category: 'completed', name: 'Done', colorToken: 'status-completed', position: 'c' },
}

export const labels: Record<string, Label> = {
  l_bug: { id: 'l_bug', teamId, name: 'bug', colorToken: '#f87171' },
  l_feat: { id: 'l_feat', teamId, name: 'feature', colorToken: '#60a5fa' },
}

export const issues: Record<string, Issue> = {
  i1: { id: 'i1', teamId, identifier: 'ENG-1', number: 1, title: 'Fix flaky sync test', stateId: 's_started', priority: 3, assigneeId: 'user_1', boardPosition: 'a' },
  i2: { id: 'i2', teamId, identifier: 'ENG-2', number: 2, title: 'Add dark-mode toggle', stateId: 's_backlog', priority: 1, boardPosition: 'b' },
  i3: { id: 'i3', teamId, identifier: 'ENG-3', number: 3, title: 'Ship v2 release notes', stateId: 's_done', priority: 2, assigneeId: 'user_2', boardPosition: 'c' },
}

// Снапшот для seeded-store: ключ бакета → записи. Декоратор $patch'ит его в app-стор
// + ставит activeWorkspaceId/currentUserId/loaded.
export function seedSnapshot() {
  return {
    workspaces: { [wsId]: workspace },
    teams, states, issues, users, members, labels,
    activeWorkspaceId: wsId,
    currentUserId: userId,
    loaded: true,
  }
}
