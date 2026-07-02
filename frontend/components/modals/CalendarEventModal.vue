<script setup lang="ts">
import { useAppStore } from '~/stores/app'
import { modalUi } from '~/lib/ui'

// calForm живёт на странице (его заполняют клики/драги по сетке календаря) — сюда приходит по ссылке
defineProps<{ form: { id: string; title: string; description: string; kind: string; participants: string[]; date: string; start: string; end: string; allDay: boolean; recurrence: string; recUntil: string } }>()
const emit = defineEmits<{ (e: 'save'): void; (e: 'delete'): void }>()
const open = defineModel<boolean>()

const store = useAppStore()
</script>

<template>
  <UModal v-model="open" :ui="modalUi">
    <div class="space-y-3 p-4">
      <span class="text-sm font-semibold text-white"><span class="text-ok">$</span> {{ form.id ? 'Edit event' : form.kind === 'meeting' ? 'New meeting' : 'New event' }}</span>
      <UInput v-model="form.title" placeholder="Title" autofocus @keydown.enter="emit('save')" />
      <div class="flex items-center gap-2">
        <USelectMenu v-model="form.kind" class="w-32" :disabled="!!form.id" :options="form.id ? [{ label: 'Event', value: 'event' }, { label: 'Day off', value: 'dayoff' }] : [{ label: 'Event', value: 'event' }, { label: 'Day off', value: 'dayoff' }, { label: 'Meeting', value: 'meeting' }]" value-attribute="value" option-attribute="label" />
        <DatePicker v-model="form.date" block class="flex-1" />
      </div>
      <div v-if="!form.allDay || form.kind === 'meeting'" class="flex items-center gap-2">
        <TimePicker v-model="form.start" class="flex-1" />
        <span class="text-tertiary">–</span>
        <TimePicker v-model="form.end" :from="form.start" class="flex-1" />
      </div>
      <TermCheck v-if="form.kind !== 'meeting'" v-model="form.allDay" label="All day" />
      <!-- повтор (не для митинга) -->
      <div v-if="form.kind !== 'meeting'" class="flex items-center gap-2">
        <USelectMenu v-model="form.recurrence" class="w-40"
          :options="[{ label: 'Does not repeat', value: 'none' }, { label: 'Daily', value: 'daily' }, { label: 'Weekly', value: 'weekly' }, { label: 'Monthly', value: 'monthly' }]"
          value-attribute="value" option-attribute="label" />
        <template v-if="form.recurrence !== 'none'">
          <span class="text-xs text-tertiary">until</span>
          <DatePicker v-model="form.recUntil" class="flex-1" placeholder="forever" />
        </template>
      </div>
      <!-- участники митинга (free/busy: видно занятость в сетке через тоггл календарей слева) -->
      <USelectMenu v-if="form.kind === 'meeting'" v-model="form.participants" multiple searchable placeholder="Invite participants…"
        :options="store.wsUsers.map((u) => ({ label: u.name || u.email, value: u.id }))" value-attribute="value" option-attribute="label" />
      <UTextarea v-if="form.kind !== 'meeting'" v-model="form.description" placeholder="Notes (optional)" :rows="2" />
      <div class="flex items-center gap-2 pt-1">
        <UButton v-if="form.id" color="red" variant="ghost" @click="emit('delete')">[ delete ]</UButton>
        <UButton class="ml-auto" color="gray" variant="ghost" @click="open = false">[ cancel ]</UButton>
        <UButton @click="emit('save')">[ {{ form.id ? 'save' : 'create' }} ]</UButton>
      </div>
    </div>
  </UModal>
</template>
