import { ref } from 'vue'
import { useAppStore } from '~/stores/app'

// Invite-стейт — модульный синглтон: им пользуются и страница (MembersModal),
// и вкладка Members в SettingsModal. Один стейт на всё приложение.
const inviteEmail = ref('')
const inviteRole = ref('member')
const inviteLink = ref('')

export function useInvites() {
  const store = useAppStore()
  const { wrap } = useActionWrap()
  const doInvite = () => wrap(async () => {
    if (!inviteEmail.value.trim()) return
    const token = await store.inviteMember(inviteEmail.value.trim(), inviteRole.value)
    inviteEmail.value = ''
    // новый юзер ещё не зарегистрирован → даём ссылку-приглашение для отправки вручную
    inviteLink.value = token ? `${location.origin}/invite/${token}` : ''
  })
  const copyInviteLink = () => { if (inviteLink.value) navigator.clipboard?.writeText(inviteLink.value) }
  return { inviteEmail, inviteRole, inviteLink, doInvite, copyInviteLink }
}
