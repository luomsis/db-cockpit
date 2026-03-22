'use client'

import { RefreshCw } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { RefreshInterval } from '@/types'

interface FilterBarProps {
  endpoints: string[]
  metrics: string[]
  selectedEndpoint: string
  selectedMetric: string
  refreshInterval: RefreshInterval
  isLoading: boolean
  onEndpointChange: (endpoint: string) => void
  onMetricChange: (metric: string) => void
  onRefreshIntervalChange: (interval: RefreshInterval) => void
  onRefresh: () => void
}

const refreshOptions: { value: RefreshInterval; label: string }[] = [
  { value: 'off', label: '关闭' },
  { value: '30s', label: '30秒' },
  { value: '1m', label: '1分钟' },
  { value: '5m', label: '5分钟' },
]

export function FilterBar({
  endpoints,
  metrics,
  selectedEndpoint,
  selectedMetric,
  refreshInterval,
  isLoading,
  onEndpointChange,
  onMetricChange,
  onRefreshIntervalChange,
  onRefresh,
}: FilterBarProps) {
  return (
    <div className="flex items-center gap-4 border-b border-border px-6 py-3">
      <div className="flex items-center gap-2">
        <label className="text-sm text-muted-foreground">Endpoint:</label>
        <select
          value={selectedEndpoint}
          onChange={(e) => onEndpointChange(e.target.value)}
          className="rounded-md border border-input bg-background px-3 py-1.5 text-sm"
          disabled={endpoints.length === 0}
        >
          <option value="">全部</option>
          {endpoints.map((ep) => (
            <option key={ep} value={ep}>
              {ep}
            </option>
          ))}
        </select>
      </div>

      <div className="flex items-center gap-2">
        <label className="text-sm text-muted-foreground">Metric:</label>
        <select
          value={selectedMetric}
          onChange={(e) => onMetricChange(e.target.value)}
          className="rounded-md border border-input bg-background px-3 py-1.5 text-sm"
          disabled={metrics.length === 0}
        >
          <option value="">全部</option>
          {metrics.map((m) => (
            <option key={m} value={m}>
              {m}
            </option>
          ))}
        </select>
      </div>

      <div className="flex items-center gap-2">
        <label className="text-sm text-muted-foreground">自动刷新:</label>
        <select
          value={refreshInterval}
          onChange={(e) => onRefreshIntervalChange(e.target.value as RefreshInterval)}
          className="rounded-md border border-input bg-background px-3 py-1.5 text-sm"
        >
          {refreshOptions.map((opt) => (
            <option key={opt.value} value={opt.value}>
              {opt.label}
            </option>
          ))}
        </select>
      </div>

      <Button
        variant="outline"
        size="sm"
        onClick={onRefresh}
        disabled={isLoading}
        className="ml-auto"
      >
        <RefreshCw className={`mr-2 h-4 w-4 ${isLoading ? 'animate-spin' : ''}`} />
        刷新
      </Button>
    </div>
  )
}