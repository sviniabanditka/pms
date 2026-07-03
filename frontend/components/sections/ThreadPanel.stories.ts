import type { Meta, StoryObj } from '@storybook-vue/nuxt'
import ThreadPanel from './ThreadPanel.vue'
import { withChat } from '../../.storybook/decorators'
import { chatMessage } from '../../stories/fixtures'

// threadParentId/threadMessages в chat-сторе.
const meta: Meta<typeof ThreadPanel> = {
  title: 'Organisms/Sections/ThreadPanel',
  component: ThreadPanel,
  decorators: [withChat({ threadParentId: 'msg_1', threadMessages: [{ ...chatMessage, id: 'r1', authorId: 'user_1', body: 'Looks good — merging.', reactions: [], replyCount: 0 }] })],
  parameters: { layout: 'fullscreen' },
}
export default meta
type Story = StoryObj<typeof ThreadPanel>
export const Open: Story = {}
