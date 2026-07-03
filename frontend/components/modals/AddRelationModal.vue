<script setup lang="ts">
import { useAppStore } from '~/stores/app'
import { labelOpts, modalUi, REL_TYPES } from '~/lib/ui'

const props = defineProps<{ issue: any }>()
const open = defineModel<boolean>()

const store = useAppStore()
const { wrap } = useActionWrap()

const relType = ref('blocks')
const relTarget = ref('')
const addRel = () => wrap(async () => { if (relTarget.value.trim()) { await store.addRelation(props.issue.id, relTarget.value, relType.value); relTarget.value = ''; open.value = false } })
</script>

<template>
  <UModal v-model="open" :ui="modalUi">
    <div v-if="issue" class="space-y-4 p-4">
      <div class="flex items-center gap-2">
        <span class="text-base font-semibold text-white"><span class="text-ok">$</span> add relation</span>
        <UButton class="ml-auto" icon="i-heroicons-x-mark" color="gray" variant="ghost" size="xs" square @click="open = false" aria-label="Close" />
      </div>
      <div>
        <label class="mb-1 block text-xs text-tertiary">Type</label>
        <USelectMenu v-model="relType" :options="labelOpts(REL_TYPES)" value-attribute="value" option-attribute="label" class="w-full" />
      </div>
      <div>
        <label class="mb-1 block text-xs text-tertiary">Target issue</label>
        <USelectMenu v-model="relTarget" searchable searchable-placeholder="Search issue…" placeholder="Select issue" value-attribute="value" option-attribute="label"
          :options="store.issuesForTeamAll(issue.teamId).filter((i) => i.id !== issue.id).map((i) => ({ label: `${i.identifier} · ${i.title}`, value: i.identifier }))" class="w-full" />
      </div>
      <div class="flex justify-end gap-2 pt-1">
        <UButton color="gray" variant="ghost" @click="open = false">[ cancel ]</UButton>
        <UButton :disabled="!relTarget" @click="addRel">[ add ]</UButton>
      </div>
    </div>
  </UModal>
</template>
