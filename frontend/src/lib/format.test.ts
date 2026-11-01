import { describe, expect, it } from 'vitest'
import {
  formatBytes,
  formatDuration,
  formatPercent,
  hostOnly,
  levelLabel,
  levelRank,
  stateLabel,
} from './format'

describe('format helpers', () => {
  it('formats durations', () => {
    expect(formatDuration(0)).toBe('--')
    expect(formatDuration(1000, 1000 + 65_000)).toBe('01:05')
    expect(formatDuration(1000, 1000 + 3_725_000)).toBe('01:02:05')
  })

  it('formats bytes and percentages', () => {
    expect(formatBytes(0)).toBe('0 B')
    expect(formatBytes(1536)).toBe('1.5 KB')
    expect(formatBytes(18_700_000)).toMatch(/MB/)
    expect(formatPercent(47.2)).toBe('47%')
  })

  it('extracts hosts', () => {
    expect(hostOnly('nps.example.com:8024')).toBe('nps.example.com')
    expect(hostOnly('nps.example.com:8024@sni.example.com')).toBe('nps.example.com')
  })

  it('maps levels and states', () => {
    expect(levelLabel('warn')).toBe('WRN')
    expect(levelRank('error')).toBeGreaterThan(levelRank('info'))
    expect(stateLabel('online')).toBe('在线')
    expect(stateLabel('reconnecting')).toBe('重连中')
  })
})
