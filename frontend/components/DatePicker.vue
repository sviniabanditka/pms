<script setup lang="ts">
// Кастомный date-picker: инпут "Mon, Jun 29" + поповер-календарь (месяц, Today, ←↑↓→).
// v-model = 'YYYY-MM-DD' (или '').
const props = withDefaults(defineProps<{ modelValue: string; placeholder?: string; block?: boolean }>(), {
  modelValue: '', placeholder: 'Pick a date', block: false,
})
const emit = defineEmits<{ (e: 'update:modelValue', v: string): void }>()

const fmtD = (d: Date) => `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
const parseD = (ds: string) => new Date(ds + 'T00:00:00')
const todayStr = () => fmtD(new Date())

const display = computed(() => props.modelValue ? parseD(props.modelValue).toLocaleDateString('en', { weekday: 'short', month: 'short', day: 'numeric' }) : '')

// показываемый месяц (первое число)
const view = ref(new Date())
watchEffect(() => { view.value = props.modelValue ? parseD(props.modelValue) : new Date() })
const monthLabel = computed(() => view.value.toLocaleDateString('en', { month: 'long', year: 'numeric' }))
function shift(n: number) { const d = new Date(view.value); d.setMonth(d.getMonth() + n); d.setDate(1); view.value = d }
function goToday() { view.value = new Date() }

const WD = ['Su', 'Mo', 'Tu', 'We', 'Th', 'Fr', 'Sa']
const cells = computed(() => {
  const y = view.value.getFullYear(), m = view.value.getMonth()
  const start = new Date(y, m, 1); start.setDate(1 - start.getDay()) // неделя с воскресенья
  const ts = todayStr()
  return Array.from({ length: 42 }, (_, i) => {
    const d = new Date(start); d.setDate(start.getDate() + i); const ds = fmtD(d)
    return { ds, day: d.getDate(), inMonth: d.getMonth() === m, isToday: ds === ts, sel: ds === props.modelValue }
  })
})
function pick(ds: string, close: () => void) { emit('update:modelValue', ds); close() }
</script>

<template>
  <UPopover :popper="{ placement: 'bottom-start' }" :ui="{ width: '' }">
    <button type="button"
      class="flex h-8 items-center gap-1.5 rounded-md border border-subtle bg-elevated px-2.5 text-sm transition-colors hover:border-secondary"
      :class="block ? 'w-full' : ''">
      <UIcon name="i-heroicons-calendar" class="text-tertiary" />
      <span :class="display ? 'text-white' : 'text-tertiary'">{{ display || placeholder }}</span>
    </button>
    <template #panel="{ close }">
      <div class="w-72 rounded-lg border border-subtle bg-surface p-3 shadow-xl">
        <div class="mb-2 flex items-center">
          <span class="font-medium text-white">{{ monthLabel }}</span>
          <div class="ml-auto flex items-center gap-1">
            <button type="button" class="rounded px-1.5 py-0.5 text-xs text-tertiary hover:bg-hover hover:text-secondary" @click="goToday">[ today ]</button>
            <button type="button" class="rounded p-0.5 text-tertiary hover:bg-hover hover:text-secondary" @click="shift(-1)"><UIcon name="i-heroicons-chevron-up" /></button>
            <button type="button" class="rounded p-0.5 text-tertiary hover:bg-hover hover:text-secondary" @click="shift(1)"><UIcon name="i-heroicons-chevron-down" /></button>
          </div>
        </div>
        <div class="grid grid-cols-7 text-center text-xs text-tertiary">
          <div v-for="w in WD" :key="w" class="py-1">{{ w }}</div>
        </div>
        <div class="grid grid-cols-7 gap-0.5 text-center text-sm">
          <button v-for="c in cells" :key="c.ds" type="button"
            class="flex h-9 items-center justify-center rounded-md transition-colors"
            :class="c.sel ? 'bg-accent font-semibold text-base' : c.isToday ? 'text-bright ring-1 ring-inset ring-subtle hover:bg-hover' : c.inMonth ? 'text-secondary hover:bg-hover' : 'text-tertiary/40 hover:bg-hover'"
            @click="pick(c.ds, close)">{{ c.day }}</button>
        </div>
      </div>
    </template>
  </UPopover>
</template>
