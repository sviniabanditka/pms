import { useLocalSearchParams, useRouter } from 'expo-router'
import { ScrollView, StyleSheet, View } from 'react-native'
import { useSafeAreaInsets } from 'react-native-safe-area-context'

import { Backdrop, Glass } from '~/lib/glass'
import { useMeta } from '~/lib/meta'
import { S } from '~/lib/store'
import { C } from '~/lib/theme'
import { Bracket, T } from '~/lib/ui'

function bodyText(body: any): string {
  if (!body) return ''
  if (typeof body === 'string') return body
  if (body.md) return body.md
  try { return JSON.stringify(body, null, 2) } catch { return '' }
}

export default function Doc() {
  const { id } = useLocalSearchParams<{ id: string }>()
  const router = useRouter()
  const insets = useSafeAreaInsets()
  useMeta()
  const doc = S.doc(id)
  return (
    <Backdrop>
      <Glass style={[styles.header, { marginTop: insets.top + 10 }]} intensity={34}>
        <Bracket label="‹ back" onPress={() => router.back()} />
      </Glass>
      <ScrollView contentContainerStyle={{ padding: 18, paddingBottom: 40 }} showsVerticalScrollIndicator={false}>
        <T s={32} style={{ marginBottom: 6 }}>{doc?.icon || '¶'}</T>
        <T v="bright" w="b" s={24} style={{ lineHeight: 30, marginBottom: 16 }}>
          {doc?.title || 'Untitled'}
        </T>
        <T v="dim" s={14} style={{ lineHeight: 22 }}>
          {bodyText(doc?.body) || '// empty'}
        </T>
      </ScrollView>
    </Backdrop>
  )
}

const styles = StyleSheet.create({
  header: { flexDirection: 'row', alignItems: 'center', marginHorizontal: 12, paddingHorizontal: 15, paddingVertical: 11, borderRadius: 16 },
})
