import { useFocusEffect, useRouter } from 'expo-router'
import { useCallback, useState } from 'react'
import { FlatList, Pressable, RefreshControl, StyleSheet, View } from 'react-native'

import { getCurrentUserId } from '~/lib/auth'
import { cachedIssues, myIssues, type Issue, type IssueScope } from '~/lib/api'
import { loadMeta, stateById, stateColor, useMeta, userById } from '~/lib/meta'
import { getWorkspaces, useActiveWs } from '~/lib/workspace'
import { C } from '~/lib/theme'
import { Bracket, Prompt, Spinner, T } from '~/lib/ui'

const PRI: Record<number, { label: string; color: string }> = {
  1: { label: 'urgent', color: C.danger },
  2: { label: 'high', color: C.warn },
  3: { label: 'med', color: '#e0af68' },
  4: { label: 'low', color: C.tertiary },
}

export default function Issues() {
  const router = useRouter()
  const wsId = useActiveWs()
  useMeta()
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

  useFocusEffect(
    useCallback(() => {
      load()
    }, [load]),
  )

  return (
    <View style={styles.wrap}>
      <View style={styles.header}>
        <View style={styles.titleRow}>
          <T w="b" s={20} numberOfLines={1} style={{ flex: 1 }}>
            <Prompt />
            {wsName}
            {offline ? <T v="warn" s={12}> · offline</T> : null}
          </T>
          <Bracket label="+ new" onPress={() => router.push('/create')} />
        </View>
        <View style={styles.toggle}>
          {(['assigned', 'created'] as IssueScope[]).map((s) => (
            <Pressable key={s} onPress={() => setScope(s)} style={[styles.tab, scope === s && styles.tabOn]}>
              <T v={scope === s ? 'bright' : 'faint'} w={scope === s ? 'm' : 'r'} s={13}>
                {s}
              </T>
            </Pressable>
          ))}
        </View>
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
          keyExtractor={(i) => i.id}
          contentContainerStyle={items.length === 0 ? styles.center : { paddingBottom: 24 }}
          refreshControl={<RefreshControl refreshing={refreshing} onRefresh={() => load(true)} tintColor={C.accent} />}
          ListEmptyComponent={<T v="faint">// no {scope} issues</T>}
          renderItem={({ item }) => {
            const p = PRI[item.priority]
            const st = stateById(item.stateId)
            const asg = userById(item.assigneeId)
            return (
              <Pressable style={styles.row} onPress={() => router.push(`/issue/${item.id}`)}>
                <View style={styles.rowTop}>
                  <T v="accent" s={12} style={styles.ident}>
                    {item.identifier}
                  </T>
                  <T v="bright" s={15} numberOfLines={2} style={{ flex: 1 }}>
                    {item.title}
                  </T>
                  {p ? (
                    <T w="m" s={10} style={[styles.pri, { color: p.color, borderColor: p.color }]}>
                      {p.label}
                    </T>
                  ) : null}
                </View>
                <View style={styles.rowMeta}>
                  {st ? (
                    <View style={styles.stateChip}>
                      <View style={[styles.dot, { backgroundColor: stateColor(st.colorToken) }]} />
                      <T v="faint" s={12}>
                        {st.name}
                      </T>
                    </View>
                  ) : null}
                  {asg ? <T v="faint" s={12}>· {asg.name}</T> : null}
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
  header: { marginBottom: 14, gap: 12 },
  titleRow: { flexDirection: 'row', alignItems: 'center', gap: 8 },
  toggle: { flexDirection: 'row', gap: 6 },
  tab: { paddingVertical: 5, paddingHorizontal: 12, borderRadius: 6, borderWidth: 1, borderColor: C.subtle },
  tabOn: { backgroundColor: C.active, borderColor: C.active },
  row: { paddingVertical: 12, borderBottomWidth: 1, borderBottomColor: C.subtle, gap: 6 },
  rowTop: { flexDirection: 'row', alignItems: 'flex-start', gap: 10 },
  ident: { width: 76, paddingTop: 2 },
  pri: { borderWidth: 1, borderRadius: 4, paddingHorizontal: 5, paddingVertical: 1, overflow: 'hidden' },
  rowMeta: { flexDirection: 'row', alignItems: 'center', gap: 8, paddingLeft: 86 },
  stateChip: { flexDirection: 'row', alignItems: 'center', gap: 6 },
  dot: { width: 8, height: 8, borderRadius: 4 },
  center: { flexGrow: 1, alignItems: 'center', justifyContent: 'center', gap: 14 },
})
