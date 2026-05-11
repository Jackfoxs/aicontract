import request from '@/utils/request'
import type {
  ApiResponse,
  PageResponse,
  ContractReview,
  ContractUploadResponse,
  ContractReviewResponse,
  ContractListParams,
  ContractReportResponse,
} from '@/types'

/**
 * 上传合同文件
 */
export const uploadContract = (formData: FormData) => {
  return request.post<ApiResponse<ContractUploadResponse>>(
    '/api/contract/upload',
    formData,
    {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
    }
  )
}

/**
 * 执行合同审核
 */
export const reviewContract = (id: number) => {
  return request.post<ApiResponse<ContractReviewResponse>>(
    `/api/contract/review/${id}`
  )
}

/**
 * 获取合同审核列表
 */
export const getContractList = (params: ContractListParams) => {
  return request.get<ApiResponse<PageResponse<ContractReview>>>(
    '/api/contract/list',
    { params }
  )
}

/**
 * 获取合同审核详情
 */
export const getContractDetail = (id: number) => {
  return request.get<ApiResponse<ContractReview>>(
    `/api/contract/review/${id}`
  )
}

/**
 * 获取合同审核报告
 */
export const getContractReport = (id: number) => {
  return request.get<ApiResponse<ContractReportResponse>>(
    `/api/contract/report/${id}`
  )
}

/**
 * 删除合同审核记录
 */
export const deleteContractReview = (id: number) => {
  return request.delete<ApiResponse<null>>(`/api/contract/review/${id}`)
}

