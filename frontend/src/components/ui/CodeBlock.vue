<script setup lang="ts">
import { ref } from 'vue'
import { Check, Copy } from 'lucide-vue-next'

const props = withDefaults(
  defineProps<{
    code: string
    title?: string
    copyable?: boolean
    maxHeight?: string
  }>(),
  { title: '', copyable: true, maxHeight: '440px' },
)

const copied = ref(false)
let timer: number | undefined

async function copy(): Promise<void> {
  try {
    if (navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(props.code)
    } else {
      const el = document.createElement('textarea')
      el.value = props.code
      document.body.appendChild(el)
      el.select()
      document.execCommand('copy')
      document.body.removeChild(el)
    }
    copied.value = true
    window.clearTimeout(timer)
    timer = window.setTimeout(() => (copied.value = false), 1600)
  } catch {
    copied.value = false
  }
}
</script>

<template>
  <div class="code-block">
    <div v-if="title || copyable" class="code-head">
      <span>{{ title || 'config' }}</span>
      <button
        v-if="copyable"
        class="code-copy"
        type="button"
        :aria-label="copied ? '已复制' : '复制'"
        @click="copy"
      >
        <Check v-if="copied" :size="13" />
        <Copy v-else :size="13" />
        <span>{{ copied ? '已复制' : '复制' }}</span>
      </button>
    </div>
    <pre class="code-body" :style="{ maxHeight }">{{ code }}</pre>
  </div>
</template>

<style scoped>
.code-copy {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 3px 7px;
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 5px;
  background: transparent;
  color: #9fb2c6;
  font-size: 10.5px;
}

.code-copy:hover {
  color: #d6e4f2;
  border-color: rgba(255, 255, 255, 0.16);
}
</style>
