import type { Meta, StoryObj } from '@storybook-vue/nuxt'
import CycleInsightsPanel from './CycleInsightsPanel.vue'
import { withStore } from '../../.storybook/decorators'
import { cycleId } from '../../stories/fixtures'

const meta: Meta<typeof CycleInsightsPanel> = {
  title: 'Organisms/Panels/CycleInsightsPanel',
  component: CycleInsightsPanel,
  decorators: [withStore()],
  parameters: { layout: 'fullscreen' },
  args: { cycleId },
}
export default meta
type Story = StoryObj<typeof CycleInsightsPanel>
export const Default: Story = {}
