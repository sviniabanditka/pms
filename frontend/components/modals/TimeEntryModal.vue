<script setup lang="ts">
import { modalUi } from '~/lib/ui'

// teForm живёт на странице (его заполняет openTimeEntry) — сюда приходит по ссылке
defineProps<{ form: { issueId: string; date: string; start: string; end: string; description: string } }>()
const emit = defineEmits<{ (e: 'save'): void }>()
const open = defineModel<boolean>()
</script>

<template>
  <UModal v-model="open" :ui="modalUi">
    <div class="space-y-3 p-4">
      <span class="text-sm font-semibold text-white"><span class="text-ok">$</span> log time</span>
      <div class="flex items-center gap-2">
        <DatePicker v-model="form.date" block class="flex-1" />
        <TimePicker v-model="form.start" class="w-28" />
        <span class="text-tertiary">–</span>
        <TimePicker v-model="form.end" :from="form.start" class="w-28" />
      </div>
      <UInput v-model="form.description" placeholder="What did you work on? (optional)" @keydown.enter="emit('save')" />
      <div class="flex justify-end gap-2 pt-1">
        <UButton color="gray" variant="ghost" @click="open = false">[ cancel ]</UButton>
        <UButton color="gray" @click="emit('save')">[ log ]</UButton>
      </div>
    </div>
  </UModal>
</template>
