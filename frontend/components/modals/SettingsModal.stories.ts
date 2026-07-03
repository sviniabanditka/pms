import type { Meta, StoryObj } from '@storybook-vue/nuxt'
import SettingsModal from './SettingsModal.vue'
import { withStore } from '../../.storybook/decorators'
import { teamId } from '../../stories/fixtures'

// ТИР-3 ДОКАЗАТЕЛЬСТВО: стор-связанный оркестратор (42 члена стора, 26 экшенов)
// рендерится в Storybook БЕЗ prop-переписи — seeded-store декоратор + мок-транспорт.
// open=true, чтобы модалка была раскрыта в канвасе.
const meta: Meta<typeof SettingsModal> = {
  title: 'Tier3/SettingsModal',
  component: SettingsModal,
  decorators: [withStore()],
  parameters: { layout: 'fullscreen' },
  args: {
    modelValue: true,
    selectedTeamId: teamId,
    notifPrefs: { mention: true, assigned: true, comment: false },
    cfgList: [
      { key: 'inbox', label: 'Inbox', icon: 'i-heroicons-inbox', visible: true },
      { key: 'reports', label: 'Reports', icon: 'i-heroicons-chart-bar', visible: false },
    ],
  },
}
export default meta
type Story = StoryObj<typeof SettingsModal>

export const General: Story = { args: { tab: 'general' } }
export const Members: Story = { args: { tab: 'members' } }
export const Labels: Story = { args: { tab: 'labels' } }
