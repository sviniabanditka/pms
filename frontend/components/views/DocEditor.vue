<script setup lang="ts">
import { fileViewUrl } from '~/lib/upload'
import { useAppStore } from '~/stores/app'

// Rich doc editor: md-тулбар + live preview, @/# меншены, paste/attach файлов,
// автосейв с дебаунсом. Роут-стейт (selectedDocId) — на странице → doc-id проп,
// навигация по breadcrumbs/закрытие — эмиты.
const props = defineProps<{ docId: string; linkCopied: boolean }>()
const emit = defineEmits<{
  (e: 'open-team', teamId: string): void
  (e: 'open-docs', teamId: string): void
  (e: 'copy-link'): void
  (e: 'close'): void
}>()

const store = useAppStore()
const { wrap } = useActionWrap()
const doc = computed(() => store.documents[props.docId])

// rich doc editor (md тулбар + live preview)
const docMode = ref<'edit' | 'split' | 'preview'>('split')
const docEl = ref<HTMLTextAreaElement | null>(null)
const docDraft = ref('')
watch(() => props.docId, (id) => { docDraft.value = id ? (store.documents[id]?.body?.md || '') : '' }, { immediate: true })
let docSaveTimer: any = null
function onDocInput() {
  if (!props.docId) return
  clearTimeout(docSaveTimer)
  docSaveTimer = setTimeout(() => store.updateDoc(props.docId, { md: docDraft.value }), 600)
}
function docWrap(pre: string, post = pre) {
  const el = docEl.value; if (!el) { docDraft.value += pre + post; onDocInput(); return }
  const s = el.selectionStart, e = el.selectionEnd, v = docDraft.value
  docDraft.value = v.slice(0, s) + pre + v.slice(s, e) + post + v.slice(e)
  onDocInput(); nextTick(() => { el.focus(); el.selectionStart = s + pre.length; el.selectionEnd = e + pre.length })
}

// inline @/# пикеры (юзеры + сущности)
const docMention = useMention(docDraft, docEl)
const docFileEl = ref<HTMLInputElement | null>(null)
// доки: вставка прямо в тело (markdown), рендер как обычно
function docInsertAtCaret(text: string) {
  const el = docEl.value
  const pos = el ? (el.selectionStart ?? docDraft.value.length) : docDraft.value.length
  docDraft.value = docDraft.value.slice(0, pos) + text + docDraft.value.slice(pos)
  onDocInput()
  nextTick(() => { if (el) { el.focus(); const c = pos + text.length; el.selectionStart = el.selectionEnd = c } })
}
const docInsertFile = (file: File) => wrap(async () => {
  const up = await store.uploadAndGetKey(file)
  const url = fileViewUrl(up.key)
  docInsertAtCaret(file.type.startsWith('image/') ? `\n![${file.name || 'image'}](${url})\n` : `[📎 ${file.name}](${url})`)
})
function onDocPaste(e: ClipboardEvent) {
  const items = e.clipboardData?.items; if (!items) return
  const fs: File[] = []
  for (const it of Array.from(items)) if (it.kind === 'file') { const f = it.getAsFile(); if (f) fs.push(f) }
  if (fs.length) { e.preventDefault(); fs.forEach(docInsertFile) }
}
const onDocFilePick = (e: Event) => { const files = (e.target as HTMLInputElement).files; if (files) Array.from(files).forEach(docInsertFile); (e.target as HTMLInputElement).value = '' }
function onDocKeydown(e: KeyboardEvent) {
  if (docMention.onKeydown(e)) return
}
</script>

<template>
  <template v-if="doc">
    <div class="sticky top-0 z-10 flex items-center gap-1.5 border-b border-subtle bg-surface px-4 py-3 text-sm">
      <template v-if="doc.teamId && store.teams[doc.teamId]">
        <button class="text-tertiary hover:text-secondary" @click="emit('open-team', doc.teamId!)">{{ store.teams[doc.teamId!].name }}</button>
        <span class="text-tertiary">/</span>
        <button class="text-tertiary hover:text-secondary" @click="emit('open-docs', doc.teamId!)">Docs</button>
        <span class="text-tertiary">/</span>
      </template>
      <UIcon name="i-heroicons-document-text" class="text-tertiary" />
      <UInput :model-value="doc.title" aria-label="Document title" variant="none" :padded="false" class="flex-1 font-medium"
        @change="(v:any) => store.updateDoc(doc.id, { title: v })" />
      <div class="flex gap-0.5">
        <UButton size="2xs" square icon="i-heroicons-pencil" :color="docMode==='edit'?'primary':'gray'" :variant="docMode==='edit'?'soft':'ghost'" title="Edit" @click="docMode='edit'" aria-label="Edit" />
        <UButton size="2xs" square icon="i-heroicons-view-columns" :color="docMode==='split'?'primary':'gray'" :variant="docMode==='split'?'soft':'ghost'" title="Split" @click="docMode='split'" aria-label="Split" />
        <UButton size="2xs" square icon="i-heroicons-eye" :color="docMode==='preview'?'primary':'gray'" :variant="docMode==='preview'?'soft':'ghost'" title="Preview" @click="docMode='preview'" aria-label="Preview" />
      </div>
      <UButton class="ml-1" :icon="linkCopied ? 'i-heroicons-check' : 'i-heroicons-link'" size="xs" color="gray" variant="ghost" square :title="linkCopied?'Copied!':'Copy link'" @click="emit('copy-link')" :aria-label="linkCopied?'Copied!':'Copy link'" />
      <UButton size="xs" color="red" variant="ghost" square icon="i-heroicons-trash" @click="store.deleteDoc(doc.id); emit('close')" aria-label="Delete" />
      <UButton size="xs" color="gray" variant="ghost" square icon="i-heroicons-x-mark" @click="emit('close')" aria-label="Close" />
    </div>
    <!-- toolbar -->
    <div v-if="docMode !== 'preview'" class="flex items-center gap-1 border-b border-subtle px-3 py-1">
      <UButton size="2xs" square color="gray" variant="ghost" class="font-bold" title="Bold" @click="docWrap('**')" aria-label="Bold">B</UButton>
      <UButton size="2xs" square color="gray" variant="ghost" class="italic" title="Italic" @click="docWrap('_')" aria-label="Italic">I</UButton>
      <UButton size="2xs" square color="gray" variant="ghost" class="font-bold" title="Heading" @click="docWrap('\n# ', '')" aria-label="Heading">H</UButton>
      <UButton size="2xs" square color="gray" variant="ghost" icon="i-heroicons-list-bullet" title="List" @click="docWrap('\n- ', '')" aria-label="List" />
      <UButton size="2xs" square color="gray" variant="ghost" icon="i-heroicons-code-bracket" title="Code" @click="docWrap('`')" aria-label="Code" />
      <UButton size="2xs" square color="gray" variant="ghost" icon="i-heroicons-link" title="Link" @click="docWrap('[', '](url)')" aria-label="Link" />
      <UButton size="2xs" square color="gray" variant="ghost" icon="i-heroicons-paper-clip" title="Attach file/image" @click="docFileEl?.click()" aria-label="Attach file/image" />
      <input ref="docFileEl" type="file" multiple class="hidden" @change="onDocFilePick" />
    </div>
    <div class="flex h-[calc(100vh-150px)]">
      <div v-if="docMode !== 'preview'" class="relative" :class="docMode === 'split' ? 'w-1/2 border-r border-subtle' : 'w-full'">
        <MentionMenu :m="docMention" />
        <textarea ref="docEl" v-model="docDraft" placeholder="Write… (markdown, @mention, #ref, paste images/files)"
          class="h-full w-full resize-none bg-transparent p-4 text-sm leading-relaxed outline-none"
          @paste="onDocPaste" @input="onDocInput(); docMention.detect()" @keydown="onDocKeydown" />
      </div>
      <div v-if="docMode !== 'edit'" :class="docMode === 'split' ? 'w-1/2' : 'w-full'" class="prose-chat overflow-y-auto p-4 text-sm leading-relaxed" v-html="renderMarkdown(docDraft) || '<span class=&quot;text-tertiary&quot;>// nothing to preview</span>'" />
    </div>
  </template>
</template>
