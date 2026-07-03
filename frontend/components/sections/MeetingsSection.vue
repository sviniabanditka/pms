<script setup lang="ts">
import { useAppStore } from '~/stores/app'

// Митинги: live/upcoming/past списки. Join (LiveKit) — общий со страницей (сайдбар) → эмит;
// сам звонок (видеогрид) — плавающая панель MeetingCall на странице.
defineEmits<{ (e: 'create'): void; (e: 'join', id: string): void }>()

const store = useAppStore()
const meeting = useMeeting()
</script>

<template>
  <div class="mx-auto max-w-3xl px-6 py-6">
    <div class="mb-5 flex items-center gap-2">
      <h1 class="text-lg font-semibold text-white">Meetings</h1>
      <UButton v-if="store.canWrite" class="ml-auto" color="gray" size="sm" @click="$emit('create')">[ new meeting ]</UButton>
    </div>

    <div v-if="store.liveMeetings.length" class="mb-2 text-xs font-medium uppercase tracking-wider text-tertiary"><span class="text-ok"># </span>Live now</div>
    <div v-for="m in store.liveMeetings" :key="m.id" class="mb-2 flex items-center gap-3 rounded-lg border border-subtle bg-elevated p-3">
      <span class="flex h-2.5 w-2.5 shrink-0 rounded-full bg-danger" />
      <div class="min-w-0 flex-1">
        <div class="truncate font-medium text-white">{{ m.kind === 'huddle' ? 'Huddle' : (m.title || 'Meeting') }}</div>
        <div class="text-xs text-tertiary">{{ store.meetingJoined(m.id).length }} in call · {{ store.meetingParts(m.id).length }} invited</div>
      </div>
      <div class="flex -space-x-1.5">
        <span v-for="p in store.meetingJoined(m.id).slice(0, 4)" :key="p.userId"
          class="flex h-6 w-6 items-center justify-center rounded-full border border-surface bg-active text-[10px] text-white" :title="store.userName(p.userId)">
          {{ store.userName(p.userId).charAt(0).toUpperCase() }}
        </span>
      </div>
      <UButton v-if="meeting.activeMeetingId.value !== m.id" color="green" size="sm" @click="$emit('join', m.id)">[ join ]</UButton>
      <UButton v-else color="gray" size="sm" variant="ghost" disabled>[ in call ]</UButton>
    </div>

    <div v-if="store.upcomingMeetings.length" class="mb-2 mt-6 text-xs font-medium uppercase tracking-wider text-tertiary"><span class="text-ok"># </span>Upcoming</div>
    <div v-for="m in store.upcomingMeetings" :key="m.id" class="group mb-2 flex items-center gap-3 rounded-lg border border-subtle bg-elevated p-3">
      <UIcon name="i-heroicons-calendar" class="shrink-0 text-tertiary" />
      <div class="min-w-0 flex-1">
        <div class="truncate font-medium text-white">{{ m.title || 'Meeting' }}</div>
        <div class="text-xs text-tertiary">
          <span v-if="m.scheduledAt">{{ new Date(m.scheduledAt).toLocaleString() }}<span v-if="m.endsAt">–{{ new Date(m.endsAt).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" }) }}</span> · </span>{{ store.meetingParts(m.id).length }} invited
        </div>
      </div>
      <div class="flex -space-x-1.5">
        <span v-for="p in store.meetingParts(m.id).slice(0, 4)" :key="p.userId"
          class="flex h-6 w-6 items-center justify-center rounded-full border border-surface bg-elevated text-[10px] text-tertiary" :title="store.userName(p.userId)">
          {{ store.userName(p.userId).charAt(0).toUpperCase() }}
        </span>
      </div>
      <UButton color="gray" size="sm" @click="$emit('join', m.id)">[ start ]</UButton>
      <UButton class="opacity-0 group-hover:opacity-100" icon="i-heroicons-trash" color="gray" variant="ghost" size="xs" square title="Delete" @click="store.endMeeting(m.id)" aria-label="Delete" />
    </div>

    <div v-if="store.pastMeetings.length" class="mb-2 mt-6 text-xs font-medium uppercase tracking-wider text-tertiary"><span class="text-ok"># </span>Past</div>
    <div v-for="m in store.pastMeetings" :key="m.id" class="group mb-2 flex items-center gap-3 rounded-lg border border-subtle/60 bg-elevated/40 p-3">
      <UIcon name="i-heroicons-clock" class="shrink-0 text-tertiary" />
      <div class="min-w-0 flex-1">
        <div class="truncate font-medium text-secondary">{{ m.title || 'Meeting' }}</div>
        <div class="text-xs text-tertiary">
          <span v-if="m.scheduledAt">{{ new Date(m.scheduledAt).toLocaleString() }}<span v-if="m.endsAt">–{{ new Date(m.endsAt).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" }) }}</span> · </span>{{ store.meetingParts(m.id).length }} invited
        </div>
      </div>
      <UButton color="gray" size="sm" title="Start again" @click="$emit('join', m.id)" aria-label="Start again">[ restart ]</UButton>
      <UButton class="opacity-0 group-hover:opacity-100" icon="i-heroicons-trash" color="gray" variant="ghost" size="xs" square title="Delete" @click="store.endMeeting(m.id)" aria-label="Delete" />
    </div>

    <div v-if="!store.scheduledMeetings.length && !store.liveMeetings.length" class="py-16 text-center text-sm text-tertiary">
      No meetings yet. Create one or start a huddle from any chat.
    </div>
  </div>
</template>
