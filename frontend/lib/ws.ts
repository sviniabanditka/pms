import { useRuntimeConfig } from '#imports'
import { getToken } from './token'

// connectWs открывает WS, подписывается на канал воркспейса.
// onDelta — на каждую дельту; onSignal — на WebRTC-сигнал (offer/answer/ice/bye).
export function connectWs(wsId: string, onDelta: (d: any) => void, onSignal?: (s: any) => void): WebSocket {
  const base = useRuntimeConfig().public.wsBase as string
  const ws = new WebSocket(`${base}/ws?token=${getToken()}`)
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
