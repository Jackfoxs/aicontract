import { http } from '@/utils/request'
import type { SearchParams, DocumentChunk, PageResponse } from '@/types'

// 搜索文档
export const searchDocuments = (params: SearchParams) => {
  return http.post<PageResponse<DocumentChunk>>('/search/documents', params)
}

