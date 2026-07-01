import { useLocalSearchParams, useRouter } from 'expo-router'
import { useMemo, useState } from 'react'
import { Pressable, ScrollView, StyleSheet, View } from 'react-native'
import Animated, { FadeIn } from 'react-native-reanimated'
import { useSafeAreaInsets } from 'react-native-safe-area-context'

import { IssueRow } from '~/components/IssueRow'
import { Backdrop, Glass, GlowDot } from '~/lib/glass'
import { stateColor, useMeta } from '~/lib/meta'
import { S } from '~/lib/store'
import { C } from '~/lib/theme'
import { T } from '~/lib/ui'

export default function TeamIssues() {
  const { id } = useLocalSearchParams<{ id: string }>()
  const router = useRouter()
  const insets = useSafeAreaInsets()
  const v = useMeta()
  const [mode, setMode] = useState<'list' | 'board'>('list')

  const team = S.team(id)
  const states = S.statesByTeam(id)
  const groups = useMemo(() => {
    const issues = S.issuesByTeam(id)
    const by = new Map<string, any[]>()
    for (const s of states) by.set(s.id, [])
    for (const i of issues) by.get(i.stateId)?.push(i)
    for (const arr of by.values()) arr.sort((a, b) => (a.updatedAt < b.updatedAt ? 1 : -1))
    return by
  }, [id, states.length, v])

  const open = (iid: string) => router.push(`/issue/${iid}`)

  return (
    <Backdrop>
      <Glass style={[styles.header, { marginTop: insets.top + 10 }]} intensity={34}>
        <Pressable onPress={() => router.back()} hitSlop={10}>
          <T v="accent" s={14} w="m">
            ‹
          </T>
        </Pressable>
        <View style={{ flex: 1 }}>
          <T w="sb" s={15} numberOfLines={1}>
            <T v="accent" s={15} w="sb">
              {team?.key}
            </T>{' '}
            {team?.name}
          </T>
          <T v="faint" s={11}>
            {S.issuesByTeam(id).length} issues
          </T>
        </View>
        <View style={styles.seg}>
          {(['list', 'board'] as const).map((m) => (
            <Pressable key={m} onPress={() => setMode(m)} style={[styles.segItem, mode === m && styles.segOn]}>
              <T v={mode === m ? 'bright' : 'faint'} w={mode === m ? 'sb' : 'r'} s={12}>
                {m}
              </T>
            </Pressable>
          ))}
        </View>
      </Glass>

      {mode === 'list' ? (
        <ScrollView contentContainerStyle={{ padding: 12, gap: 14, paddingBottom: 40 }} showsVerticalScrollIndicator={false}>
          {states.map((s) => {
            const arr = groups.get(s.id) || []
            if (!arr.length) return null
            return (
              <Animated.View key={s.id} entering={FadeIn.duration(240)}>
                <View style={styles.groupHead}>
                  <GlowDot color={stateColor(s.colorToken)} size={9} />
                  <T v="dim" w="sb" s={13}>
                    {s.name}
                  </T>
                  <T v="faint" s={12}>
                    {arr.length}
                  </T>
                </View>
                <Glass style={styles.group} intensity={18}>
                  {arr.map((i, idx) => (
                    <View key={i.id}>
                      {idx > 0 ? <View style={styles.sep} /> : null}
                      <IssueRow issue={i} onPress={() => open(i.id)} showTeam={false} showState={false} />
                    </View>
                  ))}
                </Glass>
              </Animated.View>
            )
          })}
        </ScrollView>
      ) : (
        <ScrollView horizontal showsHorizontalScrollIndicator={false} contentContainerStyle={{ padding: 12, gap: 12 }}>
          {states.map((s) => {
            const arr = groups.get(s.id) || []
            return (
              <View key={s.id} style={styles.col}>
                <View style={styles.colHead}>
                  <GlowDot color={stateColor(s.colorToken)} size={9} />
                  <T v="dim" w="sb" s={13} style={{ flex: 1 }}>
                    {s.name}
                  </T>
                  <T v="faint" s={12}>
                    {arr.length}
                  </T>
                </View>
                <ScrollView showsVerticalScrollIndicator={false} contentContainerStyle={{ gap: 8, paddingBottom: 20 }}>
                  {arr.map((i) => (
                    <Pressable key={i.id} onPress={() => open(i.id)} style={({ pressed }) => ({ transform: [{ scale: pressed ? 0.98 : 1 }] })}>
                      <Glass style={styles.bcard} intensity={20}>
                        <T v="bright" s={13.5} w="m" numberOfLines={3} style={{ lineHeight: 18 }}>
                          {i.title}
                        </T>
                        <View style={styles.bmeta}>
                          <T v="accent" s={10.5}>
                            {i.identifier}
                          </T>
                          {PRIc(i.priority)}
                        </View>
                      </Glass>
                    </Pressable>
                  ))}
                  {!arr.length ? <T v="faint" s={12} style={{ padding: 8 }}>// empty</T> : null}
                </ScrollView>
              </View>
            )
          })}
        </ScrollView>
      )}
    </Backdrop>
  )
}

function PRIc(p: number) {
  const map: Record<number, { l: string; c: string }> = { 1: { l: 'urgent', c: C.danger }, 2: { l: 'high', c: C.warn }, 3: { l: 'med', c: '#e0af68' }, 4: { l: 'low', c: C.tertiary } }
  const x = map[p]
  return x ? (
    <T s={10.5} w="m" style={{ color: x.c }}>
      · {x.l}
    </T>
  ) : null
}

const styles = StyleSheet.create({
  header: { flexDirection: 'row', alignItems: 'center', gap: 14, marginHorizontal: 12, paddingHorizontal: 15, paddingVertical: 11, borderRadius: 16 },
  seg: { flexDirection: 'row', gap: 3, backgroundColor: 'rgba(255,255,255,0.03)', borderRadius: 9, padding: 3, borderWidth: 1, borderColor: 'rgba(255,255,255,0.05)' },
  segItem: { paddingVertical: 5, paddingHorizontal: 11, borderRadius: 7 },
  segOn: { backgroundColor: 'rgba(122,162,247,0.14)' },
  groupHead: { flexDirection: 'row', alignItems: 'center', gap: 8, paddingHorizontal: 6, paddingBottom: 8 },
  group: { borderRadius: 16, overflow: 'hidden' },
  sep: { height: 1, backgroundColor: 'rgba(255,255,255,0.05)', marginLeft: 42 },
  col: { width: 270 },
  colHead: { flexDirection: 'row', alignItems: 'center', gap: 8, paddingHorizontal: 4, paddingBottom: 10 },
  bcard: { borderRadius: 12, padding: 12, gap: 8 },
  bmeta: { flexDirection: 'row', alignItems: 'center', gap: 5 },
})
