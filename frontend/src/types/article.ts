export interface Article {
  id: string
  accountName: string
  accountFakeid: string
  title: string
  link: string
  digest: string
  content: string
  publishTime: string
  publishTimestamp: number
  createdAt: string
}

export interface ArticleList {
  total: number
  articles: Article[]
}

export interface ArticleFilter {
  accountName?: string
  keyword?: string
  startDate?: string
  endDate?: string
}
