<script setup lang="ts">
import { useAppStore } from '~/stores/app'
import { useChatStore } from '~/stores/chat'
import { modalUi } from '~/lib/ui'

// Глобальная create-модалка (Linear-стиль): поля под выбранную сущность.
// type задаёт страница (openCreate), навигация после создания — через emit('created').
const props = defineProps<{ type: string; defaultTeam?: string }>()
const emit = defineEmits<{ (e: 'created', payload: { type: string; id?: string; teamId?: string }): void }>()
const open = defineModel<boolean>()

const store = useAppStore()
const chat = useChatStore()
const { wrap } = useActionWrap()

const CREATE_TYPES = [
  { label: 'Issue', value: 'issue' },
  { label: 'Doc', value: 'doc' },
  { label: 'Cycle', value: 'cycle' },
  { label: 'Milestone', value: 'milestone' },
  { label: 'Meeting', value: 'meeting' },
  { label: 'Dashboard', value: 'dashboard' },
  { label: 'Channel', value: 'channel' },
  { label: 'Team', value: 'team' },
  { label: 'Workspace', value: 'workspace' },
]
const cTeam = ref(''); const cTitle = ref('')
const cStart = ref(''); const cEnd = ref(''); const cMsDate = ref(''); const cPrivate = ref(false)
const cKey = ref(''); const cName = ref(''); const cParent = ref(''); const cSlug = ref('')
const cMeetDate = ref(''); const cMeetStart = ref('09:00'); const cMeetEnd = ref('09:30'); const cMeetParts = ref<string[]>([])
// при открытии — дефолтная команда (как это делал openCreate на странице)
watch(open, (v) => { if (v) cTeam.value = props.defaultTeam || store.teamList[0]?.id || '' })
const createTypeLabel = computed(() => CREATE_TYPES.find((c) => c.value === props.type)?.label || 'Item')
const submitCreate = () => wrap(async () => {
  const t = props.type
  if (t === 'issue' && cTeam.value && cTitle.value.trim()) await store.createIssue(cTeam.value, cTitle.value)
  else if (t === 'doc' && cTeam.value && cTitle.value.trim()) { const id = await store.createDoc(cTeam.value, cTitle.value); emit('created', { type: 'doc', id, teamId: cTeam.value }) }
  else if (t === 'dashboard' && cTitle.value.trim()) { const id = await store.createDashboard(cTitle.value); emit('created', { type: 'dashboard', id }) }
  else if (t === 'channel' && cTitle.value.trim()) { const id = await chat.createChannel(cTitle.value, cPrivate.value); cPrivate.value = false; emit('created', { type: 'channel', id }) }
  else if (t === 'cycle' && cTeam.value && cStart.value && cEnd.value) await store.createCycle(cTeam.value, cStart.value, cEnd.value)
  else if (t === 'milestone' && cTeam.value && cTitle.value.trim()) await store.createMilestone(cTeam.value, cTitle.value, cMsDate.value)
  else if (t === 'meeting' && cTitle.value.trim()) {
    const at = cMeetDate.value && cMeetStart.value ? new Date(`${cMeetDate.value}T${cMeetStart.value}:00`).toISOString() : ''
    const end = at && cMeetEnd.value ? new Date(`${cMeetDate.value}T${cMeetEnd.value}:00`).toISOString() : ''
    await store.createMeeting(cTitle.value, at, end, [...cMeetParts.value]); emit('created', { type: 'meeting' })
  }
  else if (t === 'team' && cKey.value && cName.value) await store.createTeam(cKey.value, cName.value, cParent.value)
  else if (t === 'workspace' && cSlug.value && cName.value) await store.createWorkspace(cSlug.value, cName.value)
  else return
  cTitle.value = cStart.value = cEnd.value = cMsDate.value = cKey.value = cName.value = cSlug.value = cMeetDate.value = ''
  cMeetStart.value = '09:00'; cMeetEnd.value = '09:30'; cMeetParts.value = []
  open.value = false
})
</script>

<template>
  <UModal v-model="open" :ui="modalUi">
    <div class="p-4">
      <div class="mb-4 flex items-center gap-2">
        <span class="text-base font-semibold text-white"><span class="text-ok">$</span> new {{ createTypeLabel.toLowerCase() }}</span>
        <UButton class="ml-auto" icon="i-heroicons-x-mark" color="gray" variant="ghost" size="xs" square @click="open = false" />
      </div>

      <!-- ISSUE -->
      <div v-if="type === 'issue'" class="space-y-3">
        <div><label class="mb-1 block text-xs text-tertiary">Team</label>
          <USelectMenu v-model="cTeam" class="w-full" value-attribute="value" option-attribute="label"
            :options="store.teamList.map((t) => ({ label: `${t.key} · ${t.name}`, value: t.id }))" /></div>
        <div><label class="mb-1 block text-xs text-tertiary">Title</label>
          <UInput v-model="cTitle" placeholder="Issue title" autofocus @keyup.enter="submitCreate" /></div>
      </div>

      <!-- DOC -->
      <div v-else-if="type === 'doc'" class="space-y-3">
        <div><label class="mb-1 block text-xs text-tertiary">Team</label>
          <USelectMenu v-model="cTeam" class="w-full" value-attribute="value" option-attribute="label"
            :options="store.teamList.map((t) => ({ label: `${t.key} · ${t.name}`, value: t.id }))" /></div>
        <div><label class="mb-1 block text-xs text-tertiary">Title</label>
          <UInput v-model="cTitle" placeholder="Document title" autofocus @keyup.enter="submitCreate" /></div>
      </div>

      <!-- CYCLE -->
      <div v-else-if="type === 'cycle'" class="space-y-3">
        <div><label class="mb-1 block text-xs text-tertiary">Team</label>
          <USelectMenu v-model="cTeam" class="w-full" value-attribute="value" option-attribute="label"
            :options="store.teamList.map((t) => ({ label: `${t.key} · ${t.name}`, value: t.id }))" /></div>
        <div class="flex gap-2">
          <div class="flex-1"><label class="mb-1 block text-xs text-tertiary">Start</label><DatePicker v-model="cStart" block /></div>
          <div class="flex-1"><label class="mb-1 block text-xs text-tertiary">End</label><DatePicker v-model="cEnd" block /></div>
        </div>
      </div>

      <!-- MILESTONE -->
      <div v-else-if="type === 'milestone'" class="space-y-3">
        <div><label class="mb-1 block text-xs text-tertiary">Team</label>
          <USelectMenu v-model="cTeam" class="w-full" value-attribute="value" option-attribute="label"
            :options="store.teamList.map((t) => ({ label: `${t.key} · ${t.name}`, value: t.id }))" /></div>
        <div><label class="mb-1 block text-xs text-tertiary">Name</label>
          <UInput v-model="cTitle" placeholder="Milestone name" autofocus @keyup.enter="submitCreate" /></div>
        <div><label class="mb-1 block text-xs text-tertiary">Target date</label>
          <DatePicker v-model="cMsDate" block /></div>
      </div>

      <!-- MEETING -->
      <div v-else-if="type === 'meeting'" class="space-y-3">
        <div><label class="mb-1 block text-xs text-tertiary">Title</label>
          <UInput v-model="cTitle" placeholder="Meeting title" autofocus @keyup.enter="submitCreate" /></div>
        <div><label class="mb-1 block text-xs text-tertiary">When (optional)</label>
          <div class="flex items-center gap-2">
            <DatePicker v-model="cMeetDate" block class="flex-1" placeholder="pick a date" />
            <TimePicker v-model="cMeetStart" class="w-28" />
            <span class="text-tertiary">–</span>
            <TimePicker v-model="cMeetEnd" :from="cMeetStart" class="w-28" />
          </div>
        </div>
        <div><label class="mb-1 block text-xs text-tertiary">Participants</label>
          <USelectMenu v-model="cMeetParts" multiple searchable searchable-placeholder="Search people…" placeholder="Select participants" class="w-full"
            value-attribute="value" option-attribute="label"
            :options="store.wsUsers.map((u) => ({ label: u.name || u.email, value: u.id }))" />
          <div class="mt-1 text-xs text-tertiary">You're the host; others are invited. You can start right away.</div>
        </div>
      </div>

      <!-- DASHBOARD -->
      <div v-else-if="type === 'dashboard'" class="space-y-3">
        <div><label class="mb-1 block text-xs text-tertiary">Name</label>
          <UInput v-model="cTitle" placeholder="Dashboard name" autofocus @keyup.enter="submitCreate" /></div>
      </div>

      <!-- CHANNEL -->
      <div v-else-if="type === 'channel'" class="space-y-3">
        <div><label class="mb-1 block text-xs text-tertiary">Channel name</label>
          <UInput v-model="cTitle" placeholder="general" autofocus @keyup.enter="submitCreate" /></div>
        <label class="flex cursor-pointer items-center gap-2 text-sm">
          <TermCheck v-model="cPrivate" />
          <span class="flex items-center gap-1"><UIcon name="i-heroicons-lock-closed" class="text-tertiary" /> Private — invite-only</span>
        </label>
      </div>

      <!-- TEAM -->
      <div v-else-if="type === 'team'" class="space-y-3">
        <div><label class="mb-1 block text-xs text-tertiary">Parent</label>
          <USelectMenu v-model="cParent" class="w-full" value-attribute="value" option-attribute="label"
            :options="[{ label: '— root team —', value: '' }, ...store.teamList.map((t) => ({ label: `under ${t.key}`, value: t.id }))]" /></div>
        <div class="flex gap-2">
          <div class="w-24"><label class="mb-1 block text-xs text-tertiary">Key</label><UInput v-model="cKey" placeholder="KEY" class="w-full" /></div>
          <div class="flex-1"><label class="mb-1 block text-xs text-tertiary">Name</label><UInput v-model="cName" placeholder="Team name" class="w-full" /></div>
        </div>
      </div>

      <!-- WORKSPACE -->
      <div v-else-if="type === 'workspace'" class="space-y-3">
        <div><label class="mb-1 block text-xs text-tertiary">Slug</label><UInput v-model="cSlug" placeholder="acme" /></div>
        <div><label class="mb-1 block text-xs text-tertiary">Name</label><UInput v-model="cName" placeholder="Acme Inc." /></div>
      </div>

      <div class="mt-4 flex justify-end gap-2">
        <UButton color="gray" variant="ghost" @click="open = false">[ cancel ]</UButton>
        <UButton @click="submitCreate">[ create {{ createTypeLabel.toLowerCase() }} ]</UButton>
      </div>
    </div>
  </UModal>
</template>
