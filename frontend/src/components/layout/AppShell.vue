<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import {
  LayoutDashboard,
  Cable,
  TerminalSquare,
  Settings2,
  ShieldCheck,
  Download,
  CircleDot,
} from 'lucide-vue-next'
import { useAppStore } from '@/stores/app'
import { useRuntimesStore } from '@/stores/runtimes'
import BrandMark from './BrandMark.vue'

const app = useAppStore()
const runtimes = useRuntimesStore()
const route = useRoute()

const nav = [
  { name: 'dashboard', label: '总览', icon: LayoutDashboard, to: '/' },
  { name: 'profiles', label: '连接', icon: Cable, to: '/profiles' },
  { name: 'logs', label: '日志', icon: TerminalSquare, to: '/logs' },
  { name: 'settings', label: '设置', icon: Settings2, to: '/settings' },
]

const binaryLabel = computed(() => app.binary?.version || '未配置 npc')
const binaryTone = computed(() => (app.binary ? 'ok' : 'error'))
const current = computed(() => String(route.name || 'dashboard'))
</script>

<template>
  <div class="shell">
    <aside class="rail">
      <div class="brand">
        <BrandMark :size="36" />
        <div class="brand-copy">
          <strong>NPC</strong>
          <span>客户端</span>
        </div>
      </div>

      <nav class="nav" aria-label="主导航">
        <RouterLink
          v-for="item in nav"
          :key="item.name"
          :to="item.to"
          class="nav-item"
          :class="{ active: current === item.name }"
          :aria-current="current === item.name ? 'page' : undefined"
        >
          <component :is="item.icon" :size="18" :stroke-width="1.8" />
          <span class="nav-label">{{ item.label }}</span>
          <span
            v-if="item.name === 'settings' && app.updateAvailable"
            class="nav-update-dot"
            aria-label="有可用更新"
          />
          <span v-if="item.name === 'dashboard' && runtimes.onlineCount > 0" class="nav-count mono">
            {{ runtimes.onlineCount }}
          </span>
        </RouterLink>
      </nav>

      <div class="rail-spacer" />

      <div class="rail-status">
        <RouterLink to="/settings" class="status-link">
          <span class="status-icon">
            <ShieldCheck :size="16" :stroke-width="1.8" />
            <span class="dot" :class="app.binary ? 'dot-online' : 'dot-error'" />
          </span>
          <span class="status-copy">
            <span class="status-kicker">npc core</span>
            <span class="status-value mono ellipsis">{{ binaryLabel }}</span>
          </span>
        </RouterLink>

        <RouterLink v-if="app.updateAvailable" to="/settings" class="update-chip">
          <Download :size="13" />
          <span>可更新</span>
          <span class="mono">{{ app.update.latest?.version }}</span>
        </RouterLink>

        <div class="rail-version mono">
          <CircleDot :size="12" />
          <span>v{{ app.version }}</span>
          <span class="rail-os">{{ app.appInfo?.os }}/{{ app.appInfo?.arch }}</span>
        </div>
      </div>
    </aside>

    <main class="main">
      <RouterView v-slot="{ Component }">
        <Transition name="view" mode="out-in">
          <component :is="Component" />
        </Transition>
      </RouterView>
    </main>
  </div>
</template>

<style scoped>
.shell {
  display: grid;
  grid-template-columns: var(--nav-width) minmax(0, 1fr);
  height: 100vh;
  overflow: hidden;
}

.rail {
  position: relative;
  display: flex;
  flex-direction: column;
  gap: 24px;
  padding: 22px 14px 18px;
  border-right: 1px solid var(--border-soft);
  background:
    linear-gradient(180deg, rgba(255, 255, 255, 0.018), transparent 36%),
    rgba(9, 14, 20, 0.72);
  backdrop-filter: blur(12px);
  z-index: 2;
}

[data-theme='light'] .rail {
  background: linear-gradient(180deg, rgba(255, 255, 255, 0.7), transparent 36%), var(--bg-soft);
}

.brand {
  display: flex;
  align-items: center;
  gap: 11px;
  padding: 2px 8px 4px;
}

.brand-copy {
  display: flex;
  flex-direction: column;
  line-height: 1.1;
  letter-spacing: -0.01em;
}

.brand-copy strong {
  font-size: 15px;
  font-weight: 750;
}

.brand-copy span {
  margin-top: 3px;
  color: var(--text-3);
  font-size: 10.5px;
  letter-spacing: 0.14em;
}

.nav {
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.nav-item {
  position: relative;
  display: flex;
  align-items: center;
  gap: 11px;
  min-height: 40px;
  padding: 0 11px;
  border: 1px solid transparent;
  border-radius: var(--radius-sm);
  color: var(--text-3);
  font-size: 13px;
  font-weight: 550;
  transition:
    color var(--speed) var(--ease),
    background var(--speed) var(--ease),
    border-color var(--speed) var(--ease);
}

.nav-item:hover {
  color: var(--text);
  background: var(--surface-2);
  text-decoration: none;
}

.nav-item.active {
  color: var(--text);
  background: linear-gradient(90deg, rgba(86, 224, 192, 0.1), rgba(86, 224, 192, 0.02));
  border-color: rgba(86, 224, 192, 0.16);
}

.nav-item.active::before {
  content: '';
  position: absolute;
  left: -14px;
  top: 10px;
  bottom: 10px;
  width: 2px;
  border-radius: 2px;
  background: var(--accent);
  box-shadow: 0 0 12px rgba(86, 224, 192, 0.5);
}

.nav-label {
  flex: 1;
}

.nav-update-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--warn);
  box-shadow: 0 0 0 3px var(--warn-soft);
}

.nav-count {
  min-width: 20px;
  height: 18px;
  padding: 0 5px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 999px;
  background: var(--accent-soft);
  color: var(--accent);
  font-size: 10px;
}

.rail-spacer {
  flex: 1;
}

.rail-status {
  display: flex;
  flex-direction: column;
  gap: 9px;
  padding: 12px 9px 0;
  border-top: 1px solid var(--border-soft);
}

.status-link {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 7px 5px;
  border-radius: var(--radius-sm);
  color: var(--text-2);
  min-width: 0;
}

.status-link:hover {
  color: var(--text);
  background: var(--surface-2);
  text-decoration: none;
}

.status-icon {
  position: relative;
  display: grid;
  place-items: center;
  width: 30px;
  height: 30px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--surface-2);
  flex: none;
}

.status-icon .dot {
  position: absolute;
  right: -2px;
  top: -2px;
  width: 8px;
  height: 8px;
  border: 2px solid var(--bg);
  box-shadow: none;
  animation: none;
}

.status-copy {
  display: flex;
  flex-direction: column;
  min-width: 0;
  gap: 2px;
}

.status-kicker {
  font-size: 9.5px;
  letter-spacing: 0.12em;
  text-transform: uppercase;
  color: var(--text-4);
}

.status-value {
  font-size: 11.5px;
  color: var(--text-2);
}

.update-chip {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 7px 8px;
  border: 1px solid rgba(240, 180, 90, 0.22);
  border-radius: var(--radius-sm);
  background: var(--warn-soft);
  color: var(--warn);
  font-size: 11px;
  font-weight: 600;
}

.update-chip:hover {
  text-decoration: none;
  border-color: rgba(240, 180, 90, 0.4);
}

.rail-version {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 0 5px;
  color: var(--text-4);
  font-size: 10.5px;
}

.rail-os {
  margin-left: auto;
  font-size: 9.5px;
  color: var(--text-4);
}

.main {
  min-width: 0;
  overflow: auto;
  position: relative;
}

.main::before {
  content: '';
  position: fixed;
  inset: 0 0 0 var(--nav-width);
  pointer-events: none;
  background-image:
    linear-gradient(rgba(120, 160, 190, 0.028) 1px, transparent 1px),
    linear-gradient(90deg, rgba(120, 160, 190, 0.028) 1px, transparent 1px);
  background-size: 48px 48px;
  mask-image: linear-gradient(to bottom, black, transparent 70%);
}

.view-enter-active,
.view-leave-active {
  transition:
    opacity 150ms var(--ease),
    transform 150ms var(--ease);
}

.view-enter-from {
  opacity: 0;
  transform: translateY(4px);
}

.view-leave-to {
  opacity: 0;
  transform: translateY(-3px);
}

@media (max-width: 1120px) {
  .shell {
    grid-template-columns: 78px minmax(0, 1fr);
  }

  .rail {
    align-items: center;
    padding-left: 10px;
    padding-right: 10px;
  }

  .brand {
    padding: 2px 0 4px;
  }

  .brand-copy,
  .nav-label,
  .status-copy,
  .update-chip span:not(.mono),
  .rail-version span {
    display: none;
  }

  .nav-item {
    width: 42px;
    justify-content: center;
    padding: 0;
  }

  .nav-item.active::before {
    left: -10px;
  }

  .nav-count,
  .nav-update-dot {
    position: absolute;
    right: 2px;
    top: 2px;
    min-width: 14px;
    height: 14px;
    font-size: 8px;
  }

  .status-link {
    justify-content: center;
    padding: 5px 0;
  }

  .update-chip {
    width: 34px;
    justify-content: center;
    padding: 7px 0;
  }

  .rail-version {
    justify-content: center;
  }

  .main::before {
    inset: 0 0 0 78px;
  }

  .page {
    padding: 24px 22px 44px;
  }
}
</style>