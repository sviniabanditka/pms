import { ref, computed, nextTick, type Ref } from 'vue'
import { useAppStore } from '~/stores/app'

// Инлайн @-пикер (юзеры) и #-пикер (сущности) для любого textarea.
// detect() — на input; onKeydown() — навигация (true = событие поглощено).
export function useMention(draft: Ref<string>, elRef: Ref<HTMLTextAreaElement | null>) {
  const store = useAppStore()
  const open = ref(false)
  const kind = ref<'@' | '#'>('@')
  const query = ref('')
  const index = ref(0)
  let triggerPos = -1

  const items = computed(() => {
    const q = query.value.toLowerCase()
    if (kind.value === '@') {
      return Object.values(store.users)
        .filter((u: any) => (u.name || u.email || '').toLowerCase().includes(q))
        .slice(0, 8)
        .map((u: any) => ({ label: u.name || u.email, sub: u.email, icon: 'i-heroicons-user', color: '', avatarKey: u.avatarKey, token: `@[${u.name || u.email}](${u.id}) ` }))
    }
    const res: any[] = []
    for (const i of Object.values(store.issues) as any[]) {
      if (i.archivedAt || i.deletedAt) continue
      if (i.identifier?.toLowerCase().includes(q) || (i.title || '').toLowerCase().includes(q))
        res.push({ label: i.identifier, sub: i.title, icon: 'i-heroicons-check-circle', token: `#${i.identifier} ` })
    }
    for (const d of Object.values(store.documents) as any[]) {
      if (d.deletedAt) continue
      if ((d.title || '').toLowerCase().includes(q)) res.push({ label: d.title || 'Untitled', sub: 'doc', icon: 'i-heroicons-document-text', token: `#[${d.title || 'doc'}](doc:${d.id}) ` })
    }
    for (const m of Object.values(store.meetings) as any[]) {
      if (m.deletedAt || m.status === 'ended') continue
      const name = m.kind === 'huddle' ? 'Huddle' : (m.title || 'Meeting')
      if (name.toLowerCase().includes(q)) res.push({ label: name, sub: 'meeting', icon: 'i-heroicons-video-camera', token: `#[${name}](meeting:${m.id}) ` })
    }
    for (const c of Object.values(store.channels) as any[]) {
      if (c.deletedAt || c.kind === 'dm') continue
      if ((c.name || '').toLowerCase().includes(q)) res.push({ label: '#' + c.name, sub: 'channel', icon: 'i-heroicons-hashtag', token: `#[${c.name}](channel:${c.id}) ` })
    }
    return res.slice(0, 8)
  })

  function detect() {
    const el = elRef.value
    if (!el) { open.value = false; return }
    const pos = el.selectionStart ?? 0
    const before = draft.value.slice(0, pos)
    const m = before.match(/([@#])([\w-]*)$/)
    if (m && (m.index === 0 || /[\s(]/.test(before[m.index! - 1]))) {
      kind.value = m[1] as any
      query.value = m[2]
      triggerPos = m.index!
      index.value = 0
      open.value = items.value.length > 0
    } else {
      open.value = false
    }
  }

  function apply(it: any) {
    const el = elRef.value
    if (!el) return
    const pos = el.selectionStart ?? draft.value.length
    const before = draft.value.slice(0, triggerPos)
    const after = draft.value.slice(pos)
    draft.value = before + it.token + after
    open.value = false
    nextTick(() => { el.focus(); const c = (before + it.token).length; el.selectionStart = el.selectionEnd = c })
  }

  // вернуть true если поглотили событие (parent не шлёт/не делает newline)
  function onKeydown(e: KeyboardEvent): boolean {
    if (!open.value || !items.value.length) return false
    if (e.key === 'ArrowDown') { e.preventDefault(); index.value = (index.value + 1) % items.value.length; return true }
    if (e.key === 'ArrowUp') { e.preventDefault(); index.value = (index.value - 1 + items.value.length) % items.value.length; return true }
    if (e.key === 'Enter' || e.key === 'Tab') { e.preventDefault(); apply(items.value[index.value]); return true }
    if (e.key === 'Escape') { open.value = false; return true }
    return false
  }

  return { open, kind, query, items, index, detect, apply, onKeydown }
}
