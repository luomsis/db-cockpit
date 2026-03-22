'use client'

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Statistics } from '@/types'
import { formatNumber } from '@/lib/stats-utils'

interface StatsPanelProps {
  statistics: Statistics | null
}

export function StatsPanel({ statistics }: StatsPanelProps) {
  const stats = statistics || { min: 0, max: 0, avg: 0, sum: 0, count: 0 }

  const items = [
    { label: '最小值', value: formatNumber(stats.min) },
    { label: '最大值', value: formatNumber(stats.max) },
    { label: '平均值', value: formatNumber(stats.avg) },
    { label: '总和', value: formatNumber(stats.sum) },
    { label: '数据点数', value: stats.count.toString() },
  ]

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-lg">统计信息</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="space-y-3">
          {items.map((item) => (
            <div key={item.label} className="flex justify-between">
              <span className="text-muted-foreground">{item.label}</span>
              <span className="font-medium">{item.value}</span>
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  )
}