import { useFocusEffect, useRouter } from 'expo-router'
import { useCallback, useState } from 'react'
import { FlatList, Pressable, RefreshControl, StyleSheet, View } from 'react-native'
import Animated, { FadeInDown } from 'react-native-reanimated'
import { useSafeAreaInsets } from 'react-native-safe-area-context'

import { Sidebar, type Section } from '~/components/Sidebar'
import { cachedNotifications, issueById, listNotifications, markAllRead, markRead, type Notification } from '~/lib/api'
import { Avatar, Backdrop, Glass, GlowDot } from '~/lib/glass'
import { userById } from '~/lib/meta'
import { useActiveWs } from '~/lib/workspace'
import { C } from '~/lib/theme'
import { Prompt, Spinner, T } from '~/lib/ui'

const LABEL: Record<string, string> = {
  due_soon: 'due soon',
  assigned: 'assigned to you',
  mentioned: 'mentioned you',
  commented: 'new comment',
  state_changed: 'status changed',
}

export default function Inbox() {
  const router = useRouter()
  const insets = useSafeAreaInsets()
  const wsId = useActiveWs()
  const [items, setItems] = useState<Notification[]>([])
  const [loading, setLoading] = useState(true)
  const [refreshing, setRefreshing] = useState(false)
  const [err, setErr] = useState('')
  const [drawer, setDrawer] = useState(false)

  const load = useCallback(
    async (refresh = false) => {
      if (!wsId) return
      refresh ? setRefreshing(true) : setLoading(true)
      setErr('')
      try {
        setItems(await listNotifications(wsId))
      } catch (e: any) {
        const cached = await cachedNotifications(wsId)
        if (cached.length) setItems(cached)
        else setErr(e?.message ?? 'Failed to load')
      } finally {
        setLoading(false)
        setRefreshing(false)
      }
    },
    [wsId],
  )

  useFocusEffect(
    useCallback(() => {
      load()
    }, [load]),
  )

  async function open(n: Notification) {
    if (!wsId) return
    if (!n.readAt) {
      setItems((xs) => xs.map((x) => (x.id === n.id ? { ...x, readAt: new Date().toISOString() } : x)))
      markRead(wsId, n.id).catch(() => {})
    }
    if (n.entityType === 'issue') {
      const iss = await issueById(wsId, n.entityId).catch(() => undefined)
      if (iss) router.push(`/issue/${n.entityId}`)
    }
  }

  async function allRead() {
    if (!wsId) return
    setItems((xs) => xs.map((x) => ({ ...x, readAt: x.readAt ?? new Date().toISOString() })))
    markAllRead(wsId).catch(() => {})
  }

  function onSection(s: Section) {
    if (s === 'issues') router.push('/')
    else if (s === 'settings') router.push('/profile')
  }

  const unread = items.filter((n) => !n.readAt).length

  return (
    <Backdrop>
      <Glass style={[styles.header, { marginTop: insets.top + 15 }]} intensity={34}>
        <Pressable onPress={() => setDrawer(true)} hitSlop={10} style={styles.menu}>
          <View style={styles.barIcon} />
          <View style={[styles.barIcon, { width: 11 }]} />
          <View style={styles.barIcon} />
        </Pressable>
        <View style={{ flex: 1 }}>
          <T w="sb" s={15}>
            <Prompt char="#" color={C.accent} /> inbox
          </T>
          <T v="faint" s={11}>
            {unread ? `${unread} unread` : 'all caught up'}
          </T>
        </View>
        {unread ? (
          <Pressable onPress={allRead} hitSlop={8}>
            <T v="faint" s={12}>
              mark all
            </T>
          </Pressable>
        ) : null}
      </Glass>

      {loading ? (
        <Spinner />
      ) : err ? (
        <View style={styles.center}>
          <T v="danger">{err}</T>
          <Pressable onPress={() => load()}>
            <T v="accent" w="m">
              [ retry ]
            </T>
          </Pressable>
        </View>
      ) : (
        <Glass style={styles.list} intensity={18}>
          <FlatList
            data={items}
            keyExtractor={(n) => n.id}
            contentContainerStyle={items.length === 0 ? styles.center : { paddingVertical: 4 }}
            showsVerticalScrollIndicator={false}
            refreshControl={<RefreshControl refreshing={refreshing} onRefresh={() => load(true)} tintColor={C.accent} />}
            ItemSeparatorComponent={() => <View style={styles.sep} />}
            ListEmptyComponent={<T v="faint">// inbox zero</T>}
            renderItem={({ item, index }) => {
              const actor = userById(item.actorId)
              return (
                <Animated.View entering={FadeInDown.duration(240).delay(Math.min(index, 14) * 26)}>
                  <Pressable onPress={() => open(item)} style={({ pressed }) => [styles.row, pressed && styles.rowPressed]}>
                    {!item.readAt ? <GlowDot color={C.accent} size={7} /> : <View style={{ width: 7 }} />}
                    {actor ? <Avatar name={actor.name} id={actor.id} size={26} /> : <View style={{ width: 26 }} />}
                    <View style={{ flex: 1, gap: 2 }}>
                      <T v={item.readAt ? 'dim' : 'bright'} w={item.readAt ? 'r' : 'm'} s={14}>
                        {LABEL[item.type] ?? item.type}
                      </T>
                      <T v="faint" s={11.5}>
                        {actor ? actor.name + ' · ' : ''}
                        {item.entityType}
                      </T>
                    </View>
                  </Pressable>
                </Animated.View>
              )
            }}
          />
        </Glass>
      )}

      <Sidebar visible={drawer} onClose={() => setDrawer(false)} active="inbox" onSelect={onSection} />
    </Backdrop>
  )
}

const styles = StyleSheet.create({
  header: { flexDirection: 'row', alignItems: 'center', gap: 18, marginHorizontal: 12, paddingHorizontal: 15, paddingVertical: 11, borderRadius: 18 },
  menu: { gap: 3.5, paddingVertical: 2 },
  barIcon: { width: 17, height: 2, borderRadius: 2, backgroundColor: C.secondary },
  list: { flex: 1, marginHorizontal: 12, marginTop: 12, marginBottom: 12, borderRadius: 20 },
  row: { flexDirection: 'row', alignItems: 'center', gap: 11, paddingVertical: 11, paddingHorizontal: 14 },
  rowPressed: { backgroundColor: 'rgba(122,162,247,0.07)' },
  sep: { height: 1, backgroundColor: 'rgba(255,255,255,0.05)', marginLeft: 58 },
  center: { flexGrow: 1, alignItems: 'center', justifyContent: 'center', gap: 14, paddingTop: 60 },
})
