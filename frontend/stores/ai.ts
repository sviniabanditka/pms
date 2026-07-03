import { defineStore } from 'pinia'
import { useRuntimeConfig } from '#imports'
import { useClients } from '~/lib/connect'
import { useAppStore } from '~/stores/app'

// AI-стор: read-only RPC к своей нейронке (semantic search / suggestions / themes /
// anomalies / mood / summary). Не держит стейта и не мутирует бакеты — берёт
// activeWorkspaceId из entities-стора (useAppStore). Вынесен из монолита первым как
// самый изолированный кластер.
const dec = (bytes: Uint8Array): any => JSON.parse(new TextDecoder().decode(bytes) || '[]')

export const useAiStore = defineStore('ai', {
  actions: {
    // Палитра-поиск: точные (keyword) + семантические (без дублей, семантика после).
    async serverSearch(q: string): Promise<any[]> {
      const app = useAppStore()
      if (!q.trim() || !app.activeWorkspaceId) return []
      const { query } = useClients()
      const [kwRes, sem] = await Promise.all([
        query.search({ workspaceId: app.activeWorkspaceId, query: q }),
        this.semanticSearch(q),
      ])
      const kw = (dec(kwRes.results) as any[]).map((r) => {
        if (r.type === 'issue') return { type: 'issue', id: r.id, label: `${r.subtitle} ${r.title}`, teamId: r.teamId }
        if (r.type === 'doc') return { type: 'doc', id: r.id, label: `Doc · ${r.title}`, teamId: r.teamId }
        if (r.type === 'channel') return { type: 'channel', id: r.id, label: `Channel · ${r.title}` }
        if (r.type === 'message') return { type: 'message', id: r.id, label: `${r.subtitle} · ${r.title}` }
        return { type: r.type, id: r.id, label: r.title, teamId: r.teamId }
      })
      const seen = new Set(kw.map((r) => r.id))
      return [...kw, ...sem.filter((r) => !seen.has(r.id))]
    },

    // Семантический поиск (AI Фаза 1). Пусто при ошибке/недоступности эмбеддингов.
    async semanticSearch(q: string): Promise<any[]> {
      const app = useAppStore()
      if (!q.trim() || !app.activeWorkspaceId) return []
      const { query } = useClients()
      try {
        const res = await query.semanticSearch({ workspaceId: app.activeWorkspaceId, query: q, limit: 6 })
        const rs = dec(res.results) as any[]
        return rs.map((r) => ({ type: r.type, id: r.id, label: r.title, teamId: r.teamId, semantic: true }))
      } catch {
        return []
      }
    },

    // Похожие issue по вектору (панель «similar / возможный дубль»).
    async similarIssues(issueId: string): Promise<any[]> {
      const app = useAppStore()
      if (!app.activeWorkspaceId) return []
      const { query } = useClients()
      try {
        const res = await query.similarIssues({ workspaceId: app.activeWorkspaceId, issueId, limit: 6 })
        return dec(res.results) as any[]
      } catch {
        return []
      }
    },

    // Петля фидбэка: лог реакции на подсказку (applied|dismissed) — сигнал для ретрейна.
    aiFeedback(kind: string, action: string, entityId: string, suggested: string) {
      const app = useAppStore()
      if (!app.activeWorkspaceId) return
      const base = useRuntimeConfig().public.apiBase as string
      fetch(`${base}/me/workspaces/${app.activeWorkspaceId}/ai/feedback`, {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ kind, action, entityId, suggested }),
      }).catch(() => {})
    },

    // kNN-подсказки для issue (Фаза 2): {assigneeId, labelIds, similar}.
    async suggestForIssue(issue: any): Promise<any> {
      const app = useAppStore()
      if (!app.activeWorkspaceId || !issue) return null
      const { query } = useClients()
      try {
        const text = `${issue.title || ''} ${issue.description?.md || ''}`.trim()
        const res = await query.suggest({ workspaceId: app.activeWorkspaceId, text, teamId: issue.teamId || '', limit: 6 })
        return dec(res.result) as any
      } catch {
        return null
      }
    },

    async themes(): Promise<any[]> {
      const app = useAppStore()
      if (!app.activeWorkspaceId) return []
      const { query } = useClients()
      try {
        const res = await query.themes({ workspaceId: app.activeWorkspaceId })
        return (dec(res.result) as any[]) || []
      } catch {
        return []
      }
    },

    async anomalies(): Promise<any[]> {
      const app = useAppStore()
      if (!app.activeWorkspaceId) return []
      const { query } = useClients()
      try {
        const res = await query.anomalies({ workspaceId: app.activeWorkspaceId, limit: 8 })
        return (dec(res.result) as any[]) || []
      } catch {
        return []
      }
    },

    async mood(): Promise<any> {
      const app = useAppStore()
      if (!app.activeWorkspaceId) return null
      const { query } = useClients()
      try {
        const res = await query.mood({ workspaceId: app.activeWorkspaceId })
        return dec(res.result) as any
      } catch {
        return null
      }
    },

    async summarizeIssue(issueId: string): Promise<string> {
      const app = useAppStore()
      if (!app.activeWorkspaceId) return ''
      const { query } = useClients()
      try {
        const res = await query.summarize({ workspaceId: app.activeWorkspaceId, issueId, maxSentences: 3 })
        return ((dec(res.result) as any)?.summary as string) || ''
      } catch {
        return ''
      }
    },
  },
})
