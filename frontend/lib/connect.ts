import { useRuntimeConfig } from '#imports'
import { createClient, Code, ConnectError, type Interceptor } from '@connectrpc/connect'
import { createConnectTransport } from '@connectrpc/connect-web'
import { handleUnauthorized } from './token'
import { SyncService } from '~/gen/pms/v1/sync_service_pb'
import { WorkspaceService } from '~/gen/pms/v1/workspace_service_pb'
import { TeamService } from '~/gen/pms/v1/team_service_pb'
import { IssueService } from '~/gen/pms/v1/issue_service_pb'
import { ViewService } from '~/gen/pms/v1/view_service_pb'
import { CycleService } from '~/gen/pms/v1/cycle_service_pb'
import { CustomFieldService } from '~/gen/pms/v1/custom_field_service_pb'
import { DashboardService } from '~/gen/pms/v1/dashboard_service_pb'
import { NotificationService } from '~/gen/pms/v1/notification_service_pb'
import { DocService } from '~/gen/pms/v1/doc_service_pb'
import { ChatService } from '~/gen/pms/v1/chat_service_pb'
import { CommentService } from '~/gen/pms/v1/comment_service_pb'
import { MilestoneService } from '~/gen/pms/v1/milestone_service_pb'
import { MeetingService } from '~/gen/pms/v1/meeting_service_pb'
import { CalendarService } from '~/gen/pms/v1/calendar_service_pb'
import { TimeService } from '~/gen/pms/v1/time_service_pb'
import { AutomationService } from '~/gen/pms/v1/automation_service_pb'
import { AttachmentService } from '~/gen/pms/v1/attachment_service_pb'
import { QueryService } from '~/gen/pms/v1/query_service_pb'

// Протухшая сессия (Unauthenticated на любом RPC) → редирект на /login.
const authGuard: Interceptor = (next) => async (req) => {
  try {
    return await next(req)
  } catch (e) {
    if (e instanceof ConnectError && e.code === Code.Unauthenticated) handleUnauthorized()
    throw e
  }
}

// useClients — connect-клиенты с cookie-аутентификацией (credentials:'include' → шлёт httpOnly-cookie).
export function useClients() {
  const transport = createConnectTransport({
    baseUrl: useRuntimeConfig().public.apiBase as string,
    fetch: (input, init) => fetch(input, { ...init, credentials: 'include' }),
    interceptors: [authGuard],
  })
  return {
    sync: createClient(SyncService, transport),
    workspaces: createClient(WorkspaceService, transport),
    teams: createClient(TeamService, transport),
    issues: createClient(IssueService, transport),
    views: createClient(ViewService, transport),
    cycles: createClient(CycleService, transport),
    customFields: createClient(CustomFieldService, transport),
    dashboards: createClient(DashboardService, transport),
    notifications: createClient(NotificationService, transport),
    docs: createClient(DocService, transport),
    chats: createClient(ChatService, transport),
    comments: createClient(CommentService, transport),
    milestones: createClient(MilestoneService, transport),
    meetings: createClient(MeetingService, transport),
    calendar: createClient(CalendarService, transport),
    time: createClient(TimeService, transport),
    automations: createClient(AutomationService, transport),
    attachments: createClient(AttachmentService, transport),
    query: createClient(QueryService, transport),
  }
}
