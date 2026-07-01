import { useState } from 'react'
import { StyleSheet, TextInput, View } from 'react-native'

import { login, myWorkspaces } from '~/lib/auth'
import { setAuthed } from '~/lib/authState'
import { getActiveWs, loadActiveWs } from '~/lib/workspace'
import { loadMeta } from '~/lib/meta'
import { registerForPush } from '~/lib/push'
import { C, F } from '~/lib/theme'
import { Primary, Prompt, T } from '~/lib/ui'

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
      const ws = getActiveWs()
      if (ws) await loadMeta(ws)
      registerForPush()
      setAuthed(true)
    } catch (e: any) {
      setErr(e?.message ?? 'Sign-in failed')
      setBusy(false)
    }
  }

  return (
    <View style={styles.wrap}>
      <T w="b" s={30}>
        <Prompt />stdtask
      </T>
      <T v="faint" style={{ marginBottom: 20 }}>
        // sign in to your workspace
      </T>

      <View style={styles.field}>
        <T v="ok" w="m" style={styles.dollar}>
          {'>'}
        </T>
        <TextInput
          style={styles.input}
          placeholder="email"
          placeholderTextColor={C.tertiary}
          autoCapitalize="none"
          keyboardType="email-address"
          autoCorrect={false}
          value={email}
          onChangeText={setEmail}
        />
      </View>
      <View style={styles.field}>
        <T v="ok" w="m" style={styles.dollar}>
          {'>'}
        </T>
        <TextInput
          style={styles.input}
          placeholder="password"
          placeholderTextColor={C.tertiary}
          secureTextEntry
          value={password}
          onChangeText={setPassword}
          onSubmitEditing={onSubmit}
        />
      </View>

      {err ? (
        <T v="danger" s={13}>
          {err}
        </T>
      ) : null}

      <Primary label="sign in" busy={busy} onPress={onSubmit} style={{ marginTop: 6 }} />
    </View>
  )
}

const styles = StyleSheet.create({
  wrap: { flex: 1, justifyContent: 'center', paddingHorizontal: 28, backgroundColor: C.base, gap: 12 },
  field: { flexDirection: 'row', alignItems: 'center', borderWidth: 1, borderColor: C.subtle, backgroundColor: C.elevated, borderRadius: 8, paddingHorizontal: 12 },
  dollar: { marginRight: 8 },
  input: { flex: 1, color: C.text, paddingVertical: 12, fontSize: 15, fontFamily: F.r },
})
