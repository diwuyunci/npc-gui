<script setup lang="ts">
import { CheckCircle2, Info, TriangleAlert, X, XCircle } from 'lucide-vue-next'
import { useAppStore } from '@/stores/app'

const app = useAppStore()

const icons = {
  success: CheckCircle2,
  info: Info,
  warn: TriangleAlert,
  error: XCircle,
}
</script>

<template>
  <div class="toast-stack" aria-live="polite" aria-atomic="false">
    <TransitionGroup name="toast">
      <div v-for="toast in app.toasts" :key="toast.id" class="toast" :class="'toast-' + toast.level">
        <span class="toast-icon">
          <component :is="icons[toast.level]" :size="17" :stroke-width="1.9" />
        </span>
        <div class="toast-copy">
          <strong>{{ toast.message }}</strong>
          <p v-if="toast.detail">{{ toast.detail }}</p>
        </div>
        <button class="toast-close" type="button" aria-label="关闭提示" @click="app.dismissToast(toast.id)">
          <X :size="14" />
        </button>
      </div>
    </TransitionGroup>
  </div>
</template>

<style scoped>
.toast-stack {
  position: fixed;
  right: 22px;
  bottom: 22px;
  z-index: 120;
  display: flex;
  flex-direction: column;
  gap: 9px;
  width: min(390px, calc(100vw - 44px));
  pointer-events: none;
}

.toast {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 11px 12px;
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-md);
  background: rgba(17, 25, 34, 0.96);
  box-shadow: var(--shadow-lg);
  backdrop-filter: blur(16px);
  pointer-events: auto;
}

[data-theme='light'] .toast {
  background: rgba(255, 255, 255, 0.97);
}

.toast-icon {
  display: grid;
  place-items: center;
  width: 26px;
  height: 26px;
  border-radius: 7px;
  flex: none;
  margin-top: 1px;
}

.toast-success .toast-icon {
  color: var(--ok);
  background: var(--ok-soft);
}

.toast-info .toast-icon {
  color: var(--info);
  background: var(--info-soft);
}

.toast-warn .toast-icon {
  color: var(--warn);
  background: var(--warn-soft);
}

.toast-error .toast-icon {
  color: var(--error);
  background: var(--error-soft);
}

.toast-copy {
  flex: 1;
  min-width: 0;
  padding-top: 2px;
}

.toast-copy strong {
  display: block;
  font-size: 12.5px;
  font-weight: 600;
  line-height: 1.35;
}

.toast-copy p {
  margin-top: 2px;
  color: var(--text-3);
  font-size: 11.5px;
  line-height: 1.45;
  word-break: break-word;
}

.toast-close {
  display: grid;
  place-items: center;
  width: 24px;
  height: 24px;
  flex: none;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: var(--text-4);
}

.toast-close:hover {
  color: var(--text);
  background: var(--surface-hover);
}

.toast-enter-active,
.toast-leave-active {
  transition:
    opacity 180ms var(--ease),
    transform 180ms var(--ease);
}

.toast-enter-from,
.toast-leave-to {
  opacity: 0;
  transform: translateX(14px);
}
</style>
