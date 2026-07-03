// Sync-ядро local-first стора, вынесенное из stores/app.ts как чистые функции —
// unit-тестируемо без Pinia/Vue. Стор передаёт свой state-контейнер, функции его мутируют
// (реактивность Pinia ловит мутации по ключам бакетов).
import type { Entities, Bucket } from '~/types/entities'

// entity_type (wire, snake_case) → ключ бакета в state (camelCase).
export const BUCKET: Record<string, Bucket> = {
  workspace: 'workspaces',
  team: 'teams',
  team_member: 'teamMembers',
  workflow_state: 'states',
  issue: 'issues',
  view: 'views',
  issue_relation: 'relations',
  cycle: 'cycles',
  custom_field: 'customFields',
  custom_field_value: 'customFieldValues',
  dashboard: 'dashboards',
  dashboard_widget: 'dashboardWidgets',
  workspace_member: 'members',
  notification: 'notifications',
  document: 'documents',
  channel: 'channels',
  channel_member: 'channelMembers',
  user: 'users',
  label: 'labels',
  issue_label: 'issueLabels',
  invitation: 'invitations',
  milestone: 'milestones',
  meeting: 'meetings',
  meeting_participant: 'meetingParticipants',
  calendar_event: 'calendarEvents',
  time_entry: 'timeEntries',
  automation: 'automations',
  attachment: 'attachments',
  channel_unread: 'channelUnread',
  cycle_snapshot: 'cycleSnapshots',
  issue_template: 'issueTemplates',
}

// Полностью присылаемые в bootstrap справочники → отсутствующие можно удалять из стора.
// notification исключён (лимит 100), message не в bootstrap.
export const FULL_BUCKETS = new Set([
  'issue', 'team', 'workflow_state', 'label', 'issue_label', 'issue_relation', 'cycle',
  'view', 'custom_field', 'custom_field_value', 'dashboard', 'dashboard_widget',
  'document', 'channel', 'channel_member', 'workspace_member', 'team_member', 'user', 'invitation', 'milestone',
  'meeting', 'meeting_participant', 'calendar_event', 'time_entry', 'automation', 'attachment', 'channel_unread', 'cycle_snapshot', 'issue_template',
])

// keyOf — ключ сущности в бакете. У связок нет своего id → компаунд-ключ.
export function keyOf(entityType: string, e: any): string {
  switch (entityType) {
    case 'issue_label': return `${e.issueId}:${e.labelId}`
    case 'channel_unread': return e.channelId
    case 'cycle_snapshot': return `${e.cycleId}:${e.day}`
    default: return e.id
  }
}

// applyChunk — влить пачку сущностей одного типа в state-контейнер (мутирует его).
// D7-merge: пишем только реально изменившиеся (по JSON) → нет реактивного churn при
// ре-bootstrap; неизменённые сохраняют ссылку. Для FULL_BUCKETS — подчистка пропавших.
export function applyChunk(state: Record<string, Record<string, any>>, entityType: string, arr: any[]): void {
  const bucket = BUCKET[entityType]
  if (!bucket || !Array.isArray(arr)) return
  const b = state[bucket]
  const seen = new Set<string>()
  for (const e of arr) {
    const k = keyOf(entityType, e)
    seen.add(k)
    const cur = b[k]
    // issue.description грузится лениво (bootstrap/дельта его не шлют) → сохраняем уже загруженное
    if (entityType === 'issue' && cur && e.description === undefined && cur.description) e.description = cur.description
    if (cur && JSON.stringify(cur) === JSON.stringify(e)) continue
    b[k] = e
  }
  if (FULL_BUCKETS.has(entityType)) {
    for (const k of Object.keys(b)) if (!seen.has(k)) delete b[k]
  }
}

// Пустые бакеты (для сброса среза при смене воркспейса и инициализации state).
export function emptyBuckets(): Pick<Entities, Bucket> {
  return {
    workspaces: {}, teams: {}, teamMembers: {}, states: {}, issues: {}, views: {}, relations: {},
    cycles: {}, customFields: {}, customFieldValues: {}, dashboards: {}, dashboardWidgets: {},
    members: {}, notifications: {}, documents: {}, channels: {}, channelMembers: {}, users: {},
    labels: {}, issueLabels: {}, invitations: {}, milestones: {}, meetings: {}, meetingParticipants: {},
    calendarEvents: {}, timeEntries: {}, automations: {}, attachments: {}, channelUnread: {},
    cycleSnapshots: {}, issueTemplates: {},
  }
}
