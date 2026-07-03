<script setup lang="ts">
import { useAppStore } from '~/stores/app'
import { notifTime } from '~/lib/ui'

// Строка уведомления инбокса. Молекула из InboxSection: notification пропсом,
// issue-резолв/mark-read из стора, клик → emit open.
const props = defineProps<{ notification: any }>()
const emit = defineEmits<{ (e: 'open', n: any): void }>()
const store = useAppStore()

const NOTIF_META: Record<string, { icon: string; verb: string; color: string }> = {
  assigned: { icon: 'i-heroicons-user-plus', verb: 'Assigned to you', color: '#5e6ad2' },
  state_changed: { icon: 'i-heroicons-arrow-path', verb: 'Status changed', color: '#d29922' },
  commented: { icon: 'i-heroicons-chat-bubble-left-ellipsis', verb: 'New comment', color: '#3fb950' },
  mention: { icon: 'i-heroicons-at-symbol', verb: 'Mentioned you', color: '#5e6ad2' },
}
function meta(t: string) { return NOTIF_META[t] || { icon: 'i-heroicons-bell', verb: t, color: '#7c828c' } }
</script>

<template>
  <div class="group flex cursor-pointer items-start gap-3 border-b border-subtle/40 px-4 py-3 hover:bg-hover"
    @click="emit('open', notification)">
    <span class="mt-1.5 h-2 w-2 shrink-0 rounded-full" :class="notification.readAt ? 'bg-transparent' : 'bg-accent'" />
    <span class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-elevated">
      <UIcon :name="meta(notification.type).icon" :style="{ color: meta(notification.type).color }" />
    </span>
    <div class="min-w-0 flex-1">
      <div class="flex items-center gap-2">
        <span :class="notification.readAt ? 'text-secondary' : 'font-medium'">{{ meta(notification.type).verb }}</span>
        <span class="text-xs text-tertiary">{{ store.issueById(notification.entityId)?.identifier }}</span>
        <span class="ml-auto shrink-0 text-xs text-tertiary">{{ notifTime(notification.at) }}</span>
      </div>
      <div class="truncate text-sm text-tertiary">{{ store.issueById(notification.entityId)?.title || notification.entityType }}</div>
    </div>
    <UButton v-if="!notification.readAt" class="mt-0.5 opacity-0 group-hover:opacity-100" icon="i-heroicons-check" color="gray" variant="ghost" size="2xs" square title="Mark read" @click.stop="store.markNotificationRead(notification.id)" aria-label="Mark read" />
  </div>
</template>
