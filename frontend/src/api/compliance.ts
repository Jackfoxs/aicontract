import type { AxiosResponse } from 'axios'
import request, { http } from '@/utils/request'

export interface SubmitComplianceJobPayload {
  tender_file_id: string
  response_file_id: string
  norm_set_id?: string
  selected_rule_ids?: string[]
  ai_threshold?: number
}

export function submitComplianceJob(data: SubmitComplianceJobPayload) {
  return http.post<{ job_id: string }>('/compliance/jobs', data)
}

export interface ComplianceJobListParams {
  page?: number
  page_size?: number
}

export interface ComplianceJobListItem {
  job_id: string
  status: string
  progress: number
  tender_file_id: string
  response_file_id: string
  created_at: string
  llm_model?: string
  analysis_summary?: string
}

export interface ComplianceJobListResponse {
  total: number
  list: ComplianceJobListItem[]
}

export interface ComplianceRuleQuery {
  page?: number
  page_size?: number
  keyword?: string
  include_content?: number
}

export interface ComplianceRuleOption {
  id: string
  title: string
  content_preview: string
  rule_id: string
  section_path: string
  content?: string
}

export interface ComplianceRuleListResponse {
  total: number
  list: ComplianceRuleOption[]
}

export function fetchComplianceJobs(params: ComplianceJobListParams) {
  return http.get<ComplianceJobListResponse>('/compliance/jobs', { params })
}

export function fetchComplianceRules(params: ComplianceRuleQuery) {
  return http.get<ComplianceRuleListResponse>('/compliance/rules', { params })
}

export interface ComplianceJobDetailBody {
  job: {
    job_id: string
    status: string
    progress: number
    error_message: string
    report_path_json: string
    report_path_csv: string
    report_path_pdf: string
    ai_threshold: number
    norm_set_id: string
    created_by: string
    created_at: string
    updated_at: string
    llm_model?: string
    llm_cost?: number
    analysis_summary?: string
  }
  issues: ComplianceIssueResponse[]
}

export function fetchComplianceJobDetail(jobId: string) {
  return http.get<ComplianceJobDetailBody>(`/compliance/jobs/${jobId}`)
}

export interface ComplianceIssueResponse {
  id: string
  rule_id: string
  rule_title: string
  required_content: string
  response_excerpt: string
  status: string
  match_score: number
  remark: string
  highlight_ref: string
  llm_advice?: string
  llm_model?: string
  source_type?: string
  requirement_id?: string
  requirement_name?: string
  requirement_level?: string
  gap?: string
  response_refs?: string
}

export function fetchComplianceIssues(jobId: string) {
  return http.get<ComplianceIssueResponse[]>(`/compliance/jobs/${jobId}/issues`)
}

export interface HighlightQuery {
  page?: number
  page_size?: number
  file_role?: 'tender' | 'response'
}

export interface ComplianceHighlightResponse {
  id: string
  file_role: 'tender' | 'response'
  page: number
  offset_start: number
  offset_end: number
  text: string
}

export function fetchComplianceHighlights(jobId: string, params: HighlightQuery) {
  return http.get<ComplianceHighlightResponse[]>(`/compliance/jobs/${jobId}/highlights`, { params })
}

export function retryComplianceJob(jobId: string) {
  return http.post<{ job_id: string }>(`/compliance/jobs/${jobId}/retry`)
}

export function downloadComplianceReport(jobId: string, format: 'json' | 'csv' | 'pdf'): Promise<AxiosResponse<Blob>> {
  return request.get<Blob>(`/compliance/jobs/${jobId}/report`, {
    params: { format },
    responseType: 'blob',
  })
}

export interface UploadComplianceFileResponse {
  file_id: string
  file_role: 'tender' | 'response'
  file_name: string
  file_path: string
  file_size: number
  file_type: string
  parse_method: string
  quality_score: number
  text_length: number
  page_count: number
  uploaded_at: string
  content_preview?: string
}

export function uploadComplianceFile(formData: FormData) {
  return http.upload<UploadComplianceFileResponse>('/compliance/upload', formData)
}


