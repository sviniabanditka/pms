import type { Meta, StoryObj } from '@storybook-vue/nuxt'
import CreateModal from './CreateModal.vue'
import { withStore } from '../../.storybook/decorators'
import { teamId } from '../../stories/fixtures'

// Глобальный create-свитч (поля под выбранную сущность). withStore для teamList и т.п.
const meta: Meta<typeof CreateModal> = {
  title: 'Organisms/Modals/CreateModal',
  component: CreateModal,
  decorators: [withStore()],
  parameters: { layout: 'fullscreen' },
  args: { modelValue: true, type: 'issue', defaultTeam: teamId },
}
export default meta
type Story = StoryObj<typeof CreateModal>

export const Issue: Story = { args: { type: 'issue' } }
export const Doc: Story = { args: { type: 'doc' } }
export const Team: Story = { args: { type: 'team' } }
export const Workspace: Story = { args: { type: 'workspace' } }
