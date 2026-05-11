import axios, { AxiosError, AxiosRequestConfig, AxiosResponse } from 'axios'
import { Message } from '@arco-design/web-react'
import type { ApiResponse } from '@/types'

// 创建 axios 实例
const request = axios.create({
  baseURL: '/api',
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json'
  }
})

// 请求拦截器
request.interceptors.request.use(
  (config) => {
    // 可以在这里添加token等
    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

// 响应拦截器
request.interceptors.response.use(
  (response: AxiosResponse) => {
    if (response.config.responseType === 'blob') {
      return response
    }

    const res = response.data as ApiResponse

    if (res && typeof res.code === 'number' && res.code !== 0) {
      Message.error(res.msg || '请求失败')
      return Promise.reject(new Error(res.msg || '请求失败'))
    }

    return response
  },
  (error: AxiosError) => {
    let message = '网络错误'
    
    if (error.response) {
      // 服务器返回错误状态码
      const status = error.response.status
      const data = error.response.data as any
      
      if (status === 500) {
        message = data?.msg || '服务器错误，请稍后重试'
      } else if (status === 404) {
        message = '请求的资源不存在'
      } else if (status === 403) {
        message = '没有权限访问'
      } else if (status === 401) {
        message = '未授权，请先登录'
      } else {
        message = data?.msg || error.message || '请求失败'
      }
    } else if (error.request) {
      // 请求已发出但没有收到响应
      message = '网络连接失败，请检查网络'
    }
    
    Message.error(message)
    return Promise.reject(error)
  }
)

export default request

// 封装常用请求方法
const extractData = <T>(promise: Promise<AxiosResponse<ApiResponse<T>>>): Promise<ApiResponse<T>> => {
  return promise.then((res) => res.data as ApiResponse<T>)
}

export const http = {
  get<T = any>(url: string, config?: AxiosRequestConfig): Promise<ApiResponse<T>> {
    return extractData(request.get(url, config))
  },

  post<T = any>(url: string, data?: any, config?: AxiosRequestConfig): Promise<ApiResponse<T>> {
    return extractData(request.post(url, data, config))
  },

  put<T = any>(url: string, data?: any, config?: AxiosRequestConfig): Promise<ApiResponse<T>> {
    return extractData(request.put(url, data, config))
  },

  delete<T = any>(url: string, config?: AxiosRequestConfig): Promise<ApiResponse<T>> {
    return extractData(request.delete(url, config))
  },

  upload<T = any>(url: string, formData: FormData, config?: AxiosRequestConfig): Promise<ApiResponse<T>> {
    return extractData(
      request.post(url, formData, {
        ...config,
        headers: {
          'Content-Type': 'multipart/form-data'
        }
      })
    )
  }
}

