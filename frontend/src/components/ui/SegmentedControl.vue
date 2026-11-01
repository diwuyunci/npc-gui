<script setup lang="ts">
defineProps<{
  modelValue: string
  options: { value: string; label: string }[]
  ariaLabel?: string
}>()

const emit = defineEmits<{ 'update:modelValue': [value: string] }>()
</script>

<template>
  <div class="segmented" role="group" :aria-label="ariaLabel">
    <button
      v-for="option in options"
      :key="option.value"
      type="button"
      class="segmented-item"
      :class="{ active: modelValue === option.value }"
      :aria-pressed="modelValue === option.value"
      @click="emit('update:modelValue', option.value)"
    >
      {{ option.label }}
    </button>
  </div>
</template>

<style scoped>
.segmented {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  padding: 3px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--bg-soft);
}

.segmented-item {
  min-height: 27px;
  padding: 0 10px;
  border: 1px solid transparent;
  border-radius: 5px;
  background: transparent;
  color: var(--text-3);
  font-size: 11.5px;
  font-weight: 550;
  transition:
    color var(--speed) var(--ease),
    background var(--speed) var(--ease);
}

.segmented-item:hover {
  color: var(--text);
}

.segmented-item.active {
  color: var(--text);
  background: var(--surface-3);
  border-color: var(--border);
  box-shadow: var(--shadow-sm);
}
</style>
