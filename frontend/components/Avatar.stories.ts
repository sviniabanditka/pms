import type { Meta, StoryObj } from '@storybook-vue/nuxt'
import Avatar from './Avatar.vue'
import { withDark } from '../.storybook/decorators'

// Атом: props-pure, ноль стора → образцовый design-system компонент.
const meta: Meta<typeof Avatar> = {
  title: 'Atoms/Avatar',
  component: Avatar,
  decorators: [withDark],
  argTypes: {
    size: { control: { type: 'range', min: 16, max: 96, step: 4 } },
    name: { control: 'text' },
    avatarKey: { control: 'text' },
  },
  args: { name: 'Ada Lovelace', size: 32 },
}
export default meta
type Story = StoryObj<typeof Avatar>

export const Initials: Story = {}
export const Large: Story = { args: { size: 64 } }
export const NoName: Story = { args: { name: '' } }
export const Row: Story = {
  render: (args) => ({
    components: { Avatar },
    setup: () => ({ args, names: ['Ada Lovelace', 'Alan Turing', 'Grace Hopper', ''] }),
    template: `<div style="display:flex; gap:8px; align-items:center">
      <Avatar v-for="n in names" :key="n" :name="n" :size="args.size" />
    </div>`,
  }),
}
