import { create } from 'zustand'
import type { Article, Progress, AccountStatus } from '../types'

interface ScrapeStore {
  articles: Article[]
  progress: Progress | null
  accountStatuses: AccountStatus[]
  isScrapingInProgress: boolean
  setArticles: (articles: Article[]) => void
  addArticles: (articles: Article[]) => void
  setProgress: (progress: Progress | null) => void
  addAccountStatus: (status: AccountStatus) => void
  updateAccountStatus: (accountName: string, status: AccountStatus) => void
  clearAccountStatuses: () => void
  setScrapingInProgress: (inProgress: boolean) => void
  reset: () => void
}

export const useScrapeStore = create<ScrapeStore>((set) => ({
  articles: [],
  progress: null,
  accountStatuses: [],
  isScrapingInProgress: false,

  setArticles: (articles) => set({ articles }),

  addArticles: (articles) => set((state) => ({
    articles: [...state.articles, ...articles]
  })),

  setProgress: (progress) => set({ progress }),

  addAccountStatus: (status) => set((state) => ({
    accountStatuses: [...state.accountStatuses, status]
  })),

  updateAccountStatus: (accountName, status) => set((state) => {
    const existing = state.accountStatuses.find((s) => s.accountName === accountName)
    if (existing) {
      // 更新已存在的账号
      return {
        accountStatuses: state.accountStatuses.map((s) =>
          s.accountName === accountName ? status : s
        )
      }
    } else {
      // 添加新账号
      return {
        accountStatuses: [...state.accountStatuses, status]
      }
    }
  }),

  clearAccountStatuses: () => set({ accountStatuses: [] }),

  setScrapingInProgress: (inProgress) => set({ isScrapingInProgress: inProgress }),

  reset: () => set({
    articles: [],
    progress: null,
    accountStatuses: [],
    isScrapingInProgress: false
  }),
}))
