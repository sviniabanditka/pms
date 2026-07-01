// Платформенный key-value: SecureStore на нативе, localStorage на web
// (expo-secure-store не поддерживает web).
import { Platform } from 'react-native'
import * as SecureStore from 'expo-secure-store'

const isWeb = Platform.OS === 'web'

export const storage = {
  get: (k: string): Promise<string | null> =>
    isWeb ? Promise.resolve(globalThis.localStorage?.getItem(k) ?? null) : SecureStore.getItemAsync(k),
  set: (k: string, v: string): Promise<void> =>
    isWeb ? Promise.resolve(globalThis.localStorage?.setItem(k, v)) : SecureStore.setItemAsync(k, v),
  del: (k: string): Promise<void> =>
    isWeb ? Promise.resolve(globalThis.localStorage?.removeItem(k)) : SecureStore.deleteItemAsync(k),
}
