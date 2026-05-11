import { http as request } from '@/utils/request'

export interface ChunkItem {
  id: string
  chunk_id: string
  article_id: string
  title: string
  content: string
  is_attachment: boolean
  page: number
  char_start: number
  char_end: number
  section_path: string
  rule_id: string
  anchors: string
  aliases: string
  fingerprint: string
  order_index: number
}

export async function getChunks(articleId: string) {
  return request.get<{ list: ChunkItem[] }>(`/document/chunks`, { params: { article_id: articleId } })
}

export async function updateChunk(chunkId: string, data: { title?: string; content?: string }) {
  return request.put(`/document/chunks/${chunkId}`, data)
}

export async function splitChunk(chunkId: string) {
  return request.post(`/document/chunks/split`, { chunk_id: chunkId, mode: 'articles_only' })
}

export async function mergeChunks(chunkIds: string[]) {
  return request.post(`/document/chunks/merge`, { chunk_ids: chunkIds })
}

export async function reorderChunks(orders: { chunk_id: string; order_index: number }[]) {
  return request.post(`/document/chunks/reorder`, { orders })
}

export async function deleteChunk(chunkId: string) {
  return request.delete(`/document/chunks/${chunkId}`)
}


