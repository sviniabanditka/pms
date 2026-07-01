import {
  JetBrainsMono_400Regular,
  JetBrainsMono_500Medium,
  JetBrainsMono_600SemiBold,
  JetBrainsMono_700Bold,
  useFonts,
} from '@expo-google-fonts/jetbrains-mono'
import { Stack, useRouter, useSegments, type ErrorBoundaryProps } from 'expo-router'
import { useEffect, useState } from 'react'
import { ActivityIndicator, Pressable, StyleSheet, Text, View } from 'react-native'

import { C } from '~/lib/theme'
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
  wrap: { flex: 1, backgroundColor: C.base, alignItems: 'center', justifyContent: 'center', padding: 28, gap: 14 },
  title: { color: C.danger, fontSize: 18, fontFamily: 'JetBrainsMono_700Bold' },
  msg: { color: C.tertiary, fontSize: 13, textAlign: 'center', fontFamily: 'JetBrainsMono_400Regular' },
  btn: { borderWidth: 1, borderColor: C.subtle, borderRadius: 8, paddingVertical: 10, paddingHorizontal: 20 },
  btnText: { color: C.accent, fontFamily: 'JetBrainsMono_500Medium' },
})

export default function RootLayout() {
  const [booting, setBooting] = useState(true)
  const [fontsLoaded] = useFonts({ JetBrainsMono_400Regular, JetBrainsMono_500Medium, JetBrainsMono_600SemiBold, JetBrainsMono_700Bold })
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

  if (booting || !fontsLoaded) {
    return (
      <View style={{ flex: 1, alignItems: 'center', justifyContent: 'center', backgroundColor: C.base }}>
        <ActivityIndicator color={C.accent} />
      </View>
    )
  }

  return (
    <Stack screenOptions={{ headerShown: false, contentStyle: { backgroundColor: C.base } }}>
      <Stack.Screen name="(tabs)" />
      <Stack.Screen name="login" />
      <Stack.Screen name="issue/[id]" />
      <Stack.Screen name="team/[id]" />
      <Stack.Screen name="create" options={{ presentation: 'modal' }} />
    </Stack>
  )
}
