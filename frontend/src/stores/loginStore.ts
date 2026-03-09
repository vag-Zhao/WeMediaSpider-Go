import { create } from 'zustand'
import type { LoginStatus } from '../types'

interface LoginStore {
  loginStatus: LoginStatus | null
  setLoginStatus: (status: LoginStatus) => void
  clearLoginStatus: () => void
}

export const useLoginStore = create<LoginStore>((set) => ({
  loginStatus: null,
  setLoginStatus: (status) => set({ loginStatus: status }),
  clearLoginStatus: () => set({ loginStatus: null }),
}))
