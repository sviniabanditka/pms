import type { Meta, StoryObj } from '@storybook-vue/nuxt'
import PendingAttachments from './PendingAttachments.vue'
import { withDark } from '../.storybook/decorators'

// Фейковые File[] (в браузере new File(...) валиден; URL.createObjectURL — для картинок).
function file(name: string, type: string, size = 2048) {
  const blob = new Blob([new Uint8Array(size)], { type })
  return new File([blob], name, { type })
}

const meta: Meta<typeof PendingAttachments> = {
  title: 'Atoms/PendingAttachments',
  component: PendingAttachments,
  decorators: [withDark],
}
export default meta
type Story = StoryObj<typeof PendingAttachments>

export const Mixed: Story = {
  render: () => ({
    components: { PendingAttachments },
    setup() { return { files: [file('design.pdf', 'application/pdf', 512000), file('notes.txt', 'text/plain', 1200)] } },
    template: `<PendingAttachments :files="files" @remove="() => {}" />`,
  }),
}
export const SingleFile: Story = {
  render: () => ({
    components: { PendingAttachments },
    setup() { return { files: [file('report.csv', 'text/csv', 45000)] } },
    template: `<PendingAttachments :files="files" />`,
  }),
}
