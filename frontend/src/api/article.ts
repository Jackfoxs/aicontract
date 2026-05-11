import { http } from '@/utils/request'
import type {
  Article,
  ArticleUploadParams,
  ArticleListParams,
  ArticleUpdateParams,
  PageResponse
} from '@/types'

// 上传文章
export const uploadArticle = (params: ArticleUploadParams) => {
  const formData = new FormData()
  formData.append('title', params.title)
  formData.append('type', params.type)
  if (params.content) {
    formData.append('content', params.content)
  }
  if (params.attachment) {
    formData.append('attachment', params.attachment)
  }

  return http.upload<{ article_id: number }>('/article/upload', formData)
}

// 获取文章列表
export const getArticleList = (params?: ArticleListParams) => {
  // 清理空参数，避免发送空字符串
  const cleanParams: any = {}
  if (params?.page) cleanParams.page = params.page
  if (params?.pageSize) cleanParams.pageSize = params.pageSize
  if (params?.keyword) cleanParams.keyword = params.keyword
  if (params?.type) cleanParams.type = params.type
  
  return http.get<PageResponse<Article>>('/article/list', { params: cleanParams })
}

// 获取文章详情
export const getArticleDetail = (id: string) => {
  return http.get<Article>(`/article/${id}`)
}

// 更新文章
export const updateArticle = (id: string, data: ArticleUpdateParams) => {
  return http.put(`/article/${id}`, data)
}

// 删除文章
export const deleteArticle = (id: string) => {
  return http.delete(`/article/${id}`)
}

