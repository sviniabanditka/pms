<script setup lang="ts">
import { useAppStore } from '~/stores/app'

// Страница циклов команды (teamSection=cycles): грид карточек. Навигация — эмиты.
const props = defineProps<{ teamId: string }>()
const emit = defineEmits<{
  (e: 'open-team'): void
  (e: 'open-cycle', c: any): void
  (e: 'create-cycle'): void
}>()

const store = useAppStore()
const team = computed(() => store.teams[props.teamId])
</script>

<template>
  <div class="sticky top-0 z-10 flex items-center gap-1.5 border-b border-subtle bg-surface px-4 py-3 text-sm">
    <span class="text-tertiary">{{ store.activeWorkspace?.name }}</span>
    <span class="text-tertiary">/</span>
    <button class="text-tertiary hover:text-secondary" @click="emit('open-team')">{{ team?.name }}</button>
    <span class="text-tertiary">/</span>          <span class="font-medium">cycles</span>
    <span class="text-tertiary">{{ store.teamCycles(teamId).length }}</span>
    <UButton v-if="store.canManageTeam(teamId)" class="ml-auto" size="xs" color="gray" variant="ghost" @click="emit('create-cycle')">[ new cycle ]</UButton>
  </div>
  <div class="grid grid-cols-1 gap-3 p-4 sm:grid-cols-2 xl:grid-cols-3">
    <div v-for="c in store.teamCycles(teamId)" :key="c.id"
      class="cursor-pointer rounded-lg border border-subtle bg-elevated p-4 hover:bg-hover"
      @click="emit('open-cycle', c)">
      <div class="mb-1 flex items-center gap-2">
        <UIcon name="i-heroicons-play-circle" :class="c.status === 'active' ? 'text-accent' : 'text-tertiary'" />
        <span class="font-medium">Cycle {{ c.number }}</span>
        <span class="ml-auto rounded bg-surface px-1.5 text-xs text-tertiary">{{ c.status }}</span>
      </div>
      <div class="text-xs text-tertiary">{{ String(c.startsOn).slice(0,10) }} → {{ String(c.endsOn).slice(0,10) }}</div>
      <div class="mt-2 text-xs text-tertiary">{{ store.cycleStats(c.id).completed }}/{{ store.cycleStats(c.id).scope }} done</div>
    </div>
    <div v-if="store.canManageTeam(teamId) && !store.teamCycles(teamId).length"
      class="flex h-28 cursor-pointer items-center justify-center rounded-lg border border-dashed border-subtle text-sm text-tertiary hover:bg-hover"
      @click="emit('create-cycle')">+ Create first cycle</div>
  </div>
</template>
