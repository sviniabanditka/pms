import type { Meta, StoryObj } from '@storybook-vue/nuxt'
import TimeEntryModal from './TimeEntryModal.vue'
import { withStore } from '../../.storybook/decorators'

const meta: Meta<typeof TimeEntryModal> = {
  title: 'Organisms/Modals/TimeEntryModal',
  component: TimeEntryModal,
  decorators: [withStore()],
  parameters: { layout: 'fullscreen' },
  args: {
    modelValue: true,
    form: { issueId: 'i1', date: '2026-07-03', start: '09:00', end: '10:30', description: 'Pairing on sync fix' },
  },
}
export default meta
type Story = StoryObj<typeof TimeEntryModal>

export const Default: Story = {}
