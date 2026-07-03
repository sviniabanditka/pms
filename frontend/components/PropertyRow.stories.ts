import type { Meta, StoryObj } from '@storybook-vue/nuxt'
import PropertyRow from './PropertyRow.vue'
import { withDark } from '../.storybook/decorators'

const meta: Meta<typeof PropertyRow> = {
  title: 'Molecules/PropertyRow',
  component: PropertyRow,
  decorators: [withDark],
  args: { label: 'Status' },
}
export default meta
type Story = StoryObj<typeof PropertyRow>

export const WithControl: Story = {
  render: (args) => ({
    components: { PropertyRow },
    setup: () => ({ args }),
    template: `<div style="width:340px"><PropertyRow :label="args.label">
      <template #icon><span style="color:#ff9e64">◐</span></template>
      <span style="flex:1; color:#c8d0e0">In Progress</span>
    </PropertyRow></div>`,
  }),
}
export const Stacked: Story = {
  render: () => ({
    components: { PropertyRow },
    template: `<div style="width:340px; display:grid; gap:4px">
      <PropertyRow label="Priority"><template #icon><span style="color:#ff9e64">▆</span></template><span style="flex:1;color:#c8d0e0">High</span></PropertyRow>
      <PropertyRow label="Assignee"><template #icon><span style="color:#6b7089">◔</span></template><span style="flex:1;color:#c8d0e0">Ada Lovelace</span></PropertyRow>
      <PropertyRow label="Due date"><template #icon><span style="color:#6b7089">▤</span></template><span style="flex:1;color:#c8d0e0">Aug 1</span></PropertyRow>
    </div>`,
  }),
}
