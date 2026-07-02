<script setup lang="ts">
import { useAppStore } from '~/stores/app'
import { labelOpts, modalUi, ROLES } from '~/lib/ui'

// invite-стейт живёт на странице (им же пользуется вкладка members в Settings)
defineProps<{ inviteLink: string }>()
const emit = defineEmits<{ (e: 'invite'): void; (e: 'copy-link'): void }>()
const open = defineModel<boolean>()
const inviteEmail = defineModel<string>('inviteEmail', { default: '' })
const inviteRole = defineModel<string>('inviteRole', { default: 'member' })

const store = useAppStore()
</script>

<template>
  <UModal v-model="open" :ui="modalUi">
    <div class="space-y-3 p-4">
      <div class="flex items-center gap-2">
        <UIcon name="i-heroicons-user-group" class="text-tertiary" />
        <span class="text-base font-semibold text-white"><span class="text-ok">$</span> members</span>
        <UButton class="ml-auto" icon="i-heroicons-x-mark" color="gray" variant="ghost" size="xs" square @click="open = false" />
      </div>
      <!-- invite -->
      <div v-if="store.isWsAdmin" class="flex gap-2">
        <UInput v-model="inviteEmail" placeholder="email to invite" class="flex-1" @keyup.enter="emit('invite')" />
        <USelectMenu v-model="inviteRole" :options="labelOpts(ROLES)" value-attribute="value" option-attribute="label" class="w-32" />
        <UButton :disabled="!inviteEmail.trim()" @click="emit('invite')">[ invite ]</UButton>
      </div>
      <div v-if="inviteLink" class="flex items-center gap-2 rounded-md bg-elevated px-2 py-1.5 text-xs">
        <UIcon name="i-heroicons-link" class="shrink-0 text-tertiary" />
        <span class="flex-1 truncate font-mono text-tertiary">{{ inviteLink }}</span>
        <UButton size="2xs" color="gray" @click="emit('copy-link')">[ copy ]</UButton>
      </div>
      <!-- members list -->
      <div class="max-h-72 overflow-y-auto">
        <div class="mb-1 text-xs font-medium uppercase tracking-wider text-tertiary"><span class="text-ok"># </span>Members ({{ store.workspaceMembers.length }})</div>
        <div v-for="m in store.workspaceMembers" :key="m.id" class="flex items-center gap-2 rounded-md px-2 py-1.5 hover:bg-hover">
          <span class="flex h-7 w-7 items-center justify-center rounded-full bg-elevated text-xs">{{ (m.name || '?')[0].toUpperCase() }}</span>
          <span class="flex-1 truncate text-sm">{{ m.name }}</span>
          <USelectMenu v-if="store.isWsAdmin" :model-value="m.role" :options="labelOpts(ROLES)" value-attribute="value" option-attribute="label" size="xs" class="w-28" @update:model-value="(r) => store.updateMemberRole(m.userId, r)" />
          <span v-else class="w-28 text-right text-xs text-tertiary">{{ m.role }}</span>
        </div>
        <template v-if="store.isWsAdmin && store.pendingInvites.length">
          <div class="mb-1 mt-3 text-xs font-medium uppercase tracking-wider text-tertiary"><span class="text-ok"># </span>Pending invites</div>
          <div v-for="iv in store.pendingInvites" :key="iv.id" class="group flex items-center gap-2 px-2 py-1.5 text-sm text-tertiary">
            <UIcon name="i-heroicons-envelope" /> <span class="flex-1 truncate">{{ iv.email }}</span><span class="text-xs">{{ iv.role }}</span>
            <UButton icon="i-heroicons-x-mark" color="red" variant="ghost" size="2xs" square class="opacity-0 group-hover:opacity-100" title="Revoke" @click="store.revokeInvitation(iv.id)" />
          </div>
        </template>
      </div>
    </div>
  </UModal>
</template>
