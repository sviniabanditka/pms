import type { Meta, StoryObj } from '@storybook-vue/nuxt'
import CommentItem from './CommentItem.vue'
import { withStore } from '../.storybook/decorators'

// Молекула, выделенная из IssueDetailPanel. Свой edit/react-picker стейт.
const base = {
  id: 'c1', authorId: 'user_2', issueId: 'i1', body: 'This looks **good** — one nit: rename `foo` → `bar`.',
  createdAt: '2026-07-03T10:00:00Z', reactions: [{ emoji: '👍', userId: 'user_1' }], editedAt: null, deletedAt: null,
}
const meta: Meta<typeof CommentItem> = {
  title: 'Molecules/CommentItem',
  component: CommentItem,
  decorators: [withStore()],
  parameters: { layout: 'fullscreen' },
  args: { comment: base, issueId: 'i1' },
}
export default meta
type Story = StoryObj<typeof CommentItem>

export const FromOther: Story = {}
export const Own: Story = { args: { comment: { ...base, authorId: 'user_1' } } } // currentUserId → edit/delete
export const Edited: Story = { args: { comment: { ...base, editedAt: '2026-07-03T11:00:00Z' } } }
export const Deleted: Story = { args: { comment: { ...base, deletedAt: '2026-07-03T12:00:00Z' } } }
