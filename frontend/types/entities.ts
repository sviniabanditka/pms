// Типы сущностей local-first стора. Сущности приходят из JSON.parse(bootstrap/дельта),
// поля camelCase (backend отдаёт camelCase). Индекс-сигнатура [k: string]: any —
// осознанный escape hatch: декодированные сущности могут нести поля, которых нет в
// интерфейсе (ленивые/производные), и не должны падать. Известные поля — для автокомплита.

// Entity — нетипизированная сущность (для мест, где конкретный тип не важен: myWorkspaces,
// generic-декод). Постепенно вытесняется конкретными интерфейсами ниже.
export type Entity = Record<string, any>

export interface BaseEntity {
  id: string
  [k: string]: any
}

export interface Issue extends BaseEntity {
  teamId: string
  number?: number
  identifier?: string
  title: string
  description?: any
  stateId: string
  priority?: number
  assigneeId?: string | null
  parentId?: string | null
  cycleId?: string | null
  milestoneId?: string | null
  estimate?: number | null
  boardPosition?: string
  scheduledDate?: string | null
  dueDate?: string | null
  archivedAt?: string | null
  deletedAt?: string | null
  version?: number
}

export interface Team extends BaseEntity {
  workspaceId?: string
  path?: string
  key: string
  name: string
  private?: boolean
  deletedAt?: string | null
}

export interface TeamMember extends BaseEntity {
  teamId: string
  userId: string
  role?: string
}

export interface WorkflowState extends BaseEntity {
  teamId: string
  category: string
  name: string
  colorToken?: string
  position?: string
}

export interface Label extends BaseEntity {
  teamId?: string | null
  name: string
  colorToken?: string
}

export interface IssueLabel { issueId: string; labelId: string; [k: string]: any }

export interface IssueRelation extends BaseEntity {
  sourceId: string
  targetId: string
  type: string
}

export interface View extends BaseEntity {
  name: string
  teamId?: string
  layout?: string
  filter?: any
  deletedAt?: string | null
}

export interface Cycle extends BaseEntity {
  teamId: string
  name?: string
  startsOn: string
  endsOn: string
}

export interface CycleSnapshot { cycleId: string; day: string; [k: string]: any }

export interface Milestone extends BaseEntity {
  teamId: string
  name: string
  targetDate?: string | null
  deletedAt?: string | null
}

export interface CustomField extends BaseEntity {
  teamId?: string | null
  name: string
  type: string
  config?: any
  archivedAt?: string | null
}

export interface CustomFieldValue extends BaseEntity {
  issueId: string
  fieldId: string
  value?: any
}

export interface Dashboard extends BaseEntity {
  teamId?: string | null
  name?: string
  deletedAt?: string | null
}

export interface DashboardWidget extends BaseEntity {
  dashboardId: string
  x?: number
  y?: number
  type?: string
}

export interface Workspace extends BaseEntity {
  slug?: string
  name?: string
}

export interface WorkspaceMember extends BaseEntity {
  userId: string
  role?: string
  status?: string
}

export interface User extends BaseEntity {
  name?: string
  email?: string
}

export interface Channel extends BaseEntity {
  kind: string
  name?: string
  private?: boolean
  archivedAt?: string | null
  deletedAt?: string | null
}

export interface ChannelMember extends BaseEntity {
  channelId: string
  userId: string
}

export interface ChannelUnread { channelId: string; [k: string]: any }

export interface Message extends BaseEntity {
  channelId?: string
  body?: string
  authorId?: string
  parentId?: string | null
}

export interface Comment extends BaseEntity {
  issueId?: string
  body?: string
}

export interface Notification extends BaseEntity {
  readAt?: string | null
}

export interface Document extends BaseEntity {
  teamId?: string
  title: string
  body?: any
  parentId?: string | null
  deletedAt?: string | null
}

export interface Meeting extends BaseEntity {
  channelId?: string | null
  kind: string
  status?: string
  title?: string
  deletedAt?: string | null
}

export interface MeetingParticipant extends BaseEntity {
  meetingId: string
  userId: string
}

export interface CalendarEvent extends BaseEntity {
  deletedAt?: string | null
}

export interface TimeEntry extends BaseEntity {
  issueId?: string | null
}

export interface Automation extends BaseEntity {
  teamId?: string | null
  deletedAt?: string | null
}

export interface Attachment extends BaseEntity {
  deletedAt?: string | null
}

export interface Invitation extends BaseEntity {
  acceptedAt?: string | null
}

export interface IssueTemplate extends BaseEntity {
  teamId?: string
  name: string
  deletedAt?: string | null
}

// Bucket — ключ бакета в state (значения BUCKET-мапы). Каждый = Record<id, сущность>.
export interface Entities {
  workspaces: Record<string, Workspace>
  teams: Record<string, Team>
  teamMembers: Record<string, TeamMember>
  states: Record<string, WorkflowState>
  issues: Record<string, Issue>
  views: Record<string, View>
  relations: Record<string, IssueRelation>
  cycles: Record<string, Cycle>
  customFields: Record<string, CustomField>
  customFieldValues: Record<string, CustomFieldValue>
  dashboards: Record<string, Dashboard>
  dashboardWidgets: Record<string, DashboardWidget>
  members: Record<string, WorkspaceMember>
  notifications: Record<string, Notification>
  documents: Record<string, Document>
  channels: Record<string, Channel>
  channelMembers: Record<string, ChannelMember>
  users: Record<string, User>
  labels: Record<string, Label>
  issueLabels: Record<string, IssueLabel>
  invitations: Record<string, Invitation>
  milestones: Record<string, Milestone>
  meetings: Record<string, Meeting>
  meetingParticipants: Record<string, MeetingParticipant>
  calendarEvents: Record<string, CalendarEvent>
  timeEntries: Record<string, TimeEntry>
  automations: Record<string, Automation>
  attachments: Record<string, Attachment>
  channelUnread: Record<string, ChannelUnread>
  cycleSnapshots: Record<string, CycleSnapshot>
  issueTemplates: Record<string, IssueTemplate>
}

export type Bucket = keyof Entities
