<script setup lang="ts">
import { useAppStore } from '~/stores/app'
import { useChatStore } from '~/stores/chat'

// Thread panel (плавающая): открывается через chat.threadParentId (v-if на странице)
const store = useAppStore()
const chat = useChatStore()
const { wrap } = useActionWrap()
const threadPanel = useResizable('thread', 384)
const threadParent = computed(() => chat.channelMessages(chat.activeChannelId).find((m: any) => m.id === chat.threadParentId))
const threadDraft = ref('')
const threadPending = ref<File[]>([])
const threadFileEl = ref<HTMLInputElement | null>(null)
const onThreadFilePick = (e: Event) => {
  const files = (e.target as HTMLInputElement).files; if (!files) return
  for (const f of Array.from(files)) threadPending.value = [...threadPending.value, f]
  ;(e.target as HTMLInputElement).value = ''
}
const onThreadPaste = (e: ClipboardEvent) => {
  const items = e.clipboardData?.items; if (!items) return
  for (const it of Array.from(items)) if (it.type.startsWith('image/')) { e.preventDefault(); const f = it.getAsFile(); if (f) threadPending.value = [...threadPending.value, f] }
}
const sendThread = () => wrap(async () => {
  const b = threadDraft.value.trim()
  if ((!b && !threadPending.value.length) || !chat.threadParentId) return
  const files = threadPending.value.slice()
  threadDraft.value = ''; threadPending.value = []
  await chat.sendMessageAttached(chat.activeChannelId, b, files, chat.threadParentId)
})
</script>

<template>
  <ResizeHandle @down="threadPanel.onResizeStart" />
  <aside :style="{ width: threadPanel.w.value + 'px' }" class="flex shrink-0 flex-col overflow-hidden rounded-xl border border-subtle bg-surface text-sm">
    <div class="flex shrink-0 items-center gap-2 border-b border-subtle px-4 py-3">
      <UIcon name="i-heroicons-chat-bubble-left-right" class="text-tertiary" />
      <span class="font-medium">thread</span>
      <UButton class="ml-auto" icon="i-heroicons-x-mark" color="gray" variant="ghost" size="xs" square @click="chat.closeThread()" />
    </div>
    <div class="flex-1 overflow-y-auto px-2 py-3">
      <!-- parent (full message: reactions/attachments, без reply-в-тред) -->
      <div v-if="threadParent" class="mb-1 border-b border-subtle pb-1">
        <ChatMessage :m="threadParent" :can-reply="false" />
      </div>
      <div class="mb-1 px-2 text-xs text-tertiary">{{ chat.threadMessages.length }} {{ chat.threadMessages.length === 1 ? 'reply' : 'replies' }}</div>
      <ChatMessage v-for="m in chat.threadMessages" :key="m.id" :m="m" :can-reply="false" />
    </div>
    <div v-if="store.canWrite" class="shrink-0 border-t border-subtle p-2">
      <PendingAttachments v-if="threadPending.length" class="mb-2" :files="threadPending" @remove="(i) => threadPending = threadPending.filter((_, j) => j !== i)" />
      <div class="flex gap-2 rounded-md border border-subtle bg-elevated px-2 py-1.5">
        <span class="select-none pt-px text-ok">&gt;</span>
        <textarea v-model="threadDraft" rows="2" placeholder="reply…" class="w-full resize-none bg-transparent text-sm outline-none" @paste="onThreadPaste" @keydown.enter.exact.prevent="sendThread" />
      </div>
      <div class="mt-1 flex items-center justify-end gap-2">
        <UButton icon="i-heroicons-paper-clip" color="gray" variant="ghost" size="2xs" square title="Attach" @click="threadFileEl?.click()" />
        <input ref="threadFileEl" type="file" multiple class="hidden" @change="onThreadFilePick" />
        <UButton size="2xs" :disabled="!threadDraft.trim() && !threadPending.length" @click="sendThread">[ reply ]</UButton>
      </div>
    </div>
  </aside>
</template>
