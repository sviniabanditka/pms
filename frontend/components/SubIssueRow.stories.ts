import type { Meta, StoryObj } from '@storybook-vue/nuxt'
import SubIssueRow from './SubIssueRow.vue'
import { withStore } from '../.storybook/decorators'
import { issues } from '../stories/fixtures'

const meta: Meta<typeof SubIssueRow> = {
  title: 'Molecules/SubIssueRow',
  component: SubIssueRow,
  decorators: [withStore()],
  parameters: { layout: 'fullscreen' },
  args: { issue: issues.i1 },
}
export default meta
type Story = StoryObj<typeof SubIssueRow>
export const Open: Story = {}
export const Done: Story = { args: { issue: { ...issues.i3, completedAt: '2026-07-02' } } }
