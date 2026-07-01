import { useLocalSearchParams, useRouter } from 'expo-router'
import { useCallback, useEffect, useState } from 'react'
import { ActivityIndicator, FlatList, Modal, Pressable, StyleSheet, Text, TextInput, View } from 'react-native'

import { addComment, assignIssue, getComments, getIssue, updateIssueState, type Comment, type Issue } from '~/lib/api'
import { allUsers, stateById, stateColor, statesForTeam, userById } from '~/lib/meta'
import { getActiveWs } from '~/lib/workspace'

type Picker = null | 'state' | 'assignee'

export default function IssueDetail() {
  const { id } = useLocalSearchParams<{ id: string }>()
  const router = useRouter()
  const ws = getActiveWs()!
  const [issue, setIssue] = useState<Issue | undefined>(() => getIssue(id))
  const [picker, setPicker] = useState<Picker>(null)
  const [saving, setSaving] = useState(false)
  const [comments, setComments] = useState<Comment[]>([])
  const [draft, setDraft] = useState('')
  const [sending, setSending] = useState(false)

  const refresh = () => setIssue(getIssue(id) ? { ...getIssue(id)! } : undefined)

  const loadComments = useCallback(async () => {
    if (!issue) return
    try {
      setComments(await getComments(ws, issue.id))
    } catch {
      /* пусто при ошибке */
    }
  }, [issue?.id])

  useEffect(() => {
    loadComments()
  }, [loadComments])

  if (!issue) {
    return (
      <View style={styles.center}>
        <Text style={styles.muted}>// issue not loaded — open it from the list</Text>
        <Pressable onPress={() => router.back()} style={styles.retry}>
          <Text style={styles.retryText}>[ back ]</Text>
        </Pressable>
      </View>
    )
  }

  const st = stateById(issue.stateId)
  const asg = userById(issue.assigneeId)

  async function pickState(stateId: string) {
    setPicker(null)
    setSaving(true)
    try {
      await updateIssueState(ws, issue!, stateId)
      refresh()
    } finally {
      setSaving(false)
    }
  }
  async function pickAssignee(uid: string) {
    setPicker(null)
    setSaving(true)
    try {
      await assignIssue(ws, issue!, uid)
      refresh()
    } finally {
      setSaving(false)
    }
  }
  async function send() {
    if (!draft.trim()) return
    setSending(true)
    try {
      await addComment(ws, issue!.id, draft.trim())
      setDraft('')
      await loadComments()
    } finally {
      setSending(false)
    }
  }

  return (
    <View style={styles.wrap}>
      <View style={styles.topbar}>
        <Pressable onPress={() => router.back()}>
          <Text style={styles.back}>‹ back</Text>
        </Pressable>
        {saving ? <ActivityIndicator color="#7aa2f7" size="small" /> : null}
      </View>

      <FlatList
        data={comments}
        keyExtractor={(c) => c.id}
        keyboardShouldPersistTaps="handled"
        ListHeaderComponent={
          <View style={{ gap: 12, marginBottom: 12 }}>
            <Text style={styles.ident}>{issue.identifier}</Text>
            <Text style={styles.title}>{issue.title}</Text>

            <View style={styles.chips}>
              <Pressable style={styles.chip} onPress={() => setPicker('state')}>
                <View style={[styles.dot, { backgroundColor: stateColor(st?.colorToken) }]} />
                <Text style={styles.chipText}>{st?.name ?? 'status'}</Text>
              </Pressable>
              <Pressable style={styles.chip} onPress={() => setPicker('assignee')}>
                <Text style={styles.chipText}>{asg ? asg.name : 'unassigned'}</Text>
              </Pressable>
            </View>

            {issue.description?.md ? <Text style={styles.desc}>{issue.description.md}</Text> : <Text style={styles.muted}>// no description</Text>}

            <Text style={styles.section}># comments</Text>
          </View>
        }
        ListEmptyComponent={<Text style={styles.muted}>// no comments</Text>}
        renderItem={({ item }) => (
          <View style={styles.comment}>
            <Text style={styles.author}>{userById(item.authorId)?.name ?? 'user'}</Text>
            <Text style={styles.body}>{item.body}</Text>
          </View>
        )}
        contentContainerStyle={{ paddingBottom: 16 }}
      />

      <View style={styles.composer}>
        <TextInput
          style={styles.input}
          placeholder="add a comment…"
          placeholderTextColor="#565f89"
          value={draft}
          onChangeText={setDraft}
          multiline
        />
        <Pressable style={[styles.send, (sending || !draft.trim()) && { opacity: 0.4 }]} disabled={sending || !draft.trim()} onPress={send}>
          <Text style={styles.sendText}>{sending ? '…' : 'send'}</Text>
        </Pressable>
      </View>

      <Modal visible={picker !== null} transparent animationType="fade" onRequestClose={() => setPicker(null)}>
        <Pressable style={styles.backdrop} onPress={() => setPicker(null)}>
          <View style={styles.sheet}>
            <Text style={styles.sheetTitle}>{picker === 'state' ? 'status' : 'assignee'}</Text>
            {picker === 'state'
              ? statesForTeam(issue.teamId).map((s) => (
                  <Pressable key={s.id} style={styles.opt} onPress={() => pickState(s.id)}>
                    <View style={[styles.dot, { backgroundColor: stateColor(s.colorToken) }]} />
                    <Text style={styles.optText}>{s.name}</Text>
                  </Pressable>
                ))
              : [{ id: '', name: 'unassigned', email: '' }, ...allUsers()].map((u) => (
                  <Pressable key={u.id || 'none'} style={styles.opt} onPress={() => pickAssignee(u.id)}>
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
  wrap: { flex: 1, backgroundColor: '#0b0c12', paddingTop: 56, paddingHorizontal: 16 },
  topbar: { flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between', marginBottom: 8 },
  back: { color: '#7aa2f7', fontSize: 15 },
  ident: { color: '#7aa2f7', fontSize: 13 },
  title: { color: '#c0caf5', fontSize: 20, fontWeight: '700' },
  chips: { flexDirection: 'row', gap: 8 },
  chip: { flexDirection: 'row', alignItems: 'center', gap: 6, borderWidth: 1, borderColor: '#1f2335', borderRadius: 6, paddingVertical: 6, paddingHorizontal: 10 },
  chipText: { color: '#c0caf5', fontSize: 13 },
  dot: { width: 9, height: 9, borderRadius: 5 },
  desc: { color: '#a9b1d6', fontSize: 14, lineHeight: 20 },
  section: { color: '#7aa2f7', fontSize: 13, marginTop: 8 },
  comment: { paddingVertical: 8, borderBottomWidth: 1, borderBottomColor: '#13151f', gap: 3 },
  author: { color: '#9ece6a', fontSize: 12 },
  body: { color: '#c0caf5', fontSize: 14 },
  composer: { flexDirection: 'row', alignItems: 'flex-end', gap: 8, paddingVertical: 8, borderTopWidth: 1, borderTopColor: '#1f2335' },
  input: { flex: 1, color: '#c0caf5', backgroundColor: '#13151f', borderRadius: 8, paddingHorizontal: 12, paddingVertical: 8, maxHeight: 100, fontSize: 15 },
  send: { backgroundColor: '#7aa2f7', borderRadius: 8, paddingVertical: 10, paddingHorizontal: 14 },
  sendText: { color: '#0b0c12', fontWeight: '700' },
  backdrop: { flex: 1, backgroundColor: '#000000aa', justifyContent: 'flex-end' },
  sheet: { backgroundColor: '#13151f', borderTopLeftRadius: 16, borderTopRightRadius: 16, padding: 16, gap: 4, maxHeight: '70%' },
  sheetTitle: { color: '#565f89', fontSize: 13, marginBottom: 8 },
  opt: { flexDirection: 'row', alignItems: 'center', gap: 8, paddingVertical: 12 },
  optText: { color: '#c0caf5', fontSize: 15 },
  center: { flex: 1, backgroundColor: '#0b0c12', alignItems: 'center', justifyContent: 'center', gap: 12 },
  muted: { color: '#565f89', fontSize: 13 },
  retry: { borderWidth: 1, borderColor: '#1f2335', borderRadius: 6, paddingVertical: 8, paddingHorizontal: 16 },
  retryText: { color: '#7aa2f7' },
})
