<script setup lang="ts">
import { useAppStore } from '~/stores/app'
import { PRIO, labelOpts, modalUi, ROLES } from '~/lib/ui'

// Workspace Settings (большая модалка с сайдбаром табов).
// open/tab живут на странице (открывается из нескольких мест, wsMenu ставит tab='members');
// навигация наружу (new team, statuses, open issue, logout) — через emits.
const props = defineProps<{
  selectedTeamId: string
  cfgList: { key: string; label: string; icon: string; visible: boolean }[]
  notifPrefs: Record<string, boolean>
}>()
const emit = defineEmits<{
  (e: 'new-team'): void
  (e: 'open-states', teamId: string): void
  (e: 'delete-team', team: any): void
  (e: 'open-issue', id: string): void
  (e: 'logout'): void
  (e: 'toggle-pref', key: string): void
  (e: 'toggle-section', i: number): void
  (e: 'move-section', i: number, dir: number): void
}>()
const open = defineModel<boolean>()
const tab = defineModel<string>('tab', { default: 'general' })

const store = useAppStore()
const { err, wrap } = useActionWrap()
const { inviteEmail, inviteRole, inviteLink, doInvite, copyInviteLink } = useInvites()
const { density, applyDensity } = useDensity()

// мелкие локальные хелперы (дубли одно-строчников страницы)
const d10 = (s: string) => (s ? String(s).slice(0, 10) : '')
const dayShort = (ds: string) => new Date(ds + 'T00:00:00').toLocaleDateString('en', { month: 'short', day: 'numeric' })
function copyText(t: string) { if (import.meta.client) navigator.clipboard?.writeText(t) }

const SETTINGS_TABS = [
  { key: 'general', label: 'General', icon: 'i-heroicons-cog-6-tooth', group: 'Workspace' },
  { key: 'members', label: 'Members', icon: 'i-heroicons-user-group', group: 'Workspace' },
  { key: 'teams', label: 'Teams', icon: 'i-heroicons-rectangle-stack', group: 'Workspace' },
  { key: 'labels', label: 'Labels', icon: 'i-heroicons-tag', group: 'Workspace' },
  { key: 'automations', label: 'Automations', icon: 'i-heroicons-bolt', group: 'Workspace' },
  { key: 'templates', label: 'Templates', icon: 'i-heroicons-document-duplicate', group: 'Workspace' },
  { key: 'apitokens', label: 'API tokens', icon: 'i-heroicons-key', group: 'Workspace' },
  { key: 'webhooks', label: 'Webhooks', icon: 'i-heroicons-globe-alt', group: 'Workspace' },
  { key: 'account', label: 'Account', icon: 'i-heroicons-user-circle', group: 'Personal' },
  { key: 'appearance', label: 'Appearance', icon: 'i-heroicons-swatch', group: 'Personal' },
  { key: 'notifications', label: 'Notifications', icon: 'i-heroicons-bell', group: 'Personal' },
  { key: 'sections', label: 'Sidebar', icon: 'i-heroicons-bars-3', group: 'Personal' },
]
const settingsGroups = computed(() => {
  const g: Record<string, any[]> = {}
  for (const t of SETTINGS_TABS) (g[t.group] ||= []).push(t)
  return g
})
// при открытии — свежий драфт имени и сброс инлайн-редактора правил (как делал openSettings)
const wsNameDraft = ref('')
watch(open, (v) => { if (v) { wsNameDraft.value = store.activeWorkspace?.name || ''; showAuto.value = false } })
const saveWsName = () => wrap(() => store.renameWorkspace(wsNameDraft.value.trim()))
// avatar / logo upload
const avatarEl = ref<HTMLInputElement | null>(null)
const logoEl = ref<HTMLInputElement | null>(null)
const onAvatarPick = (e: Event) => wrap(async () => {
  const f = (e.target as HTMLInputElement).files?.[0]; if (!f) return
  const up = await store.uploadAndGetKey(f); await store.setAvatar(up.key)
  if (avatarEl.value) avatarEl.value.value = ''
})
const onLogoPick = (e: Event) => wrap(async () => {
  const f = (e.target as HTMLInputElement).files?.[0]; if (!f) return
  const up = await store.uploadAndGetKey(f); await store.setLogo(up.key)
  if (logoEl.value) logoEl.value.value = ''
})
function confirmDeleteWs() {
  if (!import.meta.client || !confirm(`Delete workspace "${store.activeWorkspace?.name}"? This hides it and all its data.`)) return
  wrap(async () => { await store.deleteWorkspace(); open.value = false })
}
// labels (workspace-wide管理)
const allLabels = computed(() => Object.values(store.labels))
const newWsLabelName = ref('')
const newWsLabelColor = ref('#5e6ad2')
const LABEL_COLORS = ['#5e6ad2', '#e5484d', '#d29922', '#3fb950', '#1f9cf0', '#a371f7', '#6b747e']
const addWsLabel = () => wrap(async () => {
  const tid = props.selectedTeamId || store.teamList[0]?.id
  if (!newWsLabelName.value.trim() || !tid) return
  await store.createLabel(tid, newWsLabelName.value, newWsLabelColor.value); newWsLabelName.value = ''
})

// --- Automations (rule builder) ---
const showAuto = ref(false)
const autoForm = reactive({ id: '', name: '', teamId: '', enabled: true, event: 'status_changed', statusTarget: 'cat:completed', actions: [] as any[] })
const AUTO_EVENTS = [
  { value: 'created', label: 'Issue created' },
  { value: 'status_changed', label: 'Status changed' },
  { value: 'assignee_changed', label: 'Assignee changed' },
  { value: 'priority_changed', label: 'Priority changed' },
]
const ACTION_TYPES = [
  { value: 'set_status', label: 'Set status' },
  { value: 'set_assignee', label: 'Set assignee' },
  { value: 'set_priority', label: 'Set priority' },
  { value: 'add_label', label: 'Add label' },
  { value: 'post_message', label: 'Post to channel' },
]
const stateOpts = computed(() => Object.values(store.states).filter((s: any) => !s.deletedAt).map((s: any) => ({ value: s.id, label: `${store.teams[s.teamId]?.key || '?'} · ${s.name}` })))
const statusTargetOpts = computed(() => [
  { value: '', label: 'Any status' },
  { value: 'cat:started', label: 'Category: Started' },
  { value: 'cat:completed', label: 'Category: Completed' },
  { value: 'cat:canceled', label: 'Category: Canceled' },
  ...stateOpts.value.map((s) => ({ value: 'state:' + s.value, label: s.label })),
])
const assigneeOpts = computed(() => [{ value: 'me', label: 'Me (trigger actor)' }, { value: '', label: 'Unassign' }, ...store.wsUsers.map((u: any) => ({ value: u.id, label: u.name || u.email }))])
const labelOptsAuto = computed(() => Object.values(store.labels).filter((l: any) => !l.deletedAt).map((l: any) => ({ value: l.id, label: l.name })))
const channelOptsAuto = computed(() => store.channelList.map((c: any) => ({ value: c.id, label: c.name })))
function defaultAction(type: string) {
  if (type === 'set_status') return { type, stateId: '' }
  if (type === 'set_assignee') return { type, userId: 'me' }
  if (type === 'set_priority') return { type, priority: 2 }
  if (type === 'add_label') return { type, labelId: '' }
  return { type: 'post_message', channelId: '', text: '{issue} updated' }
}
function openAutoCreate() {
  Object.assign(autoForm, { id: '', name: '', teamId: '', enabled: true, event: 'status_changed', statusTarget: 'cat:completed', actions: [defaultAction('set_assignee')] })
  showAuto.value = true
}
function openAutoEdit(a: any) {
  const t = a.trigger || {}
  let st = ''
  if (t.toStateId) st = 'state:' + t.toStateId
  else if (t.toCategory) st = 'cat:' + t.toCategory
  Object.assign(autoForm, { id: a.id, name: a.name, teamId: a.teamId || '', enabled: a.enabled, event: t.event || 'status_changed', statusTarget: st, actions: JSON.parse(JSON.stringify(a.actions || [])) })
  showAuto.value = true
}
function addAutoAction() { autoForm.actions.push(defaultAction('set_status')) }
function onActionType(i: number) { autoForm.actions[i] = defaultAction(autoForm.actions[i].type) }
const saveAuto = () => wrap(async () => {
  if (!autoForm.name.trim()) { err.value = 'Name required'; return }
  const trigger: any = { event: autoForm.event }
  if (autoForm.event === 'status_changed' && autoForm.statusTarget) {
    if (autoForm.statusTarget.startsWith('cat:')) trigger.toCategory = autoForm.statusTarget.slice(4)
    else if (autoForm.statusTarget.startsWith('state:')) trigger.toStateId = autoForm.statusTarget.slice(6)
  }
  const p = { teamId: autoForm.teamId, name: autoForm.name, enabled: autoForm.enabled, trigger, actions: autoForm.actions }
  if (autoForm.id) await store.updateAutomation(autoForm.id, p)
  else await store.createAutomation(p)
  showAuto.value = false
})
// человекочитаемое описание правила для списка
function autoSummary(a: any) {
  const t = a.trigger || {}
  const ev = AUTO_EVENTS.find((e) => e.value === t.event)?.label || t.event
  let cond = ''
  if (t.event === 'status_changed') {
    if (t.toCategory) cond = ` → ${t.toCategory}`
    else if (t.toStateId) cond = ` → ${store.states[t.toStateId]?.name || '?'}`
  }
  const acts = (a.actions || []).map((x: any) => ACTION_TYPES.find((y) => y.value === x.type)?.label || x.type).join(', ')
  return `${ev}${cond} ⇒ ${acts}`
}

// --- Issue templates + recurring ---
const showTmpl = ref(false)
const REC_OPTS = [{ value: 'none', label: 'No repeat' }, { value: 'daily', label: 'Daily' }, { value: 'weekly', label: 'Weekly' }, { value: 'monthly', label: 'Monthly' }]
const tmplForm = reactive({ id: '', name: '', teamId: '', title: '', descriptionMd: '', priority: 0, assigneeId: '', estimate: 0, labelIds: [] as string[], recurrence: 'none' })
function openTmplCreate() {
  Object.assign(tmplForm, { id: '', name: '', teamId: store.teamList[0]?.id || '', title: '', descriptionMd: '', priority: 0, assigneeId: '', estimate: 0, labelIds: [], recurrence: 'none' })
  showTmpl.value = true
}
function openTmplEdit(t: any) {
  Object.assign(tmplForm, {
    id: t.id, name: t.name, teamId: t.teamId, title: t.title || '', descriptionMd: t.description?.md || '',
    priority: t.priority || 0, assigneeId: t.assigneeId || '', estimate: t.estimate || 0,
    labelIds: [...(t.labelIds || [])], recurrence: t.recurrence || 'none',
  })
  showTmpl.value = true
}
const saveTmpl = () => wrap(async () => {
  if (!tmplForm.name.trim()) { err.value = 'Name required'; return }
  if (!tmplForm.teamId) { err.value = 'Team required'; return }
  const p = { ...tmplForm }
  if (tmplForm.id) await store.updateTemplate(tmplForm.id, p)
  else await store.createTemplate(p)
  showTmpl.value = false
})
const spawnTmpl = (id: string) => wrap(async () => { const iid = await store.spawnTemplate(id); open.value = false; emit('open-issue', iid) })
// лейблы команды для пикера шаблона
const tmplLabelOpts = computed(() => Object.values(store.labels).filter((l: any) => !l.deletedAt && (!tmplForm.teamId || l.teamId === tmplForm.teamId)).map((l: any) => ({ value: l.id, label: l.name })))
const tmplAssigneeOpts = computed(() => [{ value: '', label: 'Unassigned' }, ...store.wsUsers.map((u: any) => ({ value: u.id, label: u.name || u.email }))])

// --- API tokens (REST /api/tokens, cookie-auth) ---
const apiTokens = ref<any[]>([])
const newTokenName = ref('')
const newTokenExpiry = ref(0) // дней; 0 = бессрочно
const newTokenReadOnly = ref(false)
const revealedToken = ref('')
const apiBase = () => (useRuntimeConfig().public.apiBase as string) || ''
async function loadApiTokens() {
  if (!store.activeWorkspaceId) return
  const r = await fetch(`${apiBase()}/api/tokens?workspaceId=${store.activeWorkspaceId}`, { credentials: 'include' })
  apiTokens.value = r.ok ? await r.json() : []
}
async function createApiToken() {
  const name = newTokenName.value.trim()
  if (!name) return
  const r = await fetch(`${apiBase()}/api/tokens`, {
    method: 'POST', credentials: 'include', headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ workspaceId: store.activeWorkspaceId, name, expiresInDays: newTokenExpiry.value || 0, readOnly: newTokenReadOnly.value }),
  })
  if (r.ok) { revealedToken.value = (await r.json()).token; newTokenName.value = ''; await loadApiTokens() }
}
async function revokeApiToken(id: string) {
  if (import.meta.client && !confirm('Revoke this token? Apps using it stop working.')) return
  await fetch(`${apiBase()}/api/tokens/${id}?workspaceId=${store.activeWorkspaceId}`, { method: 'DELETE', credentials: 'include' })
  await loadApiTokens()
}
watch(tab, (t) => { if (t === 'apitokens') { revealedToken.value = ''; loadApiTokens() } })
// --- Webhooks (REST /api/webhooks) ---
const EVENT_OPTIONS = ['*', 'issue.create', 'issue.update', 'issue.archive', 'comment.create', 'document.create', 'document.update', 'cycle.create', 'milestone.create']
const webhooksList = ref<any[]>([])
const newWebhookUrl = ref('')
const newWebhookEvents = ref<string[]>(['issue.create'])
const revealedSecret = ref('')
async function loadWebhooks() {
  if (!store.activeWorkspaceId) return
  const r = await fetch(`${apiBase()}/api/webhooks?workspaceId=${store.activeWorkspaceId}`, { credentials: 'include' })
  webhooksList.value = r.ok ? await r.json() : []
}
async function createWebhook() {
  const url = newWebhookUrl.value.trim()
  if (!url || !newWebhookEvents.value.length) return
  const r = await fetch(`${apiBase()}/api/webhooks`, {
    method: 'POST', credentials: 'include', headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ workspaceId: store.activeWorkspaceId, url, events: newWebhookEvents.value }),
  })
  if (r.ok) { revealedSecret.value = (await r.json()).secret; newWebhookUrl.value = ''; await loadWebhooks() }
}
async function toggleWebhook(wh: any) {
  await fetch(`${apiBase()}/api/webhooks/${wh.id}`, {
    method: 'PATCH', credentials: 'include', headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ workspaceId: store.activeWorkspaceId, active: !wh.active }),
  })
  await loadWebhooks()
}
async function deleteWebhook(id: string) {
  if (import.meta.client && !confirm('Delete this webhook?')) return
  await fetch(`${apiBase()}/api/webhooks/${id}?workspaceId=${store.activeWorkspaceId}`, { method: 'DELETE', credentials: 'include' })
  await loadWebhooks()
}
const deliveriesByHook = ref<Record<string, any[]>>({})
async function toggleDeliveries(id: string) {
  if (deliveriesByHook.value[id]) { const { [id]: _drop, ...rest } = deliveriesByHook.value; deliveriesByHook.value = rest; return }
  const r = await fetch(`${apiBase()}/api/webhooks/${id}/deliveries?workspaceId=${store.activeWorkspaceId}`, { credentials: 'include' })
  deliveriesByHook.value = { ...deliveriesByHook.value, [id]: r.ok ? await r.json() : [] }
}
watch(tab, (t) => { if (t === 'webhooks') { revealedSecret.value = ''; loadWebhooks() } })
// notifications prefs (стейт на странице — фильтрует Inbox)
const NOTIF_PREF_LIST = [
  { key: 'assigned', label: 'Assigned to me' },
  { key: 'state_changed', label: 'Status changes' },
  { key: 'commented', label: 'New comments' },
  { key: 'mention', label: 'Mentions' },
]
</script>

<template>
  <UModal v-model="open" :prevent-close="showAuto" :ui="{ ...modalUi, width: 'sm:max-w-3xl', rounded: 'rounded-xl' }">
    <div class="flex h-[34rem] overflow-hidden rounded-xl">
      <!-- сайдбар табов -->
      <div class="w-48 shrink-0 overflow-y-auto border-r border-subtle bg-base p-2">
        <template v-for="(tabs, grp) in settingsGroups" :key="grp">
          <div class="mb-1 mt-2 px-2 text-xs font-medium uppercase tracking-wider text-tertiary"><span class="text-ok"># </span>{{ grp }}</div>
          <div v-for="t in tabs" :key="t.key"
            class="flex h-8 cursor-pointer items-center gap-2 rounded px-2 text-sm hover:bg-hover"
            :class="tab === t.key ? 'bg-active text-white' : 'text-secondary'"
            @click="tab = t.key"><UIcon :name="t.icon" class="text-tertiary" /> {{ t.label }}</div>
        </template>
      </div>
      <!-- контент -->
      <div class="flex-1 overflow-y-auto p-5">
        <!-- GENERAL -->
        <template v-if="tab === 'general'">
          <div class="mb-4 flex items-center justify-between">
            <h2 class="text-base font-semibold text-white"><span class="text-ok">$</span> general</h2>
            <UButton icon="i-heroicons-x-mark" color="gray" variant="ghost" size="xs" square @click="open = false" />
          </div>
          <div class="space-y-4 text-sm">
            <div>
              <label class="mb-1 block text-xs text-tertiary">Logo</label>
              <div class="flex items-center gap-3">
                <Avatar :name="store.activeWorkspace?.name" :avatar-key="(store.activeWorkspace as any)?.logoKey" :size="48" />
                <UButton color="gray" variant="ghost" size="xs" @click="logoEl?.click()">[ upload ]</UButton>
                <UButton v-if="(store.activeWorkspace as any)?.logoKey" color="gray" variant="ghost" size="xs" @click="store.setLogo('')">[ remove ]</UButton>
                <input ref="logoEl" type="file" accept="image/*" class="hidden" @change="onLogoPick" />
              </div>
            </div>
            <div>
              <label class="mb-1 block text-xs text-tertiary">Workspace name</label>
              <div class="flex gap-2"><UInput v-model="wsNameDraft" class="flex-1" /><UButton :disabled="!wsNameDraft.trim() || wsNameDraft === store.activeWorkspace?.name" @click="saveWsName">[ save ]</UButton></div>
            </div>
            <div>
              <label class="mb-1 block text-xs text-tertiary">Slug</label>
              <UInput :model-value="store.activeWorkspace?.slug" disabled />
            </div>
            <div class="text-xs text-tertiary">ID: {{ store.activeWorkspaceId }}</div>
            <div v-if="store.isWsOwner" class="mt-6 rounded-lg border border-danger/30 p-3">
              <div class="mb-1 text-sm font-medium text-danger">Danger zone</div>
              <p class="mb-2 text-xs text-tertiary">Deleting the workspace hides it and all its data. This action is irreversible.</p>
              <UButton color="red" variant="ghost" @click="confirmDeleteWs">[ delete workspace ]</UButton>
            </div>
          </div>
        </template>

        <!-- MEMBERS -->
        <template v-else-if="tab === 'members'">
          <div class="mb-4 flex items-center justify-between">
            <h2 class="text-base font-semibold text-white"><span class="text-ok">$</span> members</h2>
            <UButton icon="i-heroicons-x-mark" color="gray" variant="ghost" size="xs" square @click="open = false" />
          </div>
          <div v-if="store.isWsAdmin" class="mb-4 flex gap-2">
            <UInput v-model="inviteEmail" placeholder="email to invite" class="flex-1" @keyup.enter="doInvite" />
            <USelectMenu v-model="inviteRole" :options="labelOpts(ROLES)" value-attribute="value" option-attribute="label" class="w-32" />
            <UButton :disabled="!inviteEmail.trim()" @click="doInvite">[ invite ]</UButton>
          </div>
          <div v-if="inviteLink" class="mb-4 flex items-center gap-2 rounded-md bg-elevated px-2 py-1.5 text-xs">
            <UIcon name="i-heroicons-link" class="shrink-0 text-tertiary" />
            <span class="flex-1 truncate font-mono text-tertiary">{{ inviteLink }}</span>
            <UButton size="2xs" color="gray" @click="copyInviteLink">[ copy ]</UButton>
          </div>
          <div v-for="m in store.workspaceMembers" :key="m.id" class="group flex items-center gap-2 rounded-md px-2 py-1.5 hover:bg-hover">
            <span class="flex h-7 w-7 items-center justify-center rounded-full bg-elevated text-xs">{{ (m.name || '?')[0].toUpperCase() }}</span>
            <span class="flex-1 truncate text-sm">{{ m.name }}<span v-if="m.userId === store.currentUserId" class="ml-1 text-xs text-tertiary">(you)</span></span>
            <USelectMenu v-if="store.isWsAdmin" :model-value="m.role" :options="labelOpts(ROLES)" value-attribute="value" option-attribute="label" size="xs" class="w-28" @update:model-value="(r) => store.updateMemberRole(m.userId, r)" />
            <span v-else class="w-28 text-right text-xs text-tertiary">{{ m.role }}</span>
            <UButton v-if="store.isWsAdmin && m.userId !== store.currentUserId" icon="i-heroicons-trash" color="red" variant="ghost" size="2xs" square class="opacity-0 group-hover:opacity-100" title="Remove" @click="store.removeMember(m.userId)" />
          </div>
          <template v-if="store.isWsAdmin && store.pendingInvites.length">
            <div class="mb-1 mt-3 text-xs font-medium uppercase tracking-wider text-tertiary"><span class="text-ok"># </span>Pending</div>
            <div v-for="iv in store.pendingInvites" :key="iv.id" class="group flex items-center gap-2 px-2 py-1.5 text-sm text-tertiary">
              <UIcon name="i-heroicons-envelope" /> <span class="flex-1 truncate">{{ iv.email }}</span><span class="text-xs">{{ iv.role }}</span>
              <UButton icon="i-heroicons-x-mark" color="red" variant="ghost" size="2xs" square class="opacity-0 group-hover:opacity-100" title="Revoke" @click="store.revokeInvitation(iv.id)" />
            </div>
          </template>
        </template>

        <!-- TEAMS -->
        <template v-else-if="tab === 'teams'">
          <div class="mb-4 flex items-center justify-between">
            <h2 class="text-base font-semibold text-white"><span class="text-ok">$</span> teams</h2>
            <div class="flex items-center gap-2">
              <UButton v-if="store.isWsAdmin" size="xs" @click="open = false; emit('new-team')">[ new team ]</UButton>
              <UButton icon="i-heroicons-x-mark" color="gray" variant="ghost" size="xs" square @click="open = false" />
            </div>
          </div>
          <div v-for="t in store.teamList" :key="t.id" class="rounded-md px-2 py-2 hover:bg-hover">
            <div class="group flex items-center gap-2">
              <UIcon :name="t.private ? 'i-heroicons-lock-closed' : 'i-heroicons-cube'" :class="t.private ? 'text-accent2' : 'text-tertiary'" />
              <UInput :model-value="t.name" variant="none" :padded="false" class="flex-1 font-medium" :disabled="!store.canManageTeam(t.id)" @change="(v:any) => store.updateTeam(t.id, { name: v })" />
              <span class="text-xs text-tertiary">{{ t.key }}</span>
              <UButton v-if="store.canManageTeam(t.id)" :icon="t.private ? 'i-heroicons-lock-closed' : 'i-heroicons-lock-open'" :color="t.private ? 'red' : 'gray'" variant="ghost" size="2xs" square :title="t.private ? 'Make public' : 'Make private'" @click="store.updateTeam(t.id, { private: !t.private })" />
              <USelectMenu v-if="store.canManageTeam(t.id)" :model-value="t.parentId || ''" size="2xs" class="w-32" value-attribute="value" option-attribute="label"
                :options="[{ label: '— root —', value: '' }, ...store.teamList.filter((x) => x.id !== t.id).map((x) => ({ label: x.key, value: x.id }))]"
                @update:model-value="(p) => store.moveTeam(t.id, p)" />
              <UButton icon="i-heroicons-adjustments-horizontal" color="gray" variant="ghost" size="2xs" square title="Statuses" @click="open = false; emit('open-states', t.id)" />
              <UButton v-if="store.canManageTeam(t.id)" icon="i-heroicons-trash" color="red" variant="ghost" size="2xs" square class="opacity-0 group-hover:opacity-100" title="Delete team" @click="emit('delete-team', t)" />
            </div>
            <!-- team membership (управляет lead/ws-admin) -->
            <div v-if="store.canManageTeam(t.id)" class="ml-6 mt-1.5 flex flex-wrap items-center gap-1.5">
              <span v-for="tm in store.teamMembersOf(t.id)" :key="tm.id" class="group/m flex items-center gap-1 rounded bg-elevated px-1.5 py-0.5 text-xs">
                <span :class="tm.role === 'lead' ? 'text-accent2' : 'text-secondary'">{{ store.users[tm.userId]?.name || store.users[tm.userId]?.email || '—' }}</span>
                <span v-if="tm.role === 'lead'" class="text-[10px] text-tertiary">lead</span>
                <UButton icon="i-heroicons-x-mark" color="gray" variant="ghost" size="2xs" square class="opacity-0 group-hover/m:opacity-100" title="Remove from team" @click="store.removeTeamMember(t.id, tm.userId)" />
              </span>
              <USelectMenu size="2xs" class="w-40" placeholder="+ add member" value-attribute="value" option-attribute="label"
                :options="store.workspaceMembers.filter((m) => !store.teamMembersOf(t.id).some((tm) => tm.userId === m.userId)).map((m) => ({ label: m.name, value: m.userId }))"
                @update:model-value="(uid:any) => uid && store.addTeamMember(t.id, uid)" />
            </div>
          </div>
        </template>

        <!-- LABELS -->
        <template v-else-if="tab === 'labels'">
          <div class="mb-4 flex items-center justify-between">
            <h2 class="text-base font-semibold text-white"><span class="text-ok">$</span> labels</h2>
            <UButton icon="i-heroicons-x-mark" color="gray" variant="ghost" size="xs" square @click="open = false" />
          </div>
          <div class="mb-4 flex items-center gap-2">
            <div class="flex gap-1">
              <button v-for="c in LABEL_COLORS" :key="c" class="h-5 w-5 rounded-full ring-offset-1 ring-offset-base" :class="newWsLabelColor === c ? 'ring-2 ring-white' : ''" :style="{ background: c }" @click="newWsLabelColor = c" />
            </div>
            <UInput v-model="newWsLabelName" placeholder="Label name" class="flex-1" @keyup.enter="addWsLabel" />
            <UButton :disabled="!newWsLabelName.trim()" @click="addWsLabel">[ add ]</UButton>
          </div>
          <div v-for="l in allLabels" :key="l.id" class="group flex items-center gap-2 rounded-md px-2 py-1.5 hover:bg-hover">
            <div class="flex gap-0.5">
              <button v-for="c in LABEL_COLORS" :key="c" class="h-3.5 w-3.5 rounded-full" :class="l.colorToken === c ? 'ring-1 ring-white ring-offset-1 ring-offset-base' : ''" :style="{ background: c }" @click="store.updateLabel(l.id, { colorToken: c })" />
            </div>
            <UInput :model-value="l.name" variant="none" :padded="false" class="flex-1 text-sm" @change="(v:any) => store.updateLabel(l.id, { name: v })" />
            <span class="text-xs text-tertiary">{{ store.teams[l.teamId]?.key || 'workspace' }}</span>
            <UButton icon="i-heroicons-trash" color="red" variant="ghost" size="2xs" square class="opacity-0 group-hover:opacity-100" @click="store.deleteLabel(l.id)" />
          </div>
          <div v-if="!allLabels.length" class="py-6 text-center text-sm text-tertiary">No labels yet</div>
        </template>

        <!-- AUTOMATIONS -->
        <template v-else-if="tab === 'automations'">
          <!-- редактор правила (inline, без вложенной модалки) -->
          <template v-if="showAuto">
            <div class="mb-4 flex items-center gap-2">
              <UButton icon="i-heroicons-arrow-left" color="gray" variant="ghost" size="xs" square @click="showAuto = false" />
              <h2 class="text-base font-semibold text-white"><span class="text-ok">$</span> {{ autoForm.id ? 'Edit rule' : 'New rule' }}</h2>
            </div>
            <div class="space-y-3">
              <UInput v-model="autoForm.name" placeholder="Rule name" autofocus />
              <div class="flex items-center gap-2">
                <span class="w-16 shrink-0 text-xs text-tertiary">Team</span>
                <USelectMenu v-model="autoForm.teamId" class="flex-1" :options="[{ value: '', label: 'All teams' }, ...store.teamList.map((t:any) => ({ value: t.id, label: t.name }))]" value-attribute="value" option-attribute="label" />
              </div>
              <div class="rounded-md border border-subtle p-2.5">
                <div class="mb-1.5 text-xs font-medium uppercase tracking-wider text-tertiary"><span class="text-ok"># </span>When</div>
                <div class="flex items-center gap-2">
                  <USelectMenu v-model="autoForm.event" class="flex-1" :options="AUTO_EVENTS" value-attribute="value" option-attribute="label" />
                  <USelectMenu v-if="autoForm.event === 'status_changed'" v-model="autoForm.statusTarget" class="flex-1" :options="statusTargetOpts" value-attribute="value" option-attribute="label" searchable />
                </div>
              </div>
              <div class="rounded-md border border-subtle p-2.5">
                <div class="mb-1.5 flex items-center">
                  <span class="text-xs font-medium uppercase tracking-wider text-tertiary"><span class="text-ok"># </span>Then</span>
                  <UButton class="ml-auto" color="gray" variant="ghost" size="2xs" @click="addAutoAction">[ add action ]</UButton>
                </div>
                <div v-for="(ac, i) in autoForm.actions" :key="i" class="mb-1.5 flex items-center gap-2">
                  <USelectMenu v-model="ac.type" class="w-36 shrink-0" :options="ACTION_TYPES" value-attribute="value" option-attribute="label" @update:model-value="onActionType(i)" />
                  <USelectMenu v-if="ac.type === 'set_status'" v-model="ac.stateId" class="flex-1" :options="stateOpts" value-attribute="value" option-attribute="label" searchable placeholder="state" />
                  <USelectMenu v-else-if="ac.type === 'set_assignee'" v-model="ac.userId" class="flex-1" :options="assigneeOpts" value-attribute="value" option-attribute="label" />
                  <USelectMenu v-else-if="ac.type === 'set_priority'" v-model="ac.priority" class="flex-1" :options="[0,1,2,3,4].map((p) => ({ value: p, label: PRIO[p] }))" value-attribute="value" option-attribute="label" />
                  <USelectMenu v-else-if="ac.type === 'add_label'" v-model="ac.labelId" class="flex-1" :options="labelOptsAuto" value-attribute="value" option-attribute="label" searchable placeholder="label" />
                  <template v-else-if="ac.type === 'post_message'">
                    <USelectMenu v-model="ac.channelId" class="w-32 shrink-0" :options="channelOptsAuto" value-attribute="value" option-attribute="label" placeholder="channel" />
                    <UInput v-model="ac.text" class="flex-1" placeholder="Message ({issue})" />
                  </template>
                  <UButton icon="i-heroicons-x-mark" color="gray" variant="ghost" size="2xs" square @click="autoForm.actions.splice(i, 1)" />
                </div>
                <div v-if="!autoForm.actions.length" class="py-2 text-center text-xs text-tertiary">No actions — add one.</div>
              </div>
              <div class="flex items-center gap-2 pt-1">
                <TermCheck v-model="autoForm.enabled" label="Enabled" />
                <UButton class="ml-auto" color="gray" variant="ghost" @click="showAuto = false">[ cancel ]</UButton>
                <UButton color="gray" @click="saveAuto">[ {{ autoForm.id ? 'save' : 'create' }} ]</UButton>
              </div>
            </div>
          </template>

          <!-- список правил -->
          <template v-else>
            <div class="mb-4 flex items-center justify-between">
              <h2 class="text-base font-semibold text-white"><span class="text-ok">$</span> automations</h2>
              <div class="flex items-center gap-2">
                <UButton v-if="store.isWsAdmin" color="gray" size="xs" @click="openAutoCreate">[ new rule ]</UButton>
                <UButton icon="i-heroicons-x-mark" color="gray" variant="ghost" size="xs" square @click="open = false" />
              </div>
            </div>
            <p class="mb-3 text-xs text-tertiary">When something happens to an issue, run actions automatically.</p>
            <div v-for="a in store.automationsList" :key="a.id" class="group mb-1.5 flex items-center gap-3 rounded-md border border-subtle px-3 py-2">
              <TermCheck :model-value="a.enabled" :disabled="!store.isWsAdmin" @update:model-value="(v:any) => store.isWsAdmin && store.toggleAutomation(a.id, v)" />
              <div class="min-w-0 flex-1" :class="store.isWsAdmin ? 'cursor-pointer' : ''" @click="store.isWsAdmin && openAutoEdit(a)">
                <div class="truncate text-sm font-medium" :class="a.enabled ? 'text-white' : 'text-tertiary'">{{ a.name }}</div>
                <div class="truncate text-xs text-tertiary">{{ autoSummary(a) }}<span v-if="a.teamId"> · {{ store.teams[a.teamId]?.key }}</span></div>
              </div>
              <UButton v-if="store.isWsAdmin" icon="i-heroicons-pencil" color="gray" variant="ghost" size="2xs" square @click="openAutoEdit(a)" />
              <UButton v-if="store.isWsAdmin" icon="i-heroicons-trash" color="red" variant="ghost" size="2xs" square @click="store.deleteAutomation(a.id)" />
            </div>
            <div v-if="!store.automationsList.length" class="py-6 text-center text-sm text-tertiary">No automations yet. Create your first rule.</div>
          </template>
        </template>

        <!-- TEMPLATES + recurring -->
        <template v-else-if="tab === 'templates'">
          <template v-if="showTmpl">
            <div class="mb-4 flex items-center gap-2">
              <UButton icon="i-heroicons-arrow-left" color="gray" variant="ghost" size="xs" square @click="showTmpl = false" />
              <h2 class="text-base font-semibold text-white"><span class="text-ok">$</span> {{ tmplForm.id ? 'Edit template' : 'New template' }}</h2>
            </div>
            <div class="space-y-3">
              <UInput v-model="tmplForm.name" placeholder="Template name (e.g. Weekly standup)" autofocus />
              <div class="flex items-center gap-2">
                <span class="w-16 shrink-0 text-xs text-tertiary">Team</span>
                <USelectMenu v-model="tmplForm.teamId" class="flex-1" :options="store.teamList.map((t:any) => ({ value: t.id, label: t.name }))" value-attribute="value" option-attribute="label" />
              </div>
              <div class="rounded-md border border-subtle p-2.5 space-y-2">
                <div class="text-xs font-medium uppercase tracking-wider text-tertiary"><span class="text-ok"># </span>Issue defaults</div>
                <UInput v-model="tmplForm.title" placeholder="Issue title (defaults to template name)" />
                <UTextarea v-model="tmplForm.descriptionMd" placeholder="Description (markdown)" :rows="3" />
                <div class="flex flex-wrap items-center gap-2">
                  <USelectMenu v-model="tmplForm.priority" class="w-32" :options="[0,1,2,3,4].map((p) => ({ value: p, label: PRIO[p] }))" value-attribute="value" option-attribute="label" />
                  <USelectMenu v-model="tmplForm.assigneeId" class="w-40" :options="tmplAssigneeOpts" value-attribute="value" option-attribute="label" placeholder="Assignee" />
                  <UInput v-model.number="tmplForm.estimate" type="number" min="0" step="0.5" class="w-24" placeholder="Est." />
                  <USelectMenu v-model="tmplForm.labelIds" class="w-48" :options="tmplLabelOpts" value-attribute="value" option-attribute="label" multiple searchable placeholder="Labels" />
                </div>
              </div>
              <div class="flex items-center gap-2">
                <span class="w-16 shrink-0 text-xs text-tertiary">Repeat</span>
                <USelectMenu v-model="tmplForm.recurrence" class="flex-1" :options="REC_OPTS" value-attribute="value" option-attribute="label" />
              </div>
              <p v-if="tmplForm.recurrence !== 'none'" class="text-xs text-tertiary">A new issue will be created automatically {{ tmplForm.recurrence }}.</p>
              <div class="flex items-center gap-2 pt-1">
                <UButton class="ml-auto" color="gray" variant="ghost" @click="showTmpl = false">[ cancel ]</UButton>
                <UButton color="gray" @click="saveTmpl">[ {{ tmplForm.id ? 'save' : 'create' }} ]</UButton>
              </div>
            </div>
          </template>

          <template v-else>
            <div class="mb-4 flex items-center justify-between">
              <h2 class="text-base font-semibold text-white"><span class="text-ok">$</span> templates</h2>
              <div class="flex items-center gap-2">
                <UButton color="gray" size="xs" @click="openTmplCreate">[ new template ]</UButton>
                <UButton icon="i-heroicons-x-mark" color="gray" variant="ghost" size="xs" square @click="open = false" />
              </div>
            </div>
            <p class="mb-3 text-xs text-tertiary">Reusable issue blueprints. Give one a repeat cadence and it spawns issues automatically.</p>
            <div v-for="t in store.templatesList" :key="t.id" class="group mb-1.5 flex items-center gap-3 rounded-md border border-subtle px-3 py-2">
              <div class="min-w-0 flex-1 cursor-pointer" @click="openTmplEdit(t)">
                <div class="truncate text-sm font-medium text-white">{{ t.name }}</div>
                <div class="truncate text-xs text-tertiary">
                  {{ store.teams[t.teamId]?.key }}
                  <span v-if="t.recurrence && t.recurrence !== 'none'" class="text-accent2"> · ↻ {{ t.recurrence }}</span>
                  <span v-if="t.nextRunAt && t.recurrence !== 'none'"> · next {{ dayShort(String(t.nextRunAt).slice(0,10)) }}</span>
                </div>
              </div>
              <UButton color="gray" variant="ghost" size="2xs" title="Create issue now" @click="spawnTmpl(t.id)">[ create issue ]</UButton>
              <UButton icon="i-heroicons-pencil" color="gray" variant="ghost" size="2xs" square @click="openTmplEdit(t)" />
              <UButton icon="i-heroicons-trash" color="red" variant="ghost" size="2xs" square @click="store.deleteTemplate(t.id)" />
            </div>
            <div v-if="!store.templatesList.length" class="py-6 text-center text-sm text-tertiary">No templates yet. Create your first one.</div>
          </template>
        </template>

        <!-- API TOKENS -->
        <template v-else-if="tab === 'apitokens'">
          <div class="mb-4 flex items-center justify-between">
            <h2 class="text-base font-semibold text-white"><span class="text-ok">$</span> API tokens</h2>
            <UButton icon="i-heroicons-x-mark" color="gray" variant="ghost" size="xs" square @click="open = false" />
          </div>
          <p class="mb-3 text-xs text-tertiary">Call the API as you: <code class="rounded bg-elevated px-1">Authorization: Bearer pms_…</code> on any RPC endpoint (POST <code class="rounded bg-elevated px-1">/pms.v1.&lt;Service&gt;/&lt;Method&gt;</code>). A token carries your access in this workspace.</p>
          <div class="mb-3 flex gap-2">
            <UInput v-model="newTokenName" placeholder="token name (e.g. ci-bot)" class="flex-1" @keyup.enter="createApiToken" />
            <USelectMenu v-model="newTokenExpiry" :options="[{ label: 'never', value: 0 }, { label: '30d', value: 30 }, { label: '90d', value: 90 }, { label: '1y', value: 365 }]" value-attribute="value" option-attribute="label" class="w-24" />
            <UButton :disabled="!newTokenName.trim()" @click="createApiToken">[ create ]</UButton>
          </div>
          <div class="mb-3"><TermCheck v-model="newTokenReadOnly" label="read-only (only read endpoints)" /></div>
          <div v-if="revealedToken" class="mb-3 rounded-md border border-ok/40 bg-elevated px-3 py-2">
            <div class="mb-1 text-xs text-ok">Copy now — shown once:</div>
            <div class="flex items-center gap-2">
              <code class="flex-1 truncate font-mono text-xs text-white">{{ revealedToken }}</code>
              <UButton size="2xs" color="gray" @click="copyText(revealedToken)">[ copy ]</UButton>
            </div>
          </div>
          <div v-for="t in apiTokens" :key="t.id" class="group flex items-center gap-2 rounded-md px-2 py-1.5 hover:bg-hover">
            <UIcon name="i-heroicons-key" class="shrink-0 text-tertiary" />
            <span class="flex-1 truncate text-sm">{{ t.name }} <span class="ml-1 font-mono text-xs text-tertiary">{{ t.prefix }}…</span></span>
            <span v-if="t.readOnly" class="shrink-0 rounded bg-elevated px-1 text-[10px] text-tertiary">read-only</span>
            <span class="shrink-0 text-xs text-tertiary">{{ t.lastUsedAt ? 'used ' + d10(t.lastUsedAt) : 'never used' }}</span>
            <UButton icon="i-heroicons-trash" color="red" variant="ghost" size="2xs" square class="opacity-0 group-hover:opacity-100" title="Revoke" @click="revokeApiToken(t.id)" />
          </div>
          <div v-if="!apiTokens.length" class="py-6 text-center text-sm text-tertiary">// no API tokens yet</div>
        </template>

        <!-- WEBHOOKS -->
        <template v-else-if="tab === 'webhooks'">
          <div class="mb-4 flex items-center justify-between">
            <h2 class="text-base font-semibold text-white"><span class="text-ok">$</span> webhooks</h2>
            <UButton icon="i-heroicons-x-mark" color="gray" variant="ghost" size="xs" square @click="open = false" />
          </div>
          <p class="mb-3 text-xs text-tertiary">POST JSON to your URL on events, signed with HMAC-SHA256 (<code class="rounded bg-elevated px-1">X-PMS-Signature: sha256=…</code>) using the secret shown once below.</p>
          <div class="mb-2 flex gap-2">
            <UInput v-model="newWebhookUrl" placeholder="https://your-host/hook" class="flex-1" @keyup.enter="createWebhook" />
            <UButton :disabled="!newWebhookUrl.trim() || !newWebhookEvents.length" @click="createWebhook">[ add ]</UButton>
          </div>
          <USelectMenu v-model="newWebhookEvents" :options="EVENT_OPTIONS" multiple placeholder="events" class="mb-3 w-full" />
          <div v-if="revealedSecret" class="mb-3 rounded-md border border-ok/40 bg-elevated px-3 py-2">
            <div class="mb-1 text-xs text-ok">Signing secret — shown once:</div>
            <div class="flex items-center gap-2">
              <code class="flex-1 truncate font-mono text-xs text-white">{{ revealedSecret }}</code>
              <UButton size="2xs" color="gray" @click="copyText(revealedSecret)">[ copy ]</UButton>
            </div>
          </div>
          <div v-for="wh in webhooksList" :key="wh.id" class="rounded-md px-2 py-1.5 hover:bg-hover">
            <div class="group flex items-center gap-2">
              <span class="h-2 w-2 shrink-0 rounded-full" :class="wh.active ? 'bg-ok' : 'bg-tertiary'" />
              <span class="min-w-0 flex-1 truncate text-sm">{{ wh.url }}<span class="ml-1 text-xs text-tertiary">{{ (wh.events || []).join(', ') }}</span></span>
              <UButton size="2xs" color="gray" variant="ghost" @click="toggleDeliveries(wh.id)">{{ deliveriesByHook[wh.id] ? 'hide' : 'log' }}</UButton>
              <UButton size="2xs" color="gray" variant="ghost" @click="toggleWebhook(wh)">{{ wh.active ? 'disable' : 'enable' }}</UButton>
              <UButton icon="i-heroicons-trash" color="red" variant="ghost" size="2xs" square class="opacity-0 group-hover:opacity-100" title="Delete" @click="deleteWebhook(wh.id)" />
            </div>
            <div v-if="deliveriesByHook[wh.id]" class="ml-4 mt-1 space-y-0.5">
              <div v-for="dlv in deliveriesByHook[wh.id]" :key="dlv.id" class="flex items-center gap-2 text-xs">
                <span class="w-14 shrink-0" :class="dlv.status === 'success' ? 'text-ok' : dlv.status === 'failed' ? 'text-danger' : 'text-tertiary'">{{ dlv.status }}</span>
                <span class="shrink-0 text-tertiary">{{ dlv.event }}</span>
                <span class="shrink-0 text-tertiary">{{ dlv.responseCode || '—' }}</span>
                <span class="flex-1 truncate text-right text-tertiary">{{ d10(dlv.createdAt) }} · {{ dlv.attempts }}×</span>
              </div>
              <div v-if="!deliveriesByHook[wh.id].length" class="text-xs text-tertiary">// no deliveries yet</div>
            </div>
          </div>
          <div v-if="!webhooksList.length" class="py-6 text-center text-sm text-tertiary">// no webhooks yet</div>
        </template>

        <!-- ACCOUNT -->
        <template v-else-if="tab === 'account'">
          <div class="mb-4 flex items-center justify-between">
            <h2 class="text-base font-semibold text-white"><span class="text-ok">$</span> account</h2>
            <UButton icon="i-heroicons-x-mark" color="gray" variant="ghost" size="xs" square @click="open = false" />
          </div>
          <div class="mb-4 flex items-center gap-3">
            <Avatar :name="store.userName(store.currentUserId)" :avatar-key="store.users[store.currentUserId]?.avatarKey" :size="48" />
            <div class="flex-1">
              <div class="font-medium">{{ store.users[store.currentUserId]?.name || '—' }}</div>
              <div class="text-xs text-tertiary">{{ store.users[store.currentUserId]?.email }}</div>
            </div>
            <UButton color="gray" variant="ghost" size="xs" @click="avatarEl?.click()">[ upload ]</UButton>
            <UButton v-if="store.users[store.currentUserId]?.avatarKey" color="gray" variant="ghost" size="xs" @click="store.setAvatar('')">[ remove ]</UButton>
            <input ref="avatarEl" type="file" accept="image/*" class="hidden" @change="onAvatarPick" />
          </div>
          <UButton color="red" variant="ghost" @click="emit('logout')">[ log out ]</UButton>
        </template>

        <!-- APPEARANCE -->
        <template v-else-if="tab === 'appearance'">
          <div class="mb-4 flex items-center justify-between">
            <h2 class="text-base font-semibold text-white"><span class="text-ok">$</span> appearance</h2>
            <UButton icon="i-heroicons-x-mark" color="gray" variant="ghost" size="xs" square @click="open = false" />
          </div>
          <div class="mb-1 text-xs text-tertiary">Density</div>
          <div class="flex gap-2">
            <button v-for="d in ['compact', 'default', 'comfortable']" :key="d"
              class="rounded-md border px-3 py-2 text-sm capitalize"
              :class="density === d ? 'border-accent bg-active text-white' : 'border-subtle bg-elevated text-tertiary hover:bg-hover'"
              @click="applyDensity(d)">{{ d }}</button>
          </div>
          <p class="mt-2 text-xs text-tertiary">Scales the whole interface.</p>
        </template>

        <!-- NOTIFICATIONS -->
        <template v-else-if="tab === 'notifications'">
          <div class="mb-4 flex items-center justify-between">
            <h2 class="text-base font-semibold text-white"><span class="text-ok">$</span> notifications</h2>
            <UButton icon="i-heroicons-x-mark" color="gray" variant="ghost" size="xs" square @click="open = false" />
          </div>
          <p class="mb-3 text-xs text-tertiary">Which notifications to show in the Inbox.</p>
          <div v-for="p in NOTIF_PREF_LIST" :key="p.key" class="flex items-center gap-2 rounded-md px-2 py-2 hover:bg-hover">
            <TermCheck :model-value="notifPrefs[p.key] !== false" @update:model-value="emit('toggle-pref', p.key)" />
            <span class="text-sm">{{ p.label }}</span>
          </div>
        </template>

        <!-- SECTIONS (sidebar) -->
        <template v-else-if="tab === 'sections'">
          <div class="mb-4 flex items-center justify-between">
            <h2 class="text-base font-semibold text-white"><span class="text-ok">$</span> sidebar sections</h2>
            <UButton icon="i-heroicons-x-mark" color="gray" variant="ghost" size="xs" square @click="open = false" />
          </div>
          <p class="mb-3 text-xs text-tertiary">Toggle rail sections and set their order.</p>
          <div v-for="(s, i) in cfgList" :key="s.key" class="flex items-center gap-2 rounded-md px-2 py-1.5 hover:bg-hover">
            <TermCheck :model-value="s.visible" @update:model-value="emit('toggle-section', i)" />
            <UIcon :name="s.icon" class="text-tertiary" />
            <span class="flex-1 text-sm">{{ s.label }}</span>
            <UButton icon="i-heroicons-chevron-up" color="gray" variant="ghost" size="2xs" square :disabled="i === 0" @click="emit('move-section', i, -1)" />
            <UButton icon="i-heroicons-chevron-down" color="gray" variant="ghost" size="2xs" square :disabled="i === cfgList.length - 1" @click="emit('move-section', i, 1)" />
          </div>
        </template>
      </div>
    </div>
  </UModal>
</template>
