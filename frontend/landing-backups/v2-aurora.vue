<script setup lang="ts">
import { isAuthed } from '~/lib/token'

onMounted(() => { if (isAuthed()) navigateTo('/my/assigned') })

useHead({
  title: 'PMS — Where teams move work forward',
  link: [
    { rel: 'preconnect', href: 'https://fonts.googleapis.com' },
    { rel: 'preconnect', href: 'https://fonts.gstatic.com', crossorigin: '' },
    { rel: 'stylesheet', href: 'https://fonts.googleapis.com/css2?family=Instrument+Serif:ital@0;1&family=Inter:wght@400;500;600&display=swap' },
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

onMounted(() => {
  window.addEventListener('scroll', onScroll, { passive: true }); onScroll()
  document.addEventListener('click', onAnchorClick)
})
onBeforeUnmount(() => {
  window.removeEventListener('scroll', onScroll)
  document.removeEventListener('click', onAnchorClick)
})

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

// live activity feed (signature) — какой пункт «только что» произошёл, циклически
const feed = [
  { who: 'M', c: '#7C5CFF', t: 'Maya moved', b: 'PMS-204 → In review' },
  { who: 'L', c: '#FF5CAA', t: 'Leo commented on', b: 'Billing redesign' },
  { who: '✓', c: '#22C9A3', t: 'Shipped', b: 'Cycle 12 · checkout v2' },
  { who: 'A', c: '#FFB35C', t: 'Ada started', b: 'a huddle in #design' },
]
const live = ref(0)
let timer: any
onMounted(() => {
  if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) return
  timer = setInterval(() => { live.value = (live.value + 1) % feed.length }, 2200)
})
onBeforeUnmount(() => clearInterval(timer))

const marquee = ['Northwind', 'Lumen', 'Vertex Labs', 'Cobalt', 'Forge', 'Halcyon', 'Mercury', 'Atlas', 'Beacon', 'Drift']

const features = [
  { k: 'Issues & boards', d: 'Track every piece of work — sub-issues, relations, labels, cycles. Sort it on a board or a list and keep your own saved views.', span: true },
  { k: 'Docs & chat', d: 'Write specs next to the work and talk in threaded channels. No fourth tab.' },
  { k: 'Calendar', d: 'Plan in month, week or day. Drag to schedule, see who’s free.' },
  { k: 'Automations', d: 'When work changes, the busywork runs itself — status, owners, labels, pings.' },
  { k: 'Meetings', d: 'Huddle from any channel. Video built in, no invite link.' },
  { k: 'Time & reports', d: 'Track hours, roll them up by person, task or day, export on demand.' },
]

const plans = [
  { name: 'Free', price: '$0', unit: '', blurb: 'Get your team started.', feats: ['Up to 10 teammates', 'Issues, boards & calendar', 'Realtime collaboration'], cta: 'Start free', accent: false },
  { name: 'Pro', price: '$8', unit: '/ user · mo', blurb: 'For teams shipping weekly.', feats: ['Everything in Free', 'Automations & cycles', 'Time tracking & reports', 'Unlimited members'], cta: 'Start free trial', accent: true },
  { name: 'Business', price: '$16', unit: '/ user · mo', blurb: 'For scaling orgs.', feats: ['Everything in Pro', 'Advanced permissions', 'Audit & SSO', 'Priority support'], cta: 'Start free trial', accent: false },
]
</script>

<template>
  <div class="a">
    <div class="a-aurora" aria-hidden="true">
      <span class="a-blob a-blob--1"></span>
      <span class="a-blob a-blob--2"></span>
      <span class="a-blob a-blob--3"></span>
    </div>

    <!-- NAV -->
    <header class="a-nav" :class="{ 'a-nav--solid': scrolled }">
      <div class="a-nav__in">
        <a href="/" class="a-brand"><span class="a-brand__mark"></span><span class="a-brand__name">PMS</span></a>
        <nav class="a-nav__links">
          <a href="#features">Product</a>
          <a href="#flow">How it works</a>
          <a href="#pricing">Pricing</a>
        </nav>
        <div class="a-nav__cta">
          <a href="/login" class="a-quiet">Sign in</a>
          <a href="/login" class="a-btn a-btn--solid">Start free</a>
        </div>
      </div>
    </header>

    <!-- HERO -->
    <section class="a-hero">
      <div class="a-hero__copy">
        <span class="a-eyebrow a-up" style="--i:0">The all-in-one workspace for teams</span>
        <h1 class="a-h1 a-up" style="--i:1">Where teams move<br />work <em>forward</em>.</h1>
        <p class="a-sub a-up" style="--i:2">
          Issues, docs, chat, calendar and meetings in one place — built so your team
          plans, builds and ships together, in real time.
        </p>
        <div class="a-actions a-up" style="--i:3">
          <a href="/login" class="a-btn a-btn--solid a-btn--lg">Start for free</a>
          <a href="#features" class="a-btn a-btn--glass a-btn--lg">Take the tour</a>
        </div>
        <p class="a-fineprint a-up" style="--i:4">Free to start · no credit card · your whole team in minutes</p>
      </div>

      <!-- SIGNATURE: live activity -->
      <div class="a-stage a-up" style="--i:3" aria-hidden="true">
        <div class="a-glass a-live">
          <div class="a-live__top">
            <span class="a-live__title">Workspace activity</span>
            <span class="a-online"><i></i><i></i><i></i> 3 online</span>
          </div>
          <ul class="a-live__list">
            <li v-for="(f, i) in feed" :key="f.b" class="a-live__row" :class="{ 'is-live': live === i }">
              <span class="a-ava" :style="{ '--a': f.c }">{{ f.who }}</span>
              <span class="a-live__txt"><b>{{ f.t }}</b> {{ f.b }}</span>
              <span class="a-live__dot"></span>
            </li>
          </ul>
          <div class="a-live__foot">
            <div class="a-bar"><span :style="{ width: '84%' }"></span></div>
            <span class="a-live__cap">Cycle 12 · 84% complete</span>
          </div>
        </div>
      </div>
    </section>

    <!-- MARQUEE -->
    <section class="a-marq" v-reveal>
      <p class="a-marq__lbl">Teams move faster on PMS</p>
      <div class="a-marq__track">
        <span v-for="(n, i) in [...marquee, ...marquee]" :key="i">{{ n }}</span>
      </div>
    </section>

    <!-- FEATURES -->
    <section id="features" class="a-sec">
      <header class="a-shead" v-reveal>
        <span class="a-eyebrow">One workspace, the whole loop</span>
        <h2 class="a-h2">Everything the team needs,<br /><em>nothing it doesn’t.</em></h2>
      </header>
      <div class="a-bento">
        <article v-for="(f, i) in features" :key="f.k" class="a-glass a-fcard" :class="{ 'a-fcard--wide': f.span }" v-reveal="i * 60">
          <h3 class="a-fcard__t">{{ f.k }}</h3>
          <p class="a-fcard__d">{{ f.d }}</p>
        </article>
      </div>
    </section>

    <!-- SPOTLIGHT: collaboration -->
    <section class="a-spot">
      <div class="a-spot__copy" v-reveal>
        <span class="a-eyebrow">Real-time by default</span>
        <h2 class="a-h2">Everyone on the <em>same page</em> — literally.</h2>
        <p class="a-lead">Edits land for the whole team the instant they happen. See who’s here, what changed, and pick up exactly where someone left off.</p>
        <ul class="a-checks">
          <li>Live presence and instant updates</li>
          <li>Threaded chat next to the work</li>
          <li>Comments, mentions and reactions everywhere</li>
        </ul>
      </div>
      <div class="a-spot__art" v-reveal="120">
        <div class="a-glass a-collab">
          <div class="a-collab__row"><span class="a-ava" style="--a:#7C5CFF">M</span><span class="a-bubble">Pushed the fix — can you review?</span></div>
          <div class="a-collab__row a-collab__row--r"><span class="a-bubble a-bubble--me">On it. Looks clean ✨</span><span class="a-ava" style="--a:#FF5CAA">L</span></div>
          <div class="a-collab__typing"><span class="a-ava" style="--a:#22C9A3">A</span><span class="a-dots"><i></i><i></i><i></i></span></div>
        </div>
      </div>
    </section>

    <!-- SPOTLIGHT: automations (reversed) -->
    <section class="a-spot a-spot--rev">
      <div class="a-spot__copy" v-reveal>
        <span class="a-eyebrow">Automations</span>
        <h2 class="a-h2">Set the rule once.<br />It runs <em>every time</em>.</h2>
        <p class="a-lead">Tell PMS what should happen when work changes — and stop doing it by hand. Reassign, re-label, update status, ping a channel. No scripts.</p>
        <ul class="a-checks">
          <li>Triggers on create, status, owner or priority</li>
          <li>Chain several actions in one rule</li>
          <li>Scope it to a team or the whole org</li>
        </ul>
      </div>
      <div class="a-spot__art" v-reveal="120">
        <div class="a-glass a-rule">
          <div class="a-rule__row" style="--r:0"><span class="a-rule__k">WHEN</span><span class="a-pill">moved to <b>In review</b></span></div>
          <div class="a-rule__pipe"></div>
          <div class="a-rule__row" style="--r:1"><span class="a-rule__k a-rule__k--then">DO</span><span class="a-pill a-pill--act">assign <b>reviewer</b></span></div>
          <div class="a-rule__row" style="--r:2"><span class="a-rule__k a-rule__k--then">DO</span><span class="a-pill a-pill--act">add <b>needs-review</b></span></div>
          <div class="a-rule__row" style="--r:3"><span class="a-rule__k a-rule__k--then">DO</span><span class="a-pill a-pill--act">post to <b>#reviews</b></span></div>
        </div>
      </div>
    </section>

    <!-- HOW IT WORKS -->
    <section id="flow" class="a-sec">
      <header class="a-shead" v-reveal>
        <span class="a-eyebrow">From idea to shipped</span>
        <h2 class="a-h2">Three moves. <em>On repeat.</em></h2>
      </header>
      <div class="a-steps">
        <article v-for="(s, i) in [
          { t: 'Capture', d: 'Spin up an issue in a keystroke and it’s tracked before the thought is gone.' },
          { t: 'Organize', d: 'Drop it into a cycle, a board, a view. The layout is yours to keep.' },
          { t: 'Ship', d: 'Move it to done — automations fire, the team hears about it, repeat.' },
        ]" :key="s.t" class="a-glass a-step" v-reveal="i * 90">
          <span class="a-step__n">{{ i + 1 }}</span>
          <h3 class="a-step__t">{{ s.t }}</h3>
          <p class="a-step__d">{{ s.d }}</p>
        </article>
      </div>
    </section>

    <!-- QUOTE -->
    <section class="a-quote" v-reveal>
      <p class="a-quote__mark">“</p>
      <blockquote>We replaced four tools with PMS and the team just… moves faster. Everything’s in one place, and it’s <em>instant</em>.</blockquote>
      <div class="a-quote__by"><span class="a-ava" style="--a:#7C5CFF">R</span><span>Riya Menon — Head of Product, Lumen</span></div>
    </section>

    <!-- PRICING -->
    <section id="pricing" class="a-sec">
      <header class="a-shead" v-reveal>
        <span class="a-eyebrow">Pricing</span>
        <h2 class="a-h2">Simple plans that <em>scale</em> with you.</h2>
        <p class="a-lead a-lead--c">Start free. Upgrade when the team grows. Cancel anytime.</p>
      </header>
      <div class="a-plans">
        <article v-for="(p, i) in plans" :key="p.name" class="a-glass a-plan" :class="{ 'a-plan--accent': p.accent }" v-reveal="i * 80">
          <span v-if="p.accent" class="a-plan__badge">Most popular</span>
          <h3 class="a-plan__name">{{ p.name }}</h3>
          <div class="a-plan__price"><span class="a-plan__amt">{{ p.price }}</span><span class="a-plan__unit">{{ p.unit }}</span></div>
          <p class="a-plan__blurb">{{ p.blurb }}</p>
          <ul class="a-plan__feats"><li v-for="f in p.feats" :key="f"><span class="a-tick">✓</span>{{ f }}</li></ul>
          <a href="/login" class="a-btn" :class="p.accent ? 'a-btn--solid' : 'a-btn--glass'">{{ p.cta }}</a>
        </article>
      </div>
      <p class="a-selfhost">Need to run it on your own infrastructure? <a href="/login">Self-hosting is available too →</a></p>
    </section>

    <!-- CTA -->
    <section class="a-cta" v-reveal>
      <div class="a-cta__panel">
        <h2 class="a-h2">Bring the team together.<br /><em>Start today.</em></h2>
        <p class="a-lead a-lead--c">One workspace for everything your team ships. Free to start.</p>
        <a href="/login" class="a-btn a-btn--solid a-btn--lg">Get started — it’s free</a>
      </div>
    </section>

    <!-- FOOTER -->
    <footer class="a-foot">
      <div class="a-foot__in">
        <div class="a-foot__brand">
          <a href="/" class="a-brand"><span class="a-brand__mark"></span><span class="a-brand__name">PMS</span></a>
          <p>The all-in-one workspace where teams move work forward.</p>
        </div>
        <div class="a-foot__cols">
          <div><h4>Product</h4><a href="#features">Features</a><a href="#flow">How it works</a><a href="#pricing">Pricing</a></div>
          <div><h4>Build</h4><a href="/login">Issues</a><a href="/login">Automations</a><a href="/login">Calendar</a></div>
          <div><h4>Company</h4><a href="#">About</a><a href="#">Docs</a><a href="/login">Sign in</a></div>
        </div>
      </div>
      <div class="a-foot__base"><span>© 2026 PMS</span><span>Made for teams that ship.</span></div>
    </footer>
  </div>
</template>

<style scoped>
.a {
  --bg: #07060C; --ink: #F3F1F8; --mut: #A09EB2; --faint: #6C6A7E;
  --line: rgba(255,255,255,.09); --glass: rgba(255,255,255,.035);
  --violet: #7C5CFF; --fuchsia: #FF5CAA; --indigo: #5A6BFF; --amber: #FFB35C; --mint: #22C9A3;
  --grad: linear-gradient(115deg, #7C5CFF, #C24DE0 48%, #FF5CAA);
  --serif: 'Instrument Serif', Georgia, serif;
  position: relative; background: var(--bg); color: var(--ink); overflow-x: clip; isolation: isolate;
  font-family: 'Inter var', Inter, sans-serif; font-size: 16px; line-height: 1.5;
}
.a ::selection { background: rgba(193,77,224,.34); }

/* aurora */
.a-aurora { position: fixed; inset: 0; z-index: 0; pointer-events: none; overflow: hidden; }
.a-blob { position: absolute; border-radius: 50%; filter: blur(80px); opacity: .5; }
.a-blob--1 { width: 46vw; height: 46vw; left: -8vw; top: -14vw; background: radial-gradient(circle, #7C5CFF, transparent 65%); animation: float1 26s ease-in-out infinite alternate; }
.a-blob--2 { width: 40vw; height: 40vw; right: -6vw; top: -6vw; background: radial-gradient(circle, #FF5CAA, transparent 65%); opacity: .38; animation: float2 30s ease-in-out infinite alternate; }
.a-blob--3 { width: 38vw; height: 38vw; left: 40vw; top: 24vw; background: radial-gradient(circle, #5A6BFF, transparent 65%); opacity: .3; animation: float3 34s ease-in-out infinite alternate; }
@keyframes float1 { to { transform: translate(6vw, 5vw) scale(1.12); } }
@keyframes float2 { to { transform: translate(-5vw, 7vw) scale(1.1); } }
@keyframes float3 { to { transform: translate(-7vw, -5vw) scale(1.16); } }
.a > section, .a > header, .a > footer { position: relative; z-index: 1; }

/* glass primitive */
.a-glass { background: var(--glass); border: 1px solid var(--line); border-radius: 20px; backdrop-filter: blur(20px) saturate(1.2); box-shadow: 0 30px 70px -40px rgba(0,0,0,.9), inset 0 1px 0 rgba(255,255,255,.06); }

/* nav */
.a-nav { position: fixed; inset: 0 0 auto 0; z-index: 50; transition: background .3s, border-color .3s; border-bottom: 1px solid transparent; }
.a-nav--solid { background: rgba(7,6,12,.6); backdrop-filter: blur(16px); border-color: var(--line); }
.a-nav__in { max-width: 1180px; margin: 0 auto; padding: 16px 24px; display: flex; align-items: center; gap: 30px; }
.a-brand { display: inline-flex; align-items: center; gap: 9px; text-decoration: none; color: var(--ink); }
.a-brand__mark { width: 24px; height: 24px; border-radius: 8px; background: var(--grad); box-shadow: 0 4px 18px -4px rgba(193,77,224,.7); }
.a-brand__name { font-family: var(--serif); font-size: 21px; letter-spacing: .01em; }
.a-nav__links { display: flex; gap: 26px; margin-right: auto; }
.a-nav__links a { position: relative; color: var(--mut); text-decoration: none; font-size: 14px; transition: color .18s; }
.a-nav__links a:hover { color: var(--ink); }
.a-nav__cta { display: flex; align-items: center; gap: 16px; }
.a-quiet { color: var(--mut); text-decoration: none; font-size: 14px; transition: color .18s; }
.a-quiet:hover { color: var(--ink); }

/* buttons */
.a-btn { display: inline-flex; align-items: center; justify-content: center; border-radius: 999px; font-size: 14px; font-weight: 500; padding: 9px 18px; text-decoration: none; border: 1px solid transparent; transition: transform .16s, box-shadow .22s, background .22s, border-color .22s; white-space: nowrap; }
.a-btn--lg { padding: 13px 26px; font-size: 15px; }
.a-btn--solid { color: #fff; background: var(--grad); background-size: 160% 160%; box-shadow: 0 10px 30px -8px rgba(193,77,224,.6), inset 0 1px 0 rgba(255,255,255,.25); }
.a-btn--solid:hover { transform: translateY(-2px); background-position: 100% 0; box-shadow: 0 16px 40px -8px rgba(193,77,224,.75); }
.a-btn--glass { color: var(--ink); background: var(--glass); border-color: var(--line); }
.a-btn--glass:hover { transform: translateY(-2px); background: rgba(255,255,255,.08); border-color: rgba(255,255,255,.18); }

/* hero */
.a-hero { max-width: 1180px; margin: 0 auto; padding: 168px 24px 96px; display: grid; grid-template-columns: 1.08fr .92fr; gap: 60px; align-items: center; }
.a-eyebrow { font-size: 13px; letter-spacing: .01em; color: var(--violet); font-weight: 500; }
.a-h1 { font-family: var(--serif); font-weight: 400; font-size: clamp(46px, 7vw, 84px); line-height: .98; letter-spacing: -.01em; margin: 18px 0 0; }
.a-h1 em, .a-h2 em { font-style: italic; background: var(--grad); -webkit-background-clip: text; background-clip: text; color: transparent; }
.a-sub { margin: 24px 0 0; max-width: 31rem; color: var(--mut); font-size: 18px; line-height: 1.6; }
.a-actions { display: flex; gap: 14px; margin-top: 32px; flex-wrap: wrap; }
.a-fineprint { margin-top: 20px; color: var(--faint); font-size: 13.5px; }

/* signature: live activity */
.a-stage { position: relative; }
.a-live { padding: 20px; }
.a-live__top { display: flex; align-items: center; justify-content: space-between; padding-bottom: 14px; border-bottom: 1px solid var(--line); }
.a-live__title { font-size: 13px; color: var(--mut); }
.a-online { display: inline-flex; align-items: center; gap: 7px; font-size: 12px; color: var(--mint); }
.a-online i { width: 7px; height: 7px; border-radius: 50%; background: var(--mint); box-shadow: 0 0 8px var(--mint); animation: blink 2.4s ease-in-out infinite; }
.a-online i:nth-child(2) { animation-delay: .3s; } .a-online i:nth-child(3) { animation-delay: .6s; }
@keyframes blink { 0%,100% { opacity: 1; } 50% { opacity: .4; } }
.a-live__list { list-style: none; margin: 0; padding: 8px 0; display: flex; flex-direction: column; gap: 2px; }
.a-live__row { display: flex; align-items: center; gap: 12px; padding: 11px 12px; border-radius: 12px; position: relative; transition: background .5s; }
.a-live__row.is-live { background: rgba(124,92,255,.12); }
.a-live__row.is-live::before { content: ''; position: absolute; left: 0; top: 14%; bottom: 14%; width: 3px; border-radius: 3px; background: var(--grad); }
.a-ava { flex: none; width: 32px; height: 32px; border-radius: 10px; display: grid; place-items: center; font-size: 13px; font-weight: 600; color: #fff; background: var(--a); box-shadow: 0 0 18px -6px var(--a); }
.a-live__txt { flex: 1; font-size: 13.5px; color: var(--mut); }
.a-live__txt b { color: var(--ink); font-weight: 600; }
.a-live__dot { width: 7px; height: 7px; border-radius: 50%; background: #2c2a39; }
.a-live__row.is-live .a-live__dot { background: var(--fuchsia); box-shadow: 0 0 8px var(--fuchsia); }
.a-live__foot { padding: 14px 12px 4px; border-top: 1px solid var(--line); }
.a-bar { height: 6px; border-radius: 4px; background: rgba(255,255,255,.07); overflow: hidden; }
.a-bar span { display: block; height: 100%; border-radius: 4px; background: var(--grad); }
.a-live__cap { display: block; margin-top: 8px; font-size: 12px; color: var(--faint); }

/* marquee */
.a-marq { max-width: 1180px; margin: 0 auto; padding: 8px 24px 0; text-align: center; }
.a-marq__lbl { font-size: 13px; color: var(--faint); }
.a-marq__track { display: flex; gap: 48px; margin-top: 18px; width: max-content; animation: marq 34s linear infinite; -webkit-mask-image: linear-gradient(90deg, transparent, #000 12%, #000 88%, transparent); mask-image: linear-gradient(90deg, transparent, #000 12%, #000 88%, transparent); }
.a-marq__track span { font-family: var(--serif); font-size: 24px; color: #4b495c; white-space: nowrap; }
@keyframes marq { to { transform: translateX(-50%); } }

/* sections */
.a-sec { max-width: 1180px; margin: 0 auto; padding: 130px 24px 0; scroll-margin-top: 84px; }
.a-shead { text-align: center; max-width: 42rem; margin: 0 auto 56px; }
.a-h2 { font-family: var(--serif); font-weight: 400; font-size: clamp(32px, 5vw, 56px); line-height: 1.04; letter-spacing: -.01em; margin: 14px 0 0; }
.a-lead { color: var(--mut); font-size: 17px; line-height: 1.6; margin: 16px 0 0; }
.a-lead--c { margin-left: auto; margin-right: auto; max-width: 32rem; }

/* bento */
.a-bento { display: grid; grid-template-columns: repeat(3, 1fr); gap: 16px; }
.a-fcard { padding: 26px; transition: transform .25s, border-color .25s, box-shadow .25s; }
.a-fcard--wide { grid-column: span 2; }
.a-fcard:hover { transform: translateY(-4px); border-color: rgba(255,255,255,.16); box-shadow: 0 40px 80px -40px rgba(124,92,255,.5); }
.a-fcard__t { font-family: var(--serif); font-size: 25px; font-weight: 400; }
.a-fcard__d { color: var(--mut); font-size: 14.5px; line-height: 1.55; margin: 10px 0 0; }

/* spotlight */
.a-spot { max-width: 1120px; margin: 0 auto; padding: 130px 24px 0; display: grid; grid-template-columns: 1fr 1fr; gap: 60px; align-items: center; }
.a-spot--rev .a-spot__copy { order: 2; }
.a-checks { list-style: none; padding: 0; margin: 26px 0 0; display: flex; flex-direction: column; gap: 13px; }
.a-checks li { position: relative; padding-left: 30px; color: #c8c6d6; font-size: 15.5px; }
.a-checks li::before { content: '✓'; position: absolute; left: 0; top: 0; width: 20px; height: 20px; border-radius: 7px; display: grid; place-items: center; font-size: 11px; color: #fff; background: var(--grad); }

.a-spot__art { display: grid; }
/* collab card */
.a-collab { padding: 24px; display: flex; flex-direction: column; gap: 14px; }
.a-collab__row { display: flex; align-items: flex-end; gap: 10px; }
.a-collab__row--r { flex-direction: row-reverse; }
.a-bubble { max-width: 70%; padding: 11px 14px; border-radius: 16px 16px 16px 4px; background: rgba(255,255,255,.06); border: 1px solid var(--line); font-size: 14px; color: var(--ink); }
.a-bubble--me { border-radius: 16px 16px 4px 16px; background: rgba(124,92,255,.16); border-color: rgba(124,92,255,.3); }
.a-collab__typing { display: flex; align-items: center; gap: 10px; }
.a-dots { display: inline-flex; gap: 4px; padding: 11px 14px; border-radius: 16px; background: rgba(255,255,255,.06); border: 1px solid var(--line); }
.a-dots i { width: 6px; height: 6px; border-radius: 50%; background: var(--mut); animation: typ 1.3s ease-in-out infinite; }
.a-dots i:nth-child(2) { animation-delay: .18s; } .a-dots i:nth-child(3) { animation-delay: .36s; }
@keyframes typ { 0%,100% { opacity: .3; transform: translateY(0); } 50% { opacity: 1; transform: translateY(-3px); } }
/* rule card */
.a-rule { padding: 24px; display: flex; flex-direction: column; gap: 12px; }
.rv .a-rule__row { opacity: 0; transform: translateX(-10px); }
.rv-in .a-rule__row { animation: rowIn .5s cubic-bezier(.22,1,.36,1) forwards; animation-delay: calc(var(--r,0) * 140ms + 180ms); }
@keyframes rowIn { to { opacity: 1; transform: none; } }
.a-rule__row { display: flex; align-items: center; gap: 12px; }
.a-rule__k { font-size: 11px; letter-spacing: .12em; width: 38px; color: var(--faint); font-weight: 600; }
.a-rule__k--then { color: var(--violet); }
.a-pill { background: rgba(255,255,255,.05); border: 1px solid var(--line); border-radius: 10px; padding: 9px 13px; font-size: 13.5px; color: #cdcbd9; }
.a-pill b { color: #fff; font-weight: 600; }
.a-pill--act { border-color: rgba(124,92,255,.32); background: rgba(124,92,255,.1); }
.a-rule__pipe { width: 2px; height: 16px; margin-left: 18px; border-radius: 2px; background: var(--line); position: relative; overflow: hidden; }
.a-rule__pipe::after { content: ''; position: absolute; inset: 0; background: linear-gradient(180deg, transparent, var(--violet), transparent); animation: flow 1.8s ease-in-out infinite; }
@keyframes flow { 0% { transform: translateY(-100%); } 100% { transform: translateY(100%); } }

/* steps */
.a-steps { display: grid; grid-template-columns: repeat(3, 1fr); gap: 18px; }
.a-step { padding: 30px; }
.a-step__n { display: grid; place-items: center; width: 38px; height: 38px; border-radius: 12px; font-family: var(--serif); font-size: 20px; color: #fff; background: var(--grad); box-shadow: 0 8px 22px -8px rgba(193,77,224,.6); }
.a-step__t { font-family: var(--serif); font-size: 26px; font-weight: 400; margin: 16px 0 0; }
.a-step__d { color: var(--mut); font-size: 14.5px; line-height: 1.55; margin: 8px 0 0; }

/* quote */
.a-quote { max-width: 880px; margin: 0 auto; padding: 130px 24px 0; text-align: center; }
.a-quote__mark { font-family: var(--serif); font-size: 90px; line-height: .4; background: var(--grad); -webkit-background-clip: text; background-clip: text; color: transparent; }
.a-quote blockquote { font-family: var(--serif); font-weight: 400; font-size: clamp(26px, 3.6vw, 40px); line-height: 1.28; margin: 14px 0 0; }
.a-quote__by { display: inline-flex; align-items: center; gap: 12px; margin-top: 28px; color: var(--mut); font-size: 14.5px; }

/* pricing */
.a-plans { display: grid; grid-template-columns: repeat(3, 1fr); gap: 18px; align-items: start; }
.a-plan { padding: 30px; display: flex; flex-direction: column; position: relative; }
.a-plan--accent { border-color: rgba(124,92,255,.45); box-shadow: 0 40px 90px -44px rgba(124,92,255,.7); }
.a-plan--accent::before { content: ''; position: absolute; inset: 0; border-radius: 20px; padding: 1px; background: var(--grad); -webkit-mask: linear-gradient(#000 0 0) content-box, linear-gradient(#000 0 0); -webkit-mask-composite: xor; mask-composite: exclude; opacity: .6; }
.a-plan__badge { position: absolute; top: -11px; left: 30px; font-size: 11px; letter-spacing: .04em; color: #fff; background: var(--grad); padding: 4px 11px; border-radius: 999px; }
.a-plan__name { font-family: var(--serif); font-size: 22px; font-weight: 400; }
.a-plan__price { display: flex; align-items: baseline; gap: 7px; margin: 12px 0 0; }
.a-plan__amt { font-family: var(--serif); font-size: 44px; letter-spacing: -.01em; }
.a-plan__unit { color: var(--faint); font-size: 13px; }
.a-plan__blurb { color: var(--mut); font-size: 14px; margin: 8px 0 0; }
.a-plan__feats { list-style: none; padding: 0; margin: 22px 0 24px; display: flex; flex-direction: column; gap: 11px; flex: 1; }
.a-plan__feats li { display: flex; align-items: center; gap: 10px; font-size: 14px; color: #c8c6d6; }
.a-tick { color: var(--mint); font-size: 12px; }
.a-selfhost { text-align: center; margin: 28px 0 0; color: var(--faint); font-size: 14px; }
.a-selfhost a { color: var(--violet); text-decoration: none; }
.a-selfhost a:hover { text-decoration: underline; }

/* cta */
.a-cta { max-width: 1120px; margin: 130px auto 0; padding: 0 24px; }
.a-cta__panel { position: relative; overflow: hidden; text-align: center; border-radius: 28px; padding: 84px 24px; border: 1px solid var(--line); background: radial-gradient(120% 140% at 50% 0%, rgba(124,92,255,.22), transparent 60%), rgba(255,255,255,.025); }
.a-cta__panel .a-lead { margin-left: auto; margin-right: auto; }
.a-cta__panel .a-btn { margin-top: 30px; }

/* footer */
.a-foot { max-width: 1180px; margin: 130px auto 0; padding: 50px 24px 44px; border-top: 1px solid var(--line); }
.a-foot__in { display: grid; grid-template-columns: 1.4fr 2fr; gap: 40px; }
.a-foot__brand p { color: var(--mut); font-size: 14px; margin: 14px 0 0; max-width: 22rem; line-height: 1.55; }
.a-foot__cols { display: grid; grid-template-columns: repeat(3, 1fr); gap: 24px; }
.a-foot__cols h4 { font-size: 12px; letter-spacing: .08em; text-transform: uppercase; color: var(--faint); margin: 0 0 14px; }
.a-foot__cols a { display: block; color: var(--mut); text-decoration: none; font-size: 14px; margin-bottom: 10px; transition: color .18s; }
.a-foot__cols a:hover { color: var(--ink); }
.a-foot__base { display: flex; justify-content: space-between; margin-top: 40px; padding-top: 22px; border-top: 1px solid var(--line); color: var(--faint); font-size: 13px; }

/* motion: load + reveal */
.a-up { opacity: 0; transform: translateY(18px); animation: up .8s cubic-bezier(.22,1,.36,1) forwards; animation-delay: calc(var(--i,0) * 90ms + 120ms); }
@keyframes up { to { opacity: 1; transform: none; } }
.rv { opacity: 0; transform: translateY(18px); filter: blur(7px); transition: opacity .8s cubic-bezier(.22,1,.36,1), transform .8s cubic-bezier(.22,1,.36,1), filter .8s cubic-bezier(.22,1,.36,1); transition-delay: var(--rd, 0ms); will-change: opacity, transform, filter; }
.rv-in { opacity: 1; transform: none; filter: blur(0); }

/* responsive */
@media (max-width: 900px) {
  .a-hero { grid-template-columns: 1fr; padding-top: 130px; gap: 44px; }
  .a-stage { order: 2; }
  .a-bento, .a-steps, .a-plans { grid-template-columns: 1fr; }
  .a-fcard--wide { grid-column: span 1; }
  .a-spot, .a-spot--rev .a-spot__copy { grid-template-columns: 1fr; }
  .a-spot { grid-template-columns: 1fr; }
  .a-spot--rev .a-spot__copy { order: 0; }
  .a-foot__in { grid-template-columns: 1fr; }
  .a-nav__links { display: none; }
}
@media (prefers-reduced-motion: reduce) {
  .a-up, .rv, .rv-in .a-rule__row { animation: none !important; opacity: 1 !important; transform: none !important; filter: none !important; transition: none !important; }
  .rv .a-rule__row { opacity: 1 !important; transform: none !important; }
  .a-blob, .a-marq__track, .a-online i, .a-dots i, .a-rule__pipe::after { animation: none !important; }
  .a-rule__pipe::after { display: none; }
}
</style>
