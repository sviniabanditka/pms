// Filter AST — клиентский исполнитель над локальным стором (local-first, §6.2).
// Сериализуем (JSON) → хранится в View.filter, едет в CreateView.

export type Op = 'eq' | 'neq' | 'in' | 'nin' | 'isSet' | 'isNotSet' | 'contains'

export type FilterNode =
  | { type: 'and'; children: FilterNode[] }
  | { type: 'or'; children: FilterNode[] }
  | { type: 'not'; child: FilterNode }
  | { type: 'predicate'; field: string; op: Op; value?: any }

// Доступ к полям issue (camelCase из стора).
const FIELD: Record<string, (i: any) => any> = {
  status: (i) => i.stateId,
  priority: (i) => i.priority ?? 0,
  assignee: (i) => i.assigneeId ?? null,
  title: (i) => i.title ?? '',
  project: (i) => i.projectId ?? null,
  cycle: (i) => i.cycleId ?? null,
}

export function match(node: FilterNode | null | undefined, issue: any): boolean {
  if (!node) return true
  switch (node.type) {
    case 'and':
      return node.children.every((c) => match(c, issue))
    case 'or':
      return node.children.length === 0 ? true : node.children.some((c) => match(c, issue))
    case 'not':
      return !match(node.child, issue)
    case 'predicate': {
      const get = FIELD[node.field]
      const v = get ? get(issue) : undefined
      switch (node.op) {
        case 'eq': return v === node.value
        case 'neq': return v !== node.value
        case 'in': return Array.isArray(node.value) && node.value.includes(v)
        case 'nin': return Array.isArray(node.value) && !node.value.includes(v)
        case 'isSet': return v != null && v !== ''
        case 'isNotSet': return v == null || v === ''
        case 'contains': return String(v).toLowerCase().includes(String(node.value).toLowerCase())
        default: return true
      }
    }
    default:
      return true
  }
}

export function applyFilter<T>(issues: T[], node: FilterNode | null | undefined): T[] {
  if (!node) return issues
  return issues.filter((i) => match(node, i))
}

// and(...) хелпер для сборки FilterBar.
export function and(children: FilterNode[]): FilterNode {
  return { type: 'and', children }
}
