import { useFocusEffect, useRouter } from 'expo-router'
import { useCallback, useState } from 'react'
import { ActivityIndicator, FlatList, Pressable, RefreshControl, StyleSheet, Text, View } from 'react-native'

import { getCurrentUserId } from '~/lib/auth'
import { cachedIssues, myIssues, type Issue, type IssueScope } from '~/lib/api'
import { loadMeta, stateById, stateColor, useMeta, userById } from '~/lib/meta'
import { getWorkspaces, useActiveWs } from '~/lib/workspace'

const PRI: Record<number, { label: string; color: string }> = {
  1: { label: 'urgent', color: '#f7768e' },
  2: { label: 'high', color: '#ff9e64' },
  3: { label: 'med', color: '#e0af68' },
  4: { label: 'low', color: '#565f89' },
}

export default function Issues() {
  const router = useRouter()
  const wsId = useActiveWs()
  useMeta() // перерендер когда мета догрузилась
  const wsName = getWorkspaces().find((w) => w.id === wsId)?.name ?? 'workspace'
  const [scope, setScope] = useState<IssueScope>('assigned')
  const [items, setItems] = useState<Issue[]>([])
  const [loading, setLoading] = useState(true)
  const [refreshing, setRefreshing] = useState(false)
  const [offline, setOffline] = useState(false)
  const [err, setErr] = useState('')

  const load = useCallback(
    async (refresh = false) => {
      const uid = getCurrentUserId()
      if (!wsId || !uid) return
      refresh ? setRefreshing(true) : setLoading(true)
      setErr('')
      try {
        await loadMeta(wsId)
        setItems(await myIssues(wsId, uid, scope))
        setOffline(false)
      } catch (e: any) {
        // сеть недоступна → показываем последний закэшированный снапшот
        const cached = await cachedIssues(wsId, scope)
        if (cached.length) {
          setItems(cached)
          setOffline(true)
        } else {
          setErr(e?.message ?? 'Failed to load')
        }
      } finally {
        setLoading(false)
        setRefreshing(false)
      }
    },
    [wsId, scope],
  )

  // грузим на маунт и при каждом возврате на экран (свежий список после create/detail)
  useFocusEffect(
    useCallback(() => {
      load()
    }, [load]),
  )

  return (
    <View style={styles.wrap}>
      <View style={styles.header}>
        <View style={styles.titleRow}>
          <Text style={styles.ws} numberOfLines={1}>
            <Text style={{ color: '#9ece6a' }}>~/</Text>
            {wsName}
            {offline ? <Text style={styles.offline}> · offline</Text> : null}
          </Text>
          <Pressable onPress={() => router.push('/create')} style={styles.newBtn}>
            <Text style={styles.newText}>[ + new ]</Text>
          </Pressable>
        </View>
        <View style={styles.toggle}>
          {(['assigned', 'created'] as IssueScope[]).map((s) => (
            <Pressable key={s} onPress={() => setScope(s)} style={[styles.tab, scope === s && styles.tabOn]}>
              <Text style={[styles.tabText, scope === s && styles.tabTextOn]}>{s}</Text>
            </Pressable>
          ))}
        </View>
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
          keyExtractor={(i) => i.id}
          contentContainerStyle={items.length === 0 && styles.center}
          refreshControl={<RefreshControl refreshing={refreshing} onRefresh={() => load(true)} tintColor="#7aa2f7" />}
          ListEmptyComponent={<Text style={styles.muted}>// no {scope} issues</Text>}
          renderItem={({ item }) => {
            const p = PRI[item.priority]
            const st = stateById(item.stateId)
            const asg = userById(item.assigneeId)
            return (
              <Pressable style={styles.row} onPress={() => router.push(`/issue/${item.id}`)}>
                <View style={styles.rowTop}>
                  <Text style={styles.ident}>{item.identifier}</Text>
                  <Text style={styles.title} numberOfLines={2}>
                    {item.title}
                  </Text>
                  {p ? <Text style={[styles.pri, { color: p.color, borderColor: p.color }]}>{p.label}</Text> : null}
                </View>
                <View style={styles.rowMeta}>
                  {st ? (
                    <View style={styles.stateChip}>
                      <View style={[styles.dot, { backgroundColor: stateColor(st.colorToken) }]} />
                      <Text style={styles.metaText}>{st.name}</Text>
                    </View>
                  ) : null}
                  {asg ? <Text style={styles.metaText}>· {asg.name}</Text> : null}
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
  header: { marginBottom: 12, gap: 10 },
  titleRow: { flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between' },
  ws: { color: '#c0caf5', fontSize: 20, fontWeight: '700', flex: 1 },
  offline: { color: '#e0af68', fontSize: 12, fontWeight: '400' },
  newBtn: { paddingVertical: 4, paddingHorizontal: 8 },
  newText: { color: '#7aa2f7', fontSize: 13 },
  toggle: { flexDirection: 'row', gap: 6 },
  tab: { paddingVertical: 5, paddingHorizontal: 12, borderRadius: 6, borderWidth: 1, borderColor: '#1f2335' },
  tabOn: { backgroundColor: '#1f2335' },
  tabText: { color: '#565f89', fontSize: 13 },
  tabTextOn: { color: '#c0caf5' },
  row: { paddingVertical: 11, borderBottomWidth: 1, borderBottomColor: '#13151f', gap: 5 },
  rowTop: { flexDirection: 'row', alignItems: 'center', gap: 10 },
  ident: { color: '#7aa2f7', fontSize: 12, width: 74, fontVariant: ['tabular-nums'] },
  title: { color: '#c0caf5', fontSize: 15, flex: 1 },
  pri: { fontSize: 10, borderWidth: 1, borderRadius: 4, paddingHorizontal: 5, paddingVertical: 1 },
  rowMeta: { flexDirection: 'row', alignItems: 'center', gap: 6, paddingLeft: 84 },
  stateChip: { flexDirection: 'row', alignItems: 'center', gap: 5 },
  dot: { width: 8, height: 8, borderRadius: 4 },
  metaText: { color: '#565f89', fontSize: 12 },
  center: { flexGrow: 1, alignItems: 'center', justifyContent: 'center', gap: 12 },
  muted: { color: '#565f89' },
  err: { color: '#f7768e' },
  retry: { borderWidth: 1, borderColor: '#1f2335', borderRadius: 6, paddingVertical: 8, paddingHorizontal: 16 },
  retryText: { color: '#7aa2f7' },
})
