<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import {
  Check,
  Copy,
  Database,
  ExternalLink,
  FileText,
  Info,
  Power,
  RefreshCw,
  Save,
  ShieldCheck,
} from 'lucide-vue-next'
import type { Settings } from '@/lib/types'
import { useAppStore } from '@/stores/app'
import UpdateCard from '@/components/update/UpdateCard.vue'
import NpcCorePanel from '@/components/update/NpcCorePanel.vue'
import FormField from '@/components/ui/FormField.vue'
import ToggleSwitch from '@/components/ui/ToggleSwitch.vue'
import ModalDialog from '@/components/ui/ModalDialog.vue'

const app = useAppStore()
const draft = ref<Settings>({ ...app.settings })
const saving = ref(false)
const diagnostics = ref('')
const diagnosticsOpen = ref(false)
const diagnosticsCopied = ref(false)

const dirty = computed(() => JSON.stringify(draft.value) !== JSON.stringify(app.settings))

watch(
  () => app.settings,
  (settings) => {
    if (!dirty.value) draft.value = { ...settings }
  },
  { deep: true },
)

function resetDraft(): void {
  draft.value = { ...app.settings }
}

async function save(): Promise<void> {
  if (saving.value) return
  saving.value = true
  try {
    const result = await app.saveSettings(draft.value)
    if (result.ok) {
      app.toast('success', '设置已保存')
      draft.value = { ...app.settings }
    } else {
      app.toast('error', '保存设置失败', result.message)
    }
  } finally {
    saving.value = false
  }
}

async function loadDiagnostics(): Promise<void> {
  diagnostics.value = await app.getDiagnostics()
  diagnosticsOpen.value = true
  diagnosticsCopied.value = false
}

async function copyDiagnostics(): Promise<void> {
  if (!diagnostics.value) diagnostics.value = await app.getDiagnostics()
  try {
    await navigator.clipboard.writeText(diagnostics.value)
    diagnosticsCopied.value = true
    app.toast('success', '诊断信息已复制')
  } catch {
    diagnosticsOpen.value = true
    app.toast('warn', '无法访问剪贴板', '可以在弹窗中手动复制')
  }
}

function openExternal(url?: string): void {
  if (url) void app.openExternal(url)
}
</script>

<template>
  <div class="page settings-page">
    <header class="page-header">
      <div>
        <div class="kicker">settings</div>
        <h1 class="page-title">设置</h1>
        <p class="page-subtitle">客户端更新、NPC 核心版本、日志与本机数据。</p>
      </div>
      <div class="page-actions">
        <span v-if="dirty" class="dirty-pill">有未保存的修改</span>
        <button class="btn btn-ghost" type="button" :disabled="!dirty || saving" @click="resetDraft">
          <RefreshCw :size="14" /> 重置
        </button>
        <button class="btn btn-primary" type="button" :disabled="saving" @click="save">
          <span v-if="saving" class="spinner" />
          <Save v-else :size="14" />
          保存设置
        </button>
      </div>
    </header>

    <div class="settings-stack">
      <UpdateCard />
      <NpcCorePanel
        :mirror="draft.downloadMirror || ''"
        @update:mirror="draft.downloadMirror = $event"
      />

      <section class="panel preferences">
        <div class="panel-head">
          <div class="row">
            <div class="pref-mark"><ShieldCheck :size="20" /></div>
            <div>
              <h2 class="section-title">通用偏好</h2>
              <p class="section-desc">这些选项保存在本机，不随更新丢失。</p>
            </div>
          </div>
        </div>

        <div class="pref-grid">
          <FormField label="界面主题" html-for="set-theme" hint="浅色主题仍在完善中。">
            <select id="set-theme" v-model="draft.theme" class="select">
              <option value="dark">深色（推荐）</option>
              <option value="light">浅色</option>
            </select>
          </FormField>
          <FormField label="默认日志等级" html-for="set-loglevel" hint="新建档案时使用，可单独覆盖。">
            <select id="set-loglevel" v-model="draft.defaultLogLevel" class="select">
              <option value="debug">debug</option>
              <option value="info">info</option>
              <option value="warn">warn</option>
              <option value="error">error</option>
            </select>
          </FormField>
          <FormField label="内存日志上限" html-for="set-maxlog" hint="每个连接保留的最大行数，500-50000。">
            <input id="set-maxlog" v-model.number="draft.maxLogLines" class="input mono" type="number" min="500" max="50000" step="500" />
          </FormField>
          <FormField label="下载加速前缀" optional html-for="set-mirror" hint="用于 GitHub 资产下载，留空直连。">
            <input id="set-mirror" v-model="draft.downloadMirror" class="input mono" placeholder="https://ghfast.top/" spellcheck="false" />
          </FormField>
        </div>

        <div class="switch-list">
          <div class="switch-row">
            <span><strong>启动时检查客户端更新</strong><small>最多每 24 小时访问一次 GitHub Releases。</small></span>
            <ToggleSwitch v-model="draft.autoCheckUpdates" label="自动检查更新" />
          </div>
          <div class="switch-row">
            <span><strong>日志落盘</strong><small>将采集到的 npc 日志写入 0600 权限的本地文件，按大小轮转。</small></span>
            <ToggleSwitch v-model="draft.logToDisk" label="日志落盘" />
          </div>
          <div class="switch-row">
            <span><strong>关闭时确认</strong><small>未启用托盘时，有连接运行中则关闭窗口需要二次确认。</small></span>
            <ToggleSwitch v-model="draft.confirmOnQuit" label="关闭时确认" />
          </div>
        </div>
      </section>

      <section class="panel preferences">
        <div class="panel-head">
          <div class="row">
            <div class="pref-mark neutral"><Power :size="20" /></div>
            <div>
              <h2 class="section-title">系统集成</h2>
              <p class="section-desc">开机启动、系统托盘和桌面通知；托盘开关在下次启动时生效。</p>
            </div>
          </div>
        </div>

        <div class="switch-list">
          <div class="switch-row">
            <span><strong>开机自启</strong><small>登录系统后在后台启动客户端，可通过托盘唤回窗口。</small></span>
            <ToggleSwitch v-model="draft.launchAtLogin" label="开机自启" />
          </div>
          <div class="switch-row">
            <span><strong>启用系统托盘</strong><small>在菜单栏或通知区显示 NPC 图标，快速查看状态和启停连接。</small></span>
            <ToggleSwitch v-model="draft.trayEnabled" label="启用系统托盘" />
          </div>
          <div class="switch-row">
            <span><strong>关闭时收起到托盘</strong><small>托盘可用时，关闭窗口会隐藏到托盘；退出请使用托盘菜单。</small></span>
            <ToggleSwitch v-model="draft.minimizeToTray" label="关闭时收起到托盘" />
          </div>
          <div class="switch-row">
            <span><strong>系统通知</strong><small>连接异常、恢复或发现新版本时发送系统通知。</small></span>
            <ToggleSwitch v-model="draft.notifications" label="系统通知" />
          </div>
        </div>
      </section>

      <section class="panel data-panel">
        <div class="panel-head">
          <div class="row">
            <div class="pref-mark neutral"><Database :size="20" /></div>
            <div>
              <h2 class="section-title">数据与诊断</h2>
              <p class="section-desc">配置、密钥、日志和更新缓存的本地位置。</p>
            </div>
          </div>
        </div>

        <div class="data-path">
          <span class="kicker">data directory</span>
          <button class="path-link mono" type="button" @click="app.openDataDir">
            {{ app.appInfo?.dataDir || '...' }}
          </button>
        </div>

        <div class="data-actions">
          <button class="btn btn-secondary btn-sm" type="button" @click="app.openDataDir">
            <Database :size="13" /> 打开数据目录
          </button>
          <button class="btn btn-secondary btn-sm" type="button" @click="app.openUpdateLog">
            <FileText :size="13" /> 更新日志
          </button>
          <button class="btn btn-secondary btn-sm" type="button" @click="loadDiagnostics">
            <Info :size="13" /> 查看诊断信息
          </button>
          <button class="btn btn-ghost btn-sm" type="button" @click="copyDiagnostics">
            <Copy :size="13" /> 复制诊断
          </button>
        </div>

        <div class="about-grid">
          <div class="about-item">
            <span class="label">客户端版本</span>
            <span class="mono">v{{ app.version }}</span>
          </div>
          <div class="about-item">
            <span class="label">构建提交</span>
            <span class="mono">{{ app.appInfo?.commit || 'local' }}</span>
          </div>
          <div class="about-item">
            <span class="label">构建时间</span>
            <span class="mono">{{ app.appInfo?.buildDate || 'unknown' }}</span>
          </div>
          <div class="about-item">
            <span class="label">运行平台</span>
            <span class="mono">{{ app.appInfo?.os }}/{{ app.appInfo?.arch }}</span>
          </div>
          <div class="about-item">
            <span class="label">更新仓库</span>
            <button class="mono link-btn" type="button" @click="app.openReleasePage">
              {{ app.appInfo?.repo || '开发构建，未关联' }}
            </button>
          </div>
          <div class="about-item">
            <span class="label">NPC 核心</span>
            <span class="mono">{{ app.binary?.version || '未配置' }}</span>
          </div>
        </div>

        <p class="license-note">
          NPC 客户端是 npc 的独立桌面前端，不随包分发 GPLv3 的 npc 二进制。npc 核心来自
          <button class="link-btn" type="button" @click="app.openNpcReleasePage">djylb/nps</button> 的官方发布。
        </p>
      </section>
    </div>

    <ModalDialog
      :open="diagnosticsOpen"
      title="诊断信息"
      subtitle="已自动隐藏 vkey、密码等敏感内容。"
      width="660px"
      @close="diagnosticsOpen = false"
    >
      <pre class="diagnostics">{{ diagnostics }}</pre>
      <template #footer>
        <button class="btn btn-ghost" type="button" @click="diagnosticsOpen = false">关闭</button>
        <button class="btn btn-secondary" type="button" @click="copyDiagnostics">
          <Check v-if="diagnosticsCopied" :size="14" />
          <Copy v-else :size="14" />
          {{ diagnosticsCopied ? '已复制' : '复制' }}
        </button>
      </template>
    </ModalDialog>
  </div>
</template>

<style scoped>
.settings-page {
  max-width: 1180px;
}

.dirty-pill {
  display: inline-flex;
  align-items: center;
  height: 26px;
  padding: 0 9px;
  border: 1px solid rgba(240, 180, 90, 0.22);
  border-radius: 999px;
  background: var(--warn-soft);
  color: var(--warn);
  font-size: 10.5px;
  font-weight: 600;
}

.settings-stack {
  display: flex;
  flex-direction: column;
  gap: 15px;
}

.panel-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
}

.pref-mark {
  display: grid;
  place-items: center;
  width: 40px;
  height: 40px;
  flex: none;
  margin-right: 10px;
  border: 1px solid var(--border);
  border-radius: 11px;
  background: var(--surface-2);
  color: var(--accent);
}

.pref-mark.neutral {
  color: var(--text-3);
}

.preferences,
.data-panel {
  padding: 19px;
}

.pref-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px;
  margin-top: 18px;
}

.switch-list {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
  margin-top: 16px;
}

.switch-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 11px 12px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--bg-soft);
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

.data-path {
  margin-top: 18px;
  padding: 12px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--bg-soft);
}

.path-link {
  display: block;
  margin-top: 5px;
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--text-2);
  font-size: 11.5px;
  text-align: left;
  word-break: break-all;
}

.path-link:hover {
  color: var(--accent);
}

.data-actions {
  display: flex;
  gap: 5px;
  flex-wrap: wrap;
  margin-top: 12px;
}

.about-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 1px;
  margin-top: 18px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--border);
  overflow: hidden;
}

.about-item {
  display: flex;
  flex-direction: column;
  gap: 5px;
  padding: 11px 12px;
  background: var(--surface);
}

.about-item .label {
  color: var(--text-3);
  font-size: 10px;
  font-weight: 500;
}

.about-item .mono {
  color: var(--text-2);
  font-size: 10.5px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.link-btn {
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--accent);
  font-size: 10.5px;
  text-align: left;
}

.link-btn:hover {
  text-decoration: underline;
}

.license-note {
  margin-top: 14px;
  color: var(--text-4);
  font-size: 10.5px;
  line-height: 1.6;
}

.diagnostics {
  margin: 0;
  padding: 12px;
  max-height: 46vh;
  overflow: auto;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--bg-soft);
  color: var(--text-2);
  font-family: var(--font-mono);
  font-size: 11px;
  line-height: 1.65;
  white-space: pre-wrap;
  word-break: break-word;
}

@media (max-width: 900px) {
  .pref-grid,
  .switch-list {
    grid-template-columns: 1fr;
  }

  .about-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
</style>
