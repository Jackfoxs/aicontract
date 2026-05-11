import { create } from 'zustand'
import type { DocumentChunk, SearchParams } from '@/types'
import { searchDocuments } from '@/api/search'

interface SearchState {
  results: DocumentChunk[]
  total: number
  loading: boolean
  
  // Actions
  search: (params: SearchParams) => Promise<void>
  clearResults: () => void
}

export const useSearchStore = create<SearchState>((set) => ({
  results: [],
  total: 0,
  loading: false,

  search: async (params) => {
    set({ loading: true })
    try {
      const res = await searchDocuments(params)
      set({
        results: res.data.list,
        total: res.data.total,
        loading: false
      })
    } catch (error) {
      set({ loading: false })
      throw error
    }
  },

  clearResults: () => {
    set({
      results: [],
      total: 0
    })
  }
}))

