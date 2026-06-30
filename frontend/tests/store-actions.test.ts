import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'

// Мокаем connect-клиенты: actions делают оптимистичную мутацию стора + сетевой вызов.
// Проверяем локальную мутацию и что соответствующий RPC вызван.
const calls: Record<string, any[]> = {}
const rec = (name: string) => (arg: any) => {
  ;(calls[name] ||= []).push(arg)
  if (name === 'createLabel') return Promise.resolve({ entityId: 'L9' })
  return Promise.resolve({})
}

vi.mock('~/lib/connect', () => ({
  useClients: () => ({
    teams: {
      removeTeamMember: rec('removeTeamMember'),
      leaveTeam: rec('leaveTeam'),
      addTeamMember: rec('addTeamMember'),
      createLabel: rec('createLabel'),
    },
    workspaces: {
      revokeInvitation: rec('revokeInvitation'),
      updateMemberRole: rec('updateMemberRole'),
    },
  }),
}))

import { useAppStore } from '../stores/app'

beforeEach(() => {
  setActivePinia(createPinia())
  for (const k of Object.keys(calls)) delete calls[k]
})

describe('store actions — optimistic mutations + RPC', () => {
  it('removeTeamMember deletes the local row and calls RPC', async () => {
    const s = useAppStore()
    s.activeWorkspaceId = 'ws1'
    s.teamMembers = { tm1: { id: 'tm1', teamId: 't1', userId: 'u1', role: 'member' } }
    await s.removeTeamMember('t1', 'u1')
    expect(s.teamMembers.tm1).toBeUndefined()
    expect(calls.removeTeamMember[0]).toMatchObject({ workspaceId: 'ws1', teamId: 't1', userId: 'u1' })
  })

  it('leaveTeam removes only the current user from the team', async () => {
    const s = useAppStore()
    s.activeWorkspaceId = 'ws1'
    s.currentUserId = 'me'
    s.teamMembers = {
      a: { id: 'a', teamId: 't1', userId: 'me', role: 'member' },
      b: { id: 'b', teamId: 't1', userId: 'other', role: 'lead' },
    }
    await s.leaveTeam('t1')
    expect(s.teamMembers.a).toBeUndefined()
    expect(s.teamMembers.b).toBeTruthy()
    expect(calls.leaveTeam[0]).toMatchObject({ teamId: 't1' })
  })

  it('revokeInvitation deletes local invite and calls RPC', async () => {
    const s = useAppStore()
    s.activeWorkspaceId = 'ws1'
    s.invitations = { i1: { id: 'i1', acceptedAt: null } }
    await s.revokeInvitation('i1')
    expect(s.invitations.i1).toBeUndefined()
    expect(calls.revokeInvitation[0]).toMatchObject({ id: 'i1' })
  })

  it('updateMemberRole mutates local role and calls RPC', async () => {
    const s = useAppStore()
    s.activeWorkspaceId = 'ws1'
    s.members = { m1: { id: 'm1', userId: 'u1', role: 'member' } }
    await s.updateMemberRole('u1', 'admin')
    expect(s.members.m1.role).toBe('admin')
    expect(calls.updateMemberRole[0]).toMatchObject({ userId: 'u1', role: 'admin' })
  })

  it('createLabel stores the new label from the RPC result', async () => {
    const s = useAppStore()
    s.activeWorkspaceId = 'ws1'
    const id = await s.createLabel('t1', 'bug', '#f00')
    expect(id).toBe('L9')
    expect(s.labels.L9).toMatchObject({ teamId: 't1', name: 'bug', colorToken: '#f00' })
    expect(calls.createLabel[0]).toMatchObject({ teamId: 't1', name: 'bug' })
  })
})
