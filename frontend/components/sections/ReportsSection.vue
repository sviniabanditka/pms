<script setup lang="ts">
import { useAppStore } from '~/stores/app'
import { parseD, fmtMin } from '~/lib/ui'

// Time report: сводные карточки + дневной график + разбивка + AI-инсайты.
// Стейт (диапазон/группировка/строки) — в useReport (синглтон), т.к. сайдбар
// страницы правит те же refs. Загрузка — при маунте и при смене параметров.
const store = useAppStore()
const {
  repFrom, repTo, repGroup, repUser, repRows, repTotal, repDaily, repLoading,
  loadReport, repLabel, exportReportCsv, repMax, repDailyMax, repEntries, repActiveDays, repAvgPerDay,
  aiThemes, aiOutliers, aiMood, moodPct, loadAiOnce,
} = useReport()

const dayShort = (ds: string) => parseD(ds).toLocaleDateString('en', { month: 'short', day: 'numeric' })
onMounted(() => { loadReport(); loadAiOnce() })
watch([repFrom, repTo, repGroup, repUser], () => loadReport())
</script>

<template>
  <ClientOnly>
  <div class="mx-auto w-full max-w-4xl px-6 py-6">
    <div class="mb-4 flex items-center gap-2">
      <h1 class="text-lg font-semibold text-white">Time report</h1>
      <span class="text-sm text-tertiary">{{ dayShort(repFrom) }} – {{ dayShort(repTo) }}{{ repUser ? ' · ' + (repUser === store.currentUserId ? 'Me' : store.userName(repUser)) : '' }}</span>
      <UButton class="ml-auto" color="gray" size="xs" :disabled="!repRows.length" @click="exportReportCsv">[ export csv ]</UButton>
    </div>

    <!-- сводные карточки -->
    <div class="mb-5 grid grid-cols-2 gap-3 sm:grid-cols-4">
      <div class="rounded-lg border border-subtle bg-elevated p-3">
        <div class="text-xs text-tertiary">Total</div><div class="text-2xl font-semibold text-white">{{ fmtMin(repTotal) }}</div>
      </div>
      <div class="rounded-lg border border-subtle bg-elevated p-3">
        <div class="text-xs text-tertiary">Entries</div><div class="text-2xl font-semibold text-white">{{ repEntries }}</div>
      </div>
      <div class="rounded-lg border border-subtle bg-elevated p-3">
        <div class="text-xs text-tertiary">Active days</div><div class="text-2xl font-semibold text-white">{{ repActiveDays }}</div>
      </div>
      <div class="rounded-lg border border-subtle bg-elevated p-3">
        <div class="text-xs text-tertiary">Avg / active day</div><div class="text-2xl font-semibold text-white">{{ fmtMin(repAvgPerDay) }}</div>
      </div>
    </div>

    <div v-if="repLoading" class="py-10 text-center text-sm text-tertiary">Loading…</div>
    <div v-else-if="!repRows.length" class="py-16 text-center text-sm text-tertiary">No time tracked in this range.</div>
    <template v-else>
      <!-- дневной график -->
      <div class="mb-6">
        <div class="mb-2 text-xs font-medium uppercase tracking-wider text-tertiary"><span class="text-ok"># </span>Daily</div>
        <div class="flex items-end gap-1" style="height: 120px">
          <div v-for="d in repDaily" :key="d.key" class="group/bar relative flex flex-1 flex-col items-center justify-end" :title="`${dayShort(d.key)}: ${fmtMin(d.minutes)}`">
            <div class="w-full rounded-t bg-accent/50 transition-colors group-hover/bar:bg-accent" :style="{ height: (d.minutes / repDailyMax * 96) + 'px' }" />
          </div>
        </div>
        <div class="mt-1 flex gap-1 text-[10px] text-tertiary">
          <div v-for="d in repDaily" :key="d.key" class="flex-1 truncate text-center">{{ parseD(d.key).getDate() }}</div>
        </div>
      </div>

      <!-- разбивка по группе -->
      <div class="mb-2 text-xs font-medium uppercase tracking-wider text-tertiary"><span class="text-ok"># </span>By {{ repGroup === 'user' ? 'person' : repGroup === 'issue' ? 'task' : 'day' }}</div>
      <div class="space-y-1.5">
        <div v-for="r in repRows" :key="r.key" class="flex items-center gap-3">
          <div class="w-56 shrink-0 truncate text-sm" :title="repLabel(r.key)">{{ repLabel(r.key) }}</div>
          <div class="flex-1"><TermBar :width="28" :pct="repMax ? r.minutes / repMax * 100 : 0" hide-label /></div>
          <div class="w-12 shrink-0 text-right text-xs text-tertiary">{{ repTotal ? Math.round(r.minutes / repTotal * 100) : 0 }}%</div>
          <div class="w-20 shrink-0 text-right text-sm tabular-nums text-white">{{ fmtMin(r.minutes) }}</div>
          <div class="w-10 shrink-0 text-right text-xs text-tertiary">{{ r.count }}</div>
        </div>
      </div>
    </template>

    <!-- AI insights (Фаза 4): темы + аномалии + настроение -->
    <div v-if="aiThemes.length || aiOutliers.length || (aiMood && aiMood.n)" class="mt-8 border-t border-subtle pt-6">
      <!-- настроение команды -->
      <div v-if="aiMood && aiMood.n" class="mb-6">
        <div class="mb-2 text-xs font-medium uppercase tracking-wider text-tertiary"><span class="text-accent2">≈ </span>Team mood <span class="text-tertiary">({{ aiMood.n }} comments)</span></div>
        <div class="flex items-center gap-3">
          <div class="h-2 flex-1 overflow-hidden rounded-full bg-elevated">
            <div class="h-full rounded-full transition-all" :style="{ width: moodPct + '%', background: aiMood.avg >= 0 ? '#9ece6a' : '#f7768e' }" />
          </div>
          <span class="w-24 shrink-0 text-right text-xs text-tertiary">{{ aiMood.pos }}+ / {{ aiMood.neg }}− / {{ aiMood.neu }}·</span>
        </div>
      </div>

      <!-- топик-кластеры -->
      <div v-if="aiThemes.length" class="mb-6">
        <div class="mb-2 text-xs font-medium uppercase tracking-wider text-tertiary"><span class="text-accent2">≈ </span>AI themes</div>
        <div class="space-y-1.5">
          <div v-for="t in aiThemes" :key="t.label" class="flex items-center gap-3 rounded-lg border border-subtle bg-elevated px-3 py-2">
            <span class="w-10 shrink-0 text-right text-sm tabular-nums text-white">{{ t.size }}</span>
            <span class="min-w-0 flex-1 truncate text-sm text-secondary" :title="t.label">{{ t.label }}</span>
            <span class="shrink-0 text-xs text-tertiary">{{ (t.sample || []).join(' · ') }}</span>
          </div>
        </div>
      </div>

      <!-- аномалии (выбивающиеся задачи) -->
      <div v-if="aiOutliers.length">
        <div class="mb-2 text-xs font-medium uppercase tracking-wider text-tertiary"><span class="text-accent2">≈ </span>Outliers <span class="text-tertiary">(unusual vs the rest)</span></div>
        <div class="space-y-1.5">
          <div v-for="o in aiOutliers" :key="o.identifier" class="flex items-center gap-3 rounded-lg border border-subtle bg-elevated px-3 py-2">
            <span class="shrink-0 text-xs font-medium text-accent2">{{ o.identifier }}</span>
            <span class="min-w-0 flex-1 truncate text-sm text-secondary" :title="o.title">{{ o.title }}</span>
            <span class="shrink-0 text-xs text-tertiary tabular-nums">{{ Math.round(o.score * 100) }}%</span>
          </div>
        </div>
      </div>
    </div>
  </div>
  </ClientOnly>
</template>
