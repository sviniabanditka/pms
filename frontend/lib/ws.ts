import { useRuntimeConfig } from '#imports'

// connectWs открывает WS, подписывается на канал воркспейса. Аутентификация — cookie сессии.
// onDelta — на каждую дельту. (Медиа митингов — через LiveKit SFU, не через этот WS.)
// Переподключается автоматически с экспоненциальным backoff (сеть/рестарт/сон ноута).
export function connectWs(wsId: string, onDelta: (d: any) => void): { close: () => void } {
  const base = useRuntimeConfig().public.wsBase as string
  let sock: WebSocket | null = null
  let closed = false
  let retry = 0
  let timer: ReturnType<typeof setTimeout> | null = null

  const open = () => {
    sock = new WebSocket(`${base}/ws`)
    sock.onopen = () => { retry = 0; sock!.send(JSON.stringify({ type: 'subscribe', channels: [`ws:${wsId}`] })) }
    sock.onmessage = (e) => {
      try {
        const m = JSON.parse(e.data)
        if (m.type === 'delta') onDelta(m.delta)
      } catch {}
    }
    sock.onclose = () => {
      if (closed) return
      const delay = Math.min(30000, 500 * 2 ** retry) // backoff, потолок 30с
      retry++
      timer = setTimeout(open, delay)
    }
    sock.onerror = () => sock?.close() // → onclose → reconnect
  }

  open()
  return { close: () => { closed = true; if (timer) clearTimeout(timer); sock?.close() } }
}
