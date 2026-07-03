import { ref } from 'vue'

// Лёгкий сигнал «идёт ли звонок» — БЕЗ импорта livekit-client (1.1MB). Позволяет
// странице гейтить <MeetingCall> (и грузить livekit-чанк) не таща livekit в
// главный бандл. useMeeting пишет сюда activeMeetingId.
export const activeMeetingId = ref('')

export function useMeetingSignal() {
  return { activeMeetingId }
}
