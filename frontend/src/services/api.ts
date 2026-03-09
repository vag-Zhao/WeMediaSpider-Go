// Wails API bindings
import {
  Login,
  Logout,
  GetLoginStatus,
  ClearLoginCache,
  ExportCredentials,
  ImportCredentials,
  SearchAccount,
  StartScrape,
  CancelScrape,
  ExportArticles,
  LoadConfig,
  SaveConfig,
  GetDefaultConfig,
  SelectDirectory,
  SelectSaveFile,
  ClearCache,
  ClearExpiredCache,
  GetCacheStats,
  GetAppVersion,
  ExtractArticleImages,
  BatchDownloadImages,
  CancelImageDownload,
  GetAppData,
  UpdateAppData,
  ListDataFiles,
  LoadDataFile,
  DeleteDataFile,
  GetDataDirectory,
  OpenDataFileDialog,
  CheckForUpdates,
} from '../../wailsjs/go/app/App'

// Wails runtime
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'

import type {
  LoginStatus,
  Account,
  Article,
  ScrapeConfig,
  Config,
  Progress,
  AccountStatus,
} from '../types'

// API wrapper functions
export const api = {
  // Login APIs
  login: Login,
  logout: Logout,
  getLoginStatus: GetLoginStatus,
  clearLoginCache: ClearLoginCache,
  exportCredentials: ExportCredentials,
  importCredentials: ImportCredentials,

  // Scrape APIs
  searchAccount: SearchAccount,
  startScrape: StartScrape,
  cancelScrape: CancelScrape,

  // Export APIs
  exportArticles: ExportArticles,

  // Config APIs
  loadConfig: LoadConfig,
  saveConfig: SaveConfig,
  getDefaultConfig: GetDefaultConfig,

  // File system APIs
  selectDirectory: SelectDirectory,
  selectSaveFile: SelectSaveFile,

  // Cache APIs
  clearCache: ClearCache,
  clearExpiredCache: ClearExpiredCache,
  getCacheStats: GetCacheStats,

  // Utility APIs
  getAppVersion: GetAppVersion,

  // App data APIs
  getAppData: GetAppData,
  updateAppData: UpdateAppData,

  // Image download APIs
  extractArticleImages: ExtractArticleImages,
  batchDownloadImages: BatchDownloadImages,
  cancelImageDownload: CancelImageDownload,

  // Data management APIs
  listDataFiles: ListDataFiles,
  loadDataFile: LoadDataFile,
  deleteDataFile: DeleteDataFile,
  getDataDirectory: GetDataDirectory,
  openDataFileDialog: OpenDataFileDialog,

  // Version check APIs
  checkForUpdates: CheckForUpdates,
}

// Event listeners
export const events = {
  onScrapeProgress: (callback: (progress: Progress) => void) => {
    return EventsOn('scrape:progress', callback)
  },

  offScrapeProgress: (unsubscribe: () => void) => {
    unsubscribe()
  },

  onScrapeStatus: (callback: (status: AccountStatus) => void) => {
    return EventsOn('scrape:status', callback)
  },

  offScrapeStatus: (unsubscribe: () => void) => {
    unsubscribe()
  },

  onScrapeCompleted: (callback: (data: any) => void) => {
    return EventsOn('scrape:completed', callback)
  },

  offScrapeCompleted: (unsubscribe: () => void) => {
    unsubscribe()
  },

  onScrapeError: (callback: (error: any) => void) => {
    return EventsOn('scrape:error', callback)
  },

  offScrapeError: (unsubscribe: () => void) => {
    unsubscribe()
  },

  onImageProgress: (callback: (progress: any) => void) => {
    return EventsOn('image:progress', callback)
  },

  offImageProgress: (unsubscribe: () => void) => {
    unsubscribe()
  },

  onImageCompleted: (callback: (data: any) => void) => {
    return EventsOn('image:completed', callback)
  },

  offImageCompleted: (unsubscribe: () => void) => {
    unsubscribe()
  },

  onImageError: (callback: (error: any) => void) => {
    return EventsOn('image:error', callback)
  },

  offImageError: (unsubscribe: () => void) => {
    unsubscribe()
  },
}
