// SQLite KV-кэш: warm-start (мгновенная отрисовка при перезапуске) + offline-read
// (показать последний снапшот когда сети нет). Substrate под будущий local-first —
// сейчас просто key→json. Все операции best-effort (try/catch → no-op).
import * as SQLite from 'expo-sqlite'

let dbP: Promise<SQLite.SQLiteDatabase> | null = null
function db() {
  if (!dbP) {
    dbP = SQLite.openDatabaseAsync('stdtask.db').then(async (d) => {
      await d.execAsync('CREATE TABLE IF NOT EXISTS kv (k TEXT PRIMARY KEY, v TEXT)')
      return d
    })
  }
  return dbP
}

export async function cacheGet<T>(key: string): Promise<T | null> {
  try {
    const d = await db()
    const row = await d.getFirstAsync<{ v: string }>('SELECT v FROM kv WHERE k = ?', key)
    return row ? (JSON.parse(row.v) as T) : null
  } catch {
    return null
  }
}

export async function cacheSet(key: string, val: unknown): Promise<void> {
  try {
    const d = await db()
    await d.runAsync('INSERT OR REPLACE INTO kv (k, v) VALUES (?, ?)', key, JSON.stringify(val))
  } catch {
    /* no-op (напр. web без wasm) */
  }
}
