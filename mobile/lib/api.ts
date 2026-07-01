// Data-слой поверх connect. M1: список «моих» issues через QueryService.RunView
// (unary; bootstrap-стрим не используем). RunView компилит FilterNode→SQL и отдаёт
// grouped_result = JSON-массив issue-строк.
import { clients } from './connect'
import { Operator } from '~/gen/pms/v1/filter_pb'

// Подмножество models.Issue (json-теги бэка). Имена state/assignee не приходят —
// только id (метаданные придут отдельным кэшем в след. фазе).
export type Issue = {
  id: string
  teamId: string
  number: number
  identifier: string
  title: string
  stateId: string
  priority: number
  assigneeId?: string
  updatedAt: string
}

export type IssueScope = 'assigned' | 'created'

export async function myIssues(wsId: string, uid: string, scope: IssueScope): Promise<Issue[]> {
  const field = scope === 'assigned' ? 'assignee' : 'creator'
  const res = await clients.query.runView({
    workspaceId: wsId,
    def: {
      dataSource: 'issue',
      filter: {
        node: {
          case: 'predicate',
          value: {
            field: { ref: { case: 'builtin', value: field } },
            op: Operator.EQ,
            value: { v: { case: 's', value: uid } },
          },
        },
      },
    },
  })
  const json = new TextDecoder().decode(res.groupedResult)
  const rows = JSON.parse(json) as Issue[]
  // самые свежие сверху
  return rows.sort((a, b) => (a.updatedAt < b.updatedAt ? 1 : -1))
}
