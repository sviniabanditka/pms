import type { Meta, StoryObj } from '@storybook-vue/nuxt'
import { ref } from 'vue'
import TermCheck from './TermCheck.vue'
import { withDark } from '../.storybook/decorators'

const meta: Meta<typeof TermCheck> = {
  title: 'Atoms/TermCheck',
  component: TermCheck,
  decorators: [withDark],
  args: { label: 'Enable feature', modelValue: false },
}
export default meta
type Story = StoryObj<typeof TermCheck>

// Интерактив: локальный ref, чтобы [x]/[ ] переключался в канвасе.
export const Interactive: Story = {
  render: (args) => ({
    components: { TermCheck },
    setup() { const v = ref(args.modelValue); return { v, args } },
    template: `<TermCheck v-model="v" :label="args.label" />`,
  }),
}
export const Checked: Story = { args: { modelValue: true } }
export const NoLabel: Story = { args: { label: '' } }
export const List: Story = {
  render: () => ({
    components: { TermCheck },
    setup() { return { s: ref({ a: true, b: false, c: true }) } },
    template: `<div style="display:grid; gap:6px">
      <TermCheck v-model="s.a" label="Notifications" />
      <TermCheck v-model="s.b" label="Weekly digest" />
      <TermCheck v-model="s.c" label="Sound" />
    </div>`,
  }),
}
