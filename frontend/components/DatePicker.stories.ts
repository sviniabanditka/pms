import type { Meta, StoryObj } from '@storybook-vue/nuxt'
import { ref } from 'vue'
import DatePicker from './DatePicker.vue'
import { withDark } from '../.storybook/decorators'

const meta: Meta<typeof DatePicker> = {
  title: 'Atoms/DatePicker',
  component: DatePicker,
  decorators: [withDark],
  args: { modelValue: '2026-07-03', placeholder: 'Pick a date', block: false },
}
export default meta
type Story = StoryObj<typeof DatePicker>

// v-model интерактивно (открой поповер, выбери день).
export const Interactive: Story = {
  render: (args) => ({
    components: { DatePicker },
    setup() { const v = ref(args.modelValue); return { v, args } },
    template: `<div style="width:220px"><DatePicker v-model="v" :placeholder="args.placeholder" :block="args.block" />
      <div style="margin-top:8px; color:#6b7089; font-size:12px">value: {{ v || '∅' }}</div></div>`,
  }),
}
export const Empty: Story = {
  render: () => ({
    components: { DatePicker },
    setup() { return { v: ref('') } },
    template: `<div style="width:220px"><DatePicker v-model="v" placeholder="No date" /></div>`,
  }),
}
export const Block: Story = { args: { block: true } }
