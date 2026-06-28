import { reactive } from 'vue'
import { useAppStore } from '~/stores/app'

// Кэш presigned-ссылок по s3-ключу. url(key) возвращает '' пока грузится,
// затем reactive-кэш триггерит ре-рендер. TTL ссылки 1ч — на сессию хватает.
const cache = reactive<Record<string, string>>({})
const inflight: Record<string, boolean> = {}

export function useFileUrl() {
  const store = useAppStore()
  function url(key?: string | null): string {
    if (!key) return ''
    if (cache[key]) return cache[key]
    if (!inflight[key]) {
      inflight[key] = true
      store.fileUrl(key).then((u) => { cache[key] = u }).catch(() => { inflight[key] = false })
    }
    return ''
  }
  return { url }
}
