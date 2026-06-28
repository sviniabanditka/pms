// Хранилище opaque-токена для Bearer-аутентификации SPA (cross-origin dev).
// ponytail: localStorage достаточно для скелета; httpOnly-cookie/refresh — позже.
const KEY = 'pms_token'
const UKEY = 'pms_uid'

export function getToken(): string {
  return import.meta.client ? localStorage.getItem(KEY) || '' : ''
}
export function setToken(t: string) {
  if (import.meta.client) localStorage.setItem(KEY, t)
}
export function getUser(): string {
  return import.meta.client ? localStorage.getItem(UKEY) || '' : ''
}
export function setUser(id: string) {
  if (import.meta.client) localStorage.setItem(UKEY, id)
}
export function clearToken() {
  if (import.meta.client) { localStorage.removeItem(KEY); localStorage.removeItem(UKEY) }
}
