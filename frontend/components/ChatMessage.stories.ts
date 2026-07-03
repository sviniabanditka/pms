import type { Meta, StoryObj } from '@storybook-vue/nuxt'
import ChatMessage from './ChatMessage.vue'
import { withStore } from '../.storybook/decorators'
import { chatMessage } from '../stories/fixtures'

// Молекула: читает стор (author name, currentUserId для reaction.mine, chat-экшены).
// seeded-store декоратор + m-prop фикстура.
const meta: Meta<typeof ChatMessage> = {
  title: 'Molecules/ChatMessage',
  component: ChatMessage,
  decorators: [withStore()],
  args: { m: chatMessage, canReply: true },
}
export default meta
type Story = StoryObj<typeof ChatMessage>

export const WithReactions: Story = {}
export const OwnMessage: Story = {
  args: { m: { ...chatMessage, authorId: 'user_1' } }, // currentUserId → показывает edit/delete
}
export const NoReactions: Story = {
  args: { m: { ...chatMessage, reactions: [], replyCount: 0 } },
}
