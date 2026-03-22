'use client'

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { Statistics } from '@/types'
import { formatNumber } from '@/lib/stats-utils'
import { AlertTriangle, TrendingUp, TrendingDown, Activity } from 'lucide-react'

interface MetricCardsProps {
  statistics: Statistics | null
  alertCount: number
  isLoading: boolean
}

export function MetricCards({ statistics, alertCount, isLoading }: MetricCardsProps) {
  if (isLoading) {
    return (
      <div className="grid grid-cols-4 gap-4 p-6">
        {[1, 2, 3, 4].map((i) => (
          <Card key={i}>
            <CardHeader className="pb-2">
              <Skeleton className="h-4 w-20" />
            </CardHeader>
            <CardContent>
              <Skeleton className="h-8 w-24" />
            </CardContent>
          </Card>
        ))}
      </div>
    )
  }

  const stats = statistics || { min: 0, max: 0, avg: 0, sum: 0, count: 0 }

  const cards = [
    {
      title: '平均值',
      value: formatNumber(stats.avg),
      icon: Activity,
      color: 'text-blue-500',
    },
    {
      title: '最大值',
      value: formatNumber(stats.max),
      icon: TrendingUp,
      color: 'text-green-500',
    },
    {
      title: '最小值',
      value: formatNumber(stats.min),
      icon: TrendingDown,
      color: 'text-yellow-500',
    },
    {
      title: '活跃告警',
      value: alertCount.toString(),
      icon: AlertTriangle,
      color: alertCount > 0 ? 'text-red-500' : 'text-muted-foreground',
    },
  ]

  return (
    <div className="grid grid-cols-4 gap-4 p-6">
      {cards.map((card) => (
        <Card key={card.title}>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              {card.title}
            </CardTitle>
            <card.icon className={`h-4 w-4 ${card.color}`} />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{card.value}</div>
          </CardContent>
        </Card>
      ))}
    </div>
  )
}