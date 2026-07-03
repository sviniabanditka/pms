<script setup lang="ts">
import { useAppStore } from '~/stores/app'

// Строка связи issue (blocks / blocked by / related / duplicate). Молекула из
// IssueDetailPanel: relation пропсом, target-issue и права — из стора.
const props = defineProps<{ relation: { id: string; label: string; otherId: string } }>()
const emit = defineEmits<{ (e: 'open', id: string): void }>()
const store = useAppStore()
</script>

<template>
  <div class="group flex items-center gap-2 rounded-md px-2 py-1.5 hover:bg-hover">
    <span class="text-tertiary">{{ relation.label }}</span>
    <span class="cursor-pointer hover:text-accent" @click="emit('open', relation.otherId)">{{ store.issueById(relation.otherId)?.identifier || '—' }}</span>
    <UButton v-if="store.canWrite" class="ml-auto opacity-0 group-hover:opacity-100" icon="i-heroicons-x-mark" color="gray" variant="ghost" size="2xs" square @click="store.removeRelation(relation.id)" aria-label="Close" />
  </div>
</template>
