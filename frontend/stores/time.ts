import { defineStore } from 'pinia'
import { useClients } from '~/lib/connect'
import { useAppStore } from '~/stores/app'

// Time-стор: трекинг времени (таймер + ручные записи + отчёт). timeEntries-бакет
// живёт в entities-сторе; здесь только RPC-экшены. Читает activeWorkspaceId/catchUp
// из useAppStore.
export const useTimeStore = defineStore('time', {
  actions: {
    async startTimer(issueId = '', description = '') {
      const app = useAppStore()
      const { time } = useClients()
      await time.startTimer({ workspaceId: app.activeWorkspaceId, issueId, description })
    },
    async stopTimer() {
      const app = useAppStore()
      const { time } = useClients()
      await time.stopTimer({ workspaceId: app.activeWorkspaceId })
      await app.catchUp(app.activeWorkspaceId, ['time_entry']) // ended_at проставлен на нескольких — перечитать
    },
    async addTimeEntry(issueId: string, description: string, startedAt: string, endedAt: string) {
      const app = useAppStore()
      const { time } = useClients()
      await time.addEntry({ workspaceId: app.activeWorkspaceId, issueId, description, startedAt, endedAt })
    },
    async deleteTimeEntry(id: string) {
      const app = useAppStore()
      const { time } = useClients()
      await time.deleteEntry({ workspaceId: app.activeWorkspaceId, id })
    },
    async timeReport(p: { from: string; to: string; groupBy: string; issueId?: string; userId?: string }) {
      const app = useAppStore()
      const { time } = useClients()
      const r = await time.report({ workspaceId: app.activeWorkspaceId, from: p.from, to: p.to, groupBy: p.groupBy, issueId: p.issueId || '', userId: p.userId || '' })
      return { rows: r.rows.map((x: any) => ({ key: x.key, minutes: Number(x.minutes), count: Number(x.count) })), total: Number(r.totalMinutes) }
    },
  },
})
