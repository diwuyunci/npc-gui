<script setup lang="ts">
import { onBeforeUnmount, ref } from 'vue'
import { CircleAlert, ScanLine, TriangleAlert, WandSparkles } from 'lucide-vue-next'
import type { Profile } from '@/lib/types'
import { backend, errorMessage } from '@/lib/backend'

const props = withDefaults(
  defineProps<{
    placeholder?: string
  }>(),
  { placeholder: './npc -server="host:port" -vkey="vkey" -type="tcp"' },
)

const emit = defineEmits<{ parsed: [Profile] }>()

const command = ref('')
const loading = ref(false)
const error = ref('')
const warning = ref('')
let timer: number | undefined

async function parse(): Promise<void> {
  const raw = command.value.trim()
  if (!raw) {
    error.value = '请先粘贴 NPS 启动命令'
    return
  }
  loading.value = true
  error.value = ''
  warning.value = ''
  try {
    const preview = await backend.importCommand(raw)
    if (preview.errors.length) {
      error.value = preview.errors[0]
      return
    }
    if (!preview.candidates.length) {
      error.value = '没有从命令中识别到连接'
      return
    }
    emit('parsed', preview.candidates[0].profile)
    if (preview.candidates.length > 1) {
      warning.value =
        '检测到 ' + preview.candidates.length + ' 个服务端，已填充第一个；批量导入请使用「导入命令」。'
    } else if (preview.warnings.length) {
      warning.value = preview.warnings[0]
    }
  } catch (err) {
    error.value = errorMessage(err)
  } finally {
    loading.value = false
  }
}

function clearFeedback(): void {
  error.value = ''
  warning.value = ''
}

function onPaste(): void {
  clearFeedback()
  window.clearTimeout(timer)
  timer = window.setTimeout(() => {
    void parse()
  }, 30)
}

onBeforeUnmount(() => window.clearTimeout(timer))
</script>

<template>
  <div class="command-fill">
    <div class="command-fill-head">
      <span class="fill-icon"><WandSparkles :size="15" /></span>
      <div class="fill-title">
        <strong>从 NPS 启动命令快速填充</strong>
        <small>粘贴后自动解析 server、vkey、连接方式等基础配置</small>
      </div>
      <span class="fill-badge">可选</span>
    </div>

    <div class="command-fill-row">
      <input
        v-model="command"
        class="input mono"
        type="text"
        spellcheck="false"
        :placeholder="props.placeholder"
        aria-label="NPS 启动命令"
        @input="clearFeedback"
        @paste="onPaste"
        @keydown.enter.prevent="parse"
      />
      <button class="btn btn-secondary btn-sm" type="button" :disabled="loading" @click="parse">
        <span v-if="loading" class="spinner" />
        <ScanLine v-else :size="13" />
        解析填充
      </button>
    </div>

    <p v-if="error" class="fill-error"><CircleAlert :size="13" /> {{ error }}</p>
    <p v-else-if="warning" class="fill-warning"><TriangleAlert :size="13" /> {{ warning }}</p>
    <p v-else class="fill-hint">
      支持 ./npc、npc.exe、install/start 前缀和单双引号；命令只会被解析，不会执行。
    </p>
  </div>
</template>

<style scoped>
.command-fill {
  display: flex;
  flex-direction: column;
  gap: 9px;
  padding: 13px;
  border: 1px solid rgba(86, 224, 192, 0.18);
  border-radius: var(--radius-md);
  background: linear-gradient(120deg, var(--accent-soft), transparent 60%);
}

.command-fill-head {
  display: flex;
  align-items: center;
  gap: 9px;
}

.fill-icon {
  display: grid;
  place-items: center;
  width: 28px;
  height: 28px;
  flex: none;
  border: 1px solid rgba(86, 224, 192, 0.2);
  border-radius: 7px;
  background: rgba(86, 224, 192, 0.08);
  color: var(--accent);
}

.fill-title {
  display: flex;
  flex-direction: column;
  gap: 1px;
  flex: 1;
  min-width: 0;
}

.fill-title strong {
  font-size: 11.5px;
  font-weight: 650;
}

.fill-title small {
  color: var(--text-3);
  font-size: 10px;
}

.fill-badge {
  flex: none;
  padding: 1px 6px;
  border: 1px solid var(--border);
  border-radius: 999px;
  color: var(--text-4);
  font-size: 9px;
}

.command-fill-row {
  display: flex;
  gap: 6px;
  align-items: center;
}

.command-fill-row .input {
  min-height: 34px;
  font-size: 11.5px;
}

.fill-error,
.fill-warning,
.fill-hint {
  display: flex;
  align-items: flex-start;
  gap: 5px;
  margin: 0;
  font-size: 10.5px;
  line-height: 1.45;
}

.fill-error {
  color: var(--error);
}

.fill-warning {
  color: var(--warn);
}

.fill-hint {
  color: var(--text-3);
}

@media (max-width: 640px) {
  .command-fill-row {
    flex-direction: column;
    align-items: stretch;
  }
}
</style>