<script setup lang="ts">
import { ref, computed } from 'vue'
import { useAppStore } from '~/stores/app'

const props = defineProps<{ entityType: string; entityId: string; readonly?: boolean }>()
const store = useAppStore()
const fileEl = ref<HTMLInputElement | null>(null)
const busy = ref(false)

const items = computed(() => store.attachmentsFor(props.entityType, props.entityId))
// картинки первыми, потом файлы (сохраняем визуальную группировку)
const isImage = (a: any) => String(a.contentType || '').startsWith('image/')
const ordered = computed(() => [...items.value.filter(isImage), ...items.value.filter((x) => !isImage(x))])

async function onPick(e: Event) {
  const files = (e.target as HTMLInputElement).files
  if (!files?.length) return
  busy.value = true
  try { for (const f of Array.from(files)) await store.addAttachment(props.entityType, props.entityId, f) }
  finally { busy.value = false; if (fileEl.value) fileEl.value.value = '' }
}
</script>

<template>
  <div>
    <div v-if="ordered.length" class="flex flex-wrap gap-2">
      <AttachmentTile v-for="a in ordered" :key="a.id" :attachment="a" :readonly="readonly" />
    </div>
    <button v-if="!readonly" class="mt-2 flex items-center gap-1.5 rounded-md px-2 py-1.5 text-sm text-tertiary hover:bg-elevated hover:text-secondary" :disabled="busy" @click="fileEl?.click()">
      <UIcon :name="busy ? 'i-heroicons-arrow-path' : 'i-heroicons-paper-clip'" :class="{ 'animate-spin': busy }" />
      {{ busy ? 'Uploading…' : 'Attach files' }}
    </button>
    <input ref="fileEl" type="file" multiple class="hidden" @change="onPick" />
  </div>
</template>
