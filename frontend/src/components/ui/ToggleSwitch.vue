<script setup lang="ts">
const props = withDefaults(
  defineProps<{
    modelValue: boolean
    disabled?: boolean
    label?: string
  }>(),
  { disabled: false, label: '' },
)

const emit = defineEmits<{ 'update:modelValue': [value: boolean] }>()

function toggle(): void {
  if (props.disabled) return
  emit('update:modelValue', !props.modelValue)
}

function onKey(event: KeyboardEvent): void {
  if (event.key === ' ' || event.key === 'Enter') {
    event.preventDefault()
    toggle()
  }
}
</script>

<template>
  <button
    type="button"
    role="switch"
    class="switch"
    :class="{ on: modelValue, disabled }"
    :aria-checked="modelValue"
    :aria-label="label || undefined"
    :disabled="disabled"
    @click="toggle"
    @keydown="onKey"
  >
    <span class="switch-thumb" />
  </button>
</template>

<style scoped>
.switch {
  position: relative;
  width: 36px;
  height: 21px;
  flex: none;
  padding: 0;
  border: 1px solid var(--border-strong);
  border-radius: 999px;
  background: var(--surface-3);
  transition:
    background var(--speed) var(--ease),
    border-color var(--speed) var(--ease);
}

.switch-thumb {
  position: absolute;
  top: 2px;
  left: 2px;
  width: 15px;
  height: 15px;
  border-radius: 50%;
  background: var(--text-3);
  transition:
    transform var(--speed) var(--ease),
    background var(--speed) var(--ease);
}

.switch.on {
  border-color: rgba(86, 224, 192, 0.4);
  background: var(--accent-soft);
}

.switch.on .switch-thumb {
  transform: translateX(15px);
  background: var(--accent);
}

.switch.disabled {
  opacity: 0.45;
  cursor: not-allowed;
}
</style>
