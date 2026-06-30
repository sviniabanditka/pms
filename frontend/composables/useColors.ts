// Цвета + иконки статусов/приоритетов (Linear-стиль). Токены — docs/03-frontend/theme-tokens.md.
// Tokyo-Night синтаксис-палитра (синк с лендингом/токенами)
const STATUS: Record<string, string> = {
  backlog: '#6b7089', // tertiary
  unstarted: '#7dcfff', // cyan
  started: '#ff9e64', // orange
  completed: '#9ece6a', // green
  canceled: '#565f89', // faint
}
const STATUS_ICON: Record<string, string> = {
  backlog: 'i-heroicons-minus-circle',
  unstarted: 'i-heroicons-stop-circle',
  started: 'i-heroicons-play-circle',
  completed: 'i-heroicons-check-circle',
  canceled: 'i-heroicons-x-circle',
}
const PRIO = ['#565f89', '#7aa2f7', '#e0af68', '#ff9e64', '#f7768e'] // none low med high urgent
const PRIO_ICON = [
  'i-heroicons-minus',
  'i-heroicons-bars-2',
  'i-heroicons-bars-3',
  'i-heroicons-signal',
  'i-heroicons-exclamation-triangle',
]

export function useColors() {
  return {
    statusColor: (cat: string) => STATUS[cat] || '#6b747e',
    statusIcon: (cat: string) => STATUS_ICON[cat] || 'i-heroicons-minus-circle',
    prioColor: (p: number) => PRIO[p] ?? PRIO[0],
    prioIcon: (p: number) => PRIO_ICON[p] ?? PRIO_ICON[0],
  }
}
