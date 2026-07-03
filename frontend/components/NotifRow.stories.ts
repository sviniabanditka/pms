import type { Meta, StoryObj } from '@storybook-vue/nuxt'
import NotifRow from './NotifRow.vue'
import { withStore } from '../.storybook/decorators'

const base = { id: 'n1', type: 'assigned', entityType: 'issue', entityId: 'i1', at: '2026-07-03T09:00:00Z', readAt: null }
const meta: Meta<typeof NotifRow> = {
  title: 'Molecules/NotifRow',
  component: NotifRow,
  decorators: [withStore()],
  parameters: { layout: 'fullscreen' },
  args: { notification: base },
}
export default meta
type Story = StoryObj<typeof NotifRow>
export const Assigned: Story = {}
export const Comment: Story = { args: { notification: { ...base, type: 'commented', entityId: 'i3' } } }
export const Mention: Story = { args: { notification: { ...base, type: 'mention' } } }
export const Read: Story = { args: { notification: { ...base, readAt: '2026-07-03T10:00:00Z' } } }
