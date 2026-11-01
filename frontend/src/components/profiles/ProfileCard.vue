<script setup lang="ts">
import { computed } from 'vue'
import {
  Pencil,
  Play,
  RotateCw,
  Square,
  TerminalSquare,
  TriangleAlert,
  ArrowUpRight,
} from 'lucide-vue-next'
import type { LogEntry, Profile, RuntimeInfo } from '@/lib/types'
import { connTypeLabel, formatDuration, isAlertLevel, relativeTime } from '@/lib/format'
import StatusBadge from '@/components/ui/StatusBadge.vue'

const props = defineProps<{
  profile: Profile
  runtime: RuntimeInfo
  lastLog?: LogEntry
  now: number
}>()

const emit = defineEmits<{
  start: []
  stop: []
  restart: []
  logs: []
  edit: []
}>()

const active = computed(() =>
  ['starting', 'running', 'online', 'reconnecting', 'stopping'].includes(props.runtime.state),
)
const online = computed(() => props.runtime.state === 'online')
const uptime = computed(() => formatDuration(props.runtime.startedAt, props.now))
const lastText = computed(() => props.lastLog?.text || '等待启动后输出日志')
const lastLevel = computed(() => props.lastLog?.level || 'info')
const statusClass = computed(() => {
  switch (props.runtime.state) {
    case 'online':
      return 'online'
    case 'starting':
    case 'running':
    case 'reconnecting':
      return 'pending'
    case 'error':
      return 'error'
    default:
      return 'idle'
  }
})
</script>

<template>
  <article class="profile-card" :class="'is-' + statusClass" @dblclick="emit('edit')">
    <header class="card-head">
      <div class="card-title">
        <div class="name-row">
          <h3 class="ellipsis">{{ profile.name || '未命名连接' }}</h3>
          <StatusBadge :state="runtime.state" compact />
          <span v-if="runtime.pendingRestart" class="badge badge-warn restart-badge">
            <TriangleAlert :size="11" /> 待重启
          </span>
        </div>
        <div class="server-line mono">
          <span class="server-dot" />
          <span class="ellipsis">{{ profile.server }}</span>
          <span class="type-chip">{{ connTypeLabel(profile.connType) }}</span>
          <span v-if="profile.autoStart" class="auto-chip">AUTO</span>
        </div>
      </div>
      <button class="open-btn" type="button" aria-label="编辑连接" @click="emit('edit')">
        <ArrowUpRight :size="15" />
      </button>
    </header>

    <div class="card-metrics">
      <div class="metric">
        <span class="metric-label">运行时长</span>
        <span class="metric-value mono">{{ active ? uptime : '--' }}</span>
      </div>
      <div class="metric">
        <span class="metric-label">PID</span>
        <span class="metric-value mono">{{ runtime.pid || '--' }}</span>
      </div>
      <div class="metric">
        <span class="metric-label">重连</span>
        <span class="metric-value mono" :class="{ warn: runtime.reconnects > 0 }">{{ runtime.reconnects }}</span>
      </div>
      <div class="metric">
        <span class="metric-label">最近日志</span>
        <span class="metric-value">{{ runtime.lastLogAt ? relativeTime(runtime.lastLogAt, now) : '--' }}</span>
      </div>
    </div>

    <div class="last-log" :class="{ alert: isAlertLevel(lastLevel) }">
      <span class="log-level mono">{{ lastLevel.toUpperCase().slice(0, 3) }}</span>
      <span class="log-text ellipsis">{{ lastText }}</span>
    </div>

    <div v-if="runtime.state === 'error' && runtime.lastError" class="error-line ellipsis" :title="runtime.lastError">
      <TriangleAlert :size="12" />
      <span>{{ runtime.lastError }}</span>
    </div>

    <footer class="card-foot">
      <div class="card-actions">
        <button
          v-if="!active"
          class="btn btn-primary btn-sm"
          type="button"
          @click="emit('start')"
        >
          <Play :size="13" fill="currentColor" /> 启动
        </button>
        <button v-else class="btn btn-secondary btn-sm" type="button" @click="emit('stop')">
          <Square :size="12" fill="currentColor" /> 停止
        </button>
        <button class="btn btn-ghost btn-sm btn-icon" type="button" aria-label="重启" title="重启" @click="emit('restart')">
          <RotateCw :size="14" />
        </button>
        <button class="btn btn-ghost btn-sm btn-icon" type="button" aria-label="查看日志" title="查看日志" @click="emit('logs')">
          <TerminalSquare :size="14" />
        </button>
        <button class="btn btn-ghost btn-sm btn-icon" type="button" aria-label="编辑" title="编辑" @click="emit('edit')">
          <Pencil :size="14" />
        </button>
      </div>
      <span v-if="online" class="online-mark">
        <span class="dot dot-online" /> 隧道在线
      </span>
      <span v-else class="subtle small">{{ profile.remark || '点击卡片查看配置' }}</span>
    </footer>
  </article>
</template>

<style scoped>
.profile-card {
  position: relative;
  display: flex;
  flex-direction: column;
  gap: 13px;
  padding: 17px 17px 14px;
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  background: linear-gradient(180deg, rgba(255, 255, 255, 0.018), transparent 46%), var(--surface);
  overflow: hidden;
  transition:
    border-color var(--speed) var(--ease),
    transform var(--speed) var(--ease),
    background var(--speed) var(--ease);
}

.profile-card:hover {
  border-color: var(--border-strong);
  transform: translateY(-1px);
  background: linear-gradient(180deg, rgba(255, 255, 255, 0.03), transparent 46%), var(--surface);
}

.card-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 10px;
}

.card-title {
  min-width: 0;
}

.name-row {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.name-row h3 {
  font-size: 15px;
  font-weight: 650;
  letter-spacing: -0.01em;
  min-width: 0;
}

.restart-badge {
  height: 19px;
  font-size: 9.5px;
  padding: 0 6px;
}

.server-line {
  display: flex;
  align-items: center;
  gap: 7px;
  margin-top: 7px;
  color: var(--text-3);
  font-size: 11.5px;
  min-width: 0;
}

.server-dot {
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: var(--text-4);
  flex: none;
}

.type-chip,
.auto-chip {
  flex: none;
  height: 16px;
  padding: 0 5px;
  border: 1px solid var(--border);
  border-radius: 4px;
  color: var(--text-4);
  font-family: var(--font-sans);
  font-size: 9px;
  line-height: 14px;
  letter-spacing: 0.04em;
}

.auto-chip {
  color: var(--accent-2);
  border-color: rgba(141, 162, 255, 0.2);
  background: var(--accent-2-soft);
}

.open-btn {
  display: grid;
  place-items: center;
  width: 28px;
  height: 28px;
  flex: none;
  border: 1px solid transparent;
  border-radius: 6px;
  background: transparent;
  color: var(--text-4);
  opacity: 0;
  transition:
    opacity var(--speed) var(--ease),
    background var(--speed) var(--ease),
    color var(--speed) var(--ease);
}

.profile-card:hover .open-btn {
  opacity: 1;
}

.open-btn:hover {
  background: var(--surface-3);
  color: var(--text);
}

.card-metrics {
  display: grid;
  grid-template-columns: 1.25fr 0.8fr 0.7fr 1.1fr;
  gap: 8px;
  padding: 10px 0;
  border-top: 1px solid var(--border-soft);
  border-bottom: 1px solid var(--border-soft);
}

.metric {
  display: flex;
  flex-direction: column;
  gap: 3px;
  min-width: 0;
}

.metric-label {
  color: var(--text-4);
  font-size: 9.5px;
  letter-spacing: 0.03em;
}

.metric-value {
  color: var(--text-2);
  font-size: 11.5px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.metric-value.warn {
  color: var(--warn);
}

.last-log {
  display: flex;
  align-items: center;
  gap: 7px;
  min-width: 0;
  padding: 6px 8px;
  border-radius: 6px;
  background: var(--bg-soft);
  color: var(--text-3);
  font-family: var(--font-mono);
  font-size: 10.5px;
}

[data-theme='light'] .last-log {
  background: var(--surface-2);
}

.last-log.alert {
  background: var(--warn-soft);
  color: var(--warn);
}

.log-level {
  flex: none;
  font-size: 9px;
  opacity: 0.8;
}

.log-text {
  min-width: 0;
}

.error-line {
  display: flex;
  align-items: center;
  gap: 6px;
  color: var(--error);
  font-size: 11px;
}

.card-foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  min-height: 30px;
}

.card-actions {
  display: flex;
  align-items: center;
  gap: 2px;
}

.online-mark {
  display: flex;
  align-items: center;
  gap: 6px;
  color: var(--ok);
  font-size: 10.5px;
  white-space: nowrap;
}

.online-mark .dot {
  width: 5px;
  height: 5px;
}
</style>