// Liquid-glass примитивы: матовые полупрозрачные панели (expo-blur) поверх тёмной
// базы + терминальные акценты. Web-фолбэк BlurView даёт backdrop-filter.
import { BlurView } from 'expo-blur'
import { StyleSheet, View, type ViewStyle } from 'react-native'

import { C, F } from './theme'
import { T } from './ui'

// Glass — матовая панель (карточка/бар/шит)
export function Glass({ children, style, intensity = 24 }: { children?: React.ReactNode; style?: ViewStyle | ViewStyle[]; intensity?: number }) {
  return (
    <BlurView intensity={intensity} tint="dark" style={[g.glass, style]}>
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
  glass: {
    borderRadius: 16,
    borderWidth: 1,
    borderColor: 'rgba(255,255,255,0.07)',
    backgroundColor: 'rgba(120,140,200,0.04)',
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
