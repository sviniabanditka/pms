import { useFocusEffect, useRouter } from 'expo-router'
import { useCallback, useState } from 'react'
import { FlatList, Pressable, RefreshControl, StyleSheet, View } from 'react-native'

import { Sidebar, type Section } from '~/components/Sidebar'
import { getCurrentUserId } from '~/lib/auth'
import { cachedIssues, myIssues, type Issue, type IssueScope } from '~/lib/api'
import { Avatar, Glass, Pill } from '~/lib/glass'
import { loadMeta, stateById, stateColor, teamById, useMeta, userById } from '~/lib/meta'
import { getWorkspaces, useActiveWs } from '~/lib/workspace'
import { C } from '~/lib/theme'
import { Prompt, Spinner, T } from '~/lib/ui'

const PRI: Record<number, { label: string; color: string }> = {
  1: { label: 'urgent', color: C.danger },
  2: { label: 'high', color: C.warn },
  3: { label: 'med', color: '#e0af68' },
  4: { label: 'low', color: C.tertiary },
}

function ago(iso: string) {
  const d = Date.now() - new Date(iso).getTime()
  const m = Math.floor(d / 60000)
  if (m < 1) return 'now'
  if (m < 60) return `${m}m`
  const h = Math.floor(m / 60)
  if (h < 24) return `${h}h`
  return `${Math.floor(h / 24)}d`
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
  const [drawer, setDrawer] = useState(false)

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
        } else setErr(e?.message ?? 'Failed to load')
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

  function onSection(s: Section) {
    if (s === 'inbox') router.push('/inbox')
    // остальные секции — в следующих фазах
  }

  return (
    <View style={styles.wrap}>
      {/* header */}
      <Glass style={styles.header} intensity={30}>
        <Pressable onPress={() => setDrawer(true)} hitSlop={10} style={styles.menu}>
          <View style={styles.bar} />
          <View style={[styles.bar, { width: 12 }]} />
          <View style={styles.bar} />
        </Pressable>
        <T w="sb" s={15} numberOfLines={1} style={{ flex: 1 }}>
          <Prompt />
          {wsName}
          {offline ? <T v="warn" s={11}>{'  '}offline</T> : null}
        </T>
        <Pressable style={styles.searchBtn}>
          <T v="faint" s={13}>{'>'}_</T>
        </Pressable>
      </Glass>

      {/* segmented scope */}
      <View style={styles.seg}>
        {(['assigned', 'created'] as IssueScope[]).map((s) => (
          <Pressable key={s} onPress={() => setScope(s)} style={[styles.segItem, scope === s && styles.segOn]}>
            <T v={scope === s ? 'bright' : 'faint'} w={scope === s ? 'sb' : 'r'} s={13}>
              {s}
            </T>
          </Pressable>
        ))}
        <View style={{ flex: 1 }} />
        <T v="faint" s={12}>
          {items.length}
        </T>
      </View>

      {loading ? (
        <Spinner />
      ) : err ? (
        <View style={styles.center}>
          <T v="danger">{err}</T>
          <Pressable onPress={() => load()}>
            <T v="accent" w="m">[ retry ]</T>
          </Pressable>
        </View>
      ) : (
        <FlatList
          data={items}
          keyExtractor={(i) => i.id}
          contentContainerStyle={items.length === 0 ? styles.center : { padding: 14, gap: 10, paddingBottom: 96 }}
          showsVerticalScrollIndicator={false}
          refreshControl={<RefreshControl refreshing={refreshing} onRefresh={() => load(true)} tintColor={C.accent} />}
          ListEmptyComponent={<T v="faint">// no {scope} issues</T>}
          renderItem={({ item }) => {
            const p = PRI[item.priority]
            const st = stateById(item.stateId)
            const asg = userById(item.assigneeId)
            const tk = teamById(item.teamId)?.key
            return (
              <Pressable onPress={() => router.push(`/issue/${item.id}`)}>
                <Glass style={styles.card}>
                  {p ? <View style={[styles.edge, { backgroundColor: p.color, shadowColor: p.color }]} /> : null}
                  <View style={styles.cardIn}>
                    <View style={styles.cardTop}>
                      <T v="accent" s={11.5} w="m">
                        {item.identifier}
                      </T>
                      <View style={{ flex: 1 }} />
                      {st ? <Pill color={stateColor(st.colorToken)} label={st.name} /> : null}
                    </View>
                    <T v="bright" s={15} w="m" numberOfLines={2} style={{ lineHeight: 20 }}>
                      {item.title}
                    </T>
                    <View style={styles.cardBot}>
                      {tk ? (
                        <T v="faint" s={11.5}>
                          {tk}
                        </T>
                      ) : null}
                      {p ? (
                        <T s={11.5} style={{ color: p.color }}>
                          · {p.label}
                        </T>
                      ) : null}
                      <T v="faint" s={11.5}>
                        · {ago(item.updatedAt)}
                      </T>
                      <View style={{ flex: 1 }} />
                      {asg ? <Avatar name={asg.name} id={asg.id} size={24} /> : null}
                    </View>
                  </View>
                </Glass>
              </Pressable>
            )
          }}
        />
      )}

      {/* FAB */}
      <Pressable style={styles.fab} onPress={() => router.push('/create')}>
        <Glass style={styles.fabGlass} intensity={40}>
          <T style={{ color: C.accent, fontSize: 26, marginTop: -2 }}>+</T>
        </Glass>
      </Pressable>

      <Sidebar visible={drawer} onClose={() => setDrawer(false)} active="issues" onSelect={onSection} />
    </View>
  )
}

const styles = StyleSheet.create({
  wrap: { flex: 1, backgroundColor: C.base },
  header: { flexDirection: 'row', alignItems: 'center', gap: 12, marginTop: 52, marginHorizontal: 12, paddingHorizontal: 14, paddingVertical: 12, borderRadius: 14 },
  menu: { gap: 3, paddingVertical: 2 },
  bar: { width: 18, height: 2, borderRadius: 2, backgroundColor: C.secondary },
  searchBtn: { paddingHorizontal: 6 },
  seg: { flexDirection: 'row', alignItems: 'center', gap: 8, paddingHorizontal: 18, paddingVertical: 12 },
  segItem: { paddingVertical: 5, paddingHorizontal: 12, borderRadius: 8 },
  segOn: { backgroundColor: 'rgba(122,162,247,0.1)', borderWidth: 1, borderColor: 'rgba(122,162,247,0.25)' },
  card: { borderRadius: 14 },
  edge: { position: 'absolute', left: 0, top: 10, bottom: 10, width: 3, borderRadius: 2, shadowOpacity: 0.8, shadowRadius: 4, shadowOffset: { width: 0, height: 0 } },
  cardIn: { padding: 13, paddingLeft: 16, gap: 8 },
  cardTop: { flexDirection: 'row', alignItems: 'center', gap: 8 },
  cardBot: { flexDirection: 'row', alignItems: 'center', gap: 5 },
  fab: { position: 'absolute', right: 20, bottom: 28 },
  fabGlass: { width: 56, height: 56, borderRadius: 28, alignItems: 'center', justifyContent: 'center', borderColor: 'rgba(122,162,247,0.4)' },
  center: { flexGrow: 1, alignItems: 'center', justifyContent: 'center', gap: 14, paddingTop: 60 },
})
