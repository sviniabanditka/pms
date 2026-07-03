import type { Meta, StoryObj } from '@storybook-vue/nuxt'
import StatusGlyph from './StatusGlyph.vue'
import { withDark } from '../.storybook/decorators'

const meta: Meta<typeof StatusGlyph> = {
  title: 'Atoms/StatusGlyph',
  component: StatusGlyph,
  decorators: [withDark],
  argTypes: { category: { control: 'select', options: ['backlog', 'unstarted', 'started', 'completed', 'canceled'] } },
  args: { category: 'started' },
}
export default meta
type Story = StoryObj<typeof StatusGlyph>

export const Started: Story = {}
export const All: Story = {
  render: () => ({
    components: { StatusGlyph },
    setup: () => ({ cats: ['backlog', 'unstarted', 'started', 'completed', 'canceled'] }),
    template: `<div style="display:flex; gap:16px; align-items:center">
      <span v-for="c in cats" :key="c" style="display:flex; gap:6px; align-items:center; color:#c8d0e0; font-size:13px">
        <StatusGlyph :category="c" /> {{ c }}
      </span>
    </div>`,
  }),
}
