<script setup lang="ts">
import { useAppStore } from '~/stores/app'

// Страница сохранённых представлений команды (teamSection=views): список + rename/delete.
// Применение view (openView) меняет роут-стейт страницы → эмит.
const props = defineProps<{ teamId: string }>()
const emit = defineEmits<{
  (e: 'open-team'): void
  (e: 'open-view', id: string): void
}>()

const store = useAppStore()
const { wrap } = useActionWrap()
const team = computed(() => store.teams[props.teamId])

async function renameView(v: any) {
  const name = prompt('View name', v.name)
  if (name && name !== v.name) await wrap(() => store.renameView(v.id, name))
}
async function deleteView(v: any) {
  if (confirm(`Delete view "${v.name}"?`)) await wrap(() => store.deleteView(v.id))
}
</script>

<template>
  <div class="sticky top-0 z-10 flex items-center gap-1.5 border-b border-subtle bg-surface px-4 py-3 text-sm">
    <span class="text-tertiary">{{ store.activeWorkspace?.name }}</span>
    <span class="text-tertiary">/</span>
    <button class="text-tertiary hover:text-secondary" @click="emit('open-team')">{{ team?.name }}</button>
    <span class="text-tertiary">/</span>
    <UIcon name="i-heroicons-square-3-stack-3d" class="text-tertiary" />
    <span class="font-medium">views</span>
    <span class="text-tertiary">{{ store.teamViews(teamId).length }}</span>
  </div>
  <div class="p-4">
    <div class="divide-y divide-subtle/40 overflow-hidden rounded-lg border border-subtle">
      <div v-for="v in store.teamViews(teamId)" :key="v.id"
        class="group flex cursor-pointer items-center gap-2 bg-elevated px-4 py-2.5 hover:bg-hover"
        @click="emit('open-view', v.id)">
        <UIcon name="i-heroicons-star" class="text-tertiary" />
        <span class="flex-1 truncate">{{ v.name }}</span>
        <button class="hidden text-xs text-tertiary hover:text-secondary group-hover:inline" title="rename"
          @click.stop="renameView(v)">✎</button>
        <button class="hidden text-xs text-tertiary hover:text-red-400 group-hover:inline" title="delete"
          @click.stop="deleteView(v)">×</button>
        <UIcon name="i-heroicons-chevron-right" class="text-tertiary" />
      </div>
      <div v-if="!store.teamViews(teamId).length" class="bg-elevated px-4 py-8 text-center text-sm text-tertiary">No views — save one from a filtered Issues list</div>
    </div>
  </div>
</template>
