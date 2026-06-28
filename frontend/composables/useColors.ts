// Цвета + иконки статусов/приоритетов (Linear-стиль). Токены — docs/03-frontend/theme-tokens.md.
const STATUS: Record<string, string> = {
  backlog: '#6b747e',
  unstarted: '#9aa3ad',
  started: '#d29922',
  completed: '#5e6ad2',
  canceled: '#6b747e',
}
const STATUS_ICON: Record<string, string> = {
  backlog: 'i-heroicons-minus-circle',
  unstarted: 'i-heroicons-stop-circle',
  started: 'i-heroicons-play-circle',
  completed: 'i-heroicons-check-circle',
  canceled: 'i-heroicons-x-circle',
}
const PRIO = ['#4a525b', '#6b747e', '#d29922', '#e08c3f', '#e5484d'] // none low med high urgent
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
