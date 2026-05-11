import { create } from 'zustand'
import type { ChatMessage, DocumentChunk } from '@/types'
import { chatQuery, chatStream } from '@/api/chat'

interface ChatState {
  messages: ChatMessage[]
  loading: boolean
  streaming: boolean
  currentAnswer: string
  currentDocuments: DocumentChunk[]
  
  // Actions
  sendMessage: (query: string, useStream?: boolean) => Promise<void>
  clearMessages: () => void
  addMessage: (message: ChatMessage) => void
}

export const useChatStore = create<ChatState>((set, get) => ({
  messages: [],
  loading: false,
  streaming: false,
  currentAnswer: '',
  currentDocuments: [],

  sendMessage: async (query, useStream = true) => {
    // 添加用户消息
    const userMessage: ChatMessage = {
      type: 'user',
      content: query,
      timestamp: new Date().toISOString()
    }
    
    set((state) => ({
      messages: [...state.messages, userMessage],
      loading: true,
      streaming: useStream,
      currentAnswer: '',
      currentDocuments: []
    }))

    try {
      if (useStream) {
        // 流式响应
        await chatStream(query, {
          onDocuments: (documents) => {
            set({ currentDocuments: documents })
          },
          onToken: (token) => {
            set((state) => ({
              currentAnswer: state.currentAnswer + token
            }))
          },
          onDone: () => {
            const { currentAnswer, currentDocuments } = get()
            const assistantMessage: ChatMessage = {
              type: 'assistant',
              content: currentAnswer,
              documents: currentDocuments,
              timestamp: new Date().toISOString()
            }
            
            set((state) => ({
              messages: [...state.messages, assistantMessage],
              loading: false,
              streaming: false,
              currentAnswer: '',
              currentDocuments: []
            }))
          },
          onError: (error) => {
            console.error('流式聊天错误:', error)
            set({
              loading: false,
              streaming: false,
              currentAnswer: '',
              currentDocuments: []
            })
          }
        })
      } else {
        // 普通响应
        const res = await chatQuery({ query })
        const assistantMessage: ChatMessage = {
          type: 'assistant',
          content: res.data.answer,
          documents: res.data.documents,
          timestamp: new Date().toISOString()
        }
        
        set((state) => ({
          messages: [...state.messages, assistantMessage],
          loading: false
        }))
      }
    } catch (error) {
      console.error('发送消息失败:', error)
      set({
        loading: false,
        streaming: false,
        currentAnswer: '',
        currentDocuments: []
      })
    }
  },

  clearMessages: () => {
    set({
      messages: [],
      currentAnswer: '',
      currentDocuments: []
    })
  },

  addMessage: (message) => {
    set((state) => ({
      messages: [...state.messages, message]
    }))
  }
}))

