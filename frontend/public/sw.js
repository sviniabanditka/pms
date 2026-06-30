// Минимальный offline-SW (без Workbox). Runtime-кэш: shell (навигации) + same-origin
// ассеты (_nuxt/*, css). Данные приложения уже в IndexedDB (стор hydrate на старте),
// поэтому оффлайн = закэшированный shell + idb-стор. API (:8080) и cross-origin не кэшим.
const CACHE = 'stdtask-shell-v1'

self.addEventListener('install', () => self.skipWaiting())

self.addEventListener('activate', (e) => {
  e.waitUntil(
    caches.keys().then((keys) => Promise.all(keys.filter((k) => k !== CACHE).map((k) => caches.delete(k)))).then(() => self.clients.claim())
  )
})

self.addEventListener('fetch', (e) => {
  const req = e.request
  if (req.method !== 'GET') return
  const url = new URL(req.url)
  if (url.origin !== location.origin) return // API/cross-origin → сеть как есть

  // Навигации: network-first, оффлайн → закэшированный документ или '/'.
  if (req.mode === 'navigate') {
    e.respondWith(
      fetch(req)
        .then((r) => { const c = r.clone(); caches.open(CACHE).then((cache) => cache.put(req, c)); return r })
        .catch(() => caches.match(req).then((m) => m || caches.match('/')))
    )
    return
  }

  // Ассеты (_nuxt/*, css, картинки): stale-while-revalidate.
  e.respondWith(
    caches.match(req).then((cached) => {
      const fetched = fetch(req)
        .then((r) => { if (r.ok) { const c = r.clone(); caches.open(CACHE).then((cache) => cache.put(req, c)) } return r })
        .catch(() => cached)
      return cached || fetched
    })
  )
})
