<script setup lang="ts">
import { useAppStore } from '~/stores/app'

// Чат: шапка канала + сообщения + композер. Открытие каналов (openChannel/openDM)
// живёт на странице (зовётся из сайдбара/палитры/роутера); скролл — через expose.
defineProps<{
  linkCopied: boolean
  firstUnreadMsgId: string
}>()
defineEmits<{ (e: 'copy-link'): void }>()

const store = useAppStore()
const { wrap } = useActionWrap()
const meeting = useMeeting()

const msgDraft = ref('')
const composerEl = ref<HTMLTextAreaElement | null>(null)
const showPreview = ref(false)
const currentChannel = computed(() => store.channels[store.activeChannelId])
const membersOpen = ref(false)
// управлять списком можно только в приватном канале (в публичном участники = весь воркспейс)
const canManageMembers = computed(() => !!currentChannel.value && currentChannel.value.kind === 'channel' && currentChannel.value.private && store.isChannelMember(currentChannel.value.id))
const channelActions = computed(() => {
  const c = currentChannel.value
  if (!c || c.kind === 'dm') return []
  const items: any[] = []
  if (c.private && store.isChannelMember(c.id)) items.push({ label: 'Leave channel', icon: 'i-heroicons-arrow-right-on-rectangle', click: () => wrap(() => store.leaveChannel(c.id)) })
  if (store.isChannelMember(c.id) || c.createdBy === store.currentUserId) items.push({ label: 'Archive channel', icon: 'i-heroicons-archive-box', click: () => wrap(() => store.archiveChannel(c.id)) })
  return items.length ? [items] : []
})
const channelAddable = computed(() => {
  if (!currentChannel.value) return [] as any[]
  const inCh = new Set(store.channelMembersOf(currentChannel.value.id))
  return Object.values(store.users).filter((u: any) => !inCh.has(u.id)).map((u: any) => ({ label: store.userName(u.id), value: u.id }))
})
const addMember = (uid: string) => { if (uid && currentChannel.value) wrap(() => store.addChannelMember(currentChannel.value!.id, uid)) }
watch(() => store.activeChannelId, () => { membersOpen.value = false })
function channelTitle(c: any) {
  if (!c) return ''
  return c.kind === 'dm' ? store.userName(store.dmOtherId(c.id)) : c.name
}
// inline @/# пикеры (юзеры + сущности)
const chatMention = useMention(msgDraft, composerEl)
function onComposerKeydown(e: KeyboardEvent) {
  if (chatMention.onKeydown(e)) return
  if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); sendMsg() }
}
const chatFileEl = ref<HTMLInputElement | null>(null)
const onChatFilePick = (e: Event) => {
  const files = (e.target as HTMLInputElement).files
  if (!files?.length) return
  for (const f of Array.from(files)) pendingFiles.value = [...pendingFiles.value, f] // всё (картинки+файлы) → превью в композере, шлём прикреплённым
  if (chatFileEl.value) chatFileEl.value.value = ''
}
const msgScrollEl = ref<HTMLElement | null>(null)
function scrollMessagesBottom() { nextTick(() => { const el = msgScrollEl.value; if (el) el.scrollTop = el.scrollHeight }) }
// скролл к разделителю «new messages» (позиция вычислена страницей при открытии канала)
function scrollToUnread() {
  const el = msgScrollEl.value?.querySelector('[data-unread-divider]') as HTMLElement | null
  if (el) el.scrollIntoView({ block: 'center' }); else scrollMessagesBottom()
}
defineExpose({ scrollBottom: scrollMessagesBottom, scrollToUnread })
const pendingFiles = ref<File[]>([])
const sendMsg = () => wrap(async () => {
  const b = msgDraft.value.trim()
  if ((!b && !pendingFiles.value.length) || !store.activeChannelId) return
  const files = pendingFiles.value.slice()
  msgDraft.value = ''; pendingFiles.value = []; showPreview.value = false
  await store.sendMessageAttached(store.activeChannelId, b, files)
  scrollMessagesBottom()
})
const onChatPaste = (e: ClipboardEvent) => {
  const items = e.clipboardData?.items; if (!items) return
  for (const it of Array.from(items)) {
    if (it.type.startsWith('image/')) { e.preventDefault(); const f = it.getAsFile(); if (f) pendingFiles.value = [...pendingFiles.value, f] }
  }
}
function wrapSel(pre: string, post = pre) {
  const el = composerEl.value
  if (!el) { msgDraft.value += pre + post; return }
  const s = el.selectionStart, e = el.selectionEnd, v = msgDraft.value
  msgDraft.value = v.slice(0, s) + pre + v.slice(s, e) + post + v.slice(e)
  nextTick(() => { el.focus(); el.selectionStart = s + pre.length; el.selectionEnd = e + pre.length })
}
// mentions
function insertToken(text: string) {
  const el = composerEl.value
  if (!el) { msgDraft.value += text; return }
  const s = el.selectionStart, v = msgDraft.value
  msgDraft.value = v.slice(0, s) + text + v.slice(s)
  nextTick(() => { el.focus(); el.selectionStart = el.selectionEnd = s + text.length })
}
const mentionItems = computed(() => [[...store.wsUsers, store.users[store.currentUserId]].filter(Boolean).map((u: any) => ({
  label: u.name || u.email, avatar: undefined, click: () => insertToken(`@[${u.name || u.email}](${u.id}) `),
}))])
// typing throttle
let typingTs = 0
function onComposerInput() {
  const now = Date.now()
  if (store.activeChannelId && now - typingTs > 2500) { typingTs = now; store.sendTyping(store.activeChannelId) }
}
function onMessagesScroll(e: Event) {
  const el = e.target as HTMLElement
  if (el.scrollTop < 40 && currentChannel.value) {
    const prevH = el.scrollHeight
    store.loadOlder(currentChannel.value.id).then(() => nextTick(() => { el.scrollTop = el.scrollHeight - prevH }))
  }
}
const startOrJoinHuddle = (channelId: string) => wrap(async () => {
  const existing = store.liveHuddleForChannel(channelId)
  if (existing && meeting.activeMeetingId.value === existing.id) return // уже в нём
  const mid = existing ? existing.id : await store.startHuddle(channelId)
  await meeting.join(mid)
})
</script>

<template>
  <div v-if="currentChannel" class="flex h-full flex-col">
    <div class="flex shrink-0 items-center gap-2 border-b border-subtle px-4 py-3">
      <UIcon :name="currentChannel.kind === 'dm' ? 'i-heroicons-user' : currentChannel.private ? 'i-heroicons-lock-closed' : 'i-heroicons-hashtag'" class="text-tertiary" />
      <span class="font-medium">{{ channelTitle(currentChannel) }}</span>

      <!-- channel members (count + management dropdown) -->
      <div v-if="currentChannel.kind !== 'dm'" class="relative ml-auto">
        <UButton color="gray" variant="ghost" size="xs" @click="membersOpen = !membersOpen">
          {{ store.channelParticipants(currentChannel.id).length }}
        </UButton>
        <div v-if="membersOpen" class="absolute right-0 top-9 z-30 w-64 rounded-lg border border-subtle bg-elevated p-2 shadow-xl">
          <div class="mb-1 px-1 text-xs font-medium uppercase tracking-wide text-tertiary"><span class="text-ok"># </span>Members{{ !currentChannel.private ? ' · entire workspace' : '' }}</div>
          <div class="max-h-56 overflow-y-auto">
            <div v-for="uid in store.channelParticipants(currentChannel.id)" :key="uid" class="group flex items-center gap-2 rounded px-1.5 py-1 hover:bg-hover">
              <Avatar :name="store.userName(uid)" :avatar-key="store.users[uid]?.avatarKey" :size="22" />
              <span class="min-w-0 flex-1 truncate text-sm">{{ store.userName(uid) }}{{ uid === store.currentUserId ? ' (you)' : '' }}</span>
              <UButton v-if="canManageMembers && uid !== store.currentUserId" icon="i-heroicons-x-mark" color="gray" variant="ghost" size="2xs" square title="Remove" @click="store.removeChannelMember(currentChannel.id, uid)" />
            </div>
          </div>
          <template v-if="canManageMembers">
            <div class="my-1.5 h-px bg-subtle" />
            <USelectMenu :options="channelAddable" value-attribute="value" option-attribute="label" searchable placeholder="Add member…" @update:model-value="addMember" />
          </template>
        </div>
      </div>

      <template v-if="store.liveHuddleForChannel(currentChannel.id)">
        <UButton class="ml-auto" :color="meeting.activeMeetingId.value === store.liveHuddleForChannel(currentChannel.id)?.id ? 'gray' : 'green'" variant="ghost" size="xs"
          @click="startOrJoinHuddle(currentChannel.id)">
          {{ meeting.activeMeetingId.value === store.liveHuddleForChannel(currentChannel.id)?.id ? 'In huddle' : 'Join huddle' }}
          · {{ store.meetingJoined(store.liveHuddleForChannel(currentChannel.id)!.id).length }}
        </UButton>
      </template>
      <UButton v-else class="ml-auto" color="gray" variant="ghost" size="xs" title="Start huddle" @click="startOrJoinHuddle(currentChannel.id)">[ huddle ]</UButton>
      <UButton :icon="linkCopied ? 'i-heroicons-check' : 'i-heroicons-link'" color="gray" variant="ghost" size="xs" square :title="linkCopied ? 'Copied!' : 'Copy link'" @click="$emit('copy-link')" />
      <UDropdown v-if="channelActions.length" :items="channelActions" :popper="{ placement: 'bottom-end' }">
        <UButton icon="i-heroicons-ellipsis-horizontal" color="gray" variant="ghost" size="xs" square title="Channel actions" />
      </UDropdown>
    </div>
    <!-- messages -->
    <div ref="msgScrollEl" class="flex flex-1 flex-col gap-1 overflow-y-auto px-2 py-3" @scroll="onMessagesScroll">
      <div v-if="store.hasMoreMessages[currentChannel.id]" class="py-1 text-center text-xs text-tertiary">loading older…</div>
      <template v-for="m in store.channelMessages(currentChannel.id)" :key="m.id">
      <div v-if="m.id === firstUnreadMsgId" data-unread-divider class="my-1 flex items-center gap-2 px-2 text-xs font-medium text-danger">
        <span class="h-px flex-1 bg-danger/40" /> New messages <span class="h-px flex-1 bg-danger/40" />
      </div>
      <ChatMessage :m="m" />
      </template>
      <div v-if="store.messages[currentChannel.id] === undefined" class="space-y-3 py-2">
        <div v-for="n in 6" :key="n" class="flex gap-3"><div class="skeleton h-8 w-8 rounded-full" /><div class="flex-1 space-y-1"><div class="skeleton h-3 w-24" /><div class="skeleton h-3" :style="{ width: 30 + (n * 13) % 55 + '%' }" /></div></div>
      </div>
      <div v-else-if="!store.channelMessages(currentChannel.id).length" class="py-16 text-center text-sm text-tertiary">No messages yet — say hi 👋</div>
    </div>
    <!-- typing -->
    <div class="h-5 px-4 text-xs text-tertiary">
      <span v-if="store.typingNames(currentChannel.id).length">{{ store.typingNames(currentChannel.id).join(', ') }} typing…</span>
    </div>
    <!-- composer -->
    <div v-if="!store.canWrite" class="shrink-0 px-3 pb-3 text-center text-xs text-tertiary">// read-only — guests can't post</div>
    <div v-else class="shrink-0 px-3 pb-3">
      <PendingAttachments v-if="pendingFiles.length" class="mb-1.5" :files="pendingFiles" @remove="(i) => pendingFiles = pendingFiles.filter((_, j) => j !== i)" />
      <div class="rounded-lg border border-subtle bg-elevated">
        <div class="flex items-center gap-1 border-b border-subtle px-2 py-1">
          <UButton color="gray" variant="ghost" size="2xs" square title="Bold" class="font-bold" @click="wrapSel('**')">B</UButton>
          <UButton color="gray" variant="ghost" size="2xs" square title="Italic" class="italic" @click="wrapSel('_')">I</UButton>
          <UButton icon="i-heroicons-code-bracket" color="gray" variant="ghost" size="2xs" square title="Code" @click="wrapSel('`')" />
          <UButton icon="i-heroicons-link" color="gray" variant="ghost" size="2xs" square title="Link" @click="wrapSel('[', '](url)')" />
          <UButton icon="i-heroicons-list-bullet" color="gray" variant="ghost" size="2xs" square title="List" @click="wrapSel('\n- ', '')" />
          <UDropdown :items="mentionItems" :popper="{ placement: 'top-start' }">
            <UButton icon="i-heroicons-at-symbol" color="gray" variant="ghost" size="2xs" square title="Mention" />
          </UDropdown>
          <UButton icon="i-heroicons-paper-clip" color="gray" variant="ghost" size="2xs" square title="Attach file" @click="chatFileEl?.click()" />
          <input ref="chatFileEl" type="file" multiple class="hidden" @change="onChatFilePick" />
          <UButton class="ml-auto" :icon="showPreview ? 'i-heroicons-pencil' : 'i-heroicons-eye'" color="gray" variant="ghost" size="2xs" square @click="showPreview = !showPreview" />
        </div>
        <div v-if="showPreview" class="prose-chat min-h-[60px] px-3 py-2 text-sm" v-html="renderMarkdown(msgDraft) || '<span class=&quot;text-tertiary&quot;>// nothing to preview</span>'" />
        <div v-else class="relative flex gap-2 px-3 py-2">
          <MentionMenu :m="chatMention" />
          <span class="select-none pt-px text-ok">&gt;</span>
          <textarea ref="composerEl" v-model="msgDraft" rows="3" placeholder="message… (markdown, @mention, #ref, ↵ to send)"
            class="w-full resize-none bg-transparent text-sm outline-none placeholder:text-tertiary"
            @paste="onChatPaste" @input="onComposerInput(); chatMention.detect()" @keydown="onComposerKeydown" />
        </div>
        <div class="flex items-center justify-between px-3 py-1.5">
          <span class="text-xs text-tertiary">**bold** _italic_ `code` — Enter send, Shift+Enter newline</span>
          <UButton size="xs" :disabled="!msgDraft.trim() && !pendingFiles.length" @click="sendMsg">[ send ]</UButton>
        </div>
      </div>
    </div>
  </div>
  <div v-else class="flex h-full flex-col items-center justify-center gap-2 text-tertiary">
    <UIcon name="i-heroicons-chat-bubble-left-right" class="text-3xl" />
    <span class="text-sm"><span class="text-ok">$</span> open a channel or DM</span>
  </div>
</template>
