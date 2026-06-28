import { ref } from 'vue'
import { Room, RoomEvent, Track, type RemoteTrack, type Participant } from 'livekit-client'
import { useAppStore } from '~/stores/app'

// Митинги через LiveKit SFU: 1 upstream + N downstream (без mesh N²). Сервер выдаёт JWT-токен
// на JoinMeeting, медиа полностью на SFU. Singleton-состояние живёт между ре-рендерами.

export type Part = {
  identity: string
  name: string
  isLocal: boolean
  micOn: boolean
  camOn: boolean
  videoTrack: Track | null
  audioTrack: Track | null
}

const activeMeetingId = ref('')
const participants = ref<Record<string, Part>>({})
const muted = ref(false)
const camOff = ref(false)
const connecting = ref(false)
const mediaError = ref('')

let room: Room | null = null

function partOf(p: Participant): Part {
  const cam = p.getTrackPublication(Track.Source.Camera)
  const mic = p.getTrackPublication(Track.Source.Microphone)
  return {
    identity: p.identity,
    name: p.name || p.identity,
    isLocal: p.isLocal,
    camOn: !!cam && !cam.isMuted && !!cam.track,
    micOn: !!mic && !mic.isMuted,
    videoTrack: cam?.track ?? null,
    audioTrack: mic?.track ?? null,
  }
}

function refresh() {
  if (!room) { participants.value = {}; return }
  const out: Record<string, Part> = {}
  out[room.localParticipant.identity] = partOf(room.localParticipant)
  room.remoteParticipants.forEach((p) => { out[p.identity] = partOf(p) })
  participants.value = out
}

export function useMeeting() {
  const store = useAppStore()

  async function join(meetingId: string) {
    if (activeMeetingId.value) await leave()
    connecting.value = true; mediaError.value = ''
    try {
      const { url, token } = await store.joinMeeting(meetingId)
      room = new Room({ adaptiveStream: true, dynacast: true })
      room
        .on(RoomEvent.TrackSubscribed, (track: RemoteTrack) => {
          // аудио надо явно прицепить, чтобы играло; видео цепляет VideoTile
          if (track.kind === 'audio') { const el = track.attach(); el.style.display = 'none'; document.body.appendChild(el) }
          refresh()
        })
        .on(RoomEvent.TrackUnsubscribed, (track: RemoteTrack) => { track.detach().forEach((el) => el.remove()); refresh() })
        .on(RoomEvent.ParticipantConnected, refresh)
        .on(RoomEvent.ParticipantDisconnected, refresh)
        .on(RoomEvent.LocalTrackPublished, refresh)
        .on(RoomEvent.LocalTrackUnpublished, refresh)
        .on(RoomEvent.TrackMuted, refresh)
        .on(RoomEvent.TrackUnmuted, refresh)
        .on(RoomEvent.Disconnected, () => { cleanup() })
      await room.connect(url, token)
      activeMeetingId.value = meetingId
      muted.value = false; camOff.value = false
      try { await room.localParticipant.setMicrophoneEnabled(true) } catch { muted.value = true }
      try { await room.localParticipant.setCameraEnabled(true) } catch { camOff.value = true }
      refresh()
    } catch (e: any) {
      mediaError.value = e?.message || 'Не удалось подключиться к митингу'
      cleanup()
      throw e
    } finally {
      connecting.value = false
    }
  }

  function cleanup() {
    try { room?.disconnect() } catch {}
    room = null
    activeMeetingId.value = ''
    participants.value = {}
  }

  async function leave() {
    const mid = activeMeetingId.value
    cleanup()
    if (mid) { try { await store.leaveMeeting(mid) } catch {} }
  }

  async function toggleMute() {
    if (!room) return
    muted.value = !muted.value
    await room.localParticipant.setMicrophoneEnabled(!muted.value)
    refresh()
  }
  async function toggleCam() {
    if (!room) return
    camOff.value = !camOff.value
    await room.localParticipant.setCameraEnabled(!camOff.value)
    refresh()
  }

  return { activeMeetingId, participants, muted, camOff, connecting, mediaError, join, leave, toggleMute, toggleCam }
}
