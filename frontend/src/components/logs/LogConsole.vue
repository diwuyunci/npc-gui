<script setup lang="ts">
import { nextTick, onMounted, ref, watch } from 'vue'
import { Terminal } from 'lucide-vue-next'
import type { LogEntry } from '@/lib/types'
import { formatTime, levelLabel } from '@/lib/format'

const props = withDefaults(
  defineProps<{
    entries: LogEntry[]
    wrap?: boolean
    follow?: boolean
    showProfile?: boolean
    profileName?: (id?: string) => string
    emptyText?: string
  }>(),
  {
    wrap: false,
    follow: true,
    showProfile: false,
    profileName: () => '',
    emptyText: '还没有日志。启动连接后，npc 输出会实时出现在这里。',
  },
)

const emit = defineEmits<{ 'update:follow': [value: boolean] }>()
const viewport = ref<HTMLElement | null>(null)

function scrollToBottom(): void {
  const el = viewport.value
  if (!el) return
  el.scrollTop = el.scrollHeight
}

function nearBottom(): boolean {
  const el = viewport.value
  if (!el) return true
  return el.scrollHeight - el.scrollTop - el.clientHeight < 40
}

function onScroll(): void {
  if (!props.follow) {
    if (nearBottom()) emit('update:follow', true)
    return
  }
  if (!nearBottom()) emit('update:follow', false)
}

watch(
  () => props.entries,
  async () => {
    if (!props.follow) return
    await nextTick()
    scrollToBottom()
  },
  { deep: false },
)

watch(
  () => props.follow,
  async (follow) => {
    if (follow) {
      await nextTick()
      scrollToBottom()
    }
  },
)

onMounted(() => {
  if (props.follow) scrollToBottom()
})
</script>

<template>
  <div ref="viewport" class="console" :class="{ wrap }" @scroll="onScroll">
    <div v-if="!entries.length" class="console-empty">
      <Terminal :size="24" :stroke-width="1.5" />
      <p>{{ emptyText }}</p>
    </div>
    <div v-else class="console-lines">
      <div v-for="entry in entries" :key="(entry.profileId || '') + ':' + entry.seq" class="log-line" :class="'log-' + entry.level">
        <span class="log-time mono">{{ formatTime(entry.time) }}</span>
        <span class="log-level mono">{{ levelLabel(entry.level) }}</span>
        <span v-if="showProfile" class="log-profile mono">{{ profileName(entry.profileId) }}</span>
        <span class="log-text">{{ entry.text }}</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.console {
  height: 100%;
  min-height: 0;
  overflow: auto;
  padding: 10px 0;
  background:
    linear-gradient(180deg, rgba(86, 224, 192, 0.018), transparent 30%),
    #070b10;
  font-family: var(--font-mono);
  font-size: 11.5px;
  line-height: 1.7;
}

.console-lines {
  min-width: max-content;
  padding: 2px 12px;
}

.console.wrap .console-lines {
  min-width: 0;
}

.log-line {
  display: flex;
  align-items: flex-start;
  gap: 9px;
  padding: 1px 4px;
  border-radius: 3px;
  white-space: nowrap;
}

.console.wrap .log-line {
  white-space: pre-wrap;
  word-break: break-word;
}

.log-line:hover {
  background: rgba(255, 255, 255, 0.025);
}

.log-time {
  flex: none;
  color: #4d6478;
  font-size: 10px;
  padding-top: 1px;
}

.log-level {
  flex: none;
  width: 26px;
  color: #677f94;
  font-size: 9.5px;
  padding-top: 1px;
}

.log-debug .log-level,
.log-debug .log-text {
  color: #6f879b;
}

.log-info .log-level {
  color: #6fb8ff;
}

.log-info .log-text {
  color: #b7c9d9;
}

.log-warn .log-level,
.log-warn .log-text {
  color: #f0b45a;
}

.log-error .log-level,
.log-error .log-text {
  color: #ff7d88;
}

.log-fatal .log-level,
.log-fatal .log-text,
.log-panic .log-level,
.log-panic .log-text {
  color: #ff5c6d;
  font-weight: 600;
}

.log-profile {
  flex: none;
  max-width: 120px;
  overflow: hidden;
  text-overflow: ellipsis;
  color: #7290a8;
  font-size: 10px;
}

.log-text {
  color: #b7c9d9;
  min-width: 0;
}

.console-empty {
  height: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  color: #4f667a;
  font-family: var(--font-sans);
  font-size: 12px;
  text-align: center;
  padding: 30px;
}

.console-empty p {
  max-width: 420px;
  line-height: 1.6;
}
</style>
