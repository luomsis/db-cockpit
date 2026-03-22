import { Alert } from '@/types'

export const mockAlerts: Alert[] = [
  {
    id: '1',
    name: 'CPU使用率过高',
    severity: 'critical',
    endpoint: 'mysql-prod-01',
    metric: 'cpu_usage',
    threshold: 80,
    currentValue: 92.5,
    status: 'firing',
    startedAt: '2026-03-22T10:30:00Z',
    labels: { instance: '192.168.1.10', db: 'orders' },
  },
  {
    id: '2',
    name: '慢查询数量异常',
    severity: 'warning',
    endpoint: 'mysql-prod-01',
    metric: 'slow_queries',
    threshold: 100,
    currentValue: 156,
    status: 'firing',
    startedAt: '2026-03-22T09:15:00Z',
    labels: { instance: '192.168.1.10', db: 'users' },
  },
  {
    id: '3',
    name: '连接数接近上限',
    severity: 'warning',
    endpoint: 'mysql-prod-02',
    metric: 'connections',
    threshold: 150,
    currentValue: 142,
    status: 'firing',
    startedAt: '2026-03-22T11:00:00Z',
    labels: { instance: '192.168.1.11', db: 'products' },
  },
  {
    id: '4',
    name: '磁盘空间不足',
    severity: 'info',
    endpoint: 'postgres-prod-01',
    metric: 'disk_usage',
    threshold: 85,
    currentValue: 78,
    status: 'resolved',
    startedAt: '2026-03-21T08:00:00Z',
    resolvedAt: '2026-03-22T06:30:00Z',
    labels: { instance: '192.168.1.20', db: 'analytics' },
  },
]

export function getFiringAlertsCount(alerts: Alert[]): number {
  return alerts.filter((a) => a.status === 'firing').length
}

export function getCriticalAlertsCount(alerts: Alert[]): number {
  return alerts.filter((a) => a.severity === 'critical' && a.status === 'firing').length
}