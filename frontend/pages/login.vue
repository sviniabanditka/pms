<script setup lang="ts">
import { useRuntimeConfig } from '#imports'
import { setUser } from '~/lib/token'

const email = ref('')
const password = ref('')
const mode = ref<'login' | 'register'>('login')
const error = ref('')
const busy = ref(false)
const api = useRuntimeConfig().public.apiBase as string

useHead({
  title: 'Sign in',
  link: [
    { rel: 'preconnect', href: 'https://fonts.googleapis.com' },
    { rel: 'preconnect', href: 'https://fonts.gstatic.com', crossorigin: '' },
    { rel: 'stylesheet', href: 'https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;500;600;700&display=swap' },
  ],
})

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
    const next = String(useRoute().query.next || '')
    await navigateTo(next.startsWith('/') ? next : '/')
  } catch (e: any) {
    error.value = String(e)
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <div class="lg">
    <div class="lg-blob" aria-hidden="true"></div>
    <div class="lg-win">
      <div class="lg-bar">
        <span class="lg-tl lg-tl--r"></span><span class="lg-tl lg-tl--y"></span><span class="lg-tl lg-tl--g"></span>
        <span class="lg-path">~/stdtask — {{ mode === 'login' ? 'login' : 'register' }}</span>
      </div>
      <div class="lg-body">
        <a href="/" class="lg-brand"><span class="lg-mark">S</span>stdtask</a>
        <p class="lg-hint"># {{ mode === 'login' ? 'welcome back. sign in to your workspace.' : 'create an account — your team in minutes.' }}</p>

        <label class="lg-field">
          <span class="lg-prompt">$</span>
          <input v-model="email" type="email" placeholder="email" autocomplete="email" @keyup.enter="submit" />
        </label>
        <label class="lg-field">
          <span class="lg-prompt">$</span>
          <input v-model="password" type="password" placeholder="password" autocomplete="current-password" @keyup.enter="submit" />
        </label>

        <button class="lg-btn" :disabled="busy" @click="submit">
          {{ busy ? 'running…' : (mode === 'login' ? '[ sign in ]' : '[ create account ]') }}
        </button>
        <p v-if="error" class="lg-err">✗ {{ error }}</p>

        <button class="lg-switch" @click="mode = mode === 'login' ? 'register' : 'login'">
          {{ mode === 'login' ? '// need an account? register →' : '// have an account? sign in →' }}
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.lg {
  --bg: #0b0c12; --line: #20222e; --ink: #c5cae0; --bright: #eef0fa; --mut: #6b7089; --faint: #474c61;
  --blue: #7aa2f7; --green: #9ece6a; --red: #f7768e;
  --grad: linear-gradient(100deg, #7aa2f7, #bb9af7 55%, #7dcfff);
  --m: 'JetBrains Mono', ui-monospace, monospace;
  position: relative; min-height: 100vh; display: grid; place-items: center; padding: 24px;
  background: var(--bg); color: var(--ink); font-family: var(--m); overflow: hidden;
}
.lg-blob { position: absolute; width: 60vw; height: 60vw; left: 50%; top: -20vw; transform: translateX(-50%); border-radius: 50%; background: radial-gradient(circle, rgba(122,162,247,.18), transparent 65%); filter: blur(70px); pointer-events: none; }
.lg-win { position: relative; width: 100%; max-width: 380px; border: 1px solid var(--line); border-radius: 14px; overflow: hidden; background: linear-gradient(180deg, #0f1118, #0b0c12); box-shadow: 0 40px 90px -40px rgba(0,0,0,.9), inset 0 1px 0 rgba(255,255,255,.04); }
.lg-bar { display: flex; align-items: center; gap: 8px; padding: 11px 13px; border-bottom: 1px solid var(--line); background: rgba(255,255,255,.015); }
.lg-tl { width: 11px; height: 11px; border-radius: 50%; }
.lg-tl--r { background: #f7768e; } .lg-tl--y { background: #e0af68; } .lg-tl--g { background: #9ece6a; }
.lg-path { margin-left: 8px; font-size: 11.5px; color: var(--faint); }
.lg-body { padding: 26px 24px 24px; }
.lg-brand { display: inline-flex; align-items: center; gap: 9px; text-decoration: none; color: var(--bright); font-weight: 700; font-size: 18px; }
.lg-mark { display: grid; place-items: center; width: 24px; height: 24px; border-radius: 8px; color: #0b0c12; font-weight: 700; background: var(--grad); }
.lg-hint { color: var(--mut); font-size: 12.5px; margin: 14px 0 20px; }
.lg-field { display: flex; align-items: center; gap: 9px; border: 1px solid var(--line); border-radius: 9px; padding: 0 12px; margin-bottom: 11px; background: rgba(255,255,255,.02); transition: border-color .18s; }
.lg-field:focus-within { border-color: var(--blue); }
.lg-prompt { color: var(--green); }
.lg-field input { flex: 1; background: none; border: none; outline: none; color: var(--ink); font-family: var(--m); font-size: 14px; padding: 11px 0; }
.lg-field input::placeholder { color: var(--faint); }
.lg-btn { width: 100%; margin-top: 6px; font-family: var(--m); font-weight: 700; font-size: 14px; color: #0b0c12; border: none; border-radius: 9px; padding: 12px; background: var(--grad); cursor: pointer; transition: transform .15s, box-shadow .2s, opacity .2s; box-shadow: 0 8px 26px -10px rgba(122,162,247,.7); }
.lg-btn:hover:not(:disabled) { transform: translateY(-2px); box-shadow: 0 14px 34px -10px rgba(187,154,247,.8); }
.lg-btn:disabled { opacity: .6; cursor: default; }
.lg-err { color: var(--red); font-size: 13px; margin: 12px 0 0; }
.lg-switch { display: block; width: 100%; text-align: left; margin-top: 16px; background: none; border: none; color: var(--mut); font-family: var(--m); font-size: 12.5px; cursor: pointer; transition: color .15s; }
.lg-switch:hover { color: var(--blue); }
</style>
