<script setup lang="ts">
import { computed, ref } from 'vue'
import {
  ArrowUpRight,
  CheckCircle2,
  Download,
  ExternalLink,
  FileText,
  RefreshCw,
  RotateCw,
  ShieldAlert,
  Sparkles,
  X,
} from 'lucide-vue-next'
import { useAppStore } from '@/stores/app'
import { useRuntimesStore } from '@/stores/runtimes'
import { formatBytes, formatDateTime, formatPercent } from '@/lib/format'
import ProgressBar from '@/components/ui/ProgressBar.vue'
import ModalDialog from '@/components/ui/ModalDialog.vue'

const app = useAppStore()
const runtimes = useRuntimesStore()
const notesOpen = ref(false)
const applyOpen = ref(false)

const state = computed(() => app.update.state)
const progress = computed(() => app.update.progress || app.updateProgress)
const latest = computed(() => app.update.latest)
const available = computed(() => state.value === 'available')
const ready = computed(() => state.value === 'ready')
const downloading = computed(() => state.value === 'downloading')
const checking = computed(() => state.value === 'checking')
const disabled = computed(() => state.value === 'disabled')
const failed = computed(() => state.value === 'error')

const statusText = computed(() => {
  switch (state.value) {
    case 'checking':
      return '正在检查 GitHub Releases…'
    case 'available':
      return available.value ? '发现新版本 ' + latest.value?.version : '有新版本可用'
    case 'downloading':
      return '正在下载更新'
    case 'ready':
      return '更新已下载，等待重启安装'
    case 'applying':
      return '正在应用更新'
    case 'error':
      return app.update.error || '更新检查失败'
    case 'disabled':
      return '开发构建不检查更新'
    default:
      return '当前已是最新版本'
  }
})

const tone = computed(() => {
  if (failed.value) return 'error'
  if (available.value || ready.value) return 'warn'
  if (state.value === 'idle') return 'ok'
  return 'info'
})

const notes = computed(() => latest.value?.notes || '该版本没有提供更新说明。')
const showNotes = computed(() => notes.value.length > 220)

function openNotes(): void {
  notesOpen.value = true
}

function requestApply(): void {
  if (runtimes.runningCount > 0) {
    applyOpen.value = true
    return
  }
  void apply()
}

async function apply(): Promise<void> {
  applyOpen.value = false
  await app.applyUpdate(true)
}

async function check(): Promise<void> {
  await app.checkUpdate(true)
}

async function download(): Promise<void> {
  await app.downloadUpdate()
}
</script>

<template>
  <section class="panel update-card">
    <div class="update-head">
      <div class="update-mark" :class="'tone-' + tone">
        <RotateCw v-if="state === 'applying' || checking" :size="19" class="spin-slow" />
        <Download v-else-if="available || downloading" :size="19" />
        <CheckCircle2 v-else-if="state === 'idle'" :size="19" />
        <ShieldAlert v-else-if="failed || disabled" :size="19" />
        <Sparkles v-else :size="19" />
      </div>
      <div class="update-copy">
        <div class="row">
          <h2 class="section-title">客户端更新</h2>
          <span class="badge" :class="'badge-' + tone">v{{ app.version }}</span>
          <span v-if="state === 'ready'" class="badge badge-warn">待重启</span>
        </div>
        <p class="section-desc">{{ statusText }}</p>
      </div>
      <div class="update-actions">
        <button class="btn btn-ghost btn-sm" type="button" :disabled="disabled || downloading || checking" @click="check">
          <span v-if="checking" class="spinner" />
          <RefreshCw v-else :size="13" />
          检查更新
        </button>
        <button v-if="disabled" class="btn btn-ghost btn-sm" type="button" @click="app.openReleasePage">
          GitHub <ArrowUpRight :size="12" />
        </button>
      </div>
    </div>

    <div v-if="disabled" class="dev-note">
      <FileText :size="15" />
      <div>
        <strong>当前是开发构建</strong>
        <p>发布构建会从 GitHub Releases 检查新版本。开发构建默认不联网检查，也不会误触发更新。</p>
      </div>
    </div>

    <div v-else-if="available && latest" class="update-available">
      <div class="available-top">
        <div>
          <div class="available-version">
            <strong>v{{ latest.version }}</strong>
            <span v-if="latest.publishedAt" class="subtle small">{{ formatDateTime(latest.publishedAt) }} 发布</span>
          </div>
          <p class="available-name">{{ latest.name || 'NPC 客户端新版本' }}</p>
        </div>
        <div class="available-meta">
          <span>{{ formatBytes(latest.assetSize) }}</span>
          <span class="dot-sep" />
          <span class="mono">{{ latest.assetName }}</span>
        </div>
      </div>
      <div class="release-notes" :class="{ clamped: !showNotes }">
        <p>{{ notes }}</p>
      </div>
      <div class="available-actions">
        <button v-if="showNotes" class="btn btn-ghost btn-sm" type="button" @click="openNotes">
          <FileText :size="13" /> 完整更新说明
        </button>
        <button
          class="btn btn-ghost btn-sm"
          type="button"
          @click="latest?.releaseUrl && app.openExternal(latest.releaseUrl)"
        >
          <ExternalLink :size="13" /> GitHub Release
        </button>
        <span class="grow" />
        <button class="btn btn-primary btn-sm" type="button" @click="download">
          <Download :size="13" /> 下载更新
        </button>
      </div>
    </div>

    <div v-else-if="downloading" class="download-state">
      <div class="download-row">
        <span>{{ progress?.stage || '下载更新' }}</span>
        <span class="mono">{{ formatPercent(progress?.percent) }}</span>
      </div>
      <ProgressBar :value="progress?.percent || 0" :height="6" />
      <div class="download-meta">
        <span class="mono">{{ formatBytes(progress?.received) }} / {{ formatBytes(progress?.total) }}</span>
        <button class="btn btn-ghost btn-sm" type="button" @click="app.cancelUpdate">
          <X :size="12" /> 取消
        </button>
      </div>
    </div>

    <div v-else-if="ready" class="update-ready">
      <div class="ready-mark">
        <CheckCircle2 :size="19" />
      </div>
      <div class="ready-copy">
        <strong>v{{ app.update.stagedVersion || latest?.version }} 已准备就绪</strong>
        <p>
          重启应用后完成安装。
          <span v-if="runtimes.runningCount > 0">当前有 {{ runtimes.runningCount }} 个连接运行中，更新前会先停止它们。</span>
          <span v-else>配置与连接档案会保留。</span>
        </p>
      </div>
      <div class="row">
        <button class="btn btn-ghost btn-sm" type="button" @click="app.discardUpdate">放弃</button>
        <button class="btn btn-primary btn-sm" type="button" @click="requestApply">
          <RotateCw :size="13" /> 重启并更新
        </button>
      </div>
    </div>

    <div v-else-if="failed" class="update-error">
      <ShieldAlert :size="16" />
      <div>
        <p>{{ app.update.error || '更新失败' }}</p>
        <div class="row">
          <button class="btn btn-ghost btn-sm" type="button" @click="check">重试</button>
          <button class="btn btn-ghost btn-sm" type="button" @click="app.openUpdateLog">更新日志</button>
          <button class="btn btn-ghost btn-sm" type="button" @click="app.openReleasePage">手动下载</button>
        </div>
      </div>
    </div>

    <div v-else-if="state === 'idle'" class="idle-state">
      <CheckCircle2 :size="15" />
      <span>已是最新版本。自动检查最多每 24 小时访问一次 GitHub。</span>
      <span v-if="app.update.checkedAt" class="mono subtle">上次检查 {{ formatDateTime(app.update.checkedAt) }}</span>
    </div>

    <ModalDialog :open="notesOpen" title="更新说明" :subtitle="latest?.name || ''" width="620px" @close="notesOpen = false">
      <pre class="notes-full">{{ notes }}</pre>
      <template #footer>
        <button class="btn btn-secondary" type="button" @click="notesOpen = false">关闭</button>
      </template>
    </ModalDialog>

    <ModalDialog
      :open="applyOpen"
      title="重启并安装更新"
      :subtitle="'将先停止 ' + runtimes.runningCount + ' 个正在运行的连接。'"
      width="440px"
      @close="applyOpen = false"
    >
      <p class="muted small">
        更新过程会替换应用文件并自动重启。如果替换失败，旧版本会被恢复。连接档案、密钥和日志都会保留。
      </p>
      <template #footer>
        <button class="btn btn-ghost" type="button" @click="applyOpen = false">稍后</button>
        <button class="btn btn-primary" type="button" @click="apply">
          <RotateCw :size="14" /> 停止连接并更新
        </button>
      </template>
    </ModalDialog>
  </section>
</template>

<style scoped>
.update-card {
  padding: 19px;
}

.update-head {
  display: flex;
  align-items: flex-start;
  gap: 12px;
}

.update-mark {
  display: grid;
  place-items: center;
  width: 40px;
  height: 40px;
  flex: none;
  border: 1px solid var(--border);
  border-radius: 11px;
  background: var(--surface-2);
  color: var(--info);
}

.update-mark.tone-warn {
  color: var(--warn);
  background: var(--warn-soft);
  border-color: rgba(240, 180, 90, 0.24);
}

.update-mark.tone-error {
  color: var(--error);
  background: var(--error-soft);
  border-color: rgba(255, 125, 136, 0.24);
}

.update-mark.tone-ok {
  color: var(--ok);
  background: var(--ok-soft);
  border-color: rgba(70, 213, 151, 0.24);
}

.update-copy {
  flex: 1;
  min-width: 0;
}

.update-copy .row {
  gap: 8px;
  flex-wrap: wrap;
}

.update-actions {
  display: flex;
  align-items: center;
  gap: 4px;
  flex: none;
}

.spin-slow {
  animation: spin 2.4s linear infinite;
}

.dev-note,
.update-error,
.idle-state {
  display: flex;
  align-items: flex-start;
  gap: 9px;
  margin-top: 14px;
  padding: 10px 12px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--bg-soft);
  font-size: 11.5px;
  line-height: 1.5;
}

.dev-note {
  color: var(--text-2);
}

.dev-note strong {
  display: block;
  color: var(--text);
  font-size: 12px;
}

.dev-note p {
  margin-top: 2px;
  color: var(--text-3);
}

.update-error {
  color: var(--error);
  border-color: rgba(255, 125, 136, 0.2);
  background: var(--error-soft);
}

.update-error p {
  margin-bottom: 5px;
  word-break: break-word;
}

.idle-state {
  align-items: center;
  color: var(--ok);
}

.idle-state .mono {
  margin-left: auto;
  font-size: 10px;
}

.update-available {
  margin-top: 14px;
  padding: 13px;
  border: 1px solid rgba(240, 180, 90, 0.2);
  border-radius: var(--radius-md);
  background: linear-gradient(120deg, var(--warn-soft), transparent 58%);
}

.available-top {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 14px;
}

.available-version {
  display: flex;
  align-items: baseline;
  gap: 9px;
}

.available-version strong {
  font-size: 17px;
  letter-spacing: -0.02em;
}

.available-name {
  margin-top: 2px;
  color: var(--text-2);
  font-size: 11.5px;
}

.available-meta {
  display: flex;
  align-items: center;
  gap: 7px;
  color: var(--text-3);
  font-size: 10px;
  flex-wrap: wrap;
  justify-content: flex-end;
}

.dot-sep {
  width: 3px;
  height: 3px;
  border-radius: 50%;
  background: var(--text-4);
}

.release-notes {
  margin-top: 10px;
  padding-top: 10px;
  border-top: 1px solid rgba(240, 180, 90, 0.16);
}

.release-notes p,
.notes-full {
  color: var(--text-2);
  font-family: var(--font-sans);
  font-size: 11.5px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-word;
}

.release-notes.clamped p {
  display: -webkit-box;
  -webkit-line-clamp: 4;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.notes-full {
  max-height: 48vh;
  overflow: auto;
  margin: 0;
}

.available-actions {
  display: flex;
  align-items: center;
  gap: 4px;
  margin-top: 9px;
}

.download-state {
  margin-top: 14px;
  padding: 12px;
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--bg-soft);
}

.download-row,
.download-meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  color: var(--text-2);
  font-size: 11px;
}

.download-row {
  margin-bottom: 7px;
}

.download-meta {
  margin-top: 7px;
  color: var(--text-3);
  font-size: 10px;
}

.update-ready {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-top: 14px;
  padding: 12px;
  border: 1px solid rgba(70, 213, 151, 0.2);
  border-radius: var(--radius-md);
  background: var(--ok-soft);
}

.ready-mark {
  display: grid;
  place-items: center;
  width: 32px;
  height: 32px;
  flex: none;
  border-radius: 50%;
  background: rgba(70, 213, 151, 0.14);
  color: var(--ok);
}

.ready-copy {
  flex: 1;
  min-width: 0;
}

.ready-copy strong {
  font-size: 12px;
}

.ready-copy p {
  margin-top: 2px;
  color: var(--text-2);
  font-size: 11px;
  line-height: 1.45;
}
</style>