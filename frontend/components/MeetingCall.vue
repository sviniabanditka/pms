<script setup lang="ts">
import { computed, ref } from 'vue'
import { useMeeting } from '~/composables/useMeeting'
import { useAppStore } from '~/stores/app'

const store = useAppStore()
const { activeMeetingId, participants, muted, camOff, connecting, leave, toggleMute, toggleCam } = useMeeting()

const expanded = ref(true)
const meeting = computed(() => store.meetingById(activeMeetingId.value))
// все участники LiveKit-комнаты: свой первым, затем остальные
const tiles = computed(() => {
  const all = Object.values(participants.value)
  return all.sort((a, b) => (a.isLocal ? -1 : b.isLocal ? 1 : a.name.localeCompare(b.name)))
})
function uname(p: any) { return p.isLocal ? p.name + ' (you)' : p.name }
const title = computed(() => meeting.value?.kind === 'huddle' ? 'Huddle' : (meeting.value?.title || 'Meeting'))
</script>

<template>
  <div v-if="activeMeetingId" class="fixed bottom-4 right-4 z-50 flex flex-col overflow-hidden rounded-xl border border-strong bg-surface shadow-2xl"
    :class="expanded ? 'w-[640px]' : 'w-72'">
    <!-- header -->
    <div class="flex items-center gap-2 border-b border-subtle px-3 py-2">
      <span class="flex h-2 w-2 rounded-full bg-red-500" />
      <span class="text-sm font-medium text-white">{{ title }}</span>
      <span class="text-xs text-tertiary">{{ tiles.length }}</span>
      <span v-if="connecting" class="text-xs text-tertiary">connecting…</span>
      <UButton class="ml-auto" :icon="expanded ? 'i-heroicons-minus' : 'i-heroicons-arrows-pointing-out'" color="gray" variant="ghost" size="2xs" square @click="expanded = !expanded" />
    </div>

    <!-- video grid -->
    <div v-if="expanded" class="grid grid-cols-2 gap-1.5 p-1.5">
      <VideoTile v-for="p in tiles" :key="p.identity" :track="p.videoTrack" :muted="p.isLocal" :label="uname(p)" :muted-icon="!p.micOn" :cam-off="!p.camOn" />
    </div>
    <div v-else class="flex flex-wrap gap-1 p-2">
      <span v-for="p in tiles" :key="p.identity"
        class="flex h-7 w-7 items-center justify-center rounded-full bg-elevated text-xs font-medium text-white" :title="p.name">
        {{ p.name.charAt(0).toUpperCase() }}
      </span>
    </div>

    <!-- controls -->
    <div class="flex items-center justify-center gap-2 border-t border-subtle px-3 py-2">
      <UButton :icon="muted ? 'i-heroicons-no-symbol' : 'i-heroicons-microphone'" :color="muted ? 'red' : 'gray'" variant="soft" size="sm" square :title="muted ? 'Unmute' : 'Mute'" @click="toggleMute" />
      <UButton :icon="camOff ? 'i-heroicons-video-camera-slash' : 'i-heroicons-video-camera'" :color="camOff ? 'red' : 'gray'" variant="soft" size="sm" square :title="camOff ? 'Camera on' : 'Camera off'" @click="toggleCam" />
      <UButton icon="i-heroicons-phone-x-mark" color="red" variant="solid" size="sm" @click="leave">Leave</UButton>
    </div>
  </div>
</template>
