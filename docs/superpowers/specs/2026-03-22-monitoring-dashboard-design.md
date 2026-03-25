# 监控告警大盘设计文档

日期：2026-03-22

## 1. 概述

### 1.1 目标
为DBA/运维人员提供一个简易的监控告警大盘，展示数据库性能指标趋势和告警信息。

### 1.2 目标用户
DBA/运维人员 - 需要深入分析数据库性能指标，快速定位问题。

### 1.3 核心功能
- 时序指标可视化（基于DataQuery GraphQL API）
- 关键指标摘要卡片
- 告警列表（模拟数据）
- 自动刷新

## 2. 技术选型

| 层面 | 选型 | 说明 |
|------|------|------|
| 框架 | Next.js 14 (App Router) | 现代React框架，支持SSR/SSG |
| UI组件 | Shadcn UI | 基于Radix UI，可定制性强 |
| 图表 | Recharts | React原生图表库，简单易用 |
| 数据获取 | graphql-request | 轻量GraphQL客户端 |
| 样式 | Tailwind CSS | Shadcn自带，开发效率高 |
| 状态管理 | React useState + URLSearchParams | 筛选条件URL持久化 |

## 3. 架构设计

### 3.1 数据流

```
┌─────────────┐     GraphQL      ┌─────────────┐
│  Next.js    │ ──────────────▶  │   Gateway   │
│  Frontend   │   (JWT Auth)     │  :8080      │
│             │ ◀────────────── │             │
└─────────────┘     JSON        └─────────────┘
       │
       │ API代理 (避免CORS)
       ▼
  /api/graphql -> http://localhost:8080/graphql
```

### 3.2 API代理配置

```javascript
// next.config.js
module.exports = {
  async rewrites() {
    return [
      {
        source: '/api/graphql',
        destination: 'http://localhost:8080/graphql',
      },
    ]
  },
}
```

### 3.3 认证处理

Gateway要求JWT认证，请求需携带Authorization头：

```
Authorization: Bearer tenant_id:user_id:role
```

**开发阶段简化方案**：
- 使用固定的模拟JWT token（开发模式）
- GraphQL客户端配置请求中间件自动注入Authorization头

```typescript
// lib/graphql-client.ts
import { GraphQLClient } from 'graphql-request'

const DEV_TOKEN = 'dev_tenant:dev_user:admin'  // 仅开发环境使用

export const client = new GraphQLClient('/api/graphql', {
  headers: {
    Authorization: `Bearer ${DEV_TOKEN}`,
  },
})
```

**生产环境方案**（后续迭代）：
- 用户登录后JWT存储在localStorage/cookie
- GraphQL客户端从存储中读取token

## 4. 页面布局

```
┌─────────────────────────────────────────────────────────────────┐
│ Header: 监控告警大盘                          [时间范围选择器]  │
├─────────────────────────────────────────────────────────────────┤
│ 筛选栏: [Endpoint ▼] [Metric ▼] [刷新间隔 ▼] [刷新按钮]        │
├─────────────────────────────────────────────────────────────────┤
│ ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐                │
│ │ 指标卡片 │ │ 指标卡片 │ │ 指标卡片 │ │ 告警卡片 │                │
│ └─────────┘ └─────────┘ └─────────┘ └─────────┘                │
├─────────────────────────────────────────────────────────────────┤
│ ┌─────────────────────────────────────────────────────────────┐ │
│ │                     主图表区域                               │ │
│ │              (时序趋势线图，多series叠加)                     │ │
│ └─────────────────────────────────────────────────────────────┘ │
├─────────────────────────────────────────────────────────────────┤
│ ┌───────────────────────┐  ┌───────────────────────────────────┐│
│ │      告警列表         │  │         统计面板                   ││
│ │  (模拟数据，表格形式)  │  │  (前端计算 min/max/avg)           ││
│ └───────────────────────┘  └───────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────┘
```

### 4.1 组件结构

| 组件 | 功能 |
|------|------|
| `DashboardHeader` | 标题 + 时间范围选择（1h/6h/24h/7d） |
| `FilterBar` | Endpoint/Metric下拉、刷新间隔、手动刷新 |
| `MetricCards` | 4个关键指标摘要卡片（前端计算） |
| `MainChart` | Recharts时序折线图 |
| `AlertList` | 告警表格（模拟数据） |
| `StatsPanel` | 统计摘要面板（前端计算） |

## 5. GraphQL查询

### 5.1 获取Endpoints
```graphql
query GetEndpoints {
  endpoints
}
```

### 5.2 获取Metrics
```graphql
query GetMetrics($endpoint: String!) {
  metrics(endpoint: $endpoint)
}
```

### 5.3 获取时序数据

> **注意**：`series`查询不返回statistics字段，统计信息需在前端从points数据计算。

```graphql
query GetSeriesData($endpoint: String, $metric: String, $timeRange: TimeRangeInput!) {
  series(
    endpoint: $endpoint
    metric: $metric
    timeRange: $timeRange
    limit: 10
  ) {
    meta {
      id
      endpoint
      metric
      labels {
        entries {
          key
          value
        }
      }
    }
    points {
      time
      value
    }
  }
}
```

### 5.4 时间范围计算

将相对时间范围转换为GraphQL所需的TimeRangeInput：

```typescript
// lib/time-utils.ts
type TimeRangeOption = '1h' | '6h' | '24h' | '7d'

interface TimeRangeInput {
  start: string  // ISO 8601格式
  end: string    // ISO 8601格式
}

export function toTimeRange(range: TimeRangeOption): TimeRangeInput {
  const end = new Date()
  const durations: Record<TimeRangeOption, number> = {
    '1h': 60 * 60 * 1000,
    '6h': 6 * 60 * 60 * 1000,
    '24h': 24 * 60 * 60 * 1000,
    '7d': 7 * 24 * 60 * 60 * 1000,
  }
  const start = new Date(end.getTime() - durations[range])
  return {
    start: start.toISOString(),
    end: end.toISOString(),
  }
}
```

### 5.5 前端统计计算

由于`series`查询不返回statistics，需从前端计算：

```typescript
// lib/stats-utils.ts
interface DataPoint {
  time: string
  value: number
}

interface Statistics {
  min: number
  max: number
  avg: number
  sum: number
  count: number
}

export function calculateStatistics(points: DataPoint[]): Statistics {
  if (!points || points.length === 0) {
    return { min: 0, max: 0, avg: 0, sum: 0, count: 0 }
  }
  const values = points.map(p => p.value)
  const sum = values.reduce((a, b) => a + b, 0)
  return {
    min: Math.min(...values),
    max: Math.max(...values),
    avg: sum / values.length,
    sum,
    count: values.length,
  }
}
```

## 6. 文件结构

```
web/dashboard/
├── package.json
├── next.config.js
├── tailwind.config.js
├── tsconfig.json
├── app/
│   ├── layout.tsx          # 根布局，深色主题
│   ├── page.tsx            # 大盘主页面
│   └── globals.css         # 全局样式
├── components/
│   ├── ui/                 # Shadcn组件 (自动生成)
│   │   ├── button.tsx
│   │   ├── card.tsx
│   │   ├── select.tsx
│   │   └── table.tsx
│   ├── dashboard/
│   │   ├── header.tsx
│   │   ├── filter-bar.tsx
│   │   ├── metric-cards.tsx
│   │   ├── main-chart.tsx
│   │   ├── alert-list.tsx
│   │   └── stats-panel.tsx
│   └── providers.tsx
├── lib/
│   ├── graphql-client.ts   # GraphQL客户端封装（含认证）
│   ├── queries.ts          # GraphQL查询语句
│   ├── time-utils.ts       # 时间范围工具函数
│   ├── stats-utils.ts      # 统计计算工具函数
│   └── mock-alerts.ts      # 模拟告警数据
└── types/
    └── index.ts            # TypeScript类型定义
```

## 7. 依赖

```json
{
  "dependencies": {
    "next": "^14.0.0",
    "react": "^18.2.0",
    "react-dom": "^18.2.0",
    "recharts": "^2.10.0",
    "graphql-request": "^6.1.0",
    "class-variance-authority": "^0.7.0",
    "clsx": "^2.0.0",
    "tailwind-merge": "^2.0.0",
    "lucide-react": "^0.300.0"
  },
  "devDependencies": {
    "typescript": "^5.0.0",
    "tailwindcss": "^3.4.0",
    "postcss": "^8.0.0",
    "autoprefixer": "^10.0.0",
    "@types/react": "^18.2.0",
    "@types/node": "^20.0.0"
  }
}
```

## 8. 错误处理与加载状态

### 8.1 加载状态
- 初始加载：页面骨架屏（Skeleton组件）
- 刷新数据：图表区域显示loading overlay
- 下拉切换：仅更新相关组件

### 8.2 错误处理
- GraphQL请求失败：Toast提示 + 重试按钮
- 认证失败（401）：提示"请先登录"，跳转登录页
- 无数据：显示"暂无数据"空状态
- Endpoint/Metric列表为空：下拉框禁用，提示检查后端服务

### 8.3 自动刷新
- 支持刷新间隔：关闭 / 30s / 1m / 5m
- 页面不可见时暂停刷新（`document.visibilityState`）

## 9. 模拟告警数据

### 9.1 数据结构
```typescript
interface Alert {
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
```

### 9.2 告警列表交互
- 按严重程度排序（critical > warning > info）
- 状态筛选：全部 / 触发中 / 已解决
- 颜色标识：critical红色，warning橙色，info蓝色

## 10. 测试策略

- TypeScript类型检查
- 手动测试验证功能
- 集成测试：启动Gateway服务后验证GraphQL查询
- 边界情况：无数据、服务不可用、认证失败、大量数据点