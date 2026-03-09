export type ProgressType = 'account' | 'article' | 'content'

export interface Progress {
  type: ProgressType
  current: number
  total: number
  message: string
}

export interface AccountStatus {
  accountName: string
  status: string
  message: string
  articleCount: number
}
