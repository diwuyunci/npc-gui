<script setup lang="ts">
withDefaults(
  defineProps<{
    label?: string
    hint?: string
    error?: string
    required?: boolean
    htmlFor?: string
    optional?: boolean
  }>(),
  { label: '', hint: '', error: '', required: false, htmlFor: '', optional: false },
)
</script>

<template>
  <div class="field">
    <label v-if="label" class="label" :for="htmlFor || undefined">
      {{ label }}
      <span v-if="required" class="required">*</span>
      <span v-else-if="optional" class="optional">可选</span>
    </label>
    <slot />
    <p v-if="error" class="error-text" role="alert">{{ error }}</p>
    <p v-else-if="hint" class="hint">{{ hint }}</p>
  </div>
</template>

<style scoped>
.required {
  color: var(--error);
}
</style>
