<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import {
  ArrowRight,
  Check,
  CircleAlert,
  Download,
  FileSearch,
  FolderOpen,
  KeyRound,
  Radio,
  ScanLine,
  ShieldCheck,
  Sparkles,
  TerminalSquare,
  Wifi,
} from 'lucide-vue-next'
import type { BinaryInfo, ImportPreview, Profile } from '@/lib/types'
import { emptyProfile } from '@/lib/types'
import { backend, errorMessage } from '@/lib/backend'
import { useAppStore } from '@/stores/app'
import { useProfilesStore } from '@/stores/profiles'
import BrandMark from '@/components/layout/BrandMark.vue'
import FormField from '@/components/ui/FormField.vue'
import ProgressBar from '@/components/ui/ProgressBar.vue'
import CommandFill from '@/components/profiles/CommandFill.vue'

const emit = defineEmits<{ 'skip-setup': [] }>()

const app = useAppStore()
const profiles = useProfilesStore()
const router = useRouter()

const step = ref(1)
const mode = ref<'manual' | 'command'>('manual')
const draft = ref<Profile>(emptyProfile())
const command = ref('')
const preview = ref<ImportPreview | null>(null)
const candidates = ref<BinaryInfo[]>([])
const saving = ref(false)
const parsing = ref(false)
const createdId = ref('')
const startNow = ref(true)
const errors = ref<Record<string, string>>({})

const binaryReady = computed(() => Boolean(app.binary?.path))
const existingProfiles = computed(() => profiles.items.length)
const createdProfile = computed(() => profiles.items.find((p) => p.id === createdId.value) || profiles.items[0] || null)
const progress = computed(() => app.binaryProgress)

const stepLabel = computed(() => {
  if (step.value === 1) return '准备 npc'
  if (step.value === 2) return '添加连接'
  return '完成'
})

onMounted(() => {
  if (app.binary?.path) {
    step.value = existingProfiles.value ? 3 : 2
    createdId.value = profiles.items[0]?.id || ''
  }
  if (!draft.value.name) draft.value.name = '我的第一台服务器'
})

async function detect(): Promise<void> {
  candidates.value = await app.detectNpcBinaries()
  if (!candidates.value.length) app.toast('warn', '没有自动找到 npc', '请手动选择文件或下载最新版')
}

async function pick(): Promise<void> {
  await app.pickNpcBinary()
  if (binaryReady.value) step.value = existingProfiles.value ? 3 : 2
}

async function download(): Promise<void> {
  const ok = await app.downloadNpc()
  if (ok) {
    step.value = existingProfiles.value ? 3 : 2
    app.toast('success', 'npc 已就绪', app.binary?.version)
  }
}

async function selectCandidate(item: BinaryInfo): Promise<void> {
  const ok = await app.setNpcBinary(item.path)
  if (ok) {
    step.value = existingProfiles.value ? 3 : 2
  }
}

function nextFromBinary(): void {
  if (!binaryReady.value) {
    app.toast('warn', '请先准备 npc')
    return
  }
  step.value = existingProfiles.value ? 3 : 2
  createdId.value = profiles.items[0]?.id || createdId.value
}

function applySetupCommand(p: Profile): void {
  const d = draft.value
  if (!d.name.trim()) d.name = p.name
  d.server = p.server
  d.vkey = p.vkey
  d.connType = p.connType
  d.autoReconnection = p.autoReconnection
  if (p.proxyUrl) d.proxyUrl = p.proxyUrl
  if (p.localIp) d.localIp = p.localIp
  if (p.dnsServer) d.dnsServer = p.dnsServer
  if (p.ntpServer) d.ntpServer = p.ntpServer
  if (p.ntpInterval) d.ntpInterval = p.ntpInterval
  if (p.disconnectTimeout) d.disconnectTimeout = p.disconnectTimeout
  if (p.keepalive) d.keepalive = p.keepalive
  if (p.protoVersion !== null && p.protoVersion !== undefined) d.protoVersion = p.protoVersion
  d.skipTlsVerify = p.skipTlsVerify ?? d.skipTlsVerify
  d.disableP2p = p.disableP2p ?? d.disableP2p
  if (p.logLevel) d.logLevel = p.logLevel
  errors.value = {}
}

async function saveManual(): Promise<void> {
  errors.value = {}
  if (!draft.value.name.trim()) errors.value.name = '请填写名称'
  if (!draft.value.server.trim()) errors.value.server = '请填写服务端地址'
  if (!draft.value.vkey.trim()) errors.value.vkey = '请填写 vkey'
  if (Object.keys(errors.value).length) return
  if (!binaryReady.value) {
    app.toast('warn', '请先准备 npc 二进制')
    step.value = 1
    return
  }
  saving.value = true
  try {
    const result = await profiles.save(draft.value)
    if (result.result.ok) {
      createdId.value = result.profile.id
      step.value = 3
    } else {
      errors.value = result.result.fieldErrors ?? {}
      app.toast('error', '保存连接失败', result.result.message)
    }
  } catch (err) {
    app.toast('error', '保存连接失败', errorMessage(err))
  } finally {
    saving.value = false
  }
}

async function parseCommand(): Promise<void> {
  if (!command.value.trim()) {
    app.toast('warn', '请先粘贴 NPS 启动命令')
    return
  }
  parsing.value = true
  try {
    preview.value = await backend.importCommand(command.value)
    if (preview.value.errors.length) app.toast('error', '命令解析失败', preview.value.errors[0])
  } catch (err) {
    app.toast('error', '命令解析失败', errorMessage(err))
  } finally {
    parsing.value = false
  }
}

async function importCommand(): Promise<void> {
  if (!preview.value?.candidates.length) {
    await parseCommand()
    return
  }
  if (!binaryReady.value) {
    app.toast('warn', '请先准备 npc 二进制')
    step.value = 1
    return
  }
  saving.value = true
  try {
    const result = await profiles.commitImport(preview.value.candidates.map((candidate) => candidate.profile))
    if (result.result.ok) {
      createdId.value = result.profiles[0]?.id || ''
      step.value = 3
    } else {
      app.toast('error', '导入失败', result.result.message)
    }
  } catch (err) {
    app.toast('error', '导入失败', errorMessage(err))
  } finally {
    saving.value = false
  }
}

async function finish(): Promise<void> {
  app.setupSkipped = true
  if (startNow.value && createdProfile.value) {
    try {
      const result = await backend.startProfile(createdProfile.value.id)
      if (!result.ok) app.toast('warn', '连接未能启动', result.message)
    } catch (err) {
      app.toast('warn', '连接未能启动', errorMessage(err))
    }
  }
  await router.push('/')
}

function skip(): void {
  app.setupSkipped = true
  emit('skip-setup')
  void router.push('/')
}
</script>

<template>
  <div class="setup">
    <section class="setup-visual">
      <div class="visual-grid" />
      <div class="visual-top">
        <BrandMark :size="42" with-glow />
        <span class="visual-brand">NPC 客户端</span>
      </div>

      <div class="visual-copy">
        <div class="kicker">nps enhanced · desktop</div>
        <h1>把内网服务<br /><span>带到公网。</span></h1>

        <div class="visual-points">
          <div class="visual-point">
            <span class="point-icon"><Radio :size="16" /></span>
            <div><strong>多连接并行</strong><small>每个档案一个独立 npc 进程</small></div>
          </div>
          <div class="visual-point">
            <span class="point-icon"><TerminalSquare :size="16" /></span>
            <div><strong>日志实时可见</strong><small>桥接、重连与错误不再藏在终端里</small></div>
          </div>
          <div class="visual-point">
            <span class="point-icon"><ShieldCheck :size="16" /></span>
            <div><strong>密钥不进 argv</strong><small>配置模式启动，vkey 只落在本地文件</small></div>
          </div>
        </div>
      </div>

      <div class="visual-foot mono">
        <span>npc · supervisor</span>
        <span>v{{ app.version }}</span>
      </div>
    </section>

    <section class="setup-panel">
      <div class="setup-card">
        <button class="skip-link" type="button" @click="skip">稍后设置</button>

        <div class="stepper">
          <div v-for="n in 3" :key="n" class="step" :class="{ active: step === n, done: step > n }">
            <span class="step-dot">
              <Check v-if="step > n" :size="11" />
              <span v-else>{{ n }}</span>
            </span>
            <span class="step-text">
              {{ n === 1 ? '准备 npc' : n === 2 ? '添加连接' : '完成' }}
            </span>
            <span v-if="n < 3" class="step-line" />
          </div>
        </div>

        <div v-if="step === 1" class="step-panel">
          <div class="step-head">
            <div class="step-kicker">step 01</div>
            <h2>先准备 npc</h2>
            <p>客户端负责管理进程，但实际的穿透协议由官方 npc 执行。可以自动下载，或使用本机已有文件。</p>
          </div>

          <div v-if="binaryReady" class="binary-ready">
            <span class="ready-icon"><Check :size="18" /></span>
            <div class="ready-copy">
              <strong>npc {{ app.binary?.version }} 已就绪</strong>
              <span class="mono ellipsis">{{ app.binary?.path }}</span>
            </div>
            <button class="btn btn-primary btn-sm" type="button" @click="nextFromBinary">
              下一步 <ArrowRight :size="13" />
            </button>
          </div>

          <template v-else>
            <div class="binary-options">
              <button class="binary-option primary" type="button" :disabled="app.binaryDownloading" @click="download">
                <span class="option-icon"><Download :size="18" /></span>
                <span class="option-copy">
                  <strong>下载最新 npc</strong>
                  <small>来自上游 GitHub Releases，自动校验 SHA-256</small>
                </span>
                <span v-if="app.binaryDownloading" class="spinner" />
                <ArrowRight v-else :size="16" />
              </button>
              <button class="binary-option" type="button" :disabled="app.binaryDownloading" @click="pick">
                <span class="option-icon"><FolderOpen :size="18" /></span>
                <span class="option-copy">
                  <strong>选择本地文件</strong>
                  <small>已经在使用 npc？直接指向它</small>
                </span>
                <ArrowRight :size="16" />
              </button>
              <button class="binary-option" type="button" :disabled="app.binaryDownloading" @click="detect">
                <span class="option-icon"><FileSearch :size="18" /></span>
                <span class="option-copy">
                  <strong>自动检测</strong>
                  <small>检查 PATH 和常见安装位置</small>
                </span>
                <ArrowRight :size="16" />
              </button>
            </div>

            <div v-if="app.binaryDownloading" class="download-box">
              <div class="download-row">
                <span>{{ progress?.stage || '下载 npc' }}</span>
                <span class="mono">{{ progress?.percent?.toFixed(0) || 0 }}%</span>
              </div>
              <ProgressBar :value="progress?.percent || 0" />
            </div>

            <div v-if="candidates.length" class="detected">
              <span class="kicker">detected binaries</span>
              <button
                v-for="item in candidates"
                :key="item.path"
                class="detected-item"
                type="button"
                @click="selectCandidate(item)"
              >
                <span class="dot dot-online" />
                <span class="mono ellipsis">{{ item.path }}</span>
                <span class="badge">v{{ item.version }}</span>
              </button>
            </div>
          </template>

          <div class="step-tip">
            <ShieldCheck :size="14" />
            <span>npc 以独立子进程运行；更新客户端不会替换你正在使用的 npc 版本。</span>
          </div>
        </div>

        <div v-else-if="step === 2" class="step-panel">
          <div class="step-head">
            <div class="step-kicker">step 02</div>
            <h2>添加第一个连接</h2>
            <p>从 NPS 管理台的客户端页面复制启动命令，或手动填写服务端地址、vkey 和连接方式。</p>
          </div>

          <div class="mode-tabs">
            <button class="mode-tab" :class="{ active: mode === 'manual' }" type="button" @click="mode = 'manual'">
              <Wifi :size="14" /> 手动填写
            </button>
            <button class="mode-tab" :class="{ active: mode === 'command' }" type="button" @click="mode = 'command'">
              <ScanLine :size="14" /> 粘贴命令
            </button>
          </div>

          <div v-if="mode === 'manual'" class="manual-panel">
            <CommandFill @parsed="applySetupCommand" />
            <div class="form-grid">
            <FormField label="档案名称" :error="errors.name" html-for="setup-name">
              <input id="setup-name" v-model="draft.name" class="input" placeholder="例如：公司跳板" maxlength="64" />
            </FormField>
            <FormField label="连接方式" :error="errors.connType" html-for="setup-type">
              <select id="setup-type" v-model="draft.connType" class="select">
                <option value="tcp">TCP</option>
                <option value="tls">TLS</option>
                <option value="kcp">KCP</option>
                <option value="quic">QUIC</option>
                <option value="ws">WebSocket</option>
                <option value="wss">WebSocket + TLS</option>
              </select>
            </FormField>
            <FormField class="span-2" label="服务端地址" :error="errors.server" html-for="setup-server">
              <input id="setup-server" v-model="draft.server" class="input mono" placeholder="nps.example.com:8024" spellcheck="false" />
            </FormField>
            <FormField class="span-2" label="vkey" :error="errors.vkey" html-for="setup-vkey" hint="从 NPS 客户端页面复制，保存在本地配置文件中。">
              <div class="input-affix">
                <KeyRound class="field-icon" :size="15" />
                <input id="setup-vkey" v-model="draft.vkey" class="input mono" type="password" placeholder="客户端密钥" spellcheck="false" autocomplete="off" />
              </div>
            </FormField>
            </div>
          </div>

          <div v-else class="command-panel">
            <label class="label" for="setup-command">NPS 启动命令</label>
            <textarea
              id="setup-command"
              v-model="command"
              class="textarea mono"
              rows="4"
              spellcheck="false"
              placeholder="npc -server=xxx:123 -vkey=xxx -type=tls"
            />
            <div class="row">
              <button class="btn btn-secondary btn-sm" type="button" :disabled="parsing" @click="parseCommand">
                <span v-if="parsing" class="spinner" />
                <Sparkles v-else :size="13" />
                解析命令
              </button>
              <span v-if="preview" class="hint">{{ preview.candidates.length }} 个连接</span>
            </div>
            <div v-if="preview?.errors.length" class="error-box">
              <CircleAlert :size="14" />
              <span>{{ preview.errors[0] }}</span>
            </div>
            <div v-if="preview?.warnings.length" class="warn-box">
              <span>{{ preview.warnings[0] }}</span>
            </div>
            <div v-if="preview?.candidates.length" class="parsed-list">
              <div v-for="candidate in preview.candidates" :key="candidate.profile.server" class="parsed-item">
                <span class="dot dot-online" />
                <span class="parsed-main">
                  <strong>{{ candidate.profile.name }}</strong>
                  <small class="mono">{{ candidate.profile.server }}</small>
                </span>
                <span class="badge">{{ candidate.profile.connType.toUpperCase() }}</span>
              </div>
            </div>
          </div>

          <div class="step-actions">
            <button class="btn btn-ghost" type="button" @click="step = 1">
              <ArrowRight :size="13" class="back-icon" /> 上一步
            </button>
            <button
              v-if="mode === 'manual'"
              class="btn btn-primary"
              type="button"
              :disabled="saving"
              @click="saveManual"
            >
              <span v-if="saving" class="spinner" />
              保存并继续
              <ArrowRight v-if="!saving" :size="14" />
            </button>
            <button
              v-else
              class="btn btn-primary"
              type="button"
              :disabled="saving || !preview?.candidates.length"
              @click="importCommand"
            >
              <span v-if="saving" class="spinner" />
              导入并继续
              <ArrowRight v-if="!saving" :size="14" />
            </button>
          </div>
        </div>

        <div v-else class="step-panel finish-panel">
          <div class="finish-mark">
            <span class="finish-ring"><Check :size="26" /></span>
          </div>
          <div class="step-head centered">
            <div class="step-kicker">ready</div>
            <h2>可以出发了</h2>
            <p>配置已保存。启动后可以随时在总览查看连接状态和日志。</p>
          </div>

          <div class="summary">
            <div class="summary-row">
              <span>npc 核心</span>
              <span class="mono">{{ app.binary?.version || '未配置' }}</span>
            </div>
            <div class="summary-row">
              <span>连接档案</span>
              <span class="mono">{{ createdProfile?.name || '—' }}</span>
            </div>
            <div class="summary-row">
              <span>服务端</span>
              <span class="mono">{{ createdProfile?.server || '—' }}</span>
            </div>
          </div>

          <label class="start-option">
            <span class="start-copy">
              <strong>完成后立即启动</strong>
              <small>应用会打开总览并启动这个连接</small>
            </span>
            <input v-model="startNow" type="checkbox" class="native-check" />
          </label>

          <div class="step-actions">
            <button class="btn btn-ghost" type="button" @click="step = 2">返回修改</button>
            <button class="btn btn-primary btn-lg" type="button" @click="finish">
              完成并进入总览 <ArrowRight :size="15" />
            </button>
          </div>
        </div>

        <div class="setup-foot">
          <span>{{ stepLabel }}</span>
          <button class="text-link" type="button" @click="skip">跳过引导</button>
        </div>
      </div>
    </section>
  </div>
</template>

<style scoped>
.setup {
  min-height: 100vh;
  display: grid;
  grid-template-columns: minmax(380px, 1.02fr) minmax(520px, 0.98fr);
  background: var(--bg);
  overflow: auto;
}

.setup-visual {
  position: relative;
  display: flex;
  flex-direction: column;
  min-height: 100vh;
  padding: 46px 54px 34px;
  overflow: hidden;
  border-right: 1px solid var(--border-soft);
  background:
    radial-gradient(720px 500px at 12% 8%, rgba(86, 224, 192, 0.12), transparent 64%),
    radial-gradient(600px 620px at 100% 100%, rgba(141, 162, 255, 0.1), transparent 62%),
    #070b10;
}

.visual-grid {
  position: absolute;
  inset: 0;
  background-image:
    linear-gradient(rgba(120, 160, 190, 0.045) 1px, transparent 1px),
    linear-gradient(90deg, rgba(120, 160, 190, 0.045) 1px, transparent 1px);
  background-size: 42px 42px;
  mask-image: radial-gradient(ellipse at 30% 20%, black, transparent 78%);
}

.visual-top {
  position: relative;
  display: flex;
  align-items: center;
  gap: 11px;
}

.visual-brand {
  font-size: 14px;
  font-weight: 650;
  letter-spacing: -0.01em;
}

.visual-copy {
  position: relative;
  margin-top: auto;
  margin-bottom: auto;
  max-width: 480px;
}

.visual-copy h1 {
  margin-top: 13px;
  font-size: clamp(38px, 4.4vw, 62px);
  line-height: 1.04;
  letter-spacing: -0.055em;
  font-weight: 760;
}

.visual-copy h1 span {
  color: var(--accent);
}

.visual-points {
  display: flex;
  flex-direction: column;
  gap: 12px;
  margin-top: 32px;
}

.visual-point {
  display: flex;
  align-items: center;
  gap: 11px;
}

.point-icon {
  display: grid;
  place-items: center;
  width: 30px;
  height: 30px;
  flex: none;
  border: 1px solid rgba(86, 224, 192, 0.16);
  border-radius: 8px;
  background: rgba(86, 224, 192, 0.08);
  color: var(--accent);
}

.visual-point div {
  display: flex;
  flex-direction: column;
  gap: 1px;
}

.visual-point strong {
  font-size: 12px;
  font-weight: 600;
}

.visual-point small {
  color: #7d92a6;
  font-size: 10.5px;
}

.visual-foot {
  position: relative;
  display: flex;
  justify-content: space-between;
  color: #465b6f;
  font-size: 10px;
  padding-top: 16px;
  border-top: 1px solid rgba(120, 160, 190, 0.1);
}

.setup-panel {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 42px 40px;
  background:
    radial-gradient(600px 400px at 100% 0%, rgba(141, 162, 255, 0.05), transparent 60%),
    var(--bg);
}

.setup-card {
  position: relative;
  width: min(520px, 100%);
}

.skip-link {
  position: absolute;
  right: 0;
  top: -30px;
  border: 0;
  background: transparent;
  color: var(--text-3);
  font-size: 11px;
}

.skip-link:hover {
  color: var(--text);
}

.stepper {
  display: flex;
  align-items: center;
  gap: 0;
  margin-bottom: 32px;
}

.step {
  display: flex;
  align-items: center;
  gap: 7px;
  color: var(--text-4);
  font-size: 10.5px;
  white-space: nowrap;
}

.step.active {
  color: var(--text);
}

.step.done {
  color: var(--accent);
}

.step-dot {
  display: grid;
  place-items: center;
  width: 20px;
  height: 20px;
  border: 1px solid var(--border);
  border-radius: 50%;
  background: var(--surface-2);
  font-size: 9px;
  font-family: var(--font-mono);
}

.step.active .step-dot {
  border-color: var(--accent);
  color: var(--accent);
  box-shadow: 0 0 0 3px var(--accent-soft);
}

.step.done .step-dot {
  border-color: rgba(86, 224, 192, 0.3);
  background: var(--accent-soft);
}

.step-line {
  width: 38px;
  height: 1px;
  margin: 0 9px;
  background: var(--border);
}

.step-panel {
  animation: step-in 260ms var(--ease);
}

.step-head {
  margin-bottom: 22px;
}

.step-head.centered {
  text-align: center;
}

.step-kicker {
  color: var(--accent);
  font-family: var(--font-mono);
  font-size: 9.5px;
  letter-spacing: 0.14em;
  text-transform: uppercase;
}

.step-head h2 {
  margin-top: 7px;
  font-size: 25px;
  letter-spacing: -0.035em;
}

.step-head p {
  margin-top: 7px;
  color: var(--text-2);
  font-size: 12px;
  line-height: 1.65;
}

.binary-ready {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px;
  border: 1px solid rgba(70, 213, 151, 0.2);
  border-radius: var(--radius-md);
  background: var(--ok-soft);
}

.ready-icon {
  display: grid;
  place-items: center;
  width: 34px;
  height: 34px;
  flex: none;
  border-radius: 50%;
  background: rgba(70, 213, 151, 0.14);
  color: var(--ok);
}

.ready-copy {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.ready-copy strong {
  font-size: 12px;
}

.ready-copy .mono {
  color: var(--text-3);
  font-size: 10px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.binary-options {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.binary-option {
  display: flex;
  align-items: center;
  gap: 11px;
  width: 100%;
  padding: 13px;
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--surface);
  color: var(--text);
  text-align: left;
  transition:
    border-color var(--speed) var(--ease),
    background var(--speed) var(--ease);
}

.binary-option:hover:not(:disabled) {
  border-color: var(--border-strong);
  background: var(--surface-2);
}

.binary-option.primary {
  border-color: rgba(86, 224, 192, 0.25);
  background: linear-gradient(110deg, var(--accent-soft), transparent 60%);
}

.option-icon {
  display: grid;
  place-items: center;
  width: 36px;
  height: 36px;
  flex: none;
  border-radius: 9px;
  border: 1px solid var(--border);
  background: var(--bg-soft);
  color: var(--accent);
}

.option-copy {
  display: flex;
  flex-direction: column;
  gap: 2px;
  flex: 1;
  min-width: 0;
}

.option-copy strong {
  font-size: 12px;
  font-weight: 600;
}

.option-copy small {
  color: var(--text-3);
  font-size: 10.5px;
}

.download-box {
  margin-top: 12px;
  padding: 11px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--bg-soft);
}

.download-row {
  display: flex;
  justify-content: space-between;
  margin-bottom: 7px;
  color: var(--text-2);
  font-size: 10.5px;
}

.detected {
  display: flex;
  flex-direction: column;
  gap: 4px;
  margin-top: 14px;
}

.detected-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px;
  border: 1px solid var(--border);
  border-radius: 6px;
  background: var(--bg-soft);
  color: var(--text);
  text-align: left;
}

.detected-item .mono {
  flex: 1;
  min-width: 0;
  font-size: 10.5px;
}

.step-tip {
  display: flex;
  gap: 7px;
  margin-top: 17px;
  color: var(--text-3);
  font-size: 10.5px;
  line-height: 1.5;
}

.mode-tabs {
  display: flex;
  gap: 2px;
  padding: 3px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--bg-soft);
  margin-bottom: 16px;
}

.mode-tab {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  flex: 1;
  min-height: 32px;
  border: 1px solid transparent;
  border-radius: 5px;
  background: transparent;
  color: var(--text-3);
  font-size: 11.5px;
  font-weight: 550;
}

.mode-tab.active {
  color: var(--text);
  background: var(--surface-3);
  border-color: var(--border);
}

.manual-panel {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.form-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px;
}

.form-grid .span-2 {
  grid-column: span 2;
}

.field-icon {
  position: absolute;
  left: 11px;
  top: 50%;
  transform: translateY(-50%);
  color: var(--text-4);
}

.input-affix .field-icon + .input {
  padding-left: 33px;
}

.command-panel {
  display: flex;
  flex-direction: column;
  gap: 9px;
}

.command-panel .textarea {
  min-height: 108px;
}

.error-box,
.warn-box {
  display: flex;
  align-items: center;
  gap: 7px;
  padding: 8px 10px;
  border-radius: 6px;
  font-size: 11px;
}

.error-box {
  color: var(--error);
  background: var(--error-soft);
}

.warn-box {
  color: var(--warn);
  background: var(--warn-soft);
}

.parsed-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
  margin-top: 5px;
}

.parsed-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px;
  border: 1px solid var(--border);
  border-radius: 6px;
  background: var(--bg-soft);
}

.parsed-main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 1px;
}

.parsed-main strong {
  font-size: 11.5px;
}

.parsed-main small {
  color: var(--text-3);
  font-size: 10px;
}

.step-actions {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 10px;
  margin-top: 24px;
}

.back-icon {
  transform: rotate(180deg);
}

.finish-panel {
  display: flex;
  flex-direction: column;
  align-items: center;
}

.finish-mark {
  display: grid;
  place-items: center;
  margin: 3px 0 14px;
}

.finish-ring {
  display: grid;
  place-items: center;
  width: 62px;
  height: 62px;
  border: 1px solid rgba(86, 224, 192, 0.3);
  border-radius: 50%;
  background: radial-gradient(circle, var(--accent-soft), transparent 70%);
  color: var(--accent);
  box-shadow: 0 0 30px rgba(86, 224, 192, 0.12);
}

.summary {
  width: 100%;
  margin-top: 7px;
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--bg-soft);
  overflow: hidden;
}

.summary-row {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 12px;
  font-size: 11px;
  color: var(--text-3);
}

.summary-row + .summary-row {
  border-top: 1px solid var(--border-soft);
}

.summary-row .mono {
  color: var(--text-2);
  font-size: 10.5px;
  text-align: right;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.start-option {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  margin-top: 13px;
  padding: 11px 12px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--surface);
}

.start-copy {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.start-copy strong {
  font-size: 11.5px;
}

.start-copy small {
  color: var(--text-3);
  font-size: 10px;
}

.native-check {
  width: 16px;
  height: 16px;
  accent-color: var(--accent);
}

.finish-panel .step-actions {
  width: 100%;
}

.setup-foot {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: 30px;
  padding-top: 13px;
  border-top: 1px solid var(--border-soft);
  color: var(--text-4);
  font-size: 10px;
}

.text-link {
  border: 0;
  background: transparent;
  color: var(--text-3);
  font-size: 10px;
}

.text-link:hover {
  color: var(--text);
  text-decoration: underline;
}

@keyframes step-in {
  from {
    opacity: 0;
    transform: translateY(5px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

@media (max-width: 1040px) {
  .setup {
    grid-template-columns: 1fr;
  }

  .setup-visual {
    min-height: 320px;
    padding: 30px;
    border-right: 0;
    border-bottom: 1px solid var(--border-soft);
  }

  .visual-copy {
    margin: 36px 0 0;
  }

  .visual-copy h1 {
    font-size: 40px;
  }

  .visual-points {
    display: none;
  }

  .setup-panel {
    padding: 30px 22px 50px;
  }

  .skip-link {
    top: -12px;
  }
}
</style>