import { Stack, useRouter, useSegments } from 'expo-router'
import { useEffect, useState } from 'react'
import { ActivityIndicator, View } from 'react-native'

import { loadToken, me } from '../../lib/auth'
import { setAuthed, useAuthed } from '../../lib/authState'
import { setOnUnauthorized } from '../../lib/connect'

export default function RootLayout() {
  const [booting, setBooting] = useState(true)
  const authed = useAuthed()
  const router = useRouter()
  const segments = useSegments()

  // boot: поднять токен из secure-store, проверить /me
  useEffect(() => {
    setOnUnauthorized(() => setAuthed(false))
    ;(async () => {
      await loadToken()
      const u = await me()
      setAuthed(!!u)
      setBooting(false)
    })()
  }, [])

  // редирект между /login и приложением
  useEffect(() => {
    if (booting) return
    const onLogin = segments[0] === 'login'
    if (!authed && !onLogin) router.replace('/login')
    else if (authed && onLogin) router.replace('/')
  }, [booting, authed, segments])

  if (booting) {
    return (
      <View style={{ flex: 1, alignItems: 'center', justifyContent: 'center', backgroundColor: '#0b0c12' }}>
        <ActivityIndicator color="#7aa2f7" />
      </View>
    )
  }

  return (
    <Stack screenOptions={{ headerShown: false, contentStyle: { backgroundColor: '#0b0c12' } }}>
      <Stack.Screen name="index" />
      <Stack.Screen name="login" />
    </Stack>
  )
}
