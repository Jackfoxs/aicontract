import { create } from 'zustand'
import type {
  ProcurementRequirement,
  AnalyzeRequirementResponse,
} from '@/types'

interface ProcurementState {
  // 当前需求
  currentRequirement: ProcurementRequirement | null
  setCurrentRequirement: (requirement: ProcurementRequirement | null) => void

  // 分析结果
  analysisResult: AnalyzeRequirementResponse | null
  setAnalysisResult: (result: AnalyzeRequirementResponse | null) => void

  // 需求列表
  requirementList: ProcurementRequirement[]
  setRequirementList: (list: ProcurementRequirement[]) => void

  // 历史案例
  historicalCases: any[]
  setHistoricalCases: (cases: any[]) => void

  // 加载状态
  loading: boolean
  setLoading: (loading: boolean) => void

  // 分析中状态
  analyzing: boolean
  setAnalyzing: (analyzing: boolean) => void

  // 校验中状态
  verifying: boolean
  setVerifying: (verifying: boolean) => void

  // 重置状态
  reset: () => void
}

const useProcurementStore = create<ProcurementState>((set) => ({
  currentRequirement: null,
  setCurrentRequirement: (requirement) =>
    set({ currentRequirement: requirement }),

  analysisResult: null,
  setAnalysisResult: (result) => set({ analysisResult: result }),

  requirementList: [],
  setRequirementList: (list) => set({ requirementList: list }),

  historicalCases: [],
  setHistoricalCases: (cases) => set({ historicalCases: cases }),

  loading: false,
  setLoading: (loading) => set({ loading }),

  analyzing: false,
  setAnalyzing: (analyzing) => set({ analyzing }),

  verifying: false,
  setVerifying: (verifying) => set({ verifying }),

  reset: () =>
    set({
      currentRequirement: null,
      analysisResult: null,
      requirementList: [],
      historicalCases: [],
      loading: false,
      analyzing: false,
      verifying: false,
    }),
}))

export default useProcurementStore

