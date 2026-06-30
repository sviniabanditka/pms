import { describe, it, expect } from 'vitest'
import { useColors } from '../composables/useColors'

const c = useColors()

describe('useColors — status', () => {
  it('known categories map to distinct colors', () => {
    const cats = ['backlog', 'unstarted', 'started', 'completed', 'canceled']
    const colors = cats.map(c.statusColor)
    expect(new Set(colors).size).toBe(cats.length)
  })
  it('unknown category falls back', () => {
    expect(c.statusColor('nope')).toBe('#6b747e')
    expect(c.statusGlyph('nope')).toBe('○')
  })
  it('glyphs for each category', () => {
    expect(c.statusGlyph('completed')).toBe('●')
    expect(c.statusGlyph('canceled')).toBe('⊘')
  })
})

describe('useColors — priority', () => {
  it('priorities 0..4 map to colors', () => {
    for (let p = 0; p <= 4; p++) expect(c.prioColor(p)).toMatch(/^#/)
  })
  it('out-of-range priority falls back to none', () => {
    expect(c.prioColor(99)).toBe(c.prioColor(0))
    expect(c.prioGlyph(99)).toBe(c.prioGlyph(0))
  })
  it('priority glyphs escalate', () => {
    expect(c.prioGlyph(0)).toBe('·')
    expect(c.prioGlyph(4)).toBe('█')
  })
})
