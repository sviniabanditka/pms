import { BlurView } from 'expo-blur'
import { useLocalSearchParams, useRouter } from 'expo-router'
import { useCallback, useEffect, useState } from 'react'
import { ActivityIndicator, FlatList, Modal, Pressable, StyleSheet, TextInput, View } from 'react-native'
import Animated, { FadeInDown } from 'react-native-reanimated'
import { useSafeAreaInsets } from 'react-native-safe-area-context'

import { addComment, assignIssue, getComments, getIssue, issueById, updateIssueState, type Comment, type Issue } from '~/lib/api'
import { Avatar, Backdrop, Glass, GlowDot } from '~/lib/glass'
import { allUsers, stateById, stateColor, statesForTeam, userById } from '~/lib/meta'
import { getActiveWs } from '~/lib/workspace'
import { C, F } from '~/lib/theme'
import { Chip, T } from '~/lib/ui'

type Picker = null | 'state' | 'assignee'

export default function IssueDetail() {
  const { id } = useLocalSearchParams<{ id: string }>()
  const router = useRouter()
  const insets = useSafeAreaInsets()
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
      <Backdrop>
        <View style={styles.center}>
          <ActivityIndicator color={C.accent} />
          <T v="faint">// loading issue…</T>
        </View>
      </Backdrop>
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
    <Backdrop>
      <Glass style={[styles.topbar, { marginTop: insets.top + 10 }]} intensity={34}>
        <Pressable onPress={() => router.back()} hitSlop={10}>
          <T v="accent" s={14} w="m">
            ‹ back
          </T>
        </Pressable>
        <T v="faint" s={12}>
          {issue.identifier}
        </T>
        {saving ? <ActivityIndicator color={C.accent} size="small" /> : <View style={{ width: 16 }} />}
      </Glass>

      <FlatList
        data={comments}
        keyExtractor={(c) => c.id}
        keyboardShouldPersistTaps="handled"
        contentContainerStyle={{ paddingHorizontal: 16, paddingBottom: 16 }}
        showsVerticalScrollIndicator={false}
        ListHeaderComponent={
          <Animated.View entering={FadeInDown.duration(320)} style={{ gap: 15, paddingTop: 14, marginBottom: 16 }}>
            <T v="bright" w="b" s={22} style={{ lineHeight: 29 }}>
              {issue.title}
            </T>

            <View style={styles.chips}>
              <Chip onPress={() => setPicker('state')}>
                <GlowDot color={stateColor(st?.colorToken)} size={8} />
                <T s={13}>{st?.name ?? 'status'}</T>
              </Chip>
              <Chip onPress={() => setPicker('assignee')}>
                {asg ? <Avatar name={asg.name} id={asg.id} size={18} /> : null}
                <T s={13} v={asg ? 'body' : 'faint'}>
                  {asg ? asg.name : 'unassigned'}
                </T>
              </Chip>
            </View>

            <Glass style={styles.descBox} intensity={16}>
              {issue.description?.md ? (
                <T v="dim" s={14} style={{ lineHeight: 21 }}>
                  {issue.description.md}
                </T>
              ) : (
                <T v="faint" s={13}>
                  // no description
                </T>
              )}
            </Glass>

            <T v="accent" s={13}>
              # comments
            </T>
          </Animated.View>
        }
        ListEmptyComponent={<T v="faint">// no comments</T>}
        renderItem={({ item, index }) => (
          <Animated.View entering={FadeInDown.duration(220).delay(Math.min(index, 10) * 24)} style={styles.comment}>
            <Avatar name={userById(item.authorId)?.name} id={item.authorId} size={26} />
            <View style={{ flex: 1 }}>
              <T v="ok" s={12}>
                {userById(item.authorId)?.name ?? 'user'}
              </T>
              <T s={14} style={{ marginTop: 3, lineHeight: 20 }}>
                {item.body}
              </T>
            </View>
          </Animated.View>
        )}
      />

      <Glass style={[styles.composer, { paddingBottom: insets.bottom + 8 }]} intensity={40}>
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
      </Glass>

      <Modal visible={picker !== null} transparent animationType="fade" onRequestClose={() => setPicker(null)}>
        <Pressable style={styles.backdrop} onPress={() => setPicker(null)}>
          <BlurView intensity={30} tint="dark" style={styles.sheet}>
            <T v="faint" s={12} style={{ marginBottom: 8 }}>
              # {picker === 'state' ? 'status' : 'assignee'}
            </T>
            {picker === 'state'
              ? statesForTeam(issue.teamId).map((s) => (
                  <Pressable key={s.id} style={styles.opt} onPress={() => pickState(s.id)}>
                    <GlowDot color={stateColor(s.colorToken)} size={9} />
                    <T s={15}>{s.name}</T>
                  </Pressable>
                ))
              : [{ id: '', name: 'unassigned', email: '' }, ...allUsers()].map((u) => (
                  <Pressable key={u.id || 'none'} style={styles.opt} onPress={() => pickAssignee(u.id)}>
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
  topbar: { flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between', marginHorizontal: 12, paddingHorizontal: 15, paddingVertical: 11, borderRadius: 16 },
  chips: { flexDirection: 'row', gap: 8, flexWrap: 'wrap' },
  descBox: { borderRadius: 14, padding: 14 },
  comment: { flexDirection: 'row', gap: 10, paddingVertical: 10, borderBottomWidth: 1, borderBottomColor: 'rgba(255,255,255,0.05)' },
  composer: { flexDirection: 'row', alignItems: 'flex-end', gap: 8, paddingHorizontal: 12, paddingTop: 10, marginHorizontal: 0, borderTopWidth: 1, borderTopColor: 'rgba(255,255,255,0.07)', borderRadius: 0 },
  input: { flex: 1, color: C.text, backgroundColor: 'rgba(255,255,255,0.04)', borderRadius: 12, paddingHorizontal: 13, paddingVertical: 10, maxHeight: 100, fontSize: 15, fontFamily: F.r },
  send: { backgroundColor: C.accent, borderRadius: 12, paddingVertical: 11, paddingHorizontal: 16 },
  backdrop: { flex: 1, backgroundColor: 'rgba(4,5,10,0.55)', justifyContent: 'flex-end' },
  sheet: { borderTopLeftRadius: 20, borderTopRightRadius: 20, borderWidth: 1, borderColor: 'rgba(255,255,255,0.1)', padding: 18, paddingBottom: 34, maxHeight: '70%', overflow: 'hidden' },
  opt: { flexDirection: 'row', alignItems: 'center', gap: 10, paddingVertical: 12 },
  center: { flex: 1, alignItems: 'center', justifyContent: 'center', gap: 14 },
})
