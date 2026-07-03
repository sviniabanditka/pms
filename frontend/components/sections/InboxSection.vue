<script setup lang="ts">
import { useAppStore } from '~/stores/app'

// Inbox: список уведомлений (фильтр/prefs приходят со страницы, навигация — эмитом)
const props = defineProps<{
  filter: 'all' | 'unread' | 'assigned' | 'state_changed'
  notifPrefs: Record<string, boolean>
}>()
defineEmits<{ (e: 'open', n: any): void }>()

const store = useAppStore()
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
  <NotifRow v-for="n in inboxList" :key="n.id" :notification="n" @open="$emit('open', $event)" />
  <div v-if="!inboxList.length" class="flex flex-col items-center justify-center gap-2 py-20 text-tertiary">
    <UIcon name="i-heroicons-inbox" class="text-3xl" />
    <span class="text-sm">{{ filter === 'all' ? "// inbox zero — you're all caught up" : '// nothing here' }}</span>
  </div>
</template>
