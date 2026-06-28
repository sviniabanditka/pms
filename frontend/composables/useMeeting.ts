import { ref, reactive } from 'vue'
import { useAppStore } from '~/stores/app'

// WebRTC mesh: каждый join'ящийся инициирует offer к уже подключённым пирам (избегаем glare).
// Сигналинг — через store.sendSignal/setSignalHandler (WS воркспейса).

type Remote = { userId: string; stream: MediaStream }

// --- singleton state (живёт между ре-рендерами компонентов) ---
const activeMeetingId = ref('')
const localStream = ref<MediaStream | null>(null)
const remotes = ref<Record<string, MediaStream>>({})
const muted = ref(false)
const camOff = ref(false)
const connecting = ref(false)
const mediaError = ref('')

// не-реактивные мапы соединений
const pcs = new Map<string, RTCPeerConnection>()
const pendingIce = new Map<string, RTCIceCandidateInit[]>()

const ICE = { iceServers: [{ urls: 'stun:stun.l.google.com:19302' }] }

function setRemote(userId: string, stream: MediaStream) {
  remotes.value = { ...remotes.value, [userId]: stream }
}
function dropRemote(userId: string) {
  const next = { ...remotes.value }; delete next[userId]; remotes.value = next
}

function makePeer(store: ReturnType<typeof useAppStore>, peerId: string): RTCPeerConnection {
  const pc = new RTCPeerConnection(ICE)
  localStream.value?.getTracks().forEach((t) => pc.addTrack(t, localStream.value!))
  pc.onicecandidate = (e) => {
    if (e.candidate) store.sendSignal(activeMeetingId.value, peerId, 'ice', JSON.stringify(e.candidate))
  }
  pc.ontrack = (e) => setRemote(peerId, e.streams[0])
  pc.onconnectionstatechange = () => {
    if (['failed', 'closed', 'disconnected'].includes(pc.connectionState)) dropRemote(peerId)
  }
  pcs.set(peerId, pc)
  return pc
}

async function flushIce(peerId: string, pc: RTCPeerConnection) {
  const buf = pendingIce.get(peerId)
  if (!buf) return
  for (const c of buf) { try { await pc.addIceCandidate(c) } catch {} }
  pendingIce.delete(peerId)
}

async function onSignal(store: ReturnType<typeof useAppStore>, sig: any) {
  const from = sig.fromUserId
  if (sig.meetingId !== activeMeetingId.value) return
  try {
    if (sig.kind === 'offer') {
      let pc = pcs.get(from) || makePeer(store, from)
      await pc.setRemoteDescription(JSON.parse(sig.payload))
      await flushIce(from, pc)
      const answer = await pc.createAnswer()
      await pc.setLocalDescription(answer)
      store.sendSignal(activeMeetingId.value, from, 'answer', JSON.stringify(answer))
    } else if (sig.kind === 'answer') {
      const pc = pcs.get(from); if (!pc) return
      await pc.setRemoteDescription(JSON.parse(sig.payload))
      await flushIce(from, pc)
    } else if (sig.kind === 'ice') {
      const pc = pcs.get(from)
      const cand = JSON.parse(sig.payload)
      if (pc && pc.remoteDescription) { try { await pc.addIceCandidate(cand) } catch {} }
      else { pendingIce.set(from, [...(pendingIce.get(from) || []), cand]) }
    } else if (sig.kind === 'bye') {
      pcs.get(from)?.close(); pcs.delete(from); dropRemote(from)
    }
  } catch (e) { /* glare/ordering — отдадим следующему сигналу */ }
}

export function useMeeting() {
  const store = useAppStore()

  async function join(meetingId: string) {
    if (activeMeetingId.value) await leave()
    connecting.value = true; mediaError.value = ''
    try {
      localStream.value = await navigator.mediaDevices.getUserMedia({ audio: true, video: true })
    } catch (e: any) {
      // нет камеры/доступа — пробуем только аудио
      try { localStream.value = await navigator.mediaDevices.getUserMedia({ audio: true, video: false }); camOff.value = true }
      catch (e2: any) { mediaError.value = 'Нет доступа к камере/микрофону'; connecting.value = false; throw e2 }
    }
    activeMeetingId.value = meetingId
    muted.value = false
    store.setSignalHandler((sig) => onSignal(store, sig))
    const peers = await store.joinMeeting(meetingId)
    // инициатор: шлём offer каждому уже подключённому
    for (const peerId of peers) {
      const pc = makePeer(store, peerId)
      const offer = await pc.createOffer()
      await pc.setLocalDescription(offer)
      store.sendSignal(meetingId, peerId, 'offer', JSON.stringify(offer))
    }
    connecting.value = false
  }

  async function leave() {
    const mid = activeMeetingId.value
    if (!mid) return
    for (const [peerId, pc] of pcs) { store.sendSignal(mid, peerId, 'bye', ''); pc.close() }
    pcs.clear(); pendingIce.clear()
    localStream.value?.getTracks().forEach((t) => t.stop())
    localStream.value = null
    remotes.value = {}
    activeMeetingId.value = ''
    store.setSignalHandler(null)
    try { await store.leaveMeeting(mid) } catch {}
  }

  function toggleMute() {
    muted.value = !muted.value
    localStream.value?.getAudioTracks().forEach((t) => (t.enabled = !muted.value))
  }
  function toggleCam() {
    camOff.value = !camOff.value
    localStream.value?.getVideoTracks().forEach((t) => (t.enabled = !camOff.value))
  }

  return { activeMeetingId, localStream, remotes, muted, camOff, connecting, mediaError, join, leave, toggleMute, toggleCam }
}
