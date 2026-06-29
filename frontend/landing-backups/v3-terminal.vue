<script setup lang="ts">
import { isAuthed } from '~/lib/token'

onMounted(() => { if (isAuthed()) navigateTo('/my/assigned') })

useHead({
  title: 'PMS — the workspace that runs at your speed',
  link: [
    { rel: 'preconnect', href: 'https://fonts.googleapis.com' },
    { rel: 'preconnect', href: 'https://fonts.gstatic.com', crossorigin: '' },
    { rel: 'stylesheet', href: 'https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;500;600;700&display=swap' },
  ],
})

const scrolled = ref(false)
const onScroll = () => { scrolled.value = window.scrollY > 16 }

const onAnchorClick = (e: MouseEvent) => {
  const a = (e.target as HTMLElement)?.closest?.('a[href^="#"]') as HTMLAnchorElement | null
  if (!a) return
  const id = a.getAttribute('href')!.slice(1)
  const el = id && document.getElementById(id)
  if (!el) return
  e.preventDefault()
  const reduce = window.matchMedia('(prefers-reduced-motion: reduce)').matches
  el.scrollIntoView({ behavior: reduce ? 'auto' : 'smooth', block: 'start' })
  history.replaceState(null, '', `#${id}`)
}

const reduceMotion = () => import.meta.client && window.matchMedia('(prefers-reduced-motion: reduce)').matches

const vReveal = {
  mounted(el: HTMLElement, binding: any) {
    if (typeof binding.value === 'number') el.style.setProperty('--rd', `${binding.value}ms`)
    el.classList.add('rv')
    const io = new IntersectionObserver((es) => {
      for (const e of es) if (e.isIntersecting) { el.classList.add('rv-in'); io.unobserve(el) }
    }, { threshold: 0.15, rootMargin: '0px 0px -8% 0px' })
    io.observe(el)
  },
}

// ── hero terminal: self-typing pms CLI demo ─────────────────────────
const SCRIPT = [
  { cmd: 'pms issue new "Refactor billing" -a @maya -p high', out: '✓ PMS-204 created · @maya · priority:high' },
  { cmd: 'pms move PMS-201 done', out: '✓ moved → Done · automation fired · #releases pinged' },
  { cmd: 'pms cycle status', out: '◷ Cycle 12 · 84% complete · 3 left' },
]
type Line = { kind: 'cmd' | 'out'; text: string }
const lines = ref<Line[]>([])
const typing = ref('')
const showCursor = ref(true)
let cancelled = false
const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms))

async function runTerminal() {
  if (reduceMotion()) {
    lines.value = SCRIPT.flatMap((s) => [{ kind: 'cmd', text: s.cmd } as Line, { kind: 'out', text: s.out }])
    showCursor.value = false
    return
  }
  while (!cancelled) {
    lines.value = []
    for (const step of SCRIPT) {
      typing.value = ''
      for (let i = 0; i < step.cmd.length && !cancelled; i++) {
        typing.value += step.cmd[i]
        await sleep(26 + (step.cmd[i] === ' ' ? 28 : 0))
      }
      if (cancelled) return
      lines.value.push({ kind: 'cmd', text: typing.value })
      typing.value = ''
      await sleep(280)
      lines.value.push({ kind: 'out', text: step.out })
      await sleep(900)
    }
    await sleep(2600)
  }
}
onMounted(() => {
  window.addEventListener('scroll', onScroll, { passive: true }); onScroll()
  document.addEventListener('click', onAnchorClick)
  runTerminal()
})
onBeforeUnmount(() => {
  cancelled = true
  window.removeEventListener('scroll', onScroll)
  document.removeEventListener('click', onAnchorClick)
})

// colorize a typed command line (prompt + flags) for the syntax look
function tokens(s: string) {
  return s.split(/(\s+)/).map((w) => {
    if (/^-/.test(w)) return { c: 'flag', w }
    if (/^"/.test(w) || /"$/.test(w)) return { c: 'str', w }
    if (/^@/.test(w)) return { c: 'at', w }
    if (/^PMS-/.test(w)) return { c: 'key', w }
    return { c: 'txt', w }
  })
}

const features = [
  { f: 'issues.ts', d: 'Sub-issues, relations, labels, cycles. Board or list, your saved views.' },
  { f: 'docs.md', d: 'Specs that live next to the work. Markdown, mentions, embeds.' },
  { f: 'chat.sock', d: 'Threaded channels, reactions, unread — no fourth tab.' },
  { f: 'calendar.ics', d: 'Month, week, day. Drag to schedule, see who’s free.' },
  { f: 'meet.sfu', d: 'Huddle from any channel. Video built in.' },
  { f: 'time.log', d: 'Track hours, report by person/task/day, export.' },
]

const plans = [
  { name: 'free', price: '$0', unit: '', blurb: 'Get the team started.', feats: ['10 teammates', 'Issues · board · calendar', 'Realtime sync'], cta: 'init free', accent: false },
  { name: 'pro', price: '$8', unit: '/user·mo', blurb: 'Ship every week.', feats: ['Unlimited members', 'Automations & cycles', 'Time tracking & reports', 'Meetings & huddles'], cta: 'start trial', accent: true },
  { name: 'business', price: '$16', unit: '/user·mo', blurb: 'Scale the org.', feats: ['Everything in pro', 'Roles & permissions', 'Audit · SSO', 'Priority support'], cta: 'start trial', accent: false },
]

// tail -f live log (spotlight)
const LOG = [
  { t: 'sync', c: 'g', m: 'PMS-204 updated by @maya → applied in 142ms' },
  { t: 'auto', c: 'p', m: 'rule#7 matched · assigned @leo · label +review' },
  { t: 'chat', c: 'c', m: '#design · @ada: shipping the new grid 🎉' },
  { t: 'time', c: 'o', m: '@you logged 1h 15m on PMS-198' },
  { t: 'sync', c: 'g', m: 'cycle_12 snapshot · 84% · burndown ok' },
]
const logShown = ref<number>(reduceMotion() ? LOG.length : 0)
let logTimer: any
onMounted(() => {
  if (reduceMotion()) return
  logTimer = setInterval(() => { logShown.value = logShown.value >= LOG.length ? 1 : logShown.value + 1 }, 1400)
})
onBeforeUnmount(() => clearInterval(logTimer))
</script>

<template>
  <div class="t">
    <div class="t-scan" aria-hidden="true"></div>

    <!-- NAV -->
    <header class="t-nav" :class="{ 't-nav--solid': scrolled }">
      <div class="t-nav__in">
        <a href="/" class="t-brand"><span class="t-prompt">~/</span>pms<span class="t-cur">▮</span></a>
        <nav class="t-nav__links">
          <a href="#features">./features</a>
          <a href="#flow">./how</a>
          <a href="#pricing">./pricing</a>
        </nav>
        <div class="t-nav__cta">
          <a href="/login" class="t-quiet">sign in</a>
          <a href="/login" class="t-btn">[ start free ]</a>
        </div>
      </div>
    </header>

    <!-- HERO -->
    <section class="t-hero">
      <div class="t-hero__copy">
        <span class="t-eyebrow t-up" style="--i:0"># local-first · realtime · keyboard-driven</span>
        <h1 class="t-h1 t-up" style="--i:1">The workspace<br />that runs at<br /><span class="t-grad">your speed.</span></h1>
        <p class="t-sub t-up" style="--i:2">
          Issues, docs, chat, calendar and meetings in one fast workspace.
          Every command lands instantly — for the whole team, in real time.
        </p>
        <div class="t-actions t-up" style="--i:3">
          <a href="/login" class="t-btn t-btn--solid t-btn--lg">[ start free ]</a>
          <a href="#features" class="t-btn t-btn--lg">[ ./tour ]</a>
        </div>
        <p class="t-note t-up" style="--i:4"><span class="t-ok">✓</span> no credit card &nbsp; <span class="t-ok">✓</span> whole team in minutes</p>
      </div>

      <!-- SIGNATURE: self-typing terminal -->
      <div class="t-stage t-up" style="--i:3" aria-hidden="true">
        <div class="t-term">
          <div class="t-term__bar">
            <span class="t-tl t-tl--r"></span><span class="t-tl t-tl--y"></span><span class="t-tl t-tl--g"></span>
            <span class="t-term__path">— pms — bash — 80×24</span>
          </div>
          <div class="t-term__body">
            <template v-for="(l, i) in lines" :key="i">
              <p v-if="l.kind === 'cmd'" class="t-line"><span class="t-dollar">$</span> <span v-for="(tk, j) in tokens(l.text)" :key="j" :class="'tk-' + tk.c">{{ tk.w }}</span></p>
              <p v-else class="t-line t-line--out">{{ l.text }}</p>
            </template>
            <p v-if="typing" class="t-line"><span class="t-dollar">$</span> <span v-for="(tk, j) in tokens(typing)" :key="j" :class="'tk-' + tk.c">{{ tk.w }}</span><span class="t-blink" v-if="showCursor">▮</span></p>
            <p v-else-if="showCursor" class="t-line"><span class="t-dollar">$</span> <span class="t-blink">▮</span></p>
          </div>
        </div>
      </div>
    </section>

    <!-- STACK STRIP -->
    <section class="t-strip" v-reveal>
      <span class="t-strip__lbl">// one binary, the whole stack</span>
      <div class="t-strip__tags">
        <span v-for="x in ['issues', 'boards', 'docs', 'chat', 'calendar', 'meetings', 'automations', 'time']" :key="x">{{ x }}</span>
      </div>
    </section>

    <!-- FEATURES -->
    <section id="features" class="t-sec">
      <header class="t-shead" v-reveal>
        <span class="t-eyebrow"># everything the work needs</span>
        <h2 class="t-h2">ls -la <span class="t-grad">~/workspace</span></h2>
      </header>
      <div class="t-grid">
        <article v-for="(f, i) in features" :key="f.f" class="t-card" v-reveal="i * 60">
          <div class="t-card__h"><span class="t-card__dot"></span>{{ f.f }}</div>
          <p class="t-card__d">{{ f.d }}</p>
          <span class="t-card__run">run →</span>
        </article>
      </div>
    </section>

    <!-- SPOTLIGHT: automations as config -->
    <section class="t-spot">
      <div class="t-spot__copy" v-reveal>
        <span class="t-eyebrow"># automations</span>
        <h2 class="t-h2">Write the rule once.<br /><span class="t-grad">It runs forever.</span></h2>
        <p class="t-lead">Declare what happens when work changes. PMS applies it server-side on every event — reassign, re-label, set status, ping a channel. No glue code to babysit.</p>
        <ul class="t-checks">
          <li>Triggers on create, status, owner, priority</li>
          <li>Chain actions in a single rule</li>
          <li>Scope to a team or the whole org</li>
        </ul>
      </div>
      <div class="t-spot__art" v-reveal="120">
        <div class="t-term t-term--sm">
          <div class="t-term__bar"><span class="t-tl t-tl--r"></span><span class="t-tl t-tl--y"></span><span class="t-tl t-tl--g"></span><span class="t-term__path">rule.yaml</span></div>
          <pre class="t-yaml"><span class="tk-key">when</span>:
  <span class="tk-at">event</span>: <span class="tk-str">issue.status</span>
  <span class="tk-at">to</span>: <span class="tk-str">in_review</span>
<span class="tk-key">do</span>:
  - <span class="tk-flag">assign</span>: <span class="tk-str">@reviewer</span>
  - <span class="tk-flag">label</span>: <span class="tk-str">needs-review</span>
  - <span class="tk-flag">post</span>: <span class="tk-str">#reviews</span>
<span class="t-yaml__c"># ✓ live · 0 lines of code</span></pre>
        </div>
      </div>
    </section>

    <!-- SPOTLIGHT: realtime as tail -f -->
    <section class="t-spot t-spot--rev">
      <div class="t-spot__copy" v-reveal>
        <span class="t-eyebrow"># realtime</span>
        <h2 class="t-h2">tail -f <span class="t-grad">the whole team.</span></h2>
        <p class="t-lead">Local-first store, live deltas over WebSocket. Every edit lands for everyone the instant it happens — and the app keeps working when the network blinks.</p>
        <ul class="t-checks">
          <li>Optimistic edits, zero perceived latency</li>
          <li>Live presence across the team</li>
          <li>Reconnects on its own after a drop</li>
        </ul>
      </div>
      <div class="t-spot__art" v-reveal="120">
        <div class="t-term t-term--sm">
          <div class="t-term__bar"><span class="t-tl t-tl--r"></span><span class="t-tl t-tl--y"></span><span class="t-tl t-tl--g"></span><span class="t-term__path">$ pms tail -f</span></div>
          <div class="t-log">
            <p v-for="(g, i) in LOG" :key="i" class="t-log__l" :class="{ 'is-on': i < logShown }">
              <span class="t-log__t" :class="'lt-' + g.c">[{{ g.t }}]</span> {{ g.m }}
            </p>
            <p class="t-log__cur"><span class="t-blink">▮</span></p>
          </div>
        </div>
      </div>
    </section>

    <!-- HOW IT WORKS -->
    <section id="flow" class="t-sec">
      <header class="t-shead" v-reveal>
        <span class="t-eyebrow"># the loop</span>
        <h2 class="t-h2">From thought to <span class="t-grad">shipped.</span></h2>
      </header>
      <div class="t-steps">
        <article v-for="(s, i) in [
          { c: 'pms new', t: 'Capture', d: 'Spin up an issue in a keystroke. Tracked before the thought is gone.' },
          { c: 'pms plan', t: 'Organize', d: 'Drop it in a cycle, a board, a view. The layout is yours to keep.' },
          { c: 'pms ship', t: 'Ship', d: 'Move it to done — automations fire, the team hears about it, repeat.' },
        ]" :key="s.t" class="t-card t-step" v-reveal="i * 90">
          <div class="t-step__cmd"><span class="t-dollar">$</span> {{ s.c }}</div>
          <h3 class="t-step__t">{{ s.t }}</h3>
          <p class="t-card__d">{{ s.d }}</p>
        </article>
      </div>
    </section>

    <!-- PRICING -->
    <section id="pricing" class="t-sec">
      <header class="t-shead" v-reveal>
        <span class="t-eyebrow"># pricing</span>
        <h2 class="t-h2">pms install <span class="t-grad">--plan</span></h2>
        <p class="t-lead t-lead--c">Start free. Upgrade when the team grows. Cancel anytime.</p>
      </header>
      <div class="t-plans">
        <article v-for="(p, i) in plans" :key="p.name" class="t-card t-plan" :class="{ 't-plan--accent': p.accent }" v-reveal="i * 80">
          <span v-if="p.accent" class="t-plan__badge">// popular</span>
          <div class="t-plan__name"><span class="t-prompt">--</span>{{ p.name }}</div>
          <div class="t-plan__price"><span class="t-plan__amt">{{ p.price }}</span><span class="t-plan__unit">{{ p.unit }}</span></div>
          <p class="t-plan__blurb">{{ p.blurb }}</p>
          <ul class="t-plan__feats"><li v-for="f in p.feats" :key="f"><span class="t-ok">✓</span>{{ f }}</li></ul>
          <a href="/login" class="t-btn" :class="{ 't-btn--solid': p.accent }">[ {{ p.cta }} ]</a>
        </article>
      </div>
      <p class="t-selfhost"># prefer your own servers? <a href="/login">self-hosting is one binary →</a></p>
    </section>

    <!-- CTA -->
    <section class="t-cta" v-reveal>
      <div class="t-cta__panel">
        <span class="t-eyebrow"># ready when you are</span>
        <h2 class="t-h2">$ pms login<br /><span class="t-grad">— bring the team.</span></h2>
        <p class="t-lead t-lead--c">One workspace for everything your team ships. Free to start.</p>
        <a href="/login" class="t-btn t-btn--solid t-btn--lg">[ get started ]</a>
      </div>
    </section>

    <!-- FOOTER -->
    <footer class="t-foot">
      <div class="t-foot__in">
        <div class="t-foot__brand">
          <a href="/" class="t-brand"><span class="t-prompt">~/</span>pms</a>
          <p>The workspace that runs at your speed. Local-first, realtime, all in one.</p>
        </div>
        <div class="t-foot__cols">
          <div><h4>product</h4><a href="#features">features</a><a href="#flow">how it works</a><a href="#pricing">pricing</a></div>
          <div><h4>build</h4><a href="/login">issues</a><a href="/login">automations</a><a href="/login">calendar</a></div>
          <div><h4>company</h4><a href="#">about</a><a href="#">docs</a><a href="/login">sign in</a></div>
        </div>
      </div>
      <div class="t-foot__base"><span>© 2026 pms</span><span>$ made for teams that ship</span></div>
    </footer>
  </div>
</template>

<style scoped>
.t {
  --bg: #0b0c12; --panel: #11131b; --panel2: #0e0f16; --line: #20222e;
  --ink: #c5cae0; --bright: #eef0fa; --mut: #6b7089; --faint: #474c61;
  --blue: #7aa2f7; --purple: #bb9af7; --green: #9ece6a; --cyan: #7dcfff; --orange: #ff9e64; --red: #f7768e;
  --grad: linear-gradient(100deg, #7aa2f7, #bb9af7 55%, #7dcfff);
  --m: 'JetBrains Mono', ui-monospace, SFMono-Regular, Menlo, monospace;
  position: relative; background: var(--bg); color: var(--ink); overflow-x: clip; isolation: isolate;
  font-family: var(--m); font-size: 15px; line-height: 1.55;
  background-image: radial-gradient(60% 50% at 80% -5%, rgba(122,162,247,.10), transparent 60%), radial-gradient(50% 40% at 0% 0%, rgba(187,154,247,.08), transparent 60%);
}
.t ::selection { background: rgba(122,162,247,.32); }
.t a { color: inherit; }

/* subtle scanline texture */
.t-scan { position: fixed; inset: 0; z-index: 0; pointer-events: none; opacity: .5;
  background: repeating-linear-gradient(0deg, rgba(255,255,255,.013) 0 1px, transparent 1px 3px); }
.t > section, .t > header, .t > footer { position: relative; z-index: 1; }

/* nav */
.t-nav { position: fixed; inset: 0 0 auto 0; z-index: 50; border-bottom: 1px solid transparent; transition: background .3s, border-color .3s; }
.t-nav--solid { background: rgba(11,12,18,.72); backdrop-filter: blur(12px); border-color: var(--line); }
.t-nav__in { max-width: 1140px; margin: 0 auto; padding: 15px 24px; display: flex; align-items: center; gap: 28px; }
.t-brand { display: inline-flex; align-items: center; text-decoration: none; font-weight: 700; font-size: 16px; color: var(--bright); }
.t-prompt { color: var(--green); }
.t-cur { color: var(--blue); animation: blink 1.1s steps(1) infinite; }
.t-nav__links { display: flex; gap: 22px; margin-right: auto; }
.t-nav__links a { color: var(--mut); text-decoration: none; font-size: 13.5px; transition: color .16s; }
.t-nav__links a:hover { color: var(--green); }
.t-nav__cta { display: flex; align-items: center; gap: 16px; }
.t-quiet { color: var(--mut); text-decoration: none; font-size: 13.5px; }
.t-quiet:hover { color: var(--bright); }

/* buttons (bracket style) */
.t-btn { display: inline-flex; align-items: center; justify-content: center; font-family: var(--m); font-weight: 500; font-size: 13.5px; padding: 8px 14px; border: 1px solid var(--line); border-radius: 8px; background: rgba(255,255,255,.02); color: var(--ink); text-decoration: none; transition: border-color .18s, color .18s, background .18s, transform .16s, box-shadow .2s; white-space: nowrap; }
.t-btn:hover { border-color: var(--blue); color: var(--bright); transform: translateY(-2px); }
.t-btn--lg { padding: 12px 20px; font-size: 14px; }
.t-btn--solid { border-color: transparent; color: #0b0c12; font-weight: 700; background: var(--grad); box-shadow: 0 8px 26px -10px rgba(122,162,247,.7); }
.t-btn--solid:hover { color: #0b0c12; transform: translateY(-2px); box-shadow: 0 14px 34px -10px rgba(187,154,247,.8); }

/* hero */
.t-hero { max-width: 1140px; margin: 0 auto; padding: 160px 24px 90px; display: grid; grid-template-columns: 1.02fr 1.05fr; gap: 56px; align-items: center; }
.t-eyebrow { color: var(--mut); font-size: 13px; }
.t-h1 { font-weight: 700; font-size: clamp(38px, 5.6vw, 62px); line-height: 1.06; letter-spacing: -.02em; margin: 18px 0 0; color: var(--bright); }
.t-grad { background: var(--grad); -webkit-background-clip: text; background-clip: text; color: transparent; }
.t-sub { margin: 22px 0 0; max-width: 32rem; color: var(--mut); font-size: 15px; line-height: 1.7; }
.t-actions { display: flex; gap: 12px; margin-top: 30px; flex-wrap: wrap; }
.t-note { margin-top: 22px; color: var(--faint); font-size: 13px; }
.t-ok { color: var(--green); }

/* terminal window */
.t-term { border: 1px solid var(--line); border-radius: 12px; background: linear-gradient(180deg, #0f1118, #0b0c12); box-shadow: 0 40px 80px -36px rgba(0,0,0,.9), inset 0 1px 0 rgba(255,255,255,.04); overflow: hidden; }
.t-term__bar { display: flex; align-items: center; gap: 8px; padding: 10px 13px; border-bottom: 1px solid var(--line); background: rgba(255,255,255,.015); }
.t-tl { width: 11px; height: 11px; border-radius: 50%; }
.t-tl--r { background: #f7768e; } .t-tl--y { background: #e0af68; } .t-tl--g { background: #9ece6a; }
.t-term__path { margin-left: 8px; font-size: 11.5px; color: var(--faint); }
.t-term__body { padding: 16px 16px 18px; min-height: 230px; font-size: 13.5px; }
.t-line { margin: 0 0 7px; white-space: pre-wrap; word-break: break-word; }
.t-line--out { color: var(--green); padding-left: 14px; }
.t-dollar { color: var(--blue); }
.t-blink { color: var(--blue); animation: blink 1.05s steps(1) infinite; }
@keyframes blink { 0%, 50% { opacity: 1; } 51%, 100% { opacity: 0; } }
/* syntax tokens */
.tk-txt { color: var(--ink); } .tk-flag { color: var(--orange); } .tk-str { color: var(--green); }
.tk-at { color: var(--cyan); } .tk-key { color: var(--purple); }

/* stack strip */
.t-strip { max-width: 1140px; margin: 0 auto; padding: 18px 24px 0; text-align: center; }
.t-strip__lbl { color: var(--faint); font-size: 13px; }
.t-strip__tags { display: flex; flex-wrap: wrap; justify-content: center; gap: 10px; margin-top: 16px; }
.t-strip__tags span { font-size: 13px; color: var(--mut); border: 1px solid var(--line); border-radius: 7px; padding: 5px 11px; background: rgba(255,255,255,.02); }

/* sections */
.t-sec { max-width: 1140px; margin: 0 auto; padding: 120px 24px 0; scroll-margin-top: 82px; }
.t-shead { text-align: center; max-width: 40rem; margin: 0 auto 50px; }
.t-h2 { font-weight: 700; font-size: clamp(28px, 4vw, 44px); line-height: 1.1; letter-spacing: -.02em; margin: 12px 0 0; color: var(--bright); }
.t-lead { color: var(--mut); font-size: 15px; line-height: 1.7; margin: 16px 0 0; }
.t-lead--c { max-width: 32rem; margin-left: auto; margin-right: auto; }

/* feature grid */
.t-grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 14px; }
.t-card { position: relative; border: 1px solid var(--line); border-radius: 12px; padding: 22px; background: rgba(255,255,255,.018); transition: border-color .22s, transform .22s, background .22s; }
.t-card:hover { border-color: #2f3344; transform: translateY(-3px); background: rgba(122,162,247,.04); }
.t-card__h { display: flex; align-items: center; gap: 9px; color: var(--blue); font-weight: 600; font-size: 14px; }
.t-card__dot { width: 8px; height: 8px; border-radius: 2px; background: var(--purple); }
.t-card__d { color: var(--mut); font-size: 13.5px; line-height: 1.6; margin: 11px 0 0; }
.t-card__run { display: inline-block; margin-top: 14px; font-size: 12.5px; color: var(--faint); transition: color .2s, transform .2s; }
.t-card:hover .t-card__run { color: var(--green); transform: translateX(3px); }

/* spotlight */
.t-spot { max-width: 1080px; margin: 0 auto; padding: 120px 24px 0; display: grid; grid-template-columns: 1fr 1fr; gap: 54px; align-items: center; }
.t-spot--rev .t-spot__copy { order: 2; }
.t-checks { list-style: none; padding: 0; margin: 24px 0 0; display: flex; flex-direction: column; gap: 12px; }
.t-checks li { position: relative; padding-left: 26px; color: var(--ink); font-size: 14.5px; }
.t-checks li::before { content: '✓'; position: absolute; left: 0; color: var(--green); }
.t-term--sm .t-term__body { min-height: 0; }
.t-yaml { margin: 0; padding: 18px; font-family: var(--m); font-size: 13.5px; line-height: 1.75; color: var(--ink); white-space: pre; overflow-x: auto; }
.t-yaml__c { color: var(--faint); }
/* live log */
.t-log { padding: 14px 16px 16px; font-size: 13px; min-height: 178px; }
.t-log__l { margin: 0 0 8px; color: var(--ink); opacity: 0; transform: translateY(4px); transition: opacity .35s, transform .35s; }
.t-log__l.is-on { opacity: 1; transform: none; }
.t-log__t { font-weight: 600; }
.lt-g { color: var(--green); } .lt-p { color: var(--purple); } .lt-c { color: var(--cyan); } .lt-o { color: var(--orange); }
.t-log__cur { margin: 4px 0 0; }

/* steps */
.t-steps { display: grid; grid-template-columns: repeat(3, 1fr); gap: 16px; }
.t-step__cmd { color: var(--green); font-size: 13.5px; }
.t-step__t { font-weight: 700; font-size: 20px; color: var(--bright); margin: 12px 0 0; }

/* pricing */
.t-plans { display: grid; grid-template-columns: repeat(3, 1fr); gap: 16px; align-items: start; }
.t-plan { display: flex; flex-direction: column; padding: 26px; }
.t-plan--accent { border-color: rgba(122,162,247,.45); box-shadow: 0 30px 70px -40px rgba(122,162,247,.6); }
.t-plan__badge { color: var(--green); font-size: 12px; }
.t-plan__name { font-weight: 700; font-size: 18px; color: var(--bright); margin-top: 4px; }
.t-plan__price { display: flex; align-items: baseline; gap: 6px; margin: 12px 0 0; }
.t-plan__amt { font-size: 34px; font-weight: 700; color: var(--bright); }
.t-plan__unit { color: var(--faint); font-size: 12.5px; }
.t-plan__blurb { color: var(--mut); font-size: 13.5px; margin: 8px 0 0; }
.t-plan__feats { list-style: none; padding: 0; margin: 18px 0 22px; display: flex; flex-direction: column; gap: 10px; flex: 1; }
.t-plan__feats li { display: flex; gap: 9px; font-size: 13.5px; color: var(--ink); }
.t-selfhost { text-align: center; margin: 26px 0 0; color: var(--faint); font-size: 13.5px; }
.t-selfhost a { color: var(--cyan); text-decoration: none; }
.t-selfhost a:hover { text-decoration: underline; }

/* cta */
.t-cta { max-width: 1080px; margin: 124px auto 0; padding: 0 24px; }
.t-cta__panel { position: relative; overflow: hidden; text-align: center; border: 1px solid var(--line); border-radius: 16px; padding: 72px 24px; background: radial-gradient(120% 150% at 50% 0%, rgba(122,162,247,.14), transparent 58%), rgba(255,255,255,.015); }
.t-cta__panel .t-lead { margin-left: auto; margin-right: auto; }
.t-cta__panel .t-btn { margin-top: 28px; }

/* footer */
.t-foot { max-width: 1140px; margin: 124px auto 0; padding: 46px 24px 40px; border-top: 1px solid var(--line); }
.t-foot__in { display: grid; grid-template-columns: 1.4fr 2fr; gap: 40px; }
.t-foot__brand .t-brand { font-size: 16px; }
.t-foot__brand p { color: var(--mut); font-size: 13.5px; margin: 14px 0 0; max-width: 22rem; line-height: 1.6; }
.t-foot__cols { display: grid; grid-template-columns: repeat(3, 1fr); gap: 24px; }
.t-foot__cols h4 { font-size: 12px; color: var(--faint); margin: 0 0 14px; }
.t-foot__cols a { display: block; color: var(--mut); text-decoration: none; font-size: 13.5px; margin-bottom: 10px; transition: color .16s; }
.t-foot__cols a:hover { color: var(--green); }
.t-foot__base { display: flex; justify-content: space-between; margin-top: 38px; padding-top: 20px; border-top: 1px solid var(--line); color: var(--faint); font-size: 12.5px; }

/* motion */
.t-up { opacity: 0; transform: translateY(16px); animation: up .7s cubic-bezier(.22,1,.36,1) forwards; animation-delay: calc(var(--i,0) * 85ms + 100ms); }
@keyframes up { to { opacity: 1; transform: none; } }
.rv { opacity: 0; transform: translateY(16px); transition: opacity .7s cubic-bezier(.22,1,.36,1), transform .7s cubic-bezier(.22,1,.36,1); transition-delay: var(--rd, 0ms); }
.rv-in { opacity: 1; transform: none; }

/* responsive */
@media (max-width: 900px) {
  .t-hero { grid-template-columns: 1fr; padding-top: 124px; gap: 40px; }
  .t-stage { order: 2; }
  .t-grid, .t-steps, .t-plans { grid-template-columns: 1fr; }
  .t-spot, .t-spot--rev .t-spot__copy { grid-template-columns: 1fr; }
  .t-spot { grid-template-columns: 1fr; }
  .t-spot--rev .t-spot__copy { order: 0; }
  .t-foot__in { grid-template-columns: 1fr; }
  .t-nav__links { display: none; }
}
@media (prefers-reduced-motion: reduce) {
  .t-up, .rv { animation: none !important; opacity: 1 !important; transform: none !important; transition: none !important; }
  .t-cur, .t-blink, .t-scan { animation: none !important; }
  .t-log__l { opacity: 1 !important; transform: none !important; }
}
</style>
