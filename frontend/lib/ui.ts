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
