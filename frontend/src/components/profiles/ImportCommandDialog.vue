<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { ScanLine, TriangleAlert, CircleAlert, CheckSquare, Square } from 'lucide-vue-next'
import type { ImportPreview } from '@/lib/types'
import { backend, errorMessage } from '@/lib/backend'
import { useProfilesStore } from '@/stores/profiles'
import { useAppStore } from '@/stores/app'
import ModalDialog from '@/components/ui/ModalDialog.vue'

const props = defineProps<{ open: boolean }>()
const emit = defineEmits<{ close: []; imported: [] }>()

const app = useAppStore()
const profiles = useProfilesStore()
const raw = ref('')
const preview = ref<ImportPreview | null>(null)
const parsing = ref(false)
const committing = ref(false)
const selected = ref<boolean[]>([])

const chosen = computed(() => {
  if (!preview.value) return []
  return preview.value.candidates.filter((_, index) => selected.value[index])
})

watch(
  () => props.open,
  (open) => {
    if (!open) return
    raw.value = ''
    preview.value = null
    selected.value = []
    parsing.value = false
    committing.value = false
  },
)

async function parse(): Promise<void> {
  if (!raw.value.trim()) {
    app.toast('warn', '请先粘贴 NPS 启动命令')
    return
  }
  parsing.value = true
  try {
    const result = await backend.importCommand(raw.value)
    preview.value = result
    selected.value = result.candidates.map((candidate) => candidate.selected)
    if (result.errors.length) app.toast('error', '命令解析失败', result.errors[0])
  } catch (err) {
    app.toast('error', '命令解析失败', errorMessage(err))
  } finally {
    parsing.value = false
  }
}

function toggle(index: number): void {
  selected.value = selected.value.map((value, i) => (i === index ? !value : value))
}

async function commit(): Promise<void> {
  if (!chosen.value.length) {
    app.toast('warn', '请至少选择一个连接')
    return
  }
  committing.value = true
  try {
    const result = await profiles.commitImport(chosen.value.map((item) => item.profile))
    if (result.result.ok) {
      app.toast('success', '已导入 ' + result.profiles.length + ' 个连接', result.result.warnings?.join('；'))
      emit('imported')
      emit('close')
    } else {
      app.toast('error', '导入失败', result.result.message)
    }
  } catch (err) {
    app.toast('error', '导入失败', errorMessage(err))
  } finally {
    committing.value = false
  }
}
</script>

<template>
  <ModalDialog
    :open="open"
    title="从 NPS 启动命令导入"
    subtitle="在 NPS 管理台的客户端页面复制启动命令，粘贴到这里。命令只会被解析，不会执行。"
    width="740px"
    @close="emit('close')"
  >
    <div class="import-dialog">
      <div class="field">
        <label class="label" for="import-command">启动命令</label>
        <textarea
          id="import-command"
          v-model="raw"
          class="textarea mono"
          rows="4"
          spellcheck="false"
          placeholder="npc -server=xxx:123,yyy:456 -vkey=key1,key2 -type=tls,tcp -log=off"
        />
        <p class="hint">支持多服务端、单双引号和 install/start 等前缀；vkey 不会显示在后续日志中。</p>
      </div>

      <div class="import-actions">
        <button class="btn btn-secondary" type="button" :disabled="parsing" @click="parse">
          <span v-if="parsing" class="spinner" />
          <ScanLine v-else :size="14" />
          解析命令
        </button>
        <span v-if="preview" class="subtle small">
          识别到 {{ preview.candidates.length }} 个连接
        </span>
      </div>

      <div v-if="preview?.errors.length" class="notice notice-error">
        <CircleAlert :size="15" />
        <div>
          <p v-for="error in preview.errors" :key="error">{{ error }}</p>
        </div>
      </div>

      <div v-if="preview?.warnings.length" class="notice notice-warn">
        <TriangleAlert :size="15" />
        <div>
          <p v-for="warning in preview.warnings" :key="warning">{{ warning }}</p>
        </div>
      </div>

      <div v-if="preview?.candidates.length" class="candidate-list">
        <button
          v-for="(candidate, index) in preview.candidates"
          :key="index"
          class="candidate"
          type="button"
          :class="{ muted: !selected[index] }"
          @click="toggle(index)"
        >
          <span class="check">
            <CheckSquare v-if="selected[index]" :size="16" />
            <Square v-else :size="16" />
          </span>
          <span class="candidate-main">
            <strong>{{ candidate.profile.name }}</strong>
            <span class="mono small">{{ candidate.profile.server }}</span>
          </span>
          <span class="badge">{{ candidate.profile.connType.toUpperCase() }}</span>
        </button>
      </div>

      <div v-if="preview && !preview.candidates.length && !preview.errors.length" class="empty">
        <p>没有识别到连接，请检查命令格式。</p>
      </div>
    </div>

    <template #footer>
      <button class="btn btn-ghost" type="button" @click="emit('close')">取消</button>
      <button
        class="btn btn-primary"
        type="button"
        :disabled="!chosen.length || committing"
        @click="commit"
      >
        <span v-if="committing" class="spinner" />
        导入 {{ chosen.length }} 个连接
      </button>
    </template>
  </ModalDialog>
</template>

<style scoped>
.import-dialog {
  display: flex;
  flex-direction: column;
  gap: 15px;
}

.import-actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.notice {
  display: flex;
  gap: 9px;
  padding: 10px 11px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  font-size: 12px;
  line-height: 1.5;
}

.notice p {
  margin: 0;
}

.notice-error {
  color: var(--error);
  border-color: rgba(255, 125, 136, 0.22);
  background: var(--error-soft);
}

.notice-warn {
  color: var(--warn);
  border-color: rgba(240, 180, 90, 0.22);
  background: var(--warn-soft);
}

.candidate-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
  max-height: 280px;
  overflow: auto;
}

.candidate {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  padding: 10px 11px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--bg-soft);
  color: var(--text);
  text-align: left;
  transition:
    border-color var(--speed) var(--ease),
    opacity var(--speed) var(--ease);
}

.candidate:hover {
  border-color: var(--border-strong);
}

.candidate.muted {
  opacity: 0.5;
}

.check {
  display: grid;
  place-items: center;
  color: var(--accent);
  flex: none;
}

.candidate-main {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
  flex: 1;
}

.candidate-main strong {
  font-size: 12.5px;
}

.candidate-main .mono {
  color: var(--text-3);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
