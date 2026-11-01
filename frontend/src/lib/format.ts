import type { LogLevel, RunState } from './types'

export function pad(n: number): string {
  return n < 10 ? '0' + n : String(n)
}

export function formatTime(ts?: number): string {
  if (!ts) return '--'
  const d = new Date(ts)
  return pad(d.getHours()) + ':' + pad(d.getMinutes()) + ':' + pad(d.getSeconds())
}

export function formatDateTime(ts?: number): string {
  if (!ts) return '--'
  const d = new Date(ts)
  return (
    d.getFullYear() +
    '-' +
    pad(d.getMonth() + 1) +
    '-' +
    pad(d.getDate()) +
    ' ' +
    pad(d.getHours()) +
    ':' +
    pad(d.getMinutes())
  )
}

export function formatDuration(from?: number, now = Date.now()): string {
  if (!from) return '--'
  const diff = Math.max(0, Math.floor((now - from) / 1000))
  const h = Math.floor(diff / 3600)
  const m = Math.floor((diff % 3600) / 60)
  const s = diff % 60
  if (h > 0) return String(h).padStart(2, '0') + ':' + String(m).padStart(2, '0') + ':' + String(s).padStart(2, '0')
  return String(m).padStart(2, '0') + ':' + String(s).padStart(2, '0')
}

export function relativeTime(ts?: number, now = Date.now()): string {
  if (!ts) return '刚刚'
  const diff = Math.max(0, Math.floor((now - ts) / 1000))
  if (diff < 10) return '刚刚'
  if (diff < 60) return diff + ' 秒前'
  if (diff < 3600) return Math.floor(diff / 60) + ' 分钟前'
  if (diff < 86400) return Math.floor(diff / 3600) + ' 小时前'
  return Math.floor(diff / 86400) + ' 天前'
}

export function formatBytes(bytes?: number): string {
  if (!bytes || bytes <= 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB']
  let value = bytes
  let i = 0
  while (value >= 1024 && i < units.length - 1) {
    value /= 1024
    i++
  }
  return value.toFixed(value >= 10 || i === 0 ? 0 : 1) + ' ' + units[i]
}

export function formatPercent(value?: number): string {
  if (!value) return '0%'
  return value.toFixed(value >= 10 ? 0 : 1) + '%'
}

export function hostOnly(server?: string): string {
  if (!server) return '未设置'
  let s = server
  if (s.includes('@')) s = s.split('@')[0]
  if (s.includes('/')) s = s.split('/')[0]
  const idx = s.lastIndexOf(':')
  if (idx > 0 && /^[0-9]+$/.test(s.slice(idx + 1))) s = s.slice(0, idx)
  return s || server
}

export function stateLabel(state?: RunState): string {
  switch (state) {
    case 'starting':
      return '连接中'
    case 'running':
      return '运行中'
    case 'online':
      return '在线'
    case 'reconnecting':
      return '重连中'
    case 'stopping':
      return '停止中'
    case 'error':
      return '异常'
    default:
      return '已停止'
  }
}

export function levelLabel(level: string): string {
  switch (level) {
    case 'trace':
      return 'TRC'
    case 'debug':
      return 'DBG'
    case 'info':
      return 'INF'
    case 'warn':
      return 'WRN'
    case 'error':
      return 'ERR'
    case 'fatal':
      return 'FTL'
    case 'panic':
      return 'PNC'
    default:
      return level.slice(0, 3).toUpperCase()
  }
}

export function levelRank(level: string): number {
  switch (level) {
    case 'trace':
      return 0
    case 'debug':
      return 1
    case 'info':
      return 2
    case 'warn':
      return 3
    case 'error':
      return 4
    case 'fatal':
      return 5
    case 'panic':
      return 6
    default:
      return 2
  }
}

export function isAlertLevel(level: string): boolean {
  return levelRank(level) >= 3
}

export function connTypeLabel(type?: string): string {
  if (!type) return 'TCP'
  return type.toUpperCase()
}

export function onlineState(state?: RunState): boolean {
  return state === 'online'
}

export function activeState(state?: RunState): boolean {
  return state === 'starting' || state === 'running' || state === 'online' || state === 'reconnecting' || state === 'stopping'
}

export function levelList(): LogLevel[] {
  return ['debug', 'info', 'warn', 'error']
}
