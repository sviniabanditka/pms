<script setup lang="ts">
// Строка свойства issue-панели: иконка/глиф + лейбл (w-24) + контрол (слот).
// Layout-молекула — де-дублицирует ~7 одинаковых обёрток в IssueDetailPanel.
defineProps<{ label: string }>()
</script>

<template>
  <div class="flex min-h-8 items-center gap-3">
    <span class="flex w-24 shrink-0 items-center gap-2 text-tertiary"><slot name="icon" />{{ label }}</span>
    <slot />
  </div>
</template>
