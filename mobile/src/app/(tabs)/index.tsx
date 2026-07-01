import { LinearGradient } from 'expo-linear-gradient'
import { useFocusEffect, useRouter } from 'expo-router'
import { useCallback, useState } from 'react'
import { FlatList, Pressable, RefreshControl, StyleSheet, View } from 'react-native'

import { Sidebar, type Section } from '~/components/Sidebar'
import { getCurrentUserId } from '~/lib/auth'
import { cachedIssues, myIssues, type Issue, type IssueScope } from '~/lib/api'
import { Avatar, Backdrop, Glass, Pill } from '~/lib/glass'
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
  }

  return (
    <Backdrop>
      {/* header */}
      <Glass style={styles.header} intensity={34}>
        <Pressable onPress={() => setDrawer(true)} hitSlop={10} style={styles.menu}>
          <View style={styles.bar} />
          <View style={[styles.bar, { width: 11 }]} />
          <View style={styles.bar} />
        </Pressable>
        <View style={{ flex: 1 }}>
          <T w="sb" s={15} numberOfLines={1}>
            <Prompt />
            {wsName}
          </T>
          <T v="faint" s={11}>
            {offline ? 'offline · cached' : `my issues · ${items.length}`}
          </T>
        </View>
        <Pressable style={styles.searchBtn} hitSlop={8}>
          <T v="dim" s={15}>
            {'>'}
            <T v="faint" s={15}>_</T>
          </T>
        </Pressable>
      </Glass>

      {/* segmented scope */}
      <View style={styles.segWrap}>
        <View style={styles.seg}>
          {(['assigned', 'created'] as IssueScope[]).map((s) => (
            <Pressable key={s} onPress={() => setScope(s)} style={[styles.segItem, scope === s && styles.segOn]}>
              <T v={scope === s ? 'bright' : 'faint'} w={scope === s ? 'sb' : 'r'} s={13}>
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
          <Pressable onPress={() => load()}>
            <T v="accent" w="m">
              [ retry ]
            </T>
          </Pressable>
        </View>
      ) : (
        <FlatList
          data={items}
          keyExtractor={(i) => i.id}
          contentContainerStyle={items.length === 0 ? styles.center : { paddingHorizontal: 14, paddingTop: 4, paddingBottom: 110, gap: 11 }}
          showsVerticalScrollIndicator={false}
          refreshControl={<RefreshControl refreshing={refreshing} onRefresh={() => load(true)} tintColor={C.accent} />}
          ListEmptyComponent={<T v="faint">// no {scope} issues</T>}
          renderItem={({ item }) => {
            const p = PRI[item.priority]
            const st = stateById(item.stateId)
            const asg = userById(item.assigneeId)
            const tk = teamById(item.teamId)?.key
            return (
              <Pressable onPress={() => router.push(`/issue/${item.id}`)} style={({ pressed }) => ({ transform: [{ scale: pressed ? 0.985 : 1 }] })}>
                <Glass style={styles.card} intensity={22}>
                  {p ? <View style={[styles.edge, { backgroundColor: p.color, shadowColor: p.color }]} /> : null}
                  <View style={styles.cardIn}>
                    <View style={styles.cardTop}>
                      {st ? <Pill color={stateColor(st.colorToken)} label={st.name} /> : <View />}
                      <View style={{ flex: 1 }} />
                      {p ? (
                        <T s={11} w="m" style={{ color: p.color }}>
                          {p.label}
                        </T>
                      ) : null}
                    </View>
                    <T v="bright" s={16} w="m" numberOfLines={2} style={{ lineHeight: 21 }}>
                      {item.title}
                    </T>
                    <View style={styles.cardBot}>
                      <T v="accent" s={11.5} w="m">
                        {item.identifier}
                      </T>
                      {tk ? <T v="faint" s={11.5}>· {tk}</T> : null}
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
      <Pressable onPress={() => router.push('/create')} style={({ pressed }) => [styles.fab, { transform: [{ scale: pressed ? 0.92 : 1 }] }]}>
        <LinearGradient colors={['#8fb0ff', '#7aa2f7']} style={styles.fabGrad}>
          <T style={{ color: C.base, fontSize: 28, marginTop: -3, fontWeight: '600' }}>+</T>
        </LinearGradient>
      </Pressable>

      <Sidebar visible={drawer} onClose={() => setDrawer(false)} active="issues" onSelect={onSection} />
    </Backdrop>
  )
}

const styles = StyleSheet.create({
  header: { flexDirection: 'row', alignItems: 'center', gap: 13, marginTop: 54, marginHorizontal: 12, paddingHorizontal: 15, paddingVertical: 11, borderRadius: 16 },
  menu: { gap: 3.5, paddingVertical: 2 },
  bar: { width: 17, height: 2, borderRadius: 2, backgroundColor: C.secondary },
  searchBtn: { paddingHorizontal: 4 },
  segWrap: { paddingHorizontal: 14, paddingVertical: 14 },
  seg: { flexDirection: 'row', gap: 4, backgroundColor: 'rgba(255,255,255,0.03)', borderRadius: 11, padding: 4, borderWidth: 1, borderColor: 'rgba(255,255,255,0.05)', alignSelf: 'flex-start' },
  segItem: { paddingVertical: 6, paddingHorizontal: 16, borderRadius: 8 },
  segOn: { backgroundColor: 'rgba(122,162,247,0.14)' },
  card: { borderRadius: 16 },
  edge: { position: 'absolute', left: 0, top: 12, bottom: 12, width: 3, borderRadius: 2, shadowOpacity: 0.9, shadowRadius: 5, shadowOffset: { width: 0, height: 0 } },
  cardIn: { padding: 14, paddingLeft: 17, gap: 9 },
  cardTop: { flexDirection: 'row', alignItems: 'center' },
  cardBot: { flexDirection: 'row', alignItems: 'center', gap: 5 },
  fab: { position: 'absolute', right: 20, bottom: 30, borderRadius: 30, shadowColor: C.accent, shadowOpacity: 0.5, shadowRadius: 16, shadowOffset: { width: 0, height: 6 } },
  fabGrad: { width: 58, height: 58, borderRadius: 29, alignItems: 'center', justifyContent: 'center' },
  center: { flexGrow: 1, alignItems: 'center', justifyContent: 'center', gap: 14, paddingTop: 60 },
})
