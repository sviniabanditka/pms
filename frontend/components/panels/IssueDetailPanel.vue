<script setup lang="ts">
import { useAppStore } from '~/stores/app'
import { useTimeStore } from '~/stores/time'
import { useAiStore } from '~/stores/ai'
import { PRIO, propSelUi, notifTime, fmtMin } from '~/lib/ui'

// Панель деталей задачи (M4): свойства, лейблы, сабтаски, связи, вложения, AI, комменты, активность.
// Роутовое состояние (selectedIssueId) живёт на странице → сюда приходит issue-id;
// модалки (label/relation/time-entry) тоже на странице → эмиты add-label/add-relation/log-time.
const props = defineProps<{ issueId: string; linkCopied: boolean }>()
const emit = defineEmits<{
  (e: 'open-issue', id: string): void
  (e: 'copy-link'): void
  (e: 'close'): void
  (e: 'add-label'): void
  (e: 'add-relation'): void
  (e: 'log-time', issueId: string): void
}>()

const store = useAppStore()
const time = useTimeStore()
const aiStore = useAiStore()
const { err, wrap } = useActionWrap()
const { statusColor, statusGlyph, prioColor, prioGlyph } = useColors()
const detailPanel = useResizable('detail', 384)
const selectedIssue = computed(() => store.issues[props.issueId])
const newSub = ref('')
// issue description (markdown, inline edit + md toolbar)
const descDraft = ref('')
const descEditing = ref(false)
const descEl = ref<HTMLTextAreaElement | null>(null)
function startDescEdit() { descDraft.value = selectedIssue.value?.description?.md || ''; descEditing.value = true; nextTick(() => descEl.value?.focus()) }
function saveDesc() { descEditing.value = false; if (selectedIssue.value) store.setIssueDescription(selectedIssue.value.id, descDraft.value) }
function descWrap(pre: string, post = pre) {
  const el = descEl.value; if (!el) { descDraft.value += pre + post; return }
  const s = el.selectionStart, e = el.selectionEnd, v = descDraft.value
  descDraft.value = v.slice(0, s) + pre + v.slice(s, e) + post + v.slice(e)
  nextTick(() => { el.focus(); el.selectionStart = s + pre.length; el.selectionEnd = e + pre.length })
}
// AI-резюме описания (Фаза 4)
const descSummary = ref(''); const descSummaryLoading = ref(false)
async function summarizeDesc() {
  if (!selectedIssue.value) return
  descSummaryLoading.value = true
  descSummary.value = await aiStore.summarizeIssue(selectedIssue.value.id)
  descSummaryLoading.value = false
}
const aiSuggest = ref<any>(null)
// сброс/загрузка при смене задачи (раньше — в openIssue страницы)
watch(() => props.issueId, (id) => {
  descSummary.value = ''; descEditing.value = false
  aiSuggest.value = null
  if (id && store.issues[id]) aiStore.suggestForIssue(store.issues[id]).then((r) => { if (props.issueId === id) aiSuggest.value = r })
}, { immediate: true })
const simIssues = computed(() => (aiSuggest.value?.similar || []).filter((s: any) => s.id !== props.issueId))
// подсказки, которых у issue ещё нет
const suggestLabels = computed(() => {
  const iss = selectedIssue.value; if (!iss || !aiSuggest.value) return []
  const have = store.labelsByIssue.get(iss.id) || new Set()
  return (aiSuggest.value.labelIds || []).filter((lid: string) => !have.has(lid) && store.labels[lid]).slice(0, 3)
})
const suggestAssignee = computed(() => {
  const iss = selectedIssue.value; if (!iss || !aiSuggest.value?.assigneeId) return null
  return (!iss.assigneeId && aiSuggest.value.assigneeId !== iss.assigneeId) ? aiSuggest.value.assigneeId : null
})
const PRI_LABELS = ['none', 'urgent', 'high', 'medium', 'low']
const suggestPriority = computed(() => {
  const iss = selectedIssue.value; const p = aiSuggest.value?.priority
  if (!iss || p == null || p === iss.priority) return null
  return p
})
const suggestEstimate = computed(() => {
  const iss = selectedIssue.value; const e = aiSuggest.value?.estimate
  if (!iss || e == null || iss.estimate != null) return null
  return Math.max(1, Math.round(e))
})
const slipWarn = computed(() => {
  const iss = selectedIssue.value
  return !!(aiSuggest.value?.slipRisk && iss?.dueDate && !iss.completedAt)
})
// comments
const commentDraft = ref('')
const commentEl = ref<HTMLTextAreaElement | null>(null)
const commentPending = ref<File[]>([])
const commentFileEl = ref<HTMLInputElement | null>(null)
const onCommentFilePick = (e: Event) => {
  const files = (e.target as HTMLInputElement).files; if (!files?.length) return
  for (const f of Array.from(files)) commentPending.value = [...commentPending.value, f]
  if (commentFileEl.value) commentFileEl.value.value = ''
}
const onCommentPaste = (e: ClipboardEvent) => {
  const items = e.clipboardData?.items; if (!items) return
  for (const it of Array.from(items)) if (it.type.startsWith('image/')) { e.preventDefault(); const f = it.getAsFile(); if (f) commentPending.value = [...commentPending.value, f] }
}
const editCommentId = ref('')
const editCommentDraft = ref('')
const addComment = () => wrap(async () => {
  const b = commentDraft.value.trim()
  if ((!b && !commentPending.value.length) || !selectedIssue.value) return
  const files = commentPending.value.slice()
  commentDraft.value = ''; commentPending.value = []
  await store.addCommentAttached(selectedIssue.value.id, b, files)
})
function startEditComment(c: any) { editCommentId.value = c.id; editCommentDraft.value = c.body }
const saveComment = () => wrap(async () => { if (editCommentDraft.value.trim()) await store.editComment(editCommentId.value, selectedIssue.value.id, editCommentDraft.value); editCommentId.value = '' })
const addSub = () => wrap(async () => { if (newSub.value.trim()) { await store.createSubIssue(props.issueId, newSub.value); newSub.value = '' } })
function cfMultiToggle(fieldId: string, opt: string) {
  const cur = (store.cfValue(props.issueId, fieldId) as string[]) || []
  store.setCustomFieldValue(props.issueId, fieldId, cur.includes(opt) ? cur.filter((x) => x !== opt) : [...cur, opt])
}
function copyText(t: string) { if (import.meta.client) navigator.clipboard?.writeText(t) }
const duplicateIssue = (i: any) => wrap(async () => {
  const id = await store.createIssue(i.teamId, i.title + ' (copy)', i.stateId)
  if (i.priority) await store.setIssuePriority(id, i.priority)
  if (i.assigneeId) await store.setIssueAssignee(id, i.assigneeId)
  emit('open-issue', id)
})
// completed-category state for "Mark complete"
function completedStateId(teamId: string) {
  return store.teamStates(teamId).find((s: any) => s.category === 'completed')?.id || ''
}
const issueActions = computed(() => {
  const i = selectedIssue.value
  if (!i) return []
  const doneId = completedStateId(i.teamId)
  return [
    [
      { label: 'Copy link', icon: 'i-heroicons-link', click: () => emit('copy-link') },
      { label: 'Copy ID', icon: 'i-heroicons-hashtag', click: () => copyText(i.identifier || '') },
    ],
    ...(store.canWrite ? [
      [
        ...(doneId && i.stateId !== doneId ? [{ label: 'Mark complete', icon: 'i-heroicons-check-circle', click: () => store.setIssueState(i.id, doneId) }] : []),
        { label: 'Duplicate', icon: 'i-heroicons-document-duplicate', click: () => duplicateIssue(i) },
      ],
      [
        { label: 'Archive', icon: 'i-heroicons-archive-box', click: () => { store.archiveIssue(i.id); emit('close') } },
      ],
    ] : []),
  ]
})
// inline @/# пикеры (юзеры + сущности)
const descMention = useMention(descDraft, descEl)
const commentMention = useMention(commentDraft, commentEl)
function onCommentKeydown(e: KeyboardEvent) {
  if (commentMention.onKeydown(e)) return
  if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); addComment() }
}
// вложения в описании — chat-модель: paste/пик → attachment (тумба/плашка), не inline md
const descFileEl = ref<HTMLInputElement | null>(null)
const attachFiles = (entityType: string, entityId: string, files?: FileList | File[] | null) => wrap(async () => {
  if (!files || !entityId) return
  for (const f of Array.from(files)) await store.addAttachment(entityType, entityId, f)
})
function onAttachPaste(e: ClipboardEvent, entityType: string, entityId: string) {
  const items = e.clipboardData?.items; if (!items) return
  const fs: File[] = []
  for (const it of Array.from(items)) if (it.kind === 'file') { const f = it.getAsFile(); if (f) fs.push(f) }
  if (fs.length) { e.preventDefault(); attachFiles(entityType, entityId, fs) }
}
const onDescFilePick = (e: Event) => { attachFiles('issue', props.issueId, (e.target as HTMLInputElement).files); (e.target as HTMLInputElement).value = '' }
function onDescKeydown(e: KeyboardEvent) {
  if (descMention.onKeydown(e)) return
  if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') saveDesc()
}
// reactions (комменты задачи)
const EMOJIS = ['👍', '❤️', '😄', '🎉', '👀', '🚀', '✅', '🔥']
function reactionGroups(m: any) {
  const g: Record<string, { emoji: string; count: number; mine: boolean }> = {}
  for (const r of m.reactions || []) {
    ;(g[r.emoji] ||= { emoji: r.emoji, count: 0, mine: false }).count++
    if (r.userId === store.currentUserId) g[r.emoji].mine = true
  }
  return Object.values(g)
}
const cReactPickerId = ref('')
const commentReact = (c: any, emoji: string) => { cReactPickerId.value = ''; if (selectedIssue.value) wrap(() => store.toggleCommentReaction(c.id, selectedIssue.value!.id, emoji)) }
</script>

<template>
  <ResizeHandle @down="detailPanel.onResizeStart" />
  <aside v-if="selectedIssue" :style="{ width: detailPanel.w.value + 'px' }" class="flex shrink-0 flex-col overflow-y-auto rounded-xl border border-subtle bg-surface text-sm">
    <!-- top bar -->
    <div class="flex items-center gap-2 border-b border-subtle px-4 py-2.5">
      <span class="inline-block shrink-0 w-4 text-center leading-none" :style="{ color: statusColor(store.states[selectedIssue.stateId]?.category) }">{{ statusGlyph(store.states[selectedIssue.stateId]?.category) }}</span>
      <span class="text-xs text-accent2">{{ selectedIssue.identifier }}</span>
      <UButton class="ml-auto" :icon="linkCopied ? 'i-heroicons-check' : 'i-heroicons-link'" color="gray" variant="ghost" size="xs" square :title="linkCopied ? 'Copied!' : 'Copy link'" @click="emit('copy-link')" />
      <UDropdown :items="issueActions" :popper="{ placement: 'bottom-end' }">
        <UButton icon="i-heroicons-ellipsis-horizontal" color="gray" variant="ghost" size="xs" square />
      </UDropdown>
      <UButton icon="i-heroicons-x-mark" color="gray" variant="ghost" size="xs" square @click="emit('close')" />
    </div>

    <div class="px-4 py-4">
      <!-- title -->
      <UInput :model-value="selectedIssue.title" variant="none" size="xl" class="mb-3 w-full font-semibold" :padded="false"
        :ui="{ base: 'text-white' }" :disabled="!store.canWrite"
        @change="(v:any) => store.setIssueTitle(selectedIssue.id, v)" />

      <!-- description -->
      <div class="mb-6">
        <div v-if="descEditing" class="rounded-lg border border-subtle bg-elevated">
          <div class="flex items-center gap-0.5 border-b border-subtle px-1.5 py-1">
            <UButton size="2xs" square color="gray" variant="ghost" class="font-bold" title="Bold" @click="descWrap('**')">B</UButton>
            <UButton size="2xs" square color="gray" variant="ghost" class="italic" title="Italic" @click="descWrap('_')">I</UButton>
            <UButton size="2xs" square color="gray" variant="ghost" class="font-bold" title="Heading" @click="descWrap('\n# ', '')">H</UButton>
            <UButton size="2xs" square color="gray" variant="ghost" icon="i-heroicons-list-bullet" title="List" @click="descWrap('\n- ', '')" />
            <UButton size="2xs" square color="gray" variant="ghost" icon="i-heroicons-code-bracket" title="Code" @click="descWrap('`')" />
            <UButton size="2xs" square color="gray" variant="ghost" icon="i-heroicons-link" title="Link" @click="descWrap('[', '](url)')" />
            <UButton size="2xs" square color="gray" variant="ghost" icon="i-heroicons-paper-clip" title="Attach file/image" @click="descFileEl?.click()" />
            <input ref="descFileEl" type="file" multiple class="hidden" @change="onDescFilePick" />
            <UButton class="ml-auto" size="2xs" color="gray" variant="ghost" title="Save (⌘↵)" @click="saveDesc">[ done ]</UButton>
          </div>
          <div class="relative">
            <MentionMenu :m="descMention" />
            <textarea ref="descEl" v-model="descDraft" rows="5"
              class="block w-full resize-y bg-transparent px-3 py-2 text-sm leading-relaxed text-secondary outline-none placeholder:text-tertiary"
              placeholder="Add description… (markdown, @mention, #ref)" @paste="onAttachPaste($event, 'issue', selectedIssue.id)" @input="descMention.detect()" @keydown="onDescKeydown"></textarea>
          </div>
        </div>
        <template v-else-if="selectedIssue.description?.md">
          <div
            class="prose-invert -mx-2 cursor-text rounded-md px-2 py-1.5 text-sm leading-relaxed text-secondary hover:bg-elevated [&_a]:text-accent [&_code]:rounded [&_code]:bg-elevated [&_code]:px-1 [&_h1]:mb-1 [&_h1]:mt-2 [&_h1]:text-base [&_h1]:font-semibold [&_h1]:text-white [&_h2]:mb-1 [&_h2]:mt-2 [&_h2]:font-semibold [&_h2]:text-white [&_li]:ml-4 [&_li]:list-disc [&_p]:mb-1.5 [&_strong]:text-white"
            @click="store.canWrite && startDescEdit()" v-html="renderMarkdown(selectedIssue.description.md)" />
          <!-- AI-резюме для длинных описаний -->
          <div v-if="(selectedIssue.description.md || '').length > 280" class="mt-1.5">
            <button v-if="!descSummary && !descSummaryLoading" class="text-xs text-tertiary hover:text-accent2" @click="summarizeDesc">≈ summarize</button>
            <span v-else-if="descSummaryLoading" class="text-xs text-tertiary">≈ summarizing…</span>
            <div v-else class="rounded-md border border-subtle bg-elevated px-3 py-2 text-xs leading-relaxed text-secondary">
              <span class="text-accent2">≈ TL;DR </span>{{ descSummary }}
            </div>
          </div>
        </template>
        <button v-else-if="store.canWrite" class="-mx-2 flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-sm text-tertiary hover:bg-elevated" @click="startDescEdit">
          <UIcon name="i-heroicons-bars-3-bottom-left" /> Add description…
        </button>
      </div>

      <!-- Properties (guest = read-only: inert отключает все контролы, значения видны) -->
      <div class="mb-2 text-xs font-medium uppercase tracking-wider text-tertiary"><span class="text-ok"># </span>Properties</div>
      <div class="mb-6 space-y-1" :inert="!store.canWrite" :class="{ 'opacity-70': !store.canWrite }">
        <div class="flex min-h-8 items-center gap-3">
          <span class="flex w-24 shrink-0 items-center gap-2 text-tertiary"><span class="inline-block shrink-0 w-4 text-center leading-none" :style="{ color: statusColor(store.states[selectedIssue.stateId]?.category) }">{{ statusGlyph(store.states[selectedIssue.stateId]?.category) }}</span> Status</span>
          <USelectMenu class="flex-1 rounded-md transition-colors hover:bg-elevated" variant="none" :ui="propSelUi" :model-value="selectedIssue.stateId" value-attribute="value" option-attribute="label"
            :options="store.teamStates(selectedIssue.teamId).map((s2) => ({ label: s2.name, value: s2.id }))"
            @update:model-value="store.setIssueState(selectedIssue.id, $event)" />
        </div>
        <div class="flex min-h-8 items-center gap-3">
          <span class="flex w-24 shrink-0 items-center gap-2 text-tertiary"><span class="inline-block shrink-0 w-4 text-center leading-none" :style="{ color: prioColor(selectedIssue.priority ?? 0) }">{{ prioGlyph(selectedIssue.priority ?? 0) }}</span> Priority</span>
          <USelectMenu class="flex-1 rounded-md transition-colors hover:bg-elevated" variant="none" :ui="propSelUi" :model-value="selectedIssue.priority" value-attribute="value" option-attribute="label"
            :options="PRIO.map((p, idx) => ({ label: p, value: idx }))"
            @update:model-value="store.setIssuePriority(selectedIssue.id, Number($event))" />
        </div>
        <div class="flex min-h-8 items-center gap-3">
          <span class="flex w-24 shrink-0 items-center gap-2 text-tertiary"><UIcon name="i-heroicons-user-circle" /> Assignee</span>
          <USelectMenu class="flex-1 rounded-md transition-colors hover:bg-elevated" variant="none" :ui="propSelUi" placeholder="unassigned" :model-value="selectedIssue.assigneeId || undefined" value-attribute="value" option-attribute="label"
            :options="[{ label: 'unassigned', value: '' }, ...store.wsUsers.map((u) => ({ label: u.name || u.email, value: u.id })), ...(store.users[store.currentUserId] ? [{ label: (store.users[store.currentUserId].name || store.users[store.currentUserId].email) + ' (me)', value: store.currentUserId }] : [])]"
            @update:model-value="store.setIssueAssignee(selectedIssue.id, $event)" />
        </div>
        <div class="flex min-h-8 items-center gap-3">
          <span class="flex w-24 shrink-0 items-center gap-2 text-tertiary"><UIcon name="i-heroicons-arrow-path-rounded-square" /> Cycle</span>
          <USelectMenu class="flex-1 rounded-md transition-colors hover:bg-elevated" variant="none" :ui="propSelUi" placeholder="no cycle" :model-value="selectedIssue.cycleId || undefined" value-attribute="value" option-attribute="label"
            :options="[{ label: 'no cycle', value: '' }, ...store.teamCycles(selectedIssue.teamId).map((c) => ({ label: `Cycle ${c.number} (${c.status})`, value: c.id }))]"
            @update:model-value="store.setIssueCycle(selectedIssue.id, $event)" />
        </div>
        <div class="flex min-h-8 items-center gap-3">
          <span class="flex w-24 shrink-0 items-center gap-2 text-tertiary"><UIcon name="i-heroicons-flag" /> Milestone</span>
          <USelectMenu class="flex-1 rounded-md transition-colors hover:bg-elevated" variant="none" :ui="propSelUi" placeholder="no milestone" :model-value="selectedIssue.milestoneId || undefined" value-attribute="value" option-attribute="label"
            :options="[{ label: 'no milestone', value: '' }, ...store.teamMilestones(selectedIssue.teamId).map((m) => ({ label: m.name, value: m.id }))]"
            @update:model-value="store.setIssueMilestone(selectedIssue.id, $event)" />
        </div>
        <div class="flex min-h-8 items-center gap-3">
          <span class="flex w-24 shrink-0 items-center gap-2 text-tertiary"><UIcon name="i-heroicons-calendar" /> Due date</span>
          <DatePicker class="flex-1" placeholder="No due date" :model-value="selectedIssue.scheduledDate ? String(selectedIssue.scheduledDate).slice(0,10) : ''"
            @update:model-value="(v:any) => store.setIssueSchedule(selectedIssue.id, v)" />
        </div>
        <div class="flex min-h-8 items-center gap-3">
          <span class="flex w-24 shrink-0 items-center gap-2 text-tertiary"><UIcon name="i-heroicons-scale" /> Estimate</span>
          <UInput class="flex-1 rounded-md transition-colors hover:bg-elevated" variant="none" :ui="propSelUi" type="number" min="0" :model-value="selectedIssue.estimate ?? ''" placeholder="no estimate"
            @change="(v:any) => store.setIssueEstimate(selectedIssue.id, Number(v))" />
        </div>
        <!-- Time tracking -->
        <div class="flex min-h-8 items-center gap-3">
          <span class="flex w-24 shrink-0 items-center gap-2 text-tertiary"><UIcon name="i-heroicons-clock" /> Time</span>
          <span class="flex-1 text-sm" :class="store.issueTrackedMin(selectedIssue.id) ? 'text-white' : 'text-tertiary'">{{ store.issueTrackedMin(selectedIssue.id) ? fmtMin(store.issueTrackedMin(selectedIssue.id)) : 'Not tracked' }}</span>
          <UButton v-if="store.runningTimer?.issueId === selectedIssue.id" color="red" size="2xs" @click="wrap(() => time.stopTimer())">Stop</UButton>
          <UButton v-else color="gray" size="2xs" @click="wrap(() => time.startTimer(selectedIssue.id))">Start</UButton>
          <UButton color="gray" variant="ghost" size="2xs" square icon="i-heroicons-plus" title="Add time manually" @click="emit('log-time', selectedIssue.id)" />
        </div>
        <!-- записи времени -->
        <div v-for="te in store.timeEntriesForIssue(selectedIssue.id)" :key="te.id" class="group ml-24 flex items-center gap-2 pl-3 text-xs text-tertiary">
          <UIcon :name="te.endedAt ? 'i-heroicons-check-circle' : 'i-heroicons-play-circle'" :class="te.endedAt ? '' : 'text-ok'" class="shrink-0" />
          <span class="tabular-nums">{{ new Date(te.startedAt).toLocaleDateString([], { month: 'short', day: 'numeric' }) }} {{ new Date(te.startedAt).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }) }}</span>
          <span class="flex-1 truncate">{{ te.description }}</span>
          <span v-if="te.endedAt" class="tabular-nums">{{ fmtMin(Math.max(0, Math.round((new Date(te.endedAt).getTime() - new Date(te.startedAt).getTime()) / 60000))) }}</span>
          <span v-else class="text-ok">running</span>
          <UButton class="opacity-0 group-hover:opacity-100" icon="i-heroicons-x-mark" color="gray" variant="ghost" size="2xs" square @click="wrap(() => time.deleteTimeEntry(te.id))" />
        </div>
        <!-- Custom fields -->
        <div v-for="f in store.fieldsForTeam(selectedIssue.teamId)" :key="f.id" class="flex min-h-8 items-center gap-3">
          <span class="w-24 shrink-0 truncate text-tertiary">{{ f.name }}</span>
          <div v-if="f.type==='checkbox'" class="flex-1 pl-2"><TermCheck :model-value="!!store.cfValue(selectedIssue.id, f.id)"
            @update:model-value="store.setCustomFieldValue(selectedIssue.id, f.id, $event)" /></div>
          <USelectMenu v-else-if="f.type==='select'" class="flex-1 rounded-md transition-colors hover:bg-elevated" variant="none" :ui="propSelUi" placeholder="—" value-attribute="value" option-attribute="label"
            :model-value="store.cfValue(selectedIssue.id, f.id) || undefined" :options="[{ label: '—', value: '' }, ...(f.config?.options || []).map((o: string) => ({ label: o, value: o }))]"
            @update:model-value="store.setCustomFieldValue(selectedIssue.id, f.id, $event)" />
          <div v-else-if="f.type==='multiselect'" class="flex flex-1 flex-wrap gap-1 py-1">
            <button v-for="o in (f.config?.options || [])" :key="o"
              class="rounded-md px-2 py-0.5 text-xs"
              :class="(store.cfValue(selectedIssue.id, f.id) || []).includes(o) ? 'bg-active text-white' : 'bg-elevated text-tertiary hover:bg-hover'"
              @click="cfMultiToggle(f.id, o)">{{ o }}</button>
          </div>
          <USelectMenu v-else-if="f.type==='people'" class="flex-1 rounded-md transition-colors hover:bg-elevated" variant="none" :ui="propSelUi" placeholder="—"
            :model-value="store.cfValue(selectedIssue.id, f.id) || undefined" value-attribute="value" option-attribute="label"
            :options="[{ label: '—', value: '' }, ...store.wsUsers.map((u) => ({ label: u.name || u.email, value: u.id }))]"
            @update:model-value="store.setCustomFieldValue(selectedIssue.id, f.id, $event)" />
          <USelectMenu v-else-if="f.type==='relationship'" class="flex-1 rounded-md transition-colors hover:bg-elevated" variant="none" :ui="propSelUi" placeholder="—"
            :model-value="store.cfValue(selectedIssue.id, f.id) || undefined" value-attribute="value" option-attribute="label"
            :options="[{ label: '—', value: '' }, ...store.issuesForTeamAll(selectedIssue.teamId).map((i) => ({ label: `${i.identifier} ${i.title}`, value: i.id }))]"
            @update:model-value="store.setCustomFieldValue(selectedIssue.id, f.id, $event)" />
          <UInput v-else class="flex-1 rounded-md transition-colors hover:bg-elevated" variant="none" :ui="propSelUi" :type="f.type==='number' ? 'number' : f.type==='date' ? 'date' : 'text'"
            :model-value="store.cfValue(selectedIssue.id, f.id) ?? ''" placeholder="—"
            @change="(v:any) => store.setCustomFieldValue(selectedIssue.id, f.id, f.type==='number' ? Number(v) : v)" />
        </div>
      </div>

      <!-- Labels -->
      <div class="mb-5">
        <div class="mb-1.5 text-xs font-medium uppercase tracking-wider text-tertiary"><span class="text-ok"># </span>Labels</div>
        <div class="flex flex-wrap items-center gap-1.5" :inert="!store.canWrite" :class="{ 'opacity-70': !store.canWrite }">
          <span v-for="l in store.labelsForIssue(selectedIssue.id)" :key="l.id" class="group/lbl flex items-center gap-1 rounded-full border border-subtle px-2 py-0.5 text-xs">
            <span :style="{ color: l.colorToken }">#{{ l.name }}</span>
            <UIcon name="i-heroicons-x-mark" class="cursor-pointer text-tertiary hover:text-danger" @click="store.removeLabel(selectedIssue.id, l.id)" />
          </span>
          <button class="flex items-center gap-1 rounded-full border border-dashed border-subtle px-2 py-0.5 text-xs text-tertiary hover:border-strong hover:text-secondary" @click="emit('add-label')">
            <UIcon name="i-heroicons-plus" /> Add label
          </button>
        </div>
      </div>

      <!-- Sub-issues -->
      <div class="mb-5">
        <div class="mb-1.5 flex items-center gap-2 text-xs font-medium uppercase tracking-wider text-tertiary">
          <span class="text-ok"># </span>Sub-issues
          <span v-if="store.childProgress(selectedIssue.id).total" class="rounded bg-elevated px-1.5 text-tertiary">{{ store.childProgress(selectedIssue.id).done }}/{{ store.childProgress(selectedIssue.id).total }}</span>
        </div>
        <div v-for="c in store.subIssuesOf(selectedIssue.id)" :key="c.id"
          class="flex cursor-pointer items-center gap-2 rounded-md px-2 py-1.5 hover:bg-hover" @click="emit('open-issue', c.id)">
          <span class="inline-block shrink-0 w-4 text-center leading-none" :style="{ color: statusColor(store.states[c.stateId]?.category) }">{{ statusGlyph(store.states[c.stateId]?.category) }}</span>
          <span class="text-xs text-accent2">{{ c.identifier }}</span>
          <span class="truncate" :class="{ 'text-tertiary line-through': c.completedAt }">{{ c.title }}</span>
        </div>
        <UInput v-if="store.canWrite" v-model="newSub" placeholder="+ Add sub-issue" size="xs" variant="none" class="w-full" @keyup.enter="addSub" />
      </div>

      <!-- Relations -->
      <div class="mb-4">
        <div class="mb-1.5 text-xs font-medium uppercase tracking-wider text-tertiary"><span class="text-ok"># </span>Relations</div>
        <div v-for="r in store.relationsForIssue(selectedIssue.id)" :key="r.id"
          class="group flex items-center gap-2 rounded-md px-2 py-1.5 hover:bg-hover">
          <span class="text-tertiary">{{ r.label }}</span>
          <span class="cursor-pointer hover:text-accent" @click="emit('open-issue', r.otherId)">{{ store.issueById(r.otherId)?.identifier || '—' }}</span>
          <UButton v-if="store.canWrite" class="ml-auto opacity-0 group-hover:opacity-100" icon="i-heroicons-x-mark" color="gray" variant="ghost" size="2xs" square @click="store.removeRelation(r.id)" />
        </div>
        <button v-if="store.canWrite" class="mt-1 flex items-center gap-1 rounded-md px-2 py-1.5 text-sm text-tertiary hover:bg-hover hover:text-secondary" @click="emit('add-relation')">
          <UIcon name="i-heroicons-plus" /> Add relation
        </button>
      </div>

      <!-- Attachments -->
      <div class="mb-5">
        <div class="mb-1.5 text-xs font-medium uppercase tracking-wider text-tertiary"><span class="text-ok"># </span>Attachments</div>
        <Attachments entity-type="issue" :entity-id="selectedIssue.id" />
      </div>

      <!-- AI: suggestions + similar (Фаза 1–2) -->
      <div v-if="simIssues.length || suggestLabels.length || suggestAssignee || suggestPriority !== null || suggestEstimate !== null || slipWarn" class="mb-4">
        <div class="mb-2 flex items-center gap-2 text-xs font-medium uppercase tracking-wider text-tertiary">
          <span class="text-accent2">≈ </span>AI
        </div>
        <div v-if="slipWarn" class="mb-2 flex items-center gap-1 text-xs text-danger">
          ⚠ likely to miss its due date
        </div>
        <!-- подсказки в один клик -->
        <div v-if="suggestLabels.length || suggestAssignee || suggestPriority !== null || suggestEstimate !== null" class="mb-2 flex flex-wrap gap-1.5">
          <button v-if="suggestPriority !== null" class="flex items-center gap-1 rounded border border-subtle px-2 py-0.5 text-xs text-secondary hover:border-warn hover:text-warn"
            @click="store.setIssuePriority(selectedIssue.id, suggestPriority); aiStore.aiFeedback('priority','applied',selectedIssue.id,String(suggestPriority))">
            ⚑ priority {{ PRI_LABELS[suggestPriority] }}
          </button>
          <button v-if="suggestEstimate !== null" class="flex items-center gap-1 rounded border border-subtle px-2 py-0.5 text-xs text-secondary hover:border-accent hover:text-accent"
            @click="store.setIssueEstimate(selectedIssue.id, suggestEstimate); aiStore.aiFeedback('estimate','applied',selectedIssue.id,String(suggestEstimate))">
            ≈ est {{ suggestEstimate }} pts
          </button>
          <button v-if="suggestAssignee" class="flex items-center gap-1 rounded border border-subtle px-2 py-0.5 text-xs text-secondary hover:border-accent hover:text-accent"
            @click="store.setIssueAssignee(selectedIssue.id, suggestAssignee); aiStore.aiFeedback('assignee','applied',selectedIssue.id,suggestAssignee)">
            ＋ assign {{ store.userName(suggestAssignee) }}
          </button>
          <button v-for="lid in suggestLabels" :key="lid" class="flex items-center gap-1 rounded border border-subtle px-2 py-0.5 text-xs hover:border-accent"
            :style="{ color: store.labels[lid]?.colorToken ? undefined : undefined }" @click="store.addLabel(selectedIssue.id, lid); aiStore.aiFeedback('label','applied',selectedIssue.id,lid)">
            ＋ {{ store.labels[lid]?.name }}
          </button>
        </div>
        <!-- похожие / дубли -->
        <button v-for="s in simIssues" :key="s.id" class="flex w-full items-center gap-2 rounded-sm px-2 py-1 text-left text-sm hover:bg-hover" @click="emit('open-issue', s.id)">
          <span class="shrink-0 text-xs text-accent2">{{ s.identifier }}</span>
          <span class="min-w-0 flex-1 truncate text-secondary">{{ s.title }}</span>
        </button>
      </div>

      <!-- Comments -->
      <div class="mb-4">
        <div class="mb-2 flex items-center gap-2 text-xs font-medium uppercase tracking-wider text-tertiary">
          <span class="text-ok"># </span>Comments <span class="text-tertiary">{{ store.issueComments(selectedIssue.id).length }}</span>
        </div>
        <div v-for="c in store.issueComments(selectedIssue.id)" :key="c.id" class="group mb-2 flex gap-2">
          <Avatar class="mt-0.5" :name="store.userName(c.authorId)" :avatar-key="store.users[c.authorId]?.avatarKey" :size="28" />
          <div class="min-w-0 flex-1">
            <div class="flex items-baseline gap-2">
              <span class="text-sm font-medium">{{ store.userName(c.authorId) }}</span>
              <span class="text-xs text-tertiary">{{ notifTime(c.createdAt) }}</span>
              <span v-if="c.editedAt" class="text-xs text-tertiary/60">(edited)</span>
              <div class="relative ml-auto" :class="c.authorId === store.currentUserId ? '' : 'ml-auto'">
                <UButton v-if="!c.deletedAt" class="opacity-0 group-hover:opacity-100" icon="i-heroicons-face-smile" color="gray" variant="ghost" size="2xs" square @click="cReactPickerId = cReactPickerId === c.id ? '' : c.id" />
                <div v-if="cReactPickerId === c.id" class="absolute right-0 z-20 mt-1 flex gap-0.5 rounded-md border border-subtle bg-elevated p-1 shadow-lg">
                  <button v-for="e in EMOJIS" :key="e" class="rounded p-1 text-base hover:bg-hover" @click="commentReact(c, e)">{{ e }}</button>
                </div>
              </div>
              <template v-if="c.authorId === store.currentUserId && !c.deletedAt && editCommentId !== c.id">
                <UButton class="opacity-0 group-hover:opacity-100" icon="i-heroicons-pencil" color="gray" variant="ghost" size="2xs" square @click="startEditComment(c)" />
                <UButton class="opacity-0 group-hover:opacity-100" icon="i-heroicons-trash" color="gray" variant="ghost" size="2xs" square @click="store.deleteComment(c.id, selectedIssue.id)" />
              </template>
            </div>
            <div v-if="editCommentId === c.id" class="mt-1">
              <textarea v-model="editCommentDraft" rows="2" class="w-full rounded-md border border-subtle bg-elevated px-2 py-1 text-sm outline-none" @keydown.enter.exact.prevent="saveComment" @keydown.esc="editCommentId = ''" />
              <div class="mt-1 flex gap-2"><UButton size="2xs" @click="saveComment">[ save ]</UButton><UButton size="2xs" color="gray" variant="ghost" @click="editCommentId = ''">[ cancel ]</UButton></div>
            </div>
            <div v-else-if="c.deletedAt" class="text-sm italic text-tertiary">(deleted)</div>
            <div v-else class="prose-chat text-sm leading-relaxed" v-html="renderMarkdown(c.body)" />
            <Attachments v-if="store.attachmentsFor('comment', c.id).length" class="mt-1.5" entity-type="comment" :entity-id="c.id" readonly />
            <div v-if="reactionGroups(c).length" class="mt-1 flex flex-wrap gap-1">
              <button v-for="g in reactionGroups(c)" :key="g.emoji"
                class="rounded border px-1.5 py-0.5 text-xs"
                :class="g.mine ? 'border-accent bg-accent/15 text-accent' : 'border-subtle text-secondary hover:bg-hover'"
                @click="commentReact(c, g.emoji)">{{ g.emoji }} {{ g.count }}</button>
            </div>
          </div>
        </div>
        <PendingAttachments v-if="store.canWrite && commentPending.length" class="mt-2" :files="commentPending" @remove="(i) => commentPending = commentPending.filter((_, j) => j !== i)" />
        <div v-if="store.canWrite" class="mt-2 flex items-end gap-2">
          <div class="relative flex flex-1 gap-2 rounded-md border border-subtle bg-elevated px-2.5 py-1.5">
            <MentionMenu :m="commentMention" />
            <span class="select-none pt-px text-ok">&gt;</span>
            <textarea ref="commentEl" v-model="commentDraft" rows="1" placeholder="comment… (markdown, @mention, #ref)"
              class="block w-full resize-none bg-transparent text-sm outline-none placeholder:text-tertiary"
              @paste="onCommentPaste" @input="commentMention.detect()" @keydown="onCommentKeydown" />
          </div>
          <UButton icon="i-heroicons-paper-clip" color="gray" variant="ghost" size="sm" square title="Attach" @click="commentFileEl?.click()" />
          <input ref="commentFileEl" type="file" multiple class="hidden" @change="onCommentFilePick" />
          <UButton icon="i-heroicons-paper-airplane" size="sm" :disabled="!commentDraft.trim() && !commentPending.length" @click="addComment" />
        </div>
      </div>

      <!-- Activity -->
      <div class="mb-4">
        <div class="mb-1.5 text-xs font-medium uppercase tracking-wider text-tertiary"><span class="text-ok"># </span>Activity</div>
        <div v-for="(a, idx) in store.issueActivity(selectedIssue.id)" :key="idx" class="flex items-center gap-2 py-1 text-xs text-tertiary">
          <UIcon :name="a.op === 'create' ? 'i-heroicons-plus-circle' : a.op === 'delete' ? 'i-heroicons-trash' : 'i-heroicons-pencil'" />
          <span class="text-secondary">{{ a.actorId ? store.userName(a.actorId) : 'Someone' }}</span>
          <span>{{ a.op === 'create' ? 'created' : a.op === 'delete' ? 'deleted' : 'updated' }}</span>
          <span class="ml-auto">{{ notifTime(a.at) }}</span>
        </div>
        <div v-if="!store.issueActivity(selectedIssue.id).length" class="text-xs text-tertiary/60">No activity</div>
      </div>

      <p v-if="err" class="text-xs text-danger">{{ err }}</p>
    </div>
  </aside>
</template>
