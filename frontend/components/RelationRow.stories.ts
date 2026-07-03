import type { Meta, StoryObj } from '@storybook-vue/nuxt'
import RelationRow from './RelationRow.vue'
import { withStore } from '../.storybook/decorators'

const meta: Meta<typeof RelationRow> = {
  title: 'Molecules/RelationRow',
  component: RelationRow,
  decorators: [withStore()],
  parameters: { layout: 'fullscreen' },
  args: { relation: { id: 'r1', label: 'blocks', otherId: 'i2' } },
}
export default meta
type Story = StoryObj<typeof RelationRow>
export const Blocks: Story = {}
export const BlockedBy: Story = { args: { relation: { id: 'r2', label: 'blocked by', otherId: 'i3' } } }
export const Missing: Story = { args: { relation: { id: 'r3', label: 'related', otherId: 'nope' } } }
