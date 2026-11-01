<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import {
  Copy,
  Download,
  Eye,
  EyeOff,
  FileCode2,
  Play,
  RotateCw,
  Save,
  Square,
  TerminalSquare,
  Trash2,
  TriangleAlert,
  WandSparkles,
} from 'lucide-vue-next'
import type { ConfigPreview, Profile, RuntimeInfo } from '@/lib/types'
import { emptyProfile } from '@/lib/types'
import { activeState, stateLabel } from '@/lib/format'
import { backend, errorMessage } from '@/lib/backend'
import StatusBadge from '@/components/ui/StatusBadge.vue'
import FormField from '@/components/ui/FormField.vue'
import ToggleSwitch from '@/components/ui/ToggleSwitch.vue'
import CodeBlock from '@/components/ui/CodeBlock.vue'
import CommandFill from '@/components/profiles/CommandFill.vue'

const props = defineProps<{
  profile: Profile
  runtime: RuntimeInfo
  fieldErrors: Record<string, string>
  saving: boolean
  pendingRestart: boolean
}>()

const emit = defineEmits<{
  save: [Profile]
  start: []
  stop: []
  restart: []
  duplicate: []
  delete: []
  exportConfig: []
  dirty: [boolean]
  clearErrors: []
}>()

type Tab = 'basic' | 'network' | 'advanced' | 'extensions' | 'preview'
const tab = ref<Tab>('basic')
const revealKey = ref(false)
const draft = ref<Profile>(emptyProfile())
const snapshot = ref('')
const preview = ref<ConfigPreview | null>(null)
const previewLoading = ref(false)
const previewError = ref('')
const revealPreview = ref(false)

const isNew = computed(() => !props.profile.id)
const active = computed(() => activeState(props.runtime.state))
const dirty = computed(() => JSON.stringify(draft.value) !== snapshot.value)
const statusError = computed(() => props.runtime.lastError || '')

function reset(profile: Profile): void {
  draft.value = JSON.parse(JSON.stringify(profile)) as Profile
  snapshot.value = JSON.stringify(draft.value)
  revealKey.value = false
  preview.value = null
  previewError.value = ''
  tab.value = 'basic'
}

watch(
  () => props.profile?.id,
  () => {
    if (props.profile) reset(props.profile)
  },
  { immediate: true },
)

watch(
  () => props.profile?.updatedAt,
  (value, old) => {
    if (value !== old && !dirty.value && props.profile) reset(props.profile)
  },
)

watch(
  dirty,
  (value) => emit('dirty', value),
  { immediate: true },
)

const localErrors = computed<Record<string, string>>(() => {
  const errors: Record<string, string> = {}
  if (!draft.value.name.trim()) errors.name = '请填写档案名称'
  if (!draft.value.server.trim()) errors.server = '请填写服务端地址'
  else if (draft.value.server.includes(',')) errors.server = '一个档案只能填写一个服务端'
  if (!draft.value.vkey.trim()) errors.vkey = '请填写 vkey'
  if (!draft.value.connType) errors.connType = '请选择连接方式'
  return errors
})

const errors = computed(() => ({ ...localErrors.value, ...props.fieldErrors }))

const protoAuto = computed({
  get: () => draft.value.protoVersion === null || draft.value.protoVersion === undefined,
  set: (value: boolean) => {
    draft.value.protoVersion = value ? null : 0
  },
})

const autoStartModel = computed({
  get: () => draft.value.autoStart ?? false,
  set: (value: boolean) => {
    draft.value.autoStart = value
  },
})

const cryptModel = computed({
  get: () => draft.value.crypt ?? false,
  set: (value: boolean) => {
    draft.value.crypt = value
  },
})

const compressModel = computed({
  get: () => draft.value.compress ?? false,
  set: (value: boolean) => {
    draft.value.compress = value
  },
})

const skipVerifyModel = computed({
  get: () => draft.value.skipTlsVerify ?? false,
  set: (value: boolean) => {
    draft.value.skipTlsVerify = value
  },
})

const disableP2pModel = computed({
  get: () => draft.value.disableP2p ?? false,
  set: (value: boolean) => {
    draft.value.disableP2p = value
  },
})

const snippets: Record<string, { label: string; text: string }> = {
  tcp: {
    label: 'TCP 隧道',
    text: '[tcp]\nmode=tcp\nserver_port=10000\ntarget_addr=127.0.0.1:8080',
  },
  udp: {
    label: 'UDP 隧道',
    text: '[udp]\nmode=udp\nserver_port=12253\ntarget_addr=10.0.0.2:53',
  },
  web: {
    label: 'HTTP 域名',
    text: '[web1]\nhost=app.example.com\ntarget_addr=127.0.0.1:8080\nscheme=all',
  },
  socks5: {
    label: 'SOCKS5',
    text: '[socks5]\nmode=socks5\nserver_port=19009\nbasic_username=user\nbasic_password=pass',
  },
  p2p: {
    label: 'P2P',
    text: '[p2p_ssh]\nmode=p2p\npassword=change-me\ntarget_addr=10.0.0.2:22\nfallback_secret=true',
  },
  file: {
    label: '文件服务',
    text: '[file]\nmode=file\nserver_port=9100\nlocal_path=/Users/you/share\nstrip_pre=/web/\nread_only=false',
  },
  health: {
    label: '健康检查',
    text: '[health_check_web]\nhealth_check_timeout=1\nhealth_check_max_failed=3\nhealth_check_interval=1\nhealth_check_type=http\nhealth_check_target=127.0.0.1:8080',
  },
}

function appendSnippet(key: string): void {
  const snippet = snippets[key]
  if (!snippet) return
  const current = draft.value.extraSections?.trimEnd() || ''
  draft.value.extraSections = (current ? current + '\n\n' : '') + snippet.text
}

function applyCommandProfile(p: Profile): void {
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
  emit('clearErrors')
}

function save(): void {
  if (Object.keys(localErrors.value).length) return
  emit('save', JSON.parse(JSON.stringify(draft.value)) as Profile)
}

function markSaved(): void {
  snapshot.value = JSON.stringify(draft.value)
  emit('dirty', false)
}

defineExpose({ markSaved })

async function loadPreview(): Promise<void> {
  if (!props.profile.id) {
    preview.value = null
    return
  }
  previewLoading.value = true
  previewError.value = ''
  try {
    preview.value = await backend.previewConfig(props.profile.id, revealPreview.value)
  } catch (err) {
    previewError.value = errorMessage(err)
  } finally {
    previewLoading.value = false
  }
}

watch([() => tab.value, () => revealPreview.value, () => props.profile.id], () => {
  if (tab.value === 'preview') void loadPreview()
})

const sections = [
  { id: 'basic' as Tab, label: '基础' },
  { id: 'network' as Tab, label: '网络' },
  { id: 'advanced' as Tab, label: '高级' },
  { id: 'extensions' as Tab, label: '扩展段落' },
  { id: 'preview' as Tab, label: '配置预览' },
]
</script>

<template>
  <div class="editor">
    <header class="editor-head">
      <div class="editor-title">
        <div class="row">
          <h2>{{ draft.name || (isNew ? '新建连接' : '未命名连接') }}</h2>
          <StatusBadge :state="runtime.state" compact />
          <span v-if="pendingRestart" class="badge badge-warn">
            <TriangleAlert :size="11" /> 配置待重启生效
          </span>
          <span v-if="dirty" class="dirty-mark">未保存</span>
        </div>
        <p class="mono small">{{ draft.server || '等待填写服务端地址' }}</p>
      </div>
      <div class="editor-actions">
        <button
          v-if="!active"
          class="btn btn-secondary btn-sm"
          type="button"
          :disabled="isNew || saving"
          @click="emit('start')"
        >
          <Play :size="13" fill="currentColor" /> 启动
        </button>
        <template v-else>
          <button class="btn btn-secondary btn-sm" type="button" @click="emit('stop')">
            <Square :size="12" fill="currentColor" /> 停止
          </button>
          <button class="btn btn-ghost btn-sm btn-icon" type="button" aria-label="重启" title="重启" @click="emit('restart')">
            <RotateCw :size="14" />
          </button>
        </template>
        <button class="btn btn-ghost btn-sm btn-icon" type="button" aria-label="复制档案" title="复制档案" :disabled="isNew || saving" @click="emit('duplicate')">
          <Copy :size="14" />
        </button>
        <button class="btn btn-ghost btn-sm btn-icon danger-icon" type="button" aria-label="删除档案" title="删除档案" :disabled="isNew || saving" @click="emit('delete')">
          <Trash2 :size="14" />
        </button>
        <button class="btn btn-primary btn-sm" type="button" :disabled="saving" @click="save">
          <span v-if="saving" class="spinner" />
          <Save v-else :size="14" />
          保存
        </button>
      </div>
    </header>

    <div v-if="statusError && runtime.state === 'error'" class="status-error">
      <TriangleAlert :size="14" />
      <span class="ellipsis">{{ statusError }}</span>
      <button class="btn btn-ghost btn-sm" type="button" @click="emit('restart')">重试</button>
    </div>

    <nav class="tabs" aria-label="配置分区">
      <button
        v-for="section in sections"
        :key="section.id"
        type="button"
        class="tab"
        :class="{ active: tab === section.id }"
        @click="tab = section.id"
      >
        {{ section.label }}
      </button>
    </nav>

    <div class="editor-body">
      <section v-if="tab === 'basic'" class="form-section">
        <div class="form-section-head">
          <h3>连接身份</h3>
          <p>服务端地址、vkey 和连接方式来自 NPS 管理台客户端页面。</p>
        </div>
        <CommandFill v-if="isNew" @parsed="applyCommandProfile" />
        <div class="field-grid">
          <FormField label="档案名称" required :error="errors.name" html-for="pf-name">
            <input id="pf-name" v-model="draft.name" class="input" :class="{ 'has-error': errors.name }" placeholder="例如：公司跳板" maxlength="64" />
          </FormField>
          <FormField label="连接方式" required :error="errors.connType" html-for="pf-type">
            <select id="pf-type" v-model="draft.connType" class="select">
              <option value="tcp">TCP</option>
              <option value="tls">TLS</option>
              <option value="kcp">KCP</option>
              <option value="quic">QUIC</option>
              <option value="ws">WebSocket</option>
              <option value="wss">WebSocket + TLS</option>
            </select>
          </FormField>
          <FormField
            class="span-2"
            label="服务端地址"
            required
            :error="errors.server"
            html-for="pf-server"
            hint="host:port，可带 @SNI 或 /ws_path；一个档案只填一个服务端。"
          >
            <input
              id="pf-server"
              v-model="draft.server"
              class="input mono"
              :class="{ 'has-error': errors.server }"
              placeholder="nps.example.com:8024"
              spellcheck="false"
            />
          </FormField>
          <FormField class="span-2" label="vkey" required :error="errors.vkey" html-for="pf-vkey" hint="保存在本地配置中，不会出现在启动参数里。">
            <div class="input-affix">
              <input
                id="pf-vkey"
                v-model="draft.vkey"
                class="input mono"
                :class="{ 'has-error': errors.vkey }"
                :type="revealKey ? 'text' : 'password'"
                placeholder="从 NPS 客户端页面复制"
                spellcheck="false"
                autocomplete="off"
              />
              <button class="affix btn btn-ghost btn-sm btn-icon" type="button" :aria-label="revealKey ? '隐藏 vkey' : '显示 vkey'" @click="revealKey = !revealKey">
                <EyeOff v-if="revealKey" :size="15" />
                <Eye v-else :size="15" />
              </button>
            </div>
          </FormField>
          <FormField label="备注" optional html-for="pf-remark">
            <input id="pf-remark" v-model="draft.remark" class="input" placeholder="可选，例如：办公区出口" maxlength="120" />
          </FormField>
          <FormField label="日志等级" html-for="pf-log">
            <select id="pf-log" v-model="draft.logLevel" class="select">
              <option value="debug">debug — 调试细节</option>
              <option value="info">info — 推荐</option>
              <option value="warn">warn — 仅警告</option>
              <option value="error">error — 仅错误</option>
            </select>
          </FormField>
        </div>

        <div class="switch-grid">
          <div class="switch-row">
            <span>
              <strong>自动重连</strong>
              <small>断线后由 npc 自动恢复连接</small>
            </span>
            <ToggleSwitch v-model="draft.autoReconnection" label="自动重连" />
          </div>
          <div class="switch-row">
            <span>
              <strong>随应用启动</strong>
              <small>打开 NPC 客户端时自动启动此档案</small>
            </span>
            <ToggleSwitch v-model="autoStartModel" label="随应用启动" />
          </div>
        </div>
      </section>

      <section v-else-if="tab === 'network'" class="form-section">
        <div class="form-section-head">
          <h3>网络出口</h3>
          <p>代理、DNS、NTP 与断线参数。留空使用 npc 默认值。</p>
        </div>
        <div class="field-grid">
          <FormField class="span-2" label="代理地址" optional :error="errors.proxyUrl" html-for="pf-proxy" hint="支持 socks5://、socks4://、http://、https://；留空直连。">
            <input id="pf-proxy" v-model="draft.proxyUrl" class="input mono" :class="{ 'has-error': errors.proxyUrl }" placeholder="socks5://user:pass@127.0.0.1:1080" spellcheck="false" />
          </FormField>
          <FormField label="本地出口 IP" optional :error="errors.localIp" html-for="pf-localip" hint="多网卡时指定出站源地址。">
            <input id="pf-localip" v-model="draft.localIp" class="input mono" placeholder="192.168.1.20" spellcheck="false" />
          </FormField>
          <FormField label="DNS 服务器" optional html-for="pf-dns">
            <input id="pf-dns" v-model="draft.dnsServer" class="input mono" placeholder="8.8.8.8" spellcheck="false" />
          </FormField>
          <FormField label="NTP 服务器" optional html-for="pf-ntp">
            <input id="pf-ntp" v-model="draft.ntpServer" class="input mono" placeholder="pool.ntp.org" spellcheck="false" />
          </FormField>
          <FormField label="NTP 间隔" optional html-for="pf-ntp-int" hint="分钟，0 表示使用默认。">
            <input id="pf-ntp-int" v-model.number="draft.ntpInterval" class="input mono" type="number" min="0" max="1440" />
          </FormField>
          <FormField label="断线超时" optional html-for="pf-timeout" hint="秒，0 使用 npc 默认值。">
            <input id="pf-timeout" v-model.number="draft.disconnectTimeout" class="input mono" type="number" min="0" max="86400" />
          </FormField>
        </div>
      </section>

      <section v-else-if="tab === 'advanced'" class="form-section">
        <div class="form-section-head">
          <h3>传输与认证</h3>
          <p>除非服务端明确要求，否则保持默认即可。</p>
        </div>
        <div class="switch-grid">
          <div class="switch-row">
            <span><strong>加密传输</strong><small>crypt=true，写入 npc.conf</small></span>
            <ToggleSwitch v-model="cryptModel" label="加密传输" />
          </div>
          <div class="switch-row">
            <span><strong>压缩传输</strong><small>compress=true，写入 npc.conf</small></span>
            <ToggleSwitch v-model="compressModel" label="压缩传输" />
          </div>
          <div class="switch-row">
            <span><strong>跳过 TLS 校验</strong><small>仅用于自签服务端，存在中间人风险</small></span>
            <ToggleSwitch v-model="skipVerifyModel" label="跳过 TLS 校验" />
          </div>
          <div class="switch-row">
            <span><strong>禁用 P2P</strong><small>强制所有流量走服务端中转</small></span>
            <ToggleSwitch v-model="disableP2pModel" label="禁用 P2P" />
          </div>
        </div>

        <div class="field-grid">
          <FormField label="最大连接数" optional html-for="pf-maxconn" hint="0 表示不限制。">
            <input id="pf-maxconn" v-model.number="draft.maxConn" class="input mono" type="number" min="0" />
          </FormField>
          <FormField label="速率限制" optional html-for="pf-rate" hint="单位由 nps 解释，0 表示不限制。">
            <input id="pf-rate" v-model.number="draft.rateLimit" class="input mono" type="number" min="0" />
          </FormField>
          <FormField label="流量限制" optional html-for="pf-flow" hint="0 表示不限制。">
            <input id="pf-flow" v-model.number="draft.flowLimit" class="input mono" type="number" min="0" />
          </FormField>
          <FormField label="保活周期" optional html-for="pf-keepalive" hint="秒，0 使用 npc 默认值。">
            <input id="pf-keepalive" v-model.number="draft.keepalive" class="input mono" type="number" min="0" />
          </FormField>
          <FormField label="pprof 地址" optional html-for="pf-pprof">
            <input id="pf-pprof" v-model="draft.pprofAddr" class="input mono" placeholder="127.0.0.1:6060" spellcheck="false" />
          </FormField>
          <FormField label="socks / web 用户名" optional html-for="pf-webuser">
            <input id="pf-webuser" v-model="draft.webUsername" class="input" placeholder="可选" autocomplete="off" />
          </FormField>
          <FormField label="socks / web 密码" optional html-for="pf-webpass">
            <input id="pf-webpass" v-model="draft.webPassword" class="input" type="password" placeholder="可选" autocomplete="new-password" />
          </FormField>
          <FormField label="Basic 认证用户名" optional html-for="pf-basicuser">
            <input id="pf-basicuser" v-model="draft.basicUsername" class="input" placeholder="可选" autocomplete="off" />
          </FormField>
          <FormField label="Basic 认证密码" optional html-for="pf-basicpass">
            <input id="pf-basicpass" v-model="draft.basicPassword" class="input" type="password" placeholder="可选" autocomplete="new-password" />
          </FormField>
        </div>

        <div class="protocol-row">
          <div class="switch-row compact-row">
            <span><strong>跟随 npc 默认协议</strong><small>一般不需要修改，旧服务端可能需要指定</small></span>
            <ToggleSwitch v-model="protoAuto" label="跟随默认协议" />
          </div>
          <input
            v-if="!protoAuto"
            v-model.number="draft.protoVersion"
            class="input mono proto-input"
            type="number"
            min="0"
            max="32"
            aria-label="协议版本"
          />
        </div>
      </section>

      <section v-else-if="tab === 'extensions'" class="form-section">
        <div class="form-section-head">
          <h3>扩展配置</h3>
          <p>原样透传给 npc.conf。适合网页、TCP/UDP、SOCKS5、P2P、文件服务和健康检查。</p>
        </div>

        <FormField
          label="额外 common 键值"
          optional
          :error="errors.extraCommon"
          html-for="pf-extra-common"
          hint="每行 key=value；不能覆盖表单已管理的 server_addr、vkey 等键。"
        >
          <textarea id="pf-extra-common" v-model="draft.extraCommon" class="textarea mono" rows="4" spellcheck="false" placeholder="remark=my-client" />
        </FormField>

        <div class="snippet-row">
          <span class="hint">快速插入段落：</span>
          <button
            v-for="(snippet, key) in snippets"
            :key="key"
            class="btn btn-ghost btn-sm"
            type="button"
            @click="appendSnippet(key)"
          >
            + {{ snippet.label }}
          </button>
        </div>

        <FormField
          label="额外 INI 段落"
          optional
          :error="errors.extraSections"
          html-for="pf-extra-sections"
          hint="每个段落以 [name] 开头；禁止重复段落和 [common]；相对路径以档案目录为工作目录。"
        >
          <textarea
            id="pf-extra-sections"
            v-model="draft.extraSections"
            class="textarea mono"
            rows="13"
            spellcheck="false"
            placeholder="[tcp]&#10;mode=tcp&#10;server_port=10000&#10;target_addr=127.0.0.1:8080"
          />
        </FormField>

        <div class="extension-note">
          <WandSparkles :size="15" />
          <p>扩展段落会在启动时发送给服务端；如果服务端已有同名隧道，可能会提示端口或域名被占用。</p>
        </div>
      </section>

      <section v-else class="form-section preview-section">
        <div class="form-section-head">
          <h3>配置预览</h3>
          <p>预览基于已保存的配置；修改后先保存再刷新。</p>
        </div>
        <div class="preview-toolbar">
          <div class="row">
            <button class="btn btn-ghost btn-sm" type="button" @click="revealPreview = !revealPreview">
              <EyeOff v-if="revealPreview" :size="13" />
              <Eye v-else :size="13" />
              {{ revealPreview ? '隐藏密钥' : '显示密钥' }}
            </button>
            <button class="btn btn-ghost btn-sm" type="button" :disabled="previewLoading" @click="loadPreview">
              <RotateCw :size="13" /> 刷新
            </button>
          </div>
          <button class="btn btn-secondary btn-sm" type="button" :disabled="isNew" @click="emit('exportConfig')">
            <Download :size="13" /> 导出配置
          </button>
        </div>
        <div v-if="isNew" class="preview-empty">
          <FileCode2 :size="22" />
          <p>保存档案后即可预览生成的 npc.conf。</p>
        </div>
        <div v-else-if="previewError" class="notice notice-error">{{ previewError }}</div>
        <div v-else-if="previewLoading" class="preview-empty">
          <span class="spinner" /> 正在生成配置…
        </div>
        <template v-else-if="preview">
          <CodeBlock :code="preview.content" title="npc.conf" />
          <CodeBlock :code="preview.command" title="启动命令（含配置路径，不含 vkey）" />
          <p class="hint mono">{{ preview.path }}</p>
        </template>
      </section>
    </div>

    <footer class="editor-foot">
      <div class="row">
        <span class="hint">{{ isNew ? '保存后会生成独立的 npc.conf。' : '密钥仅保存在本机配置目录。' }}</span>
      </div>
      <div class="row">
        <button class="btn btn-primary" type="button" :disabled="saving" @click="save">
          <span v-if="saving" class="spinner" />
          <Save v-else :size="14" />
          保存配置
        </button>
      </div>
    </footer>
  </div>
</template>

<style scoped>
.editor {
  display: flex;
  flex-direction: column;
  min-height: 100%;
}

.editor-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 18px;
  padding: 20px 22px 16px;
  border-bottom: 1px solid var(--border-soft);
}

.editor-title {
  min-width: 0;
}

.editor-title h2 {
  font-size: 18px;
  font-weight: 650;
  letter-spacing: -0.02em;
}

.editor-title p {
  margin-top: 5px;
  color: var(--text-3);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.dirty-mark {
  color: var(--warn);
  font-size: 10px;
  font-weight: 600;
}

.editor-actions {
  display: flex;
  align-items: center;
  gap: 3px;
  flex-wrap: wrap;
  justify-content: flex-end;
}

.danger-icon:hover:not(:disabled) {
  color: var(--error);
  background: var(--error-soft);
}

.status-error {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 9px 22px;
  border-bottom: 1px solid rgba(255, 125, 136, 0.18);
  background: var(--error-soft);
  color: var(--error);
  font-size: 11.5px;
}

.status-error span {
  flex: 1;
  min-width: 0;
}

.tabs {
  display: flex;
  gap: 2px;
  padding: 0 22px;
  border-bottom: 1px solid var(--border-soft);
  overflow-x: auto;
  overflow-y: hidden;
  overscroll-behavior-x: contain;
}

.tab {
  position: relative;
  padding: 11px 10px 10px;
  border: 0;
  background: transparent;
  color: var(--text-3);
  font-size: 12px;
  font-weight: 550;
  white-space: nowrap;
}

.tab:hover {
  color: var(--text);
}

.tab.active {
  color: var(--text);
}

.tab.active::after {
  content: '';
  position: absolute;
  left: 9px;
  right: 9px;
  bottom: 0;
  height: 2px;
  border-radius: 2px;
  background: var(--accent);
}

.editor-body {
  flex: 1;
  padding: 22px;
  overflow: auto;
}

.form-section {
  display: flex;
  flex-direction: column;
  gap: 18px;
  max-width: 900px;
}

.form-section-head h3 {
  font-size: 13px;
  font-weight: 650;
}

.form-section-head p {
  margin-top: 3px;
  color: var(--text-3);
  font-size: 11.5px;
}

.switch-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
}

.switch-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  padding: 11px 12px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--bg-soft);
  cursor: pointer;
}

[data-theme='light'] .switch-row {
  background: var(--surface-2);
}

.switch-row span:first-child {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.switch-row strong {
  font-size: 11.5px;
  font-weight: 600;
}

.switch-row small {
  color: var(--text-3);
  font-size: 10px;
  line-height: 1.4;
}

.compact-row {
  flex: 1;
}

.protocol-row {
  display: flex;
  align-items: center;
  gap: 10px;
}

.proto-input {
  width: 100px;
}

.snippet-row {
  display: flex;
  align-items: center;
  gap: 2px;
  flex-wrap: wrap;
}

.extension-note {
  display: flex;
  gap: 9px;
  padding: 10px 12px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--accent-2-soft);
  color: var(--accent-2);
  font-size: 11.5px;
  line-height: 1.5;
}

.extension-note p {
  color: var(--text-2);
}

.preview-section {
  max-width: 100%;
}

.preview-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.preview-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 9px;
  min-height: 180px;
  color: var(--text-3);
  border: 1px dashed var(--border);
  border-radius: var(--radius-md);
  font-size: 12px;
}

.notice-error {
  padding: 10px 12px;
  border: 1px solid rgba(255, 125, 136, 0.22);
  border-radius: var(--radius-sm);
  background: var(--error-soft);
  color: var(--error);
  font-size: 12px;
}

.editor-foot {
  position: sticky;
  bottom: 0;
  z-index: 2;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 13px 22px;
  border-top: 1px solid var(--border);
  background: rgba(15, 23, 32, 0.94);
  backdrop-filter: blur(12px);
}

[data-theme='light'] .editor-foot {
  background: rgba(255, 255, 255, 0.94);
}

@media (max-width: 900px) {
  .editor-head {
    flex-direction: column;
  }

  .switch-grid {
    grid-template-columns: 1fr;
  }

  .editor-actions {
    justify-content: flex-start;
  }

  .editor-foot {
    flex-direction: column;
    align-items: stretch;
  }
}
</style>