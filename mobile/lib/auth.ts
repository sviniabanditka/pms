// Auth: session-токен из POST /auth/login (бэк уже отдаёт token в body), хранится
// в storage (SecureStore/localStorage), шлётся как Bearer (см. connect.ts). Кэш в
// памяти, чтобы интерсептор не дёргал storage на каждый RPC. Плюс кэш userId.
import { API_BASE } from './config'
import { storage } from './storage'

const KEY = 'pms_token'
let cached: string | null = null
let currentUserId: string | null = null

export function getCachedToken() {
  return cached
}
export function getCurrentUserId() {
  return currentUserId
}

export async function loadToken() {
  cached = await storage.get(KEY)
  return cached
}

async function setToken(tok: string) {
  cached = tok
  await storage.set(KEY, tok)
}

async function clearToken() {
  cached = null
  currentUserId = null
  await storage.del(KEY)
}

export type User = { id: string; email: string; name: string }

export async function login(email: string, password: string): Promise<User> {
  const res = await fetch(`${API_BASE}/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, password }),
  })
  if (!res.ok) {
    throw new Error(res.status === 401 ? 'Invalid email or password' : `Sign-in failed (${res.status})`)
  }
  const body = (await res.json()) as User & { token: string }
  await setToken(body.token)
  currentUserId = body.id
  return { id: body.id, email: body.email, name: body.name }
}

// /auth/me отдаёт { user_id } — нормализуем и кэшируем id (нужен для фильтров «мои»).
export async function me(): Promise<{ id: string } | null> {
  if (!cached) return null
  const res = await fetch(`${API_BASE}/auth/me`, { headers: { Authorization: `Bearer ${cached}` } })
  if (!res.ok) {
    if (res.status === 401) await clearToken()
    return null
  }
  const body = (await res.json()) as { user_id: string }
  currentUserId = body.user_id
  return { id: body.user_id }
}

export type Workspace = { id: string; name: string; slug?: string }

export async function myWorkspaces(): Promise<Workspace[]> {
  if (!cached) return []
  const res = await fetch(`${API_BASE}/me/workspaces`, { headers: { Authorization: `Bearer ${cached}` } })
  if (!res.ok) return []
  return ((await res.json()) as Workspace[]) ?? []
}

export async function logout() {
  if (cached) {
    await fetch(`${API_BASE}/auth/logout`, { method: 'POST', headers: { Authorization: `Bearer ${cached}` } }).catch(() => {})
  }
  await clearToken()
}
