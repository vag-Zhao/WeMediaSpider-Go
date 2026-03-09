export interface Config {
  maxPages: number
  requestInterval: number
  maxWorkers: number
  includeContent: boolean
  cacheExpireHours: number
  outputDir: string
}

export interface ScrapeConfig {
  accounts: string[]
  startDate: string
  endDate: string
  maxPages: number
  requestInterval: number
  includeContent: boolean
  keywordFilter: string
  maxWorkers: number
}
