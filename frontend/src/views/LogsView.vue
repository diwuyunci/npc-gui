<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import {
  ArrowDownToLine,
  Eraser,
  Search,
  TerminalSquare,
  WrapText,
} from 'lucide-vue-next'
import type { LogEntry } from '@/lib/types'
import { levelRank } from '@/lib/format'
import { backend, errorMessage } from '@/lib/backend'
import { useAppStore } from '@/stores/app'
import { useProfilesStore } from '@/stores/profiles'
import { useRuntimesStore } from '@/stores/runtimes'
import { useLogsStore } from '@/stores/logs'
import LogConsole from '@/components/logs/LogConsole.vue'
import SegmentedControl from '@/components/ui/SegmentedControl.vue'
import EmptyState from '@/components/ui/EmptyState.vue'

const app = useAppStore()
const profiles = useProfilesStore()
const runtimes = useRuntimesStore()
const logs = useLogsStore()
const route = useRoute()

const selected = ref<string>('all')
const minLevel = ref<string>('debug')
const search = ref('')
const follow = ref(true)
const wrap = ref(false)
const loading = ref(false)

const levelOptions = [
  { value: 'debug', label: '全部' },
  { value: 'info', label: '信息+' },
  { value: 'warn', label: '警告+' },
  { value: 'error', label: '错误' },
]

const profileOptions = computed(() => [
  { id: 'all', name: '全部连接' },
  ...profiles.items.map((p) => ({ id: p.id, name: p.name || '未命名连接' })),
])

const selectedName = computed(() => {
  if (selected.value === 'all') return '全部连接'
  return profiles.items.find((p) => p.id === selected.value)?.name || '未知连接'
})

const currentState = computed(() => {
  if (selected.value === 'all') return null
  return runtimes.get(selected.value).state
})

function sourceEntries(): LogEntry[] {
  if (selected.value === 'all') return logs.all
  return logs.get(selected.value)
}

const filtered = computed(() => {
  const threshold = levelRank(minLevel.value)
  const q = search.value.trim().toLowerCase()
  return sourceEntries()
    .filter((entry) => levelRank(entry.level) >= threshold)
    .filter((entry) => {
      if (!q) return true
      return entry.text.toLowerCase().includes(q) || entry.level.includes(q)
    })
    .slice(-3000)
})

const total = computed(() => sourceEntries().length)
const truncated = computed(() => sourceEntries().length > 3000)

function profileName(id?: string): string {
  if (!id) return '系统'
  return profiles.items.find((p) => p.id === id)?.name || id.slice(0, 6)
}

async function loadHistory(id: string): Promise<void> {
  if (id === 'all') return
  loading.value = true
  try {
    const page = await backend.getLogs(id, 0, 5000)
    logs.addPage(page)
  } catch (err) {
    app.toast('error', '读取日志失败', errorMessage(err))
  } finally {
    loading.value = false
  }
}

function selectProfile(id: string): void {
  selected.value = id
  follow.value = true
  if (id !== 'all') void loadHistory(id)
}

async function clear(): Promise<void> {
  if (selected.value === 'all') {
    app.toast('warn', '请选择具体连接后再清空日志')
    return
  }
  try {
    await backend.clearLogs(selected.value)
    logs.clear(selected.value)
    app.toast('success', '日志已清空')
  } catch (err) {
    app.toast('error', '清空失败', errorMessage(err))
  }
}

async function exportLogs(): Promise<void> {
  if (selected.value === 'all') {
    app.toast('warn', '请选择具体连接后再导出日志')
    return
  }
  try {
    const result = await backend.exportLogs(selected.value)
    if (result.result.ok) app.toast('success', '日志已导出', result.value)
    else app.toast('info', '导出', result.result.message)
  } catch (err) {
    app.toast('error', '导出失败', errorMessage(err))
  }
}

function scrollBottom(): void {
  follow.value = true
}

watch(
  () => route.query.profile,
  (value) => {
    if (typeof value === 'string' && profiles.items.some((p) => p.id === value)) {
      selectProfile(value)
    }
  },
  { immediate: true },
)

onMounted(() => {
  if (selected.value !== 'all') void loadHistory(selected.value)
})
</script>

<template>
  <div class="page logs-page">
    <header class="page-header">
      <div>
        <div class="kicker">console</div>
        <h1 class="page-title">运行日志</h1>
        <p class="page-subtitle">
          {{ selectedName }} · 最新日志实时追加，最多保留 5000 行内存缓冲。
        </p>
      </div>
      <div class="page-actions">
        <button class="btn btn-secondary" type="button" @click="exportLogs">
          <ArrowDownToLine :size="14" /> 导出
        </button>
        <button class="btn btn-ghost" type="button" @click="clear">
          <Eraser :size="14" /> 清空
        </button>
      </div>
    </header>

    <div class="logs-layout">
      <aside class="profiles-rail panel">
        <div class="rail-head">
          <span class="kicker">sources</span>
          <span class="mono tiny subtle">{{ profiles.items.length + 1 }}</span>
        </div>
        <button
          class="source-item"
          :class="{ active: selected === 'all' }"
          type="button"
          @click="selectProfile('all')"
        >
          <span class="source-icon"><TerminalSquare :size="15" /></span>
          <span class="source-copy">
            <span>全部连接</span>
            <small>合并所有进程输出</small>
          </span>
          <span class="source-count mono">{{ logs.all.length }}</span>
        </button>
        <button
          v-for="profile in profiles.items"
          :key="profile.id"
          class="source-item"
          :class="{ active: selected === profile.id }"
          type="button"
          @click="selectProfile(profile.id)"
        >
          <span class="dot" :class="runtimes.get(profile.id).state === 'online' ? 'dot-online' : runtimes.get(profile.id).state === 'error' ? 'dot-error' : ''" />
          <span class="source-copy">
            <span class="ellipsis">{{ profile.name || '未命名连接' }}</span>
            <small class="mono ellipsis">{{ profile.server }}</small>
          </span>
          <span class="source-count mono">{{ logs.get(profile.id).length }}</span>
        </button>

        <div class="rail-foot">
          <span class="hint">共 {{ total }} 行</span>
          <span v-if="loading" class="spinner" />
        </div>
      </aside>

      <section class="console-panel panel">
        <div class="console-toolbar">
          <div class="toolbar-left">
            <SegmentedControl v-model="minLevel" :options="levelOptions" aria-label="日志等级" />
            <div class="search-box">
              <Search :size="13" />
              <input v-model="search" class="search-input" placeholder="搜索日志…" aria-label="搜索日志" />
            </div>
          </div>
          <div class="toolbar-right">
            <span v-if="currentState" class="badge" :class="currentState === 'online' ? 'badge-ok' : currentState === 'error' ? 'badge-error' : 'badge-warn'">
              {{ currentState }}
            </span>
            <button
              class="toolbar-btn"
              :class="{ active: wrap }"
              type="button"
              title="自动换行"
              aria-label="自动换行"
              @click="wrap = !wrap"
            >
              <WrapText :size="14" />
            </button>
            <button
              class="toolbar-btn"
              :class="{ active: follow }"
              type="button"
              title="跟随最新日志"
              aria-label="跟随最新日志"
              @click="scrollBottom"
            >
              <ArrowDownToLine :size="14" />
              <span>跟随</span>
            </button>
          </div>
        </div>

        <div v-if="truncated" class="truncate-tip">仅显示最后 3000 行，更多日志请在磁盘文件中查看。</div>

        <div class="console-wrap">
          <LogConsole
            v-model:follow="follow"
            :entries="filtered"
            :wrap="wrap"
            :show-profile="selected === 'all'"
            :profile-name="profileName"
            :empty-text="selected === 'all' ? '还没有任何连接输出日志。' : '该连接还没有日志。启动后 npc 的输出会实时显示。'"
          />
        </div>

        <footer class="console-foot">
          <span class="mono">{{ filtered.length }} / {{ total }} 行</span>
          <span class="subtle small">日志可能包含 vkey 派生凭据，导出前请确认分享范围。</span>
        </footer>
      </section>
    </div>
  </div>
</template>

<style scoped>
.logs-page {
  max-width: 1520px;
  height: 100vh;
  display: flex;
  flex-direction: column;
  padding-bottom: 24px;
}

.logs-layout {
  display: grid;
  grid-template-columns: 272px minmax(0, 1fr);
  gap: 13px;
  flex: 1;
  min-height: 0;
}

.profiles-rail {
  display: flex;
  flex-direction: column;
  min-height: 0;
  overflow: hidden;
  padding: 8px;
}

.rail-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 8px 10px;
}

.source-item {
  display: flex;
  align-items: center;
  gap: 9px;
  width: 100%;
  padding: 9px 8px;
  border: 1px solid transparent;
  border-radius: 7px;
  background: transparent;
  color: var(--text);
  text-align: left;
}

.source-item:hover {
  background: var(--surface-2);
}

.source-item.active {
  background: var(--surface-3);
  border-color: var(--border);
}

.source-icon {
  display: grid;
  place-items: center;
  width: 26px;
  height: 26px;
  flex: none;
  border-radius: 6px;
  background: var(--bg-soft);
  color: var(--text-3);
}

.source-copy {
  display: flex;
  flex-direction: column;
  gap: 1px;
  min-width: 0;
  flex: 1;
  font-size: 11.5px;
}

.source-copy small {
  color: var(--text-3);
  font-size: 9.5px;
}

.source-count {
  color: var(--text-4);
  font-size: 9.5px;
  flex: none;
}

.source-item > .dot {
  width: 5px;
  height: 5px;
  flex: none;
}

.rail-foot {
  margin-top: auto;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 8px 4px;
  border-top: 1px solid var(--border-soft);
}

.console-panel {
  display: flex;
  flex-direction: column;
  min-width: 0;
  min-height: 0;
  overflow: hidden;
}

.console-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 9px 12px;
  border-bottom: 1px solid var(--border-soft);
  background: var(--surface);
}

.toolbar-left,
.toolbar-right {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.search-box {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 0 8px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--bg-soft);
  color: var(--text-3);
}

.search-box:focus-within {
  border-color: var(--accent);
  box-shadow: 0 0 0 3px var(--accent-soft);
}

.search-input {
  width: 190px;
  height: 30px;
  border: 0;
  outline: 0;
  background: transparent;
  color: var(--text);
  font-size: 11.5px;
}

.toolbar-btn {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  height: 30px;
  padding: 0 8px;
  border: 1px solid transparent;
  border-radius: 5px;
  background: transparent;
  color: var(--text-3);
  font-size: 10.5px;
}

.toolbar-btn:hover {
  background: var(--surface-2);
  color: var(--text);
}

.toolbar-btn.active {
  color: var(--accent);
  background: var(--accent-soft);
}

.truncate-tip {
  padding: 5px 12px;
  border-bottom: 1px solid var(--border-soft);
  background: var(--warn-soft);
  color: var(--warn);
  font-size: 10px;
}

.console-wrap {
  flex: 1;
  min-height: 0;
  border-bottom: 1px solid var(--border-soft);
}

.console-foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 7px 12px;
  background: var(--surface);
  color: var(--text-3);
  font-size: 10px;
}

@media (max-width: 1000px) {
  .logs-layout {
    grid-template-columns: 1fr;
  }

  .profiles-rail {
    flex-direction: row;
    overflow-x: auto;
    max-height: 68px;
  }

  .rail-head,
  .rail-foot {
    display: none;
  }

  .source-item {
    width: auto;
    flex: none;
  }

  .source-copy small,
  .source-count {
    display: none;
  }

  .console-toolbar {
    flex-direction: column;
    align-items: stretch;
  }

  .search-input {
    width: 100%;
  }
}
</style>
