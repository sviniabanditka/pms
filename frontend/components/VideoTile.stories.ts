import type { Meta, StoryObj } from '@storybook-vue/nuxt'
import VideoTile from './VideoTile.vue'
import { withDark } from '../.storybook/decorators'

// Без реального LiveKit-трека (track:null) рендерится placeholder (инициал + label) —
// именно это состояние и нужно в дизайн-системе (камера выключена / нет трека).
const meta: Meta<typeof VideoTile> = {
  title: 'Atoms/VideoTile',
  component: VideoTile,
  decorators: [withDark],
  render: (args) => ({
    components: { VideoTile },
    setup: () => ({ args }),
    template: `<div style="width:240px; height:160px"><VideoTile v-bind="args" :track="null" /></div>`,
  }),
  args: { track: null, label: 'Ada Lovelace' },
}
export default meta
type Story = StoryObj<typeof VideoTile>

export const CameraOff: Story = { args: { camOff: true } }
export const Muted: Story = { args: { mutedIcon: true } }
