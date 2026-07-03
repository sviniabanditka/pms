import type { Meta, StoryObj } from '@storybook-vue/nuxt'
import MeetingsSection from './MeetingsSection.vue'
import { withStore } from '../../.storybook/decorators'

const meta: Meta<typeof MeetingsSection> = {
  title: 'Organisms/Sections/MeetingsSection',
  component: MeetingsSection,
  decorators: [withStore()],
  parameters: { layout: 'fullscreen' },
}
export default meta
type Story = StoryObj<typeof MeetingsSection>
export const Default: Story = {}
