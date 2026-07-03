import { isAuthed } from '~/lib/token'

// Глобальный auth-гейт. Закрывает SSR-дыру: раньше проверка была только клиентская
// (onMounted), SSR отдавал пустой shell неавторизованному. Теперь:
//  - SSR: наличие httpOnly-cookie сессии (pms_session; JS её не читает, но сервер видит в запросе);
//  - client: локальный uid (isAuthed) — cookie httpOnly, из JS недоступна.
// Публичные роуты (лендинг/логин/инвайт) пропускаем.
const PUBLIC = ['/', '/login']

function isPublic(path: string): boolean {
  return PUBLIC.includes(path) || path.startsWith('/invite/')
}

export default defineNuxtRouteMiddleware((to) => {
  if (isPublic(to.path)) return
  const authed = import.meta.server ? !!useCookie('pms_session').value : isAuthed()
  if (!authed) return navigateTo('/login')
})
