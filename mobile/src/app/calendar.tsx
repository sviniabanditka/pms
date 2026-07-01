import { SectionList, StyleSheet, View } from 'react-native'

import { Screen } from '~/components/Screen'
import { Glass, GlowDot } from '~/lib/glass'
import { useMeta } from '~/lib/meta'
import { S } from '~/lib/store'
import { C } from '~/lib/theme'
import { T } from '~/lib/ui'

const KC: Record<string, string> = { event: C.accent, meeting: C.danger, dayoff: C.ok }
function dayKey(s: string) { return new Date(s).toLocaleDateString(undefined, { weekday: 'short', month: 'short', day: 'numeric' }) }
function tstr(s: string, allDay?: boolean) { return allDay ? 'all day' : new Date(s).toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' }) }

export default function Calendar() {
  useMeta()
  const items: any[] = []
  for (const e of S.calendarEvents()) items.push({ id: 'e' + e.id, title: e.title, at: e.startsAt, kind: e.kind || 'event', allDay: e.allDay })
  for (const m of S.meetings()) if (m.scheduledAt) items.push({ id: 'm' + m.id, title: m.title || 'Meeting', at: m.scheduledAt, kind: 'meeting' })
  const now = Date.now()
  const upcoming = items.filter((i) => new Date(i.at).getTime() > now - 12 * 3600000).sort((a, b) => (a.at < b.at ? -1 : 1))
  const byDay = new Map<string, any[]>()
  for (const i of upcoming) { const k = dayKey(i.at); if (!byDay.has(k)) byDay.set(k, []); byDay.get(k)!.push(i) }
  const sections = [...byDay.entries()].map(([title, data]) => ({ title, data }))
  return (
    <Screen title="calendar" active="calendar" titleColor={C.info} sub={`${upcoming.length} upcoming`}>
      <SectionList
        sections={sections}
        keyExtractor={(i: any) => i.id}
        contentContainerStyle={{ padding: 12, paddingBottom: 40 }}
        showsVerticalScrollIndicator={false}
        stickySectionHeadersEnabled={false}
        ListEmptyComponent={<T v="faint" style={{ padding: 20 }}>// nothing scheduled</T>}
        renderSectionHeader={({ section }: any) => (
          <T v="accent" s={12} w="sb" style={{ paddingHorizontal: 4, paddingTop: 14, paddingBottom: 8 }}>{section.title}</T>
        )}
        renderItem={({ item }: any) => (
          <Glass style={styles.row} intensity={18}>
            <GlowDot color={KC[item.kind] || C.accent} size={8} />
            <T v="bright" s={14} style={{ flex: 1 }} numberOfLines={1}>{item.title}</T>
            <T v="faint" s={11.5}>{tstr(item.at, item.allDay)}</T>
          </Glass>
        )}
        SectionSeparatorComponent={() => <View style={{ height: 6 }} />}
        ItemSeparatorComponent={() => <View style={{ height: 8 }} />}
      />
    </Screen>
  )
}

const styles = StyleSheet.create({ row: { flexDirection: 'row', alignItems: 'center', gap: 10, borderRadius: 12, padding: 13 } })
