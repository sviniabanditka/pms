// Общий шелл секции: Backdrop + glass-хедер (гамбургер→Sidebar) + централизованная
// навигация по секциям. Убирает бойлерплейт с каждого экрана.
import { useRouter } from 'expo-router'
import { useState, type ReactNode } from 'react'
import { Pressable, StyleSheet, View } from 'react-native'
import { useSafeAreaInsets } from 'react-native-safe-area-context'

import { Sidebar, type Section } from '~/components/Sidebar'
import { Backdrop, Glass } from '~/lib/glass'
import { C } from '~/lib/theme'
import { Prompt, T } from '~/lib/ui'

const ROUTE: Record<Section, string> = {
  issues: '/',
  inbox: '/inbox',
  cycles: '/cycles',
  docs: '/docs',
  chat: '/chat',
  meetings: '/meetings',
  calendar: '/calendar',
  dashboards: '/dashboards',
  reports: '/reports',
  settings: '/profile',
}

export function sectionRoute(router: ReturnType<typeof useRouter>, s: Section) {
  router.push(ROUTE[s] as any)
}

export function Screen({
  title,
  sub,
  titleChar = '#',
  titleColor = C.accent,
  active,
  right,
  children,
}: {
  title: string
  sub?: string
  titleChar?: string
  titleColor?: string
  active: Section
  right?: ReactNode
  children: ReactNode
}) {
  const insets = useSafeAreaInsets()
  const router = useRouter()
  const [drawer, setDrawer] = useState(false)

  return (
    <Backdrop>
      <Glass style={[styles.header, { marginTop: insets.top + 15 }]} intensity={34}>
        <Pressable onPress={() => setDrawer(true)} hitSlop={10} style={styles.menu}>
          <View style={styles.barIcon} />
          <View style={[styles.barIcon, { width: 11 }]} />
          <View style={styles.barIcon} />
        </Pressable>
        <View style={{ flex: 1 }}>
          <T w="sb" s={15} numberOfLines={1}>
            <Prompt char={titleChar} color={titleColor} /> {title}
          </T>
          {sub ? (
            <T v="faint" s={11}>
              {sub}
            </T>
          ) : null}
        </View>
        {right}
      </Glass>

      {children}

      <Sidebar visible={drawer} onClose={() => setDrawer(false)} active={active} onSelect={(s) => sectionRoute(router, s)} />
    </Backdrop>
  )
}

const styles = StyleSheet.create({
  header: { flexDirection: 'row', alignItems: 'center', gap: 18, marginHorizontal: 12, paddingHorizontal: 15, paddingVertical: 11, borderRadius: 18 },
  menu: { gap: 3.5, paddingVertical: 2 },
  barIcon: { width: 17, height: 2, borderRadius: 2, backgroundColor: C.secondary },
})
