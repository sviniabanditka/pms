import type { Meta, StoryObj } from '@storybook-vue/nuxt'
import IssueBoard from './IssueBoard.vue'
import { withStore } from '../../.storybook/decorators'
import { teamId } from '../../stories/fixtures'

const meta: Meta<typeof IssueBoard> = {
  title: 'Organisms/Views/IssueBoard',
  component: IssueBoard,
  decorators: [withStore()],
  parameters: { layout: 'fullscreen' },
  args: { teamId, selectedIssueId: '' },
}
export default meta
type Story = StoryObj<typeof IssueBoard>
export const Default: Story = {}
