// Web/default вариант кэша (localStorage). Нативный — cache.native.ts (expo-sqlite);
// Metro сам выбирает по платформе. Разделено, потому что expo-sqlite на web тянет
// wasm-воркер и ломает бандл. API идентичен native. Best-effort (try/catch → no-op).

export async function cacheGet<T>(key: string): Promise<T | null> {
  try {
    const v = globalThis.localStorage?.getItem(key)
    return v ? (JSON.parse(v) as T) : null
  } catch {
    return null
  }
}

export async function cacheSet(key: string, val: unknown): Promise<void> {
  try {
    globalThis.localStorage?.setItem(key, JSON.stringify(val))
  } catch {
    /* no-op */
  }
}
