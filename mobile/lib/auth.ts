// Auth: session-токен из POST /login (бэк уже отдаёт token в body), хранится в
// expo-secure-store, шлётся как Bearer (см. connect.ts). Кэш в памяти, чтобы
// интерсептор не дёргал secure-store на каждый RPC.
import { Platform } from 'react-native'
import * as SecureStore from 'expo-secure-store'
import { API_BASE } from './config'

// expo-secure-store не поддерживает web → на web падаем в localStorage (dev-удобство).
const isWeb = Platform.OS === 'web'
const store = {
  get: (k: string) => (isWeb ? Promise.resolve(globalThis.localStorage?.getItem(k) ?? null) : SecureStore.getItemAsync(k)),
  set: (k: string, v: string) => (isWeb ? Promise.resolve(globalThis.localStorage?.setItem(k, v)) : SecureStore.setItemAsync(k, v)),
  del: (k: string) => (isWeb ? Promise.resolve(globalThis.localStorage?.removeItem(k)) : SecureStore.deleteItemAsync(k)),
}

const KEY = 'pms_token'
let cached: string | null = null

export function getCachedToken() {
  return cached
}

export async function loadToken() {
  cached = await store.get(KEY)
  return cached
}

async function setToken(tok: string) {
  cached = tok
  await store.set(KEY, tok)
}

async function clearToken() {
  cached = null
  await store.del(KEY)
}

export type User = { id: string; email: string; name: string }

export async function login(email: string, password: string): Promise<User> {
  const res = await fetch(`${API_BASE}/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, password }),
  })
  if (!res.ok) {
    throw new Error(res.status === 401 ? 'Неверный email или пароль' : `Ошибка входа (${res.status})`)
  }
  const body = (await res.json()) as User & { token: string }
  await setToken(body.token)
  return { id: body.id, email: body.email, name: body.name }
}

export async function me(): Promise<User | null> {
  if (!cached) return null
  const res = await fetch(`${API_BASE}/auth/me`, { headers: { Authorization: `Bearer ${cached}` } })
  if (!res.ok) {
    if (res.status === 401) await clearToken()
    return null
  }
  return (await res.json()) as User
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
