<script setup lang="ts">
import { useLightbox } from '~/composables/useLightbox'
const { src, alt, close } = useLightbox()
</script>

<template>
  <Teleport to="body">
    <Transition name="lb">
      <div v-if="src" class="fixed inset-0 z-[100] flex items-center justify-center bg-black/80 p-8" @click="close">
        <!-- картинка натурального размера, не больше экрана-с-отступом; маленькая → маленькая -->
        <img :src="src" :alt="alt" class="max-h-full max-w-full rounded-lg object-contain shadow-2xl" @click.stop />
        <button class="absolute right-4 top-4 flex h-9 w-9 items-center justify-center rounded-full bg-surface/80 text-tertiary hover:text-white" @click="close">
          <UIcon name="i-heroicons-x-mark" class="text-xl" />
        </button>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.lb-enter-active, .lb-leave-active { transition: opacity 0.15s ease; }
.lb-enter-from, .lb-leave-to { opacity: 0; }
</style>
