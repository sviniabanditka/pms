// Аутентификация — httpOnly-cookie сессии (pms_session), её ставит/чистит бэкенд.
// JS токен не хранит (D3). В localStorage держим только uid — для мгновенного currentUserId
// и client-side guard'а; реальная авторизация всегда по cookie (credentials:'include').
const UKEY = 'pms_uid'

export function getUser(): string {
  return import.meta.client ? localStorage.getItem(UKEY) || '' : ''
}
export function setUser(id: string) {
  if (import.meta.client) localStorage.setItem(UKEY, id)
}
export function isAuthed(): boolean {
  return !!getUser()
}
export function clearAuth() {
  if (import.meta.client) localStorage.removeItem(UKEY)
}

// Сессия протухла (401) — чистим локальный uid и уводим на /login (полный переход, сбрасывает стор).
export function handleUnauthorized() {
  if (!import.meta.client) return
  clearAuth()
  if (location.pathname !== '/login') location.href = '/login'
}
