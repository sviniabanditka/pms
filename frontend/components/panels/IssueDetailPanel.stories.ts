import type { Meta, StoryObj } from '@storybook-vue/nuxt'
import IssueDetailPanel from './IssueDetailPanel.vue'
import { withStore } from '../../.storybook/decorators'

// Самый связанный компонент (50 членов стора, 41 экшен) — рендерится через
// seeded-store + мок-транспорт БЕЗ prop-переписи. Доказательство масштаба Тир-3.
const meta: Meta<typeof IssueDetailPanel> = {
  title: 'Organisms/Panels/IssueDetailPanel',
  component: IssueDetailPanel,
  decorators: [withStore()],
  parameters: { layout: 'fullscreen' },
  args: { issueId: 'i1', linkCopied: false },
}
export default meta
type Story = StoryObj<typeof IssueDetailPanel>
export const Started: Story = {}
export const Done: Story = { args: { issueId: 'i3' } }
