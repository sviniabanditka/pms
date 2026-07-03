import type { Meta, StoryObj } from '@storybook-vue/nuxt'
import { ref } from 'vue'
import TimePicker from './TimePicker.vue'
import { withDark } from '../.storybook/decorators'

const meta: Meta<typeof TimePicker> = {
  title: 'Atoms/TimePicker',
  component: TimePicker,
  decorators: [withDark],
  args: { modelValue: '09:00', step: 15 },
}
export default meta
type Story = StoryObj<typeof TimePicker>

export const Interactive: Story = {
  render: (args) => ({
    components: { TimePicker },
    setup() { const v = ref(args.modelValue); return { v, args } },
    template: `<div style="width:180px"><TimePicker v-model="v" :step="args.step" />
      <div style="margin-top:8px; color:#8b91a8; font-size:12px">value: {{ v || '∅' }}</div></div>`,
  }),
}
// `from` → показывает длительность рядом.
export const WithDuration: Story = {
  render: () => ({
    components: { TimePicker },
    setup() { return { from: ref('09:00'), to: ref('10:30') } },
    template: `<div style="display:flex; gap:8px; width:400px">
      <TimePicker v-model="from" />
      <span style="color:#8b91a8">–</span>
      <TimePicker v-model="to" :from="from" />
    </div>`,
  }),
}
export const Empty: Story = {
  render: () => ({
    components: { TimePicker },
    setup() { return { v: ref('') } },
    template: `<div style="width:180px"><TimePicker v-model="v" placeholder="--:--" /></div>`,
  }),
}
