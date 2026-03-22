import { DataPoint, Statistics } from '@/types'

export function calculateStatistics(points: DataPoint[]): Statistics {
  if (!points || points.length === 0) {
    return { min: 0, max: 0, avg: 0, sum: 0, count: 0 }
  }
  const values = points.map((p) => p.value)
  const sum = values.reduce((a, b) => a + b, 0)
  return {
    min: Math.min(...values),
    max: Math.max(...values),
    avg: sum / values.length,
    sum,
    count: values.length,
  }
}

export function formatNumber(num: number): string {
  if (num >= 1000000) {
    return (num / 1000000).toFixed(2) + 'M'
  }
  if (num >= 1000) {
    return (num / 1000).toFixed(2) + 'K'
  }
  return num.toFixed(2)
}