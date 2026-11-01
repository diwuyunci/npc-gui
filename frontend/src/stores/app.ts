import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { backend, errorMessage } from '@/lib/backend'
import type {
  AppInfo,
  BinaryInfo,
  BinaryResult,
  Bootstrap,
  OperationResult,
  Settings,
  UpdateProgress,
  UpdateStateInfo,
  Toast,
} from '@/lib/types'
import { useProfilesStore } from './profiles'
import { useRuntimesStore } from './runtimes'
import { useLogsStore } from './logs'

const defaultSettings: Settings = {
  autoCheckUpdates: true,
  theme: 'dark',
  locale: 'zh-CN',
  defaultLogLevel: 'info',
  maxLogLines: 5000,
  confirmOnQuit: true,
  logToDisk: true,
  launchAtLogin: false,
  trayEnabled: true,
  minimizeToTray: true,
  notifications: true,
}

export const useAppStore = defineStore('app', () => {
  const bootstrapped = ref(false)
  const loading = ref(true)
  const fatal = ref('')
  const appInfo = ref<AppInfo | null>(null)
  const settings = ref<Settings>({ ...defaultSettings })
  const binary = ref<BinaryInfo | null>(null)
  const binaryCandidates = ref<BinaryInfo[]>([])
  const binaryProgress = ref<UpdateProgress | null>(null)
  const binaryDownloading = ref(false)
  const update = ref<UpdateStateInfo>({ state: 'idle', currentVersion: 'dev' })
  const updateProgress = ref<UpdateProgress | null>(null)
  const warnings = ref<string[]>([])
  const toasts = ref<Toast[]>([])
  const closeRequest = ref<number | null>(null)
  const setupSkipped = ref(false)
  const firstRun = ref(false)
  const theme = computed(() => settings.value.theme || 'dark')
  const version = computed(() => appInfo.value?.version || 'dev')
  const updateAvailable = computed(
    () => update.value.state === 'available' || update.value.state === 'ready',
  )
  const updateReady = computed(() => update.value.state === 'ready')
  const needsSetup = computed(() => {
    const profiles = useProfilesStore()
    return !binary.value?.path && profiles.items.length === 0
  })

  let toastSeq = 0

  function toast(level: Toast['level'], message: string, detail?: string): void {
    const item: Toast = { id: ++toastSeq, level, message, detail }
    toasts.value = [...toasts.value, item]
    window.setTimeout(() => dismissToast(item.id), level === 'error' ? 7000 : 4200)
  }

  function dismissToast(id: number): void {
    toasts.value = toasts.value.filter((t) => t.id !== id)
  }

  function applyBootstrap(data: Bootstrap): void {
    appInfo.value = data.app
    settings.value = { ...defaultSettings, ...data.settings }
    binary.value = data.binary ?? null
    binaryCandidates.value = data.binaryCandidates ?? []
    update.value = data.update
    warnings.value = data.warnings ?? []
    firstRun.value = data.firstRun
    useProfilesStore().setAll(data.profiles ?? [])
    useRuntimesStore().setAll(data.runtimes ?? [])
    document.documentElement.dataset.theme = settings.value.theme || 'dark'
  }

  async function load(): Promise<void> {
    loading.value = true
    fatal.value = ''
    try {
      const data = await backend.getBootstrap()
      applyBootstrap(data)
      const recent = await backend.getAllRecentLogs(400)
      useLogsStore().hydrate(recent)
      bootstrapped.value = true
    } catch (err) {
      fatal.value = errorMessage(err)
    } finally {
      loading.value = false
    }
  }

  async function refresh(): Promise<void> {
    try {
      const data = await backend.getBootstrap()
      applyBootstrap(data)
    } catch (err) {
      toast('error', '刷新状态失败', errorMessage(err))
    }
  }

  async function saveSettings(next: Settings): Promise<OperationResult> {
    try {
      const res = await backend.saveSettings(next)
      if (res.result.ok) {
        settings.value = { ...res.settings }
        document.documentElement.dataset.theme = settings.value.theme || 'dark'
      }
      return res.result
    } catch (err) {
      return { ok: false, message: errorMessage(err) }
    }
  }

  async function detectNpcBinaries(): Promise<BinaryInfo[]> {
    try {
      const items = await backend.detectNpcBinaries()
      binaryCandidates.value = items ?? []
      return items ?? []
    } catch (err) {
      toast('error', '检测 npc 失败', errorMessage(err))
      return []
    }
  }

  function applyBinaryResult(res: BinaryResult, successMessage?: string): boolean {
    binaryCandidates.value = res.candidates ?? binaryCandidates.value
    if (res.result.ok && res.binary) {
      binary.value = res.binary
      if (successMessage) toast('success', successMessage, res.binary.version)
      return true
    }
    if (!res.result.ok && res.result.message) toast('error', 'npc 配置失败', res.result.message)
    return false
  }

  async function pickNpcBinary(): Promise<boolean> {
    try {
      const res = await backend.pickNpcBinary()
      return applyBinaryResult(res, '已切换 npc')
    } catch (err) {
      toast('error', '选择 npc 失败', errorMessage(err))
      return false
    }
  }

  async function setNpcBinary(path: string): Promise<boolean> {
    try {
      const res = await backend.setNpcBinary(path)
      return applyBinaryResult(res, '已切换 npc')
    } catch (err) {
      toast('error', '设置 npc 失败', errorMessage(err))
      return false
    }
  }

  async function downloadNpc(): Promise<boolean> {
    if (binaryDownloading.value) return false
    binaryDownloading.value = true
    binaryProgress.value = { stage: '准备下载', received: 0, total: 0, percent: 0 }
    try {
      const res = await backend.downloadNpc()
      return applyBinaryResult(res, 'npc 下载完成')
    } catch (err) {
      toast('error', '下载 npc 失败', errorMessage(err))
      return false
    } finally {
      binaryDownloading.value = false
      binaryProgress.value = null
    }
  }

  async function cancelNpcDownload(): Promise<void> {
    try {
      await backend.cancelNpcDownload()
    } finally {
      binaryDownloading.value = false
    }
  }

  async function checkUpdate(force = true): Promise<void> {
    try {
      const res = await backend.checkAppUpdate(force)
      update.value = res.state
      if (!res.result.ok) toast('warn', '检查更新失败', res.result.message)
      else if (res.state.state === 'idle') toast('success', '当前已是最新版本')
    } catch (err) {
      toast('error', '检查更新失败', errorMessage(err))
    }
  }

  async function downloadUpdate(): Promise<void> {
    try {
      const res = await backend.downloadAppUpdate()
      update.value = res.state
      if (!res.result.ok) toast('error', '下载更新失败', res.result.message)
    } catch (err) {
      toast('error', '下载更新失败', errorMessage(err))
    }
  }

  async function cancelUpdate(): Promise<void> {
    await backend.cancelAppUpdate()
  }

  async function applyUpdate(stopAll: boolean): Promise<boolean> {
    try {
      const res = await backend.applyAppUpdate(stopAll)
      if (res.ok) {
        toast('success', '正在重启并更新', res.message)
        return true
      }
      toast('error', '更新失败', res.message)
      return false
    } catch (err) {
      toast('error', '更新失败', errorMessage(err))
      return false
    }
  }

  async function discardUpdate(): Promise<void> {
    await backend.discardAppUpdate()
    await refreshUpdate()
  }

  async function refreshUpdate(): Promise<void> {
    try {
      update.value = await backend.getUpdateState()
    } catch {
      /* ignore */
    }
  }

  function setCloseRequest(count: number | null): void {
    closeRequest.value = count
  }

  async function confirmQuit(): Promise<void> {
    try {
      await backend.confirmQuit()
    } catch (err) {
      toast('error', '退出失败', errorMessage(err))
    }
  }

  async function openDataDir(): Promise<void> {
    const res = await backend.openDataDir()
    if (!res.ok) toast('error', '打开目录失败', res.message)
  }

  async function openUpdateLog(): Promise<void> {
    const res = await backend.openUpdateLog()
    if (!res.ok) toast('warn', '更新日志', res.message)
  }

  async function openReleasePage(): Promise<void> {
    const res = await backend.openReleasePage()
    if (!res.ok) toast('info', '开发构建', res.message)
  }

  async function openNpcReleasePage(): Promise<void> {
    await backend.openNpcReleasePage()
  }

  async function openExternal(url: string): Promise<void> {
    await backend.openExternal(url)
  }

  async function getDiagnostics(): Promise<string> {
    try {
      return await backend.getDiagnostics()
    } catch (err) {
      return errorMessage(err)
    }
  }

  return {
    bootstrapped,
    loading,
    fatal,
    appInfo,
    settings,
    binary,
    binaryCandidates,
    binaryProgress,
    binaryDownloading,
    update,
    updateProgress,
    warnings,
    toasts,
    closeRequest,
    setupSkipped,
    firstRun,
    theme,
    version,
    updateAvailable,
    updateReady,
    needsSetup,
    toast,
    dismissToast,
    load,
    refresh,
    saveSettings,
    detectNpcBinaries,
    pickNpcBinary,
    setNpcBinary,
    downloadNpc,
    cancelNpcDownload,
    checkUpdate,
    downloadUpdate,
    cancelUpdate,
    applyUpdate,
    discardUpdate,
    refreshUpdate,
    setCloseRequest,
    confirmQuit,
    openDataDir,
    openUpdateLog,
    openReleasePage,
    openNpcReleasePage,
    openExternal,
    getDiagnostics,
  }
})
