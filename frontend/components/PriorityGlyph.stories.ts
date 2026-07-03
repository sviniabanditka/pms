import type { Meta, StoryObj } from '@storybook-vue/nuxt'
import PriorityGlyph from './PriorityGlyph.vue'
import { withDark } from '../.storybook/decorators'

const meta: Meta<typeof PriorityGlyph> = {
  title: 'Atoms/PriorityGlyph',
  component: PriorityGlyph,
  decorators: [withDark],
  argTypes: { priority: { control: { type: 'range', min: 0, max: 4, step: 1 } } },
  args: { priority: 3 },
}
export default meta
type Story = StoryObj<typeof PriorityGlyph>

export const High: Story = {}
export const All: Story = {
  render: () => ({
    components: { PriorityGlyph },
    setup: () => ({ prios: [['None', 0], ['Low', 1], ['Medium', 2], ['High', 3], ['Urgent', 4]] }),
    template: `<div style="display:flex; gap:16px; align-items:center">
      <span v-for="[label,p] in prios" :key="p" style="display:flex; gap:6px; align-items:center; color:#c8d0e0; font-size:13px">
        <PriorityGlyph :priority="p" /> {{ label }}
      </span>
    </div>`,
  }),
}
