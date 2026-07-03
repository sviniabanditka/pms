import type { Meta, StoryObj } from '@storybook-vue/nuxt'
import InboxSection from './InboxSection.vue'
import { withStore } from '../../.storybook/decorators'

const meta: Meta<typeof InboxSection> = {
  title: 'Organisms/Sections/InboxSection',
  component: InboxSection,
  decorators: [withStore()],
  parameters: { layout: 'fullscreen' },
  args: { filter: 'all', notifPrefs: { assigned: true, comment: true } },
}
export default meta
type Story = StoryObj<typeof InboxSection>
export const Default: Story = {}
