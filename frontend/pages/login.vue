<script setup lang="ts">
import { useRuntimeConfig } from '#imports'
import { setUser } from '~/lib/token'

const email = ref('bob@example.com')
const password = ref('supersecret')
const mode = ref<'login' | 'register'>('login')
const error = ref('')
const busy = ref(false)
const api = useRuntimeConfig().public.apiBase as string

async function submit() {
  error.value = ''
  busy.value = true
  try {
    const body: any = { email: email.value, password: password.value }
    if (mode.value === 'register') body.name = email.value.split('@')[0]
    const res = await fetch(`${api}/auth/${mode.value}`, {
      method: 'POST',
      credentials: 'include',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    })
    if (!res.ok) { error.value = (await res.json()).error || 'failed'; return }
    const data = await res.json()
    setUser(data.id)
    await navigateTo('/')
  } catch (e: any) {
    error.value = String(e)
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <div class="flex min-h-screen items-center justify-center bg-base">
    <UCard class="w-80">
      <template #header>
        <div class="flex items-center gap-2">
          <span class="flex h-6 w-6 items-center justify-center rounded bg-accent text-xs font-bold text-white">P</span>
          <span class="font-medium">PMS — {{ mode === 'login' ? 'Sign in' : 'Sign up' }}</span>
        </div>
      </template>

      <div class="space-y-3">
        <UInput v-model="email" placeholder="email" icon="i-heroicons-envelope" size="md" @keyup.enter="submit" />
        <UInput v-model="password" type="password" placeholder="password" icon="i-heroicons-lock-closed" size="md" @keyup.enter="submit" />
        <UButton :loading="busy" block size="md" @click="submit">
          {{ mode === 'login' ? 'Sign in' : 'Create account' }}
        </UButton>
        <p v-if="error" class="text-sm text-red-400">{{ error }}</p>
        <UButton variant="link" color="gray" size="xs" class="px-0" @click="mode = mode === 'login' ? 'register' : 'login'">
          {{ mode === 'login' ? 'Need an account? Sign up' : 'Have an account? Sign in' }}
        </UButton>
      </div>
    </UCard>
  </div>
</template>
