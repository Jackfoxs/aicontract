import { create } from 'zustand'
import type {
  ContractReview,
  ContractUploadResponse,
  ContractReviewResponse,
} from '@/types'

interface ContractState {
  // 当前合同审核
  currentReview: ContractReview | null
  setCurrentReview: (review: ContractReview | null) => void

  // 上传响应
  uploadResponse: ContractUploadResponse | null
  setUploadResponse: (response: ContractUploadResponse | null) => void

  // 审核结果
  reviewResult: ContractReviewResponse | null
  setReviewResult: (result: ContractReviewResponse | null) => void

  // 审核列表
  reviewList: ContractReview[]
  setReviewList: (list: ContractReview[]) => void

  // 加载状态
  loading: boolean
  setLoading: (loading: boolean) => void

  // 上传中状态
  uploading: boolean
  setUploading: (uploading: boolean) => void

  // 审核中状态
  reviewing: boolean
  setReviewing: (reviewing: boolean) => void

  // 审核进度
  reviewProgress: number
  setReviewProgress: (progress: number) => void

  // 重置状态
  reset: () => void
}

const useContractStore = create<ContractState>((set) => ({
  currentReview: null,
  setCurrentReview: (review) => set({ currentReview: review }),

  uploadResponse: null,
  setUploadResponse: (response) => set({ uploadResponse: response }),

  reviewResult: null,
  setReviewResult: (result) => set({ reviewResult: result }),

  reviewList: [],
  setReviewList: (list) => set({ reviewList: list }),

  loading: false,
  setLoading: (loading) => set({ loading }),

  uploading: false,
  setUploading: (uploading) => set({ uploading }),

  reviewing: false,
  setReviewing: (reviewing) => set({ reviewing }),

  reviewProgress: 0,
  setReviewProgress: (progress) => set({ reviewProgress: progress }),

  reset: () =>
    set({
      currentReview: null,
      uploadResponse: null,
      reviewResult: null,
      reviewList: [],
      loading: false,
      uploading: false,
      reviewing: false,
      reviewProgress: 0,
    }),
}))

export default useContractStore

