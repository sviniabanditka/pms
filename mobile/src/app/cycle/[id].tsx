import { useLocalSearchParams, useRouter } from 'expo-router'
import { useMemo } from 'react'
import { ScrollView, StyleSheet, View } from 'react-native'
import { useSafeAreaInsets } from 'react-native-safe-area-context'

import { IssueRow } from '~/components/IssueRow'
import { Backdrop, Glass, GlowDot } from '~/lib/glass'
import { stateColor, useMeta } from '~/lib/meta'
import { S } from '~/lib/store'
import { C } from '~/lib/theme'
import { Bracket, T } from '~/lib/ui'

export default function Cycle() {
  const { id } = useLocalSearchParams<{ id: string }>()
  const router = useRouter()
  const insets = useSafeAreaInsets()
  const v = useMeta()
  const cyc = S.cycle(id)
  const states = cyc ? S.statesByTeam(cyc.teamId) : []
  const groups = useMemo(() => {
    const by = new Map<string, any[]>()
    for (const s of states) by.set(s.id, [])
    for (const i of S.issuesByCycle(id)) by.get(i.stateId)?.push(i)
    return by
  }, [id, states.length, v])
  return (
    <Backdrop>
      <Glass style={[styles.header, { marginTop: insets.top + 10 }]} intensity={34}>
        <Bracket label="‹" onPress={() => router.back()} />
        <T w="sb" s={15} style={{ flex: 1 }}>Cycle {cyc?.number}{cyc?.name ? ` · ${cyc.name}` : ''}</T>
      </Glass>
      <ScrollView contentContainerStyle={{ padding: 12, gap: 14, paddingBottom: 40 }} showsVerticalScrollIndicator={false}>
        {states.map((s: any) => {
          const arr = groups.get(s.id) || []
          if (!arr.length) return null
          return (
            <View key={s.id}>
              <View style={styles.gh}>
                <GlowDot color={stateColor(s.colorToken)} size={9} />
                <T v="dim" w="sb" s={13}>{s.name}</T>
                <T v="faint" s={12}>{arr.length}</T>
              </View>
              <Glass style={styles.group} intensity={18}>
                {arr.map((i: any, idx: number) => (
                  <View key={i.id}>
                    {idx > 0 ? <View style={styles.sep} /> : null}
                    <IssueRow issue={i} onPress={() => router.push(`/issue/${i.id}`)} showTeam={false} showState={false} />
                  </View>
                ))}
              </Glass>
            </View>
          )
        })}
      </ScrollView>
    </Backdrop>
  )
}

const styles = StyleSheet.create({
  header: { flexDirection: 'row', alignItems: 'center', gap: 12, marginHorizontal: 12, paddingHorizontal: 15, paddingVertical: 11, borderRadius: 16 },
  gh: { flexDirection: 'row', alignItems: 'center', gap: 8, paddingHorizontal: 6, paddingBottom: 8 },
  group: { borderRadius: 16, overflow: 'hidden' },
  sep: { height: 1, backgroundColor: 'rgba(255,255,255,0.05)', marginLeft: 42 },
})
