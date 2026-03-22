import { TimeRangeOption, TimeRangeInput } from '@/types'

const DURATIONS: Record<TimeRangeOption, number> = {
  '1h': 60 * 60 * 1000,
  '6h': 6 * 60 * 60 * 1000,
  '24h': 24 * 60 * 60 * 1000,
  '7d': 7 * 24 * 60 * 60 * 1000,
}

export function toTimeRange(range: TimeRangeOption): TimeRangeInput {
  const end = new Date()
  const start = new Date(end.getTime() - DURATIONS[range])
  return {
    start: start.toISOString(),
    end: end.toISOString(),
  }
}

export function formatTime(isoString: string): string {
  const date = new Date(isoString)
  return date.toLocaleString('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
}