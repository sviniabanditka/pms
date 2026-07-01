import { useFocusEffect, useRouter } from 'expo-router'
import { useCallback, useState } from 'react'
import { FlatList, Pressable, RefreshControl, StyleSheet, View } from 'react-native'

import { cachedNotifications, issueById, listNotifications, markAllRead, markRead, type Notification } from '~/lib/api'
import { userById } from '~/lib/meta'
import { useActiveWs } from '~/lib/workspace'
import { C } from '~/lib/theme'
import { Bracket, Prompt, Spinner, T } from '~/lib/ui'

const LABEL: Record<string, string> = {
  due_soon: 'due soon',
  assigned: 'assigned to you',
  mentioned: 'mentioned you',
  commented: 'new comment',
  state_changed: 'status changed',
}

export default function Inbox() {
  const router = useRouter()
  const wsId = useActiveWs()
  const [items, setItems] = useState<Notification[]>([])
  const [loading, setLoading] = useState(true)
  const [refreshing, setRefreshing] = useState(false)
  const [err, setErr] = useState('')

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

  const unread = items.filter((n) => !n.readAt).length

  return (
    <View style={styles.wrap}>
      <View style={styles.header}>
        <T w="b" s={20}>
          <Prompt char="#" color={C.accent} /> inbox {unread ? <T v="accent" s={15}>{unread}</T> : null}
        </T>
        {unread ? <Bracket label="mark all read" color={C.tertiary} onPress={allRead} /> : null}
      </View>

      {loading ? (
        <Spinner />
      ) : err ? (
        <View style={styles.center}>
          <T v="danger">{err}</T>
          <Bracket label="retry" onPress={() => load()} />
        </View>
      ) : (
        <FlatList
          data={items}
          keyExtractor={(n) => n.id}
          contentContainerStyle={items.length === 0 ? styles.center : { paddingBottom: 24 }}
          refreshControl={<RefreshControl refreshing={refreshing} onRefresh={() => load(true)} tintColor={C.accent} />}
          ListEmptyComponent={<T v="faint">// inbox zero</T>}
          renderItem={({ item }) => {
            const actor = userById(item.actorId)?.name
            return (
              <Pressable style={styles.row} onPress={() => open(item)}>
                {!item.readAt ? <View style={styles.unreadDot} /> : <View style={styles.dotSpace} />}
                <View style={{ flex: 1 }}>
                  <T v={item.readAt ? 'faint' : 'bright'} s={15} w={item.readAt ? 'r' : 'm'}>
                    {LABEL[item.type] ?? item.type}
                  </T>
                  <T v="faint" s={12} style={{ marginTop: 2 }}>
                    {actor ? actor + ' · ' : ''}
                    {item.entityType}
                  </T>
                </View>
              </Pressable>
            )
          }}
        />
      )}
    </View>
  )
}

const styles = StyleSheet.create({
  wrap: { flex: 1, backgroundColor: C.base, paddingTop: 64, paddingHorizontal: 16 },
  header: { flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between', marginBottom: 14 },
  row: { flexDirection: 'row', alignItems: 'center', gap: 12, paddingVertical: 13, borderBottomWidth: 1, borderBottomColor: C.subtle },
  unreadDot: { width: 8, height: 8, borderRadius: 4, backgroundColor: C.accent },
  dotSpace: { width: 8 },
  center: { flexGrow: 1, alignItems: 'center', justifyContent: 'center', gap: 14 },
})
