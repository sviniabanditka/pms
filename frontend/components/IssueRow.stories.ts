import type { Meta, StoryObj } from '@storybook-vue/nuxt'
import IssueRow from './IssueRow.vue'
import { withStore } from '../.storybook/decorators'
import { issues } from '../stories/fixtures'

// Молекула, выделенная из IssueList. issue + флаги выбора — пропсами.
const meta: Meta<typeof IssueRow> = {
  title: 'Molecules/IssueRow',
  component: IssueRow,
  decorators: [withStore()],
  parameters: { layout: 'fullscreen' },
  args: { issue: issues.i1 },
}
export default meta
type Story = StoryObj<typeof IssueRow>

export const Default: Story = {}
export const Selected: Story = { args: { selected: true, selectionActive: true } }
export const Active: Story = { args: { active: true } }
export const NoPriorityNoAssignee: Story = { args: { issue: issues.i2 } }
