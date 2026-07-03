import type { Meta, StoryObj } from '@storybook-vue/nuxt'
import ReportsSection from './ReportsSection.vue'
import { withStore } from '../../.storybook/decorators'

// useReport дёргает time.report + ai.themes/anomalies/mood → мок-транспорт (benign).
const meta: Meta<typeof ReportsSection> = {
  title: 'Organisms/Sections/ReportsSection',
  component: ReportsSection,
  decorators: [withStore()],
  parameters: { layout: 'fullscreen' },
}
export default meta
type Story = StoryObj<typeof ReportsSection>
export const Default: Story = {}
