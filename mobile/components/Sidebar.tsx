// Слайд-ин сайдбар со всеми секциями десктопа (структура паритета) в liquid-glass.
import { BlurView } from 'expo-blur'
import { useRouter } from 'expo-router'
import { useEffect, useRef } from 'react'
import { Animated, Dimensions, Pressable, ScrollView, StyleSheet, View } from 'react-native'

import { allTeams } from '~/lib/meta'
import { getWorkspaces, useActiveWs } from '~/lib/workspace'
import { C } from '~/lib/theme'
import { GlowDot } from '~/lib/glass'
import { Prompt, T } from '~/lib/ui'

const W = Math.min(320, Dimensions.get('window').width * 0.84)

export type Section = 'issues' | 'cycles' | 'docs' | 'chat' | 'meetings' | 'calendar' | 'dashboards' | 'inbox' | 'reports' | 'settings'

const SECTIONS: { key: Section; label: string; glyph: string; color: string }[] = [
  { key: 'issues', label: 'issues', glyph: '▚', color: C.accent },
  { key: 'cycles', label: 'cycles', glyph: '◷', color: C.warn },
  { key: 'dashboards', label: 'dashboards', glyph: '▤', color: C.info },
  { key: 'docs', label: 'docs', glyph: '¶', color: C.ok },
  { key: 'chat', label: 'chat', glyph: '#', color: C.accent2 },
  { key: 'meetings', label: 'meetings', glyph: '◉', color: C.danger },
  { key: 'calendar', label: 'calendar', glyph: '▦', color: C.info },
  { key: 'inbox', label: 'inbox', glyph: '✉', color: C.accent },
  { key: 'reports', label: 'reports', glyph: '∿', color: C.ok },
]

export function Sidebar({ visible, onClose, active, onSelect }: { visible: boolean; onClose: () => void; active: Section; onSelect: (s: Section) => void }) {
  const router = useRouter()
  const activeWs = useActiveWs()
  const x = useRef(new Animated.Value(-W)).current
  const fade = useRef(new Animated.Value(0)).current
  const wsName = getWorkspaces().find((w) => w.id === activeWs)?.name ?? 'workspace'
  const teams = allTeams()

  useEffect(() => {
    Animated.parallel([
      Animated.timing(x, { toValue: visible ? 0 : -W, duration: 220, useNativeDriver: true }),
      Animated.timing(fade, { toValue: visible ? 1 : 0, duration: 220, useNativeDriver: true }),
    ]).start()
  }, [visible])

  if (!visible) return null

  return (
    <View style={StyleSheet.absoluteFill} pointerEvents="box-none">
      <Animated.View style={[StyleSheet.absoluteFill, { opacity: fade }]} pointerEvents={visible ? 'auto' : 'none'}>
        <Pressable style={[StyleSheet.absoluteFill, { backgroundColor: 'rgba(4,5,10,0.6)' }]} onPress={onClose} />
      </Animated.View>

      <Animated.View style={[styles.panel, { transform: [{ translateX: x }] }]}>
        <BlurView intensity={40} tint="dark" style={StyleSheet.absoluteFill} />
        <View style={styles.sheen} />

        <View style={styles.header}>
          <Pressable style={styles.ws} onPress={onClose}>
            <T w="b" s={17} numberOfLines={1}>
              <Prompt />
              {wsName}
            </T>
            <T v="faint" s={12}>
              ▾
            </T>
          </Pressable>
          <View style={styles.search}>
            <T v="ok" w="m" s={13}>
              {'>'}
            </T>
            <T v="faint" s={13}>
              search…
            </T>
          </View>
        </View>

        <ScrollView showsVerticalScrollIndicator={false} contentContainerStyle={{ paddingBottom: 24 }}>
          {SECTIONS.map((s) => {
            const on = s.key === active
            return (
              <Pressable
                key={s.key}
                style={[styles.item, on && styles.itemOn]}
                onPress={() => {
                  onSelect(s.key)
                  onClose()
                }}
              >
                {on ? <View style={[styles.activeBar, { backgroundColor: s.color, shadowColor: s.color }]} /> : null}
                <T style={{ width: 22, textAlign: 'center', color: s.color, fontSize: 14 }}>{s.glyph}</T>
                <T v={on ? 'bright' : 'dim'} w={on ? 'sb' : 'r'} s={15}>
                  {s.label}
                </T>
              </Pressable>
            )
          })}

          <View style={styles.sep} />
          <T v="accent" s={12} style={{ paddingHorizontal: 18, marginBottom: 6 }}>
            # teams
          </T>
          {teams.map((t) => (
            <Pressable
              key={t.id}
              style={styles.team}
              onPress={() => {
                onClose()
                router.push(`/team/${t.id}`)
              }}
            >
              <GlowDot color={C.tertiary} size={6} />
              <T v="dim" s={14}>
                {t.key}
              </T>
              <T v="faint" s={13} numberOfLines={1} style={{ flex: 1 }}>
                {t.name}
              </T>
            </Pressable>
          ))}
        </ScrollView>

        <View style={styles.footer}>
          <Pressable style={styles.item} onPress={() => { onSelect('settings'); onClose() }}>
            <T style={{ width: 22, textAlign: 'center', color: C.tertiary, fontSize: 14 }}>⚙</T>
            <T v="dim" s={15}>
              settings
            </T>
          </Pressable>
        </View>
      </Animated.View>
    </View>
  )
}

const styles = StyleSheet.create({
  panel: { position: 'absolute', left: 0, top: 0, bottom: 0, width: W, paddingTop: 60, borderRightWidth: 1, borderRightColor: 'rgba(255,255,255,0.08)', backgroundColor: 'rgba(11,12,18,0.72)' },
  sheen: { position: 'absolute', top: 0, left: 0, right: 0, height: 120, backgroundColor: 'rgba(122,162,247,0.05)' },
  header: { paddingHorizontal: 16, gap: 12, marginBottom: 12 },
  ws: { flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between' },
  search: { flexDirection: 'row', alignItems: 'center', gap: 8, borderWidth: 1, borderColor: 'rgba(255,255,255,0.08)', borderRadius: 10, paddingHorizontal: 12, paddingVertical: 9, backgroundColor: 'rgba(255,255,255,0.03)' },
  item: { flexDirection: 'row', alignItems: 'center', gap: 12, paddingVertical: 11, paddingHorizontal: 16 },
  itemOn: { backgroundColor: 'rgba(122,162,247,0.08)' },
  activeBar: { position: 'absolute', left: 0, top: 6, bottom: 6, width: 3, borderRadius: 2, shadowOpacity: 0.8, shadowRadius: 4, shadowOffset: { width: 0, height: 0 } },
  sep: { height: 1, backgroundColor: 'rgba(255,255,255,0.06)', marginVertical: 12, marginHorizontal: 16 },
  team: { flexDirection: 'row', alignItems: 'center', gap: 10, paddingVertical: 8, paddingHorizontal: 18 },
  footer: { borderTopWidth: 1, borderTopColor: 'rgba(255,255,255,0.06)', paddingVertical: 6, paddingBottom: 24 },
})
