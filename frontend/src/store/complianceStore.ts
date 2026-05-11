import { create } from 'zustand'
import { devtools } from 'zustand/middleware'

export type ComplianceJobStatus =
  | 'pending'
  | 'running'
  | 'success'
  | 'failed'

export interface ComplianceJob {
  job_id: string
  status: ComplianceJobStatus
  progress: number
  tender_file_id: string
  response_file_id: string
  created_at: string
  llm_model?: string
  analysis_summary?: string
}

export interface ComplianceIssue {
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

export interface ComplianceHighlight {
  id: string
  file_role: 'tender' | 'response'
  page: number
  offset_start: number
  offset_end: number
  text: string
}

export interface ComplianceJobDetail {
  job: {
    job_id: string
    status: ComplianceJobStatus
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
  issues: ComplianceIssue[]
}

interface Pagination {
  page: number
  pageSize: number
  total: number
}

interface ComplianceState {
  jobs: ComplianceJob[]
  pagination: Pagination
  currentJobId?: string
  currentDetail?: ComplianceJobDetail
  highlights: ComplianceHighlight[]
  loading: boolean
  issueLoading: boolean
  highlightLoading: boolean
  setJobs: (jobs: ComplianceJob[], total: number, page: number, pageSize: number) => void
  setCurrentJobId: (jobId?: string) => void
  setDetail: (detail?: ComplianceJobDetail) => void
  setHighlights: (data: ComplianceHighlight[]) => void
  setLoading: (loading: boolean) => void
  setIssueLoading: (loading: boolean) => void
  setHighlightLoading: (loading: boolean) => void
}

export const useComplianceStore = create<ComplianceState>()(
  devtools((set) => ({
    jobs: [],
    pagination: { page: 1, pageSize: 10, total: 0 },
    currentJobId: undefined,
    currentDetail: undefined,
    highlights: [],
    loading: false,
    issueLoading: false,
    highlightLoading: false,
    setJobs: (jobs, total, page, pageSize) =>
      set({ jobs, pagination: { page, pageSize, total } }),
    setCurrentJobId: (jobId) => set({ currentJobId: jobId }),
    setDetail: (detail) => set({ currentDetail: detail }),
    setHighlights: (data) => set({ highlights: data }),
    setLoading: (loading) => set({ loading }),
    setIssueLoading: (loading) => set({ issueLoading: loading }),
    setHighlightLoading: (loading) => set({ highlightLoading: loading }),
  }))
)


