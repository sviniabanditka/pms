<script setup lang="ts">
import { isAuthed } from '~/lib/token'

// Лендинг — публичная страница для разлогиненных. Залогиненных уводим в приложение.
onMounted(() => { if (isAuthed()) navigateTo('/my/assigned') })

useHead({
  title: 'PMS — Project management that keeps up',
  link: [
    { rel: 'preconnect', href: 'https://fonts.googleapis.com' },
    { rel: 'preconnect', href: 'https://fonts.gstatic.com', crossorigin: '' },
    { rel: 'stylesheet', href: 'https://fonts.googleapis.com/css2?family=Space+Grotesk:wght@400;500;600;700&family=JetBrains+Mono:wght@400;500;600&display=swap' },
  ],
})

// nav: фон при скролле
const scrolled = ref(false)
const onScroll = () => { scrolled.value = window.scrollY > 16 }

// плавный скролл по якорным ссылкам (с учётом фикс-навбара через scroll-margin-top)
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

onMounted(() => {
  window.addEventListener('scroll', onScroll, { passive: true }); onScroll()
  document.addEventListener('click', onAnchorClick)
})
onBeforeUnmount(() => {
  window.removeEventListener('scroll', onScroll)
  document.removeEventListener('click', onAnchorClick)
})

// scroll-reveal директива (v-reveal): появление при попадании во вьюпорт
const vReveal = {
  mounted(el: HTMLElement, binding: any) {
    if (typeof binding.value === 'number') el.style.setProperty('--reveal-delay', `${binding.value}ms`)
    el.classList.add('reveal')
    const io = new IntersectionObserver((entries) => {
      for (const e of entries) {
        if (e.isIntersecting) { el.classList.add('reveal-in'); io.unobserve(el) }
      }
    }, { threshold: 0.16, rootMargin: '0px 0px -8% 0px' })
    io.observe(el)
  },
}

const features = [
  { tag: 'BOARD', title: 'Issues that move', body: 'Drag across states, group by anything, keyboard through it all. Sub-issues, relations, labels, cycles — every detail of the work in one tracker.' },
  { tag: 'CALENDAR', title: 'Plan in time', body: 'Personal calendar with month, week and day grids. Drag to schedule, see teammates’ free/busy, subscribe over iCal.' },
  { tag: 'AUTOMATE', title: 'Rules that run', body: 'When an issue is created or changes state, set status, reassign, add labels, or post to a channel. No glue code.' },
  { tag: 'TRACK', title: 'Time, accounted', body: 'Start a timer or log hours after the fact. Reports roll up by person, task or day — export when finance asks.' },
  { tag: 'MEET', title: 'Call without leaving', body: 'Schedule a meeting or start a huddle from any channel. Video over an SFU, no extra tab, no extra invite.' },
  { tag: 'REALTIME', title: 'Always in sync', body: 'Local-first store, live deltas over WebSocket. Edits land instantly for everyone — and the app keeps working offline.' },
]

const steps = [
  { n: 'PMS-001', t: 'Capture', d: 'Spin up an issue in a keystroke. Title, assignee, priority, a sub-task — it’s tracked before the thought is gone.' },
  { n: 'PMS-014', t: 'Organize', d: 'Sort it into a cycle, a board, a saved view. Filters compile to real queries; the layout is yours to keep.' },
  { n: 'PMS-092', t: 'Ship', d: 'Move it to Done and the automations fire, the burndown ticks, the channel hears about it. Repeat.' },
]

const plans = [
  { name: 'Free', price: '$0', unit: 'forever', blurb: 'For a small team finding its feet.', feats: ['Up to 10 members', 'Issues, board & calendar', 'Realtime sync', 'Community support'], cta: 'Start free', accent: false },
  { name: 'Team', price: '$8', unit: 'per user / mo', blurb: 'For teams shipping every week.', feats: ['Unlimited members', 'Automations & cycles', 'Time tracking & reports', 'Meetings & huddles', 'Priority support'], cta: 'Start free trial', accent: true },
  { name: 'Self-host', price: 'Free', unit: 'your servers', blurb: 'Your data, your infrastructure.', feats: ['Single binary + Postgres', 'Row-level multi-tenancy', 'S3-compatible storage', 'No seat limits'], cta: 'Read the docs', accent: false },
]
</script>

<template>
  <div class="lp">
    <!-- ambient background -->
    <div class="lp-aurora" aria-hidden="true"></div>
    <div class="lp-grid" aria-hidden="true"></div>

    <!-- NAV -->
    <header class="lp-nav" :class="{ 'lp-nav--solid': scrolled }">
      <div class="lp-nav__in">
        <a href="/" class="lp-brand">
          <span class="lp-brand__mark">P</span>
          <span class="lp-brand__name">PMS</span>
        </a>
        <nav class="lp-nav__links">
          <a href="#features">Features</a>
          <a href="#workflow">Workflow</a>
          <a href="#pricing">Pricing</a>
        </nav>
        <div class="lp-nav__cta">
          <a href="/login" class="lp-link-quiet">Log in</a>
          <a href="/login" class="lp-btn lp-btn--solid">Get started</a>
        </div>
      </div>
    </header>

    <!-- HERO -->
    <section class="lp-hero">
      <div class="lp-hero__copy">
        <span class="lp-eyebrow lp-load" style="--i:0">LOCAL-FIRST&nbsp;·&nbsp;REALTIME&nbsp;·&nbsp;KEYBOARD-FIRST</span>
        <h1 class="lp-h1 lp-load" style="--i:1">Project management<br />that keeps up<span class="lp-dot">.</span></h1>
        <p class="lp-sub lp-load" style="--i:2">
          Issues, docs, chat, calendar and meetings in one fast, self-hostable workspace.
          Every edit is instant and lands for the whole team in real time.
        </p>
        <div class="lp-hero__actions lp-load" style="--i:3">
          <a href="/login" class="lp-btn lp-btn--solid lp-btn--lg">Get started — it’s free</a>
          <a href="#features" class="lp-btn lp-btn--ghost lp-btn--lg">See how it works</a>
        </div>
        <div class="lp-hero__meta lp-load" style="--i:4">
          <span><i class="lp-pulse"></i> Sync under 200&nbsp;ms</span>
          <span>Self-host in one binary</span>
        </div>
      </div>

      <!-- SIGNATURE: live work surface -->
      <div class="lp-stage lp-load" style="--i:3" aria-hidden="true">
        <div class="lp-window">
          <div class="lp-window__bar">
            <span class="lp-tl"></span><span class="lp-tl"></span><span class="lp-tl"></span>
            <span class="lp-window__path">workspace / engineering / board</span>
            <span class="lp-presence">
              <i style="--a:#6E7BFF"></i><i style="--a:#A78BFA"></i><i style="--a:#5CE1E6"></i>
            </span>
          </div>
          <div class="lp-board">
            <div class="lp-col" style="--c:0">
              <div class="lp-col__h">Backlog <b>3</b></div>
              <div class="lp-card" style="--d:1"><span class="lp-key">PMS-118</span><p>Compile filter AST to SQL</p><span class="lp-st lp-st--gray"></span></div>
              <div class="lp-card" style="--d:2"><span class="lp-key">PMS-121</span><p>iCal feed line-folding</p><span class="lp-st lp-st--gray"></span></div>
              <div class="lp-card lp-card--ghost" style="--d:3"></div>
            </div>
            <div class="lp-col" style="--c:1">
              <div class="lp-col__h">In progress <b>2</b></div>
              <div class="lp-card lp-card--live" style="--d:2"><span class="lp-key">PMS-092</span><p>Incremental delta sync</p><span class="lp-st lp-st--iris"></span></div>
              <div class="lp-card" style="--d:3"><span class="lp-key">PMS-095</span><p>Invite accept flow</p><span class="lp-st lp-st--iris"></span></div>
            </div>
            <div class="lp-col" style="--c:2">
              <div class="lp-col__h">Done <b>5</b></div>
              <div class="lp-card lp-card--dim" style="--d:3"><span class="lp-key">PMS-088</span><p>Automations engine</p><span class="lp-st lp-st--live"></span></div>
              <div class="lp-card lp-card--dim" style="--d:4"><span class="lp-key">PMS-081</span><p>Personal calendar</p><span class="lp-st lp-st--live"></span></div>
            </div>
            <div class="lp-scan"></div>
          </div>
        </div>
        <div class="lp-stage__glow"></div>
      </div>
    </section>

    <!-- PROOF -->
    <section class="lp-proof" v-reveal>
      <div v-for="(s, i) in [
        { k: 'Realtime', v: '< 200ms', d: 'edit-to-everyone' },
        { k: 'One binary', v: 'Self-host', d: 'Postgres + S3' },
        { k: 'Multi-tenant', v: 'RLS', d: 'isolation at the row' },
        { k: 'All-in-one', v: '6-in-1', d: 'issues → meetings' },
      ]" :key="s.k" class="lp-proof__cell" v-reveal="i * 80">
        <span class="lp-proof__k">{{ s.k }}</span>
        <span class="lp-proof__v">{{ s.v }}</span>
        <span class="lp-proof__d">{{ s.d }}</span>
      </div>
    </section>

    <!-- FEATURES -->
    <section id="features" class="lp-section">
      <header class="lp-shead" v-reveal>
        <span class="lp-eyebrow">EVERYTHING THE WORK NEEDS</span>
        <h2 class="lp-h2">One workspace, the whole loop</h2>
        <p class="lp-lead">Stop stitching five tools together. Plan, build, talk and ship without ever switching tabs.</p>
      </header>
      <div class="lp-bento">
        <article v-for="(f, i) in features" :key="f.tag" class="lp-fcard" v-reveal="i * 70">
          <span class="lp-fcard__tag">{{ f.tag }}</span>
          <h3 class="lp-fcard__t">{{ f.title }}</h3>
          <p class="lp-fcard__b">{{ f.body }}</p>
          <span class="lp-fcard__arrow">→</span>
        </article>
      </div>
    </section>

    <!-- DEEP DIVE: Automations -->
    <section class="lp-split">
      <div class="lp-split__copy" v-reveal>
        <span class="lp-eyebrow">AUTOMATIONS</span>
        <h2 class="lp-h2">Describe the rule.<br />It runs itself.</h2>
        <p class="lp-lead">When something happens to an issue, decide what should happen next — once. The engine applies it on every change, server-side, no scripts to maintain.</p>
        <ul class="lp-checks">
          <li>Triggers on create, status, assignee or priority</li>
          <li>Set status, reassign, add a label, post to a channel</li>
          <li>Scoped to a team or the whole workspace</li>
        </ul>
      </div>
      <div class="lp-split__art" v-reveal="120">
        <div class="lp-rule">
          <div class="lp-rule__row" style="--r:0">
            <span class="lp-rule__when">WHEN</span>
            <span class="lp-chip">issue moves to <b>In review</b></span>
          </div>
          <div class="lp-rule__line"></div>
          <div class="lp-rule__row" style="--r:1"><span class="lp-rule__then">THEN</span><span class="lp-chip lp-chip--act">assign <b>reviewer on rota</b></span></div>
          <div class="lp-rule__row" style="--r:2"><span class="lp-rule__then">THEN</span><span class="lp-chip lp-chip--act">add label <b>needs-review</b></span></div>
          <div class="lp-rule__row" style="--r:3"><span class="lp-rule__then">THEN</span><span class="lp-chip lp-chip--act">post to <b>#eng-reviews</b></span></div>
        </div>
      </div>
    </section>

    <!-- DEEP DIVE: Realtime (reversed) -->
    <section class="lp-split lp-split--rev">
      <div class="lp-split__copy" v-reveal>
        <span class="lp-eyebrow">LOCAL-FIRST</span>
        <h2 class="lp-h2">Instant now.<br />In sync always.</h2>
        <p class="lp-lead">The app reads from a local store, so every interaction is immediate. Changes stream as deltas over WebSocket and merge for everyone — and nothing breaks when the network blinks.</p>
        <ul class="lp-checks">
          <li>Zero-latency edits, optimistic by default</li>
          <li>Live presence and cursors across the team</li>
          <li>Reconnects on its own after a drop</li>
        </ul>
      </div>
      <div class="lp-split__art" v-reveal="120">
        <div class="lp-sync">
          <div class="lp-sync__node"><span class="lp-avatar" style="--a:#6E7BFF">A</span><span class="lp-sync__lbl">edits PMS-092</span></div>
          <div class="lp-sync__beam"></div>
          <div class="lp-sync__hub">⚡</div>
          <div class="lp-sync__beam lp-sync__beam--2"></div>
          <div class="lp-sync__node lp-sync__node--r"><span class="lp-avatar" style="--a:#A78BFA">M</span><span class="lp-sync__lbl">sees it land</span></div>
        </div>
      </div>
    </section>

    <!-- WORKFLOW (a real sequence → numbering earns its place) -->
    <section id="workflow" class="lp-section">
      <header class="lp-shead" v-reveal>
        <span class="lp-eyebrow">THE LOOP</span>
        <h2 class="lp-h2">From thought to shipped</h2>
        <p class="lp-lead">Three moves, the same every time. The tool gets out of the way so the team keeps moving.</p>
      </header>
      <div class="lp-steps">
        <article v-for="(s, i) in steps" :key="s.t" class="lp-step" v-reveal="i * 90">
          <span class="lp-step__key">{{ s.n }}</span>
          <h3 class="lp-step__t">{{ s.t }}</h3>
          <p class="lp-step__d">{{ s.d }}</p>
          <span v-if="i < steps.length - 1" class="lp-step__arc">→</span>
        </article>
      </div>
    </section>

    <!-- PRICING -->
    <section id="pricing" class="lp-section">
      <header class="lp-shead" v-reveal>
        <span class="lp-eyebrow">PRICING</span>
        <h2 class="lp-h2">Start free. Host it yourself.</h2>
        <p class="lp-lead">No card to try it. Run it on our cloud, or take the binary and run it on yours.</p>
      </header>
      <div class="lp-plans">
        <article v-for="(p, i) in plans" :key="p.name" class="lp-plan" :class="{ 'lp-plan--accent': p.accent }" v-reveal="i * 90">
          <span v-if="p.accent" class="lp-plan__badge">Most popular</span>
          <h3 class="lp-plan__name">{{ p.name }}</h3>
          <div class="lp-plan__price"><span class="lp-plan__amt">{{ p.price }}</span><span class="lp-plan__unit">{{ p.unit }}</span></div>
          <p class="lp-plan__blurb">{{ p.blurb }}</p>
          <ul class="lp-plan__feats">
            <li v-for="f in p.feats" :key="f"><span class="lp-tick">✓</span>{{ f }}</li>
          </ul>
          <a href="/login" class="lp-btn" :class="p.accent ? 'lp-btn--solid' : 'lp-btn--ghost'">{{ p.cta }}</a>
        </article>
      </div>
    </section>

    <!-- CTA -->
    <section class="lp-cta" v-reveal>
      <div class="lp-cta__panel">
        <div class="lp-cta__glow"></div>
        <h2 class="lp-h2">Ship the next thing faster.</h2>
        <p class="lp-lead">Bring your team into one workspace today. Free to start, yours to host.</p>
        <a href="/login" class="lp-btn lp-btn--solid lp-btn--lg">Get started</a>
      </div>
    </section>

    <!-- FOOTER -->
    <footer class="lp-footer">
      <div class="lp-footer__in">
        <div class="lp-footer__brand">
          <a href="/" class="lp-brand"><span class="lp-brand__mark">P</span><span class="lp-brand__name">PMS</span></a>
          <p>Project management that keeps up. Local-first, realtime, self-hostable.</p>
        </div>
        <div class="lp-footer__cols">
          <div><h4>Product</h4><a href="#features">Features</a><a href="#workflow">Workflow</a><a href="#pricing">Pricing</a></div>
          <div><h4>Build</h4><a href="/login">Issues</a><a href="/login">Automations</a><a href="/login">Calendar</a></div>
          <div><h4>Company</h4><a href="#">About</a><a href="#">Docs</a><a href="/login">Sign in</a></div>
        </div>
      </div>
      <div class="lp-footer__base">
        <span>© 2026 PMS</span>
        <span class="lp-mono">built for teams that ship</span>
      </div>
    </footer>
  </div>
</template>

<style scoped>
/* ---------- palette + type ---------- */
.lp {
  --ink: #08080A; --panel: #0F1014; --panel2: #15161C; --line: #1C1D24;
  --txt: #ECECEE; --mist: #8A8F9C; --faint: #5A5F6B;
  --indigo: #6E7BFF; --iris: #A78BFA; --live: #5CE1E6;
  --disp: 'Space Grotesk', 'Inter var', sans-serif;
  --mono: 'JetBrains Mono', ui-monospace, monospace;
  position: relative; background: var(--ink); color: var(--txt);
  overflow-x: clip; isolation: isolate;
  font-family: 'Inter var', Inter, sans-serif;
}
.lp ::selection { background: rgba(110,123,255,.32); }

/* ---------- ambient bg ---------- */
.lp-aurora {
  position: fixed; inset: -20% -10% auto; height: 90vh; z-index: 0; pointer-events: none;
  background:
    radial-gradient(40% 50% at 22% 18%, rgba(110,123,255,.20), transparent 70%),
    radial-gradient(38% 46% at 80% 8%, rgba(167,139,250,.16), transparent 70%),
    radial-gradient(30% 40% at 60% 30%, rgba(92,225,230,.07), transparent 70%);
  filter: blur(8px); animation: drift 40s ease-in-out infinite alternate;
}
@keyframes drift { 0% { transform: translate3d(-1.5%, -1%, 0) scale(1); } 100% { transform: translate3d(2%, 1.5%, 0) scale(1.05); } }
.lp-grid {
  position: fixed; inset: 0; z-index: 0; pointer-events: none; opacity: .5;
  background-image: linear-gradient(var(--line) 1px, transparent 1px), linear-gradient(90deg, var(--line) 1px, transparent 1px);
  background-size: 64px 64px;
  -webkit-mask-image: radial-gradient(120% 80% at 50% 0%, #000 35%, transparent 78%);
  mask-image: radial-gradient(120% 80% at 50% 0%, #000 35%, transparent 78%);
}
.lp > section, .lp > header, .lp > footer { position: relative; z-index: 1; }

/* ---------- nav ---------- */
.lp-nav { position: fixed; top: 0; left: 0; right: 0; z-index: 50; transition: background .3s, border-color .3s, backdrop-filter .3s; border-bottom: 1px solid transparent; }
.lp-nav--solid { background: rgba(8,8,10,.72); backdrop-filter: blur(14px); border-color: var(--line); }
.lp-nav__in { max-width: 1200px; margin: 0 auto; padding: 16px 24px; display: flex; align-items: center; gap: 32px; }
.lp-brand { display: inline-flex; align-items: center; gap: 9px; text-decoration: none; }
.lp-brand__mark { display: grid; place-items: center; width: 28px; height: 28px; border-radius: 8px; font-family: var(--disp); font-weight: 700; color: #fff; background: linear-gradient(140deg, var(--indigo), var(--iris)); box-shadow: 0 4px 16px rgba(110,123,255,.4); }
.lp-brand__name { font-family: var(--disp); font-weight: 600; letter-spacing: .02em; color: var(--txt); }
.lp-nav__links { display: flex; gap: 26px; margin-right: auto; }
.lp-nav__links a { position: relative; color: var(--mist); text-decoration: none; font-size: 14px; transition: color .18s; }
.lp-nav__links a::after { content: ''; position: absolute; left: 0; right: 100%; bottom: -4px; height: 1.5px; border-radius: 2px; background: linear-gradient(90deg, var(--indigo), var(--iris)); transition: right .28s cubic-bezier(.22,1,.36,1); }
.lp-nav__links a:hover { color: var(--txt); }
.lp-nav__links a:hover::after { right: 0; }
.lp-nav__cta { display: flex; align-items: center; gap: 16px; }
.lp-link-quiet { color: var(--mist); text-decoration: none; font-size: 14px; transition: color .18s; }
.lp-link-quiet:hover { color: var(--txt); }

/* ---------- buttons ---------- */
.lp-btn { display: inline-flex; align-items: center; justify-content: center; gap: 8px; border-radius: 10px; font-size: 14px; font-weight: 500; padding: 9px 16px; text-decoration: none; transition: transform .16s, box-shadow .2s, background .2s, border-color .2s; border: 1px solid transparent; white-space: nowrap; }
.lp-btn--lg { padding: 13px 22px; font-size: 15px; border-radius: 12px; }
.lp-btn--solid { color: #fff; background: linear-gradient(140deg, var(--indigo), #5a67e8); box-shadow: 0 8px 24px -8px rgba(110,123,255,.7), inset 0 1px 0 rgba(255,255,255,.18); }
.lp-btn--solid:hover { transform: translateY(-2px); box-shadow: 0 14px 32px -8px rgba(110,123,255,.85), inset 0 1px 0 rgba(255,255,255,.22); }
.lp-btn--ghost { color: var(--txt); background: rgba(255,255,255,.03); border-color: var(--line); }
.lp-btn--ghost:hover { background: rgba(255,255,255,.07); border-color: #2c2e38; transform: translateY(-2px); }

/* ---------- hero ---------- */
.lp-hero { max-width: 1200px; margin: 0 auto; padding: 150px 24px 90px; display: grid; grid-template-columns: 1.05fr 1.15fr; gap: 56px; align-items: center; }
.lp-eyebrow { font-family: var(--mono); font-size: 11.5px; letter-spacing: .18em; color: var(--indigo); font-weight: 500; }
.lp-h1 { font-family: var(--disp); font-weight: 700; font-size: clamp(40px, 6vw, 68px); line-height: 1.02; letter-spacing: -.025em; margin: 18px 0 0; }
.lp-dot { color: var(--indigo); }
.lp-sub { margin: 22px 0 0; max-width: 30rem; color: var(--mist); font-size: 17px; line-height: 1.6; }
.lp-hero__actions { display: flex; gap: 14px; margin-top: 30px; flex-wrap: wrap; }
.lp-hero__meta { display: flex; gap: 22px; margin-top: 26px; color: var(--faint); font-size: 13px; }
.lp-hero__meta span { display: inline-flex; align-items: center; gap: 8px; }
.lp-pulse { width: 8px; height: 8px; border-radius: 50%; background: var(--live); box-shadow: 0 0 0 0 rgba(92,225,230,.6); animation: pulse 2s infinite; }
@keyframes pulse { 0% { box-shadow: 0 0 0 0 rgba(92,225,230,.55); } 70% { box-shadow: 0 0 0 9px rgba(92,225,230,0); } 100% { box-shadow: 0 0 0 0 rgba(92,225,230,0); } }

/* ---------- signature: work surface ---------- */
.lp-stage { position: relative; }
.lp-stage__glow { position: absolute; inset: 8% 4%; z-index: -1; background: radial-gradient(60% 60% at 50% 40%, rgba(110,123,255,.28), transparent 72%); filter: blur(40px); }
.lp-window { border: 1px solid var(--line); border-radius: 16px; background: linear-gradient(180deg, #0E0F14, #0A0B0F); box-shadow: 0 40px 80px -32px rgba(0,0,0,.8), inset 0 1px 0 rgba(255,255,255,.04); overflow: hidden; transform: perspective(1600px) rotateX(4deg) rotateY(-6deg); }
.lp-window__bar { display: flex; align-items: center; gap: 7px; padding: 11px 14px; border-bottom: 1px solid var(--line); background: rgba(255,255,255,.015); }
.lp-tl, .lp-window__bar .lp-tl, .lp-window__bar > span:nth-child(-n+3) { width: 10px; height: 10px; border-radius: 50%; background: #2a2c36; }
.lp-window__path { margin-left: 10px; font-family: var(--mono); font-size: 11px; color: var(--faint); }
.lp-presence { margin-left: auto; display: inline-flex; }
.lp-presence i { width: 18px; height: 18px; border-radius: 50%; background: var(--a); border: 2px solid #0A0B0F; margin-left: -6px; box-shadow: 0 0 8px -1px var(--a); }
.lp-board { position: relative; display: grid; grid-template-columns: repeat(3, 1fr); gap: 12px; padding: 16px; min-height: 280px; }
.lp-col__h { font-size: 11.5px; color: var(--mist); margin-bottom: 10px; display: flex; gap: 7px; align-items: center; }
.lp-col__h b { color: var(--faint); font-weight: 500; }
.lp-card { position: relative; background: #13141A; border: 1px solid var(--line); border-radius: 9px; padding: 10px 11px; margin-bottom: 9px; }
.lp-card p { font-size: 12.5px; color: #d3d5db; margin: 6px 0 0; line-height: 1.35; }
.lp-card--dim { opacity: .62; }
.lp-card--ghost { background: transparent; border-style: dashed; height: 42px; }
.lp-key { font-family: var(--mono); font-size: 10.5px; color: var(--indigo); }
.lp-st { position: absolute; top: 11px; right: 11px; width: 8px; height: 8px; border-radius: 50%; }
.lp-st--gray { background: #3a3d48; }
.lp-st--iris { background: var(--iris); box-shadow: 0 0 8px -1px var(--iris); }
.lp-st--live { background: var(--live); box-shadow: 0 0 8px -1px var(--live); }

/* cards settle in on load (stagger by --d), then quiet */
.lp-card { opacity: 0; transform: translateY(8px); animation: cardIn .6s cubic-bezier(.22,1,.36,1) forwards; animation-delay: calc(var(--d, 0) * 90ms + 520ms); }
@keyframes cardIn { to { opacity: 1; transform: none; } }
.lp-card--dim { animation-name: cardInDim; }
@keyframes cardInDim { to { opacity: .6; transform: none; } }
/* the active card breathes a soft accent ring — single quiet focal point */
.lp-card--live { border-color: rgba(110,123,255,.32); }
.lp-card--live::after { content: ''; position: absolute; inset: -1px; border-radius: 9px; box-shadow: 0 0 0 1px rgba(110,123,255,.35), 0 0 24px -6px rgba(110,123,255,.5); animation: breathe 3.6s ease-in-out infinite; }
@keyframes breathe { 0%, 100% { opacity: .35; } 50% { opacity: 1; } }
/* slow light sweep across the board */
.lp-scan { position: absolute; left: 0; right: 0; top: 0; height: 64px; pointer-events: none; background: linear-gradient(180deg, transparent, rgba(110,123,255,.10), transparent); animation: scan 7s cubic-bezier(.5,0,.5,1) infinite; }
@keyframes scan { 0% { transform: translateY(-64px); opacity: 0; } 12%, 70% { opacity: 1; } 100% { transform: translateY(300px); opacity: 0; } }

/* ---------- proof ---------- */
.lp-proof { max-width: 1100px; margin: 10px auto 0; padding: 0 24px; display: grid; grid-template-columns: repeat(4, 1fr); gap: 1px; background: var(--line); border: 1px solid var(--line); border-radius: 16px; overflow: hidden; }
.lp-proof__cell { background: var(--ink); padding: 26px 22px; display: flex; flex-direction: column; gap: 5px; }
.lp-proof__k { font-family: var(--mono); font-size: 11px; letter-spacing: .12em; color: var(--faint); text-transform: uppercase; }
.lp-proof__v { font-family: var(--disp); font-size: 28px; font-weight: 600; letter-spacing: -.02em; background: linear-gradient(120deg, #fff, var(--iris)); -webkit-background-clip: text; background-clip: text; color: transparent; }
.lp-proof__d { font-size: 13px; color: var(--mist); }

/* ---------- generic section ---------- */
.lp-section { max-width: 1200px; margin: 0 auto; padding: 110px 24px 0; scroll-margin-top: 88px; }
.lp-shead { text-align: center; max-width: 40rem; margin: 0 auto 52px; }
.lp-h2 { font-family: var(--disp); font-weight: 600; font-size: clamp(28px, 4vw, 44px); line-height: 1.08; letter-spacing: -.025em; margin: 14px 0 0; }
.lp-lead { color: var(--mist); font-size: 17px; line-height: 1.6; margin: 16px 0 0; }
.lp-shead .lp-lead { margin-left: auto; margin-right: auto; }

/* ---------- bento ---------- */
.lp-bento { display: grid; grid-template-columns: repeat(3, 1fr); gap: 16px; }
.lp-fcard { position: relative; border: 1px solid var(--line); border-radius: 16px; padding: 26px; background: linear-gradient(180deg, rgba(255,255,255,.02), transparent); overflow: hidden; transition: border-color .25s, transform .25s, background .25s; }
.lp-fcard::after { content: ''; position: absolute; inset: 0; border-radius: 16px; padding: 1px; background: linear-gradient(140deg, rgba(110,123,255,.5), transparent 40%); -webkit-mask: linear-gradient(#000 0 0) content-box, linear-gradient(#000 0 0); -webkit-mask-composite: xor; mask-composite: exclude; opacity: 0; transition: opacity .3s; }
.lp-fcard:hover { transform: translateY(-4px); background: linear-gradient(180deg, rgba(255,255,255,.045), transparent); }
.lp-fcard:hover::after { opacity: 1; }
.lp-fcard__tag { font-family: var(--mono); font-size: 11px; letter-spacing: .14em; color: var(--indigo); }
.lp-fcard__t { font-family: var(--disp); font-weight: 600; font-size: 20px; letter-spacing: -.01em; margin: 14px 0 0; }
.lp-fcard__b { color: var(--mist); font-size: 14.5px; line-height: 1.55; margin: 10px 0 0; }
.lp-fcard__arrow { position: absolute; top: 26px; right: 26px; color: var(--faint); transition: transform .25s, color .25s; }
.lp-fcard:hover .lp-fcard__arrow { color: var(--indigo); transform: translate(3px, -3px); }

/* ---------- split deep-dives ---------- */
.lp-split { max-width: 1120px; margin: 0 auto; padding: 110px 24px 0; display: grid; grid-template-columns: 1fr 1fr; gap: 56px; align-items: center; }
.lp-split--rev .lp-split__copy { order: 2; }
.lp-checks { list-style: none; padding: 0; margin: 24px 0 0; display: flex; flex-direction: column; gap: 12px; }
.lp-checks li { position: relative; padding-left: 28px; color: var(--secondary, #b9bcc6); font-size: 15px; }
.lp-checks li::before { content: '✓'; position: absolute; left: 0; top: -1px; width: 18px; height: 18px; border-radius: 6px; display: grid; place-items: center; font-size: 11px; color: var(--live); background: rgba(92,225,230,.12); }

.lp-split__art { border: 1px solid var(--line); border-radius: 18px; background: linear-gradient(180deg, #0E0F14, #0A0B0F); padding: 28px; box-shadow: 0 30px 60px -30px rgba(0,0,0,.8); }
/* rule card */
.lp-rule { display: flex; flex-direction: column; gap: 12px; }
.lp-rule__row { display: flex; align-items: center; gap: 12px; }
/* строки правила «исполняются» каскадом, когда блок появляется */
.reveal .lp-rule__row { opacity: 0; transform: translateX(-10px); }
.reveal-in .lp-rule__row { animation: ruleIn .5s cubic-bezier(.22,1,.36,1) forwards; animation-delay: calc(var(--r, 0) * 140ms + 200ms); }
@keyframes ruleIn { to { opacity: 1; transform: none; } }
.lp-rule__when, .lp-rule__then { font-family: var(--mono); font-size: 11px; width: 42px; color: var(--faint); }
.lp-rule__then { color: var(--indigo); }
.lp-chip { background: #14151B; border: 1px solid var(--line); border-radius: 9px; padding: 9px 13px; font-size: 13.5px; color: #d3d5db; }
.lp-chip b { color: #fff; font-weight: 600; }
.lp-chip--act { border-color: rgba(110,123,255,.3); background: rgba(110,123,255,.06); }
.lp-rule__line { width: 2px; height: 18px; margin-left: 21px; border-radius: 2px; background: var(--line); position: relative; overflow: hidden; }
.lp-rule__line::after { content: ''; position: absolute; inset: 0; background: linear-gradient(180deg, transparent, var(--indigo), transparent); animation: flowDown 1.8s ease-in-out infinite; }
@keyframes flowDown { 0% { transform: translateY(-100%); } 100% { transform: translateY(100%); } }
/* sync diagram */
.lp-sync { display: grid; grid-template-columns: 1fr auto 1fr; align-items: center; gap: 10px; }
.lp-sync__node { display: flex; flex-direction: column; align-items: center; gap: 8px; }
.lp-sync__node--r { }
.lp-avatar { width: 40px; height: 40px; border-radius: 12px; display: grid; place-items: center; font-family: var(--disp); font-weight: 600; color: #fff; background: var(--a); box-shadow: 0 0 22px -6px var(--a); }
.lp-sync__lbl { font-size: 12px; color: var(--mist); }
.lp-sync__hub { width: 44px; height: 44px; border-radius: 12px; display: grid; place-items: center; background: #14151B; border: 1px solid var(--line); font-size: 18px; animation: hubPulse 1.8s ease-in-out infinite; }
@keyframes hubPulse { 0%, 100% { box-shadow: 0 0 0 0 rgba(110,123,255,0); } 50% { box-shadow: 0 0 18px -2px rgba(110,123,255,.55); } }
.lp-sync__beam { position: relative; height: 2px; border-radius: 2px; background: var(--line); overflow: hidden; }
.lp-sync__beam::after { content: ''; position: absolute; top: 0; left: 0; height: 100%; width: 45%; border-radius: 2px; background: linear-gradient(90deg, transparent, var(--indigo), transparent); animation: beamRun 1.8s cubic-bezier(.5,0,.5,1) infinite; }
.lp-sync__beam--2::after { animation-delay: .9s; background: linear-gradient(90deg, transparent, var(--live), transparent); }
@keyframes beamRun { 0% { transform: translateX(-120%); } 100% { transform: translateX(340%); } }

/* ---------- workflow steps ---------- */
.lp-steps { display: grid; grid-template-columns: repeat(3, 1fr); gap: 18px; }
.lp-step { position: relative; border: 1px solid var(--line); border-radius: 16px; padding: 28px; background: linear-gradient(180deg, rgba(255,255,255,.02), transparent); }
.lp-step__key { font-family: var(--mono); font-size: 12px; color: var(--indigo); }
.lp-step__t { font-family: var(--disp); font-weight: 600; font-size: 22px; margin: 12px 0 0; }
.lp-step__d { color: var(--mist); font-size: 14.5px; line-height: 1.55; margin: 10px 0 0; }
.lp-step__arc { position: absolute; top: 50%; right: -13px; z-index: 2; color: var(--faint); font-size: 18px; }

/* ---------- pricing ---------- */
.lp-plans { display: grid; grid-template-columns: repeat(3, 1fr); gap: 18px; align-items: start; }
.lp-plan { position: relative; border: 1px solid var(--line); border-radius: 18px; padding: 30px; background: linear-gradient(180deg, rgba(255,255,255,.02), transparent); display: flex; flex-direction: column; }
.lp-plan--accent { border-color: rgba(110,123,255,.4); background: linear-gradient(180deg, rgba(110,123,255,.08), transparent); box-shadow: 0 30px 60px -34px rgba(110,123,255,.6); }
.lp-plan__badge { position: absolute; top: -11px; left: 30px; font-family: var(--mono); font-size: 10.5px; letter-spacing: .1em; text-transform: uppercase; color: #fff; background: linear-gradient(140deg, var(--indigo), var(--iris)); padding: 4px 10px; border-radius: 6px; }
.lp-plan__name { font-family: var(--disp); font-weight: 600; font-size: 18px; }
.lp-plan__price { display: flex; align-items: baseline; gap: 8px; margin: 14px 0 0; }
.lp-plan__amt { font-family: var(--disp); font-weight: 700; font-size: 38px; letter-spacing: -.03em; }
.lp-plan__unit { color: var(--faint); font-size: 13px; }
.lp-plan__blurb { color: var(--mist); font-size: 14px; margin: 8px 0 0; }
.lp-plan__feats { list-style: none; padding: 0; margin: 20px 0 24px; display: flex; flex-direction: column; gap: 11px; flex: 1; }
.lp-plan__feats li { display: flex; align-items: center; gap: 10px; font-size: 14px; color: #c2c5cd; }
.lp-tick { color: var(--live); font-size: 12px; }

/* ---------- cta ---------- */
.lp-cta { max-width: 1120px; margin: 130px auto 0; padding: 0 24px; }
.lp-cta__panel { position: relative; overflow: hidden; text-align: center; border: 1px solid var(--line); border-radius: 24px; padding: 72px 24px; background: linear-gradient(180deg, #0E0F15, #0A0B0F); }
.lp-cta__glow { position: absolute; inset: auto 0 -60% 0; height: 120%; background: radial-gradient(50% 60% at 50% 100%, rgba(110,123,255,.35), transparent 70%); pointer-events: none; }
.lp-cta__panel .lp-h2, .lp-cta__panel .lp-lead { position: relative; }
.lp-cta__panel .lp-lead { margin-left: auto; margin-right: auto; max-width: 32rem; }
.lp-cta__panel .lp-btn { position: relative; margin-top: 28px; }

/* ---------- footer ---------- */
.lp-footer { max-width: 1200px; margin: 110px auto 0; padding: 48px 24px 40px; border-top: 1px solid var(--line); }
.lp-footer__in { display: grid; grid-template-columns: 1.4fr 2fr; gap: 40px; }
.lp-footer__brand p { color: var(--mist); font-size: 14px; margin: 14px 0 0; max-width: 22rem; line-height: 1.55; }
.lp-footer__cols { display: grid; grid-template-columns: repeat(3, 1fr); gap: 24px; }
.lp-footer__cols h4 { font-size: 12px; text-transform: uppercase; letter-spacing: .1em; color: var(--faint); margin: 0 0 14px; }
.lp-footer__cols a { display: block; color: var(--mist); text-decoration: none; font-size: 14px; margin-bottom: 10px; transition: color .18s; }
.lp-footer__cols a:hover { color: var(--txt); }
.lp-footer__base { display: flex; justify-content: space-between; margin-top: 40px; padding-top: 22px; border-top: 1px solid var(--line); color: var(--faint); font-size: 13px; }
.lp-mono { font-family: var(--mono); }

/* ---------- motion: load + reveal ---------- */
.lp-load { opacity: 0; transform: translateY(16px); animation: rise .7s cubic-bezier(.22,1,.36,1) forwards; animation-delay: calc(var(--i, 0) * 90ms + 120ms); }
@keyframes rise { to { opacity: 1; transform: none; } }
.reveal { opacity: 0; transform: translateY(18px); filter: blur(7px); transition: opacity .8s cubic-bezier(.22,1,.36,1), transform .8s cubic-bezier(.22,1,.36,1), filter .8s cubic-bezier(.22,1,.36,1); transition-delay: var(--reveal-delay, 0ms); will-change: opacity, transform, filter; }
.reveal-in { opacity: 1; transform: none; filter: blur(0); }

/* ---------- responsive ---------- */
@media (max-width: 900px) {
  .lp-hero { grid-template-columns: 1fr; padding-top: 122px; gap: 40px; }
  .lp-stage { order: 2; }
  .lp-bento, .lp-steps, .lp-plans { grid-template-columns: 1fr; }
  .lp-split, .lp-split--rev .lp-split__copy { grid-template-columns: 1fr; }
  .lp-split { grid-template-columns: 1fr; }
  .lp-split--rev .lp-split__copy { order: 0; }
  .lp-proof { grid-template-columns: repeat(2, 1fr); }
  .lp-footer__in { grid-template-columns: 1fr; }
  .lp-step__arc { display: none; }
  .lp-nav__links { display: none; }
}
@media (prefers-reduced-motion: reduce) {
  .lp-load, .reveal, .lp-card, .reveal-in .lp-rule__row { animation: none !important; opacity: 1 !important; transform: none !important; transition: none !important; filter: none !important; }
  .reveal .lp-rule__row { opacity: 1 !important; transform: none !important; }
  .lp-card--dim { opacity: .6 !important; }
  .lp-scan, .lp-card--live::after, .lp-aurora, .lp-pulse, .lp-sync__hub, .lp-sync__beam::after, .lp-rule__line::after { animation: none !important; }
  .lp-scan, .lp-rule__line::after, .lp-sync__beam::after { display: none; }
}
</style>
