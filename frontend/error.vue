<script setup lang="ts">
const props = defineProps<{ error: { statusCode?: number; message?: string } }>()
useHead({
  title: 'Error',
  link: [
    { rel: 'preconnect', href: 'https://fonts.googleapis.com' },
    { rel: 'preconnect', href: 'https://fonts.gstatic.com', crossorigin: '' },
    { rel: 'stylesheet', href: 'https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;500;700&display=swap' },
  ],
})
const code = props.error?.statusCode || 500
const msg = code === 404 ? 'no such route' : (props.error?.message || 'something broke')
</script>

<template>
  <div class="er">
    <div class="er-blob" aria-hidden="true"></div>
    <div class="er-win">
      <div class="er-bar"><span class="er-tl er-tl--r"></span><span class="er-tl er-tl--y"></span><span class="er-tl er-tl--g"></span><span class="er-path">~/pms — error</span></div>
      <div class="er-body">
        <p class="er-line"><span class="er-prompt">$</span> pms open <span class="er-arg">{{ $route?.path || '/' }}</span></p>
        <p class="er-code">{{ code }}</p>
        <p class="er-msg"><span class="er-x">✗</span> {{ msg }}</p>
        <button class="er-btn" @click="clearError({ redirect: '/' })">[ cd ~/ ]</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.er {
  --bg: #0b0c12; --line: #20222e; --ink: #c5cae0; --bright: #eef0fa; --mut: #6b7089; --faint: #474c61;
  --blue: #7aa2f7; --green: #9ece6a; --red: #f7768e;
  --grad: linear-gradient(100deg, #7aa2f7, #bb9af7 55%, #7dcfff);
  --m: 'JetBrains Mono', ui-monospace, monospace;
  position: relative; min-height: 100vh; display: grid; place-items: center; padding: 24px;
  background: var(--bg); color: var(--ink); font-family: var(--m); overflow: hidden;
}
.er-blob { position: absolute; width: 60vw; height: 60vw; left: 50%; top: -20vw; transform: translateX(-50%); border-radius: 50%; background: radial-gradient(circle, rgba(247,118,142,.14), transparent 65%); filter: blur(70px); pointer-events: none; }
.er-win { position: relative; width: 100%; max-width: 420px; border: 1px solid var(--line); border-radius: 14px; overflow: hidden; background: linear-gradient(180deg, #0f1118, #0b0c12); box-shadow: 0 40px 90px -40px rgba(0,0,0,.9); }
.er-bar { display: flex; align-items: center; gap: 8px; padding: 11px 13px; border-bottom: 1px solid var(--line); background: rgba(255,255,255,.015); }
.er-tl { width: 11px; height: 11px; border-radius: 50%; }
.er-tl--r { background: #f7768e; } .er-tl--y { background: #e0af68; } .er-tl--g { background: #9ece6a; }
.er-path { margin-left: 8px; font-size: 11.5px; color: var(--faint); }
.er-body { padding: 30px 26px 28px; }
.er-line { margin: 0; font-size: 13.5px; color: var(--mut); }
.er-prompt { color: var(--blue); }
.er-arg { color: var(--green); }
.er-code { font-size: 72px; font-weight: 700; letter-spacing: -.03em; margin: 14px 0 0; background: var(--grad); -webkit-background-clip: text; background-clip: text; color: transparent; }
.er-msg { color: var(--ink); font-size: 14px; margin: 6px 0 0; }
.er-x { color: var(--red); }
.er-btn { margin-top: 24px; font-family: var(--m); font-weight: 700; font-size: 14px; color: #0b0c12; border: none; border-radius: 9px; padding: 11px 18px; background: var(--grad); cursor: pointer; transition: transform .15s, box-shadow .2s; box-shadow: 0 8px 26px -10px rgba(122,162,247,.7); }
.er-btn:hover { transform: translateY(-2px); box-shadow: 0 14px 34px -10px rgba(187,154,247,.8); }
</style>
