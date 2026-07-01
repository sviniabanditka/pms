// Компактная строка issue (Linear-плотность) — переиспользуется в my-issues, команде, цикле.
import { Pressable, StyleSheet, View } from 'react-native'

import { Avatar, GlowDot } from '~/lib/glass'
import { stateById, stateColor, teamById, userById } from '~/lib/meta'
import { C } from '~/lib/theme'
import { T } from '~/lib/ui'

export const PRI: Record<number, { label: string; color: string }> = {
  1: { label: 'urgent', color: C.danger },
  2: { label: 'high', color: C.warn },
  3: { label: 'med', color: '#e0af68' },
  4: { label: 'low', color: C.tertiary },
}

export function ago(iso: string) {
  const d = Date.now() - new Date(iso).getTime()
  const m = Math.floor(d / 60000)
  if (m < 1) return 'now'
  if (m < 60) return `${m}m`
  const h = Math.floor(m / 60)
  if (h < 24) return `${h}h`
  return `${Math.floor(h / 24)}d`
}

export function IssueRow({ issue, onPress, showTeam = true, showState = true }: { issue: any; onPress: () => void; showTeam?: boolean; showState?: boolean }) {
  const p = PRI[issue.priority]
  const st = stateById(issue.stateId)
  const asg = userById(issue.assigneeId)
  const tk = showTeam ? teamById(issue.teamId)?.key : null
  return (
    <Pressable onPress={onPress} style={({ pressed }) => [styles.row, pressed && styles.rowPressed]}>
      <View style={[styles.priBar, { backgroundColor: p ? p.color : 'transparent', shadowColor: p?.color }]} />
      {showState && st ? <GlowDot color={stateColor(st.colorToken)} size={9} /> : null}
      <View style={{ flex: 1, gap: 2 }}>
        <T v="bright" s={14.5} w="m" numberOfLines={1}>
          {issue.title}
        </T>
        <View style={styles.meta}>
          <T v="accent" s={11}>
            {issue.identifier}
          </T>
          {tk ? (
            <T v="faint" s={11}>
              · {tk}
            </T>
          ) : null}
          <T v="faint" s={11}>
            · {ago(issue.updatedAt)}
          </T>
        </View>
      </View>
      {asg ? <Avatar name={asg.name} id={asg.id} size={22} /> : null}
    </Pressable>
  )
}

const styles = StyleSheet.create({
  row: { flexDirection: 'row', alignItems: 'center', gap: 11, paddingVertical: 10, paddingHorizontal: 14 },
  rowPressed: { backgroundColor: 'rgba(122,162,247,0.07)' },
  priBar: { width: 3, height: 26, borderRadius: 2, shadowOpacity: 0.9, shadowRadius: 4, shadowOffset: { width: 0, height: 0 } },
  meta: { flexDirection: 'row', alignItems: 'center', gap: 4 },
})
