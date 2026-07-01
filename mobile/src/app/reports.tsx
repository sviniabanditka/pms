import { ScrollView, StyleSheet, View } from 'react-native'

import { Screen } from '~/components/Screen'
import { Avatar, Glass } from '~/lib/glass'
import { useMeta, userById } from '~/lib/meta'
import { S } from '~/lib/store'
import { C } from '~/lib/theme'
import { T } from '~/lib/ui'

function hrs(ms: number) { return (ms / 3600000).toFixed(1) + 'h' }

export default function Reports() {
  useMeta()
  const entries = S.timeEntries()
  const byUser = new Map<string, number>()
  let total = 0
  for (const e of entries) {
    if (!e.endedAt) continue
    const d = new Date(e.endedAt).getTime() - new Date(e.startedAt).getTime()
    if (d <= 0) continue
    byUser.set(e.userId, (byUser.get(e.userId) || 0) + d)
    total += d
  }
  const rows = [...byUser.entries()].sort((a, b) => b[1] - a[1])
  return (
    <Screen title="reports" active="reports" titleColor={C.ok} sub="time tracked">
      <ScrollView contentContainerStyle={{ padding: 12, gap: 12, paddingBottom: 40 }} showsVerticalScrollIndicator={false}>
        <Glass style={styles.total} intensity={24}>
          <T v="faint" s={12}>total tracked</T>
          <T v="bright" w="b" s={30} style={{ marginTop: 4 }}>{hrs(total)}</T>
          <T v="faint" s={11} style={{ marginTop: 2 }}>{entries.length} entries · {rows.length} people</T>
        </Glass>
        <T v="accent" s={13} style={{ paddingHorizontal: 4, marginTop: 4 }}># by person</T>
        {rows.map(([uid, ms]) => {
          const u = userById(uid)
          return (
            <Glass key={uid} style={styles.row} intensity={18}>
              <Avatar name={u?.name} id={uid} size={26} />
              <T v="body" s={14} style={{ flex: 1 }}>{u?.name || 'user'}</T>
              <T v="bright" w="sb" s={14}>{hrs(ms)}</T>
            </Glass>
          )
        })}
        {!rows.length ? <T v="faint" style={{ padding: 12 }}>// no time entries</T> : null}
      </ScrollView>
    </Screen>
  )
}

const styles = StyleSheet.create({
  total: { borderRadius: 18, padding: 18, alignItems: 'flex-start' },
  row: { flexDirection: 'row', alignItems: 'center', gap: 11, borderRadius: 12, padding: 12 },
})
