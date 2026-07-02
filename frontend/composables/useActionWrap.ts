import { ref } from 'vue'
import { useAppStore } from '~/stores/app'

// Общая ошибка последнего экшена (инлайн-места) — модульный синглтон,
// один err на всё приложение (как и было, когда всё жило в [...slug].vue).
const err = ref('')

// useActionWrap — единая обёртка экшенов: ошибка в err + toast; опционально
// catchUp указанных типов, чтобы откатить optimistic-стейт до серверной правды.
export function useActionWrap() {
  const toast = useToast()
  const store = useAppStore()
  async function wrap(fn: () => Promise<any>, types?: string[]) {
    err.value = ''
    try { await fn() } catch (e: any) {
      err.value = e?.message || String(e)
      toast.add({ title: 'action failed', description: err.value, color: 'red' })
      if (types?.length && store.activeWorkspaceId) store.catchUp(store.activeWorkspaceId, types).catch(() => {})
    }
  }
  return { err, wrap }
}
