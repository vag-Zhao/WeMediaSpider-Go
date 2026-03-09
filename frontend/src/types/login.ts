export interface LoginStatus {
  isLoggedIn: boolean
  loginTime?: string
  expireTime?: string
  hoursSinceLogin?: number
  hoursUntilExpire?: number
  token?: string
  message: string
}
