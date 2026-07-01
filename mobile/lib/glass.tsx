// Liquid-glass примитивы: матовые полупрозрачные панели (expo-blur) поверх тёмной
// базы + терминальные акценты. Web-фолбэк BlurView даёт backdrop-filter.
import { BlurView } from 'expo-blur'
import { LinearGradient } from 'expo-linear-gradient'
import { StyleSheet, View, type StyleProp, type ViewStyle } from 'react-native'

import { C, F } from './theme'
import { T } from './ui'

// Backdrop — фон экрана: глубокая база + насыщенные цветовые washes, чтобы стекло
// подхватывало цвет (liquid-эффект).
export function Backdrop({ children }: { children: React.ReactNode }) {
  return (
    <View style={{ flex: 1, backgroundColor: C.base }}>
      <LinearGradient colors={['rgba(122,162,247,0.20)', 'rgba(122,162,247,0.05)', 'transparent']} style={g.washTop} pointerEvents="none" />
      <LinearGradient colors={['rgba(125,207,255,0.10)', 'transparent']} start={{ x: 1, y: 0 }} end={{ x: 0.2, y: 0.5 }} style={g.washTR} pointerEvents="none" />
      <LinearGradient colors={['transparent', 'rgba(187,154,247,0.12)']} start={{ x: 0, y: 0.3 }} end={{ x: 0.9, y: 1 }} style={g.washBottom} pointerEvents="none" />
      {children}
    </View>
  )
}

// Glass — матовая стеклянная панель: блик сверху + светящаяся верхняя кромка + тинт.
export function Glass({ children, style, intensity = 44 }: { children?: React.ReactNode; style?: StyleProp<ViewStyle>; intensity?: number }) {
  return (
    <BlurView intensity={intensity} tint="dark" style={[g.glass, style]}>
      <LinearGradient
        colors={['rgba(255,255,255,0.16)', 'rgba(255,255,255,0.04)', 'transparent']}
        locations={[0, 0.18, 0.55]}
        style={g.sheen}
        pointerEvents="none"
      />
      <View style={g.edge} pointerEvents="none" />
      {children}
    </BlurView>
  )
}

// GlowDot — статус-точка с мягким свечением
export function GlowDot({ color, size = 8 }: { color: string; size?: number }) {
  return <View style={{ width: size, height: size, borderRadius: size, backgroundColor: color, shadowColor: color, shadowOpacity: 0.9, shadowRadius: 5, shadowOffset: { width: 0, height: 0 } }} />
}

// Pill — статус-таблетка (точка + имя) на стекле
export function Pill({ color, label }: { color: string; label: string }) {
  return (
    <View style={[g.pill, { borderColor: color + '55' }]}>
      <GlowDot color={color} size={7} />
      <T v="dim" s={11.5} w="m">
        {label}
      </T>
    </View>
  )
}

const AV = ['#7aa2f7', '#9ece6a', '#bb9af7', '#7dcfff', '#ff9e64', '#f7768e']
function avColor(seed: string) {
  let h = 0
  for (let i = 0; i < seed.length; i++) h = (h * 31 + seed.charCodeAt(i)) >>> 0
  return AV[h % AV.length]
}

// Avatar — инициалы в кружке (цвет от имени)
export function Avatar({ name, id, size = 26 }: { name?: string; id?: string; size?: number }) {
  const c = avColor(id || name || '?')
  const initials = (name || '?')
    .split(' ')
    .map((w) => w[0])
    .slice(0, 2)
    .join('')
    .toUpperCase()
  return (
    <View style={{ width: size, height: size, borderRadius: size, backgroundColor: c + '22', borderWidth: 1, borderColor: c + '66', alignItems: 'center', justifyContent: 'center' }}>
      <T style={{ color: c, fontFamily: F.sb, fontSize: size * 0.4 }}>{initials}</T>
    </View>
  )
}

const g = StyleSheet.create({
  washTop: { position: 'absolute', top: 0, left: 0, right: 0, height: 460 },
  washTR: { position: 'absolute', top: 0, right: 0, width: 360, height: 360 },
  washBottom: { position: 'absolute', bottom: 0, left: 0, right: 0, height: 360 },
  sheen: { position: 'absolute', top: 0, left: 0, right: 0, height: 60 },
  edge: { position: 'absolute', top: 0, left: 14, right: 14, height: 1, backgroundColor: 'rgba(255,255,255,0.22)' },
  glass: {
    borderRadius: 18,
    borderWidth: 1,
    borderColor: 'rgba(255,255,255,0.12)',
    backgroundColor: 'rgba(146,168,232,0.07)',
    overflow: 'hidden',
  },
  pill: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 6,
    borderWidth: 1,
    borderRadius: 999,
    paddingVertical: 3,
    paddingHorizontal: 9,
    backgroundColor: 'rgba(255,255,255,0.03)',
  },
})
