import { describe, it, expect, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useAppStore } from '../stores/app'

beforeEach(() => setActivePinia(createPinia()))

function withRole(role: string) {
  const s = useAppStore()
  s.currentUserId = 'u1'
  s.members = { m1: { id: 'm1', userId: 'u1', role } }
  return s
}

describe('store — role getters', () => {
  it('myRole reflects current user membership', () => {
    expect(withRole('admin').myRole).toBe('admin')
    expect(withRole('member').myRole).toBe('member')
  })
  it('isWsOwner only for owner', () => {
    expect(withRole('owner').isWsOwner).toBe(true)
    expect(withRole('admin').isWsOwner).toBe(false)
  })
  it('isWsAdmin for owner and admin', () => {
    expect(withRole('owner').isWsAdmin).toBe(true)
    expect(withRole('admin').isWsAdmin).toBe(true)
    expect(withRole('member').isWsAdmin).toBe(false)
    expect(withRole('guest').isWsAdmin).toBe(false)
  })
  it('canWrite for member+ but not guest', () => {
    expect(withRole('member').canWrite).toBe(true)
    expect(withRole('admin').canWrite).toBe(true)
    expect(withRole('guest').canWrite).toBe(false)
    expect(withRole('guest').isGuest).toBe(true)
  })
})

describe('store — canManageTeam (ltree inheritance)', () => {
  function seedTeams(role: string) {
    const s = useAppStore()
    s.currentUserId = 'u1'
    s.members = { m1: { id: 'm1', userId: 'u1', role } }
    s.teams = {
      parent: { id: 'parent', path: 'p', deletedAt: null },
      child: { id: 'child', path: 'p.c', deletedAt: null },
      other: { id: 'other', path: 'o', deletedAt: null },
    }
    s.teamMembers = { tm1: { id: 'tm1', teamId: 'parent', userId: 'u1', role: 'lead' } }
    return s
  }
  it('lead manages own team and nested subteams; not unrelated', () => {
    const s = seedTeams('member')
    expect(s.canManageTeam('parent')).toBe(true)
    expect(s.canManageTeam('child')).toBe(true)
    expect(s.canManageTeam('other')).toBe(false)
  })
  it('a non-lead member manages nothing', () => {
    const s = seedTeams('member')
    s.teamMembers = { tm1: { id: 'tm1', teamId: 'parent', userId: 'u1', role: 'member' } }
    expect(s.canManageTeam('parent')).toBe(false)
  })
  it('ws-admin manages any team (bypass)', () => {
    const s = seedTeams('admin')
    s.teamMembers = {}
    expect(s.canManageTeam('other')).toBe(true)
  })
  it('prefix match requires a dot boundary (p must not match p2)', () => {
    const s = seedTeams('member')
    s.teams = { p2: { id: 'p2', path: 'p2', deletedAt: null }, ...s.teams }
    expect(s.canManageTeam('p2')).toBe(false)
  })
})

describe('store — derived indexes & lists', () => {
  it('pendingInvites excludes accepted', () => {
    const s = useAppStore()
    s.invitations = {
      i1: { id: 'i1', acceptedAt: null },
      i2: { id: 'i2', acceptedAt: '2026-01-01' },
    }
    expect(s.pendingInvites.map((i: any) => i.id)).toEqual(['i1'])
  })
  it('activeIssuesByTeam groups non-archived/deleted by team', () => {
    const s = useAppStore()
    s.issues = {
      a: { id: 'a', teamId: 't1', archivedAt: null, deletedAt: null },
      b: { id: 'b', teamId: 't1', archivedAt: 'x', deletedAt: null },
      c: { id: 'c', teamId: 't1', archivedAt: null, deletedAt: 'x' },
      d: { id: 'd', teamId: 't2', archivedAt: null, deletedAt: null },
    }
    const m = s.activeIssuesByTeam
    expect(m.get('t1')?.length).toBe(1)
    expect(m.get('t2')?.length).toBe(1)
  })
  it('labelsByIssue maps issue → set of label ids', () => {
    const s = useAppStore()
    s.issueLabels = {
      k1: { issueId: 'a', labelId: 'L1' },
      k2: { issueId: 'a', labelId: 'L2' },
      k3: { issueId: 'b', labelId: 'L1' },
    }
    const m = s.labelsByIssue
    expect(m.get('a')?.size).toBe(2)
    expect(m.get('b')?.size).toBe(1)
  })
  it('teamMembersOf filters by team', () => {
    const s = useAppStore()
    s.teamMembers = {
      x: { id: 'x', teamId: 't1', userId: 'u1', role: 'lead' },
      y: { id: 'y', teamId: 't2', userId: 'u2', role: 'member' },
    }
    expect(s.teamMembersOf('t1').length).toBe(1)
    expect(s.teamMembersOf('t1')[0].userId).toBe('u1')
  })
})

describe('store — applyChunk (sync core)', () => {
  it('keys entities by id and prunes missing for FULL_BUCKETS', () => {
    const s = useAppStore()
    s.applyChunk('issue', [{ id: 'a', title: '1' }, { id: 'b', title: '2' }])
    expect(Object.keys(s.issues).sort()).toEqual(['a', 'b'])
    // issue is a FULL_BUCKET → a second chunk without 'b' prunes it
    s.applyChunk('issue', [{ id: 'a', title: '1' }])
    expect(Object.keys(s.issues)).toEqual(['a'])
  })

  it('uses composite key for issue_label (issueId:labelId)', () => {
    const s = useAppStore()
    s.applyChunk('issue_label', [{ issueId: 'i1', labelId: 'L1' }, { issueId: 'i1', labelId: 'L2' }])
    expect(Object.keys(s.issueLabels).sort()).toEqual(['i1:L1', 'i1:L2'])
  })

  it('uses composite key for cycle_snapshot (cycleId:day)', () => {
    const s = useAppStore()
    s.applyChunk('cycle_snapshot', [{ cycleId: 'c1', day: '2026-06-30' }])
    expect(s.cycleSnapshots['c1:2026-06-30']).toBeTruthy()
  })

  it('preserves lazily-loaded issue description when a chunk omits it', () => {
    const s = useAppStore()
    s.issues = { a: { id: 'a', title: 't', description: { md: 'kept' } } }
    s.applyChunk('issue', [{ id: 'a', title: 't2' }]) // no description in the chunk
    expect(s.issues.a.title).toBe('t2')
    expect(s.issues.a.description).toEqual({ md: 'kept' })
  })

  it('routes team_member chunk into teamMembers bucket', () => {
    const s = useAppStore()
    s.applyChunk('team_member', [{ id: 'tm1', teamId: 't1', userId: 'u1', role: 'lead' }])
    expect(s.teamMembers.tm1).toBeTruthy()
  })

  it('ignores unknown entity types', () => {
    const s = useAppStore()
    s.applyChunk('bogus', [{ id: 'x' }])
    // nothing thrown, no bucket created
    expect((s as any).bogus).toBeUndefined()
  })
})
