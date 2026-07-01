import { FlatList, StyleSheet, View } from 'react-native'
import Animated, { FadeInDown } from 'react-native-reanimated'

import { Screen } from '~/components/Screen'
import { Avatar, Glass, GlowDot } from '~/lib/glass'
import { useMeta, userById } from '~/lib/meta'
import { S } from '~/lib/store'
import { C } from '~/lib/theme'
import { T } from '~/lib/ui'

const STC: Record<string, string> = { live: C.ok, scheduled: C.accent, ended: C.tertiary }
function when(s?: string) { return s ? new Date(s).toLocaleString(undefined, { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' }) : '' }

export default function Meetings() {
  useMeta()
  const list = S.meetings()
  return (
    <Screen title="meetings" active="meetings" titleColor={C.danger} sub={`${list.length} total`}>
      <FlatList
        data={list}
        keyExtractor={(m: any) => m.id}
        contentContainerStyle={{ padding: 12, gap: 10, paddingBottom: 40 }}
        showsVerticalScrollIndicator={false}
        ListEmptyComponent={<T v="faint" style={{ padding: 20 }}>// no meetings</T>}
        renderItem={({ item, index }: any) => {
          const host = userById(item.createdBy)
          return (
            <Animated.View entering={FadeInDown.duration(220).delay(Math.min(index, 12) * 24)}>
              <Glass style={styles.card} intensity={20}>
                <GlowDot color={STC[item.status] || C.tertiary} size={9} />
                <View style={{ flex: 1 }}>
                  <T v="bright" w="m" s={15} numberOfLines={1}>{item.title || 'Meeting'}</T>
                  <T v="faint" s={11.5} style={{ marginTop: 2 }}>{item.status} · {when(item.scheduledAt)}</T>
                </View>
                {host ? <Avatar name={host.name} id={host.id} size={24} /> : null}
              </Glass>
            </Animated.View>
          )
        }}
      />
    </Screen>
  )
}

const styles = StyleSheet.create({ card: { flexDirection: 'row', alignItems: 'center', gap: 11, borderRadius: 14, padding: 14 } })
