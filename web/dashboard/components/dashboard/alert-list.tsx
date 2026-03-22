'use client'

import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Alert } from '@/types'
import { formatTime } from '@/lib/time-utils'

interface AlertListProps {
  alerts: Alert[]
}

export function AlertList({ alerts }: AlertListProps) {
  const sortedAlerts = [...alerts].sort((a, b) => {
    const severityOrder = { critical: 0, warning: 1, info: 2 }
    return severityOrder[a.severity] - severityOrder[b.severity]
  })

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-lg">告警列表</CardTitle>
      </CardHeader>
      <CardContent>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>级别</TableHead>
              <TableHead>名称</TableHead>
              <TableHead>Endpoint</TableHead>
              <TableHead>当前值/阈值</TableHead>
              <TableHead>状态</TableHead>
              <TableHead>触发时间</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {sortedAlerts.length === 0 ? (
              <TableRow>
                <TableCell colSpan={6} className="text-center text-muted-foreground">
                  暂无告警
                </TableCell>
              </TableRow>
            ) : (
              sortedAlerts.map((alert) => (
                <TableRow key={alert.id}>
                  <TableCell>
                    <Badge variant={alert.severity}>
                      {alert.severity === 'critical'
                        ? '严重'
                        : alert.severity === 'warning'
                        ? '警告'
                        : '信息'}
                    </Badge>
                  </TableCell>
                  <TableCell className="font-medium">{alert.name}</TableCell>
                  <TableCell>{alert.endpoint}</TableCell>
                  <TableCell>
                    <span className={alert.status === 'firing' ? 'text-red-500' : ''}>
                      {alert.currentValue}
                    </span>
                    {' / '}
                    {alert.threshold}
                  </TableCell>
                  <TableCell>
                    <Badge variant={alert.status === 'firing' ? 'destructive' : 'secondary'}>
                      {alert.status === 'firing' ? '触发中' : '已解决'}
                    </Badge>
                  </TableCell>
                  <TableCell>{formatTime(alert.startedAt)}</TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  )
}