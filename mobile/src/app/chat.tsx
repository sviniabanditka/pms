import { FlatList, StyleSheet, View } from 'react-native'
import Animated, { FadeInDown } from 'react-native-reanimated'

import { Screen } from '~/components/Screen'
import { Glass } from '~/lib/glass'
import { useMeta } from '~/lib/meta'
import { S } from '~/lib/store'
import { C } from '~/lib/theme'
import { T } from '~/lib/ui'

export default function Chat() {
  useMeta()
  const channels = S.channels()
  return (
    <Screen title="chat" active="chat" titleColor={C.accent2} sub={`${channels.length} channels`}>
      <FlatList
        data={channels}
        keyExtractor={(c: any) => c.id}
        contentContainerStyle={{ padding: 12, gap: 8, paddingBottom: 40 }}
        showsVerticalScrollIndicator={false}
        renderItem={({ item, index }: any) => {
          const unread = S.unread(item.id)
          return (
            <Animated.View entering={FadeInDown.duration(220).delay(Math.min(index, 12) * 22)}>
              <Glass style={styles.row} intensity={18}>
                <T v="faint" s={16} w="m">#</T>
                <T v={unread ? 'bright' : 'dim'} w={unread ? 'm' : 'r'} s={15} style={{ flex: 1 }} numberOfLines={1}>{item.name}</T>
                {unread ? <View style={styles.badge}><T s={11} w="sb" style={{ color: C.base }}>{unread}</T></View> : null}
                {item.private ? <T v="faint" s={12}>🔒</T> : null}
              </Glass>
            </Animated.View>
          )
        }}
      />
    </Screen>
  )
}

const styles = StyleSheet.create({
  row: { flexDirection: 'row', alignItems: 'center', gap: 10, borderRadius: 12, padding: 14 },
  badge: { backgroundColor: C.accent, borderRadius: 10, minWidth: 20, paddingHorizontal: 6, paddingVertical: 1, alignItems: 'center' },
})
