import { useRouter } from 'expo-router'
import { useCallback, useEffect, useState } from 'react'
import { ActivityIndicator, FlatList, Pressable, RefreshControl, StyleSheet, Text, View } from 'react-native'

import { issueById, listNotifications, markAllRead, markRead, type Notification } from '~/lib/api'
import { userById } from '~/lib/meta'
import { useActiveWs } from '~/lib/workspace'

// type → человекочитаемо
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
        setErr(e?.message ?? 'Failed to load')
      } finally {
        setLoading(false)
        setRefreshing(false)
      }
    },
    [wsId],
  )

  useEffect(() => {
    load()
  }, [load])

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
        <Text style={styles.h1}>
          <Text style={{ color: '#9ece6a' }}>#</Text> inbox {unread ? <Text style={styles.count}>{unread}</Text> : null}
        </Text>
        {unread ? (
          <Pressable onPress={allRead}>
            <Text style={styles.markAll}>[ mark all read ]</Text>
          </Pressable>
        ) : null}
      </View>

      {loading ? (
        <ActivityIndicator color="#7aa2f7" style={{ marginTop: 40 }} />
      ) : err ? (
        <View style={styles.center}>
          <Text style={styles.err}>{err}</Text>
          <Pressable onPress={() => load()} style={styles.retry}>
            <Text style={styles.retryText}>[ retry ]</Text>
          </Pressable>
        </View>
      ) : (
        <FlatList
          data={items}
          keyExtractor={(n) => n.id}
          contentContainerStyle={items.length === 0 && styles.center}
          refreshControl={<RefreshControl refreshing={refreshing} onRefresh={() => load(true)} tintColor="#7aa2f7" />}
          ListEmptyComponent={<Text style={styles.muted}>// inbox zero</Text>}
          renderItem={({ item }) => {
            const actor = userById(item.actorId)?.name
            return (
              <Pressable style={styles.row} onPress={() => open(item)}>
                {!item.readAt ? <View style={styles.unreadDot} /> : <View style={styles.dotSpace} />}
                <View style={{ flex: 1 }}>
                  <Text style={[styles.type, !item.readAt && { color: '#c0caf5' }]}>{LABEL[item.type] ?? item.type}</Text>
                  <Text style={styles.sub}>
                    {actor ? actor + ' · ' : ''}
                    {item.entityType}
                  </Text>
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
  wrap: { flex: 1, backgroundColor: '#0b0c12', paddingTop: 64, paddingHorizontal: 16 },
  header: { flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between', marginBottom: 12 },
  h1: { color: '#c0caf5', fontSize: 20, fontWeight: '700' },
  count: { color: '#7aa2f7', fontSize: 15 },
  markAll: { color: '#565f89', fontSize: 13 },
  row: { flexDirection: 'row', alignItems: 'center', gap: 10, paddingVertical: 12, borderBottomWidth: 1, borderBottomColor: '#13151f' },
  unreadDot: { width: 8, height: 8, borderRadius: 4, backgroundColor: '#7aa2f7' },
  dotSpace: { width: 8 },
  type: { color: '#565f89', fontSize: 15 },
  sub: { color: '#565f89', fontSize: 12, marginTop: 2 },
  center: { flexGrow: 1, alignItems: 'center', justifyContent: 'center', gap: 12 },
  muted: { color: '#565f89' },
  err: { color: '#f7768e' },
  retry: { borderWidth: 1, borderColor: '#1f2335', borderRadius: 6, paddingVertical: 8, paddingHorizontal: 16 },
  retryText: { color: '#7aa2f7' },
})
