<script setup lang="ts">
import { useAppStore } from '~/stores/app'
import { labelOpts, modalUi, STATE_CATS } from '~/lib/ui'

const props = defineProps<{ teamId: string; teamKey?: string }>()
const open = defineModel<boolean>()

const store = useAppStore()
const { err, wrap } = useActionWrap()
const { statusColor, statusGlyph } = useColors()

const newStateName = ref('')
const newStateCat = ref('unstarted')
const addState = () => wrap(async () => { if (newStateName.value.trim()) { await store.createState(props.teamId, newStateName.value, newStateCat.value); newStateName.value = '' } })
const delState = (id: string) => wrap(() => store.deleteState(id))
</script>

<template>
  <UModal v-model="open" :ui="modalUi">
    <div class="space-y-3 p-4">
      <div class="flex items-center gap-2">
        <span class="text-base font-semibold text-white"><span class="text-ok">$</span> workflow statuses · {{ teamKey }}</span>
        <UButton class="ml-auto" icon="i-heroicons-x-mark" color="gray" variant="ghost" size="xs" square @click="open = false" />
      </div>
      <div v-for="st in store.teamStates(teamId)" :key="st.id" class="flex items-center gap-2">
        <span class="inline-block shrink-0 w-4 text-center leading-none" :style="{ color: statusColor(st.category) }">{{ statusGlyph(st.category) }}</span>
        <UInput :model-value="st.name" size="sm" class="flex-1" @change="(v:any) => store.updateState(st.id, { name: v })" />
        <USelectMenu :model-value="st.category" :options="labelOpts(STATE_CATS)" value-attribute="value" option-attribute="label" size="sm" class="w-36" @update:model-value="(c) => store.updateState(st.id, { category: c })" />
        <UButton icon="i-heroicons-trash" color="gray" variant="ghost" size="2xs" square title="Delete (must be empty)" @click="delState(st.id)" />
      </div>
      <div class="flex items-center gap-2 border-t border-subtle pt-3">
        <UInput v-model="newStateName" placeholder="New status name" size="sm" class="flex-1" @keyup.enter="addState" />
        <USelectMenu v-model="newStateCat" :options="labelOpts(STATE_CATS)" value-attribute="value" option-attribute="label" size="sm" class="w-36" />
        <UButton icon="i-heroicons-plus" size="xs" square @click="addState" />
      </div>
      <p v-if="err" class="text-xs text-danger">{{ err }}</p>
    </div>
  </UModal>
</template>
