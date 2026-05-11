import { create } from 'zustand'
import { Message } from '@arco-design/web-react'
import type { Article, ArticleListParams } from '@/types'
import * as articleApi from '@/api/article'

interface ArticleState {
  articles: Article[]
  total: number
  loading: boolean
  currentArticle: Article | null
  
  // Actions
  fetchArticles: (params?: ArticleListParams) => Promise<void>
  fetchArticleDetail: (id: string) => Promise<void>
  uploadArticle: (params: any) => Promise<boolean>
  updateArticle: (id: string, data: any) => Promise<boolean>
  deleteArticle: (id: string) => Promise<boolean>
  setCurrentArticle: (article: Article | null) => void
}

export const useArticleStore = create<ArticleState>((set) => ({
  articles: [],
  total: 0,
  loading: false,
  currentArticle: null,

  fetchArticles: async (params) => {
    set({ loading: true })
    try {
      const res = await articleApi.getArticleList(params)
      set({
        articles: res.data.list || [],
        total: res.data.total || 0,
        loading: false
      })
    } catch (error) {
      set({ 
        articles: [],
        total: 0,
        loading: false 
      })
      console.error('获取文章列表失败:', error)
    }
  },

  fetchArticleDetail: async (id) => {
    set({ loading: true })
    try {
      const res = await articleApi.getArticleDetail(id)
      set({
        currentArticle: res.data,
        loading: false
      })
    } catch (error) {
      set({ loading: false })
      throw error
    }
  },

  uploadArticle: async (params) => {
    set({ loading: true })
    try {
      await articleApi.uploadArticle(params)
      Message.success('上传成功')
      set({ loading: false })
      return true
    } catch (error) {
      set({ loading: false })
      return false
    }
  },

  updateArticle: async (id, data) => {
    set({ loading: true })
    try {
      await articleApi.updateArticle(id, data)
      Message.success('更新成功')
      set({ loading: false })
      return true
    } catch (error) {
      set({ loading: false })
      return false
    }
  },

  deleteArticle: async (id) => {
    set({ loading: true })
    try {
      await articleApi.deleteArticle(id)
      Message.success('删除成功')
      set({ loading: false })
      return true
    } catch (error) {
      set({ loading: false })
      return false
    }
  },

  setCurrentArticle: (article) => {
    set({ currentArticle: article })
  }
}))

