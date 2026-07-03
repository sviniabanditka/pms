import type { Meta, StoryObj } from '@storybook-vue/nuxt'
import WidgetModal from './WidgetModal.vue'
import { withStore } from '../../.storybook/decorators'

const meta: Meta<typeof WidgetModal> = {
  title: 'Organisms/Modals/WidgetModal',
  component: WidgetModal,
  decorators: [withStore()],
  parameters: { layout: 'fullscreen' },
  args: { modelValue: true, dashboardId: 'dash_1' },
}
export default meta
type Story = StoryObj<typeof WidgetModal>

export const Default: Story = {}
