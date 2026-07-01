import { BlurView } from 'expo-blur'
import { useRouter } from 'expo-router'
import { useState } from 'react'
import { Modal, Pressable, StyleSheet, TextInput, View } from 'react-native'
import Animated, { FadeInDown } from 'react-native-reanimated'
import { useSafeAreaInsets } from 'react-native-safe-area-context'

import { createIssue } from '~/lib/api'
import { Avatar, Backdrop, Glass } from '~/lib/glass'
import { allTeams, allUsers } from '~/lib/meta'
import { getActiveWs } from '~/lib/workspace'
import { C, F } from '~/lib/theme'
import { Primary, T } from '~/lib/ui'

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
  const insets = useSafeAreaInsets()
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
  const asg = allUsers().find((u) => u.id === assigneeId)

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
    <Backdrop>
      <View style={[styles.topbar, { marginTop: insets.top + 12 }]}>
        <Pressable onPress={() => router.back()} hitSlop={10}>
          <T v="accent" s={14} w="m">
            [ cancel ]
          </T>
        </Pressable>
        <T v="faint" s={13}>
          # new issue
        </T>
        <View style={{ width: 62 }} />
      </View>

      <Animated.View entering={FadeInDown.duration(340)} style={styles.body}>
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
          <Pressable onPress={() => setPicker('team')}>
            <Glass style={styles.chip} intensity={22}>
              <T s={13} v="accent" w="m">
                {teamKey}
              </T>
            </Glass>
          </Pressable>
          <Pressable onPress={() => setPicker('priority')}>
            <Glass style={styles.chip} intensity={22}>
              <T s={13} v={priority ? 'body' : 'faint'}>
                {priName}
              </T>
            </Glass>
          </Pressable>
          <Pressable onPress={() => setPicker('assignee')}>
            <Glass style={styles.chip} intensity={22}>
              {asg ? <Avatar name={asg.name} id={asg.id} size={18} /> : null}
              <T s={13} v={asg ? 'body' : 'faint'}>
                {asg ? asg.name : 'unassigned'}
              </T>
            </Glass>
          </Pressable>
        </View>

        {err ? (
          <T v="danger" s={13}>
            {err}
          </T>
        ) : null}
      </Animated.View>

      <Primary label="create issue" busy={busy} onPress={submit} style={[styles.create, { marginBottom: insets.bottom + 20 }]} />

      <Modal visible={picker !== null} transparent animationType="fade" onRequestClose={() => setPicker(null)}>
        <Pressable style={styles.backdrop} onPress={() => setPicker(null)}>
          <BlurView intensity={30} tint="dark" style={styles.sheet}>
            {picker === 'team' &&
              teams.map((t) => (
                <Pressable key={t.id} style={styles.opt} onPress={() => { setTeamId(t.id); setPicker(null) }}>
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
                <Pressable key={p.v} style={styles.opt} onPress={() => { setPriority(p.v); setPicker(null) }}>
                  <T s={15}>{p.label}</T>
                </Pressable>
              ))}
            {picker === 'assignee' &&
              [{ id: '', name: 'unassigned' }, ...allUsers()].map((u) => (
                <Pressable key={u.id || 'none'} style={styles.opt} onPress={() => { setAssigneeId(u.id); setPicker(null) }}>
                  {u.id ? <Avatar name={u.name} id={u.id} size={22} /> : <View style={{ width: 22 }} />}
                  <T s={15}>{u.name}</T>
                </Pressable>
              ))}
          </BlurView>
        </Pressable>
      </Modal>
    </Backdrop>
  )
}

const styles = StyleSheet.create({
  topbar: { flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between', paddingHorizontal: 16 },
  body: { paddingHorizontal: 18, paddingTop: 20, gap: 18, flex: 1 },
  titleInput: { color: C.bright, fontSize: 20, minHeight: 60, fontFamily: F.m, lineHeight: 26 },
  chips: { flexDirection: 'row', gap: 8, flexWrap: 'wrap' },
  chip: { flexDirection: 'row', alignItems: 'center', gap: 6, borderRadius: 10, paddingVertical: 8, paddingHorizontal: 12 },
  create: { marginHorizontal: 18 },
  backdrop: { flex: 1, backgroundColor: 'rgba(4,5,10,0.55)', justifyContent: 'flex-end' },
  sheet: { borderTopLeftRadius: 20, borderTopRightRadius: 20, borderWidth: 1, borderColor: 'rgba(255,255,255,0.1)', padding: 18, paddingBottom: 34, maxHeight: '70%', overflow: 'hidden' },
  opt: { flexDirection: 'row', alignItems: 'center', gap: 10, paddingVertical: 12 },
})
