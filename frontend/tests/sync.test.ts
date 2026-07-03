import { describe, it, expect } from 'vitest'
import { applyChunk, keyOf, emptyBuckets, BUCKET, FULL_BUCKETS } from '~/lib/sync'

describe('lib/sync keyOf', () => {
  it('uses id by default', () => {
    expect(keyOf('issue', { id: 'i1' })).toBe('i1')
  })
  it('compound key for link entities without own id', () => {
    expect(keyOf('issue_label', { issueId: 'i1', labelId: 'l1' })).toBe('i1:l1')
    expect(keyOf('channel_unread', { channelId: 'c1' })).toBe('c1')
    expect(keyOf('cycle_snapshot', { cycleId: 'cy1', day: '2026-07-03' })).toBe('cy1:2026-07-03')
  })
})

describe('lib/sync applyChunk', () => {
  it('upserts into the mapped bucket', () => {
    const s = emptyBuckets() as any
    applyChunk(s, 'issue', [{ id: 'i1', title: 'A' }, { id: 'i2', title: 'B' }])
    expect(Object.keys(s.issues)).toEqual(['i1', 'i2'])
    expect(s.issues.i2.title).toBe('B')
  })

  it('ignores unknown entity types and non-arrays', () => {
    const s = emptyBuckets() as any
    applyChunk(s, 'nope', [{ id: 'x' }])
    applyChunk(s, 'issue', null as any)
    expect(Object.keys(s.issues)).toEqual([])
  })

  it('D7-merge keeps object reference for unchanged entities', () => {
    const s = emptyBuckets() as any
    applyChunk(s, 'team', [{ id: 't1', name: 'X' }])
    const ref = s.teams.t1
    applyChunk(s, 'team', [{ id: 't1', name: 'X' }]) // identical
    expect(s.teams.t1).toBe(ref) // same reference, not rewritten
    applyChunk(s, 'team', [{ id: 't1', name: 'Y' }]) // changed
    expect(s.teams.t1).not.toBe(ref)
    expect(s.teams.t1.name).toBe('Y')
  })

  it('FULL_BUCKETS prune drops entities missing from the chunk', () => {
    const s = emptyBuckets() as any
    applyChunk(s, 'label', [{ id: 'l1' }, { id: 'l2' }])
    applyChunk(s, 'label', [{ id: 'l1' }]) // l2 gone
    expect(Object.keys(s.labels)).toEqual(['l1'])
  })

  it('non-FULL bucket (notification) does NOT prune', () => {
    const s = emptyBuckets() as any
    expect(FULL_BUCKETS.has('notification')).toBe(false)
    applyChunk(s, 'notification', [{ id: 'n1' }, { id: 'n2' }])
    applyChunk(s, 'notification', [{ id: 'n1' }])
    expect(Object.keys(s.notifications).sort()).toEqual(['n1', 'n2']) // kept
  })

  it('preserves lazily-loaded issue.description when a chunk omits it', () => {
    const s = emptyBuckets() as any
    s.issues.i1 = { id: 'i1', title: 'A', description: 'loaded' }
    applyChunk(s, 'issue', [{ id: 'i1', title: 'A' }]) // no description field
    expect(s.issues.i1.description).toBe('loaded')
  })

  it('BUCKET maps every entity type to a real bucket in emptyBuckets', () => {
    const s = emptyBuckets() as any
    for (const [et, bucket] of Object.entries(BUCKET)) {
      expect(s[bucket], `bucket for ${et}`).toBeDefined()
    }
  })
})
