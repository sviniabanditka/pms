<script setup lang="ts">
import { useAppStore } from '~/stores/app'
import { labelOpts } from '~/lib/ui'

// Страница вех команды (teamSection=milestones): список + детали выбранной вехи.
// Выбранная веха (ms-id) — на странице (нужна pageTitle/сбросам) → проп + эмиты.
const props = defineProps<{ teamId: string; msId: string }>()
const emit = defineEmits<{
  (e: 'open-team'): void
  (e: 'open-issue', id: string): void
  (e: 'open-ms', id: string): void
  (e: 'close-ms'): void
  (e: 'create-milestone'): void
}>()

const store = useAppStore()
const team = computed(() => store.teams[props.teamId])
const selectedMs = computed(() => store.milestones[props.msId])
const todayISO = new Date().toISOString().slice(0, 10)
function msOverdue(m: any) { return m.targetDate && String(m.targetDate).slice(0, 10) < todayISO && m.status !== 'done' }
const { statusColor, statusGlyph } = useColors()
</script>

<template>
  <!-- детали выбранной вехи -->
  <template v-if="selectedMs">
    <div class="sticky top-0 z-10 border-b border-subtle bg-surface px-4 py-3">
      <div class="flex items-center gap-1.5 text-sm">
        <span class="text-tertiary">{{ store.activeWorkspace?.name }}</span>
        <span class="text-tertiary">/</span>
        <button class="text-tertiary hover:text-secondary" @click="emit('open-team')">{{ team?.name }}</button>
        <span class="text-tertiary">/</span>
        <button class="text-tertiary hover:text-secondary" @click="emit('close-ms')">Milestones</button>
        <span class="text-tertiary">/</span>
        <UIcon name="i-heroicons-flag" :class="selectedMs.status === 'done' ? 'text-accent' : selectedMs.status === 'active' ? 'text-warn' : 'text-tertiary'" />
        <span class="font-medium">{{ selectedMs.name }}</span>
        <USelectMenu :model-value="selectedMs.status" :options="labelOpts(['planned', 'active', 'done'])" value-attribute="value" option-attribute="label" size="xs" class="w-28" @update:model-value="(st) => store.updateMilestone(selectedMs.id, { status: st })" />
        <span v-if="selectedMs.targetDate" class="text-xs" :class="msOverdue(selectedMs) ? 'text-danger' : 'text-tertiary'">{{ String(selectedMs.targetDate).slice(0,10) }}{{ msOverdue(selectedMs) ? ' · overdue' : '' }}</span>
        <span class="ml-auto text-xs text-tertiary">{{ store.milestoneProgress(selectedMs.id).done }}/{{ store.milestoneProgress(selectedMs.id).total }}</span>
      </div>
      <TermBar class="mt-2 block" :width="20" :done="store.milestoneProgress(selectedMs.id).done" :total="store.milestoneProgress(selectedMs.id).total" />
    </div>
    <div v-for="i in store.milestoneIssues(selectedMs.id)" :key="i.id"
      class="flex cursor-pointer items-center gap-3 border-b border-subtle/40 px-4 py-2.5 hover:bg-hover" @click="emit('open-issue', i.id)">
      <span class="inline-block shrink-0 w-4 text-center leading-none" :style="{ color: statusColor(store.states[i.stateId]?.category) }">{{ statusGlyph(store.states[i.stateId]?.category) }}</span>
      <span class="min-w-14 shrink-0 whitespace-nowrap text-sm text-accent2">{{ i.identifier }}</span>
      <span class="flex-1 truncate">{{ i.title }}</span>
      <span v-if="i.assigneeId" class="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-elevated text-[10px]">{{ (store.userName(i.assigneeId) || '?')[0].toUpperCase() }}</span>
    </div>
    <div v-if="!store.milestoneIssues(selectedMs.id).length" class="py-10 text-center text-sm text-tertiary">// no issues — assign via Milestone field</div>
  </template>

  <!-- список вех -->
  <template v-else>
    <div class="sticky top-0 z-10 flex items-center gap-1.5 border-b border-subtle bg-surface px-4 py-3 text-sm">
      <span class="text-tertiary">{{ store.activeWorkspace?.name }}</span>
      <span class="text-tertiary">/</span>
      <button class="text-tertiary hover:text-secondary" @click="emit('open-team')">{{ team?.name }}</button>
      <span class="text-tertiary">/</span>            <span class="font-medium">milestones</span>
      <span class="text-tertiary">{{ store.teamMilestones(teamId).length }}</span>
      <UButton v-if="store.canManageTeam(teamId)" class="ml-auto" size="xs" color="gray" variant="ghost" @click="emit('create-milestone')">[ new milestone ]</UButton>
    </div>
    <div class="p-4">
      <div class="space-y-2">
        <div v-for="m in store.teamMilestones(teamId)" :key="m.id"
          class="group cursor-pointer rounded-lg border border-subtle bg-elevated p-3 hover:border-strong" @click="emit('open-ms', m.id)">
          <div class="flex items-center gap-2">
            <UIcon name="i-heroicons-flag" :class="m.status === 'done' ? 'text-accent' : m.status === 'active' ? 'text-warn' : 'text-tertiary'" />
            <span class="flex-1 truncate font-medium">{{ m.name }}</span>
            <span class="rounded bg-surface px-1.5 text-xs text-tertiary">{{ m.status }}</span>
            <span v-if="m.targetDate" class="text-xs" :class="msOverdue(m) ? 'text-danger' : 'text-tertiary'">{{ String(m.targetDate).slice(5,10) }}</span>
            <UButton icon="i-heroicons-trash" color="gray" variant="ghost" size="2xs" square class="opacity-0 group-hover:opacity-100" @click.stop="store.deleteMilestone(m.id)" />
          </div>
          <div class="mt-2 flex items-center gap-2 text-xs text-tertiary">
            <TermBar :width="14" :done="store.milestoneProgress(m.id).done" :total="store.milestoneProgress(m.id).total" />
            <span>{{ store.milestoneProgress(m.id).done }}/{{ store.milestoneProgress(m.id).total }}</span>
          </div>
        </div>
        <div v-if="!store.teamMilestones(teamId).length" class="py-10 text-center text-sm text-tertiary">// no milestones</div>
      </div>
    </div>
  </template>
</template>
