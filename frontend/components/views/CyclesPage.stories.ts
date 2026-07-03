import type { Meta, StoryObj } from '@storybook-vue/nuxt'
import CyclesPage from './CyclesPage.vue'
import { withStore } from '../../.storybook/decorators'
import { teamId } from '../../stories/fixtures'

const meta: Meta<typeof CyclesPage> = {
  title: 'Organisms/Views/CyclesPage',
  component: CyclesPage,
  decorators: [withStore()],
  parameters: { layout: 'fullscreen' },
  args: { teamId },
}
export default meta
type Story = StoryObj<typeof CyclesPage>
export const Default: Story = {}
