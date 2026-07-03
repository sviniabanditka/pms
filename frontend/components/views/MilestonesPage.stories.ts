import type { Meta, StoryObj } from '@storybook-vue/nuxt'
import MilestonesPage from './MilestonesPage.vue'
import { withStore } from '../../.storybook/decorators'
import { teamId, msId } from '../../stories/fixtures'

const meta: Meta<typeof MilestonesPage> = {
  title: 'Organisms/Views/MilestonesPage',
  component: MilestonesPage,
  decorators: [withStore()],
  parameters: { layout: 'fullscreen' },
  args: { teamId, msId: '' },
}
export default meta
type Story = StoryObj<typeof MilestonesPage>
export const List: Story = {}
export const Detail: Story = { args: { msId } }
