export interface SeriesMeta {
  id: string
  endpoint: string
  metric: string
  labels: Record<string, string>
  labels_hash: string
  created_at: string
}

export interface DataPoint {
  time: string
  value: number
}

// Series data from REST API (flat structure)
export interface Series {
  id: string
  endpoint: string
  metric: string
  labels: Record<string, string>
  labels_hash: string
  created_at: string
  points: DataPoint[]
  aggregated_points?: AggregatedPoint[]
  statistics?: Statistics
}

export interface AggregatedPoint {
  time: string
  value: number
  count: number
}

export interface Statistics {
  min: number
  max: number
  avg: number
  sum: number
  count: number
}

// 告警类型
export interface Alert {
  id: string
  name: string
  severity: 'critical' | 'warning' | 'info'
  endpoint: string
  metric: string
  threshold: number
  currentValue: number
  status: 'firing' | 'resolved'
  startedAt: string
  resolvedAt?: string
  labels: Record<string, string>
}

// UI状态类型
export type TimeRangeOption = '1h' | '6h' | '24h' | '7d'
export type RefreshInterval = 'off' | '30s' | '1m' | '5m'

export interface TimeRangeInput {
  start: string
  end: string
}

// Dashboard状态
export interface DashboardState {
  selectedEndpoint: string
  selectedMetric: string
  timeRange: TimeRangeOption
  refreshInterval: RefreshInterval
}