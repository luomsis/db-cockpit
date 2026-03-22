'use client'

import { TimeRangeOption } from '@/types'

interface HeaderProps {
  timeRange: TimeRangeOption
  onTimeRangeChange: (range: TimeRangeOption) => void
}

const timeRangeOptions: { value: TimeRangeOption; label: string }[] = [
  { value: '1h', label: '最近1小时' },
  { value: '6h', label: '最近6小时' },
  { value: '24h', label: '最近24小时' },
  { value: '7d', label: '最近7天' },
]

export function Header({ timeRange, onTimeRangeChange }: HeaderProps) {
  return (
    <header className="flex items-center justify-between border-b border-border px-6 py-4">
      <h1 className="text-xl font-semibold">监控告警大盘</h1>
      <div className="flex items-center gap-2">
        <span className="text-sm text-muted-foreground">时间范围:</span>
        <select
          value={timeRange}
          onChange={(e) => onTimeRangeChange(e.target.value as TimeRangeOption)}
          className="rounded-md border border-input bg-background px-3 py-1.5 text-sm"
        >
          {timeRangeOptions.map((opt) => (
            <option key={opt.value} value={opt.value}>
              {opt.label}
            </option>
          ))}
        </select>
      </div>
    </header>
  )
}