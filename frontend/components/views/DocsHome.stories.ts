import type { Meta, StoryObj } from '@storybook-vue/nuxt'
import DocsHome from './DocsHome.vue'
import { withStore } from '../../.storybook/decorators'
import { teamId } from '../../stories/fixtures'

const meta: Meta<typeof DocsHome> = {
  title: 'Organisms/Views/DocsHome',
  component: DocsHome,
  decorators: [withStore()],
  parameters: { layout: 'fullscreen' },
  args: { teamId },
}
export default meta
type Story = StoryObj<typeof DocsHome>
export const Default: Story = {}
