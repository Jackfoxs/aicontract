import request from '@/utils/request'
import type {
  ApiResponse,
  PageResponse,
  ProcurementRequirement,
  AnalyzeRequirementRequest,
  AnalyzeRequirementResponse,
  VerifyParametersRequest,
  VerifyParametersResponse,
  HistoricalCasesRequest,
  ProcurementListParams,
} from '@/types'

/**
 * 分析采购需求，生成技术参数
 */
export const analyzeProcurementRequirement = (data: AnalyzeRequirementRequest) => {
  return request.post<ApiResponse<AnalyzeRequirementResponse>>(
    '/api/procurement/analyze',
    data
  )
}

/**
 * 校验已有参数
 */
export const verifyParameters = (data: VerifyParametersRequest) => {
  return request.post<ApiResponse<VerifyParametersResponse>>(
    '/api/procurement/verify',
    data
  )
}

/**
 * 获取历史案例
 */
export const getHistoricalCases = (params: HistoricalCasesRequest) => {
  return request.get<ApiResponse<any[]>>('/api/procurement/historical-cases', {
    params,
  })
}

/**
 * 获取采购需求列表
 */
export const getProcurementList = (params: ProcurementListParams) => {
  return request.get<ApiResponse<PageResponse<ProcurementRequirement>>>(
    '/api/procurement/list',
    { params }
  )
}

/**
 * 获取采购需求详情
 */
export const getProcurementDetail = (id: number) => {
  return request.get<ApiResponse<ProcurementRequirement>>(
    `/api/procurement/${id}`
  )
}

/**
 * 删除采购需求
 */
export const deleteProcurementRequirement = (id: number) => {
  return request.delete<ApiResponse<null>>(`/api/procurement/${id}`)
}

