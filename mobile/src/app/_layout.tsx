import { Stack, useRouter, useSegments, type ErrorBoundaryProps } from 'expo-router'
import { useEffect, useState } from 'react'
import { ActivityIndicator, Pressable, StyleSheet, Text, View } from 'react-native'

import { loadToken, me, myWorkspaces } from '~/lib/auth'
import { setAuthed, useAuthed } from '~/lib/authState'
import { setOnUnauthorized } from '~/lib/connect'
import { getActiveWs, loadActiveWs } from '~/lib/workspace'
import { loadMeta } from '~/lib/meta'
import { onPushTap, registerForPush } from '~/lib/push'

// перехват ошибок рендера (expo-router) — брендовый экран вместо красного краша
export function ErrorBoundary({ error, retry }: ErrorBoundaryProps) {
  return (
    <View style={eb.wrap}>
      <Text style={eb.title}>// something broke</Text>
      <Text style={eb.msg} numberOfLines={4}>
        {error.message}
      </Text>
      <Pressable style={eb.btn} onPress={retry}>
        <Text style={eb.btnText}>[ retry ]</Text>
      </Pressable>
    </View>
  )
}

const eb = StyleSheet.create({
  wrap: { flex: 1, backgroundColor: '#0b0c12', alignItems: 'center', justifyContent: 'center', padding: 28, gap: 14 },
  title: { color: '#f7768e', fontSize: 18, fontWeight: '700' },
  msg: { color: '#565f89', fontSize: 13, textAlign: 'center' },
  btn: { borderWidth: 1, borderColor: '#1f2335', borderRadius: 8, paddingVertical: 10, paddingHorizontal: 20 },
  btnText: { color: '#7aa2f7', fontWeight: '700' },
})

export default function RootLayout() {
  const [booting, setBooting] = useState(true)
  const authed = useAuthed()
  const router = useRouter()
  const segments = useSegments()

  // boot: токен из storage → /me → воркспейсы + активный
  useEffect(() => {
    setOnUnauthorized(() => setAuthed(false))
    ;(async () => {
      await loadToken()
      const u = await me()
      if (u) {
        await loadActiveWs(await myWorkspaces())
        const ws = getActiveWs()
        if (ws) await loadMeta(ws)
        registerForPush() // best-effort, реальное устройство
      }
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

  // тап по пушу → deep-link на issue
  useEffect(() => onPushTap((path) => router.push(path as any)), [])

  if (booting) {
    return (
      <View style={{ flex: 1, alignItems: 'center', justifyContent: 'center', backgroundColor: '#0b0c12' }}>
        <ActivityIndicator color="#7aa2f7" />
      </View>
    )
  }

  return (
    <Stack screenOptions={{ headerShown: false, contentStyle: { backgroundColor: '#0b0c12' } }}>
      <Stack.Screen name="(tabs)" />
      <Stack.Screen name="login" />
      <Stack.Screen name="issue/[id]" />
      <Stack.Screen name="create" options={{ presentation: 'modal' }} />
    </Stack>
  )
}
