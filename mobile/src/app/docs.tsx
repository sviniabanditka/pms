import { useRouter } from 'expo-router'
import { FlatList, Pressable, StyleSheet } from 'react-native'
import Animated, { FadeInDown } from 'react-native-reanimated'

import { Screen } from '~/components/Screen'
import { Glass } from '~/lib/glass'
import { useMeta } from '~/lib/meta'
import { S } from '~/lib/store'
import { C } from '~/lib/theme'
import { T } from '~/lib/ui'

export default function Docs() {
  useMeta()
  const router = useRouter()
  const docs = S.docs().sort((a: any, b: any) => (a.position < b.position ? -1 : 1))
  return (
    <Screen title="docs" active="docs" titleColor={C.ok} sub={`${docs.length} documents`}>
      <FlatList
        data={docs}
        keyExtractor={(d: any) => d.id}
        contentContainerStyle={{ padding: 12, gap: 10, paddingBottom: 40 }}
        showsVerticalScrollIndicator={false}
        renderItem={({ item, index }: any) => (
          <Animated.View entering={FadeInDown.duration(240).delay(Math.min(index, 12) * 24)}>
            <Pressable onPress={() => router.push(`/doc/${item.id}`)} style={({ pressed }) => ({ transform: [{ scale: pressed ? 0.99 : 1 }] })}>
              <Glass style={styles.card} intensity={20}>
                <T s={20}>{item.icon || '¶'}</T>
                <T v="bright" w="m" s={15} numberOfLines={1} style={{ flex: 1 }}>
                  {item.title || 'Untitled'}
                </T>
              </Glass>
            </Pressable>
          </Animated.View>
        )}
      />
    </Screen>
  )
}

const styles = StyleSheet.create({
  card: { flexDirection: 'row', alignItems: 'center', gap: 12, borderRadius: 14, padding: 15 },
})
