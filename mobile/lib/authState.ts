// Крошечный shared-стейт «залогинен ли» — чтобы login/logout мгновенно двигали
// auth-гейт в _layout. ponytail: useSyncExternalStore вместо Context/zustand.
import { useSyncExternalStore } from 'react'

let authed = false
const subs = new Set<() => void>()

export function setAuthed(v: boolean) {
  authed = v
  subs.forEach((f) => f())
}
export function getAuthed() {
  return authed
}
export function useAuthed() {
  return useSyncExternalStore(
    (cb) => {
      subs.add(cb)
      return () => subs.delete(cb)
    },
    getAuthed,
    getAuthed,
  )
}
