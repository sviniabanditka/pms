import type { Meta, StoryObj } from '@storybook-vue/nuxt'
import MembersModal from './MembersModal.vue'
import { withStore } from '../../.storybook/decorators'

const meta: Meta<typeof MembersModal> = {
  title: 'Organisms/Modals/MembersModal',
  component: MembersModal,
  decorators: [withStore()],
  parameters: { layout: 'fullscreen' },
  args: { modelValue: true, inviteLink: '', inviteEmail: '', inviteRole: 'member' },
}
export default meta
type Story = StoryObj<typeof MembersModal>

export const Default: Story = {}
export const WithInviteLink: Story = { args: { inviteLink: 'https://stdtask.app/invite/abc123' } }
