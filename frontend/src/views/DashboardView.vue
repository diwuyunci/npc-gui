<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import {
  ArrowRight,
  Cable,
  CircleAlert,
  Download,
  Plus,
  Power,
  Radio,
  RefreshCw,
  ScanLine,
  ShieldAlert,
  Square,
  TerminalSquare,
  TriangleAlert,
} from 'lucide-vue-next'
import { useAppStore } from '@/stores/app'
import { useProfilesStore } from '@/stores/profiles'
import { useRuntimesStore } from '@/stores/runtimes'
import { useLogsStore } from '@/stores/logs'
import { useNow } from '@/composables/useNow'
import { activeState, formatDateTime, isAlertLevel, relativeTime } from '@/lib/format'
import { backend, errorMessage } from '@/lib/backend'
import type { Profile, RuntimeInfo } from '@/lib/types'
import ProfileCard from '@/components/profiles/ProfileCard.vue'
import ImportCommandDialog from '@/components/profiles/ImportCommandDialog.vue'
import EmptyState from '@/components/ui/EmptyState.vue'
import StatusBadge from '@/components/ui/StatusBadge.vue'
import BrandMark from '@/components/layout/BrandMark.vue'

const app = useAppStore()
const profiles = useProfilesStore()
const runtimes = useRuntimesStore()
const logs = useLogsStore()
const router = useRouter()
const now = useNow()
const importOpen = ref(false)
const globalBusy = ref(false)

const sorted = computed<{ profile: Profile; runtime: RuntimeInfo }[]>(() => {
  return [...profiles.items]
    .map((profile) => ({ profile, runtime: runtimes.get(profile.id) }))
    .sort((a, b) => {
      const ar = activeState(a.runtime.state) ? 0 : 1
      const br = activeState(b.runtime.state) ? 0 : 1
      if (ar !== br) return ar - br
      return b.profile.updatedAt - a.profile.updatedAt
    })
})

const alerts = computed(() => logs.alerts.slice(-7).reverse())
const profileName = (id?: string) => profiles.items.find((p) => p.id === id)?.name || '未知连接'

async function start(profile: Profile): Promise<void> {
  try {
    const result = await backend.startProfile(profile.id)
    if (!result.ok) app.toast('error', '启动失败', result.message)
  } catch (err) {
    app.toast('error', '启动失败', errorMessage(err))
  }
}

async function stop(profile: Profile): Promise<void> {
  try {
    const result = await backend.stopProfile(profile.id)
    if (!result.ok) app.toast('error', '停止失败', result.message)
  } catch (err) {
    app.toast('error', '停止失败', errorMessage(err))
  }
}

async function restart(profile: Profile): Promise<void> {
  try {
    const result = await backend.restartProfile(profile.id)
    if (!result.ok) app.toast('error', '重启失败', result.message)
  } catch (err) {
    app.toast('error', '重启失败', errorMessage(err))
  }
}

async function startAll(): Promise<void> {
  if (globalBusy.value) return
  globalBusy.value = true
  try {
    const results = await backend.startAll()
    const failed = results.filter((r) => !r.ok)
    if (failed.length) app.toast('warn', '部分连接启动失败', failed[0].message)
    else app.toast('success', '已启动全部连接')
  } catch (err) {
    app.toast('error', '启动全部失败', errorMessage(err))
  } finally {
    globalBusy.value = false
  }
}

async function stopAll(): Promise<void> {
  if (globalBusy.value) return
  globalBusy.value = true
  try {
    const results = await backend.stopAll()
    const failed = results.filter((r) => !r.ok)
    if (failed.length) app.toast('warn', '部分连接停止失败', failed[0].message)
    else app.toast('success', '已停止全部连接')
  } catch (err) {
    app.toast('error', '停止全部失败', errorMessage(err))
  } finally {
    globalBusy.value = false
  }
}

function edit(profile: Profile): void {
  profiles.select(profile.id)
  void router.push({ name: 'profiles', query: { id: profile.id } })
}

function openLogs(profile: Profile): void {
  void router.push({ name: 'logs', query: { profile: profile.id } })
}

function openLogsById(id?: string): void {
  if (!id) return
  void router.push({ name: 'logs', query: { profile: id } })
}

function newProfile(): void {
  profiles.select('')
  void router.push({ name: 'profiles', query: { new: '1' } })
}

function goLogs(): void {
  void router.push({ name: 'logs' })
}

function lastLogFor(id: string) {
  return logs.get(id).slice(-1)[0]
}

const updateLabel = computed(() => app.update.latest?.version || '')
</script>

<template>
  <div class="page dashboard">
    <header class="page-header">
      <div>
        <div class="kicker">overview</div>
        <h1 class="page-title">连接总览</h1>
        <p class="page-subtitle">本机 npc 进程、连接状态与最近事件，一切都在这里。</p>
      </div>
      <div class="page-actions">
        <button class="btn btn-secondary" type="button" :disabled="globalBusy" @click="stopAll">
          <Square :size="13" /> 全部停止
        </button>
        <button class="btn btn-primary" type="button" :disabled="globalBusy" @click="startAll">
          <Power :size="14" /> 全部启动
        </button>
      </div>
    </header>

    <div v-if="!app.binary" class="banner banner-warn">
      <ShieldAlert :size="17" />
      <div class="banner-copy">
        <strong>尚未配置 npc 二进制</strong>
        <p>下载官方 npc 或选择一个本地文件，然后才能启动连接。</p>
      </div>
      <button class="btn btn-secondary btn-sm" type="button" @click="router.push('/settings')">
        前往设置 <ArrowRight :size="13" />
      </button>
    </div>

    <div v-if="app.updateAvailable" class="banner banner-update">
      <Download :size="17" />
      <div class="banner-copy">
        <strong>发现客户端新版本 {{ updateLabel }}</strong>
        <p>当前版本 {{ app.version }}，可前往设置检查并更新。</p>
      </div>
      <button class="btn btn-secondary btn-sm" type="button" @click="router.push('/settings')">
        查看更新 <ArrowRight :size="13" />
      </button>
    </div>

    <div v-if="app.warnings.length" class="banner banner-warn">
      <TriangleAlert :size="17" />
      <div class="banner-copy">
        <strong>有需要留意的本地状态</strong>
        <p v-for="warning in app.warnings.slice(0, 2)" :key="warning">{{ warning }}</p>
      </div>
    </div>

    <section class="signal-panel">
      <div class="signal-copy">
        <div class="kicker">link status</div>
        <h2>{{ runtimes.onlineCount > 0 ? '隧道已就绪' : '等待建立连接' }}</h2>
        <p>
          {{
            runtimes.onlineCount > 0
              ? 'npc 已与服务端建立加密桥接，流量路径保持在线。'
              : '启动一个连接档案，npc 会在后台维护自动重连与心跳。'
          }}
        </p>
        <div class="signal-actions">
          <button class="btn btn-secondary btn-sm" type="button" @click="goLogs">
            <TerminalSquare :size="13" /> 查看日志
          </button>
          <button class="btn btn-ghost btn-sm" type="button" @click="router.push('/profiles')">
            <Cable :size="13" /> 管理连接
          </button>
        </div>
      </div>

      <div class="signal-visual" :class="{ live: runtimes.onlineCount > 0 }">
        <div class="node local-node">
          <BrandMark :size="26" />
          <div>
            <span class="node-kicker">local</span>
            <strong>本机</strong>
          </div>
        </div>
        <div class="beam">
          <span class="beam-line" />
          <span class="beam-packet p1" />
          <span class="beam-packet p2" />
          <span class="beam-packet p3" />
          <span class="beam-core"><Radio :size="13" /></span>
        </div>
        <div class="node remote-node">
          <div>
            <span class="node-kicker">server</span>
            <strong>NPS 服务端</strong>
          </div>
          <StatusBadge :state="runtimes.onlineCount > 0 ? 'online' : 'stopped'" compact />
        </div>
      </div>

      <div class="signal-foot">
        <div class="signal-stat">
          <span class="mono value">{{ runtimes.onlineCount }}</span>
          <span class="label">在线</span>
        </div>
        <div class="signal-stat">
          <span class="mono value">{{ runtimes.runningCount }}</span>
          <span class="label">运行进程</span>
        </div>
        <div class="signal-stat">
          <span class="mono value">{{ runtimes.reconnects }}</span>
          <span class="label">累计重连</span>
        </div>
        <div class="signal-stat wide">
          <span class="mono value small">{{ app.binary?.version || '未配置' }}</span>
          <span class="label">npc core</span>
        </div>
      </div>
    </section>

    <div class="dashboard-grid">
      <section class="profiles-area">
        <div class="section-head">
          <div>
            <h2 class="section-title">连接档案</h2>
            <p class="section-desc">双击卡片可以进入编辑，运行状态会实时更新。</p>
          </div>
          <div class="row">
            <button class="btn btn-ghost btn-sm" type="button" @click="importOpen = true">
              <ScanLine :size="13" /> 导入命令
            </button>
            <button class="btn btn-secondary btn-sm" type="button" @click="newProfile">
              <Plus :size="14" /> 新建
            </button>
          </div>
        </div>

        <div v-if="sorted.length" class="profile-grid">
          <ProfileCard
            v-for="{ profile, runtime } in sorted"
            :key="profile.id"
            :profile="profile"
            :runtime="runtime"
            :last-log="lastLogFor(profile.id)"
            :now="now"
            @start="start(profile)"
            @stop="stop(profile)"
            @restart="restart(profile)"
            @logs="openLogs(profile)"
            @edit="edit(profile)"
          />
        </div>

        <div v-else class="panel">
          <EmptyState
            title="还没有连接档案"
            description="从 NPS 管理台复制启动命令，或手动创建一个连接。"
          >
            <template #icon><Cable :size="21" /></template>
            <template #actions>
              <button class="btn btn-secondary btn-sm" type="button" @click="importOpen = true">
                <ScanLine :size="13" /> 导入命令
              </button>
              <button class="btn btn-primary btn-sm" type="button" @click="newProfile">
                <Plus :size="14" /> 新建连接
              </button>
            </template>
          </EmptyState>
        </div>
      </section>

      <aside class="side-area">
        <section class="panel activity-panel">
          <div class="section-head">
            <div>
              <h2 class="section-title">最近事件</h2>
              <p class="section-desc">来自所有连接的警告与错误。</p>
            </div>
            <button class="btn btn-ghost btn-sm btn-icon" type="button" aria-label="刷新状态" @click="app.refresh">
              <RefreshCw :size="14" />
            </button>
          </div>

          <div v-if="alerts.length" class="activity-list">
            <button
              v-for="entry in alerts"
              :key="entry.profileId + ':' + entry.seq"
              class="activity-item"
              type="button"
              @click="openLogsById(entry.profileId)"
            >
              <span class="activity-level" :class="'level-' + entry.level">
                {{ entry.level.toUpperCase().slice(0, 3) }}
              </span>
              <span class="activity-copy">
                <span class="activity-name">{{ profileName(entry.profileId) }}</span>
                <span class="activity-text">{{ entry.text }}</span>
              </span>
              <span class="activity-time mono">{{ relativeTime(entry.time) }}</span>
            </button>
          </div>
          <EmptyState v-else title="暂无异常事件" description="连接稳定时这里会保持安静。">
            <template #icon><CircleAlert :size="20" /></template>
          </EmptyState>
        </section>

        <section class="panel quick-panel">
          <div class="section-head">
            <div>
              <h2 class="section-title">快速开始</h2>
              <p class="section-desc">把服务端配置变成一条可用连接。</p>
            </div>
          </div>
          <div class="quick-actions">
            <button class="quick-item" type="button" @click="importOpen = true">
              <span class="quick-icon"><ScanLine :size="17" /></span>
              <span>
                <strong>粘贴 NPS 命令</strong>
                <small>自动拆分为独立配置</small>
              </span>
            </button>
            <button class="quick-item" type="button" @click="newProfile">
              <span class="quick-icon"><Plus :size="17" /></span>
              <span>
                <strong>手动新建连接</strong>
                <small>填写地址、vkey 与连接方式</small>
              </span>
            </button>
            <button class="quick-item" type="button" @click="router.push('/settings')">
              <span class="quick-icon"><Download :size="17" /></span>
              <span>
                <strong>管理 npc 版本</strong>
                <small>{{ app.binary ? '当前 ' + app.binary.version : '尚未安装 npc' }}</small>
              </span>
            </button>
          </div>
        </section>

        <section class="panel meta-panel">
          <div class="meta-row">
            <span>数据目录</span>
            <button class="meta-link mono ellipsis" type="button" @click="app.openDataDir">
              {{ app.appInfo?.dataDir || '...' }}
            </button>
          </div>
          <div class="meta-row">
            <span>客户端版本</span>
            <span class="mono">v{{ app.version }}</span>
          </div>
          <div class="meta-row">
            <span>上次状态刷新</span>
            <span class="mono">{{ formatDateTime(now) }}</span>
          </div>
        </section>
      </aside>
    </div>

    <ImportCommandDialog :open="importOpen" @close="importOpen = false" @imported="app.refresh" />
  </div>
</template>

<style scoped>
.dashboard {
  max-width: 1520px;
}

.banner {
  display: flex;
  align-items: center;
  gap: 13px;
  padding: 12px 15px;
  margin-bottom: 16px;
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--surface);
}

.banner-warn {
  border-color: rgba(240, 180, 90, 0.22);
  background: linear-gradient(90deg, var(--warn-soft), var(--surface) 70%);
  color: var(--warn);
}

.banner-update {
  border-color: rgba(141, 162, 255, 0.22);
  background: linear-gradient(90deg, var(--accent-2-soft), var(--surface) 70%);
  color: var(--accent-2);
}

.banner-copy {
  flex: 1;
  min-width: 0;
}

.banner-copy strong {
  font-size: 12.5px;
}

.banner-copy p {
  margin-top: 2px;
  color: var(--text-2);
  font-size: 11.5px;
  line-height: 1.45;
  word-break: break-word;
}

.signal-panel {
  position: relative;
  display: grid;
  grid-template-columns: minmax(260px, 1fr) minmax(320px, 1.15fr);
  grid-template-rows: auto auto;
  gap: 18px 30px;
  padding: 25px 28px 0;
  margin-bottom: 24px;
  border: 1px solid var(--border);
  border-radius: var(--radius-xl);
  background:
    radial-gradient(520px 220px at 78% 10%, rgba(86, 224, 192, 0.08), transparent 70%),
    linear-gradient(140deg, var(--surface) 0%, var(--surface-2) 100%);
  overflow: hidden;
}

[data-theme='light'] .signal-panel {
  background:
    radial-gradient(520px 220px at 78% 10%, rgba(14, 159, 132, 0.08), transparent 70%),
    linear-gradient(140deg, #fff 0%, var(--surface-2) 100%);
}

.signal-panel::after {
  content: '';
  position: absolute;
  inset: 0;
  pointer-events: none;
  background-image:
    linear-gradient(rgba(120, 160, 190, 0.035) 1px, transparent 1px),
    linear-gradient(90deg, rgba(120, 160, 190, 0.035) 1px, transparent 1px);
  background-size: 34px 34px;
  mask-image: radial-gradient(ellipse at 85% 0%, black, transparent 60%);
}

.signal-copy {
  position: relative;
  z-index: 1;
  padding-bottom: 18px;
}

.signal-copy h2 {
  margin-top: 9px;
  font-size: 28px;
  letter-spacing: -0.035em;
  font-weight: 700;
}

.signal-copy p {
  max-width: 460px;
  margin-top: 7px;
  color: var(--text-2);
  font-size: 12.5px;
  line-height: 1.6;
}

.signal-actions {
  display: flex;
  gap: 6px;
  margin-top: 15px;
}

.signal-visual {
  position: relative;
  z-index: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  min-height: 110px;
  padding-bottom: 18px;
}

.node {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 9px 11px;
  border: 1px solid var(--border);
  border-radius: 10px;
  background: var(--bg-soft);
  box-shadow: var(--shadow-sm);
}

.node-kicker {
  display: block;
  color: var(--text-4);
  font-size: 8.5px;
  letter-spacing: 0.12em;
  text-transform: uppercase;
}

.node strong {
  display: block;
  font-size: 11.5px;
  font-weight: 600;
  white-space: nowrap;
}

.remote-node {
  gap: 12px;
}

.beam {
  position: relative;
  width: clamp(70px, 12vw, 150px);
  height: 34px;
  flex: none;
}

.beam-line {
  position: absolute;
  left: 0;
  right: 0;
  top: 50%;
  height: 1px;
  background: linear-gradient(90deg, transparent, var(--border-strong), transparent);
}

.beam-core {
  position: absolute;
  left: 50%;
  top: 50%;
  transform: translate(-50%, -50%);
  display: grid;
  place-items: center;
  width: 25px;
  height: 25px;
  border-radius: 50%;
  border: 1px solid var(--border-strong);
  background: var(--surface);
  color: var(--text-3);
}

.live .beam-line {
  background: linear-gradient(90deg, transparent, rgba(86, 224, 192, 0.6), transparent);
}

.live .beam-core {
  color: var(--accent);
  border-color: rgba(86, 224, 192, 0.4);
  box-shadow: 0 0 18px rgba(86, 224, 192, 0.25);
}

.beam-packet {
  position: absolute;
  top: 50%;
  width: 4px;
  height: 4px;
  border-radius: 50%;
  background: var(--accent);
  opacity: 0;
  transform: translateY(-50%);
}

.live .beam-packet {
  opacity: 1;
  animation: packet 2.6s linear infinite;
}

.live .p2 {
  animation-delay: 0.9s;
}

.live .p3 {
  animation-delay: 1.8s;
}

@keyframes packet {
  from {
    left: 2%;
    opacity: 0;
  }
  12% {
    opacity: 1;
  }
  88% {
    opacity: 1;
  }
  to {
    left: 96%;
    opacity: 0;
  }
}

.signal-foot {
  position: relative;
  z-index: 1;
  grid-column: 1 / -1;
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  border-top: 1px solid var(--border-soft);
}

.signal-stat {
  display: flex;
  align-items: baseline;
  gap: 8px;
  padding: 13px 0 14px;
}

.signal-stat + .signal-stat {
  padding-left: 20px;
  border-left: 1px solid var(--border-soft);
}

.signal-stat .value {
  font-size: 18px;
  color: var(--text);
  letter-spacing: -0.02em;
}

.signal-stat .value.small {
  font-size: 13px;
}

.signal-stat .label {
  color: var(--text-3);
  font-size: 10.5px;
}

.dashboard-grid {
  display: grid;
  grid-template-columns: minmax(0, 1.55fr) minmax(310px, 0.75fr);
  gap: 22px;
  align-items: start;
}

.profiles-area {
  min-width: 0;
}

.profile-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 13px;
}

.side-area {
  display: flex;
  flex-direction: column;
  gap: 16px;
  position: sticky;
  top: 0;
}

.activity-panel {
  padding: 17px;
}

.activity-list {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.activity-item {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  width: 100%;
  padding: 8px 6px;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: var(--text);
  text-align: left;
}

.activity-item:hover {
  background: var(--surface-2);
}

.activity-level {
  flex: none;
  margin-top: 2px;
  padding: 1px 4px;
  border-radius: 3px;
  font-family: var(--font-mono);
  font-size: 8.5px;
  background: var(--surface-3);
  color: var(--text-3);
}

.level-warn {
  color: var(--warn);
  background: var(--warn-soft);
}

.level-error,
.level-fatal,
.level-panic {
  color: var(--error);
  background: var(--error-soft);
}

.activity-copy {
  min-width: 0;
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 1px;
}

.activity-name {
  color: var(--text-2);
  font-size: 10px;
}

.activity-text {
  color: var(--text-3);
  font-size: 11px;
  line-height: 1.4;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.activity-time {
  flex: none;
  color: var(--text-4);
  font-size: 9px;
  margin-top: 2px;
}

.quick-panel {
  padding: 17px;
}

.quick-actions {
  display: flex;
  flex-direction: column;
  gap: 5px;
}

.quick-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 9px 7px;
  border: 1px solid transparent;
  border-radius: 7px;
  background: transparent;
  color: var(--text);
  text-align: left;
}

.quick-item:hover {
  background: var(--surface-2);
  border-color: var(--border-soft);
}

.quick-icon {
  display: grid;
  place-items: center;
  width: 31px;
  height: 31px;
  flex: none;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--bg-soft);
  color: var(--accent);
}

.quick-item span:last-child {
  display: flex;
  flex-direction: column;
  gap: 1px;
  min-width: 0;
}

.quick-item strong {
  font-size: 11.5px;
  font-weight: 600;
}

.quick-item small {
  color: var(--text-3);
  font-size: 10px;
}

.meta-panel {
  padding: 13px 16px;
  display: flex;
  flex-direction: column;
  gap: 9px;
}

.meta-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  font-size: 10.5px;
  color: var(--text-3);
}

.meta-link {
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--text-3);
  font-size: 9.5px;
  max-width: 210px;
}

.meta-link:hover {
  color: var(--accent);
}

@media (max-width: 1240px) {
  .dashboard-grid {
    grid-template-columns: 1fr;
  }

  .side-area {
    position: static;
    display: grid;
    grid-template-columns: 1fr 1fr;
    align-items: start;
  }

  .meta-panel {
    grid-column: 1 / -1;
  }
}

@media (max-width: 980px) {
  .signal-panel {
    grid-template-columns: 1fr;
  }

  .signal-visual {
    justify-content: flex-start;
  }

  .side-area {
    grid-template-columns: 1fr;
  }

  .meta-panel {
    grid-column: auto;
  }

  .profile-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 720px) {
  .signal-foot {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .signal-stat:nth-child(3) {
    padding-left: 0;
    border-left: 0;
  }

  .signal-stat:nth-child(n + 3) {
    border-top: 1px solid var(--border-soft);
  }
}
</style>