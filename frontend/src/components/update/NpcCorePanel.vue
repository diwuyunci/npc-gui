<script setup lang="ts">
import { computed, ref } from 'vue'
import {
  ArrowUpRight,
  Check,
  Download,
  FileSearch,
  FolderOpen,
  ShieldAlert,
  ShieldCheck,
  X,
} from 'lucide-vue-next'
import type { BinaryInfo } from '@/lib/types'
import { useAppStore } from '@/stores/app'
import { formatBytes, formatPercent } from '@/lib/format'
import ProgressBar from '@/components/ui/ProgressBar.vue'

const props = defineProps<{ mirror: string }>()
const emit = defineEmits<{ 'update:mirror': [value: string] }>()

const app = useAppStore()
const candidates = ref<BinaryInfo[]>([])

const binary = computed(() => app.binary)
const progress = computed(() => app.binaryProgress)
const hasBinary = computed(() => Boolean(binary.value?.path))

async function detect(): Promise<void> {
  const items = await app.detectNpcBinaries()
  candidates.value = items
  app.toast(items.length ? 'success' : 'warn', items.length ? '找到 ' + items.length + ' 个 npc' : '没有找到 npc，请手动选择')
}

async function choose(): Promise<void> {
  await app.pickNpcBinary()
}

async function download(): Promise<void> {
  await app.downloadNpc()
}

async function useCandidate(item: BinaryInfo): Promise<void> {
  await app.setNpcBinary(item.path)
}

function sourceLabel(item: BinaryInfo): string {
  if (item.managed) return 'GUI 下载'
  return item.source || '本地文件'
}

function onMirrorInput(event: Event): void {
  emit('update:mirror', (event.target as HTMLInputElement).value)
}
</script>

<template>
  <section class="panel npc-panel">
    <div class="panel-head">
      <div class="row">
        <div class="npc-mark" :class="{ ready: hasBinary }">
          <ShieldCheck :size="20" />
        </div>
        <div>
          <h2 class="section-title">NPC 核心</h2>
          <p class="section-desc">从上游 djylb/nps 获取 npc，或使用已有的可执行文件。</p>
        </div>
      </div>
      <div class="row">
        <button class="btn btn-ghost btn-sm" type="button" @click="app.openNpcReleasePage">
          <ArrowUpRight :size="12" /> 上游 Releases
        </button>
      </div>
    </div>

    <div v-if="binary" class="binary-current">
      <div class="binary-version">
        <span class="badge badge-ok"><Check :size="11" /> v{{ binary.version }}</span>
        <span class="mono subtle small">core {{ binary.coreVersion || 'unknown' }}</span>
        <span v-if="binary.managed" class="badge badge-info">GUI 管理</span>
        <span v-else class="badge">{{ binary.source || '本地' }}</span>
      </div>
      <div class="binary-path mono ellipsis" :title="binary.path">{{ binary.path }}</div>
      <div v-if="binary.sha256" class="binary-sha mono ellipsis" :title="binary.sha256">sha256 {{ binary.sha256 }}</div>
      <div class="binary-actions">
        <button class="btn btn-secondary btn-sm" type="button" :disabled="app.binaryDownloading" @click="detect">
          <FileSearch :size="13" /> 重新检测
        </button>
        <button class="btn btn-secondary btn-sm" type="button" :disabled="app.binaryDownloading" @click="choose">
          <FolderOpen :size="13" /> 选择文件
        </button>
        <button class="btn btn-primary btn-sm" type="button" :disabled="app.binaryDownloading" @click="download">
          <span v-if="app.binaryDownloading" class="spinner" />
          <Download v-else :size="13" />
          下载最新版
        </button>
      </div>
    </div>

    <div v-else class="binary-missing">
      <ShieldAlert :size="20" />
      <div>
        <strong>还没有可用的 npc</strong>
        <p>下载官方最新版，或选择一个已经在使用的 npc 可执行文件。</p>
      </div>
      <div class="row">
        <button class="btn btn-secondary btn-sm" type="button" :disabled="app.binaryDownloading" @click="detect">
          <FileSearch :size="13" /> 自动检测
        </button>
        <button class="btn btn-secondary btn-sm" type="button" :disabled="app.binaryDownloading" @click="choose">
          <FolderOpen :size="13" /> 选择文件
        </button>
        <button class="btn btn-primary btn-sm" type="button" :disabled="app.binaryDownloading" @click="download">
          <span v-if="app.binaryDownloading" class="spinner" />
          <Download v-else :size="13" />
          下载最新版
        </button>
      </div>
    </div>

    <div v-if="app.binaryDownloading" class="download-block">
      <div class="download-row">
        <span>{{ progress?.stage || '下载 npc' }}</span>
        <span class="mono">{{ formatPercent(progress?.percent) }}</span>
      </div>
      <ProgressBar :value="progress?.percent || 0" :height="5" />
      <div class="download-row meta">
        <span class="mono">{{ formatBytes(progress?.received) }} / {{ formatBytes(progress?.total) }}</span>
        <button class="btn btn-ghost btn-sm" type="button" @click="app.cancelNpcDownload">
          <X :size="12" /> 取消
        </button>
      </div>
    </div>

    <div class="mirror-row">
      <label class="label" for="npc-mirror">下载加速前缀</label>
      <input
        id="npc-mirror"
        class="input mono"
        :value="props.mirror"
        placeholder="可选，例如 https://ghfast.top/"
        spellcheck="false"
        @input="onMirrorInput"
      />
      <span class="hint">仅作用于 GitHub 下载链接；留空直连。修改后需保存设置。</span>
    </div>

    <div v-if="candidates.length" class="candidates">
      <div class="candidates-title">
        <span class="kicker">detected</span>
        <span class="subtle small">点击切换</span>
      </div>
      <button
        v-for="item in candidates"
        :key="item.path"
        class="candidate-row"
        type="button"
        :class="{ selected: item.path === binary?.path }"
        @click="useCandidate(item)"
      >
        <span class="candidate-dot" />
        <span class="candidate-main">
          <span class="mono">{{ item.path }}</span>
          <small>{{ sourceLabel(item) }} · v{{ item.version || '?' }} · core {{ item.coreVersion || '?' }}</small>
        </span>
        <Check v-if="item.path === binary?.path" :size="14" class="check" />
      </button>
    </div>
  </section>
</template>

<style scoped>
.npc-panel {
  padding: 19px;
}

.panel-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
}

.npc-mark {
  display: grid;
  place-items: center;
  width: 40px;
  height: 40px;
  border: 1px solid var(--border);
  border-radius: 11px;
  background: var(--surface-2);
  color: var(--text-3);
}

.npc-mark.ready {
  color: var(--ok);
  background: var(--ok-soft);
  border-color: rgba(70, 213, 151, 0.24);
}

.binary-current {
  margin-top: 15px;
  padding: 13px;
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--bg-soft);
}

[data-theme='light'] .binary-current {
  background: var(--surface-2);
}

.binary-version {
  display: flex;
  align-items: center;
  gap: 7px;
  flex-wrap: wrap;
}

.binary-path {
  margin-top: 8px;
  color: var(--text-2);
  font-size: 11.5px;
}

.binary-sha {
  margin-top: 3px;
  color: var(--text-4);
  font-size: 9.5px;
}

.binary-actions {
  display: flex;
  align-items: center;
  gap: 5px;
  margin-top: 11px;
  flex-wrap: wrap;
}

.binary-missing {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-top: 15px;
  padding: 13px;
  border: 1px solid rgba(240, 180, 90, 0.2);
  border-radius: var(--radius-md);
  background: var(--warn-soft);
  color: var(--warn);
}

.binary-missing > div:nth-child(2) {
  flex: 1;
}

.binary-missing strong {
  font-size: 12px;
}

.binary-missing p {
  margin-top: 2px;
  color: var(--text-2);
  font-size: 11px;
}

.download-block {
  margin-top: 12px;
  padding: 11px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--surface-2);
}

.download-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  font-size: 11px;
  color: var(--text-2);
}

.download-row + .progress {
  margin-top: 7px;
}

.download-row.meta {
  margin-top: 7px;
  color: var(--text-3);
  font-size: 10px;
}

.mirror-row {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin-top: 14px;
}

.mirror-row .label {
  font-size: 11.5px;
}

.mirror-row .input {
  min-height: 34px;
  font-size: 11.5px;
}

.candidates {
  margin-top: 13px;
  border-top: 1px solid var(--border-soft);
  padding-top: 11px;
}

.candidates-title {
  display: flex;
  justify-content: space-between;
  margin-bottom: 5px;
}

.candidate-row {
  display: flex;
  align-items: center;
  gap: 9px;
  width: 100%;
  padding: 8px;
  border: 1px solid transparent;
  border-radius: 6px;
  background: transparent;
  color: var(--text);
  text-align: left;
}

.candidate-row:hover {
  background: var(--surface-2);
}

.candidate-row.selected {
  border-color: var(--border);
  background: var(--surface-2);
}

.candidate-dot {
  width: 5px;
  height: 5px;
  flex: none;
  border-radius: 50%;
  background: var(--text-4);
}

.candidate-row.selected .candidate-dot {
  background: var(--ok);
}

.candidate-main {
  display: flex;
  flex-direction: column;
  gap: 1px;
  min-width: 0;
  flex: 1;
}

.candidate-main .mono {
  font-size: 10.5px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.candidate-main small {
  color: var(--text-3);
  font-size: 9.5px;
}

.check {
  color: var(--ok);
  flex: none;
}
</style>