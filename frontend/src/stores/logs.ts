import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import type { LogEntry, LogPage } from '@/lib/types'

const MAX_LINES = 5000

export const useLogsStore = defineStore('logs', () => {
  const byProfile = ref<Record<string, LogEntry[]>>({})
  const seen = new Map<string, Set<number>>()

  const all = computed(() => {
    const merged: LogEntry[] = []
    for (const entries of Object.values(byProfile.value)) merged.push(...entries)
    merged.sort((a, b) => (a.time === b.time ? a.seq - b.seq : a.time - b.time))
    return merged
  })

  const alerts = computed(() =>
    all.value
      .filter((entry) => ['warn', 'error', 'fatal', 'panic'].includes(entry.level))
      .slice(-40),
  )

  function ensureSeen(id: string): Set<number> {
    let set = seen.get(id)
    if (!set) {
      set = new Set<number>()
      seen.set(id, set)
    }
    return set
  }

  function merge(profileId: string, entries: LogEntry[]): void {
    if (!profileId || !entries?.length) return
    const set = ensureSeen(profileId)
    const current = byProfile.value[profileId] ?? []
    const additions: LogEntry[] = []
    for (const entry of entries) {
      if (set.has(entry.seq)) continue
      set.add(entry.seq)
      additions.push({ ...entry, profileId: entry.profileId || profileId })
    }
    if (!additions.length) return
    let next = current.concat(additions)
    if (next.length > MAX_LINES) {
      const removed = next.slice(0, next.length - MAX_LINES)
      for (const entry of removed) set.delete(entry.seq)
      next = next.slice(-MAX_LINES)
    }
    byProfile.value = { ...byProfile.value, [profileId]: next }
  }

  function addPage(page: LogPage): void {
    merge(page.profileId || page.entries[0]?.profileId || '', page.entries)
  }

  function hydrate(entries: LogEntry[]): void {
    if (!entries?.length) return
    const grouped = new Map<string, LogEntry[]>()
    for (const entry of entries) {
      const id = entry.profileId || ''
      if (!id) continue
      const list = grouped.get(id) ?? []
      list.push(entry)
      grouped.set(id, list)
    }
    for (const [id, list] of grouped) merge(id, list)
  }

  function setForProfile(profileId: string, entries: LogEntry[]): void {
    seen.delete(profileId)
    byProfile.value = { ...byProfile.value, [profileId]: [] }
    merge(profileId, entries)
  }

  function get(profileId: string): LogEntry[] {
    return byProfile.value[profileId] ?? []
  }

  function clear(profileId: string): void {
    seen.delete(profileId)
    byProfile.value = { ...byProfile.value, [profileId]: [] }
  }

  function lastSeq(profileId: string): number {
    const list = byProfile.value[profileId] ?? []
    return list.length ? list[list.length - 1].seq : 0
  }

  return { byProfile, all, alerts, addPage, hydrate, setForProfile, get, clear, lastSeq }
})
