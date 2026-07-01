import { ScrollView, StyleSheet, Text, TouchableOpacity, View } from 'react-native'

import { getCurrentUserId, logout } from '~/lib/auth'
import { setAuthed } from '~/lib/authState'
import { unregisterForPush } from '~/lib/push'
import { getWorkspaces, setActiveWs, useActiveWs } from '~/lib/workspace'

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
      <Text style={styles.h1}>Profile</Text>
      <Text style={styles.muted} numberOfLines={1}>
        user · {uid ?? '—'}
      </Text>

      <Text style={styles.label}># workspaces</Text>
      <ScrollView style={{ marginTop: 4 }}>
        {workspaces.map((w) => {
          const on = w.id === activeWs
          return (
            <TouchableOpacity key={w.id} style={[styles.row, on && styles.rowOn]} onPress={() => setActiveWs(w.id)}>
              <Text style={[styles.wsName, on && { color: '#7aa2f7' }]}>
                <Text style={{ color: '#9ece6a' }}>~/</Text>
                {w.name}
              </Text>
              {on ? <Text style={styles.active}>active</Text> : null}
            </TouchableOpacity>
          )
        })}
        {workspaces.length === 0 ? <Text style={styles.muted}>// no workspaces</Text> : null}
      </ScrollView>

      <TouchableOpacity style={styles.logout} onPress={onLogout}>
        <Text style={styles.logoutText}>[ sign out ]</Text>
      </TouchableOpacity>
    </View>
  )
}

const styles = StyleSheet.create({
  wrap: { flex: 1, backgroundColor: '#0b0c12', paddingTop: 64, paddingHorizontal: 20, gap: 6 },
  h1: { color: '#c0caf5', fontSize: 24, fontWeight: '700' },
  label: { color: '#7aa2f7', fontSize: 13, marginTop: 20 },
  muted: { color: '#565f89', fontSize: 13 },
  row: { flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between', borderWidth: 1, borderColor: '#1f2335', backgroundColor: '#13151f', borderRadius: 8, padding: 14, marginBottom: 8 },
  rowOn: { borderColor: '#7aa2f7' },
  wsName: { color: '#c0caf5', fontSize: 16, fontWeight: '600' },
  active: { color: '#7aa2f7', fontSize: 11 },
  logout: { marginTop: 'auto', marginBottom: 32, borderWidth: 1, borderColor: '#1f2335', borderRadius: 8, paddingVertical: 14, alignItems: 'center' },
  logoutText: { color: '#f7768e', fontWeight: '700' },
})
