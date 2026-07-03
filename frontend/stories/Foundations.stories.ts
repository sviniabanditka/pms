import type { Meta, StoryObj } from '@storybook-vue/nuxt'
import { useColors } from '~/composables/useColors'
import { withDark } from '../.storybook/decorators'

// Foundations — источник правды дизайн-токенов (цвета/глифы/шрифт), тянутся из
// useColors (не хардкод), поэтому не разъезжаются с приложением.
const meta: Meta = { title: 'Foundations/Tokens', decorators: [withDark] }
export default meta
type Story = StoryObj

const STATUS = ['backlog', 'unstarted', 'started', 'completed', 'canceled']
const PRIO = ['None', 'Low', 'Medium', 'High', 'Urgent']

export const StatusColors: Story = {
  render: () => ({
    setup() { return { c: useColors(), STATUS } },
    template: `<div style="display:grid; gap:10px">
      <div v-for="s in STATUS" :key="s" style="display:flex; align-items:center; gap:12px">
        <span :style="{ color: c.statusColor(s), fontSize:'18px', width:'20px', textAlign:'center' }">{{ c.statusGlyph(s) }}</span>
        <span :style="{ width:'14px', height:'14px', borderRadius:'4px', background: c.statusColor(s) }"></span>
        <code style="color:#c8d0e0">{{ s }}</code>
        <code style="color:#6b7089">{{ c.statusColor(s) }}</code>
      </div>
    </div>`,
  }),
}

export const PriorityColors: Story = {
  render: () => ({
    setup() { return { c: useColors(), PRIO } },
    template: `<div style="display:grid; gap:10px">
      <div v-for="(p,i) in PRIO" :key="p" style="display:flex; align-items:center; gap:12px">
        <span :style="{ color: c.prioColor(i), fontSize:'16px', width:'20px', textAlign:'center' }">{{ c.prioGlyph(i) }}</span>
        <span :style="{ width:'14px', height:'14px', borderRadius:'4px', background: c.prioColor(i) }"></span>
        <code style="color:#c8d0e0">{{ p }}</code>
        <code style="color:#6b7089">{{ c.prioColor(i) }}</code>
      </div>
    </div>`,
  }),
}

export const Typography: Story = {
  render: () => ({
    template: `<div style="display:grid; gap:14px; font-family:'JetBrains Mono',monospace; color:#c8d0e0">
      <div style="font-size:28px; font-weight:700">JetBrains Mono 700 — H1 28</div>
      <div style="font-size:20px; font-weight:600">Semibold 600 — H2 20</div>
      <div style="font-size:16px; font-weight:500">Medium 500 — H3 16</div>
      <div style="font-size:14px">Regular 400 — body 14</div>
      <div style="font-size:12px; color:#6b7089">Tertiary 12 — meta / captions</div>
      <div style="font-size:14px">Digits: 0123456789 · Glyphs: ○◔◐●⊘ ·▂▄▆█ [x] [ ]</div>
    </div>`,
  }),
}
