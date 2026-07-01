import { useState } from 'react'
import { ActivityIndicator, Pressable, StyleSheet, Text, TextInput, View } from 'react-native'

import { login, myWorkspaces } from '~/lib/auth'
import { setAuthed } from '~/lib/authState'
import { loadActiveWs } from '~/lib/workspace'

export default function Login() {
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [busy, setBusy] = useState(false)
  const [err, setErr] = useState('')

  async function onSubmit() {
    if (!email || !password) return
    setBusy(true)
    setErr('')
    try {
      await login(email.trim(), password)
      await loadActiveWs(await myWorkspaces())
      setAuthed(true) // _layout уведёт на /
    } catch (e: any) {
      setErr(e?.message ?? 'Sign-in failed')
    } finally {
      setBusy(false)
    }
  }

  return (
    <View style={styles.wrap}>
      <Text style={styles.brand}>
        <Text style={{ color: '#9ece6a' }}>~/</Text>stdtask
      </Text>
      <Text style={styles.sub}>// sign in to your workspace</Text>

      <TextInput
        style={styles.input}
        placeholder="email"
        placeholderTextColor="#565f89"
        autoCapitalize="none"
        keyboardType="email-address"
        autoCorrect={false}
        value={email}
        onChangeText={setEmail}
      />
      <TextInput
        style={styles.input}
        placeholder="password"
        placeholderTextColor="#565f89"
        secureTextEntry
        value={password}
        onChangeText={setPassword}
        onSubmitEditing={onSubmit}
      />

      {err ? <Text style={styles.err}>{err}</Text> : null}

      <Pressable style={[styles.btn, busy && { opacity: 0.5 }]} disabled={busy} onPress={onSubmit}>
        {busy ? <ActivityIndicator color="#0b0c12" /> : <Text style={styles.btnText}>[ sign in ]</Text>}
      </Pressable>
    </View>
  )
}

const styles = StyleSheet.create({
  wrap: { flex: 1, justifyContent: 'center', paddingHorizontal: 28, backgroundColor: '#0b0c12', gap: 12 },
  brand: { color: '#c0caf5', fontSize: 30, fontWeight: '700' },
  sub: { color: '#565f89', marginBottom: 16 },
  input: {
    borderWidth: 1,
    borderColor: '#1f2335',
    backgroundColor: '#13151f',
    color: '#c0caf5',
    borderRadius: 8,
    paddingHorizontal: 14,
    paddingVertical: 12,
    fontSize: 16,
  },
  err: { color: '#f7768e', fontSize: 13 },
  btn: { backgroundColor: '#7aa2f7', borderRadius: 8, paddingVertical: 14, alignItems: 'center', marginTop: 4 },
  btnText: { color: '#0b0c12', fontWeight: '700', fontSize: 16 },
})
