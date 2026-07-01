import { ScrollView, StyleSheet, View } from 'react-native'

import { Screen } from '~/components/Screen'
import { Glass } from '~/lib/glass'
import { useMeta } from '~/lib/meta'
import { S } from '~/lib/store'
import { C } from '~/lib/theme'
import { T } from '~/lib/ui'

export default function Dashboards() {
  useMeta()
  const dashboards = S.dashboards()
  return (
    <Screen title="dashboards" active="dashboards" titleColor={C.info}>
      <ScrollView contentContainerStyle={{ padding: 12, gap: 18, paddingBottom: 40 }} showsVerticalScrollIndicator={false}>
        {dashboards.map((d: any) => {
          const widgets = S.widgetsByDashboard(d.id)
          return (
            <View key={d.id} style={{ gap: 10 }}>
              <T v="bright" w="sb" s={16} style={{ paddingHorizontal: 4 }}><T v="accent" s={16} w="sb"># </T>{d.name}</T>
              <View style={styles.grid}>
                {widgets.map((w: any) => (
                  <Glass key={w.id} style={styles.widget} intensity={20}>
                    <T v="faint" s={11}>{w.type}</T>
                    <T v="bright" w="m" s={14} numberOfLines={2} style={{ marginTop: 4 }}>{w.title || w.type}</T>
                  </Glass>
                ))}
                {!widgets.length ? <T v="faint" s={13} style={{ padding: 8 }}>// no widgets</T> : null}
              </View>
            </View>
          )
        })}
        {!dashboards.length ? <T v="faint" style={{ padding: 12 }}>// no dashboards</T> : null}
      </ScrollView>
    </Screen>
  )
}

const styles = StyleSheet.create({
  grid: { flexDirection: 'row', flexWrap: 'wrap', gap: 10 },
  widget: { borderRadius: 14, padding: 14, width: '47.5%', minHeight: 84, justifyContent: 'flex-start' },
})
