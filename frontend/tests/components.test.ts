// @vitest-environment happy-dom
import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import TermCheck from '../components/TermCheck.vue'
import TermBar from '../components/TermBar.vue'

describe('TermCheck', () => {
  it('shows x only when checked', () => {
    expect(mount(TermCheck, { props: { modelValue: false } }).text()).not.toContain('x')
    expect(mount(TermCheck, { props: { modelValue: true } }).text()).toContain('x')
  })
  it('emits the toggled value on click', async () => {
    const w = mount(TermCheck, { props: { modelValue: false } })
    await w.find('button').trigger('click')
    expect(w.emitted('update:modelValue')![0]).toEqual([true])
  })
  it('shows the optional label', () => {
    expect(mount(TermCheck, { props: { modelValue: true, label: 'done' } }).text()).toContain('done')
  })
})

describe('TermBar', () => {
  it('fills proportionally to done/total and labels the percent', () => {
    const w = mount(TermBar, { props: { done: 3, total: 4, width: 12 } })
    expect((w.text().match(/█/g) || []).length).toBe(9) // round(0.75*12)
    expect(w.text()).toContain('75%')
  })
  it('supports an explicit pct', () => {
    const w = mount(TermBar, { props: { pct: 50, width: 12 } })
    expect((w.text().match(/█/g) || []).length).toBe(6)
    expect(w.text()).toContain('50%')
  })
  it('clamps and hides the label when asked', () => {
    const w = mount(TermBar, { props: { pct: 250, width: 10, hideLabel: true } })
    expect((w.text().match(/█/g) || []).length).toBe(10) // clamped to 100%
    expect(w.text()).not.toContain('%')
  })
  it('zero progress renders no filled cells', () => {
    const w = mount(TermBar, { props: { done: 0, total: 5, width: 8 } })
    expect((w.text().match(/█/g) || []).length).toBe(0)
    expect(w.text()).toContain('0%')
  })
})
