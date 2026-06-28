import { useRuntimeConfig } from '#imports'

// connectWs открывает WS, подписывается на канал воркспейса.
// Аутентификация — cookie сессии (same-site handshake шлёт её автоматически).
// onDelta — на каждую дельту; onSignal — на WebRTC-сигнал (offer/answer/ice/bye).
export function connectWs(wsId: string, onDelta: (d: any) => void, onSignal?: (s: any) => void): WebSocket {
  const base = useRuntimeConfig().public.wsBase as string
  const ws = new WebSocket(`${base}/ws`)
  ws.onopen = () => ws.send(JSON.stringify({ type: 'subscribe', channels: [`ws:${wsId}`] }))
  ws.onmessage = (e) => {
    try {
      const m = JSON.parse(e.data)
      if (m.type === 'delta') onDelta(m.delta)
      else if (m.type === 'signal' && onSignal) onSignal(m.signal)
    } catch {}
  }
  return ws
}
