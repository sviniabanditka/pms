import { describe, it, expect, beforeEach, vi } from 'vitest'
import { ref } from 'vue'
import { setActivePinia, createPinia } from 'pinia'
import { useAppStore } from '../stores/app'
import { useMention } from '../composables/useMention'

beforeEach(() => setActivePinia(createPinia()))

function fakeEl(pos: number) {
  return { selectionStart: pos, selectionEnd: pos, focus() {} } as any
}

function setup(text: string, pos: number, users: any[] = []) {
  const store = useAppStore()
  for (const u of users) store.users[u.id] = u
  const draft = ref(text)
  const el = ref(fakeEl(pos))
  return { store, draft, ...useMention(draft, el), draftRef: draft }
}

describe('useMention — detect', () => {
  it('opens on @query with a leading boundary and matching users', () => {
    const m = setup('hi @ma', 6, [{ id: 'u1', name: 'Maya', email: 'maya@e.com' }])
    m.detect()
    expect(m.kind.value).toBe('@')
    expect(m.query.value).toBe('ma')
    expect(m.open.value).toBe(true)
    expect(m.items.value.length).toBe(1)
  })

  it('does not open when @ is glued to a previous word (no boundary)', () => {
    const m = setup('email@ma', 8, [{ id: 'u1', name: 'Maya', email: 'maya@e.com' }])
    m.detect()
    expect(m.open.value).toBe(false)
  })

  it('does not open when no users match the query', () => {
    const m = setup('hi @zzz', 7, [{ id: 'u1', name: 'Maya', email: 'maya@e.com' }])
    m.detect()
    expect(m.open.value).toBe(false)
  })
})

describe('useMention — apply', () => {
  it('replaces the @query with the item token', () => {
    const m = setup('hi @ma', 6, [{ id: 'u1', name: 'Maya', email: 'maya@e.com' }])
    m.detect()
    m.apply(m.items.value[0])
    expect(m.draftRef.value).toBe('hi @[Maya](u1) ')
    expect(m.open.value).toBe(false)
  })
})

describe('useMention — onKeydown', () => {
  const users = [
    { id: 'u1', name: 'Ann', email: 'a@e.com' },
    { id: 'u2', name: 'Andy', email: 'b@e.com' },
  ]
  it('ArrowDown/ArrowUp cycle the selected index', () => {
    const m = setup('@a', 2, users)
    m.detect()
    expect(m.items.value.length).toBe(2)
    const down = { key: 'ArrowDown', preventDefault: vi.fn() } as any
    expect(m.onKeydown(down)).toBe(true)
    expect(m.index.value).toBe(1)
    const up = { key: 'ArrowUp', preventDefault: vi.fn() } as any
    m.onKeydown(up)
    expect(m.index.value).toBe(0)
  })
  it('Enter applies the highlighted item', () => {
    const m = setup('@a', 2, users)
    m.detect()
    m.onKeydown({ key: 'Enter', preventDefault: vi.fn() } as any)
    expect(m.draftRef.value).toContain('@[Ann](u1)')
    expect(m.open.value).toBe(false)
  })
  it('Escape closes; passthrough returns false when closed', () => {
    const m = setup('@a', 2, users)
    m.detect()
    expect(m.onKeydown({ key: 'Escape', preventDefault: vi.fn() } as any)).toBe(true)
    expect(m.open.value).toBe(false)
    expect(m.onKeydown({ key: 'x', preventDefault: vi.fn() } as any)).toBe(false)
  })
})
