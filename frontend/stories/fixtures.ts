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

// issue↔label связки (компаунд-ключ issueId:labelId, как в lib/sync.keyOf)
export const issueLabels = {
  'i1:l_bug': { issueId: 'i1', labelId: 'l_bug' },
  'i1:l_feat': { issueId: 'i1', labelId: 'l_feat' },
  'i2:l_feat': { issueId: 'i2', labelId: 'l_feat' },
}

export const issues: Record<string, Issue> = {
  i1: { id: 'i1', teamId, identifier: 'ENG-1', number: 1, title: 'Fix flaky sync test', stateId: 's_started', priority: 3, assigneeId: 'user_1', boardPosition: 'a' },
  i2: { id: 'i2', teamId, identifier: 'ENG-2', number: 2, title: 'Add dark-mode toggle', stateId: 's_backlog', priority: 1, boardPosition: 'b' },
  i3: { id: 'i3', teamId, identifier: 'ENG-3', number: 3, title: 'Ship v2 release notes', stateId: 's_done', priority: 2, assigneeId: 'user_2', boardPosition: 'c' },
}

export const cycleId = 'cy_1'
export const cycles = {
  [cycleId]: { id: cycleId, teamId, number: 4, name: 'Cycle 4', startsOn: '2026-06-22', endsOn: '2026-07-06' },
}
// burndown-слепки: остаток падает 8→2 за неделю
export const cycleSnapshots = Object.fromEntries(
  [8, 7, 7, 5, 4, 2].map((remainingCount, i) => {
    const day = `2026-06-2${2 + i}`
    return [`${cycleId}:${day}`, { cycleId, day, remainingCount, remainingPoints: 0, totalPoints: 0 }]
  }),
)

export const attachments = {
  a1: { id: 'a1', entityType: 'issue', entityId: 'i1', filename: 'trace.log', s3Key: 'k1', contentType: 'text/plain', size: 24000, createdAt: '2026-07-01' },
  a2: { id: 'a2', entityType: 'issue', entityId: 'i1', filename: 'screenshot.png', s3Key: 'k2', contentType: 'image/png', size: 180000, createdAt: '2026-07-02' },
}

export const docId = 'doc_1'
export const documents = {
  [docId]: { id: docId, teamId, title: 'Onboarding', body: { md: '# Onboarding\n\nWelcome to **Engineering**.\n\n- read the runbook\n- set up your env' }, parentId: null },
  doc_2: { id: 'doc_2', teamId, title: 'Runbook', body: { md: 'On-call runbook.' }, parentId: null },
}

export const msId = 'ms_1'
export const milestones = {
  [msId]: { id: msId, teamId, name: 'v2 launch', targetDate: '2026-08-01' },
  ms_2: { id: 'ms_2', teamId, name: 'v3 planning', targetDate: '2026-10-01' },
}

export const views = {
  v_1: { id: 'v_1', teamId, name: 'My open bugs', layout: 'list', filter: {} },
  v_2: { id: 'v_2', teamId, name: 'This cycle', layout: 'board', filter: {} },
}

// Сообщение чата (DTO-форма как из GetMessages): для ChatMessage.
export const chatMessage = {
  id: 'msg_1', channelId: 'ch_1', authorId: 'user_2', body: 'Deployed **v2** to staging — please smoke-test the sync path.',
  createdAt: '2026-07-03T10:00:00Z', reactions: [{ emoji: '🚀', userId: 'user_1' }, { emoji: '🚀', userId: 'user_2' }], replyCount: 2,
}

// Снапшот для seeded-store: ключ бакета → записи. Декоратор $patch'ит его в app-стор
// + ставит activeWorkspaceId/currentUserId/loaded.
export function seedSnapshot() {
  return {
    workspaces: { [wsId]: workspace },
    teams, states, issues, issueLabels, users, members, labels, cycles, cycleSnapshots, attachments,
    documents, milestones, views,
    activeWorkspaceId: wsId,
    currentUserId: userId,
    loaded: true,
  }
}
