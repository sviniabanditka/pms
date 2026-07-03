import type { Meta, StoryObj } from '@storybook-vue/nuxt'
import { ref } from 'vue'
import MentionMenu from './MentionMenu.vue'
import { withDark } from '../.storybook/decorators'

// MentionMenu принимает useMention-инстанс `m` (open/items/index/kind/apply).
// Собираем фейковый `m` с нужной реактивной формой.
function fakeMention(kind: '@' | '#', items: any[], index = 0) {
  return {
    open: ref(true),
    items: ref(items),
    index: ref(index),
    kind: ref(kind),
    apply: () => {},
  }
}

const meta: Meta<typeof MentionMenu> = {
  title: 'Atoms/MentionMenu',
  component: MentionMenu,
  decorators: [withDark],
}
export default meta
type Story = StoryObj<typeof MentionMenu>

const wrap = (m: any) => ({
  components: { MentionMenu },
  setup() { return { m } },
  template: `<div style="position:relative; height:220px; width:320px">
    <MentionMenu :m="m" />
  </div>`,
})

export const People: Story = {
  render: () => wrap(fakeMention('@', [
    { label: 'Ada Lovelace', sub: 'ada@acme.dev', avatarKey: null },
    { label: 'Alan Turing', sub: 'alan@acme.dev', avatarKey: null },
    { label: 'Grace Hopper', sub: 'grace@acme.dev', avatarKey: null },
  ], 1)),
}
export const Issues: Story = {
  render: () => wrap(fakeMention('#', [
    { label: 'ENG-1', sub: 'Fix flaky sync test', icon: 'i-heroicons-hashtag' },
    { label: 'ENG-2', sub: 'Add dark-mode toggle', icon: 'i-heroicons-hashtag' },
  ])),
}
export const NoMatches: Story = { render: () => wrap(fakeMention('@', [])) }
