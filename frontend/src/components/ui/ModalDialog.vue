<script setup lang="ts">
import { nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { X } from 'lucide-vue-next'

const props = withDefaults(
  defineProps<{
    open: boolean
    title: string
    subtitle?: string
    width?: string
    closeOnBackdrop?: boolean
  }>(),
  { width: '560px', subtitle: '', closeOnBackdrop: true },
)

const emit = defineEmits<{ close: [] }>()

const panel = ref<HTMLElement | null>(null)
const titleId = 'modal-title-' + Math.random().toString(36).slice(2, 8)
let previousFocus: HTMLElement | null = null

function focusables(): HTMLElement[] {
  if (!panel.value) return []
  return Array.from(
    panel.value.querySelectorAll<HTMLElement>(
      'button:not([disabled]), [href], input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])',
    ),
  ).filter((el) => el.offsetParent !== null || el === document.activeElement)
}

function onKeydown(event: KeyboardEvent): void {
  if (!props.open) return
  if (event.key === 'Escape') {
    event.preventDefault()
    emit('close')
    return
  }
  if (event.key !== 'Tab') return
  const nodes = focusables()
  if (!nodes.length) return
  const first = nodes[0]
  const last = nodes[nodes.length - 1]
  if (event.shiftKey && document.activeElement === first) {
    event.preventDefault()
    last.focus()
  } else if (!event.shiftKey && document.activeElement === last) {
    event.preventDefault()
    first.focus()
  }
}

watch(
  () => props.open,
  async (open) => {
    if (open) {
      previousFocus = document.activeElement as HTMLElement | null
      window.addEventListener('keydown', onKeydown)
      await nextTick()
      focusables()[0]?.focus()
    } else {
      window.removeEventListener('keydown', onKeydown)
      previousFocus?.focus?.()
      previousFocus = null
    }
  },
  { immediate: true },
)

onBeforeUnmount(() => {
  window.removeEventListener('keydown', onKeydown)
})

function backdrop(): void {
  if (props.closeOnBackdrop) emit('close')
}
</script>

<template>
  <Teleport to="body">
    <Transition name="modal">
      <div v-if="open" class="modal-backdrop" @mousedown.self="backdrop">
        <div
          ref="panel"
          class="modal-panel"
          role="dialog"
          aria-modal="true"
          :aria-labelledby="titleId"
          :style="{ width }"
        >
          <header class="modal-head">
            <div>
              <h2 :id="titleId">{{ title }}</h2>
              <p v-if="subtitle">{{ subtitle }}</p>
            </div>
            <button class="modal-close" type="button" aria-label="关闭" @click="emit('close')">
              <X :size="17" />
            </button>
          </header>
          <div class="modal-body">
            <slot />
          </div>
          <footer v-if="$slots.footer" class="modal-foot">
            <slot name="footer" />
          </footer>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.modal-backdrop {
  position: fixed;
  inset: 0;
  z-index: 100;
  display: grid;
  place-items: center;
  padding: 24px;
  background: rgba(3, 7, 11, 0.66);
  backdrop-filter: blur(7px);
}

.modal-panel {
  max-width: calc(100vw - 40px);
  max-height: calc(100vh - 48px);
  display: flex;
  flex-direction: column;
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-lg);
  background: var(--surface);
  box-shadow: var(--shadow-lg);
  overflow: hidden;
}

.modal-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 18px;
  padding: 18px 20px 14px;
  border-bottom: 1px solid var(--border-soft);
}

.modal-head h2 {
  font-size: 16px;
  font-weight: 650;
  letter-spacing: -0.01em;
}

.modal-head p {
  margin-top: 4px;
  color: var(--text-3);
  font-size: 12px;
  line-height: 1.5;
}

.modal-close {
  display: grid;
  place-items: center;
  width: 30px;
  height: 30px;
  flex: none;
  border: 1px solid transparent;
  border-radius: 7px;
  background: transparent;
  color: var(--text-3);
}

.modal-close:hover {
  background: var(--surface-2);
  color: var(--text);
}

.modal-body {
  padding: 18px 20px;
  overflow: auto;
}

.modal-foot {
  display: flex;
  justify-content: flex-end;
  gap: 9px;
  padding: 13px 20px 16px;
  border-top: 1px solid var(--border-soft);
  background: rgba(255, 255, 255, 0.01);
}

.modal-enter-active,
.modal-leave-active {
  transition: opacity 160ms var(--ease);
}

.modal-enter-active .modal-panel,
.modal-leave-active .modal-panel {
  transition: transform 180ms var(--ease);
}

.modal-enter-from,
.modal-leave-to {
  opacity: 0;
}

.modal-enter-from .modal-panel,
.modal-leave-to .modal-panel {
  transform: translateY(8px) scale(0.985);
}
</style>
