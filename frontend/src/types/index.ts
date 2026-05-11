// API 响应类型
export interface ApiResponse<T = any> {
  code: number
  data: T
  msg: string
}

// 分页响应
export interface PageResponse<T> {
  list: T[]
  total: number
  page: number
  pageSize: number
}

// 文章类型
export interface Article {
  id: string  // 使用string避免JavaScript大整数精度丢失
  title: string
  type: string
  content: string
  attachment?: string
  attachment_content?: string
  has_attachment: boolean
  created_at: string
  updated_at: string
}

// 文档块类型
export interface DocumentChunk {
  id: string  // 使用string避免JavaScript大整数精度丢失
  article_id: string  // 使用string避免JavaScript大整数精度丢失
  title: string
  content: string
  relevance?: number
  created_at: string
}

// 聊天消息类型
export interface ChatMessage {
  type: 'user' | 'assistant'
  content: string
  documents?: DocumentChunk[]
  timestamp?: string
}

// 文章上传参数
export interface ArticleUploadParams {
  title: string
  type: string
  content?: string
  attachment?: File
}

// 文章列表查询参数
export interface ArticleListParams {
  page?: number
  pageSize?: number
  keyword?: string
  type?: string
}

// 文章更新参数
export interface ArticleUpdateParams {
  title?: string
  type?: string
  content?: string
}

// 搜索参数
export interface SearchParams {
  query: string
  type?: string
  page?: number
  pageSize?: number
}

// 聊天查询参数
export interface ChatQueryParams {
  query: string
}

// 流式事件类型
export interface StreamEvent {
  type: 'documents' | 'token' | 'done' | 'error'
  data: any
}

// ==================== 采购需求模块 ====================

// 采购需求
export interface ProcurementRequirement {
  id: string  // 使用string避免JavaScript大整数精度丢失
  requirement_text: string
  device_type?: string
  department?: string
  budget?: number
  generated_params: any // JSON对象
  compliance_issues: ComplianceIssue[]
  suggestions: string[]
  historical_cases: any[]
  analysis_quality: number
  status: 'draft' | 'analyzing' | 'completed' | 'failed'
  created_at: string
  updated_at: string
  processing_time: number
  llm_cost: number
}

// 合规性问题
export interface ComplianceIssue {
  type: 'missing' | 'invalid' | 'suboptimal'
  severity: 'high' | 'medium' | 'low'
  field: string
  description: string
  suggestion: string
  reference?: string
}

// 采购需求分析请求
export interface AnalyzeRequirementRequest {
  requirement_text: string
  device_type?: string
  department?: string
  budget?: number
}

// 采购需求分析响应
export interface AnalyzeRequirementResponse {
  requirement_id: string  // 使用string避免JavaScript大整数精度丢失
  generated_params: any
  compliance_issues: ComplianceIssue[]
  suggestions: string[]
  historical_cases: any[]
  analysis_quality: number
  processing_time: number
  llm_cost: number
}

// 参数校验请求
export interface VerifyParametersRequest {
  requirement_id: string  // 使用string避免JavaScript大整数精度丢失
  parameters: any
}

// 参数校验响应
export interface VerifyParametersResponse {
  is_valid: boolean
  compliance_issues: ComplianceIssue[]
  suggestions: string[]
  overall_score: number
}

// 历史案例查询请求
export interface HistoricalCasesRequest {
  query: string
  device_type?: string
  limit?: number
}

// 采购需求列表查询参数
export interface ProcurementListParams {
  page?: number
  page_size?: number
  status?: string
  device_type?: string
  start_date?: string
  end_date?: string
}

// ==================== 合同审核模块 ====================

// 合同审核记录
export interface ContractReview {
  id: string  // 使用string避免JavaScript大整数精度丢失
  procurement_id?: string  // 使用string避免JavaScript大整数精度丢失
  contract_title: string
  contract_path: string
  contract_content?: string
  extracted_fields?: ExtractedFields
  risk_items: RiskItem[]
  suggestions: ReviewSuggestion[]
  overall_risk: 'High' | 'Medium' | 'Low'
  review_status: 'pending' | 'reviewing' | 'completed' | 'failed'
  review_time: number
  llm_cost: number
  created_at: string
  updated_at: string
}

// 提取的字段
export interface ExtractedFields {
  parties_info?: PartiesInfo
  amount_info?: AmountInfo
  device_info?: DeviceInfo
  contract_dates?: ContractDates
  confidence: number
}

// 参与方信息
export interface PartiesInfo {
  party_a: ContractParty
  party_b: ContractParty
  other_parties?: ContractParty[]
  confidence: number
}

// 合同参与方
export interface ContractParty {
  role: string
  name: string
  unified_social_credit_code?: string
  address?: string
  contact?: string
  confidence: number
}

// 金额信息
export interface AmountInfo {
  total_amount: number
  currency: string
  amount_in_words: string
  confidence: number
}

// 设备信息
export interface DeviceInfo {
  name: string
  model?: string
  quantity?: number
  unit?: string
  confidence: number
}

// 合同日期
export interface ContractDates {
  signing_date?: string
  effective_date?: string
  expiration_date?: string
  delivery_date?: string
  confidence: number
}

// 风险项
export interface RiskItem {
  type: 'Compliance' | 'Consistency' | 'MissingClause' | 'AbnormalClause' | 'LogicalConflict'
  severity: 'High' | 'Medium' | 'Low'
  description: string
  location?: string
  suggestion: string
  basis: string
}

// 审核建议
export interface ReviewSuggestion {
  type: 'Optimization' | 'Clarification' | 'Standardize'
  description: string
  target_clause?: string
}

// 合同上传请求
export interface ContractUploadRequest {
  file: File
  procurement_id?: string  // 使用string避免JavaScript大整数精度丢失
  contract_title?: string
}

// 合同上传响应
export interface ContractUploadResponse {
  review_id: string  // 使用string避免JavaScript大整数精度丢失
  contract_title: string
  contract_path: string
  parse_preview?: {
    content_length: number
    quality_score: number
    extracted_fields: ExtractedFields
  }
}

// 合同审核响应
export interface ContractReviewResponse {
  review_id: string  // 使用string避免JavaScript大整数精度丢失
  overall_risk: string
  risk_items: RiskItem[]
  suggestions: ReviewSuggestion[]
  extracted_fields: ExtractedFields
  review_time: number
  llm_cost: number
}

// 合同列表查询参数
export interface ContractListParams {
  page?: number
  page_size?: number
  review_status?: string
  overall_risk?: string
  start_date?: string
  end_date?: string
}

// 审核报告
export interface ContractReportResponse {
  review: ContractReview
  report_markdown: string
  report_html?: string
}

