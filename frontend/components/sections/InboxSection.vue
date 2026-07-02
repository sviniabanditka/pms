<script setup lang="ts">
import { useAppStore } from '~/stores/app'
import { notifTime } from '~/lib/ui'

// Inbox: список уведомлений (фильтр/prefs приходят со страницы, навигация — эмитом)
const props = defineProps<{
  filter: 'all' | 'unread' | 'assigned' | 'state_changed'
  notifPrefs: Record<string, boolean>
}>()
defineEmits<{ (e: 'open', n: any): void }>()

const store = useAppStore()
const NOTIF_META: Record<string, { icon: string; verb: string; color: string }> = {
  assigned: { icon: 'i-heroicons-user-plus', verb: 'Assigned to you', color: '#5e6ad2' },
  state_changed: { icon: 'i-heroicons-arrow-path', verb: 'Status changed', color: '#d29922' },
  commented: { icon: 'i-heroicons-chat-bubble-left-ellipsis', verb: 'New comment', color: '#3fb950' },
  mention: { icon: 'i-heroicons-at-symbol', verb: 'Mentioned you', color: '#5e6ad2' },
}
function notifMeta(t: string) { return NOTIF_META[t] || { icon: 'i-heroicons-bell', verb: t, color: '#7c828c' } }
const inboxList = computed(() => {
  // скрываем типы, выключенные в настройках уведомлений
  const all = store.myNotifications.filter((n: any) => props.notifPrefs[n.type] !== false)
  if (props.filter === 'all') return all
  if (props.filter === 'unread') return all.filter((n: any) => !n.readAt)
  return all.filter((n: any) => n.type === props.filter)
})
</script>

<template>
  <div class="sticky top-0 z-10 flex items-center gap-2 border-b border-subtle bg-surface px-4 py-3 text-sm">
    <UIcon name="i-heroicons-inbox" class="text-tertiary" />
    <span class="font-medium">inbox</span>
    <span v-if="store.unreadCount" class="rounded bg-accent/20 px-1.5 text-xs text-accent">{{ store.unreadCount }}</span>
    <UButton v-if="store.unreadCount" class="ml-auto" size="xs" color="gray" variant="ghost" @click="store.markAllNotificationsRead()">[ mark all read ]</UButton>
  </div>
  <div v-for="n in inboxList" :key="n.id"
    class="group flex cursor-pointer items-start gap-3 border-b border-subtle/40 px-4 py-3 hover:bg-hover"
    @click="$emit('open', n)">
    <span class="mt-1.5 h-2 w-2 shrink-0 rounded-full" :class="n.readAt ? 'bg-transparent' : 'bg-accent'" />
    <span class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-elevated">
      <UIcon :name="notifMeta(n.type).icon" :style="{ color: notifMeta(n.type).color }" />
    </span>
    <div class="min-w-0 flex-1">
      <div class="flex items-center gap-2">
        <span :class="n.readAt ? 'text-secondary' : 'font-medium'">{{ notifMeta(n.type).verb }}</span>
        <span class="text-xs text-tertiary">{{ store.issueById(n.entityId)?.identifier }}</span>
        <span class="ml-auto shrink-0 text-xs text-tertiary">{{ notifTime(n.at) }}</span>
      </div>
      <div class="truncate text-sm text-tertiary">{{ store.issueById(n.entityId)?.title || n.entityType }}</div>
    </div>
    <UButton v-if="!n.readAt" class="mt-0.5 opacity-0 group-hover:opacity-100" icon="i-heroicons-check" color="gray" variant="ghost" size="2xs" square title="Mark read" @click.stop="store.markNotificationRead(n.id)" />
  </div>
  <div v-if="!inboxList.length" class="flex flex-col items-center justify-center gap-2 py-20 text-tertiary">
    <UIcon name="i-heroicons-inbox" class="text-3xl" />
    <span class="text-sm">{{ filter === 'all' ? "// inbox zero — you're all caught up" : '// nothing here' }}</span>
  </div>
</template>
