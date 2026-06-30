// ConnectRPC-клиенты для нативного клиента. Зеркало frontend/lib/connect.ts, но
// auth = Bearer-токен (не cookie), формат JSON (надёжнее на RN fetch, чем binary).
import { createClient, Code, ConnectError, type Interceptor } from '@connectrpc/connect'
import { createConnectTransport } from '@connectrpc/connect-web'
import { API_BASE } from './config'
import { getCachedToken } from './auth'

import { WorkspaceService } from '../gen/pms/v1/workspace_service_pb'
import { TeamService } from '../gen/pms/v1/team_service_pb'
import { IssueService } from '../gen/pms/v1/issue_service_pb'
import { QueryService } from '../gen/pms/v1/query_service_pb'
import { NotificationService } from '../gen/pms/v1/notification_service_pb'
import { CommentService } from '../gen/pms/v1/comment_service_pb'

// Колбэк на протухший токен — ставит auth-гейт (_layout), чтобы увести на /login.
let onUnauthorized: (() => void) | null = null
export function setOnUnauthorized(fn: () => void) {
  onUnauthorized = fn
}

const bearer: Interceptor = (next) => async (req) => {
  const tok = getCachedToken()
  if (tok) req.header.set('Authorization', `Bearer ${tok}`)
  try {
    return await next(req)
  } catch (e) {
    if (e instanceof ConnectError && e.code === Code.Unauthenticated) onUnauthorized?.()
    throw e
  }
}

const transport = createConnectTransport({
  baseUrl: API_BASE,
  useBinaryFormat: false, // ponytail: JSON безопаснее на RN fetch; включить binary если важен размер
  interceptors: [bearer],
})

export const clients = {
  workspaces: createClient(WorkspaceService, transport),
  teams: createClient(TeamService, transport),
  issues: createClient(IssueService, transport),
  query: createClient(QueryService, transport),
  notifications: createClient(NotificationService, transport),
  comments: createClient(CommentService, transport),
}
