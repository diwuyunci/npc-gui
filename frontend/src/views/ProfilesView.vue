<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  Cable,
  FileJson,
  Plus,
  ScanLine,
  Search,
  ShieldAlert,
  Trash2,
  TriangleAlert,
} from 'lucide-vue-next'
import type { Profile, RuntimeInfo } from '@/lib/types'
import { emptyProfile } from '@/lib/types'
import { backend, errorMessage } from '@/lib/backend'
import { activeState, stateLabel } from '@/lib/format'
import { useAppStore } from '@/stores/app'
import { useProfilesStore } from '@/stores/profiles'
import { useRuntimesStore } from '@/stores/runtimes'
import ProfileEditor from '@/components/profiles/ProfileEditor.vue'
import ImportCommandDialog from '@/components/profiles/ImportCommandDialog.vue'
import StatusBadge from '@/components/ui/StatusBadge.vue'
import ModalDialog from '@/components/ui/ModalDialog.vue'
import EmptyState from '@/components/ui/EmptyState.vue'

const app = useAppStore()
const profiles = useProfilesStore()
const runtimes = useRuntimesStore()
const route = useRoute()
const router = useRouter()

const creating = ref(false)
const draft = ref<Profile>(emptyProfile())
const saving = ref(false)
const revision = ref(0)
const fieldErrors = ref<Record<string, string>>({})
const editorDirty = ref(false)
const editor = ref<InstanceType<typeof ProfileEditor> | null>(null)
const importOpen = ref(false)
const deleteTarget = ref<Profile | null>(null)
const discardOpen = ref(false)
const pendingSelectId = ref<string | null>(null)
const pendingCreate = ref(false)

const current = computed<Profile>(() => {
  if (creating.value) return draft.value
  return profiles.selected ?? emptyProfile()
})

const currentRuntime = computed<RuntimeInfo>(() => {
  if (creating.value) return { profileId: '', state: 'stopped', reconnects: 0, pendingRestart: false }
  return runtimes.get(profiles.selectedId)
})

const hasEditor = computed(() => creating.value || Boolean(profiles.selected))

function makeNewDraft(): Profile {
  const p = emptyProfile()
  p.name = ''
  p.logLevel = app.settings.defaultLogLevel || 'info'
  p.autoReconnection = true
  return p
}

function requestCreate(): void {
  if (editorDirty.value) {
    pendingCreate.value = true
    pendingSelectId.value = null
    discardOpen.value = true
    return
  }
  startCreate()
}

function startCreate(): void {
  creating.value = true
  draft.value = makeNewDraft()
  profiles.select('')
  fieldErrors.value = {}
  editorDirty.value = false
  void router.replace({ name: 'profiles', query: { new: '1' } })
}

function requestSelect(id: string): void {
  if (id === profiles.selectedId && !creating.value) return
  if (editorDirty.value) {
    pendingSelectId.value = id
    pendingCreate.value = false
    discardOpen.value = true
    return
  }
  selectNow(id)
}

function selectNow(id: string): void {
  creating.value = false
  draft.value = emptyProfile()
  profiles.select(id)
  fieldErrors.value = {}
  editorDirty.value = false
  void router.replace({ name: 'profiles', query: { id } })
}

function confirmDiscard(): void {
  discardOpen.value = false
  editorDirty.value = false
  if (pendingCreate.value) {
    pendingCreate.value = false
    startCreate()
    return
  }
  if (pendingSelectId.value) {
    const id = pendingSelectId.value
    pendingSelectId.value = null
    selectNow(id)
  }
}

function cancelDiscard(): void {
  discardOpen.value = false
  pendingSelectId.value = null
  pendingCreate.value = false
}

function onDirty(value: boolean): void {
  editorDirty.value = value
}

async function save(profile: Profile): Promise<void> {
  if (saving.value) return
  saving.value = true
  fieldErrors.value = {}
  try {
    const result = await profiles.save(profile)
    if (result.result.ok) {
      creating.value = false
      draft.value = emptyProfile()
      app.toast('success', '已保存连接', result.profile.name)
      revision.value++
      editorDirty.value = false
      await router.replace({ name: 'profiles', query: { id: result.profile.id } })
      if (result.pendingRestart) app.toast('warn', '配置已保存', '重启该连接后生效')
    } else {
      fieldErrors.value = result.result.fieldErrors ?? {}
      app.toast('error', '保存失败', result.result.message)
    }
  } catch (err) {
    app.toast('error', '保存失败', errorMessage(err))
  } finally {
    saving.value = false
  }
}

async function duplicate(): Promise<void> {
  const source = profiles.selected
  if (!source) return
  try {
    const result = await profiles.duplicate(source.id)
    if (result.result.ok) {
      app.toast('success', '已复制连接', result.profile.name)
      selectNow(result.profile.id)
    } else {
      app.toast('error', '复制失败', result.result.message)
    }
  } catch (err) {
    app.toast('error', '复制失败', errorMessage(err))
  }
}

function askDelete(): void {
  if (profiles.selected) deleteTarget.value = profiles.selected
}

async function confirmDelete(): Promise<void> {
  if (!deleteTarget.value) return
  const target = deleteTarget.value
  deleteTarget.value = null
  try {
    const result = await profiles.deleteProfile(target.id)
    if (result.ok) {
      app.toast('success', '已删除连接', target.name)
      editorDirty.value = false
      const next = profiles.items[0]
      if (next) selectNow(next.id)
      else {
        creating.value = false
        profiles.select('')
        void router.replace({ name: 'profiles' })
      }
    } else {
      app.toast('error', '删除失败', result.message)
    }
  } catch (err) {
    app.toast('error', '删除失败', errorMessage(err))
  }
}

async function importFromFile(): Promise<void> {
  try {
    const result = await profiles.importFromFile()
    if (result.result.ok) {
      app.toast('success', '已导入 ' + result.profiles.length + ' 个连接', result.result.warnings?.join('；'))
      if (result.profiles[0]) selectNow(result.profiles[0].id)
    } else {
      app.toast('error', '导入失败', result.result.message)
    }
  } catch (err) {
    app.toast('error', '导入失败', errorMessage(err))
  }
}

async function start(): Promise<void> {
  if (!profiles.selected) return
  const result = await backend.startProfile(profiles.selected.id)
  if (!result.ok) app.toast('error', '启动失败', result.message)
}

async function stop(): Promise<void> {
  if (!profiles.selected) return
  const result = await backend.stopProfile(profiles.selected.id)
  if (!result.ok) app.toast('error', '停止失败', result.message)
}

async function restart(): Promise<void> {
  if (!profiles.selected) return
  const result = await backend.restartProfile(profiles.selected.id)
  if (!result.ok) app.toast('error', '重启失败', result.message)
}

async function exportConfig(): Promise<void> {
  if (!profiles.selected) return
  try {
    const result = await backend.exportConfig(profiles.selected.id)
    if (result.result.ok) app.toast('success', '配置已导出', result.value)
    else app.toast('info', '导出', result.result.message)
  } catch (err) {
    app.toast('error', '导出失败', errorMessage(err))
  }
}

function importDone(): void {
  const first = profiles.items[profiles.items.length - 1]
  if (first) selectNow(first.id)
}

function statusDot(id: string): string {
  const runtime = runtimes.get(id)
  if (runtime.state === 'online') return 'dot-online'
  if (runtime.state === 'error') return 'dot-error'
  if (activeState(runtime.state)) return 'dot-warn'
  return ''
}

onMounted(() => {
  const id = typeof route.query.id === 'string' ? route.query.id : ''
  const isNew = route.query.new === '1'
  if (isNew) {
    startCreate()
    return
  }
  if (id) {
    const found = profiles.items.find((p) => p.id === id)
    if (found) {
      profiles.select(id)
      return
    }
  }
  if (!profiles.selectedId && profiles.items[0]) {
    profiles.select(profiles.items[0].id)
  }
})

watch(
  () => profiles.items.length,
  (length) => {
    if (!length) {
      creating.value = false
      profiles.select('')
    }
  },
)
</script>

<template>
  <div class="page profiles-page">
    <header class="page-header">
      <div>
        <div class="kicker">profiles</div>
        <h1 class="page-title">连接档案</h1>
        <p class="page-subtitle">每个档案对应一个独立的 npc 进程和配置文件，可同时运行多个。</p>
      </div>
      <div class="page-actions">
        <button class="btn btn-ghost" type="button" @click="importFromFile">
          <FileJson :size="14" /> 导入文件
        </button>
        <button class="btn btn-secondary" type="button" @click="importOpen = true">
          <ScanLine :size="14" /> 导入命令
        </button>
        <button class="btn btn-primary" type="button" @click="requestCreate">
          <Plus :size="14" /> 新建连接
        </button>
      </div>
    </header>

    <div v-if="!app.binary" class="inline-banner">
      <ShieldAlert :size="15" />
      <span>尚未配置 npc，保存后仍无法启动。请先到设置中获取 npc。</span>
      <router-link to="/settings" class="banner-link">去设置</router-link>
    </div>

    <div class="profiles-layout">
      <aside class="list-panel panel">
        <div class="list-head">
          <div class="search-box">
            <Search :size="14" />
            <input v-model="profiles.search" class="search-input" placeholder="搜索名称或地址" aria-label="搜索连接" />
          </div>
          <span class="list-count mono">{{ profiles.filtered.length }}</span>
        </div>

        <div class="profile-list">
          <button
            v-if="creating"
            class="list-item active"
            type="button"
          >
            <span class="dot dot-warn" />
            <span class="list-copy">
              <span class="list-name">新建连接</span>
              <span class="list-server mono">草稿</span>
            </span>
            <span class="badge">未保存</span>
          </button>

          <button
            v-for="profile in profiles.filtered"
            :key="profile.id"
            class="list-item"
            :class="{ active: !creating && profile.id === profiles.selectedId }"
            type="button"
            @click="requestSelect(profile.id)"
          >
            <span class="dot" :class="statusDot(profile.id)" />
            <span class="list-copy">
              <span class="list-name">{{ profile.name || '未命名连接' }}</span>
              <span class="list-server mono">{{ profile.server }}</span>
            </span>
            <StatusBadge :state="runtimes.get(profile.id).state" :show-dot="false" compact />
          </button>

          <div v-if="!profiles.filtered.length && profiles.items.length" class="list-empty">
            没有匹配的连接
          </div>
          <div v-if="!profiles.items.length && !creating" class="list-empty">
            还没有连接档案
          </div>
        </div>

        <div class="list-foot">
          <span class="hint">{{ runtimes.onlineCount }} 在线 / {{ profiles.items.length }} 档案</span>
          <button class="btn btn-ghost btn-sm btn-icon" type="button" aria-label="新建连接" @click="requestCreate">
            <Plus :size="14" />
          </button>
        </div>
      </aside>

      <section class="editor-panel panel">
        <ProfileEditor
          v-if="hasEditor"
          ref="editor"
          :key="(creating ? 'new' : profiles.selectedId) + ':' + revision"
          :profile="current"
          :runtime="currentRuntime"
          :field-errors="fieldErrors"
          :saving="saving"
          :pending-restart="currentRuntime.pendingRestart"
          @save="save"
          @start="start"
          @stop="stop"
          @restart="restart"
          @duplicate="duplicate"
          @delete="askDelete"
          @export-config="exportConfig"
          @dirty="onDirty"
          @clear-errors="fieldErrors = {}"
        />
        <EmptyState
          v-else
          title="选择一个连接"
          description="从左侧选择已保存的连接，或新建一个档案开始。"
        >
          <template #icon><Cable :size="21" /></template>
          <template #actions>
            <button class="btn btn-primary btn-sm" type="button" @click="requestCreate">
              <Plus :size="14" /> 新建连接
            </button>
          </template>
        </EmptyState>
      </section>
    </div>

    <ImportCommandDialog :open="importOpen" @close="importOpen = false" @imported="importDone" />

    <ModalDialog
      :open="discardOpen"
      title="放弃未保存的更改？"
      subtitle="当前档案有尚未保存的修改。"
      width="420px"
      @close="cancelDiscard"
    >
      <p class="muted small">继续操作会丢失这些修改。</p>
      <template #footer>
        <button class="btn btn-ghost" type="button" @click="cancelDiscard">继续编辑</button>
        <button class="btn btn-danger" type="button" @click="confirmDiscard">放弃更改</button>
      </template>
    </ModalDialog>

    <ModalDialog
      :open="deleteTarget !== null"
      title="删除连接档案"
      :subtitle="deleteTarget?.name || ''"
      width="440px"
      @close="deleteTarget = null"
    >
      <div class="delete-warning">
        <Trash2 :size="18" />
        <p>将删除该档案的本地配置与生成文件。服务端的客户端记录不会自动删除。</p>
      </div>
      <template #footer>
        <button class="btn btn-ghost" type="button" @click="deleteTarget = null">取消</button>
        <button class="btn btn-danger" type="button" @click="confirmDelete">删除</button>
      </template>
    </ModalDialog>
  </div>
</template>

<style scoped>
.profiles-page {
  max-width: 1520px;
  display: flex;
  flex-direction: column;
  min-height: 100vh;
}

.inline-banner {
  display: flex;
  align-items: center;
  gap: 9px;
  padding: 9px 12px;
  margin-bottom: 14px;
  border: 1px solid rgba(240, 180, 90, 0.22);
  border-radius: var(--radius-sm);
  background: var(--warn-soft);
  color: var(--warn);
  font-size: 11.5px;
}

.inline-banner span {
  flex: 1;
}

.banner-link {
  color: var(--warn);
  font-weight: 600;
}

.profiles-layout {
  display: grid;
  grid-template-columns: 318px minmax(0, 1fr);
  gap: 14px;
  align-items: stretch;
  min-height: 0;
  flex: 1;
}

.list-panel {
  display: flex;
  flex-direction: column;
  min-height: 0;
  max-height: calc(100vh - 190px);
  overflow: hidden;
}

.list-head {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px;
  border-bottom: 1px solid var(--border-soft);
}

.search-box {
  display: flex;
  align-items: center;
  gap: 7px;
  flex: 1;
  min-width: 0;
  padding: 0 9px;
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
  width: 100%;
  height: 32px;
  border: 0;
  outline: 0;
  background: transparent;
  color: var(--text);
  font-size: 12px;
}

.list-count {
  color: var(--text-4);
  font-size: 10.5px;
  min-width: 20px;
  text-align: right;
}

.profile-list {
  flex: 1;
  overflow: auto;
  padding: 6px;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.list-item {
  display: flex;
  align-items: center;
  gap: 9px;
  width: 100%;
  padding: 9px 9px;
  border: 1px solid transparent;
  border-radius: 7px;
  background: transparent;
  color: var(--text);
  text-align: left;
  transition:
    background var(--speed) var(--ease),
    border-color var(--speed) var(--ease);
}

.list-item:hover {
  background: var(--surface-2);
}

.list-item.active {
  background: var(--surface-3);
  border-color: var(--border);
}

.list-item > .dot {
  width: 6px;
  height: 6px;
  flex: none;
}

.list-copy {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
  flex: 1;
}

.list-name {
  font-size: 12.5px;
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.list-server {
  color: var(--text-3);
  font-size: 10px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.list-item :deep(.status) {
  font-size: 9.5px;
  height: 19px;
  padding: 0 6px;
  border: 0;
  background: transparent;
  opacity: 0.85;
}

.list-empty {
  padding: 36px 10px;
  text-align: center;
  color: var(--text-3);
  font-size: 11.5px;
}

.list-foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 8px 12px;
  border-top: 1px solid var(--border-soft);
}

.editor-panel {
  min-width: 0;
  min-height: 0;
  max-height: calc(100vh - 190px);
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.delete-warning {
  display: flex;
  gap: 11px;
  align-items: flex-start;
  color: var(--error);
}

.delete-warning p {
  color: var(--text-2);
  font-size: 12px;
  line-height: 1.55;
}

@media (max-width: 1100px) {
  .profiles-layout {
    grid-template-columns: 270px minmax(0, 1fr);
  }
}

@media (max-width: 860px) {
  .profiles-layout {
    grid-template-columns: 1fr;
  }

  .list-panel {
    max-height: 280px;
  }

  .editor-panel {
    max-height: none;
  }
}
</style>