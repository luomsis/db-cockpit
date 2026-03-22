'use client'

import { useState, useEffect, useCallback } from 'react'
import { Header } from '@/components/dashboard/header'
import { FilterBar } from '@/components/dashboard/filter-bar'
import { MetricCards } from '@/components/dashboard/metric-cards'
import { MainChart } from '@/components/dashboard/main-chart'
import { AlertList } from '@/components/dashboard/alert-list'
import { StatsPanel } from '@/components/dashboard/stats-panel'
import { graphqlClient } from '@/lib/graphql-client'
import { GET_ENDPOINTS, GET_METRICS, GET_SERIES_DATA } from '@/lib/queries'
import { toTimeRange } from '@/lib/time-utils'
import { calculateStatistics } from '@/lib/stats-utils'
import { mockAlerts, getFiringAlertsCount } from '@/lib/mock-alerts'
import {
  TimeRangeOption,
  RefreshInterval,
  Series,
  Statistics,
} from '@/types'

export default function DashboardPage() {
  // 筛选状态
  const [timeRange, setTimeRange] = useState<TimeRangeOption>('1h')
  const [selectedEndpoint, setSelectedEndpoint] = useState('')
  const [selectedMetric, setSelectedMetric] = useState('')
  const [refreshInterval, setRefreshInterval] = useState<RefreshInterval>('off')

  // 数据状态
  const [endpoints, setEndpoints] = useState<string[]>([])
  const [metrics, setMetrics] = useState<string[]>([])
  const [series, setSeries] = useState<Series[]>([])
  const [statistics, setStatistics] = useState<Statistics | null>(null)

  // 加载状态
  const [isLoading, setIsLoading] = useState(false)
  const [isLoadingEndpoints, setIsLoadingEndpoints] = useState(false)
  const [isLoadingMetrics, setIsLoadingMetrics] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // 获取endpoints
  const fetchEndpoints = useCallback(async () => {
    setIsLoadingEndpoints(true)
    try {
      const data = await graphqlClient.request<{ endpoints: string[] }>(GET_ENDPOINTS)
      setEndpoints(data.endpoints || [])
    } catch (err) {
      console.error('Failed to fetch endpoints:', err)
      setEndpoints([])
    } finally {
      setIsLoadingEndpoints(false)
    }
  }, [])

  // 获取metrics
  const fetchMetrics = useCallback(async (endpoint: string) => {
    if (!endpoint) {
      setMetrics([])
      return
    }
    setIsLoadingMetrics(true)
    try {
      const data = await graphqlClient.request<{ metrics: string[] }>(
        GET_METRICS,
        { endpoint }
      )
      setMetrics(data.metrics || [])
    } catch (err) {
      console.error('Failed to fetch metrics:', err)
      setMetrics([])
    } finally {
      setIsLoadingMetrics(false)
    }
  }, [])

  // 获取时序数据
  const fetchSeriesData = useCallback(async () => {
    setIsLoading(true)
    setError(null)
    try {
      const timeRangeInput = toTimeRange(timeRange)
      const variables: Record<string, unknown> = { timeRange: timeRangeInput }
      if (selectedEndpoint) variables.endpoint = selectedEndpoint
      if (selectedMetric) variables.metric = selectedMetric

      const data = await graphqlClient.request<{ series: Series[] }>(
        GET_SERIES_DATA,
        variables
      )

      const seriesData = data.series || []
      setSeries(seriesData)

      // 计算统计信息
      const allPoints = seriesData.flatMap((s) => s.points)
      setStatistics(calculateStatistics(allPoints))
    } catch (err) {
      console.error('Failed to fetch series data:', err)
      setError('获取数据失败，请检查后端服务是否正常运行')
      setSeries([])
      setStatistics(null)
    } finally {
      setIsLoading(false)
    }
  }, [timeRange, selectedEndpoint, selectedMetric])

  // 初始化：获取endpoints
  useEffect(() => {
    fetchEndpoints()
  }, [fetchEndpoints])

  // 当endpoint变化时获取metrics
  useEffect(() => {
    fetchMetrics(selectedEndpoint)
  }, [selectedEndpoint, fetchMetrics])

  // 当筛选条件变化时获取数据
  useEffect(() => {
    fetchSeriesData()
  }, [fetchSeriesData])

  // 自动刷新
  useEffect(() => {
    if (refreshInterval === 'off') return

    const intervals: Record<RefreshInterval, number> = {
      off: 0,
      '30s': 30000,
      '1m': 60000,
      '5m': 300000,
    }

    const interval = setInterval(() => {
      if (document.visibilityState === 'visible') {
        fetchSeriesData()
      }
    }, intervals[refreshInterval])

    return () => clearInterval(interval)
  }, [refreshInterval, fetchSeriesData])

  const alertCount = getFiringAlertsCount(mockAlerts)

  return (
    <main className="min-h-screen">
      <Header timeRange={timeRange} onTimeRangeChange={setTimeRange} />

      <FilterBar
        endpoints={endpoints}
        metrics={metrics}
        selectedEndpoint={selectedEndpoint}
        selectedMetric={selectedMetric}
        refreshInterval={refreshInterval}
        isLoading={isLoading}
        onEndpointChange={setSelectedEndpoint}
        onMetricChange={setSelectedMetric}
        onRefreshIntervalChange={setRefreshInterval}
        onRefresh={fetchSeriesData}
      />

      {error && (
        <div className="mx-6 mt-4 rounded-md border border-red-500/50 bg-red-500/10 p-4 text-red-500">
          {error}
        </div>
      )}

      <MetricCards
        statistics={statistics}
        alertCount={alertCount}
        isLoading={isLoading}
      />

      <MainChart series={series} isLoading={isLoading} />

      <div className="grid grid-cols-3 gap-4 px-6 pb-6">
        <div className="col-span-2">
          <AlertList alerts={mockAlerts} />
        </div>
        <div>
          <StatsPanel statistics={statistics} />
        </div>
      </div>
    </main>
  )
}