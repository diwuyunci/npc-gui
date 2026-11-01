<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(
  defineProps<{
    value?: number
    indeterminate?: boolean
    tone?: 'accent' | 'warn' | 'error' | 'info'
    height?: number
  }>(),
  { value: 0, indeterminate: false, tone: 'accent', height: 5 },
)

const width = computed(() => Math.max(0, Math.min(100, props.value || 0)))
</script>

<template>
  <div
    class="progress"
    :class="['tone-' + tone, { indeterminate }]"
    :style="{ height: height + 'px' }"
    role="progressbar"
    :aria-valuenow="indeterminate ? undefined : Math.round(width)"
    aria-valuemin="0"
    aria-valuemax="100"
  >
    <span :style="indeterminate ? undefined : { width: width + '%' }" />
  </div>
</template>

<style scoped>
.progress {
  width: 100%;
  overflow: hidden;
  border-radius: 999px;
  background: var(--surface-3);
}

.progress span {
  display: block;
  height: 100%;
  border-radius: inherit;
  background: var(--accent);
  transition: width 160ms var(--ease);
}

.tone-accent span {
  background: linear-gradient(90deg, var(--accent-strong), var(--accent));
}

.tone-warn span {
  background: var(--warn);
}

.tone-error span {
  background: var(--error);
}

.tone-info span {
  background: var(--info);
}

.progress.indeterminate span {
  width: 38%;
  background: linear-gradient(90deg, transparent, var(--accent), transparent);
  animation: indeterminate 1.15s var(--ease) infinite;
}

@keyframes indeterminate {
  from {
    transform: translateX(-110%);
  }
  to {
    transform: translateX(330%);
  }
}
</style>
