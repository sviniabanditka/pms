<script setup lang="ts">
import { ref, computed } from 'vue'
import { useAppStore } from '~/stores/app'
import { fileViewUrl } from '~/lib/upload'
import { useLightbox } from '~/composables/useLightbox'

const props = defineProps<{ entityType: string; entityId: string; readonly?: boolean }>()
const store = useAppStore()
const lightbox = useLightbox()
const fileEl = ref<HTMLInputElement | null>(null)
const busy = ref(false)

const items = computed(() => store.attachmentsFor(props.entityType, props.entityId))
const isImage = (a: any) => String(a.contentType || '').startsWith('image/')
const fmtSize = (n: number) => (n > 1048576 ? (n / 1048576).toFixed(1) + ' MB' : n > 1024 ? Math.round(n / 1024) + ' KB' : n + ' B')
function fileIcon(a: any) {
  const ct = String(a.contentType || ''); const n = String(a.filename || '').toLowerCase()
  if (ct.startsWith('video/')) return 'i-heroicons-film'
  if (ct.startsWith('audio/')) return 'i-heroicons-musical-note'
  if (ct === 'application/pdf' || n.endsWith('.pdf')) return 'i-heroicons-document-text'
  if (/\.(zip|rar|7z|tar|gz)$/.test(n)) return 'i-heroicons-archive-box'
  if (/\.(doc|docx|txt|md|rtf)$/.test(n)) return 'i-heroicons-document-text'
  if (/\.(xls|xlsx|csv)$/.test(n)) return 'i-heroicons-table-cells'
  return 'i-heroicons-document'
}

async function onPick(e: Event) {
  const files = (e.target as HTMLInputElement).files
  if (!files?.length) return
  busy.value = true
  try { for (const f of Array.from(files)) await store.addAttachment(props.entityType, props.entityId, f) }
  finally { busy.value = false; if (fileEl.value) fileEl.value.value = '' }
}
function dl(a: any) { if (import.meta.client) window.open(fileViewUrl(a.s3Key) + '?dl=1', '_blank') }
</script>

<template>
  <div>
    <div v-if="items.length" class="flex flex-wrap gap-2">
      <!-- картинка: превью, клик → лайтбокс -->
      <button v-for="a in items.filter(isImage)" :key="a.id" class="group relative h-24 w-24 overflow-hidden rounded-lg border border-subtle bg-elevated"
        @click="lightbox.open(fileViewUrl(a.s3Key), a.filename)">
        <img :src="fileViewUrl(a.s3Key)" class="h-full w-full object-cover" :alt="a.filename" />
        <span v-if="!readonly" class="absolute right-1 top-1 hidden h-5 w-5 items-center justify-center rounded-full bg-black/60 text-white group-hover:flex hover:text-red-400"
          title="Remove" @click.stop="store.deleteAttachment(a.id)"><UIcon name="i-heroicons-x-mark" class="text-xs" /></span>
      </button>
      <!-- файл: плашка с иконкой типа + скачать -->
      <div v-for="a in items.filter((x) => !isImage(x))" :key="a.id"
        class="group flex w-56 items-center gap-2.5 rounded-lg border border-subtle bg-elevated px-3 py-2.5">
        <span class="flex h-9 w-9 shrink-0 items-center justify-center rounded-md bg-active/40 text-accent"><UIcon :name="fileIcon(a)" class="text-lg" /></span>
        <span class="min-w-0 flex-1">
          <span class="block truncate text-sm text-secondary">{{ a.filename }}</span>
          <span class="block text-[11px] text-tertiary">{{ fmtSize(Number(a.size)) }}</span>
        </span>
        <button class="shrink-0 text-tertiary hover:text-white" title="Download" @click="dl(a)"><UIcon name="i-heroicons-arrow-down-tray" /></button>
        <button v-if="!readonly" class="shrink-0 text-tertiary opacity-0 hover:text-red-400 group-hover:opacity-100" title="Remove" @click="store.deleteAttachment(a.id)"><UIcon name="i-heroicons-x-mark" /></button>
      </div>
    </div>
    <button v-if="!readonly" class="mt-2 flex items-center gap-1.5 rounded-md px-2 py-1.5 text-sm text-tertiary hover:bg-elevated hover:text-secondary" :disabled="busy" @click="fileEl?.click()">
      <UIcon :name="busy ? 'i-heroicons-arrow-path' : 'i-heroicons-paper-clip'" :class="{ 'animate-spin': busy }" />
      {{ busy ? 'Uploading…' : 'Attach files' }}
    </button>
    <input ref="fileEl" type="file" multiple class="hidden" @change="onPick" />
  </div>
</template>
