<script setup lang="ts">
import { onMounted, onUnmounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAppStore } from '@/stores/app'
import { useProfilesStore } from '@/stores/profiles'
import { useRuntimesStore } from '@/stores/runtimes'
import { useLogsStore } from '@/stores/logs'
import { onBackendEvent, type Unsubscribe } from '@/lib/events'
import type {
  BinaryResult,
  LogPage,
  RuntimeInfo,
  Settings,
  UpdateProgress,
  UpdateStateInfo,
} from '@/lib/types'
import BootScreen from '@/components/layout/BootScreen.vue'
import ToastStack from '@/components/ui/ToastStack.vue'
import ModalDialog from '@/components/ui/ModalDialog.vue'
import BrandMark from '@/components/layout/BrandMark.vue'

const app = useAppStore()
const profiles = useProfilesStore()
const runtimes = useRuntimesStore()
const logs = useLogsStore()
const router = useRouter()
const route = useRoute()

let subscriptions: Unsubscribe[] = []
let subscribed = false
let refreshing = false

function subscribe(): void {
  if (subscribed) return
  subscribed = true
  subscriptions = [
    onBackendEvent<RuntimeInfo>('runtime:state', (info) => {
      if (info.state === 'starting') logs.clear(info.profileId)
      runtimes.upsert(info)
    }),
    onBackendEvent<LogPage>('runtime:logs', (page) => logs.addPage(page)),
    onBackendEvent<UpdateProgress>('binary:progress', (progress) => {
      app.binaryProgress = progress
    }),
    onBackendEvent<BinaryResult>('binary:info', (res) => {
      if (res.binary) app.binary = res.binary
      if (res.candidates) app.binaryCandidates = res.candidates
    }),
    onBackendEvent<UpdateStateInfo>('update:state', (state) => {
      app.update = state
    }),
    onBackendEvent<UpdateProgress>('update:progress', (progress) => {
      app.updateProgress = progress
    }),
    onBackendEvent<Settings>('settings:changed', (settings) => {
      app.settings = { ...settings }
      document.documentElement.dataset.theme = settings.theme || 'dark'
    }),
    onBackendEvent<{ level?: string; message?: string }>('app:notice', (notice) => {
      if (!notice?.message) return
      const level = (notice.level || 'info') as 'info' | 'success' | 'warn' | 'error'
      app.toast(level, notice.message)
    }),
    onBackendEvent<{ running: number }>('app:close-request', (payload) => {
      app.setCloseRequest(payload?.running ?? 0)
    }),
    onBackendEvent<null>('profiles:changed', () => {
      if (refreshing) return
      refreshing = true
      window.setTimeout(async () => {
        refreshing = false
        await app.refresh()
      }, 120)
    }),
  ]
}

function unsubscribe(): void {
  for (const fn of subscriptions) fn()
  subscriptions = []
  subscribed = false
}

async function boot(): Promise<void> {
  subscribe()
  await app.load()
  if (app.bootstrapped && app.needsSetup && !app.setupSkipped && route.name !== 'setup') {
    await router.replace({ name: 'setup' })
  }
}

function skipSetup(): void {
  app.setupSkipped = true
}

function confirmQuit(): void {
  app.setCloseRequest(null)
  void app.confirmQuit()
}

function cancelQuit(): void {
  app.setCloseRequest(null)
}

function onBeforeUnload(event: BeforeUnloadEvent): void {
  if (app.closeRequest !== null) {
    event.preventDefault()
  }
}

watch(
  () => app.settings.theme,
  (theme) => {
    document.documentElement.dataset.theme = theme || 'dark'
  },
  { immediate: true },
)

watch(
  () => route.name,
  (name) => {
    if (name === 'setup') return
    if (app.bootstrapped && app.needsSetup && !app.setupSkipped) {
      void router.replace({ name: 'setup' })
    }
  },
)

onMounted(() => {
  void boot()
  window.addEventListener('beforeunload', onBeforeUnload)
})

onUnmounted(() => {
  unsubscribe()
  window.removeEventListener('beforeunload', onBeforeUnload)
})
</script>

<template>
  <div class="app-root">
    <BootScreen v-if="app.loading" />

    <div v-else-if="app.fatal" class="fatal-screen">
      <div class="fatal-card">
        <BrandMark :size="46" />
        <h1>应用初始化失败</h1>
        <p>{{ app.fatal }}</p>
        <div class="fatal-actions">
          <button class="btn btn-primary" type="button" @click="boot">重试</button>
          <button class="btn btn-ghost" type="button" @click="app.openDataDir">打开数据目录</button>
        </div>
      </div>
    </div>

    <RouterView v-else v-slot="{ Component }">
      <component :is="Component" @skip-setup="skipSetup" />
    </RouterView>

    <ToastStack />
    <ModalDialog
      :open="app.closeRequest !== null"
      title="仍有连接正在运行"
      subtitle="关闭应用会停止所有 npc 进程。"
      width="440px"
      @close="cancelQuit"
    >
      <div class="close-modal">
        <div class="close-count">{{ app.closeRequest ?? 0 }}</div>
        <div>
          <p>个连接仍在运行</p>
          <p class="muted">退出后这些隧道会立即断开，服务端将显示客户端离线。</p>
        </div>
      </div>
      <template #footer>
        <button class="btn btn-ghost" type="button" @click="cancelQuit">取消</button>
        <button class="btn btn-danger" type="button" @click="confirmQuit">停止并退出</button>
      </template>
    </ModalDialog>
  </div>
</template>

<style scoped>
.app-root {
  min-height: 100vh;
}

.close-modal {
  display: flex;
  gap: 18px;
  align-items: center;
  padding: 6px 0 10px;
}

.close-count {
  font-family: var(--font-mono);
  font-size: 44px;
  line-height: 1;
  color: var(--warn);
  min-width: 58px;
  text-align: center;
}

.close-modal p {
  margin: 0 0 4px;
}

.fatal-screen {
  min-height: 100vh;
  display: grid;
  place-items: center;
  padding: 24px;
}

.fatal-card {
  width: min(520px, 100%);
  padding: 32px;
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  background: var(--surface);
  text-align: center;
}

.fatal-card h1 {
  font-size: 22px;
  margin: 18px 0 8px;
}

.fatal-card p {
  color: var(--text-2);
  line-height: 1.6;
}

.fatal-actions {
  display: flex;
  gap: 10px;
  justify-content: center;
  margin-top: 18px;
}
</style>