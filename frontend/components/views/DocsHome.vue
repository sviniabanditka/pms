<script setup lang="ts">
import { useAppStore } from '~/stores/app'

// Домашняя страница доков команды (teamSection=docs): грид последних доков.
const props = defineProps<{ teamId: string }>()
const emit = defineEmits<{
  (e: 'open-team'): void
  (e: 'open-doc', id: string): void
  (e: 'create-doc'): void
}>()

const store = useAppStore()
const team = computed(() => store.teams[props.teamId])
</script>

<template>
  <div class="sticky top-0 z-10 flex items-center gap-1.5 border-b border-subtle bg-surface px-4 py-3 text-sm">
    <span class="text-tertiary">{{ store.activeWorkspace?.name }}</span>
    <span class="text-tertiary">/</span>
    <button class="text-tertiary hover:text-secondary" @click="emit('open-team')">{{ team?.name }}</button>
    <span class="text-tertiary">/</span>
    <UIcon name="i-heroicons-document-text" class="text-tertiary" />
    <span class="font-medium">docs</span>
    <span class="text-tertiary">{{ store.teamDocs(teamId).length }}</span>
    <UButton v-if="store.canWrite" class="ml-auto" size="xs" color="gray" variant="ghost" @click="emit('create-doc')">[ new doc ]</UButton>
  </div>
  <div class="p-4">
    <div class="mb-2 text-xs font-medium uppercase tracking-wider text-tertiary"><span class="text-ok"># </span>Recent</div>
    <div class="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-3">
      <div v-for="d in store.teamDocs(teamId)" :key="d.id"
        class="cursor-pointer rounded-lg border border-subtle bg-elevated p-4 hover:bg-hover"
        @click="emit('open-doc', d.id)">
        <div class="flex items-center gap-2"><UIcon name="i-heroicons-document" class="text-tertiary" /><span class="truncate font-medium">{{ d.title }}</span></div>
        <div class="mt-1 line-clamp-2 text-xs text-tertiary">{{ d.body?.md || 'Empty document' }}</div>
      </div>
      <div v-if="store.canWrite && !store.teamDocs(teamId).length"
        class="flex h-28 cursor-pointer items-center justify-center rounded-lg border border-dashed border-subtle text-sm text-tertiary hover:bg-hover"
        @click="emit('create-doc')">+ Create first doc</div>
    </div>
  </div>
</template>
