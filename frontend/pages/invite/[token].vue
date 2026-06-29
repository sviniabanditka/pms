<script setup lang="ts">
import { isAuthed } from '~/lib/token'
import { useAppStore } from '~/stores/app'

const route = useRoute()
const token = String(route.params.token || '')
const state = ref<'pending' | 'error'>('pending')
const error = ref('')

onMounted(async () => {
  if (!token) { state.value = 'error'; error.value = 'Invalid invite link'; return }
  // не залогинен → на /login, вернёмся сюда после входа/регистрации
  if (!isAuthed()) { await navigateTo(`/login?next=${encodeURIComponent('/invite/' + token)}`); return }
  try {
    await useAppStore().acceptInvitation(token)
    // полная перезагрузка: приложение перетянет список воркспейсов и откроет присоединённый
    window.location.href = '/'
  } catch (e: any) {
    state.value = 'error'
    error.value = e?.message?.replace(/^\[[^\]]+\]\s*/, '') || 'Could not accept invitation'
  }
})
</script>

<template>
  <div class="flex min-h-screen items-center justify-center bg-base text-white">
    <div class="w-80 rounded-lg border border-subtle bg-elevated p-6 text-center">
      <template v-if="state === 'pending'">
        <UIcon name="i-heroicons-arrow-path" class="mx-auto mb-2 h-6 w-6 animate-spin text-tertiary" />
        <p class="text-sm text-tertiary">Joining workspace…</p>
      </template>
      <template v-else>
        <UIcon name="i-heroicons-x-circle" class="mx-auto mb-2 h-6 w-6 text-red-400" />
        <p class="mb-3 text-sm">{{ error }}</p>
        <UButton color="gray" block @click="navigateTo('/')">Go to app</UButton>
      </template>
    </div>
  </div>
</template>
