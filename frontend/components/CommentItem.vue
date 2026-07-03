<script setup lang="ts">
import { ref } from 'vue'
import { useAppStore } from '~/stores/app'
import { notifTime } from '~/lib/ui'

// Один комментарий задачи. Молекула: comment+issueId пропсами, derived (author/
// attachments/reactions) из стора. Edit- и react-picker-стейт СВОЙ (раньше жил в
// панели как общий cReactPickerId/editCommentId — вынесли, панель полегчала).
const props = defineProps<{ comment: any; issueId: string }>()
const store = useAppStore()
const { wrap } = useActionWrap()

const EMOJIS = ['👍', '❤️', '😄', '🎉', '👀', '🚀', '✅', '🔥']
const pickerOpen = ref(false)
const editing = ref(false)
const draft = ref('')

function reactionGroups(m: any) {
  const g: Record<string, { emoji: string; count: number; mine: boolean }> = {}
  for (const r of m.reactions || []) {
    ;(g[r.emoji] ||= { emoji: r.emoji, count: 0, mine: false }).count++
    if (r.userId === store.currentUserId) g[r.emoji].mine = true
  }
  return Object.values(g)
}
function react(emoji: string) { pickerOpen.value = false; wrap(() => store.toggleCommentReaction(props.comment.id, props.issueId, emoji)) }
function startEdit() { editing.value = true; draft.value = props.comment.body }
const save = () => wrap(async () => { if (draft.value.trim()) await store.editComment(props.comment.id, props.issueId, draft.value); editing.value = false })
</script>

<template>
  <div class="group mb-2 flex gap-2">
    <Avatar class="mt-0.5" :name="store.userName(comment.authorId)" :avatar-key="store.users[comment.authorId]?.avatarKey" :size="28" />
    <div class="min-w-0 flex-1">
      <div class="flex items-baseline gap-2">
        <span class="text-sm font-medium">{{ store.userName(comment.authorId) }}</span>
        <span class="text-xs text-tertiary">{{ notifTime(comment.createdAt) }}</span>
        <span v-if="comment.editedAt" class="text-xs text-tertiary">(edited)</span>
        <div class="relative ml-auto">
          <UButton v-if="!comment.deletedAt" class="opacity-0 group-hover:opacity-100" icon="i-heroicons-face-smile" color="gray" variant="ghost" size="2xs" square @click="pickerOpen = !pickerOpen" aria-label="Add reaction" />
          <div v-if="pickerOpen" class="absolute right-0 z-20 mt-1 flex gap-0.5 rounded-md border border-subtle bg-elevated p-1 shadow-lg">
            <button v-for="e in EMOJIS" :key="e" class="rounded p-1 text-base hover:bg-hover" @click="react(e)">{{ e }}</button>
          </div>
        </div>
        <template v-if="comment.authorId === store.currentUserId && !comment.deletedAt && !editing">
          <UButton class="opacity-0 group-hover:opacity-100" icon="i-heroicons-pencil" color="gray" variant="ghost" size="2xs" square @click="startEdit" aria-label="Edit" />
          <UButton class="opacity-0 group-hover:opacity-100" icon="i-heroicons-trash" color="gray" variant="ghost" size="2xs" square @click="store.deleteComment(comment.id, issueId)" aria-label="Delete" />
        </template>
      </div>
      <div v-if="editing" class="mt-1">
        <textarea v-model="draft" rows="2" class="w-full rounded-md border border-subtle bg-elevated px-2 py-1 text-sm outline-none" @keydown.enter.exact.prevent="save" @keydown.esc="editing = false" />
        <div class="mt-1 flex gap-2"><UButton size="2xs" @click="save">[ save ]</UButton><UButton size="2xs" color="gray" variant="ghost" @click="editing = false">[ cancel ]</UButton></div>
      </div>
      <div v-else-if="comment.deletedAt" class="text-sm italic text-tertiary">(deleted)</div>
      <div v-else class="prose-chat text-sm leading-relaxed" v-html="renderMarkdown(comment.body)" />
      <Attachments v-if="store.attachmentsFor('comment', comment.id).length" class="mt-1.5" entity-type="comment" :entity-id="comment.id" readonly />
      <div v-if="reactionGroups(comment).length" class="mt-1 flex flex-wrap gap-1">
        <button v-for="g in reactionGroups(comment)" :key="g.emoji"
          class="rounded border px-1.5 py-0.5 text-xs"
          :class="g.mine ? 'border-accent bg-accent/15 text-accent' : 'border-subtle text-secondary hover:bg-hover'"
          @click="react(g.emoji)">{{ g.emoji }} {{ g.count }}</button>
      </div>
    </div>
  </div>
</template>
