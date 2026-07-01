import { LinearGradient } from 'expo-linear-gradient'
import { useFocusEffect, useRouter } from 'expo-router'
import { useCallback, useState } from 'react'
import { FlatList, Pressable, RefreshControl, StyleSheet, View } from 'react-native'
import Animated, { FadeInDown } from 'react-native-reanimated'
import { useSafeAreaInsets } from 'react-native-safe-area-context'

import { Sidebar, type Section } from '~/components/Sidebar'
import { sectionRoute } from '~/components/Screen'
import { getCurrentUserId } from '~/lib/auth'
import { cachedIssues, myIssues, type Issue, type IssueScope } from '~/lib/api'
import { Avatar, Backdrop, Glass, GlowDot } from '~/lib/glass'
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
  const insets = useSafeAreaInsets()
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
    else if (s === 'settings') router.push('/profile')
  }

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
            <T v="faint" s={15}>
              _
            </T>
          </T>
        </Pressable>
      </Glass>

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
        <Glass style={styles.list} intensity={18}>
          <FlatList
            data={items}
            keyExtractor={(i) => i.id}
            contentContainerStyle={items.length === 0 ? styles.center : { paddingVertical: 4 }}
            showsVerticalScrollIndicator={false}
            refreshControl={<RefreshControl refreshing={refreshing} onRefresh={() => load(true)} tintColor={C.accent} />}
            ItemSeparatorComponent={() => <View style={styles.sep} />}
            ListEmptyComponent={<T v="faint">// no {scope} issues</T>}
            renderItem={({ item, index }) => {
              const p = PRI[item.priority]
              const st = stateById(item.stateId)
              const asg = userById(item.assigneeId)
              const tk = teamById(item.teamId)?.key
              return (
                <Animated.View entering={FadeInDown.duration(240).delay(Math.min(index, 14) * 26)}>
                  <Pressable onPress={() => router.push(`/issue/${item.id}`)} style={({ pressed }) => [styles.row, pressed && styles.rowPressed]}>
                    <View style={[styles.priBar, { backgroundColor: p ? p.color : 'transparent', shadowColor: p?.color }]} />
                    {st ? <GlowDot color={stateColor(st.colorToken)} size={9} /> : null}
                    <View style={{ flex: 1, gap: 2 }}>
                      <T v="bright" s={14.5} w="m" numberOfLines={1}>
                        {item.title}
                      </T>
                      <View style={styles.meta}>
                        <T v="accent" s={11}>
                          {item.identifier}
                        </T>
                        {tk ? <T v="faint" s={11}>· {tk}</T> : null}
                        <T v="faint" s={11}>
                          · {ago(item.updatedAt)}
                        </T>
                      </View>
                    </View>
                    {asg ? <Avatar name={asg.name} id={asg.id} size={22} /> : null}
                  </Pressable>
                </Animated.View>
              )
            }}
          />
        </Glass>
      )}

      <Pressable onPress={() => router.push('/create')} style={({ pressed }) => [styles.fab, { transform: [{ scale: pressed ? 0.92 : 1 }] }]}>
        <LinearGradient colors={['#8fb0ff', '#7aa2f7']} style={styles.fabGrad}>
          <T style={{ color: C.base, fontSize: 28, marginTop: -3, fontWeight: '600' }}>+</T>
        </LinearGradient>
      </Pressable>

      <Sidebar visible={drawer} onClose={() => setDrawer(false)} active="issues" onSelect={(x) => sectionRoute(router, x)} />
    </Backdrop>
  )
}

const styles = StyleSheet.create({
  header: { flexDirection: 'row', alignItems: 'center', gap: 18, marginHorizontal: 12, paddingHorizontal: 15, paddingVertical: 11, borderRadius: 18 },
  menu: { gap: 3.5, paddingVertical: 2 },
  barIcon: { width: 17, height: 2, borderRadius: 2, backgroundColor: C.secondary },
  searchBtn: { paddingHorizontal: 4 },
  segWrap: { paddingHorizontal: 14, paddingVertical: 12 },
  seg: { flexDirection: 'row', gap: 4, backgroundColor: 'rgba(255,255,255,0.03)', borderRadius: 12, padding: 4, borderWidth: 1, borderColor: 'rgba(255,255,255,0.05)', alignSelf: 'flex-start' },
  segItem: { paddingVertical: 6, paddingHorizontal: 16, borderRadius: 9 },
  segOn: { backgroundColor: 'rgba(122,162,247,0.14)' },
  list: { flex: 1, marginHorizontal: 12, marginBottom: 12, borderRadius: 20 },
  row: { flexDirection: 'row', alignItems: 'center', gap: 11, paddingVertical: 10, paddingHorizontal: 14 },
  rowPressed: { backgroundColor: 'rgba(122,162,247,0.07)' },
  priBar: { width: 3, height: 26, borderRadius: 2, shadowOpacity: 0.9, shadowRadius: 4, shadowOffset: { width: 0, height: 0 } },
  meta: { flexDirection: 'row', alignItems: 'center', gap: 4 },
  sep: { height: 1, backgroundColor: 'rgba(255,255,255,0.05)', marginLeft: 42 },
  fab: { position: 'absolute', right: 20, bottom: 30, borderRadius: 30, shadowColor: C.accent, shadowOpacity: 0.5, shadowRadius: 16, shadowOffset: { width: 0, height: 6 } },
  fabGrad: { width: 58, height: 58, borderRadius: 29, alignItems: 'center', justifyContent: 'center' },
  center: { flexGrow: 1, alignItems: 'center', justifyContent: 'center', gap: 14, paddingTop: 60 },
})
