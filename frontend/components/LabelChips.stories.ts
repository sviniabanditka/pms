import type { Meta, StoryObj } from '@storybook-vue/nuxt'
import LabelChips from './LabelChips.vue'
import { withDark } from '../.storybook/decorators'
import { labels } from '../stories/fixtures'

const meta: Meta<typeof LabelChips> = {
  title: 'Molecules/LabelChips',
  component: LabelChips,
  decorators: [withDark],
}
export default meta
type Story = StoryObj<typeof LabelChips>

export const Two: Story = { args: { labels: Object.values(labels) } }
export const Many: Story = {
  args: {
    labels: [
      { id: '1', name: 'bug', colorToken: '#f87171' },
      { id: '2', name: 'feature', colorToken: '#60a5fa' },
      { id: '3', name: 'p0', colorToken: '#fbbf24' },
      { id: '4', name: 'backend', colorToken: '#34d399' },
      { id: '5', name: 'needs-design', colorToken: '#a78bfa' },
    ] as any,
  },
}
export const Empty: Story = { args: { labels: [] } }
