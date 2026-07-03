import type { Meta, StoryObj } from '@storybook-vue/nuxt'
import IssueList from './IssueList.vue'
import { withStore } from '../../.storybook/decorators'
import { teamId } from '../../stories/fixtures'

const meta: Meta<typeof IssueList> = {
  title: 'Organisms/Views/IssueList',
  component: IssueList,
  decorators: [withStore()],
  parameters: { layout: 'fullscreen' },
  // store.layout='list' по умолчанию → listRows наполняется
  args: { teamId, selectedIssueId: '', selectedIds: new Set<string>(), filtersShown: false, scrollEl: null },
}
export default meta
type Story = StoryObj<typeof IssueList>
export const Default: Story = {}
export const WithSelection: Story = { args: { selectedIds: new Set(['i1', 'i2']) } }
