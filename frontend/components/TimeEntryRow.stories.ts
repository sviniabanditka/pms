import type { Meta, StoryObj } from '@storybook-vue/nuxt'
import TimeEntryRow from './TimeEntryRow.vue'
import { withDark } from '../.storybook/decorators'

const meta: Meta<typeof TimeEntryRow> = {
  title: 'Molecules/TimeEntryRow',
  component: TimeEntryRow,
  decorators: [withDark],
  parameters: { layout: 'fullscreen' },
}
export default meta
type Story = StoryObj<typeof TimeEntryRow>
export const Finished: Story = { args: { entry: { id: 't1', startedAt: '2026-07-03T09:00:00Z', endedAt: '2026-07-03T10:30:00Z', description: 'Pairing on sync fix' } } }
export const Running: Story = { args: { entry: { id: 't2', startedAt: '2026-07-03T11:00:00Z', endedAt: null, description: 'Code review' } } }
