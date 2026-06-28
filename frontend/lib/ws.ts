import { useRuntimeConfig } from '#imports'

// connectWs открывает WS, подписывается на канал воркспейса. Аутентификация — cookie сессии.
// onDelta — на каждую дельту. (Медиа митингов — через LiveKit SFU, не через этот WS.)
export function connectWs(wsId: string, onDelta: (d: any) => void): WebSocket {
  const base = useRuntimeConfig().public.wsBase as string
  const ws = new WebSocket(`${base}/ws`)
  ws.onopen = () => ws.send(JSON.stringify({ type: 'subscribe', channels: [`ws:${wsId}`] }))
  ws.onmessage = (e) => {
    try {
      const m = JSON.parse(e.data)
      if (m.type === 'delta') onDelta(m.delta)
    } catch {}
  }
  return ws
}
