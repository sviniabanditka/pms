<script setup lang="ts">
import { useAppStore } from '~/stores/app'
import { PRIO, labelOpts, modalUi } from '~/lib/ui'

const props = defineProps<{ dashboardId: string }>()
const open = defineModel<boolean>()

const store = useAppStore()
const { wrap } = useActionWrap()

// add-widget form
const wType = ref('note')
const wTitle = ref('')
const wNote = ref('')
const wStatPrio = ref('')
const wViewId = ref('')
const addWidget = () => wrap(async () => {
  let config: any = {}
  if (wType.value === 'note') config = { text: wNote.value }
  else if (wType.value === 'stat') config = { priority: wStatPrio.value }
  else if (wType.value === 'view') config = { viewId: wViewId.value }
  await store.addWidget(props.dashboardId, wType.value, wTitle.value, config)
  wTitle.value = ''; wNote.value = ''
})
</script>

<template>
  <UModal v-model="open" :ui="modalUi">
    <div class="space-y-3 p-4">
      <span class="text-sm font-semibold text-white"><span class="text-ok">$</span> add widget</span>
      <USelectMenu v-model="wType" :options="labelOpts(['note', 'stat', 'view', 'chart'])" value-attribute="value" option-attribute="label" />
      <UInput v-model="wTitle" placeholder="Title (optional)" />
      <UTextarea v-if="wType==='note'" v-model="wNote" placeholder="Text…" :rows="3" />
      <USelectMenu v-else-if="wType==='stat'" v-model="wStatPrio" value-attribute="value" option-attribute="label"
        :options="[{ label: 'all issues', value: '' }, ...PRIO.map((p, idx) => ({ label: `priority: ${p}`, value: String(idx) }))]" />
      <USelectMenu v-else-if="wType==='view'" v-model="wViewId" value-attribute="value" option-attribute="label"
        :options="[{ label: 'pick a view…', value: '' }, ...Object.values(store.views).filter((x) => !x.deletedAt).map((v) => ({ label: v.name, value: v.id }))]" />
      <div class="flex justify-end gap-2 pt-1">
        <UButton color="gray" variant="ghost" @click="open = false">[ cancel ]</UButton>
        <UButton @click="addWidget(); open = false">[ add ]</UButton>
      </div>
    </div>
  </UModal>
</template>
