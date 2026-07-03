import type { Meta, StoryObj } from '@storybook-vue/nuxt'
import AddLabelModal from './AddLabelModal.vue'
import { withStore } from '../../.storybook/decorators'
import { issues } from '../../stories/fixtures'

const meta: Meta<typeof AddLabelModal> = {
  title: 'Organisms/Modals/AddLabelModal',
  component: AddLabelModal,
  decorators: [withStore()],
  parameters: { layout: 'fullscreen' },
  args: { modelValue: true, issue: issues.i1 },
}
export default meta
type Story = StoryObj<typeof AddLabelModal>

export const Default: Story = {}
