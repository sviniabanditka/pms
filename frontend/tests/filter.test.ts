import { describe, it, expect } from 'vitest'
import { match, applyFilter, type FilterNode } from '../lib/filter'

const issue = (o: Partial<Record<string, any>>) => ({ priority: 0, title: '', stateId: '', assigneeId: null, ...o })

describe('Filter AST match', () => {
  it('null filter matches all', () => {
    expect(match(null, issue({}))).toBe(true)
  })

  it('eq predicate', () => {
    const f: FilterNode = { type: 'predicate', field: 'priority', op: 'eq', value: 3 }
    expect(match(f, issue({ priority: 3 }))).toBe(true)
    expect(match(f, issue({ priority: 2 }))).toBe(false)
  })

  it('in predicate', () => {
    const f: FilterNode = { type: 'predicate', field: 'priority', op: 'in', value: [3, 4] }
    expect(match(f, issue({ priority: 4 }))).toBe(true)
    expect(match(f, issue({ priority: 1 }))).toBe(false)
  })

  it('contains on title (case-insensitive)', () => {
    const f: FilterNode = { type: 'predicate', field: 'title', op: 'contains', value: 'bug' }
    expect(match(f, issue({ title: 'Fix BUG now' }))).toBe(true)
    expect(match(f, issue({ title: 'feature' }))).toBe(false)
  })

  it('isSet / isNotSet on assignee', () => {
    expect(match({ type: 'predicate', field: 'assignee', op: 'isSet' }, issue({ assigneeId: 'u1' }))).toBe(true)
    expect(match({ type: 'predicate', field: 'assignee', op: 'isNotSet' }, issue({ assigneeId: null }))).toBe(true)
  })

  it('and / or / not', () => {
    const and: FilterNode = { type: 'and', children: [
      { type: 'predicate', field: 'priority', op: 'in', value: [0, 1, 2, 3, 4] },
      { type: 'predicate', field: 'title', op: 'contains', value: 'x' },
    ] }
    expect(match(and, issue({ title: 'xy' }))).toBe(true)
    expect(match(and, issue({ title: 'ab' }))).toBe(false)
    expect(match({ type: 'not', child: { type: 'predicate', field: 'priority', op: 'eq', value: 1 } }, issue({ priority: 2 }))).toBe(true)
    expect(match({ type: 'or', children: [] }, issue({}))).toBe(true)
  })

  it('applyFilter filters a list', () => {
    const list = [issue({ priority: 4 }), issue({ priority: 1 }), issue({ priority: 4 })]
    const f: FilterNode = { type: 'predicate', field: 'priority', op: 'eq', value: 4 }
    expect(applyFilter(list, f).length).toBe(2)
  })
})
