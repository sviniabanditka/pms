import type { Meta, StoryObj } from '@storybook-vue/nuxt'
import ChatSection from './ChatSection.vue'
import { withChat } from '../../.storybook/decorators'

// Читает chat-стор (activeChannelId/messages) → withChat сидит и app, и chat.
const meta: Meta<typeof ChatSection> = {
  title: 'Organisms/Sections/ChatSection',
  component: ChatSection,
  decorators: [withChat()],
  parameters: { layout: 'fullscreen' },
  args: { linkCopied: false, firstUnreadMsgId: '' },
}
export default meta
type Story = StoryObj<typeof ChatSection>
export const Channel: Story = {}
