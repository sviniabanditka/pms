// Мини UI-кит в терминальном стиле десктопа. Всё моно (JetBrains Mono).
import { ActivityIndicator, Pressable, StyleSheet, Text, type TextProps, View } from 'react-native'
import { C, F } from './theme'

type TVariant = 'body' | 'dim' | 'faint' | 'bright' | 'accent' | 'ok' | 'danger' | 'warn' | 'info'
const COLOR: Record<TVariant, string> = {
  body: C.text,
  dim: C.secondary,
  faint: C.tertiary,
  bright: C.bright,
  accent: C.accent,
  ok: C.ok,
  danger: C.danger,
  warn: C.warn,
  info: C.info,
}

// T — моно-текст. v=цвет-вариант, w=вес (r|m|sb|b), s=размер.
export function T({ v = 'body', w = 'r', s, style, ...p }: TextProps & { v?: TVariant; w?: keyof typeof F; s?: number }) {
  return <Text {...p} style={[{ fontFamily: F[w], color: COLOR[v], fontSize: s ?? 14 }, style]} />
}

// Bracket — терминальная кнопка [ label ]
export function Bracket({ label, onPress, color = C.accent, disabled }: { label: string; onPress: () => void; color?: string; disabled?: boolean }) {
  return (
    <Pressable onPress={onPress} disabled={disabled} style={({ pressed }) => [{ opacity: disabled ? 0.4 : pressed ? 0.6 : 1 }]} hitSlop={8}>
      <Text style={{ fontFamily: F.m, color, fontSize: 13 }}>[ {label} ]</Text>
    </Pressable>
  )
}

// Primary — заполненная акцентная CTA (login/create/send)
export function Primary({ label, onPress, busy, disabled, style }: { label: string; onPress: () => void; busy?: boolean; disabled?: boolean; style?: any }) {
  return (
    <Pressable onPress={onPress} disabled={disabled || busy} style={[ui.primary, (disabled || busy) && { opacity: 0.5 }, style]}>
      {busy ? <ActivityIndicator color={C.base} /> : <Text style={ui.primaryText}>{label}</Text>}
    </Pressable>
  )
}

// Chip — обводка-«кнопка» (статус/ассайни/фильтр)
export function Chip({ children, onPress, active }: { children: React.ReactNode; onPress?: () => void; active?: boolean }) {
  return (
    <Pressable onPress={onPress} style={[ui.chip, active && { borderColor: C.accent, backgroundColor: C.active }]}>
      {children}
    </Pressable>
  )
}

// Prompt — «~/», «#», «$» цветной префикс
export function Prompt({ char = '~/', color = C.ok }: { char?: string; color?: string }) {
  return <Text style={{ fontFamily: F.m, color }}>{char}</Text>
}

export function Spinner() {
  return (
    <View style={{ marginTop: 44 }}>
      <ActivityIndicator color={C.accent} />
    </View>
  )
}

const ui = StyleSheet.create({
  primary: { backgroundColor: C.accent, borderRadius: 8, paddingVertical: 13, alignItems: 'center' },
  primaryText: { fontFamily: F.b, color: C.base, fontSize: 15 },
  chip: { flexDirection: 'row', alignItems: 'center', gap: 6, borderWidth: 1, borderColor: C.subtle, borderRadius: 6, paddingVertical: 7, paddingHorizontal: 11, backgroundColor: C.elevated },
})
