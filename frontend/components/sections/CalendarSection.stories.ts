import type { Meta, StoryObj } from '@storybook-vue/nuxt'
import CalendarSection from './CalendarSection.vue'
import { withStore } from '../../.storybook/decorators'
import { userId } from '../../stories/fixtures'

const meta: Meta<typeof CalendarSection> = {
  title: 'Organisms/Sections/CalendarSection',
  component: CalendarSection,
  decorators: [withStore()],
  parameters: { layout: 'fullscreen' },
  args: { calUsers: [userId], anchor: '2026-07' },
}
export default meta
type Story = StoryObj<typeof CalendarSection>
export const Month: Story = {}
