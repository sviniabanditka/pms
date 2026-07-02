<script setup lang="ts">
import { useAppStore } from '~/stores/app'
import { modalUi } from '~/lib/ui'

const open = defineModel<boolean>()
// url заполняет openIcal на странице; regenerate обновляет его отсюда
const url = defineModel<string>('url', { default: '' })

const store = useAppStore()
const { wrap } = useActionWrap()

const icalCopied = ref(false)
const rotateIcal = () => wrap(async () => { url.value = await store.icalFeedUrl(true) })
function copyIcal() { navigator.clipboard?.writeText(url.value); icalCopied.value = true; setTimeout(() => (icalCopied.value = false), 1500) }
</script>

<template>
  <UModal v-model="open" :ui="modalUi">
    <div class="space-y-3 p-4">
      <span class="text-sm font-semibold text-white"><span class="text-ok">$</span> subscribe to your calendar</span>
      <p class="text-xs text-tertiary">Add this URL in Google Calendar / Apple Calendar (subscribe by URL). Read-only: your events, assigned deadlines and meetings. Keep it private — anyone with the link can view.</p>
      <div class="flex items-center gap-2">
        <UInput :model-value="url" readonly class="flex-1" :ui="{ base: 'font-mono text-xs' }" @focus="(e: any) => e.target.select()" />
        <UButton :icon="icalCopied ? 'i-heroicons-check' : 'i-heroicons-clipboard-document'" color="gray" @click="copyIcal">{{ icalCopied ? 'Copied' : 'Copy' }}</UButton>
      </div>
      <div class="flex items-center gap-2 pt-1">
        <UButton color="gray" variant="ghost" @click="rotateIcal">[ regenerate ]</UButton>
        <span class="text-xs text-tertiary">Regenerating invalidates the old link.</span>
        <UButton class="ml-auto" color="gray" variant="ghost" @click="open = false">[ close ]</UButton>
      </div>
    </div>
  </UModal>
</template>
