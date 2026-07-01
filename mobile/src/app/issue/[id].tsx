import { useLocalSearchParams, useRouter } from 'expo-router'
import { useCallback, useEffect, useState } from 'react'
import { ActivityIndicator, FlatList, Modal, Pressable, StyleSheet, TextInput, View } from 'react-native'

import { addComment, assignIssue, getComments, getIssue, issueById, updateIssueState, type Comment, type Issue } from '~/lib/api'
import { allUsers, stateById, stateColor, statesForTeam, userById } from '~/lib/meta'
import { getActiveWs } from '~/lib/workspace'
import { C, F } from '~/lib/theme'
import { Bracket, Chip, Prompt, T } from '~/lib/ui'

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

  useEffect(() => {
    if (!issue && id) issueById(getActiveWs()!, id).then((i) => i && setIssue({ ...i }))
  }, [id])

  const loadComments = useCallback(async () => {
    if (!issue) return
    try {
      setComments(await getComments(ws, issue.id))
    } catch {
      /* пусто */
    }
  }, [issue?.id])

  useEffect(() => {
    loadComments()
  }, [loadComments])

  if (!issue) {
    return (
      <View style={styles.center}>
        <T v="faint">// loading issue…</T>
        <Bracket label="back" onPress={() => router.back()} />
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
        <Bracket label="‹ back" color={C.accent} onPress={() => router.back()} />
        {saving ? <ActivityIndicator color={C.accent} size="small" /> : null}
      </View>

      <FlatList
        data={comments}
        keyExtractor={(c) => c.id}
        keyboardShouldPersistTaps="handled"
        contentContainerStyle={{ paddingBottom: 16 }}
        ListHeaderComponent={
          <View style={{ gap: 14, marginBottom: 14 }}>
            <T v="accent" s={13}>
              {issue.identifier}
            </T>
            <T v="bright" w="b" s={21} style={{ lineHeight: 27 }}>
              {issue.title}
            </T>

            <View style={styles.chips}>
              <Chip onPress={() => setPicker('state')}>
                <View style={[styles.dot, { backgroundColor: stateColor(st?.colorToken) }]} />
                <T s={13}>{st?.name ?? 'status'}</T>
              </Chip>
              <Chip onPress={() => setPicker('assignee')}>
                <T s={13} v={asg ? 'body' : 'faint'}>
                  {asg ? asg.name : 'unassigned'}
                </T>
              </Chip>
            </View>

            {issue.description?.md ? (
              <T v="dim" s={14} style={{ lineHeight: 21 }}>
                {issue.description.md}
              </T>
            ) : (
              <T v="faint" s={13}>
                // no description
              </T>
            )}

            <T v="accent" s={13} style={{ marginTop: 8 }}>
              # comments
            </T>
          </View>
        }
        ListEmptyComponent={<T v="faint">// no comments</T>}
        renderItem={({ item }) => (
          <View style={styles.comment}>
            <T v="ok" s={12}>
              {userById(item.authorId)?.name ?? 'user'}
            </T>
            <T s={14} style={{ marginTop: 3 }}>
              {item.body}
            </T>
          </View>
        )}
      />

      <View style={styles.composer}>
        <TextInput
          style={styles.input}
          placeholder="> add a comment…"
          placeholderTextColor={C.tertiary}
          value={draft}
          onChangeText={setDraft}
          multiline
        />
        <Pressable style={[styles.send, (sending || !draft.trim()) && { opacity: 0.4 }]} disabled={sending || !draft.trim()} onPress={send}>
          <T w="b" s={13} style={{ color: C.base }}>
            {sending ? '…' : 'send'}
          </T>
        </Pressable>
      </View>

      <Modal visible={picker !== null} transparent animationType="fade" onRequestClose={() => setPicker(null)}>
        <Pressable style={styles.backdrop} onPress={() => setPicker(null)}>
          <View style={styles.sheet}>
            <T v="faint" s={12} style={{ marginBottom: 8 }}>
              # {picker === 'state' ? 'status' : 'assignee'}
            </T>
            {picker === 'state'
              ? statesForTeam(issue.teamId).map((s) => (
                  <Pressable key={s.id} style={styles.opt} onPress={() => pickState(s.id)}>
                    <View style={[styles.dot, { backgroundColor: stateColor(s.colorToken) }]} />
                    <T s={15}>{s.name}</T>
                  </Pressable>
                ))
              : [{ id: '', name: 'unassigned', email: '' }, ...allUsers()].map((u) => (
                  <Pressable key={u.id || 'none'} style={styles.opt} onPress={() => pickAssignee(u.id)}>
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
  wrap: { flex: 1, backgroundColor: C.base, paddingTop: 56, paddingHorizontal: 16 },
  topbar: { flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between', marginBottom: 10 },
  chips: { flexDirection: 'row', gap: 8 },
  dot: { width: 9, height: 9, borderRadius: 5 },
  comment: { paddingVertical: 9, borderBottomWidth: 1, borderBottomColor: C.subtle },
  composer: { flexDirection: 'row', alignItems: 'flex-end', gap: 8, paddingVertical: 8, borderTopWidth: 1, borderTopColor: C.subtle },
  input: { flex: 1, color: C.text, backgroundColor: C.elevated, borderRadius: 8, paddingHorizontal: 12, paddingVertical: 9, maxHeight: 100, fontSize: 15, fontFamily: F.r },
  send: { backgroundColor: C.accent, borderRadius: 8, paddingVertical: 10, paddingHorizontal: 16 },
  backdrop: { flex: 1, backgroundColor: '#000000bb', justifyContent: 'flex-end' },
  sheet: { backgroundColor: C.elevated, borderTopLeftRadius: 16, borderTopRightRadius: 16, borderWidth: 1, borderColor: C.subtle, padding: 16, maxHeight: '70%' },
  opt: { flexDirection: 'row', alignItems: 'center', gap: 8, paddingVertical: 12 },
  center: { flex: 1, backgroundColor: C.base, alignItems: 'center', justifyContent: 'center', gap: 14 },
})
