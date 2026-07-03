import type { Meta, StoryObj } from '@storybook-vue/nuxt'
import AttachmentTile from './AttachmentTile.vue'
import { withStore } from '../.storybook/decorators'

const meta: Meta<typeof AttachmentTile> = {
  title: 'Molecules/AttachmentTile',
  component: AttachmentTile,
  decorators: [withStore()],
  parameters: { layout: 'fullscreen' },
}
export default meta
type Story = StoryObj<typeof AttachmentTile>
export const File: Story = { args: { attachment: { id: 'a1', filename: 'trace.log', s3Key: 'k1', contentType: 'text/plain', size: 24000 } } }
export const Pdf: Story = { args: { attachment: { id: 'a2', filename: 'spec.pdf', s3Key: 'k2', contentType: 'application/pdf', size: 512000 } } }
export const Archive: Story = { args: { attachment: { id: 'a3', filename: 'build.zip', s3Key: 'k3', contentType: 'application/zip', size: 4200000 } } }
export const Readonly: Story = { args: { attachment: { id: 'a4', filename: 'notes.txt', s3Key: 'k4', contentType: 'text/plain', size: 1200 }, readonly: true } }
