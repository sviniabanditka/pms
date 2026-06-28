<script setup lang="ts">
import { ref, computed } from 'vue'
import { useAppStore } from '~/stores/app'

// Одно сообщение чата: тело + вложения + реакции + edit/delete + reply-в-тред.
// Используется и в основном списке, и в панели треда → одинаковый функционал.
const props = withDefaults(defineProps<{ m: any; canReply?: boolean }>(), { canReply: true })
const store = useAppStore()

const EMOJIS = ['👍', '❤️', '😄', '🎉', '👀', '🚀', '✅', '🔥']
const editing = ref(false)
const editDraft = ref('')
const pickerOpen = ref(false)

function reactionGroups(m: any) {
  const g: Record<string, { emoji: string; count: number; mine: boolean }> = {}
  for (const r of m.reactions || []) {
    ;(g[r.emoji] ||= { emoji: r.emoji, count: 0, mine: false }).count++
    if (r.userId === store.currentUserId) g[r.emoji].mine = true
  }
  return Object.values(g)
}
const react = (emoji: string) => { pickerOpen.value = false; store.toggleReaction(props.m.id, emoji) }
function startEdit() { editing.value = true; editDraft.value = props.m.body }
async function saveEdit() { if (editDraft.value.trim()) await store.editMessage(props.m.id, editDraft.value); editing.value = false }
const mine = computed(() => props.m.authorId === store.currentUserId)

function notifTime(at: any) {
  const d = new Date(String(at))
  if (isNaN(+d)) return String(at).slice(5, 10)
  return d.toLocaleDateString('en', { month: 'short', day: 'numeric' })
}
</script>

<template>
  <div class="group relative flex gap-3 rounded-md px-2 py-1.5 hover:bg-hover">
    <Avatar class="mt-0.5" :name="store.userName(m.authorId)" :avatar-key="store.users[m.authorId]?.avatarKey" :size="32" />
    <div class="min-w-0 flex-1">
      <div class="flex items-baseline gap-2">
        <span class="text-sm font-medium">{{ store.userName(m.authorId) }}</span>
        <span class="text-xs text-tertiary">{{ notifTime(m.createdAt) }}</span>
        <span v-if="m.editedAt" class="text-xs text-tertiary/60">(edited)</span>
      </div>

      <div v-if="editing" class="mt-1">
        <textarea v-model="editDraft" rows="2" class="w-full rounded-md border border-subtle bg-elevated px-2 py-1 text-sm outline-none" @keydown.enter.exact.prevent="saveEdit" @keydown.esc="editing = false" />
        <div class="mt-1 flex gap-2"><UButton size="2xs" @click="saveEdit">Save</UButton><UButton size="2xs" color="gray" variant="ghost" @click="editing = false">Cancel</UButton></div>
      </div>
      <template v-else>
        <div v-if="m.deletedAt" class="text-sm italic text-tertiary">(message deleted)</div>
        <div v-else class="prose-chat text-sm leading-relaxed" v-html="renderMarkdown(m.body)" />
        <Attachments v-if="store.attachmentsFor('message', m.id).length" class="mt-1.5" entity-type="message" :entity-id="m.id" readonly />
        <div v-if="reactionGroups(m).length" class="mt-1 flex flex-wrap gap-1">
          <button v-for="g in reactionGroups(m)" :key="g.emoji"
            class="flex items-center gap-1 rounded-full border px-1.5 py-0.5 text-xs"
            :class="g.mine ? 'border-accent bg-accent/15 text-white' : 'border-subtle hover:bg-hover'"
            @click="react(g.emoji)">{{ g.emoji }} {{ g.count }}</button>
        </div>
        <button v-if="canReply && m.replyCount && !m.parentId" class="mt-1 flex items-center gap-1 text-xs text-accent hover:underline" @click="store.loadThread(m.id)">
          <UIcon name="i-heroicons-chat-bubble-left-right" /> {{ m.replyCount }} {{ m.replyCount === 1 ? 'reply' : 'replies' }}
        </button>
      </template>
    </div>

    <div v-if="!m.deletedAt && !editing" class="absolute -top-3 right-2 hidden items-center gap-0.5 rounded-md border border-subtle bg-elevated px-1 py-0.5 shadow group-hover:flex">
      <div class="relative">
        <UButton icon="i-heroicons-face-smile" color="gray" variant="ghost" size="2xs" square title="React" @click="pickerOpen = !pickerOpen" />
        <div v-if="pickerOpen" class="absolute bottom-7 right-0 z-20 flex gap-1 rounded-md border border-subtle bg-elevated p-1 shadow-lg">
          <button v-for="e in EMOJIS" :key="e" class="rounded p-1 text-base hover:bg-hover" @click="react(e)">{{ e }}</button>
        </div>
      </div>
      <UButton v-if="canReply && !m.parentId" icon="i-heroicons-chat-bubble-oval-left" color="gray" variant="ghost" size="2xs" square title="Reply in thread" @click="store.loadThread(m.id)" />
      <UButton v-if="mine" icon="i-heroicons-pencil" color="gray" variant="ghost" size="2xs" square title="Edit" @click="startEdit" />
      <UButton v-if="mine" icon="i-heroicons-trash" color="gray" variant="ghost" size="2xs" square title="Delete" @click="store.deleteMessage(m.id)" />
    </div>
  </div>
</template>
