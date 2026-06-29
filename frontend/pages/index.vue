<script setup lang="ts">
import { isAuthed } from '~/lib/token'

onMounted(() => { if (isAuthed()) navigateTo('/my/assigned') })

const DESC = 'Issues, docs, chat, calendar and meetings in one fast workspace. Local-first and realtime — every change lands instantly for the whole team.'
useHead({
  title: 'PMS — the workspace that runs at your speed',
  meta: [
    { name: 'description', content: DESC },
    { property: 'og:type', content: 'website' },
    { property: 'og:title', content: 'PMS — the workspace that runs at your speed' },
    { property: 'og:description', content: DESC },
    { property: 'og:image', content: '/og.svg' },
    { name: 'twitter:card', content: 'summary_large_image' },
    { name: 'twitter:title', content: 'PMS — the workspace that runs at your speed' },
    { name: 'twitter:description', content: DESC },
    { name: 'twitter:image', content: '/og.svg' },
  ],
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
  menuOpen.value = false
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

// ── product UI data (это веб-приложение, не CLI) ──
const nav = ['Inbox', 'My work', 'Board', 'Calendar', 'Docs', 'Chat']
const issues = [
  { k: 'PMS-204', t: 'Refactor billing flow', s: 'prog', p: 'High', pc: '#ff9e64', a: 'M', c: '#bb9af7' },
  { k: 'PMS-201', t: 'Calendar drag-resize', s: 'done', p: 'Med', pc: '#7dcfff', a: 'L', c: '#7dcfff' },
  { k: 'PMS-198', t: 'Automations → post to channel', s: 'prog', p: 'High', pc: '#ff9e64', a: 'A', c: '#9ece6a' },
  { k: 'PMS-195', t: 'iCal feed line-folding', s: 'todo', p: 'Low', pc: '#6b7089', a: 'Y', c: '#ff9e64' },
  { k: 'PMS-190', t: 'Realtime reconnect backoff', s: 'todo', p: 'Med', pc: '#7dcfff', a: 'M', c: '#bb9af7' },
]
const acts = [
  { tag: 'moved', c: 'b', who: 'M', wc: '#bb9af7', m: 'Maya moved PMS-204 → In review' },
  { tag: 'auto', c: 'p', who: '⚙', wc: '#7aa2f7', m: 'Rule fired · assigned @leo · +needs-review' },
  { tag: 'chat', c: 'c', who: 'A', wc: '#9ece6a', m: 'Ada: shipping the new grid 🎉' },
  { tag: 'done', c: 'g', who: '✓', wc: '#9ece6a', m: 'PMS-201 shipped · cycle 12' },
]
const liveRow = ref(0)
const actLive = ref(reduceMotion() ? acts.length - 1 : 0)
let tick: any
onMounted(() => {
  if (reduceMotion()) { liveRow.value = -1; return }
  let n = 0
  tick = setInterval(() => { n++; liveRow.value = n % issues.length; actLive.value = n % acts.length }, 2000)
})
onBeforeUnmount(() => clearInterval(tick))

const features = [
  { f: 'issues.ts', d: 'Sub-issues, relations, labels, cycles. Board or list, your saved views.' },
  { f: 'docs.md', d: 'Specs that live next to the work. Markdown, mentions, embeds.' },
  { f: 'chat.sock', d: 'Threaded channels, reactions, unread — no fourth tab.' },
  { f: 'calendar.ics', d: 'Month, week, day. Drag to schedule, see who’s free.' },
  { f: 'meet.sfu', d: 'Huddle from any channel. Video built in.' },
  { f: 'time.log', d: 'Track hours, report by person/task/day, export.' },
]
// pricing toggle (annual −20%)
const billing = ref<'monthly' | 'annual'>('monthly')
const plans = [
  { name: 'free', m: 0, blurb: 'Get the team started.', feats: ['10 teammates', 'Issues · board · calendar', 'Realtime sync'], cta: 'start free', accent: false },
  { name: 'pro', m: 8, blurb: 'Ship every week.', feats: ['Unlimited members', 'Automations & cycles', 'Time tracking & reports', 'Meetings & huddles'], cta: 'start trial', accent: true },
  { name: 'business', m: 16, blurb: 'Scale the org.', feats: ['Everything in pro', 'Roles & permissions', 'Audit · SSO', 'Priority support'], cta: 'start trial', accent: false },
]
function priceOf(p: { m: number }) {
  if (p.m === 0) return { amt: '$0', unit: '' }
  const v = billing.value === 'annual' ? Math.round(p.m * 0.8) : p.m
  return { amt: '$' + v, unit: billing.value === 'annual' ? '/user·mo · billed yearly' : '/user·mo' }
}

// mobile nav
const menuOpen = ref(false)

// ⌘K command palette demo (real, filterable)
const kq = ref('')
const kcmds = [
  { i: '⌘', t: 'Create issue', s: 'C' },
  { i: '#', t: 'Go to board', s: 'G B' },
  { i: '◷', t: 'Start a timer', s: 'T' },
  { i: '⚡', t: 'New automation', s: '' },
  { i: '◰', t: 'Open calendar', s: 'G C' },
  { i: '✎', t: 'New doc', s: '' },
  { i: '☎', t: 'Start a huddle', s: '' },
  { i: '@', t: 'Assign to me', s: 'I' },
]
const kfiltered = computed(() => {
  const q = kq.value.trim().toLowerCase()
  return q ? kcmds.filter((c) => c.t.toLowerCase().includes(q)) : kcmds
})

const faqs = [
  { q: 'Is there really a free plan?', a: 'Yes — up to 10 teammates with issues, boards, calendar and realtime sync. No card, no trial clock.' },
  { q: 'Can we migrate from Jira or Linear?', a: 'Import issues, statuses and assignees from CSV today; native importers are on the way. Your keys and history come along.' },
  { q: 'How does realtime work offline?', a: 'PMS is local-first: the app reads a local store, so it keeps working when the network drops and resyncs deltas automatically on reconnect.' },
  { q: 'Where does our data live?', a: 'On our cloud by default, isolated per workspace with row-level security. Prefer your own servers? It self-hosts as a single binary with Postgres + S3.' },
  { q: 'Do you bill per seat?', a: 'Pro and Business are per active user, monthly or annual (save 20%). Free is flat. Cancel anytime.' },
]

const cmp = [
  { f: 'Issues, docs, chat, calendar & meetings in one', pms: true, jira: false, linear: false },
  { f: 'Local-first, instant realtime', pms: true, jira: false, linear: 'part' },
  { f: 'Built-in video huddles', pms: true, jira: false, linear: false },
  { f: 'Time tracking & reports', pms: true, jira: 'part', linear: false },
  { f: 'Self-hostable', pms: true, jira: true, linear: false },
  { f: 'Free tier', pms: true, jira: true, linear: true },
]
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
        <button class="t-burger" :aria-expanded="menuOpen" aria-label="Menu" @click="menuOpen = !menuOpen">{{ menuOpen ? '✕' : '≡' }}</button>
      </div>
      <div v-if="menuOpen" class="t-mobile">
        <a href="#features">./features</a>
        <a href="#flow">./how</a>
        <a href="#pricing">./pricing</a>
        <a href="/login" class="t-quiet">sign in</a>
        <a href="/login" class="t-btn t-btn--solid">[ start free ]</a>
      </div>
    </header>

    <!-- HERO -->
    <section class="t-hero">
      <div class="t-hero__copy">
        <span class="t-eyebrow t-up" style="--i:0"># local-first · realtime · keyboard-driven</span>
        <h1 class="t-h1 t-up" style="--i:1">The workspace<br />that runs at<br /><span class="t-grad">your speed</span><span class="t-cur">▮</span></h1>
        <p class="t-sub t-up" style="--i:2">
          Issues, docs, chat, calendar and meetings in one fast web app.
          Every change lands instantly — for the whole team, in real time.
        </p>
        <div class="t-actions t-up" style="--i:3">
          <a href="/login" class="t-btn t-btn--solid t-btn--lg">[ start free ]</a>
          <a href="#features" class="t-btn t-btn--lg">[ take the tour ]</a>
        </div>
        <p class="t-note t-up" style="--i:4"><span class="t-ok">✓</span> no credit card &nbsp; <span class="t-ok">✓</span> whole team in minutes</p>
      </div>

      <!-- SIGNATURE: the actual web app -->
      <div class="t-stage t-up" style="--i:3" aria-hidden="true">
        <div class="t-app">
          <div class="t-app__bar">
            <span class="t-tl t-tl--r"></span><span class="t-tl t-tl--y"></span><span class="t-tl t-tl--g"></span>
            <span class="t-url"><span class="t-url__lock">🔒</span>app.pms.dev/engineering</span>
          </div>
          <div class="t-app__body">
            <aside class="t-side">
              <div class="t-side__ws">~/ acme</div>
              <a v-for="(n, i) in nav" :key="n" class="t-side__i" :class="{ 'is-active': i === 2 }">{{ n }}</a>
            </aside>
            <div class="t-main">
              <div class="t-main__h"><span>Board · Active</span><span class="t-main__new">+ New</span></div>
              <div class="t-rows">
                <div v-for="(r, i) in issues" :key="r.k" class="t-row" :class="{ 'is-live': liveRow === i }">
                  <span class="t-row__s" :class="'s-' + r.s"></span>
                  <span class="t-row__k">{{ r.k }}</span>
                  <span class="t-row__t">{{ r.t }}</span>
                  <span class="t-row__p" :style="{ color: r.pc }">{{ r.p }}</span>
                  <span class="t-row__a" :style="{ '--a': r.c }">{{ r.a }}</span>
                  <span v-if="liveRow === i" class="t-row__edit">editing…</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>

    <!-- STACK STRIP -->
    <section class="t-strip" v-reveal>
      <span class="t-strip__lbl">// one workspace, the whole stack</span>
      <div class="t-strip__tags">
        <span v-for="x in ['issues', 'boards', 'docs', 'chat', 'calendar', 'meetings', 'automations', 'time']" :key="x">{{ x }}</span>
      </div>
    </section>

    <!-- FEATURES -->
    <section id="features" class="t-sec">
      <header class="t-shead" v-reveal>
        <span class="t-eyebrow"># everything the work needs — six tools, one tab</span>
        <h2 class="t-h2"><span class="t-prompt">$</span> pms ls <span class="t-grad">~/workspace</span></h2>
      </header>
      <div class="t-grid">
        <article v-for="(f, i) in features" :key="f.f" class="t-card" v-reveal="i * 60">
          <div class="t-card__h"><span class="t-card__dot"></span>{{ f.f }}</div>
          <p class="t-card__d">{{ f.d }}</p>
          <span class="t-card__run">open →</span>
        </article>
      </div>
    </section>

    <!-- SPOTLIGHT: automations (in-app rule builder) -->
    <section class="t-spot">
      <div class="t-spot__copy" v-reveal>
        <span class="t-eyebrow"># automations — build it once, it runs forever</span>
        <h2 class="t-h2"><span class="t-prompt">$</span> pms automate <span class="t-grad">--forever</span></h2>
        <p class="t-lead">Point and click the rule in the app. PMS applies it server-side on every event — reassign, re-label, set status, ping a channel. No code, no glue.</p>
        <ul class="t-checks">
          <li>Triggers on create, status, owner, priority</li>
          <li>Chain actions in a single rule</li>
          <li>Scope to a team or the whole org</li>
        </ul>
      </div>
      <div class="t-spot__art" v-reveal="120">
        <div class="t-panel">
          <div class="t-panel__h">Automation · New rule</div>
          <div class="t-rule">
            <div class="t-rule__row" style="--r:0"><span class="t-rule__k">WHEN</span><span class="t-pill">status → <b>In review</b></span></div>
            <div class="t-rule__pipe"></div>
            <div class="t-rule__row" style="--r:1"><span class="t-rule__k t-rule__k--then">THEN</span><span class="t-pill t-pill--act">assign <b>@reviewer</b></span></div>
            <div class="t-rule__row" style="--r:2"><span class="t-rule__k t-rule__k--then">THEN</span><span class="t-pill t-pill--act">add label <b>needs-review</b></span></div>
            <div class="t-rule__row" style="--r:3"><span class="t-rule__k t-rule__k--then">THEN</span><span class="t-pill t-pill--act">post to <b>#reviews</b></span></div>
            <button class="t-rule__save" type="button">Save rule</button>
          </div>
        </div>
      </div>
    </section>

    <!-- SPOTLIGHT: realtime (in-app activity) -->
    <section class="t-spot t-spot--rev">
      <div class="t-spot__copy" v-reveal>
        <span class="t-eyebrow"># realtime — everyone on the same page</span>
        <h2 class="t-h2"><span class="t-prompt">$</span> pms sync <span class="t-grad">--live</span></h2>
        <p class="t-lead">Local-first store, live updates over WebSocket. Every edit lands for the whole team the instant it happens — and the app keeps working when the network blinks.</p>
        <ul class="t-checks">
          <li>Optimistic edits, zero perceived latency</li>
          <li>Live presence across the team</li>
          <li>Reconnects on its own after a drop</li>
        </ul>
      </div>
      <div class="t-spot__art" v-reveal="120">
        <div class="t-panel">
          <div class="t-panel__h">Activity<span class="t-online"><i></i><i></i><i></i> 3 online</span></div>
          <ul class="t-act">
            <li v-for="(a, i) in acts" :key="a.m" class="t-act__row" :class="{ 'is-live': actLive === i }">
              <span class="t-act__av" :style="{ '--a': a.wc }">{{ a.who }}</span>
              <span class="t-act__m">{{ a.m }}</span>
              <span class="t-act__tag" :class="'lt-' + a.c">{{ a.tag }}</span>
            </li>
          </ul>
        </div>
      </div>
    </section>

    <!-- ⌘K palette -->
    <section class="t-sec">
      <div class="t-kband">
        <div class="t-kband__copy" v-reveal>
          <span class="t-eyebrow"># keyboard-first</span>
          <h2 class="t-h2">Hit <span class="t-grad">⌘K</span>, do anything.</h2>
          <p class="t-lead">Every action is a keystroke away. Create, assign, jump, automate — without lifting your hands off the keys. Try the filter →</p>
        </div>
        <div class="t-kbar" v-reveal="100">
          <div class="t-kbar__in">
            <div class="t-kbar__search"><span class="t-dollar">⌘K</span><input v-model="kq" class="t-kbar__input" placeholder="type a command…" aria-label="Command palette demo" /></div>
            <ul class="t-kbar__list">
              <li v-for="c in kfiltered" :key="c.t" class="t-kbar__item">
                <span class="t-kbar__i">{{ c.i }}</span><span class="t-kbar__t">{{ c.t }}</span><span v-if="c.s" class="t-kbar__s">{{ c.s }}</span>
              </li>
              <li v-if="!kfiltered.length" class="t-kbar__empty">no matches</li>
            </ul>
          </div>
        </div>
      </div>
    </section>

    <!-- HOW IT WORKS -->
    <section id="flow" class="t-sec">
      <header class="t-shead" v-reveal>
        <span class="t-eyebrow"># the loop — from thought to shipped</span>
        <h2 class="t-h2"><span class="t-prompt">$</span> pms ship <span class="t-grad">--repeat</span></h2>
      </header>
      <div class="t-steps">
        <article v-for="(s, i) in [
          { c: 'capture', t: 'Capture', d: 'Spin up an issue in a keystroke. Tracked before the thought is gone.' },
          { c: 'organize', t: 'Organize', d: 'Drop it in a cycle, a board, a view. The layout is yours to keep.' },
          { c: 'ship', t: 'Ship', d: 'Move it to done — automations fire, the team hears about it, repeat.' },
        ]" :key="s.t" class="t-card t-step" v-reveal="i * 90">
          <div class="t-step__n">{{ String(i + 1).padStart(2, '0') }}</div>
          <h3 class="t-step__t">{{ s.t }}</h3>
          <p class="t-card__d">{{ s.d }}</p>
        </article>
      </div>
    </section>

    <!-- COMPARISON -->
    <section class="t-sec">
      <header class="t-shead" v-reveal>
        <span class="t-eyebrow"># diff — pms vs the rest</span>
        <h2 class="t-h2"><span class="t-prompt">$</span> pms diff <span class="t-grad">--tools</span></h2>
      </header>
      <div class="t-cmp" v-reveal>
        <div class="t-cmp__head"><span></span><span class="t-cmp__me">pms</span><span>Jira</span><span>Linear</span></div>
        <div v-for="r in cmp" :key="r.f" class="t-cmp__row">
          <span class="t-cmp__f">{{ r.f }}</span>
          <span class="t-cmp__c"><i :class="r.pms === true ? 'ok' : r.pms === 'part' ? 'part' : 'no'">{{ r.pms === true ? '✓' : r.pms === 'part' ? '~' : '·' }}</i></span>
          <span class="t-cmp__c"><i :class="r.jira === true ? 'ok' : r.jira === 'part' ? 'part' : 'no'">{{ r.jira === true ? '✓' : r.jira === 'part' ? '~' : '·' }}</i></span>
          <span class="t-cmp__c"><i :class="r.linear === true ? 'ok' : r.linear === 'part' ? 'part' : 'no'">{{ r.linear === true ? '✓' : r.linear === 'part' ? '~' : '·' }}</i></span>
        </div>
      </div>
    </section>

    <!-- FAQ -->
    <section class="t-sec">
      <header class="t-shead" v-reveal>
        <span class="t-eyebrow"># faq</span>
        <h2 class="t-h2"><span class="t-prompt">$</span> pms <span class="t-grad">--help</span></h2>
      </header>
      <div class="t-faq" v-reveal>
        <details v-for="f in faqs" :key="f.q" class="t-faq__i">
          <summary><span class="t-faq__q">{{ f.q }}</span><span class="t-faq__x">+</span></summary>
          <p class="t-faq__a">{{ f.a }}</p>
        </details>
      </div>
    </section>

    <!-- PRICING -->
    <section id="pricing" class="t-sec">
      <header class="t-shead" v-reveal>
        <span class="t-eyebrow"># pricing — plans that scale with you</span>
        <h2 class="t-h2"><span class="t-prompt">$</span> pms install <span class="t-grad">--plan</span></h2>
        <p class="t-lead t-lead--c">Start free. Upgrade when the team grows. Cancel anytime.</p>
        <div class="t-toggle" v-reveal>
          <button :class="{ 'is-on': billing === 'monthly' }" @click="billing = 'monthly'">monthly</button>
          <button :class="{ 'is-on': billing === 'annual' }" @click="billing = 'annual'">annual <span class="t-toggle__save">−20%</span></button>
        </div>
      </header>
      <div class="t-plans">
        <article v-for="(p, i) in plans" :key="p.name" class="t-card t-plan" :class="{ 't-plan--accent': p.accent }" v-reveal="i * 80">
          <span v-if="p.accent" class="t-plan__badge">// popular</span>
          <div class="t-plan__name"><span class="t-prompt">--</span>{{ p.name }}</div>
          <div class="t-plan__price"><span class="t-plan__amt">{{ priceOf(p).amt }}</span><span class="t-plan__unit">{{ priceOf(p).unit }}</span></div>
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
        <span class="t-eyebrow"># ready when you are — bring the team</span>
        <h2 class="t-h2"><span class="t-prompt">$</span> pms <span class="t-grad">login</span><span class="t-cur">▮</span></h2>
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
      <div class="t-foot__base"><span>© 2026 pms</span><span>made for teams that ship</span></div>
    </footer>
  </div>
</template>

<style scoped>
.t {
  --bg: #0b0c12; --panel: #11131b; --line: #20222e;
  --ink: #c5cae0; --bright: #eef0fa; --mut: #6b7089; --faint: #474c61;
  --blue: #7aa2f7; --purple: #bb9af7; --green: #9ece6a; --cyan: #7dcfff; --orange: #ff9e64; --red: #f7768e;
  --grad: linear-gradient(100deg, #7aa2f7, #bb9af7 55%, #7dcfff);
  --m: 'JetBrains Mono', ui-monospace, SFMono-Regular, Menlo, monospace;
  position: relative; background: var(--bg); color: var(--ink); overflow-x: clip; isolation: isolate;
  font-family: var(--m); font-size: 15px; line-height: 1.55;
  background-image: radial-gradient(60% 50% at 80% -5%, rgba(122,162,247,.10), transparent 60%), radial-gradient(50% 40% at 0% 0%, rgba(187,154,247,.08), transparent 60%);
}
.t ::selection { background: rgba(122,162,247,.32); }

.t-scan { position: fixed; inset: 0; z-index: 0; pointer-events: none; opacity: .5; background: repeating-linear-gradient(0deg, rgba(255,255,255,.013) 0 1px, transparent 1px 3px); }
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

/* buttons */
.t-btn { display: inline-flex; align-items: center; justify-content: center; font-family: var(--m); font-weight: 500; font-size: 13.5px; padding: 8px 14px; border: 1px solid var(--line); border-radius: 8px; background: rgba(255,255,255,.02); color: var(--ink); text-decoration: none; transition: border-color .18s, color .18s, background .18s, transform .16s, box-shadow .2s; white-space: nowrap; }
.t-btn:hover { border-color: var(--blue); color: var(--bright); transform: translateY(-2px); }
.t-btn--lg { padding: 12px 20px; font-size: 14px; }
.t-btn--solid { border-color: transparent; color: #0b0c12; font-weight: 700; background: var(--grad); box-shadow: 0 8px 26px -10px rgba(122,162,247,.7); }
.t-btn--solid:hover { color: #0b0c12; transform: translateY(-2px); box-shadow: 0 14px 34px -10px rgba(187,154,247,.8); }

/* hero */
.t-hero { max-width: 1140px; margin: 0 auto; padding: 160px 24px 90px; display: grid; grid-template-columns: .95fr 1.1fr; gap: 56px; align-items: center; }
.t-eyebrow { color: var(--mut); font-size: 13px; }
.t-h1 { font-weight: 700; font-size: clamp(38px, 5.6vw, 62px); line-height: 1.06; letter-spacing: -.02em; margin: 18px 0 0; color: var(--bright); }
.t-grad { background: var(--grad); -webkit-background-clip: text; background-clip: text; color: transparent; white-space: nowrap; }
.t-sub { margin: 22px 0 0; max-width: 32rem; color: var(--mut); font-size: 15px; line-height: 1.7; }
.t-actions { display: flex; gap: 12px; margin-top: 30px; flex-wrap: wrap; }
.t-note { margin-top: 22px; color: var(--faint); font-size: 13px; }
.t-ok { color: var(--green); }

/* app window (web product) */
.t-app { border: 1px solid var(--line); border-radius: 12px; background: linear-gradient(180deg, #0f1118, #0b0c12); box-shadow: 0 40px 80px -36px rgba(0,0,0,.9), inset 0 1px 0 rgba(255,255,255,.04); overflow: hidden; }
.t-app__bar { display: flex; align-items: center; gap: 8px; padding: 10px 13px; border-bottom: 1px solid var(--line); background: rgba(255,255,255,.015); }
.t-tl { width: 11px; height: 11px; border-radius: 50%; }
.t-tl--r { background: #f7768e; } .t-tl--y { background: #e0af68; } .t-tl--g { background: #9ece6a; }
.t-url { margin-left: 10px; display: inline-flex; align-items: center; gap: 7px; font-size: 11.5px; color: var(--mut); background: rgba(255,255,255,.04); border: 1px solid var(--line); border-radius: 7px; padding: 3px 11px; }
.t-url__lock { font-size: 9px; opacity: .7; }
.t-app__body { display: grid; grid-template-columns: 132px 1fr; min-height: 286px; }
.t-side { border-right: 1px solid var(--line); padding: 14px 10px; display: flex; flex-direction: column; gap: 2px; background: rgba(255,255,255,.01); }
.t-side__ws { color: var(--green); font-size: 12px; padding: 4px 8px 10px; }
.t-side__i { padding: 7px 8px; border-radius: 7px; font-size: 12.5px; color: var(--mut); cursor: default; }
.t-side__i.is-active { background: rgba(122,162,247,.12); color: var(--blue); }
.t-main { padding: 14px 16px; }
.t-main__h { display: flex; justify-content: space-between; align-items: center; font-size: 12.5px; color: var(--mut); padding-bottom: 10px; }
.t-main__new { color: var(--green); border: 1px solid var(--line); border-radius: 6px; padding: 3px 9px; }
.t-rows { display: flex; flex-direction: column; }
.t-row { display: flex; align-items: center; gap: 11px; padding: 9px 10px; border-radius: 8px; position: relative; transition: background .4s; }
.t-row.is-live { background: rgba(122,162,247,.09); }
.t-row__s { width: 9px; height: 9px; border-radius: 50%; flex: none; }
.s-todo { background: #565f89; } .s-prog { background: var(--blue); box-shadow: 0 0 8px -1px var(--blue); } .s-done { background: var(--green); box-shadow: 0 0 8px -1px var(--green); }
.t-row__k { color: var(--purple); font-size: 12px; flex: none; }
.t-row__t { color: var(--ink); font-size: 13px; flex: 1; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.t-row__p { font-size: 11.5px; flex: none; }
.t-row__a { flex: none; width: 22px; height: 22px; border-radius: 6px; display: grid; place-items: center; font-size: 11px; font-weight: 600; color: #0b0c12; background: var(--a); }
.t-row__edit { position: absolute; right: 10px; top: -7px; font-size: 10px; color: var(--blue); background: var(--bg); border: 1px solid var(--line); border-radius: 5px; padding: 1px 6px; animation: editPulse 2s ease-in-out infinite; }
@keyframes editPulse { 0%, 100% { opacity: .55; } 50% { opacity: 1; } }
@keyframes blink { 0%, 50% { opacity: 1; } 51%, 100% { opacity: 0; } }

/* stack strip */
.t-strip { max-width: 1140px; margin: 0 auto; padding: 18px 24px 0; text-align: center; }
.t-strip__lbl { color: var(--faint); font-size: 13px; }
.t-strip__tags { display: flex; flex-wrap: wrap; justify-content: center; gap: 10px; margin-top: 16px; }
.t-strip__tags span { font-size: 13px; color: var(--mut); border: 1px solid var(--line); border-radius: 7px; padding: 5px 11px; background: rgba(255,255,255,.02); transition: color .18s, border-color .18s; }
.t-strip__tags span:hover { color: var(--blue); border-color: rgba(122,162,247,.4); }

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
/* команды в узкой колонке spotlight — одной строкой, чуть мельче, без переноса флага */
.t-spot__copy .t-h2 { white-space: nowrap; font-size: clamp(25px, 3.4vw, 36px); }
.t-checks { list-style: none; padding: 0; margin: 24px 0 0; display: flex; flex-direction: column; gap: 12px; }
.t-checks li { position: relative; padding-left: 26px; color: var(--ink); font-size: 14.5px; }
.t-checks li::before { content: '✓'; position: absolute; left: 0; color: var(--green); }

/* in-app panel (rule builder + activity) */
.t-panel { border: 1px solid var(--line); border-radius: 12px; background: linear-gradient(180deg, #0f1118, #0b0c12); box-shadow: 0 30px 70px -40px rgba(0,0,0,.85); overflow: hidden; }
.t-panel__h { display: flex; justify-content: space-between; align-items: center; padding: 12px 16px; border-bottom: 1px solid var(--line); font-size: 12.5px; color: var(--mut); background: rgba(255,255,255,.015); }
.t-rule { padding: 18px 16px; display: flex; flex-direction: column; gap: 11px; }
.rv .t-rule__row { opacity: 0; transform: translateX(-10px); }
.rv-in .t-rule__row { animation: rowIn .5s cubic-bezier(.22,1,.36,1) forwards; animation-delay: calc(var(--r,0) * 130ms + 160ms); }
@keyframes rowIn { to { opacity: 1; transform: none; } }
.t-rule__row { display: flex; align-items: center; gap: 11px; }
.t-rule__k { font-size: 11px; letter-spacing: .1em; width: 42px; text-align: center; color: var(--faint); font-weight: 600; }
.t-rule__k--then { color: var(--purple); }
.t-pill { background: rgba(255,255,255,.05); border: 1px solid var(--line); border-radius: 8px; padding: 8px 12px; font-size: 13px; color: #cdcbd9; }
.t-pill b { color: var(--bright); }
.t-pill--act { border-color: rgba(122,162,247,.32); background: rgba(122,162,247,.1); }
.t-rule__pipe { width: 2px; height: 14px; margin-left: 20px; border-radius: 2px; background: var(--line); }
.t-rule__save { align-self: flex-start; margin-top: 6px; font-family: var(--m); font-size: 12.5px; color: #0b0c12; font-weight: 700; border: none; border-radius: 7px; padding: 7px 14px; background: var(--grad); cursor: pointer; }
/* activity */
.t-online { display: inline-flex; align-items: center; gap: 6px; color: var(--green); font-size: 11.5px; }
.t-online i { width: 6px; height: 6px; border-radius: 50%; background: var(--green); box-shadow: 0 0 6px var(--green); animation: editPulse 2.4s ease-in-out infinite; }
.t-online i:nth-child(2) { animation-delay: .3s; } .t-online i:nth-child(3) { animation-delay: .6s; }
.t-act { list-style: none; margin: 0; padding: 10px 12px; display: flex; flex-direction: column; gap: 3px; }
.t-act__row { display: flex; align-items: center; gap: 11px; padding: 9px 10px; border-radius: 9px; transition: background .4s; }
.t-act__row.is-live { background: rgba(122,162,247,.1); }
.t-act__av { flex: none; width: 26px; height: 26px; border-radius: 8px; display: grid; place-items: center; font-size: 12px; font-weight: 600; color: #0b0c12; background: var(--a); }
.t-act__m { flex: 1; font-size: 13px; color: var(--ink); }
.t-act__tag { flex: none; font-size: 11px; }
.lt-g { color: var(--green); } .lt-p { color: var(--purple); } .lt-c { color: var(--cyan); } .lt-b { color: var(--blue); }

/* steps */
.t-steps { display: grid; grid-template-columns: repeat(3, 1fr); gap: 16px; }
.t-step__n { color: var(--blue); font-size: 13px; font-weight: 700; }
.t-step__t { font-weight: 700; font-size: 20px; color: var(--bright); margin: 10px 0 0; }

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

/* mobile nav */
.t-burger { display: none; background: none; border: 1px solid var(--line); border-radius: 8px; color: var(--ink); font-size: 18px; width: 38px; height: 36px; cursor: pointer; }
.t-mobile { display: none; }
@media (max-width: 900px) {
  .t-nav__cta { display: none; }
  .t-burger { display: block; }
  .t-mobile { display: flex; flex-direction: column; gap: 4px; padding: 8px 24px 18px; border-bottom: 1px solid var(--line); background: rgba(11,12,18,.96); backdrop-filter: blur(12px); }
  .t-mobile a { padding: 11px 4px; text-decoration: none; color: var(--mut); font-size: 14px; }
  .t-mobile a.t-btn--solid { margin-top: 6px; justify-content: center; }
}

/* ⌘K band */
.t-kband { display: grid; grid-template-columns: 1fr 1fr; gap: 48px; align-items: center; }
.t-kbar { border: 1px solid var(--line); border-radius: 14px; background: linear-gradient(180deg, #0f1118, #0b0c12); box-shadow: 0 30px 70px -40px rgba(0,0,0,.85); overflow: hidden; }
.t-kbar__search { display: flex; align-items: center; gap: 10px; padding: 14px 16px; border-bottom: 1px solid var(--line); }
.t-kbar__input { flex: 1; background: none; border: none; outline: none; color: var(--ink); font-family: var(--m); font-size: 14px; }
.t-kbar__input::placeholder { color: var(--faint); }
.t-kbar__list { list-style: none; margin: 0; padding: 8px; display: flex; flex-direction: column; gap: 2px; max-height: 240px; overflow: auto; }
.t-kbar__item { display: flex; align-items: center; gap: 12px; padding: 9px 11px; border-radius: 9px; font-size: 13.5px; transition: background .14s; }
.t-kbar__item:hover { background: rgba(122,162,247,.1); }
.t-kbar__i { width: 20px; text-align: center; color: var(--purple); }
.t-kbar__t { flex: 1; color: var(--ink); }
.t-kbar__s { color: var(--faint); font-size: 11.5px; border: 1px solid var(--line); border-radius: 5px; padding: 1px 7px; }
.t-kbar__empty { padding: 14px; color: var(--faint); font-size: 13px; text-align: center; }

/* comparison */
.t-cmp { border: 1px solid var(--line); border-radius: 14px; overflow: hidden; }
.t-cmp__head, .t-cmp__row { display: grid; grid-template-columns: 1fr 84px 84px 84px; align-items: center; }
.t-cmp__head { padding: 14px 18px; border-bottom: 1px solid var(--line); background: rgba(255,255,255,.02); font-size: 12.5px; color: var(--mut); }
.t-cmp__head span:not(:first-child) { text-align: center; }
.t-cmp__me { color: var(--blue); font-weight: 700; }
.t-cmp__row { padding: 13px 18px; border-bottom: 1px solid var(--line); }
.t-cmp__row:last-child { border-bottom: none; }
.t-cmp__f { font-size: 13.5px; color: var(--ink); }
.t-cmp__c { text-align: center; }
.t-cmp__c i { font-style: normal; font-weight: 700; }
.t-cmp__c i.ok { color: var(--green); }
.t-cmp__c i.part { color: var(--orange); }
.t-cmp__c i.no { color: var(--faint); }

/* faq */
.t-faq { max-width: 760px; margin: 0 auto; display: flex; flex-direction: column; gap: 10px; }
.t-faq__i { border: 1px solid var(--line); border-radius: 11px; background: rgba(255,255,255,.015); overflow: hidden; }
.t-faq__i summary { display: flex; align-items: center; justify-content: space-between; gap: 12px; padding: 16px 18px; cursor: pointer; list-style: none; }
.t-faq__i summary::-webkit-details-marker { display: none; }
.t-faq__q { font-size: 14.5px; color: var(--bright); }
.t-faq__x { color: var(--blue); font-size: 18px; transition: transform .2s; }
.t-faq__i[open] .t-faq__x { transform: rotate(45deg); }
.t-faq__a { margin: 0; padding: 0 18px 18px; color: var(--mut); font-size: 14px; line-height: 1.65; }

/* pricing toggle */
.t-toggle { display: inline-flex; gap: 3px; margin: 22px auto 0; padding: 3px; border: 1px solid var(--line); border-radius: 999px; background: rgba(255,255,255,.02); }
.t-toggle button { font-family: var(--m); font-size: 13px; color: var(--mut); background: none; border: none; padding: 7px 16px; border-radius: 999px; cursor: pointer; transition: color .15s, background .15s; }
.t-toggle button.is-on { color: var(--bright); background: rgba(122,162,247,.16); }
.t-toggle__save { color: var(--green); font-size: 11px; }

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
  .t-kband { grid-template-columns: 1fr; gap: 28px; }
  .t-cmp__head, .t-cmp__row { grid-template-columns: 1fr 52px 52px 52px; }
  .t-cmp__f { font-size: 12.5px; }
}
@media (prefers-reduced-motion: reduce) {
  .t-up, .rv { animation: none !important; opacity: 1 !important; transform: none !important; transition: none !important; }
  .t-cur, .t-scan, .t-row__edit, .t-online i { animation: none !important; }
  .t-row__edit { display: none; }
}
</style>
