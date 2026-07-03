import type { Meta, StoryObj } from '@storybook-vue/nuxt'
import Burndown from './Burndown.vue'
import { withStore } from '../.storybook/decorators'
import { cycleId } from '../stories/fixtures'

// Молекула: cycleBurndown-геттер читает cycleSnapshots-бакет → seeded-store.
const meta: Meta<typeof Burndown> = {
  title: 'Molecules/Burndown',
  component: Burndown,
  decorators: [withStore()],
  args: { cycleId },
}
export default meta
type Story = StoryObj<typeof Burndown>

export const Default: Story = {}
export const Empty: Story = {
  decorators: [withStore({ cycleSnapshots: {} })],
  args: { cycleId },
}
