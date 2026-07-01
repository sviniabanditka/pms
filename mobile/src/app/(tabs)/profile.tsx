import { useRouter } from 'expo-router'
import { useState } from 'react'
import { Pressable, ScrollView, StyleSheet, View } from 'react-native'
import Animated, { FadeInDown } from 'react-native-reanimated'
import { useSafeAreaInsets } from 'react-native-safe-area-context'

import { Sidebar, type Section } from '~/components/Sidebar'
import { getCurrentUserId, logout } from '~/lib/auth'
import { setAuthed } from '~/lib/authState'
import { unregisterForPush } from '~/lib/push'
import { Avatar, Backdrop, Glass } from '~/lib/glass'
import { getWorkspaces, setActiveWs, useActiveWs } from '~/lib/workspace'
import { C } from '~/lib/theme'
import { Bracket, Prompt, T } from '~/lib/ui'

export default function Profile() {
  const router = useRouter()
  const insets = useSafeAreaInsets()
  const activeWs = useActiveWs()
  const workspaces = getWorkspaces()
  const uid = getCurrentUserId()
  const [drawer, setDrawer] = useState(false)

  async function onLogout() {
    await unregisterForPush()
    await logout()
    setAuthed(false)
  }

  function onSection(s: Section) {
    if (s === 'issues') router.push('/')
    else if (s === 'inbox') router.push('/inbox')
  }

  return (
    <Backdrop>
      <Glass style={[styles.header, { marginTop: insets.top + 15 }]} intensity={34}>
        <Pressable onPress={() => setDrawer(true)} hitSlop={10} style={styles.menu}>
          <View style={styles.barIcon} />
          <View style={[styles.barIcon, { width: 11 }]} />
          <View style={styles.barIcon} />
        </Pressable>
        <T w="sb" s={15} style={{ flex: 1 }}>
          <Prompt char="#" color={C.accent} /> profile
        </T>
      </Glass>

      <ScrollView contentContainerStyle={{ padding: 14, gap: 12 }} showsVerticalScrollIndicator={false}>
        <Animated.View entering={FadeInDown.duration(300)}>
          <Glass style={styles.userCard} intensity={24}>
            <Avatar name={undefined} id={uid ?? 'me'} size={44} />
            <View style={{ flex: 1 }}>
              <T v="bright" w="sb" s={15}>
                account
              </T>
              <T v="faint" s={11} numberOfLines={1}>
                {uid ?? '—'}
              </T>
            </View>
          </Glass>
        </Animated.View>

        <T v="accent" s={13} style={{ paddingHorizontal: 4, marginTop: 6 }}>
          # workspaces
        </T>
        {workspaces.map((w, i) => {
          const on = w.id === activeWs
          return (
            <Animated.View key={w.id} entering={FadeInDown.duration(260).delay(i * 40)}>
              <Pressable onPress={() => setActiveWs(w.id)} style={({ pressed }) => ({ transform: [{ scale: pressed ? 0.99 : 1 }] })}>
                <Glass style={[styles.wsCard, on && styles.wsOn]} intensity={20}>
                  <T v={on ? 'accent' : 'body'} w="sb" s={15} style={{ flex: 1 }}>
                    <Prompt />
                    {w.name}
                  </T>
                  {on ? <T v="accent" s={11}>active</T> : null}
                </Glass>
              </Pressable>
            </Animated.View>
          )
        })}

        <View style={{ paddingVertical: 20, paddingHorizontal: 4 }}>
          <Bracket label="sign out" color={C.danger} onPress={onLogout} />
        </View>
      </ScrollView>

      <Sidebar visible={drawer} onClose={() => setDrawer(false)} active="settings" onSelect={onSection} />
    </Backdrop>
  )
}

const styles = StyleSheet.create({
  header: { flexDirection: 'row', alignItems: 'center', gap: 18, marginHorizontal: 12, paddingHorizontal: 15, paddingVertical: 11, borderRadius: 18 },
  menu: { gap: 3.5, paddingVertical: 2 },
  barIcon: { width: 17, height: 2, borderRadius: 2, backgroundColor: C.secondary },
  userCard: { flexDirection: 'row', alignItems: 'center', gap: 13, borderRadius: 16, padding: 15 },
  wsCard: { flexDirection: 'row', alignItems: 'center', borderRadius: 14, padding: 15 },
  wsOn: { borderColor: 'rgba(122,162,247,0.4)' },
})
