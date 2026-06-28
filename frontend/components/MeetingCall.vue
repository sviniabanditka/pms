<script setup lang="ts">
import { computed, ref } from 'vue'
import { useMeeting } from '~/composables/useMeeting'
import { useAppStore } from '~/stores/app'

const store = useAppStore()
const { activeMeetingId, localStream, remotes, muted, camOff, connecting, leave, toggleMute, toggleCam } = useMeeting()

const expanded = ref(true)
const meeting = computed(() => store.meetingById(activeMeetingId.value))
const remoteList = computed(() => Object.entries(remotes.value).map(([userId, stream]) => ({ userId, stream })))
const joinedMuted = computed(() => {
  const m: Record<string, boolean> = {}
  for (const p of store.meetingJoined(activeMeetingId.value)) m[p.userId] = false
  return m
})
function userName(id: string) { const u = store.users[id]; return u ? (u.name || u.email) : 'User' }
const meName = computed(() => userName(store.currentUserId))
const title = computed(() => meeting.value?.kind === 'huddle' ? 'Huddle' : (meeting.value?.title || 'Meeting'))
</script>

<template>
  <div v-if="activeMeetingId" class="fixed bottom-4 right-4 z-50 flex flex-col overflow-hidden rounded-xl border border-strong bg-surface shadow-2xl"
    :class="expanded ? 'w-[640px]' : 'w-72'">
    <!-- header -->
    <div class="flex items-center gap-2 border-b border-subtle px-3 py-2">
      <span class="flex h-2 w-2 rounded-full bg-red-500" />
      <span class="text-sm font-medium text-white">{{ title }}</span>
      <span class="text-xs text-tertiary">{{ remoteList.length + 1 }}</span>
      <span v-if="connecting" class="text-xs text-tertiary">connecting…</span>
      <UButton class="ml-auto" :icon="expanded ? 'i-heroicons-minus' : 'i-heroicons-arrows-pointing-out'" color="gray" variant="ghost" size="2xs" square @click="expanded = !expanded" />
    </div>

    <!-- video grid -->
    <div v-if="expanded" class="grid gap-1.5 p-1.5" :class="remoteList.length === 0 ? 'grid-cols-1' : remoteList.length <= 1 ? 'grid-cols-2' : 'grid-cols-2'">
      <VideoTile :stream="localStream" muted :label="meName + ' (you)'" :muted-icon="muted" :cam-off="camOff" />
      <VideoTile v-for="r in remoteList" :key="r.userId" :stream="r.stream" :label="userName(r.userId)" :muted-icon="joinedMuted[r.userId]" />
    </div>
    <div v-else class="flex flex-wrap gap-1 p-2">
      <span v-for="p in store.meetingJoined(activeMeetingId)" :key="p.userId"
        class="flex h-7 w-7 items-center justify-center rounded-full bg-elevated text-xs font-medium text-white" :title="userName(p.userId)">
        {{ userName(p.userId).charAt(0).toUpperCase() }}
      </span>
    </div>

    <!-- controls -->
    <div class="flex items-center justify-center gap-2 border-t border-subtle px-3 py-2">
      <UButton :icon="muted ? 'i-heroicons-microphone-slash' : 'i-heroicons-microphone'" :color="muted ? 'red' : 'gray'" variant="soft" size="sm" square :title="muted ? 'Unmute' : 'Mute'" @click="toggleMute" />
      <UButton :icon="camOff ? 'i-heroicons-video-camera-slash' : 'i-heroicons-video-camera'" :color="camOff ? 'red' : 'gray'" variant="soft" size="sm" square :title="camOff ? 'Camera on' : 'Camera off'" @click="toggleCam" />
      <UButton icon="i-heroicons-phone-x-mark" color="red" variant="solid" size="sm" @click="leave">Leave</UButton>
    </div>
  </div>
</template>
