import type { Meta, StoryObj } from '@storybook-vue/nuxt'
import Attachments from './Attachments.vue'
import { withStore } from '../.storybook/decorators'

// Молекула: attachmentsFor читает attachments-бакет (фикстуры на issue i1) →
// seeded-store. Аплоад/удаление уходят в мок-транспорт.
const meta: Meta<typeof Attachments> = {
  title: 'Molecules/Attachments',
  component: Attachments,
  decorators: [withStore()],
  args: { entityType: 'issue', entityId: 'i1' },
}
export default meta
type Story = StoryObj<typeof Attachments>

export const WithFiles: Story = {}
export const Readonly: Story = { args: { readonly: true } }
export const Empty: Story = { args: { entityId: 'i_none' } }
