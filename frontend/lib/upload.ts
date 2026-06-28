import { useRuntimeConfig } from '#imports'
import { getToken } from './token'

export type Uploaded = { key: string; filename: string; contentType: string; size: number }

// Стабильная ссылка на объект (302→presigned на бэкенде). Годится для <img>/markdown-эмбедов.
export function fileViewUrl(key?: string | null): string {
  if (!key) return ''
  const base = useRuntimeConfig().public.apiBase as string
  return `${base}/files/view/${key}`
}

// uploadFile — multipart POST на бэкенд (он кладёт в S3). Возвращает ключ + мету.
export async function uploadFile(workspaceId: string, file: File): Promise<Uploaded> {
  const base = useRuntimeConfig().public.apiBase as string
  const fd = new FormData()
  fd.append('workspace_id', workspaceId)
  fd.append('file', file)
  const res = await fetch(`${base}/files/upload`, {
    method: 'POST',
    headers: { Authorization: 'Bearer ' + getToken() },
    body: fd,
  })
  if (!res.ok) throw new Error('upload failed: ' + res.status)
  return res.json()
}
