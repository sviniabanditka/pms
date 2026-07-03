import type { Meta, StoryObj } from '@storybook-vue/nuxt'
import TeamCalendar from './TeamCalendar.vue'
import { withStore } from '../../.storybook/decorators'
import { teamId } from '../../stories/fixtures'

const meta: Meta<typeof TeamCalendar> = {
  title: 'Organisms/Views/TeamCalendar',
  component: TeamCalendar,
  decorators: [withStore()],
  parameters: { layout: 'fullscreen' },
  args: { teamId },
}
export default meta
type Story = StoryObj<typeof TeamCalendar>
export const Default: Story = {}
