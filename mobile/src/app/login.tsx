import { useState } from 'react'
import { StyleSheet, TextInput } from 'react-native'
import Animated, { FadeInDown } from 'react-native-reanimated'

import { login, myWorkspaces } from '~/lib/auth'
import { setAuthed } from '~/lib/authState'
import { getActiveWs, loadActiveWs } from '~/lib/workspace'
import { loadMeta } from '~/lib/meta'
import { registerForPush } from '~/lib/push'
import { Backdrop, Glass } from '~/lib/glass'
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
    <Backdrop>
      <Animated.View entering={FadeInDown.duration(400)} style={styles.wrap}>
        <T w="b" s={34}>
          <Prompt />stdtask
        </T>
        <T v="faint" style={{ marginBottom: 22 }}>
          // sign in to your workspace
        </T>

        <Glass style={styles.field} intensity={26}>
          <T v="ok" w="m" s={15} style={styles.prompt}>
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
        </Glass>
        <Glass style={styles.field} intensity={26}>
          <T v="ok" w="m" s={15} style={styles.prompt}>
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
        </Glass>

        {err ? (
          <T v="danger" s={13}>
            {err}
          </T>
        ) : null}

        <Primary label="sign in" busy={busy} onPress={onSubmit} style={{ marginTop: 8 }} />
      </Animated.View>
    </Backdrop>
  )
}

const styles = StyleSheet.create({
  wrap: { flex: 1, justifyContent: 'center', paddingHorizontal: 26, gap: 12 },
  field: { flexDirection: 'row', alignItems: 'center', borderRadius: 14, paddingHorizontal: 14 },
  prompt: { marginRight: 8 },
  input: { flex: 1, color: C.text, paddingVertical: 15, fontSize: 15, fontFamily: F.r },
})
