import type { Meta, StoryObj } from '@storybook-vue/nuxt'
import MeetingCall from './MeetingCall.vue'
import { useMeeting } from '~/composables/useMeeting'
import { withStore } from '../.storybook/decorators'

// LiveKit-звонок. useMeeting — singleton-рефы; сетим фейковых участников
// (videoTrack:null → VideoTile рендерит placeholder). meetingById → seeded-store
// (фикстура mt1). Ноль реального LiveKit.
function part(identity: string, name: string, isLocal: boolean, micOn = true, camOn = false) {
  return { identity, name, isLocal, micOn, camOn, videoTrack: null, audioTrack: null }
}

function seedMeeting(parts: Record<string, any>, connecting = false) {
  const m = useMeeting()
  m.activeMeetingId.value = 'mt1'
  m.participants.value = parts
  m.connecting.value = connecting
  m.muted.value = false
  m.camOff.value = true
}

const meta: Meta<typeof MeetingCall> = {
  title: 'Organisms/MeetingCall',
  component: MeetingCall,
  decorators: [withStore()],
  parameters: { layout: 'fullscreen' },
}
export default meta
type Story = StoryObj<typeof MeetingCall>

export const TwoParticipants: Story = {
  render: () => ({
    components: { MeetingCall },
    setup() {
      seedMeeting({ me: part('me', 'Ada Lovelace', true), u2: part('u2', 'Alan Turing', false, false) })
      return {}
    },
    template: '<MeetingCall />',
  }),
}
export const Solo: Story = {
  render: () => ({
    components: { MeetingCall },
    setup() { seedMeeting({ me: part('me', 'Ada Lovelace', true) }); return {} },
    template: '<MeetingCall />',
  }),
}
export const Connecting: Story = {
  render: () => ({
    components: { MeetingCall },
    setup() { seedMeeting({ me: part('me', 'Ada Lovelace', true) }, true); return {} },
    template: '<MeetingCall />',
  }),
}
