<script setup lang="ts">
import { useAppStore } from '~/stores/app'
import { modalUi } from '~/lib/ui'

const props = defineProps<{ issue: any }>()
const open = defineModel<boolean>()

const store = useAppStore()
const { wrap } = useActionWrap()

const newLabelName = ref('')
const createLabelInIssue = () => wrap(async () => {
  if (!newLabelName.value.trim() || !props.issue) return
  const id = await store.createLabel(props.issue.teamId, newLabelName.value)
  newLabelName.value = ''
  await store.addLabel(props.issue.id, id)
})
const addLabelFromModal = (lid: any) => lid && wrap(async () => { await store.addLabel(props.issue.id, lid) })
</script>

<template>
  <UModal v-model="open" :ui="modalUi">
    <div v-if="issue" class="space-y-4 p-4">
      <div class="flex items-center gap-2">
        <span class="text-base font-semibold text-white"><span class="text-ok">$</span> add label</span>
        <UButton class="ml-auto" icon="i-heroicons-x-mark" color="gray" variant="ghost" size="xs" square @click="open = false" />
      </div>
      <div>
        <div class="mb-1.5 text-xs font-medium uppercase tracking-wider text-tertiary"><span class="text-ok"># </span>Existing</div>
        <div class="flex flex-wrap gap-1.5">
          <button v-for="l in store.teamLabels(issue.teamId).filter((l) => !store.labelsForIssue(issue.id).some((x) => x.id === l.id))" :key="l.id"
            class="flex items-center gap-1 rounded-full border border-subtle px-2.5 py-1 text-sm hover:border-strong hover:bg-hover"
            @click="addLabelFromModal(l.id); open = false">
            <span :style="{ color: l.colorToken }">#{{ l.name }}</span>
          </button>
          <span v-if="!store.teamLabels(issue.teamId).filter((l) => !store.labelsForIssue(issue.id).some((x) => x.id === l.id)).length" class="text-sm text-tertiary">All labels added</span>
        </div>
      </div>
      <div class="border-t border-subtle pt-3">
        <div class="mb-1.5 text-xs font-medium uppercase tracking-wider text-tertiary"><span class="text-ok"># </span>Create new</div>
        <div class="flex gap-2">
          <UInput v-model="newLabelName" size="sm" placeholder="Label name" class="flex-1" @keyup.enter="createLabelInIssue(); open = false" />
          <UButton @click="createLabelInIssue(); open = false">[ create ]</UButton>
        </div>
      </div>
    </div>
  </UModal>
</template>
