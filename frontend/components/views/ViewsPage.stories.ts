import type { Meta, StoryObj } from '@storybook-vue/nuxt'
import ViewsPage from './ViewsPage.vue'
import { withStore } from '../../.storybook/decorators'
import { teamId } from '../../stories/fixtures'

const meta: Meta<typeof ViewsPage> = {
  title: 'Organisms/Views/ViewsPage',
  component: ViewsPage,
  decorators: [withStore()],
  parameters: { layout: 'fullscreen' },
  args: { teamId },
}
export default meta
type Story = StoryObj<typeof ViewsPage>
export const Default: Story = {}
