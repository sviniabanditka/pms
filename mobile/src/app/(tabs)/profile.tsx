import { ScrollView, StyleSheet, TouchableOpacity, View } from 'react-native'

import { getCurrentUserId, logout } from '~/lib/auth'
import { setAuthed } from '~/lib/authState'
import { unregisterForPush } from '~/lib/push'
import { getWorkspaces, setActiveWs, useActiveWs } from '~/lib/workspace'
import { C } from '~/lib/theme'
import { Bracket, Prompt, T } from '~/lib/ui'

export default function Profile() {
  const activeWs = useActiveWs()
  const workspaces = getWorkspaces()
  const uid = getCurrentUserId()

  async function onLogout() {
    await unregisterForPush()
    await logout()
    setAuthed(false)
  }

  return (
    <View style={styles.wrap}>
      <T w="b" s={20}>
        <Prompt char="#" color={C.accent} /> profile
      </T>
      <T v="faint" s={12} numberOfLines={1} style={{ marginTop: 4 }}>
        user · {uid ?? '—'}
      </T>

      <T v="accent" s={13} style={{ marginTop: 24, marginBottom: 6 }}>
        # workspaces
      </T>
      <ScrollView>
        {workspaces.map((w) => {
          const on = w.id === activeWs
          return (
            <TouchableOpacity key={w.id} style={[styles.row, on && styles.rowOn]} onPress={() => setActiveWs(w.id)}>
              <T v={on ? 'accent' : 'body'} w="sb" s={15}>
                <Prompt />
                {w.name}
              </T>
              {on ? <T v="accent" s={11}>active</T> : null}
            </TouchableOpacity>
          )
        })}
        {workspaces.length === 0 ? <T v="faint">// no workspaces</T> : null}
      </ScrollView>

      <View style={styles.footer}>
        <Bracket label="sign out" color={C.danger} onPress={onLogout} />
      </View>
    </View>
  )
}

const styles = StyleSheet.create({
  wrap: { flex: 1, backgroundColor: C.base, paddingTop: 64, paddingHorizontal: 20 },
  row: { flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between', borderWidth: 1, borderColor: C.subtle, backgroundColor: C.elevated, borderRadius: 8, padding: 14, marginBottom: 8 },
  rowOn: { borderColor: C.accent },
  footer: { paddingVertical: 24, alignItems: 'flex-start' },
})
