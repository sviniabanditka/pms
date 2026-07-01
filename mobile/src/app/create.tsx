import { useRouter } from 'expo-router'
import { useState } from 'react'
import { Modal, Pressable, StyleSheet, TextInput, View } from 'react-native'

import { createIssue } from '~/lib/api'
import { allTeams, allUsers } from '~/lib/meta'
import { getActiveWs } from '~/lib/workspace'
import { C, F } from '~/lib/theme'
import { Chip, Primary, T } from '~/lib/ui'

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

  const teamKey = teams.find((t) => t.id === teamId)?.key ?? 'team'
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
          <T v="accent" s={13} w="m">
            [ cancel ]
          </T>
        </Pressable>
        <T v="faint" s={13}>
          # new issue
        </T>
        <View style={{ width: 60 }} />
      </View>

      <TextInput
        style={styles.titleInput}
        placeholder="Issue title"
        placeholderTextColor={C.tertiary}
        value={title}
        onChangeText={setTitle}
        autoFocus
        multiline
      />

      <View style={styles.chips}>
        <Chip onPress={() => setPicker('team')}>
          <T s={13}>{teamKey}</T>
        </Chip>
        <Chip onPress={() => setPicker('priority')}>
          <T s={13} v={priority ? 'body' : 'faint'}>
            {priName}
          </T>
        </Chip>
        <Chip onPress={() => setPicker('assignee')}>
          <T s={13} v={assigneeId ? 'body' : 'faint'}>
            {asgName}
          </T>
        </Chip>
      </View>

      {err ? (
        <T v="danger" s={13}>
          {err}
        </T>
      ) : null}

      <Primary label="create" busy={busy} onPress={submit} style={styles.create} />

      <Modal visible={picker !== null} transparent animationType="fade" onRequestClose={() => setPicker(null)}>
        <Pressable style={styles.backdrop} onPress={() => setPicker(null)}>
          <View style={styles.sheet}>
            {picker === 'team' &&
              teams.map((t) => (
                <Pressable
                  key={t.id}
                  style={styles.opt}
                  onPress={() => {
                    setTeamId(t.id)
                    setPicker(null)
                  }}
                >
                  <T s={15}>
                    <T v="accent" s={15}>
                      {t.key}
                    </T>{' '}
                    · {t.name}
                  </T>
                </Pressable>
              ))}
            {picker === 'priority' &&
              PRIORITIES.map((p) => (
                <Pressable
                  key={p.v}
                  style={styles.opt}
                  onPress={() => {
                    setPriority(p.v)
                    setPicker(null)
                  }}
                >
                  <T s={15}>{p.label}</T>
                </Pressable>
              ))}
            {picker === 'assignee' &&
              [{ id: '', name: 'unassigned' }, ...allUsers()].map((u) => (
                <Pressable
                  key={u.id || 'none'}
                  style={styles.opt}
                  onPress={() => {
                    setAssigneeId(u.id)
                    setPicker(null)
                  }}
                >
                  <T s={15}>{u.name}</T>
                </Pressable>
              ))}
          </View>
        </Pressable>
      </Modal>
    </View>
  )
}

const styles = StyleSheet.create({
  wrap: { flex: 1, backgroundColor: C.base, paddingTop: 56, paddingHorizontal: 16, gap: 16 },
  topbar: { flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between' },
  titleInput: { color: C.bright, fontSize: 18, minHeight: 60, fontFamily: F.m, lineHeight: 24 },
  chips: { flexDirection: 'row', gap: 8, flexWrap: 'wrap' },
  create: { marginTop: 'auto', marginBottom: 32 },
  backdrop: { flex: 1, backgroundColor: '#000000bb', justifyContent: 'flex-end' },
  sheet: { backgroundColor: C.elevated, borderTopLeftRadius: 16, borderTopRightRadius: 16, borderWidth: 1, borderColor: C.subtle, padding: 16, maxHeight: '70%' },
  opt: { paddingVertical: 12 },
})
