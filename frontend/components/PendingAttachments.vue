<script setup lang="ts">
import { computed } from 'vue'

// Превью ещё-не-отправленных вложений прямо в композере: картинки — тумбы, файлы — плашки.
const props = defineProps<{ files: File[] }>()
const emit = defineEmits<{ remove: [index: number] }>()

const previews = computed(() => props.files.map((f) => ({
  name: f.name, size: f.size, isImage: f.type.startsWith('image/'),
  url: f.type.startsWith('image/') && import.meta.client ? URL.createObjectURL(f) : '',
})))
const fmtSize = (n: number) => (n > 1048576 ? (n / 1048576).toFixed(1) + ' MB' : n > 1024 ? Math.round(n / 1024) + ' KB' : n + ' B')
</script>

<template>
  <div class="flex flex-wrap gap-2">
    <div v-for="(p, i) in previews" :key="i" class="group relative">
      <div v-if="p.isImage" class="h-16 w-16 overflow-hidden rounded-lg border border-subtle bg-elevated">
        <img :src="p.url" class="h-full w-full object-cover" alt="" />
      </div>
      <div v-else class="flex w-44 items-center gap-2 rounded-lg border border-subtle bg-elevated px-2.5 py-2">
        <UIcon name="i-heroicons-document" class="shrink-0 text-tertiary" />
        <span class="min-w-0 flex-1"><span class="block truncate text-xs text-secondary">{{ p.name }}</span><span class="block text-[10px] text-tertiary">{{ fmtSize(p.size) }}</span></span>
      </div>
      <button class="absolute -right-1.5 -top-1.5 flex h-5 w-5 items-center justify-center rounded-full bg-surface text-tertiary shadow hover:text-danger"
        title="Remove" @click="emit('remove', i)"><UIcon name="i-heroicons-x-mark" class="text-xs" /></button>
    </div>
  </div>
</template>
