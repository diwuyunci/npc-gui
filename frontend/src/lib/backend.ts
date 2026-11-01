import type { Backend } from './types'
import { mockBackend } from './mock'

type WailsMethod = (...args: unknown[]) => Promise<unknown>
type WailsApp = Record<string, WailsMethod>

declare global {
  interface Window {
    go?: { main?: { App?: WailsApp } }
    runtime?: {
      EventsOn?: (name: string, cb: (...args: unknown[]) => void) => void
      EventsOff?: (name: string) => void
    }
  }
}

function wailsApp(): WailsApp | null {
  return window.go?.main?.App ?? null
}

export const isWails = Boolean(wailsApp())
export const isMock = !isWails

function makeWailsBackend(app: WailsApp): Backend {
  const call = (name: string, ...args: unknown[]) => {
    const fn = app[name]
    if (!fn) {
      return Promise.reject(new Error('Wails binding not found: ' + name))
    }
    return fn(...args)
  }
  return {
    getBootstrap: () => call('GetBootstrap') as ReturnType<Backend['getBootstrap']>,
    saveProfile: (p) => call('SaveProfile', p) as ReturnType<Backend['saveProfile']>,
    deleteProfile: (id) => call('DeleteProfile', id) as ReturnType<Backend['deleteProfile']>,
    duplicateProfile: (id) => call('DuplicateProfile', id) as ReturnType<Backend['duplicateProfile']>,
    importCommand: (raw) => call('ImportCommand', raw) as ReturnType<Backend['importCommand']>,
    commitImport: (p) => call('CommitImport', p) as ReturnType<Backend['commitImport']>,
    importProfiles: () => call('ImportProfiles') as ReturnType<Backend['importProfiles']>,
    exportProfiles: (ids, secrets) => call('ExportProfiles', ids, secrets) as ReturnType<Backend['exportProfiles']>,
    startProfile: (id) => call('StartProfile', id) as ReturnType<Backend['startProfile']>,
    stopProfile: (id) => call('StopProfile', id) as ReturnType<Backend['stopProfile']>,
    restartProfile: (id) => call('RestartProfile', id) as ReturnType<Backend['restartProfile']>,
    startAll: () => call('StartAll') as ReturnType<Backend['startAll']>,
    stopAll: () => call('StopAll') as ReturnType<Backend['stopAll']>,
    getLogs: (id, after, limit) => call('GetLogs', id, after, limit) as ReturnType<Backend['getLogs']>,
    getAllRecentLogs: (limit) => call('GetAllRecentLogs', limit) as ReturnType<Backend['getAllRecentLogs']>,
    clearLogs: (id) => call('ClearLogs', id) as ReturnType<Backend['clearLogs']>,
    exportLogs: (id) => call('ExportLogs', id) as ReturnType<Backend['exportLogs']>,
    previewConfig: (id, reveal) => call('PreviewConfig', id, reveal) as ReturnType<Backend['previewConfig']>,
    exportConfig: (id) => call('ExportConfig', id) as ReturnType<Backend['exportConfig']>,
    getSettings: () => call('GetSettings') as ReturnType<Backend['getSettings']>,
    saveSettings: (s) => call('SaveSettings', s) as ReturnType<Backend['saveSettings']>,
    detectNpcBinaries: () => call('DetectNpcBinaries') as ReturnType<Backend['detectNpcBinaries']>,
    validateNpcBinary: (path) => call('ValidateNpcBinary', path) as ReturnType<Backend['validateNpcBinary']>,
    setNpcBinary: (path) => call('SetNpcBinary', path) as ReturnType<Backend['setNpcBinary']>,
    pickNpcBinary: () => call('PickNpcBinary') as ReturnType<Backend['pickNpcBinary']>,
    downloadNpc: () => call('DownloadNpc') as ReturnType<Backend['downloadNpc']>,
    cancelNpcDownload: () => call('CancelNpcDownload') as ReturnType<Backend['cancelNpcDownload']>,
    checkAppUpdate: (force) => call('CheckAppUpdate', force) as ReturnType<Backend['checkAppUpdate']>,
    downloadAppUpdate: () => call('DownloadAppUpdate') as ReturnType<Backend['downloadAppUpdate']>,
    cancelAppUpdate: () => call('CancelAppUpdate') as ReturnType<Backend['cancelAppUpdate']>,
    discardAppUpdate: () => call('DiscardAppUpdate') as ReturnType<Backend['discardAppUpdate']>,
    applyAppUpdate: (stopAll) => call('ApplyAppUpdate', stopAll) as ReturnType<Backend['applyAppUpdate']>,
    getUpdateState: () => call('GetUpdateState') as ReturnType<Backend['getUpdateState']>,
    openUpdateLog: () => call('OpenUpdateLog') as ReturnType<Backend['openUpdateLog']>,
    openDataDir: () => call('OpenDataDir') as ReturnType<Backend['openDataDir']>,
    openExternal: (url) => call('OpenExternal', url) as ReturnType<Backend['openExternal']>,
    openReleasePage: () => call('OpenReleasePage') as ReturnType<Backend['openReleasePage']>,
    openNpcReleasePage: () => call('OpenNpcReleasePage') as ReturnType<Backend['openNpcReleasePage']>,
    getDiagnostics: () => call('GetDiagnostics') as ReturnType<Backend['getDiagnostics']>,
    confirmQuit: () => call('ConfirmQuit') as Promise<void>,
  }
}

export const backend: Backend = isWails && wailsApp() ? makeWailsBackend(wailsApp() as WailsApp) : mockBackend

export function errorMessage(err: unknown): string {
  if (typeof err === 'string') return err
  if (err && typeof err === 'object' && 'message' in err) return String((err as { message: unknown }).message)
  return '操作失败，请查看日志'
}
