import { beforeEach, describe, expect, it } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import type { LogEntry } from '@/lib/types'
import { useLogsStore } from './logs'

function entry(profileId: string, seq: number, text = 'line'): LogEntry {
  return { profileId, seq, time: 1000 + seq, level: 'info', text }
}

describe('logs store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('merges pages and deduplicates by sequence', () => {
    const logs = useLogsStore()
    logs.addPage({ profileId: 'p1', entries: [entry('p1', 1), entry('p1', 2)], lastSeq: 2, truncated: false })
    logs.addPage({ profileId: 'p1', entries: [entry('p1', 2), entry('p1', 3)], lastSeq: 3, truncated: false })
    expect(logs.get('p1').map((item) => item.seq)).toEqual([1, 2, 3])
    expect(logs.all.length).toBe(3)
  })

  it('keeps profiles isolated and sorts all entries', () => {
    const logs = useLogsStore()
    logs.hydrate([entry('p2', 1), entry('p1', 1)])
    expect(logs.get('p1')).toHaveLength(1)
    expect(logs.get('p2')).toHaveLength(1)
    expect(logs.all.map((item) => item.time)).toEqual([1001, 1001])
  })

  it('clears one profile without touching another', () => {
    const logs = useLogsStore()
    logs.hydrate([entry('p1', 1), entry('p2', 1)])
    logs.clear('p1')
    expect(logs.get('p1')).toHaveLength(0)
    expect(logs.get('p2')).toHaveLength(1)
  })

  it('tracks alert levels', () => {
    const logs = useLogsStore()
    logs.hydrate([
      { ...entry('p1', 1), level: 'info' },
      { ...entry('p1', 2), level: 'error', text: 'boom' },
    ])
    expect(logs.alerts).toHaveLength(1)
    expect(logs.alerts[0].text).toBe('boom')
  })
})
