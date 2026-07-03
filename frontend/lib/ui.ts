// Общие UI-константы страницы и модалок (терминальная тема)
export const PRIO = ['None', 'Low', 'Med', 'High', 'Urgent']
export const cap = (s: string) => (s ? s.charAt(0).toUpperCase() + s.slice(1).replace(/_/g, ' ') : s)
export const labelOpts = (arr: string[]) => arr.map((v) => ({ label: cap(v), value: v }))
// borderless ghost look for issue-detail property controls
export const propSelUi = { padding: { sm: 'px-2 py-1' }, base: 'cursor-pointer' }
// Фон модалок = основной контент (surface). ВАЖНО: нужен dark:-вариант, иначе дефолтный
// `dark:bg-gray-900` побеждает в twMerge при .dark теме (см. mergeConfig=defuTwMerge).
export const modalUi = { background: 'dark:bg-surface', ring: 'ring-1 ring-subtle' }
export const STATE_CATS = ['backlog', 'unstarted', 'started', 'completed', 'canceled']
export const REL_TYPES = ['blocks', 'blocked_by', 'related', 'duplicate', 'duplicate_of']
export const ROLES = ['owner', 'admin', 'member', 'guest']
// короткая дата уведомления/коммента: "Jun 12" (невалидная строка → как есть)
export function notifTime(at: any) {
  const s = String(at); const d = new Date(s)
  if (isNaN(+d)) return s.slice(5, 10)
  return d.toLocaleDateString('en', { month: 'short', day: 'numeric' })
}

// --- Общие дата/время-хелперы (страница + Calendar/Reports секции) ---
export const fmtD = (d: Date) => `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
export const parseD = (ds: string) => new Date(ds + 'T00:00:00')
export const d10 = (s?: string | null) => (s ? String(s).slice(0, 10) : '') // ISO → YYYY-MM-DD для input[type=date]
// ISO (UTC) → локальные дата/время/минуты. События хранятся в UTC, показываем в таймзоне юзера.
export const localDate = (iso: string) => fmtD(new Date(iso))
export const localHM = (iso: string) => { const d = new Date(iso); return `${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}` }
export const localMin = (iso: string) => { const d = new Date(iso); return d.getHours() * 60 + d.getMinutes() }
// минуты → "1h 15m" / "45m"
export const fmtMin = (m: number) => m < 60 ? `${m}m` : `${Math.floor(m / 60)}h ${String(m % 60).padStart(2, '0')}m`
export const CAL_PALETTE = ['#5e6ad2', '#3fb950', '#d29922', '#f85149', '#a371f7', '#1f9cf0', '#db61a2', '#3bb2a8']
export function colorForUser(id: string) { let h = 0; for (const c of id) h = (h * 31 + c.charCodeAt(0)) >>> 0; return CAL_PALETTE[h % CAL_PALETTE.length] }
