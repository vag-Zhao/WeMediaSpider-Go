import { create } from 'zustand'
import type { Config } from '../types'

interface ConfigStore {
  config: Config | null
  setConfig: (config: Config) => void
}

export const useConfigStore = create<ConfigStore>((set) => ({
  config: null,
  setConfig: (config) => set({ config }),
}))
