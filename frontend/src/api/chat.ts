import { http } from '@/utils/request'
import type { ChatQueryParams, DocumentChunk } from '@/types'

// 普通问答
export const chatQuery = (params: ChatQueryParams) => {
  return http.post<{
    answer: string
    documents: DocumentChunk[]
  }>('/chat/query', params)
}

// 流式问答
export const chatStream = async (
  query: string,
  callbacks: {
    onDocuments?: (documents: DocumentChunk[]) => void
    onToken?: (token: string) => void
    onDone?: () => void
    onError?: (error: string) => void
  }
) => {
  const response = await fetch('/api/chat/stream', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Accept': 'text/event-stream'
    },
    body: JSON.stringify({ query })
  })

  if (!response.ok) {
    throw new Error('流式请求失败')
  }

  const reader = response.body?.getReader()
  if (!reader) {
    throw new Error('无法读取响应流')
  }

  const decoder = new TextDecoder()
  let buffer = ''

  try {
    while (true) {
      const { done, value } = await reader.read()
      if (done) break

      buffer += decoder.decode(value, { stream: true })
      const lines = buffer.split('\n')
      
      // 保留最后一个不完整的行
      buffer = lines.pop() || ''

      for (const line of lines) {
        if (line.startsWith('data: ')) {
          try {
            const event = JSON.parse(line.slice(6))

            switch (event.type) {
              case 'documents':
                callbacks.onDocuments?.(event.data)
                break
              case 'token':
                callbacks.onToken?.(event.data.content)
                break
              case 'done':
                callbacks.onDone?.()
                return
              case 'error':
                callbacks.onError?.(event.data.message)
                return
            }
          } catch (e) {
            console.error('解析SSE事件失败:', e)
          }
        }
      }
    }
  } finally {
    reader.releaseLock()
  }
}

