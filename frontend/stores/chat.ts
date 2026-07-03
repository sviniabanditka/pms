import { defineStore } from 'pinia'
import { useClients } from '~/lib/connect'
import { useAppStore } from '~/stores/app'

// Chat-стор: сообщения / треды / typing / каналы+DM. Сущностные бакеты
// (channels, channelMembers, channelUnread) и связанные геттеры живут в
// entities-сторе (useAppStore); здесь только чат-стейт (messages/threads/typing)
// и RPC-экшены. Читает activeWorkspaceId/currentUserId/catchUp/bootstrap и мутирует
// бакеты channels/channelUnread через useAppStore. Вынесен из монолита.
const dec = (bytes: Uint8Array): any => JSON.parse(new TextDecoder().decode(bytes) || '[]')

export const useChatStore = defineStore('chat', {
  state: () => ({
    messages: {} as Record<string, any[]>, // channelId -> messages (DTO)
    hasMoreMessages: {} as Record<string, boolean>,
    activeChannelId: '' as string,
    threadParentId: '' as string,
    threadMessages: [] as any[],
    typing: {} as Record<string, Record<string, number>>, // channelId -> userId -> ts
  }),

  getters: {
    channelMessages: (s) => (channelId: string) => s.messages[channelId] || [],
    typingNames() {
      return (channelId: string) => {
        const app = useAppStore()
        const t = this.typing[channelId] || {}
        return Object.keys(t)
          .filter((uid) => t[uid] && uid !== app.currentUserId)
          .map((uid) => app.users[uid]?.name || app.users[uid]?.email || 'Someone')
      }
    },
  },

  actions: {
    async loadMessages(channelId: string) {
      const app = useAppStore()
      const { chats } = useClients()
      this.activeChannelId = channelId
      const res = await chats.getMessages({ workspaceId: app.activeWorkspaceId, channelId, limit: 50 })
      this.messages[channelId] = dec(res.messages)
      this.hasMoreMessages[channelId] = res.hasMore
    },
    // подгрузка истории вверх (бесконечный скролл)
    async loadOlder(channelId: string) {
      const app = useAppStore()
      const cur = this.messages[channelId] || []
      if (!cur.length || !this.hasMoreMessages[channelId]) return
      const { chats } = useClients()
      const res = await chats.getMessages({ workspaceId: app.activeWorkspaceId, channelId, before: cur[0].id, limit: 50 })
      const older = dec(res.messages)
      this.messages[channelId] = [...older, ...cur]
      this.hasMoreMessages[channelId] = res.hasMore
    },
    // фоновый рефетч активного канала (по WS-дельте) без сброса скролла
    async refetchActive() {
      const app = useAppStore()
      if (!this.activeChannelId) return
      const { chats } = useClients()
      const n = (this.messages[this.activeChannelId] || []).length || 50
      const res = await chats.getMessages({ workspaceId: app.activeWorkspaceId, channelId: this.activeChannelId, limit: Math.min(100, Math.max(50, n)) })
      this.messages[this.activeChannelId] = dec(res.messages)
      this.hasMoreMessages[this.activeChannelId] = res.hasMore
      if (this.threadParentId) this.loadThread(this.threadParentId)
    },
    async sendMessage(channelId: string, body: string, parentId = '') {
      const app = useAppStore()
      const { chats } = useClients()
      const res = await chats.sendMessage({ workspaceId: app.activeWorkspaceId, channelId, body, parentId })
      if (parentId) { await this.loadThread(parentId); await this.refetchActive() }
      else await this.refetchActive()
      return res.entityId
    },
    async sendMessageAttached(channelId: string, body: string, files: File[], parentId = '') {
      const app = useAppStore()
      const { chats, attachments } = useClients()
      const ups = [] as { key: string; filename: string; contentType: string; size: number }[]
      for (const f of files) ups.push(await app.uploadAndGetKey(f))
      const res = await chats.sendMessage({ workspaceId: app.activeWorkspaceId, channelId, body, parentId })
      for (const up of ups) {
        await attachments.createAttachment({
          workspaceId: app.activeWorkspaceId, entityType: 'message', entityId: res.entityId,
          filename: up.filename, contentType: up.contentType, size: BigInt(up.size), s3Key: up.key,
        })
      }
      if (ups.length) await app.catchUp(app.activeWorkspaceId, ['attachment'])
      if (parentId) await this.loadThread(parentId)
      await this.refetchActive()
    },
    async editMessage(id: string, body: string) {
      const app = useAppStore()
      const { chats } = useClients()
      await chats.editMessage({ workspaceId: app.activeWorkspaceId, id, body })
      await this.refetchActive()
      if (this.threadParentId) await this.loadThread(this.threadParentId)
    },
    async deleteMessage(id: string) {
      const app = useAppStore()
      const { chats } = useClients()
      await chats.deleteMessage({ workspaceId: app.activeWorkspaceId, id })
      await this.refetchActive()
      if (this.threadParentId) await this.loadThread(this.threadParentId)
    },
    async toggleReaction(messageId: string, emoji: string) {
      const app = useAppStore()
      const { chats } = useClients()
      await chats.toggleReaction({ workspaceId: app.activeWorkspaceId, messageId, emoji })
      await this.refetchActive()
      if (this.threadParentId) await this.loadThread(this.threadParentId)
    },
    async sendTyping(channelId: string) {
      const app = useAppStore()
      const { chats } = useClients()
      try { await chats.sendTyping({ workspaceId: app.activeWorkspaceId, channelId }) } catch {}
    },
    async loadThread(parentId: string) {
      const app = useAppStore()
      this.threadParentId = parentId
      const { chats } = useClients()
      const res = await chats.getMessages({ workspaceId: app.activeWorkspaceId, channelId: this.activeChannelId, parentId, limit: 100 })
      this.threadMessages = dec(res.messages)
    },
    closeThread() { this.threadParentId = ''; this.threadMessages = [] },
    markTyping(channelId: string, userId: string) {
      ;(this.typing[channelId] ||= {})[userId] = Date.now()
      setTimeout(() => { if (this.typing[channelId]) { delete this.typing[channelId][userId] } }, 4000)
    },
    async createChannel(name: string, isPrivate = false): Promise<string> {
      const app = useAppStore()
      const { chats } = useClients()
      const res = await chats.createChannel({ workspaceId: app.activeWorkspaceId, name, private: isPrivate })
      app.channels[res.entityId] = { id: res.entityId, kind: 'channel', name, private: isPrivate }
      await app.catchUp(app.activeWorkspaceId, ['channel', 'channel_member']) // подтянуть channel_member (себя)
      return res.entityId
    },
    async addChannelMember(channelId: string, userId: string) {
      const app = useAppStore()
      const { chats } = useClients()
      await chats.addChannelMember({ workspaceId: app.activeWorkspaceId, channelId, userId })
      await app.catchUp(app.activeWorkspaceId, ['channel', 'channel_member'])
    },
    async removeChannelMember(channelId: string, userId: string) {
      const app = useAppStore()
      const { chats } = useClients()
      await chats.removeChannelMember({ workspaceId: app.activeWorkspaceId, channelId, userId })
      await app.catchUp(app.activeWorkspaceId, ['channel', 'channel_member'])
    },
    async leaveChannel(channelId: string) {
      const app = useAppStore()
      const { chats } = useClients()
      await chats.removeChannelMember({ workspaceId: app.activeWorkspaceId, channelId, userId: app.currentUserId })
      if (this.activeChannelId === channelId) this.activeChannelId = ''
      await app.catchUp(app.activeWorkspaceId, ['channel', 'channel_member'])
    },
    async archiveChannel(channelId: string) {
      const app = useAppStore()
      const { chats } = useClients()
      await chats.archiveChannel({ workspaceId: app.activeWorkspaceId, id: channelId })
      if (this.activeChannelId === channelId) this.activeChannelId = ''
      if (app.channels[channelId]) app.channels[channelId].archivedAt = new Date().toISOString()
    },
    async createDM(userId: string): Promise<string> {
      const app = useAppStore()
      const { chats } = useClients()
      const res = await chats.createDM({ workspaceId: app.activeWorkspaceId, otherUserId: userId })
      if (!app.channels[res.entityId]) app.channels[res.entityId] = { id: res.entityId, kind: 'dm' }
      await this.loadMessages(res.entityId)
      return res.entityId
    },
    // --- Chat unread ---
    async markChannelRead(channelId: string) {
      const app = useAppStore()
      if (app.channelUnread[channelId]) app.channelUnread[channelId] = { ...app.channelUnread[channelId], count: 0, lastReadAt: new Date().toISOString() }
      else app.channelUnread[channelId] = { channelId, count: 0, lastReadAt: new Date().toISOString() }
      const { chats } = useClients()
      try { await chats.markChannelRead({ workspaceId: app.activeWorkspaceId, channelId }) } catch {}
    },
    bumpUnread(channelId: string) {
      const app = useAppStore()
      const cur = app.channelUnread[channelId]
      app.channelUnread[channelId] = { channelId, lastReadAt: cur?.lastReadAt || null, count: Number(cur?.count || 0) + 1 }
    },
  },
})
