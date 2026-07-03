<script setup lang="ts">
import { computed } from 'vue'
import { useAppStore } from '~/stores/app'
import { fileViewUrl } from '~/lib/upload'
import { useLightbox } from '~/composables/useLightbox'

// Один аттачмент: картинка (превью → лайтбокс) или файл (плашка + скачать).
// Молекула из Attachments.vue.
const props = defineProps<{ attachment: any; readonly?: boolean }>()
const store = useAppStore()
const lightbox = useLightbox()

const isImage = computed(() => String(props.attachment.contentType || '').startsWith('image/'))
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
function dl() { if (import.meta.client) window.open(fileViewUrl(props.attachment.s3Key) + '?dl=1', '_blank') }
</script>

<template>
  <button v-if="isImage" class="group relative h-24 w-24 overflow-hidden rounded-lg border border-subtle bg-elevated"
    @click="lightbox.open(fileViewUrl(attachment.s3Key), attachment.filename)">
    <img :src="fileViewUrl(attachment.s3Key)" class="h-full w-full object-cover" :alt="attachment.filename" />
    <span v-if="!readonly" class="absolute right-1 top-1 hidden h-5 w-5 items-center justify-center rounded-full bg-black/60 text-white group-hover:flex hover:text-danger"
      title="Remove" @click.stop="store.deleteAttachment(attachment.id)"><UIcon name="i-heroicons-x-mark" class="text-xs" /></span>
  </button>
  <div v-else class="group flex w-56 items-center gap-2.5 rounded-lg border border-subtle bg-elevated px-3 py-2.5">
    <span class="flex h-9 w-9 shrink-0 items-center justify-center rounded-md bg-active/40 text-accent"><UIcon :name="fileIcon(attachment)" class="text-lg" /></span>
    <span class="min-w-0 flex-1">
      <span class="block truncate text-sm text-secondary">{{ attachment.filename }}</span>
      <span class="block text-[11px] text-tertiary">{{ fmtSize(Number(attachment.size)) }}</span>
    </span>
    <button class="shrink-0 text-tertiary hover:text-white" title="Download" @click="dl"><UIcon name="i-heroicons-arrow-down-tray" /></button>
    <button v-if="!readonly" class="shrink-0 text-tertiary opacity-0 hover:text-danger group-hover:opacity-100" title="Remove" @click="store.deleteAttachment(attachment.id)"><UIcon name="i-heroicons-x-mark" /></button>
  </div>
</template>
