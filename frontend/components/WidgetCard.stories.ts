import type { Meta, StoryObj } from '@storybook-vue/nuxt'
import WidgetCard from './WidgetCard.vue'
import { withStore } from '../.storybook/decorators'
import { dashboardId } from '../stories/fixtures'

const meta: Meta<typeof WidgetCard> = {
  title: 'Molecules/WidgetCard',
  component: WidgetCard,
  decorators: [withStore()],
  parameters: { layout: 'fullscreen' },
  args: { dashboardId },
}
export default meta
type Story = StoryObj<typeof WidgetCard>
export const Note: Story = { args: { widget: { id: 'w1', type: 'note', title: 'Note', config: { text: 'Sprint goal: ship v2' } } } }
export const Stat: Story = { args: { widget: { id: 'w2', type: 'stat', title: 'Open issues', config: {} } } }
export const Chart: Story = { args: { widget: { id: 'w3', type: 'chart', title: 'By status', config: {} } } }
