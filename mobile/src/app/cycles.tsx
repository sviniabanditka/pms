import { useRouter } from 'expo-router'
import { ScrollView, Pressable, StyleSheet, View } from 'react-native'
import Animated, { FadeInDown } from 'react-native-reanimated'

import { Screen } from '~/components/Screen'
import { Glass } from '~/lib/glass'
import { useMeta } from '~/lib/meta'
import { S } from '~/lib/store'
import { C } from '~/lib/theme'
import { T } from '~/lib/ui'

function dstr(s?: string) { return s ? new Date(s).toLocaleDateString(undefined, { month: 'short', day: 'numeric' }) : '—' }

export default function Cycles() {
  useMeta()
  const router = useRouter()
  const teams = S.teams()
  return (
    <Screen title="cycles" active="cycles" titleColor={C.warn}>
      <ScrollView contentContainerStyle={{ padding: 12, gap: 16, paddingBottom: 40 }} showsVerticalScrollIndicator={false}>
        {teams.map((t: any) => {
          const cs = S.cyclesByTeam(t.id)
          if (!cs.length) return null
          return (
            <View key={t.id} style={{ gap: 8 }}>
              <T v="dim" w="sb" s={13} style={{ paddingHorizontal: 6 }}>
                <T v="accent" s={13} w="sb">{t.key}</T> {t.name}
              </T>
              {cs.map((c: any, i: number) => {
                const n = S.issuesByCycle(c.id).length
                return (
                  <Animated.View key={c.id} entering={FadeInDown.duration(220).delay(i * 30)}>
                    <Pressable onPress={() => router.push(`/cycle/${c.id}`)} style={({ pressed }) => ({ transform: [{ scale: pressed ? 0.99 : 1 }] })}>
                      <Glass style={styles.card} intensity={20}>
                        <View style={{ flex: 1 }}>
                          <T v="bright" w="m" s={15}>Cycle {c.number}{c.name ? ` · ${c.name}` : ''}</T>
                          <T v="faint" s={11.5} style={{ marginTop: 3 }}>{dstr(c.startsOn)} — {dstr(c.endsOn)} · {n} issues</T>
                        </View>
                        <T v="faint" s={16}>›</T>
                      </Glass>
                    </Pressable>
                  </Animated.View>
                )
              })}
            </View>
          )
        })}
      </ScrollView>
    </Screen>
  )
}

const styles = StyleSheet.create({
  card: { flexDirection: 'row', alignItems: 'center', gap: 10, borderRadius: 14, padding: 14 },
})
