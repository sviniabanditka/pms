import type { Meta, StoryObj } from '@storybook-vue/nuxt'
import CalendarEventModal from './CalendarEventModal.vue'
import { withStore } from '../../.storybook/decorators'

const base = {
  id: '', title: '', description: '', kind: 'event', participants: [] as string[],
  date: '2026-07-06', start: '10:00', end: '11:00', allDay: false, recurrence: 'none', recUntil: '',
}

const meta: Meta<typeof CalendarEventModal> = {
  title: 'Organisms/Modals/CalendarEventModal',
  component: CalendarEventModal,
  decorators: [withStore()],
  parameters: { layout: 'fullscreen' },
  args: { modelValue: true, form: { ...base } },
}
export default meta
type Story = StoryObj<typeof CalendarEventModal>

export const NewEvent: Story = {}
export const EditEvent: Story = { args: { form: { ...base, id: 'ev_1', title: 'Sprint review', description: 'demo + retro' } } }
export const Meeting: Story = { args: { form: { ...base, kind: 'meeting', title: 'Design sync' } } }
