import type {
  Backend,
  BinaryInfo,
  BinaryResult,
  Bootstrap,
  ConfigPreview,
  ImportCandidate,
  ImportPreview,
  LogEntry,
  LogPage,
  OperationResult,
  Profile,
  ProfileSaveResult,
  ProfilesSaveResult,
  RuntimeInfo,
  Settings,
  SettingsSaveResult,
  StringResult,
  UpdateActionResult,
  UpdateInfo,
  UpdateStateInfo,
  UpdateStateResult,
} from './types'
import { clone } from './types'
import { mockEmit } from './mockBus'

const now = Date.now()
const sleep = (ms: number) => new Promise<void>((resolve) => setTimeout(resolve, ms))
const ok = (warnings: string[] = []): OperationResult => ({ ok: true, warnings })
const fail = (message: string, code = 'mock'): OperationResult => ({ ok: false, code, message })

function id(prefix = 'demo'): string {
  return prefix + '-' + Math.random().toString(36).slice(2, 8)
}

const profiles: Profile[] = [
  {
    id: 'demo-hq',
    name: '公司跳板',
    server: 'nps.example.com:8024',
    vkey: '8f3c2a91d7e04b5c',
    connType: 'tls',
    dnsServer: '8.8.8.8',
    ntpInterval: 5,
    autoReconnection: true,
    disconnectTimeout: 30,
    autoStart: true,
    logLevel: 'info',
    remark: '办公区出口',
    createdAt: now - 86400000 * 12,
    updatedAt: now - 3600000,
  },
  {
    id: 'demo-home',
    name: '家庭 NAS',
    server: 'home.example.net:8024',
    vkey: 'a1b2c3d4e5f60718',
    connType: 'ws',
    dnsServer: '8.8.8.8',
    ntpInterval: 5,
    autoReconnection: true,
    disconnectTimeout: 30,
    logLevel: 'info',
    remark: '群晖 DSM 反向代理',
    createdAt: now - 86400000 * 5,
    updatedAt: now - 7200000,
  },
  {
    id: 'demo-lab',
    name: '测试环境',
    server: 'lab.example.org:9100',
    vkey: '9988776655443322',
    connType: 'kcp',
    dnsServer: '1.1.1.1',
    ntpInterval: 5,
    autoReconnection: true,
    disconnectTimeout: 15,
    logLevel: 'debug',
    createdAt: now - 86400000 * 2,
    updatedAt: now - 1800000,
  },
]

const runtimes: Record<string, RuntimeInfo> = {
  'demo-hq': {
    profileId: 'demo-hq',
    state: 'online',
    pid: 4821,
    startedAt: now - 754000,
    onlineSince: now - 748000,
    lastLogAt: now - 12000,
    reconnects: 1,
    binaryVersion: '0.34.10',
    pendingRestart: false,
  },
  'demo-home': {
    profileId: 'demo-home',
    state: 'stopped',
    reconnects: 0,
    pendingRestart: false,
  },
  'demo-lab': {
    profileId: 'demo-lab',
    state: 'error',
    reconnects: 3,
    lastError: 'Failed to connect: dial tcp 10.0.0.9:9100: i/o timeout',
    pendingRestart: true,
  },
}

const logs: Record<string, LogEntry[]> = {
  'demo-hq': [
    logEntry('demo-hq', 'info', 'Loading configuration file /Users/demo/Library/Application Support/npc-gui/profiles/demo-hq/npc.conf successfully'),
    logEntry('demo-hq', 'info', 'the version of client is 0.34.10, the core version of client is 0.34.0'),
    logEntry('demo-hq', 'info', 'Successful connection with server nps.example.com:8024'),
    logEntry('demo-hq', 'info', 'web access login username:user password:demo'),
    logEntry('demo-hq', 'info', 'P2P local port 2000: punched public path public.example.com:39012 -> 10.0.0.12:2000'),
    logEntry('demo-hq', 'warn', 'P2P direct path unavailable, falling back to secret relay'),
    logEntry('demo-hq', 'info', 'tunnel ready: 4 hosts, 2 tasks'),
  ],
  'demo-home': [
    logEntry('demo-home', 'info', 'Client closed!'),
  ],
  'demo-lab': [
    logEntry('demo-lab', 'info', 'Start server: lab.example.org:9100 vkey: 9988**** type: kcp'),
    logEntry('demo-lab', 'error', 'Failed to connect: dial tcp 10.0.0.9:9100: i/o timeout'),
    logEntry('demo-lab', 'warn', 'The connection server failed and will be reconnected in five seconds.'),
    logEntry('demo-lab', 'error', 'The connection server failed and will be reconnected in five seconds, error dial tcp 10.0.0.9:9100: i/o timeout'),
  ],
}

for (const list of Object.values(logs)) {
  list.forEach((entry, index) => {
    entry.seq = index + 1
  })
}

let seq = Object.fromEntries(Object.entries(logs).map(([k, v]) => [k, v.length])) as Record<string, number>

function logEntry(profileId: string, level: string, text: string, timestamp = Date.now()): LogEntry {
  return { seq: 0, profileId, time: timestamp, level, text }
}

function nextEntry(profileId: string, level: string, text: string): LogEntry {
  seq[profileId] = (seq[profileId] ?? 0) + 1
  return { seq: seq[profileId], profileId, time: Date.now(), level, text }
}

const timers = new Map<string, number>()

function emitState(profileId: string): void {
  const runtime = runtimes[profileId]
  if (runtime) mockEmit('runtime:state', clone(runtime))
}

function emitLog(profileId: string, level: string, text: string): void {
  const entry = nextEntry(profileId, level, text)
  const list = logs[profileId] ?? []
  list.push(entry)
  if (list.length > 5000) list.splice(0, list.length - 5000)
  logs[profileId] = list
  const page: LogPage = { profileId, entries: [entry], lastSeq: entry.seq, truncated: false }
  mockEmit('runtime:logs', page)
}

function setState(profileId: string, patch: Partial<RuntimeInfo>): void {
  const base = runtimes[profileId] ?? { profileId, state: 'stopped', reconnects: 0, pendingRestart: false }
  runtimes[profileId] = { ...base, ...patch }
  emitState(profileId)
}

function startTimer(profileId: string): void {
  clearTimer(profileId)
  const timer = window.setInterval(() => {
    const runtime = runtimes[profileId]
    if (!runtime || runtime.state !== 'online') return
    const messages = [
      'bridge ping ok',
      'P2P path healthy, fallback not required',
      'tunnel keepalive acknowledged',
      'received 1.2 KiB on tunnel',
    ]
    emitLog(profileId, 'debug', messages[Math.floor(Math.random() * messages.length)])
  }, 9000)
  timers.set(profileId, timer)
}

function clearTimer(profileId: string): void {
  const timer = timers.get(profileId)
  if (timer) {
    window.clearInterval(timer)
    timers.delete(profileId)
  }
}

async function startProfile(idValue: string): Promise<OperationResult> {
  const profile = profiles.find((p) => p.id === idValue)
  if (!profile) return fail('档案不存在')
  const runtime = runtimes[idValue]
  if (runtime && ['starting', 'running', 'online', 'reconnecting', 'stopping'].includes(runtime.state)) {
    return fail('该档案已经在运行', 'already_running')
  }
  setState(idValue, {
    state: 'starting',
    pid: Math.floor(4000 + Math.random() * 2000),
    startedAt: Date.now(),
    lastError: '',
    pendingRestart: false,
  })
  emitLog(idValue, 'info', 'Start server: ' + profile.server + ' type: ' + profile.connType)
  emitLog(idValue, 'info', 'Loading configuration file npc.conf successfully')
  const pid = runtimes[idValue]?.pid
  window.setTimeout(() => {
    if (runtimes[idValue]?.state !== 'starting') return
    setState(idValue, { state: 'running', pid, onlineSince: 0 })
    emitLog(idValue, 'info', 'connecting to bridge ...')
  }, 500)
  window.setTimeout(() => {
    if (runtimes[idValue]?.state !== 'running') return
    setState(idValue, { state: 'online', onlineSince: Date.now() })
    emitLog(idValue, 'info', 'Successful connection with server ' + profile.server)
    emitLog(idValue, 'info', 'web access login username:user password:demo')
    startTimer(idValue)
  }, 1300)
  return ok()
}

async function stopProfile(idValue: string): Promise<OperationResult> {
  const runtime = runtimes[idValue]
  if (!runtime || runtime.state === 'stopped') return ok()
  clearTimer(idValue)
  setState(idValue, { state: 'stopping' })
  emitLog(idValue, 'info', 'Client closed!')
  await sleep(300)
  setState(idValue, {
    state: 'stopped',
    pid: 0,
    startedAt: 0,
    onlineSince: 0,
    lastError: '',
    pendingRestart: false,
  })
  return ok()
}

let binary: BinaryInfo = {
  path: '/usr/local/bin/npc',
  version: '0.34.10',
  coreVersion: '0.34.0',
  source: 'PATH',
  sha256: 'b640c2a499a78ab106ba361e068391566ad2b5f8cb4dba009a8a8697171a8311',
  managed: false,
}

const updateInfo: UpdateInfo = {
  version: '0.2.0',
  tag: 'v0.2.0',
  name: 'NPC 客户端 v0.2.0',
  notes:
    '新增\n- 客户端自我更新与失败回滚\n- 日志搜索与导出\n- 更稳定的进程退出处理\n\n修复\n- 修复 macOS 应用包内二进制路径识别\n- 修复多档案同时停止时的状态错乱',
  releaseUrl: 'https://github.com/example/npc-gui/releases/tag/v0.2.0',
  assetName: 'npc-gui_0.2.0_darwin_arm64.app.zip',
  assetSize: 18_700_000,
  downloadUrl: 'https://github.com/example/npc-gui/releases/download/v0.2.0/npc-gui_0.2.0_darwin_arm64.app.zip',
  sha256: '7d3f0c4b2f0f3d9a6e2a5b1c8d7e6f5a4b3c2d1e0f9a8b7c6d5e4f3a2b1c0d9e',
  publishedAt: now - 3600000 * 7,
}

let updateState: UpdateStateInfo = {
  state: 'available',
  currentVersion: '0.1.0',
  latest: updateInfo,
  checkedAt: now - 120000,
}

const settings: Settings = {
  npcBinaryPath: binary.path,
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

const appInfo = {
  productName: 'NPC 客户端',
  version: '0.1.0',
  commit: 'mock',
  buildDate: '2026-10-31T08:00:00+08:00',
  repo: 'example/npc-gui',
  os: 'darwin',
  arch: 'arm64',
  updateEnabled: true,
  dataDir: '/Users/demo/Library/Application Support/npc-gui',
}

function bootstrap(): Bootstrap {
  return clone({
    app: appInfo,
    profiles,
    runtimes: Object.values(runtimes),
    settings,
    binary,
    binaryCandidates: [binary],
    update: updateState,
    warnings: [],
    firstRun: false,
  })
}

function mockConfig(profile: Profile, reveal = false): string {
  const vkey = reveal ? profile.vkey : '********'
  return [
    '# Generated by NPC GUI. Manual changes will be overwritten.',
    '[common]',
    'server_addr=' + profile.server,
    'conn_type=' + profile.connType,
    'vkey=' + vkey,
    'dns_server=' + (profile.dnsServer || '8.8.8.8'),
    'auto_reconnection=' + String(profile.autoReconnection),
  ].join('\n')
}

function parseImport(raw: string): ImportPreview {
  const tokens: string[] = []
  let current = ''
  let quote = ''
  for (const ch of raw.replace(/，/g, ',').replace(/：/g, ':')) {
    if (quote) {
      if (ch === quote) quote = ''
      else current += ch
    } else if (ch === '"' || ch === "'") {
      quote = ch
    } else if (/\s/.test(ch)) {
      if (current) {
        tokens.push(current)
        current = ''
      }
    } else {
      current += ch
    }
  }
  if (current) tokens.push(current)
  const values: Record<string, string> = {}
  for (let i = 0; i < tokens.length; i++) {
    const token = tokens[i]
    if (!token.startsWith('-')) continue
    const clean = token.replace(/^-+/, '')
    const eq = clean.indexOf('=')
    if (eq > 0) {
      values[clean.slice(0, eq)] = clean.slice(eq + 1)
    } else if (i + 1 < tokens.length && !tokens[i + 1].startsWith('-')) {
      values[clean.replace(/-/g, '_')] = tokens[++i]
    }
  }
  const server = values.server || values.s
  const vkey = values.vkey || values.k
  if (!server || !vkey) {
    return { candidates: [], warnings: [], errors: ['命令中缺少 -server 或 -vkey'] }
  }
  const types = (values.type || values.t || 'tcp').split(',')
  const servers = server.split(',')
  const candidates: ImportCandidate[] = servers.map((item, index) => {
    const p = {
      ...clone(profiles[0]),
      id: '',
      name: item.split('@')[0].replace(/:[0-9]+$/, '') || 'NPS 连接',
      server: item.trim(),
      vkey: vkey.split(',')[index] || vkey.split(',')[0],
      connType: (types[index] || types[0] || 'tcp').trim(),
      createdAt: 0,
      updatedAt: 0,
      autoStart: false,
    }
    return { profile: p, selected: true, note: '解析自启动命令' }
  })
  return {
    candidates,
    warnings: servers.length > 1 ? ['检测到多个服务端，将拆分为独立档案'] : [],
    errors: [],
  }
}

export const mockBackend: Backend = {
  async getBootstrap() {
    return bootstrap()
  },
  async saveProfile(p) {
    const copy = clone(p)
    const index = profiles.findIndex((item) => item.id === copy.id)
    const nowMs = Date.now()
    if (index >= 0) {
      copy.updatedAt = nowMs
      profiles[index] = copy
    } else {
      copy.id = id('profile')
      copy.createdAt = nowMs
      copy.updatedAt = nowMs
      profiles.push(copy)
      runtimes[copy.id] = { profileId: copy.id, state: 'stopped', reconnects: 0, pendingRestart: false }
    }
    mockEmit('profiles:changed', null)
    return { profile: clone(copy), pendingRestart: false, result: ok() } as ProfileSaveResult
  },
  async deleteProfile(profileId) {
    const idx = profiles.findIndex((item) => item.id === profileId)
    if (idx >= 0) profiles.splice(idx, 1)
    delete runtimes[profileId]
    delete logs[profileId]
    mockEmit('profiles:changed', null)
    return ok()
  },
  async duplicateProfile(profileId) {
    const p = profiles.find((item) => item.id === profileId)
    if (!p) return { profile: {} as Profile, pendingRestart: false, result: fail('档案不存在') }
    const copy = clone(p)
    copy.id = ''
    copy.name = p.name + ' 副本'
    return this.saveProfile(copy)
  },
  async importCommand(raw) {
    return parseImport(raw)
  },
  async commitImport(items) {
    const saved: Profile[] = []
    for (const item of items) {
      const copy = clone(item)
      copy.id = id('profile')
      copy.createdAt = Date.now()
      copy.updatedAt = Date.now()
      profiles.push(copy)
      runtimes[copy.id] = { profileId: copy.id, state: 'stopped', reconnects: 0, pendingRestart: false }
      saved.push(copy)
    }
    mockEmit('profiles:changed', null)
    return { profiles: clone(saved), result: ok() } as ProfilesSaveResult
  },
  async importProfiles() {
    return { profiles: [], result: fail('Mock 模式不读取本机文件') } as ProfilesSaveResult
  },
  async exportProfiles() {
    return { value: '/Users/demo/Downloads/npc-gui-profiles.json', result: ok() } as StringResult
  },
  async startProfile(profileId) {
    return startProfile(profileId)
  },
  async stopProfile(profileId) {
    return stopProfile(profileId)
  },
  async restartProfile(profileId) {
    await stopProfile(profileId)
    return startProfile(profileId)
  },
  async startAll() {
    const out: OperationResult[] = []
    for (const p of profiles) {
      if (runtimes[p.id]?.state === 'stopped' || !runtimes[p.id]) out.push(await startProfile(p.id))
    }
    return out
  },
  async stopAll() {
    const out: OperationResult[] = []
    for (const p of profiles) out.push(await stopProfile(p.id))
    return out
  },
  async getLogs(profileId, afterSeq, limit) {
    const list = (logs[profileId] ?? []).filter((entry) => entry.seq > afterSeq)
    const sliced = list.slice(-limit)
    return {
      profileId,
      entries: clone(sliced),
      lastSeq: list.length ? list[list.length - 1].seq : 0,
      truncated: list.length > limit,
    }
  },
  async getAllRecentLogs(limit) {
    const all = Object.values(logs).flat().sort((a, b) => a.time - b.time)
    return clone(all.slice(-limit))
  },
  async clearLogs(profileId) {
    logs[profileId] = []
    seq[profileId] = 0
    return ok()
  },
  async exportLogs() {
    return { value: '/Users/demo/Downloads/npc-log.log', result: ok() } as StringResult
  },
  async previewConfig(profileId, reveal) {
    const p = profiles.find((item) => item.id === profileId)
    return {
      path: '/Users/demo/Library/Application Support/npc-gui/profiles/' + profileId + '/npc.conf',
      content: p ? mockConfig(p, reveal) : '',
      command: 'npc -config=npc.conf -log=stdout -debug=false',
    } as ConfigPreview
  },
  async exportConfig() {
    return { value: '/Users/demo/Downloads/npc.conf', result: ok() } as StringResult
  },
  async getSettings() {
    return clone(settings)
  },
  async saveSettings(next) {
    Object.assign(settings, next)
    return { settings: clone(settings), result: ok() } as SettingsSaveResult
  },
  async detectNpcBinaries() {
    return [clone(binary)]
  },
  async validateNpcBinary() {
    return { binary, candidates: [binary], result: ok() } as BinaryResult
  },
  async setNpcBinary(pathValue) {
    binary = { ...binary, path: pathValue, source: '自定义', managed: false }
    return { binary, candidates: [binary], result: ok() } as BinaryResult
  },
  async pickNpcBinary() {
    return { binary, candidates: [binary], result: ok() } as BinaryResult
  },
  async downloadNpc() {
    for (let i = 0; i <= 20; i++) {
      await sleep(90)
      mockEmit('binary:progress', {
        stage: '下载 npc',
        received: i * 400000,
        total: 8000000,
        percent: i * 5,
      })
    }
    binary = { ...binary, managed: true, source: 'GUI 下载' }
    const result = { binary, candidates: [binary], result: ok() } as BinaryResult
    mockEmit('binary:info', result)
    return result
  },
  async cancelNpcDownload() {
    return ok()
  },
  async checkAppUpdate() {
    updateState = { ...updateState, state: 'checking', error: '' }
    mockEmit('update:state', clone(updateState))
    await sleep(650)
    updateState = {
      ...updateState,
      state: 'available',
      currentVersion: appInfo.version,
      latest: updateInfo,
      checkedAt: Date.now(),
    }
    mockEmit('update:state', clone(updateState))
    return { state: clone(updateState), result: ok() } as UpdateStateResult
  },
  async downloadAppUpdate() {
    updateState = { ...updateState, state: 'downloading', progress: { stage: '下载更新', received: 0, total: updateInfo.assetSize ?? 0, percent: 0 } }
    mockEmit('update:state', clone(updateState))
    for (let i = 0; i <= 40; i++) {
      await sleep(60)
      const progress = { stage: '下载更新', received: (updateInfo.assetSize ?? 0) * i / 40, total: updateInfo.assetSize ?? 0, percent: i * 2.5 }
      updateState = { ...updateState, progress }
      mockEmit('update:progress', progress)
    }
    updateState = { ...updateState, state: 'ready', progress: null, stagedVersion: updateInfo.version }
    mockEmit('update:state', clone(updateState))
    return { state: clone(updateState), result: ok() } as UpdateStateResult
  },
  async cancelAppUpdate() {
    return ok()
  },
  async discardAppUpdate() {
    updateState = { ...updateState, state: 'available', progress: null, stagedVersion: '' }
    mockEmit('update:state', clone(updateState))
    return ok()
  },
  async applyAppUpdate() {
    updateState = { ...updateState, state: 'applying' }
    mockEmit('update:state', clone(updateState))
    return { ok: true, message: 'Mock 模式下不会真正重启' } as UpdateActionResult
  },
  async getUpdateState() {
    return clone(updateState)
  },
  async openUpdateLog() {
    return ok()
  },
  async openDataDir() {
    return ok()
  },
  async openExternal() {
    return ok()
  },
  async openReleasePage() {
    return ok()
  },
  async openNpcReleasePage() {
    return ok()
  },
  async getDiagnostics() {
    return 'NPC GUI 0.1.0 (mock)\nprofiles: 3\nnpc: 0.34.10'
  },
  async confirmQuit() {
    return
  },
}