<script setup lang="ts">
import { computed } from 'vue'
import { stateLabel } from '@/lib/format'
import type { RunState } from '@/lib/types'

const props = withDefaults(
  defineProps<{
    state?: RunState
    showDot?: boolean
    compact?: boolean
  }>(),
  { state: 'stopped', showDot: true, compact: false },
)

const tone = computed(() => {
  switch (props.state) {
    case 'online':
      return 'ok'
    case 'starting':
    case 'running':
    case 'reconnecting':
      return 'warn'
    case 'error':
      return 'error'
    case 'stopping':
      return 'info'
    default:
      return 'muted'
  }
})

const dotClass = computed(() => {
  switch (props.state) {
    case 'online':
      return 'dot-online'
    case 'starting':
    case 'running':
    case 'reconnecting':
      return 'dot-warn'
    case 'error':
      return 'dot-error'
    case 'stopping':
      return 'dot-info'
    default:
      return ''
  }
})
</script>

<template>
  <span class="status" :class="['status-' + tone, { compact }]">
    <span v-if="showDot" class="dot" :class="dotClass" />
    <span>{{ stateLabel(state) }}</span>
  </span>
</template>

<style scoped>
.status {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 22px;
  padding: 0 8px;
  border: 1px solid var(--border);
  border-radius: 999px;
  background: var(--surface-2);
  color: var(--text-2);
  font-size: 11px;
  font-weight: 600;
  white-space: nowrap;
}

.status.compact {
  height: 20px;
  padding: 0 7px;
  font-size: 10.5px;
}

.status-ok {
  color: var(--ok);
  border-color: rgba(70, 213, 151, 0.22);
  background: var(--ok-soft);
}

.status-warn {
  color: var(--warn);
  border-color: rgba(240, 180, 90, 0.22);
  background: var(--warn-soft);
}

.status-error {
  color: var(--error);
  border-color: rgba(255, 125, 136, 0.22);
  background: var(--error-soft);
}

.status-info {
  color: var(--info);
  border-color: rgba(111, 184, 255, 0.22);
  background: var(--info-soft);
}

.status-muted {
  color: var(--text-3);
}

.status :deep(.dot) {
  width: 6px;
  height: 6px;
}
</style>
