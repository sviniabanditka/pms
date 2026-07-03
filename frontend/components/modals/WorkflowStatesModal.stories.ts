import type { Meta, StoryObj } from '@storybook-vue/nuxt'
import WorkflowStatesModal from './WorkflowStatesModal.vue'
import { withStore } from '../../.storybook/decorators'
import { teamId } from '../../stories/fixtures'

const meta: Meta<typeof WorkflowStatesModal> = {
  title: 'Organisms/Modals/WorkflowStatesModal',
  component: WorkflowStatesModal,
  decorators: [withStore()],
  parameters: { layout: 'fullscreen' },
  args: { modelValue: true, teamId, teamKey: 'ENG' },
}
export default meta
type Story = StoryObj<typeof WorkflowStatesModal>

export const Default: Story = {}
