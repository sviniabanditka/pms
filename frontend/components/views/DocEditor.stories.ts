import type { Meta, StoryObj } from '@storybook-vue/nuxt'
import DocEditor from './DocEditor.vue'
import { withStore } from '../../.storybook/decorators'
import { docId } from '../../stories/fixtures'

const meta: Meta<typeof DocEditor> = {
  title: 'Organisms/Views/DocEditor',
  component: DocEditor,
  decorators: [withStore()],
  parameters: { layout: 'fullscreen' },
  args: { docId, linkCopied: false },
}
export default meta
type Story = StoryObj<typeof DocEditor>
export const Default: Story = {}
