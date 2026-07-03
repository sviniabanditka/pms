import type { Meta, StoryObj } from '@storybook-vue/nuxt'
import IcalModal from './IcalModal.vue'
import { withStore } from '../../.storybook/decorators'

const meta: Meta<typeof IcalModal> = {
  title: 'Organisms/Modals/IcalModal',
  component: IcalModal,
  decorators: [withStore()],
  parameters: { layout: 'fullscreen' },
  args: { modelValue: true, url: 'https://stdtask.app/ical/u_1/feed.ics' },
}
export default meta
type Story = StoryObj<typeof IcalModal>

export const Default: Story = {}
