import type { Meta, StoryObj } from '@storybook-vue/nuxt'
import TermBar from './TermBar.vue'
import { withDark } from '../.storybook/decorators'

const meta: Meta<typeof TermBar> = {
  title: 'Atoms/TermBar',
  component: TermBar,
  decorators: [withDark],
  argTypes: {
    pct: { control: { type: 'range', min: 0, max: 100, step: 1 } },
    width: { control: { type: 'range', min: 4, max: 40, step: 1 } },
  },
  args: { pct: 60, width: 12 },
}
export default meta
type Story = StoryObj<typeof TermBar>

export const ByPercent: Story = {}
export const ByCount: Story = { args: { pct: undefined, done: 3, total: 8 } }
export const Empty: Story = { args: { pct: 0 } }
export const Full: Story = { args: { pct: 100 } }
export const NoLabel: Story = { args: { hideLabel: true } }
export const Scale: Story = {
  render: () => ({
    components: { TermBar },
    template: `<div style="display:grid; gap:6px">
      <TermBar v-for="p in [0,20,40,60,80,100]" :key="p" :pct="p" :width="16" />
    </div>`,
  }),
}
