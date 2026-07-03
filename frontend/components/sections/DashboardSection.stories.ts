import type { Meta, StoryObj } from '@storybook-vue/nuxt'
import DashboardSection from './DashboardSection.vue'
import { withStore } from '../../.storybook/decorators'
import { dashboardId } from '../../stories/fixtures'

const meta: Meta<typeof DashboardSection> = {
  title: 'Organisms/Sections/DashboardSection',
  component: DashboardSection,
  decorators: [withStore()],
  parameters: { layout: 'fullscreen' },
  args: { dashboardId },
}
export default meta
type Story = StoryObj<typeof DashboardSection>
export const WithWidgets: Story = {}
export const Empty: Story = { args: { dashboardId: 'dash_none' } }
