'use client'

import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from 'recharts'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { Series } from '@/types'
import { formatTime } from '@/lib/time-utils'

interface MainChartProps {
  series: Series[]
  isLoading: boolean
}

const COLORS = ['#8884d8', '#82ca9d', '#ffc658', '#ff7300', '#00C49F', '#FFBB28', '#FF8042']

export function MainChart({ series, isLoading }: MainChartProps) {
  if (isLoading) {
    return (
      <div className="px-6 pb-6">
        <Card>
          <CardHeader>
            <Skeleton className="h-6 w-32" />
          </CardHeader>
          <CardContent>
            <Skeleton className="h-[300px] w-full" />
          </CardContent>
        </Card>
      </div>
    )
  }

  if (series.length === 0) {
    return (
      <div className="px-6 pb-6">
        <Card>
          <CardHeader>
            <CardTitle>时序趋势图</CardTitle>
          </CardHeader>
          <CardContent className="flex h-[300px] items-center justify-center text-muted-foreground">
            暂无数据
          </CardContent>
        </Card>
      </div>
    )
  }

  // 合并所有series的数据点，按时间对齐
  const timeMap = new Map<string, Record<string, string | number>>()

  series.forEach((s, index) => {
    const label = s.meta.labels.entries
      .map((e) => `${e.key}=${e.value}`)
      .join(', ') || `Series ${index + 1}`

    s.points.forEach((point) => {
      const timeKey = point.time
      if (!timeMap.has(timeKey)) {
        timeMap.set(timeKey, { time: timeKey })
      }
      timeMap.get(timeKey)![label] = point.value
    })
  })

  const chartData = Array.from(timeMap.values()).sort(
    (a, b) => new Date(a.time).getTime() - new Date(b.time).getTime()
  )

  const seriesLabels = series.map((s, index) =>
    s.meta.labels.entries.map((e) => `${e.key}=${e.value}`).join(', ') ||
    `Series ${index + 1}`
  )

  return (
    <div className="px-6 pb-6">
      <Card>
        <CardHeader>
          <CardTitle>时序趋势图</CardTitle>
        </CardHeader>
        <CardContent>
          <ResponsiveContainer width="100%" height={300}>
            <LineChart data={chartData}>
              <CartesianGrid strokeDasharray="3 3" stroke="#374151" />
              <XAxis
                dataKey="time"
                tickFormatter={formatTime}
                stroke="#9CA3AF"
                fontSize={12}
              />
              <YAxis stroke="#9CA3AF" fontSize={12} />
              <Tooltip
                contentStyle={{
                  backgroundColor: '#1F2937',
                  border: '1px solid #374151',
                  borderRadius: '6px',
                }}
                labelFormatter={formatTime}
              />
              <Legend />
              {seriesLabels.map((label, index) => (
                <Line
                  key={label}
                  type="monotone"
                  dataKey={label}
                  stroke={COLORS[index % COLORS.length]}
                  dot={false}
                  strokeWidth={2}
                />
              ))}
            </LineChart>
          </ResponsiveContainer>
        </CardContent>
      </Card>
    </div>
  )
}