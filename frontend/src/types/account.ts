export interface Account {
  name: string
  fakeid: string
  alias: string
  signature: string
  avatar: string
  qrCode: string
  serviceType: number
}

export interface SearchResult {
  total: number
  accounts: Account[]
}
