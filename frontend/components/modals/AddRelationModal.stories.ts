import type { Meta, StoryObj } from '@storybook-vue/nuxt'
import AddRelationModal from './AddRelationModal.vue'
import { withStore } from '../../.storybook/decorators'
import { issues } from '../../stories/fixtures'

const meta: Meta<typeof AddRelationModal> = {
  title: 'Organisms/Modals/AddRelationModal',
  component: AddRelationModal,
  decorators: [withStore()],
  parameters: { layout: 'fullscreen' },
  args: { modelValue: true, issue: issues.i1 },
}
export default meta
type Story = StoryObj<typeof AddRelationModal>

export const Default: Story = {}
