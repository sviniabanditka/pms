import { ref, computed } from 'vue'
import { useAppStore } from '~/stores/app'
import { useAiStore } from '~/stores/ai'
import { fmtD, fmtMin } from '~/lib/ui'

// Time-report стейт — модульный синглтон: им пользуются и сайдбар страницы
// (диапазон/группировка/участник), и SectionsReportsSection (таблица+графики).
const repGroup = ref<'user' | 'issue' | 'day'>('user')
const repUser = ref('') // фильтр по участнику (userId или '')
const repPreset = ref('month')
const repFrom = ref(''); const repTo = ref('')
function setRange(preset: string) {
  repPreset.value = preset
  const n = new Date(); const to = fmtD(n)
  if (preset === 'today') { repFrom.value = to; repTo.value = to }
  else if (preset === 'week') { const d = new Date(n); d.setDate(n.getDate() - ((n.getDay() + 6) % 7)); repFrom.value = fmtD(d); repTo.value = to }
  else if (preset === 'month') { repFrom.value = fmtD(new Date(n.getFullYear(), n.getMonth(), 1)); repTo.value = to }
  else if (preset === 'last7') { const d = new Date(n); d.setDate(n.getDate() - 6); repFrom.value = fmtD(d); repTo.value = to }
  else if (preset === 'last30') { const d = new Date(n); d.setDate(n.getDate() - 29); repFrom.value = fmtD(d); repTo.value = to }
  // custom — оставляем как есть
}
const repRows = ref<any[]>([]); const repTotal = ref(0); const repDaily = ref<any[]>([]); const repLoading = ref(false)
const repMax = computed(() => Math.max(1, ...repRows.value.map((r) => r.minutes)))
const repDailyMax = computed(() => Math.max(1, ...repDaily.value.map((r) => r.minutes)))
const repEntries = computed(() => repRows.value.reduce((a, r) => a + r.count, 0))
const repActiveDays = computed(() => repDaily.value.length)
const repAvgPerDay = computed(() => repActiveDays.value ? Math.round(repTotal.value / repActiveDays.value) : 0)
// AI-инсайты (Фаза 4): темы + аномалии + настроение
const aiThemes = ref<any[]>([]); const aiOutliers = ref<any[]>([]); const aiMood = ref<any>(null)
const aiLoaded = ref(false)
const moodPct = computed(() => aiMood.value?.n ? Math.round((aiMood.value.avg + 1) / 2 * 100) : 0)

export function useReport() {
  const store = useAppStore()
  const ai = useAiStore()
  const { err } = useActionWrap()
  if (!repFrom.value) setRange('month')
  async function loadReport() {
    if (!store.activeWorkspaceId) return
    repLoading.value = true
    try {
      const fromIso = new Date(`${repFrom.value}T00:00:00`).toISOString()
      const toD = new Date(`${repTo.value}T00:00:00`); toD.setDate(toD.getDate() + 1) // конец дня включительно
      const toIso = toD.toISOString()
      const uid = repUser.value || undefined
      const grouped = await store.timeReport({ from: fromIso, to: toIso, groupBy: repGroup.value, userId: uid })
      repRows.value = grouped.rows; repTotal.value = grouped.total
      const daily = repGroup.value === 'day' ? grouped : await store.timeReport({ from: fromIso, to: toIso, groupBy: 'day', userId: uid })
      repDaily.value = [...daily.rows].sort((a, b) => String(a.key).localeCompare(String(b.key)))
    } catch (e: any) { err.value = e?.message || String(e) } finally { repLoading.value = false }
  }
  function repLabel(key: string) {
    if (repGroup.value === 'user') return store.userName(key) || key
    if (repGroup.value === 'issue') return key ? (store.issues[key] ? `${store.issues[key].identifier} ${store.issues[key].title}` : '(unknown)') : 'No task'
    return key
  }
  function exportReportCsv() {
    const rows = [['Group', 'Minutes', 'Time', 'Entries'], ...repRows.value.map((r) => [repLabel(r.key), r.minutes, fmtMin(r.minutes), r.count])]
    const csv = rows.map((row) => row.map((c) => `"${String(c).replace(/"/g, '""')}"`).join(',')).join('\n')
    const blob = new Blob([csv], { type: 'text/csv' }); const url = URL.createObjectURL(blob)
    const a = document.createElement('a'); a.href = url; a.download = `time-report-${repFrom.value}_${repTo.value}.csv`; a.click(); URL.revokeObjectURL(url)
  }
  // AI-инсайты грузим один раз за сессию (при первом заходе в Reports)
  function loadAiOnce() {
    if (aiLoaded.value) return
    aiLoaded.value = true
    ai.themes().then((t) => { aiThemes.value = t })
    ai.anomalies().then((o) => { aiOutliers.value = o })
    ai.mood().then((m) => { aiMood.value = m })
  }
  return {
    repGroup, repUser, repPreset, repFrom, repTo, setRange,
    repRows, repTotal, repDaily, repLoading, loadReport, repLabel, exportReportCsv,
    repMax, repDailyMax, repEntries, repActiveDays, repAvgPerDay,
    aiThemes, aiOutliers, aiMood, moodPct, loadAiOnce,
  }
}
