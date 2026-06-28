<script setup lang="ts">
// Дропдаун автодополнения @/# — рендерится в relative-обёртке над textarea.
const props = defineProps<{ m: any }>()
</script>

<template>
  <div v-if="props.m.open.value" class="absolute bottom-full left-2 z-30 mb-1 max-h-64 w-72 overflow-y-auto rounded-lg border border-strong bg-surface py-1 shadow-2xl">
    <button v-for="(it, i) in props.m.items.value" :key="i"
      class="flex w-full items-center gap-2 px-2.5 py-1.5 text-left text-sm"
      :class="i === props.m.index.value ? 'bg-active text-white' : 'text-secondary hover:bg-hover'"
      @mousedown.prevent="props.m.apply(it)" @mouseenter="props.m.index.value = i">
      <Avatar v-if="props.m.kind.value === '@'" :name="it.label" :avatar-key="it.avatarKey" :size="20" />
      <UIcon v-else :name="it.icon" class="shrink-0 text-tertiary" />
      <span class="truncate font-medium">{{ it.label }}</span>
      <span v-if="it.sub" class="truncate text-xs text-tertiary">{{ it.sub }}</span>
    </button>
    <div v-if="!props.m.items.value.length" class="px-2.5 py-1.5 text-xs text-tertiary">No matches</div>
  </div>
</template>
