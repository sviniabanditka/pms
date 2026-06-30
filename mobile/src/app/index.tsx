import { useEffect, useState } from 'react'
import { ActivityIndicator, Pressable, ScrollView, StyleSheet, Text, View } from 'react-native'

import { logout, myWorkspaces, type Workspace } from '../../lib/auth'
import { setAuthed } from '../../lib/authState'
import { clients } from '../../lib/connect'

export default function Home() {
  const [loading, setLoading] = useState(true)
  const [ws, setWs] = useState<Workspace[]>([])
  // M0-проба: подтвердить что connect-RPC (Bearer + интерсептор) реально ходит на сервер
  const [rpcCheck, setRpcCheck] = useState('проверяю connect…')

  useEffect(() => {
    ;(async () => {
      const list = await myWorkspaces()
      setWs(list)
      setLoading(false)
      if (list[0]) {
        try {
          const res = await clients.query.search({ workspaceId: list[0].id, query: 'a' })
          setRpcCheck(`connect OK · search → ${res.results?.length ?? 0} рез.`)
        } catch (e: any) {
          setRpcCheck(`connect FAIL · ${e?.message ?? e}`)
        }
      } else {
        setRpcCheck('нет воркспейсов — connect не проверен')
      }
    })()
  }, [])

  async function onLogout() {
    await logout()
    setAuthed(false)
  }

  return (
    <View style={styles.wrap}>
      <Text style={styles.h1}>
        <Text style={{ color: '#9ece6a' }}>~/</Text>stdtask
      </Text>
      <Text style={styles.check}>{rpcCheck}</Text>

      <Text style={styles.label}># воркспейсы</Text>
      {loading ? (
        <ActivityIndicator color="#7aa2f7" style={{ marginTop: 12 }} />
      ) : ws.length === 0 ? (
        <Text style={styles.muted}>// нет воркспейсов</Text>
      ) : (
        <ScrollView style={{ marginTop: 4 }}>
          {ws.map((w) => (
            <View key={w.id} style={styles.row}>
              <Text style={styles.wsName}>
                <Text style={{ color: '#9ece6a' }}>~/</Text>
                {w.name}
              </Text>
              {w.slug ? <Text style={styles.muted}>{w.slug}</Text> : null}
            </View>
          ))}
        </ScrollView>
      )}

      <Pressable style={styles.btn} onPress={onLogout}>
        <Text style={styles.btnText}>[ выйти ]</Text>
      </Pressable>
    </View>
  )
}

const styles = StyleSheet.create({
  wrap: { flex: 1, backgroundColor: '#0b0c12', paddingHorizontal: 24, paddingTop: 72, gap: 8 },
  h1: { color: '#c0caf5', fontSize: 26, fontWeight: '700' },
  check: { color: '#565f89', fontSize: 13, marginBottom: 16 },
  label: { color: '#7aa2f7', fontSize: 13, marginTop: 8 },
  muted: { color: '#565f89', fontSize: 13 },
  row: { borderWidth: 1, borderColor: '#1f2335', backgroundColor: '#13151f', borderRadius: 8, padding: 14, marginBottom: 8 },
  wsName: { color: '#c0caf5', fontSize: 16, fontWeight: '600' },
  btn: { marginTop: 'auto', marginBottom: 32, borderWidth: 1, borderColor: '#1f2335', borderRadius: 8, paddingVertical: 14, alignItems: 'center' },
  btnText: { color: '#f7768e', fontWeight: '700' },
})
