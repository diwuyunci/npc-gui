import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import type { RuntimeInfo } from '@/lib/types'

export const useRuntimesStore = defineStore('runtimes', () => {
  const map = ref<Record<string, RuntimeInfo>>({})

  const list = computed(() => Object.values(map.value))
  const runningCount = computed(
    () =>
      list.value.filter((item) =>
        ['starting', 'running', 'online', 'reconnecting', 'stopping'].includes(item.state),
      ).length,
  )
  const onlineCount = computed(() => list.value.filter((item) => item.state === 'online').length)
  const reconnects = computed(() => list.value.reduce((sum, item) => sum + (item.reconnects || 0), 0))

  function setAll(items: RuntimeInfo[]): void {
    const next: Record<string, RuntimeInfo> = {}
    for (const item of items) next[item.profileId] = item
    map.value = next
  }

  function upsert(item: RuntimeInfo): void {
    if (!item?.profileId) return
    map.value = { ...map.value, [item.profileId]: { ...map.value[item.profileId], ...item } }
  }

  function get(id: string): RuntimeInfo {
    return map.value[id] ?? { profileId: id, state: 'stopped', reconnects: 0, pendingRestart: false }
  }

  function stateOf(id: string): RuntimeInfo['state'] {
    return get(id).state
  }

  return { map, list, runningCount, onlineCount, reconnects, setAll, upsert, get, stateOf }
})
