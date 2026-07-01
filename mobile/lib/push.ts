// Push: запрос прав, получение Expo-токена, регистрация на бэке, тап→deep-link.
// Только реальное устройство (не web/симулятор). Best-effort: любые сбои — no-op.
import Constants from 'expo-constants'
import * as Device from 'expo-device'
import * as Notifications from 'expo-notifications'
import { Platform } from 'react-native'

import { registerDevice, unregisterDevice } from './api'

let currentToken: string | null = null

// показывать баннер даже когда приложение открыто
Notifications.setNotificationHandler({
  handleNotification: async () => ({
    shouldShowBanner: true,
    shouldShowList: true,
    shouldPlaySound: false,
    shouldSetBadge: false,
  }),
})

export async function registerForPush(): Promise<void> {
  if (Platform.OS === 'web' || !Device.isDevice) return // пуш только на реальном устройстве
  try {
    const existing = await Notifications.getPermissionsAsync()
    let status = existing.status
    if (status !== 'granted') status = (await Notifications.requestPermissionsAsync()).status
    if (status !== 'granted') return

    const projectId = Constants.expoConfig?.extra?.eas?.projectId ?? Constants.easConfig?.projectId
    const tok = (await Notifications.getExpoPushTokenAsync(projectId ? { projectId } : undefined)).data
    currentToken = tok
    await registerDevice(tok, Platform.OS)
  } catch {
    /* нет EAS projectId / нет прав / симулятор — молча пропускаем */
  }
}

export async function unregisterForPush(): Promise<void> {
  if (!currentToken) return
  try {
    await unregisterDevice(currentToken)
  } catch {
    /* ignore */
  }
  currentToken = null
}

// подписка на тап по пушу → deep-link на issue. Возвращает cleanup.
export function onPushTap(navigate: (path: string) => void): () => void {
  const sub = Notifications.addNotificationResponseReceivedListener((resp) => {
    const data = resp.notification.request.content.data as { entityType?: string; entityId?: string }
    if (data?.entityType === 'issue' && data.entityId) navigate(`/issue/${data.entityId}`)
  })
  return () => sub.remove()
}
