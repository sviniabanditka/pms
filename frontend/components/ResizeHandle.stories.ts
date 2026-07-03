import type { Meta, StoryObj } from '@storybook-vue/nuxt'
import ResizeHandle from './ResizeHandle.vue'
import { withDark } from '../.storybook/decorators'

// Атом: drag-ручка ресайза (emit 'down'). Показываем в контексте панели.
const meta: Meta<typeof ResizeHandle> = {
  title: 'Atoms/ResizeHandle',
  component: ResizeHandle,
  decorators: [withDark],
}
export default meta
type Story = StoryObj<typeof ResizeHandle>

export const InPanel: Story = {
  render: () => ({
    components: { ResizeHandle },
    template: `<div style="display:flex; height:180px; border:1px solid #2a2f3a; border-radius:8px; overflow:hidden">
      <div style="width:200px; padding:12px; color:#6b7089; font-size:12px">panel</div>
      <ResizeHandle @down="() => {}" />
      <div style="flex:1; padding:12px; color:#c8d0e0; font-size:12px">content — hover the divider</div>
    </div>`,
  }),
}
