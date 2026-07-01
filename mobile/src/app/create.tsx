import { useRouter } from 'expo-router'
import { useState } from 'react'
import { ActivityIndicator, Modal, Pressable, StyleSheet, Text, TextInput, View } from 'react-native'

import { createIssue } from '~/lib/api'
import { allTeams, allUsers } from '~/lib/meta'
import { getActiveWs } from '~/lib/workspace'

const PRIORITIES = [
  { v: 0, label: 'none' },
  { v: 1, label: 'urgent' },
  { v: 2, label: 'high' },
  { v: 3, label: 'medium' },
  { v: 4, label: 'low' },
]
type Picker = null | 'team' | 'priority' | 'assignee'

export default function Create() {
  const router = useRouter()
  const ws = getActiveWs()!
  const teams = allTeams()
  const [title, setTitle] = useState('')
  const [teamId, setTeamId] = useState(teams[0]?.id ?? '')
  const [priority, setPriority] = useState(0)
  const [assigneeId, setAssigneeId] = useState('')
  const [picker, setPicker] = useState<Picker>(null)
  const [busy, setBusy] = useState(false)
  const [err, setErr] = useState('')

  const teamName = teams.find((t) => t.id === teamId)?.key ?? 'team'
  const priName = PRIORITIES.find((p) => p.v === priority)!.label
  const asgName = allUsers().find((u) => u.id === assigneeId)?.name ?? 'unassigned'

  async function submit() {
    if (!title.trim() || !teamId) {
      setErr('title and team required')
      return
    }
    setBusy(true)
    setErr('')
    try {
      await createIssue(ws, { teamId, title: title.trim(), priority, assigneeId: assigneeId || undefined })
      router.back()
    } catch (e: any) {
      setErr(e?.message ?? 'Create failed')
      setBusy(false)
    }
  }

  return (
    <View style={styles.wrap}>
      <View style={styles.topbar}>
        <Pressable onPress={() => router.back()}>
          <Text style={styles.back}>‹ cancel</Text>
        </Pressable>
        <Text style={styles.h1}>new issue</Text>
        <View style={{ width: 50 }} />
      </View>

      <TextInput
        style={styles.titleInput}
        placeholder="Issue title"
        placeholderTextColor="#565f89"
        value={title}
        onChangeText={setTitle}
        autoFocus
        multiline
      />

      <View style={styles.chips}>
        <Pressable style={styles.chip} onPress={() => setPicker('team')}>
          <Text style={styles.chipText}>{teamName}</Text>
        </Pressable>
        <Pressable style={styles.chip} onPress={() => setPicker('priority')}>
          <Text style={styles.chipText}>{priName}</Text>
        </Pressable>
        <Pressable style={styles.chip} onPress={() => setPicker('assignee')}>
          <Text style={styles.chipText}>{asgName}</Text>
        </Pressable>
      </View>

      {err ? <Text style={styles.err}>{err}</Text> : null}

      <Pressable style={[styles.create, busy && { opacity: 0.5 }]} disabled={busy} onPress={submit}>
        {busy ? <ActivityIndicator color="#0b0c12" /> : <Text style={styles.createText}>[ create ]</Text>}
      </Pressable>

      <Modal visible={picker !== null} transparent animationType="fade" onRequestClose={() => setPicker(null)}>
        <Pressable style={styles.backdrop} onPress={() => setPicker(null)}>
          <View style={styles.sheet}>
            {picker === 'team' &&
              teams.map((t) => (
                <Pressable key={t.id} style={styles.opt} onPress={() => { setTeamId(t.id); setPicker(null) }}>
                  <Text style={styles.optText}>{t.key} · {t.name}</Text>
                </Pressable>
              ))}
            {picker === 'priority' &&
              PRIORITIES.map((p) => (
                <Pressable key={p.v} style={styles.opt} onPress={() => { setPriority(p.v); setPicker(null) }}>
                  <Text style={styles.optText}>{p.label}</Text>
                </Pressable>
              ))}
            {picker === 'assignee' &&
              [{ id: '', name: 'unassigned' }, ...allUsers()].map((u) => (
                <Pressable key={u.id || 'none'} style={styles.opt} onPress={() => { setAssigneeId(u.id); setPicker(null) }}>
                  <Text style={styles.optText}>{u.name}</Text>
                </Pressable>
              ))}
          </View>
        </Pressable>
      </Modal>
    </View>
  )
}

const styles = StyleSheet.create({
  wrap: { flex: 1, backgroundColor: '#0b0c12', paddingTop: 56, paddingHorizontal: 16, gap: 16 },
  topbar: { flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between' },
  back: { color: '#7aa2f7', fontSize: 15, width: 60 },
  h1: { color: '#c0caf5', fontSize: 16, fontWeight: '700' },
  titleInput: { color: '#c0caf5', fontSize: 18, minHeight: 60 },
  chips: { flexDirection: 'row', gap: 8, flexWrap: 'wrap' },
  chip: { borderWidth: 1, borderColor: '#1f2335', borderRadius: 6, paddingVertical: 8, paddingHorizontal: 12 },
  chipText: { color: '#c0caf5', fontSize: 13 },
  err: { color: '#f7768e' },
  create: { backgroundColor: '#7aa2f7', borderRadius: 8, paddingVertical: 14, alignItems: 'center', marginTop: 'auto', marginBottom: 32 },
  createText: { color: '#0b0c12', fontWeight: '700', fontSize: 16 },
  backdrop: { flex: 1, backgroundColor: '#000000aa', justifyContent: 'flex-end' },
  sheet: { backgroundColor: '#13151f', borderTopLeftRadius: 16, borderTopRightRadius: 16, padding: 16, maxHeight: '70%' },
  opt: { paddingVertical: 12 },
  optText: { color: '#c0caf5', fontSize: 15 },
})
