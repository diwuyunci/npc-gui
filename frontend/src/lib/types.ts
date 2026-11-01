export type RunState =
  | 'stopped'
  | 'starting'
  | 'running'
  | 'online'
  | 'reconnecting'
  | 'stopping'
  | 'error'

export type UpdateState =
  | 'disabled'
  | 'idle'
  | 'checking'
  | 'available'
  | 'downloading'
  | 'ready'
  | 'applying'
  | 'error'

export type LogLevel = 'trace' | 'debug' | 'info' | 'warn' | 'error' | 'fatal' | 'panic'

export interface AppInfo {
  productName: string
  version: string
  commit: string
  buildDate: string
  repo?: string
  os: string
  arch: string
  updateEnabled: boolean
  dataDir: string
}

export interface Profile {
  id: string
  name: string
  server: string
  vkey: string
  connType: string
  localIp?: string
  proxyUrl?: string
  dnsServer?: string
  ntpServer?: string
  ntpInterval?: number
  autoReconnection: boolean
  disconnectTimeout?: number
  extraCommon?: string
  extraSections?: string
  autoStart?: boolean
  logLevel?: string
  crypt?: boolean
  compress?: boolean
  maxConn?: number
  rateLimit?: number
  flowLimit?: number
  remark?: string
  basicUsername?: string
  basicPassword?: string
  webUsername?: string
  webPassword?: string
  pprofAddr?: string
  protoVersion?: number | null
  skipTlsVerify?: boolean
  disableP2p?: boolean
  keepalive?: number
  createdAt: number
  updatedAt: number
}

export interface Settings {
  npcBinaryPath?: string
  downloadMirror?: string
  autoCheckUpdates: boolean
  theme: string
  locale: string
  defaultLogLevel: string
  maxLogLines: number
  confirmOnQuit: boolean
  logToDisk: boolean
  launchAtLogin: boolean
  trayEnabled: boolean
  minimizeToTray: boolean
  notifications: boolean
  lastUpdateCheck?: number
}

export interface BinaryInfo {
  path: string
  version?: string
  coreVersion?: string
  source?: string
  sha256?: string
  tag?: string
  managed: boolean
  installedAt?: number
}

export interface RuntimeInfo {
  profileId: string
  state: RunState
  pid?: number
  startedAt?: number
  onlineSince?: number
  lastLogAt?: number
  lastError?: string
  reconnects: number
  configHash?: string
  binaryVersion?: string
  pendingRestart: boolean
}

export interface LogEntry {
  seq: number
  profileId?: string
  time: number
  level: LogLevel | string
  text: string
  stream?: string
}

export interface LogPage {
  profileId?: string
  entries: LogEntry[]
  lastSeq: number
  truncated: boolean
}

export interface OperationResult {
  ok: boolean
  code?: string
  message?: string
  profileId?: string
  fieldErrors?: Record<string, string>
  warnings?: string[]
}

export interface UpdateInfo {
  version: string
  tag: string
  name?: string
  notes?: string
  releaseUrl?: string
  assetName?: string
  assetSize?: number
  downloadUrl?: string
  sha256?: string
  publishedAt?: number
}

export interface UpdateProgress {
  stage: string
  received: number
  total: number
  percent: number
  bytesPerSec?: number
}

export interface UpdateStateInfo {
  state: UpdateState
  currentVersion: string
  latest?: UpdateInfo | null
  progress?: UpdateProgress | null
  error?: string
  checkedAt?: number
  stagedVersion?: string
}

export interface UpdateStateResult {
  state: UpdateStateInfo
  result: OperationResult
}

export interface UpdateActionResult {
  ok: boolean
  message?: string
}

export interface ConfigPreview {
  path: string
  content: string
  command: string
}

export interface ImportCandidate {
  profile: Profile
  selected: boolean
  note?: string
}

export interface ImportPreview {
  candidates: ImportCandidate[]
  warnings: string[]
  errors: string[]
}

export interface Bootstrap {
  app: AppInfo
  profiles: Profile[]
  runtimes: RuntimeInfo[]
  settings: Settings
  binary?: BinaryInfo | null
  binaryCandidates: BinaryInfo[]
  update: UpdateStateInfo
  warnings: string[]
  firstRun: boolean
}

export interface BinaryResult {
  binary?: BinaryInfo | null
  candidates: BinaryInfo[]
  result: OperationResult
}

export interface ProfileSaveResult {
  profile: Profile
  pendingRestart: boolean
  result: OperationResult
}

export interface ProfilesSaveResult {
  profiles: Profile[]
  result: OperationResult
}

export interface SettingsSaveResult {
  settings: Settings
  result: OperationResult
}

export interface StringResult {
  value?: string
  result: OperationResult
}

export interface Toast {
  id: number
  level: 'info' | 'success' | 'warn' | 'error'
  message: string
  detail?: string
}

export interface Backend {
  getBootstrap(): Promise<Bootstrap>
  saveProfile(profile: Profile): Promise<ProfileSaveResult>
  deleteProfile(id: string): Promise<OperationResult>
  duplicateProfile(id: string): Promise<ProfileSaveResult>
  importCommand(raw: string): Promise<ImportPreview>
  commitImport(profiles: Profile[]): Promise<ProfilesSaveResult>
  importProfiles(): Promise<ProfilesSaveResult>
  exportProfiles(ids: string[], includeSecrets: boolean): Promise<StringResult>
  startProfile(id: string): Promise<OperationResult>
  stopProfile(id: string): Promise<OperationResult>
  restartProfile(id: string): Promise<OperationResult>
  startAll(): Promise<OperationResult[]>
  stopAll(): Promise<OperationResult[]>
  getLogs(id: string, afterSeq: number, limit: number): Promise<LogPage>
  getAllRecentLogs(limit: number): Promise<LogEntry[]>
  clearLogs(id: string): Promise<OperationResult>
  exportLogs(id: string): Promise<StringResult>
  previewConfig(id: string, reveal: boolean): Promise<ConfigPreview>
  exportConfig(id: string): Promise<StringResult>
  getSettings(): Promise<Settings>
  saveSettings(settings: Settings): Promise<SettingsSaveResult>
  detectNpcBinaries(): Promise<BinaryInfo[]>
  validateNpcBinary(path: string): Promise<BinaryResult>
  setNpcBinary(path: string): Promise<BinaryResult>
  pickNpcBinary(): Promise<BinaryResult>
  downloadNpc(): Promise<BinaryResult>
  cancelNpcDownload(): Promise<OperationResult>
  checkAppUpdate(force: boolean): Promise<UpdateStateResult>
  downloadAppUpdate(): Promise<UpdateStateResult>
  cancelAppUpdate(): Promise<OperationResult>
  discardAppUpdate(): Promise<OperationResult>
  applyAppUpdate(stopAll: boolean): Promise<UpdateActionResult>
  getUpdateState(): Promise<UpdateStateInfo>
  openUpdateLog(): Promise<OperationResult>
  openDataDir(): Promise<OperationResult>
  openExternal(url: string): Promise<OperationResult>
  openReleasePage(): Promise<OperationResult>
  openNpcReleasePage(): Promise<OperationResult>
  getDiagnostics(): Promise<string>
  confirmQuit(): Promise<void>
}

export function emptyProfile(): Profile {
  const now = Date.now()
  return {
    id: '',
    name: '',
    server: '',
    vkey: '',
    connType: 'tls',
    dnsServer: '8.8.8.8',
    ntpInterval: 5,
    autoReconnection: true,
    logLevel: 'info',
    createdAt: now,
    updatedAt: now,
  }
}

export function clone<T>(value: T): T {
  return JSON.parse(JSON.stringify(value)) as T
}
