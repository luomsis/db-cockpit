# 监控告警大盘实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 创建一个简易的监控告警大盘前端应用，展示时序指标趋势和模拟告警数据。

**Architecture:** Next.js 14 SPA通过API代理调用Gateway GraphQL API，使用Shadcn UI组件和Recharts图表，深色主题Grafana风格。

**Tech Stack:** Next.js 14, React 18, TypeScript, Shadcn UI, Recharts, graphql-request, Tailwind CSS

---

## 文件结构

```
web/dashboard/
├── package.json                    # 项目依赖
├── next.config.js                  # Next.js配置（含API代理）
├── tailwind.config.ts              # Tailwind配置
├── postcss.config.js               # PostCSS配置
├── tsconfig.json                   # TypeScript配置
├── components.json                 # Shadcn UI配置
├── app/
│   ├── layout.tsx                  # 根布局（深色主题）
│   ├── page.tsx                    # 大盘主页面
│   └── globals.css                 # 全局样式
├── components/
│   ├── ui/                         # Shadcn组件（CLI生成）
│   │   ├── button.tsx
│   │   ├── card.tsx
│   │   ├── select.tsx
│   │   ├── table.tsx
│   │   ├── skeleton.tsx
│   │   └── badge.tsx
│   ├── dashboard/
│   │   ├── header.tsx              # 标题+时间选择
│   │   ├── filter-bar.tsx          # 筛选栏
│   │   ├── metric-cards.tsx        # 指标卡片
│   │   ├── main-chart.tsx          # 主图表
│   │   ├── alert-list.tsx          # 告警列表
│   │   └── stats-panel.tsx         # 统计面板
│   └── providers.tsx               # ThemeProvider
├── lib/
│   ├── utils.ts                    # cn工具函数（Shadcn需要）
│   ├── graphql-client.ts           # GraphQL客户端
│   ├── queries.ts                  # GraphQL查询
│   ├── time-utils.ts               # 时间工具
│   ├── stats-utils.ts              # 统计计算
│   └── mock-alerts.ts              # 模拟告警数据
└── types/
    └── index.ts                    # 类型定义
```

---

## Task 1: 项目初始化

**Files:**
- Create: `web/dashboard/package.json`
- Create: `web/dashboard/next.config.js`
- Create: `web/dashboard/tsconfig.json`
- Create: `web/dashboard/tailwind.config.ts`
- Create: `web/dashboard/postcss.config.js`

- [ ] **Step 1: 创建项目目录**

```bash
mkdir -p web/dashboard
cd web/dashboard
```

- [ ] **Step 2: 创建 package.json**

```json
{
  "name": "db-cockpit-dashboard",
  "version": "0.1.0",
  "private": true,
  "scripts": {
    "dev": "next dev",
    "build": "next build",
    "start": "next start",
    "lint": "next lint"
  },
  "dependencies": {
    "next": "14.2.3",
    "react": "^18.2.0",
    "react-dom": "^18.2.0",
    "recharts": "^2.12.7",
    "graphql-request": "^6.1.0",
    "class-variance-authority": "^0.7.0",
    "clsx": "^2.1.1",
    "tailwind-merge": "^2.3.0",
    "lucide-react": "^0.378.0"
  },
  "devDependencies": {
    "typescript": "^5.4.5",
    "@types/node": "^20.12.12",
    "@types/react": "^18.3.2",
    "@types/react-dom": "^18.3.0",
    "tailwindcss": "^3.4.3",
    "postcss": "^8.4.38",
    "autoprefixer": "^10.4.19"
  }
}
```

- [ ] **Step 3: 创建 next.config.js**

```javascript
/** @type {import('next').NextConfig} */
const nextConfig = {
  async rewrites() {
    return [
      {
        source: '/api/graphql',
        destination: 'http://localhost:8080/graphql',
      },
    ]
  },
}

module.exports = nextConfig
```

- [ ] **Step 4: 创建 tsconfig.json**

```json
{
  "compilerOptions": {
    "lib": ["dom", "dom.iterable", "esnext"],
    "allowJs": true,
    "skipLibCheck": true,
    "strict": true,
    "noEmit": true,
    "esModuleInterop": true,
    "module": "esnext",
    "moduleResolution": "bundler",
    "resolveJsonModule": true,
    "isolatedModules": true,
    "jsx": "preserve",
    "incremental": true,
    "plugins": [
      {
        "name": "next"
      }
    ],
    "paths": {
      "@/*": ["./*"]
    }
  },
  "include": ["next-env.d.ts", "**/*.ts", "**/*.tsx", ".next/types/**/*.ts"],
  "exclude": ["node_modules"]
}
```

- [ ] **Step 5: 创建 tailwind.config.ts**

```typescript
import type { Config } from 'tailwindcss'

const config: Config = {
  darkMode: ['class'],
  content: [
    './pages/**/*.{js,ts,jsx,tsx,mdx}',
    './components/**/*.{js,ts,jsx,tsx,mdx}',
    './app/**/*.{js,ts,jsx,tsx,mdx}',
  ],
  theme: {
    extend: {
      colors: {
        background: 'hsl(var(--background))',
        foreground: 'hsl(var(--foreground))',
        card: {
          DEFAULT: 'hsl(var(--card))',
          foreground: 'hsl(var(--card-foreground))',
        },
        muted: {
          DEFAULT: 'hsl(var(--muted))',
          foreground: 'hsl(var(--muted-foreground))',
        },
        border: 'hsl(var(--border))',
        input: 'hsl(var(--input))',
        primary: {
          DEFAULT: 'hsl(var(--primary))',
          foreground: 'hsl(var(--primary-foreground))',
        },
        secondary: {
          DEFAULT: 'hsl(var(--secondary))',
          foreground: 'hsl(var(--secondary-foreground))',
        },
        destructive: {
          DEFAULT: 'hsl(var(--destructive))',
          foreground: 'hsl(var(--destructive-foreground))',
        },
      },
      borderRadius: {
        lg: 'var(--radius)',
        md: 'calc(var(--radius) - 2px)',
        sm: 'calc(var(--radius) - 4px)',
      },
    },
  },
  plugins: [],
}
export default config
```

- [ ] **Step 6: 创建 postcss.config.js**

```javascript
module.exports = {
  plugins: {
    tailwindcss: {},
    autoprefixer: {},
  },
}
```

- [ ] **Step 7: 安装依赖**

```bash
cd web/dashboard && npm install
```

Expected: 依赖安装成功，生成node_modules和package-lock.json

- [ ] **Step 8: Commit**

```bash
git add web/dashboard/package.json web/dashboard/package-lock.json web/dashboard/next.config.js web/dashboard/tsconfig.json web/dashboard/tailwind.config.ts web/dashboard/postcss.config.js
git commit -m "feat(dashboard): initialize Next.js project with config files"
```

---

## Task 2: 创建全局样式和布局

**Files:**
- Create: `web/dashboard/app/globals.css`
- Create: `web/dashboard/app/layout.tsx`
- Create: `web/dashboard/components/providers.tsx`
- Create: `web/dashboard/lib/utils.ts`

- [ ] **Step 1: 创建 app 目录**

```bash
mkdir -p web/dashboard/app web/dashboard/components web/dashboard/lib web/dashboard/types
```

- [ ] **Step 2: 创建 lib/utils.ts (Shadcn需要的cn函数)**

```typescript
import { type ClassValue, clsx } from 'clsx'
import { twMerge } from 'tailwind-merge'

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}
```

- [ ] **Step 3: 创建 app/globals.css**

```css
@tailwind base;
@tailwind components;
@tailwind utilities;

@layer base {
  :root {
    --background: 222.2 84% 4.9%;
    --foreground: 210 40% 98%;
    --card: 222.2 84% 4.9%;
    --card-foreground: 210 40% 98%;
    --popover: 222.2 84% 4.9%;
    --popover-foreground: 210 40% 98%;
    --muted: 217.2 32.6% 17.5%;
    --muted-foreground: 215 20.2% 65.1%;
    --accent: 217.2 32.6% 17.5%;
    --accent-foreground: 210 40% 98%;
    --border: 217.2 32.6% 17.5%;
    --input: 217.2 32.6% 17.5%;
    --primary: 210 40% 98%;
    --primary-foreground: 222.2 47.4% 11.2%;
    --secondary: 217.2 32.6% 17.5%;
    --secondary-foreground: 210 40% 98%;
    --destructive: 0 62.8% 30.6%;
    --destructive-foreground: 210 40% 98%;
    --ring: 210 40% 98%;
    --radius: 0.5rem;
  }
}

@layer base {
  * {
    @apply border-border;
  }
  body {
    @apply bg-background text-foreground;
  }
}
```

- [ ] **Step 4: 创建 components/providers.tsx**

```typescript
'use client'

export function Providers({ children }: { children: React.ReactNode }) {
  return <>{children}</>
}
```

- [ ] **Step 5: 创建 app/layout.tsx**

```typescript
import type { Metadata } from 'next'
import { Inter } from 'next/font/google'
import './globals.css'
import { Providers } from '@/components/providers'

const inter = Inter({ subsets: ['latin'] })

export const metadata: Metadata = {
  title: '监控告警大盘 - DB Cockpit',
  description: '数据库智能驾驶舱监控大盘',
}

export default function RootLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <html lang="zh-CN" className="dark">
      <body className={inter.className}>
        <Providers>{children}</Providers>
      </body>
    </html>
  )
}
```

- [ ] **Step 6: 创建基础 app/page.tsx**

```typescript
export default function Home() {
  return (
    <main className="min-h-screen p-4">
      <h1 className="text-2xl font-bold">监控告警大盘</h1>
      <p className="text-muted-foreground mt-2">正在加载...</p>
    </main>
  )
}
```

- [ ] **Step 7: 验证开发服务器可以启动**

```bash
cd web/dashboard && npm run dev &
sleep 5
curl -s http://localhost:3000 | head -20
pkill -f "next dev"
```

Expected: 返回包含"监控告警大盘"的HTML内容

- [ ] **Step 8: Commit**

```bash
git add web/dashboard/app web/dashboard/components/providers.tsx web/dashboard/lib/utils.ts
git commit -m "feat(dashboard): add global styles and root layout with dark theme"
```

---

## Task 3: 创建类型定义和工具函数

**Files:**
- Create: `web/dashboard/types/index.ts`
- Create: `web/dashboard/lib/graphql-client.ts`
- Create: `web/dashboard/lib/queries.ts`
- Create: `web/dashboard/lib/time-utils.ts`
- Create: `web/dashboard/lib/stats-utils.ts`
- Create: `web/dashboard/lib/mock-alerts.ts`

- [ ] **Step 1: 创建 types/index.ts**

```typescript
// GraphQL响应类型
export interface LabelEntry {
  key: string
  value: string
}

export interface Labels {
  entries: LabelEntry[]
}

export interface SeriesMeta {
  id: string
  endpoint: string
  metric: string
  labels: Labels
}

export interface DataPoint {
  time: string
  value: number
}

export interface Series {
  meta: SeriesMeta
  points: DataPoint[]
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
```

- [ ] **Step 2: 创建 lib/graphql-client.ts**

```typescript
import { GraphQLClient } from 'graphql-request'

// 开发环境使用模拟token
const DEV_TOKEN = 'dev_tenant:dev_user:admin'

export const graphqlClient = new GraphQLClient('/api/graphql', {
  headers: {
    Authorization: `Bearer ${DEV_TOKEN}`,
  },
})
```

- [ ] **Step 3: 创建 lib/queries.ts**

```typescript
import { gql } from 'graphql-request'

export const GET_ENDPOINTS = gql`
  query GetEndpoints {
    endpoints
  }
`

export const GET_METRICS = gql`
  query GetMetrics($endpoint: String!) {
    metrics(endpoint: $endpoint)
  }
`

export const GET_SERIES_DATA = gql`
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
`
```

- [ ] **Step 4: 创建 lib/time-utils.ts**

```typescript
import { TimeRangeOption, TimeRangeInput } from '@/types'

const DURATIONS: Record<TimeRangeOption, number> = {
  '1h': 60 * 60 * 1000,
  '6h': 6 * 60 * 60 * 1000,
  '24h': 24 * 60 * 60 * 1000,
  '7d': 7 * 24 * 60 * 60 * 1000,
}

export function toTimeRange(range: TimeRangeOption): TimeRangeInput {
  const end = new Date()
  const start = new Date(end.getTime() - DURATIONS[range])
  return {
    start: start.toISOString(),
    end: end.toISOString(),
  }
}

export function formatTime(isoString: string): string {
  const date = new Date(isoString)
  return date.toLocaleString('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
}
```

- [ ] **Step 5: 创建 lib/stats-utils.ts**

```typescript
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
```

- [ ] **Step 6: 创建 lib/mock-alerts.ts**

```typescript
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
```

- [ ] **Step 7: Commit**

```bash
git add web/dashboard/types web/dashboard/lib/graphql-client.ts web/dashboard/lib/queries.ts web/dashboard/lib/time-utils.ts web/dashboard/lib/stats-utils.ts web/dashboard/lib/mock-alerts.ts
git commit -m "feat(dashboard): add types and utility functions"
```

---

## Task 4: 创建Shadcn UI组件

**Files:**
- Create: `web/dashboard/components/ui/button.tsx`
- Create: `web/dashboard/components/ui/card.tsx`
- Create: `web/dashboard/components/ui/select.tsx`
- Create: `web/dashboard/components/ui/table.tsx`
- Create: `web/dashboard/components/ui/skeleton.tsx`
- Create: `web/dashboard/components/ui/badge.tsx`

- [ ] **Step 1: 创建 components/ui 目录**

```bash
mkdir -p web/dashboard/components/ui web/dashboard/components/dashboard
```

- [ ] **Step 2: 创建 components/ui/button.tsx**

```typescript
import * as React from 'react'
import { Slot } from '@radix-ui/react-slot'
import { cva, type VariantProps } from 'class-variance-authority'

import { cn } from '@/lib/utils'

const buttonVariants = cva(
  'inline-flex items-center justify-center whitespace-nowrap rounded-md text-sm font-medium ring-offset-background transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50',
  {
    variants: {
      variant: {
        default: 'bg-primary text-primary-foreground hover:bg-primary/90',
        destructive:
          'bg-destructive text-destructive-foreground hover:bg-destructive/90',
        outline:
          'border border-input bg-background hover:bg-accent hover:text-accent-foreground',
        secondary:
          'bg-secondary text-secondary-foreground hover:bg-secondary/80',
        ghost: 'hover:bg-accent hover:text-accent-foreground',
        link: 'text-primary underline-offset-4 hover:underline',
      },
      size: {
        default: 'h-10 px-4 py-2',
        sm: 'h-9 rounded-md px-3',
        lg: 'h-11 rounded-md px-8',
        icon: 'h-10 w-10',
      },
    },
    defaultVariants: {
      variant: 'default',
      size: 'default',
    },
  }
)

export interface ButtonProps
  extends React.ButtonHTMLAttributes<HTMLButtonElement>,
    VariantProps<typeof buttonVariants> {
  asChild?: boolean
}

const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
  ({ className, variant, size, asChild = false, ...props }, ref) => {
    const Comp = asChild ? Slot : 'button'
    return (
      <Comp
        className={cn(buttonVariants({ variant, size, className }))}
        ref={ref}
        {...props}
      />
    )
  }
)
Button.displayName = 'Button'

export { Button, buttonVariants }
```

- [ ] **Step 3: 安装Radix UI依赖**

```bash
cd web/dashboard && npm install @radix-ui/react-slot @radix-ui/react-select
```

Expected: 依赖安装成功

- [ ] **Step 4: 创建 components/ui/card.tsx**

```typescript
import * as React from 'react'

import { cn } from '@/lib/utils'

const Card = React.forwardRef<
  HTMLDivElement,
  React.HTMLAttributes<HTMLDivElement>
>(({ className, ...props }, ref) => (
  <div
    ref={ref}
    className={cn(
      'rounded-lg border bg-card text-card-foreground shadow-sm',
      className
    )}
    {...props}
  />
))
Card.displayName = 'Card'

const CardHeader = React.forwardRef<
  HTMLDivElement,
  React.HTMLAttributes<HTMLDivElement>
>(({ className, ...props }, ref) => (
  <div
    ref={ref}
    className={cn('flex flex-col space-y-1.5 p-6', className)}
    {...props}
  />
))
CardHeader.displayName = 'CardHeader'

const CardTitle = React.forwardRef<
  HTMLParagraphElement,
  React.HTMLAttributes<HTMLHeadingElement>
>(({ className, ...props }, ref) => (
  <h3
    ref={ref}
    className={cn(
      'text-2xl font-semibold leading-none tracking-tight',
      className
    )}
    {...props}
  />
))
CardTitle.displayName = 'CardTitle'

const CardDescription = React.forwardRef<
  HTMLParagraphElement,
  React.HTMLAttributes<HTMLParagraphElement>
>(({ className, ...props }, ref) => (
  <p
    ref={ref}
    className={cn('text-sm text-muted-foreground', className)}
    {...props}
  />
))
CardDescription.displayName = 'CardDescription'

const CardContent = React.forwardRef<
  HTMLDivElement,
  React.HTMLAttributes<HTMLDivElement>
>(({ className, ...props }, ref) => (
  <div ref={ref} className={cn('p-6 pt-0', className)} {...props} />
))
CardContent.displayName = 'CardContent'

const CardFooter = React.forwardRef<
  HTMLDivElement,
  React.HTMLAttributes<HTMLDivElement>
>(({ className, ...props }, ref) => (
  <div
    ref={ref}
    className={cn('flex items-center p-6 pt-0', className)}
    {...props}
  />
))
CardFooter.displayName = 'CardFooter'

export { Card, CardHeader, CardFooter, CardTitle, CardDescription, CardContent }
```

- [ ] **Step 5: 创建 components/ui/select.tsx**

```typescript
import * as React from 'react'
import * as SelectPrimitive from '@radix-ui/react-select'
import { Check, ChevronDown, ChevronUp } from 'lucide-react'

import { cn } from '@/lib/utils'

const Select = SelectPrimitive.Root

const SelectGroup = SelectPrimitive.Group

const SelectValue = SelectPrimitive.Value

const SelectTrigger = React.forwardRef<
  React.ElementRef<typeof SelectPrimitive.Trigger>,
  React.ComponentPropsWithoutRef<typeof SelectPrimitive.Trigger>
>(({ className, children, ...props }, ref) => (
  <SelectPrimitive.Trigger
    ref={ref}
    className={cn(
      'flex h-10 w-full items-center justify-between rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 [&>span]:line-clamp-1',
      className
    )}
    {...props}
  >
    {children}
    <SelectPrimitive.Icon asChild>
      <ChevronDown className="h-4 w-4 opacity-50" />
    </SelectPrimitive.Icon>
  </SelectPrimitive.Trigger>
))
SelectTrigger.displayName = SelectPrimitive.Trigger.displayName

const SelectScrollUpButton = React.forwardRef<
  React.ElementRef<typeof SelectPrimitive.ScrollUpButton>,
  React.ComponentPropsWithoutRef<typeof SelectPrimitive.ScrollUpButton>
>(({ className, ...props }, ref) => (
  <SelectPrimitive.ScrollUpButton
    ref={ref}
    className={cn(
      'flex cursor-default items-center justify-center py-1',
      className
    )}
    {...props}
  >
    <ChevronUp className="h-4 w-4" />
  </SelectPrimitive.ScrollUpButton>
))
SelectScrollUpButton.displayName = SelectPrimitive.ScrollUpButton.displayName

const SelectScrollDownButton = React.forwardRef<
  React.ElementRef<typeof SelectPrimitive.ScrollDownButton>,
  React.ComponentPropsWithoutRef<typeof SelectPrimitive.ScrollDownButton>
>(({ className, ...props }, ref) => (
  <SelectPrimitive.ScrollDownButton
    ref={ref}
    className={cn(
      'flex cursor-default items-center justify-center py-1',
      className
    )}
    {...props}
  >
    <ChevronDown className="h-4 w-4" />
  </SelectPrimitive.ScrollDownButton>
))
SelectScrollDownButton.displayName =
  SelectPrimitive.ScrollDownButton.displayName

const SelectContent = React.forwardRef<
  React.ElementRef<typeof SelectPrimitive.Content>,
  React.ComponentPropsWithoutRef<typeof SelectPrimitive.Content>
>(({ className, children, position = 'popper', ...props }, ref) => (
  <SelectPrimitive.Portal>
    <SelectPrimitive.Content
      ref={ref}
      className={cn(
        'relative z-50 max-h-96 min-w-[8rem] overflow-hidden rounded-md border bg-popover text-popover-foreground shadow-md data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95 data-[side=bottom]:slide-in-from-top-2 data-[side=left]:slide-in-from-right-2 data-[side=right]:slide-in-from-left-2 data-[side=top]:slide-in-from-bottom-2',
        position === 'popper' &&
          'data-[side=bottom]:translate-y-1 data-[side=left]:-translate-x-1 data-[side=right]:translate-x-1 data-[side=top]:-translate-y-1',
        className
      )}
      position={position}
      {...props}
    >
      <SelectScrollUpButton />
      <SelectPrimitive.Viewport
        className={cn(
          'p-1',
          position === 'popper' &&
            'h-[var(--radix-select-trigger-height)] w-full min-w-[var(--radix-select-trigger-width)]'
        )}
      >
        {children}
      </SelectPrimitive.Viewport>
      <SelectScrollDownButton />
    </SelectPrimitive.Content>
  </SelectPrimitive.Portal>
))
SelectContent.displayName = SelectPrimitive.Content.displayName

const SelectLabel = React.forwardRef<
  React.ElementRef<typeof SelectPrimitive.Label>,
  React.ComponentPropsWithoutRef<typeof SelectPrimitive.Label>
>(({ className, ...props }, ref) => (
  <SelectPrimitive.Label
    ref={ref}
    className={cn('py-1.5 pl-8 pr-2 text-sm font-semibold', className)}
    {...props}
  />
))
SelectLabel.displayName = SelectPrimitive.Label.displayName

const SelectItem = React.forwardRef<
  React.ElementRef<typeof SelectPrimitive.Item>,
  React.ComponentPropsWithoutRef<typeof SelectPrimitive.Item>
>(({ className, children, ...props }, ref) => (
  <SelectPrimitive.Item
    ref={ref}
    className={cn(
      'relative flex w-full cursor-default select-none items-center rounded-sm py-1.5 pl-8 pr-2 text-sm outline-none focus:bg-accent focus:text-accent-foreground data-[disabled]:pointer-events-none data-[disabled]:opacity-50',
      className
    )}
    {...props}
  >
    <span className="absolute left-2 flex h-3.5 w-3.5 items-center justify-center">
      <SelectPrimitive.ItemIndicator>
        <Check className="h-4 w-4" />
      </SelectPrimitive.ItemIndicator>
    </span>

    <SelectPrimitive.ItemText>{children}</SelectPrimitive.ItemText>
  </SelectPrimitive.Item>
))
SelectItem.displayName = SelectPrimitive.Item.displayName

const SelectSeparator = React.forwardRef<
  React.ElementRef<typeof SelectPrimitive.Separator>,
  React.ComponentPropsWithoutRef<typeof SelectPrimitive.Separator>
>(({ className, ...props }, ref) => (
  <SelectPrimitive.Separator
    ref={ref}
    className={cn('-mx-1 my-1 h-px bg-muted', className)}
    {...props}
  />
))
SelectSeparator.displayName = SelectPrimitive.Separator.displayName

export {
  Select,
  SelectGroup,
  SelectValue,
  SelectTrigger,
  SelectContent,
  SelectLabel,
  SelectItem,
  SelectSeparator,
  SelectScrollUpButton,
  SelectScrollDownButton,
}
```

- [ ] **Step 6: 创建 components/ui/table.tsx**

```typescript
import * as React from 'react'

import { cn } from '@/lib/utils'

const Table = React.forwardRef<
  HTMLTableElement,
  React.HTMLAttributes<HTMLTableElement>
>(({ className, ...props }, ref) => (
  <div className="relative w-full overflow-auto">
    <table
      ref={ref}
      className={cn('w-full caption-bottom text-sm', className)}
      {...props}
    />
  </div>
))
Table.displayName = 'Table'

const TableHeader = React.forwardRef<
  HTMLTableSectionElement,
  React.HTMLAttributes<HTMLTableSectionElement>
>(({ className, ...props }, ref) => (
  <thead ref={ref} className={cn('[&_tr]:border-b', className)} {...props} />
))
TableHeader.displayName = 'TableHeader'

const TableBody = React.forwardRef<
  HTMLTableSectionElement,
  React.HTMLAttributes<HTMLTableSectionElement>
>(({ className, ...props }, ref) => (
  <tbody
    ref={ref}
    className={cn('[&_tr:last-child]:border-0', className)}
    {...props}
  />
))
TableBody.displayName = 'TableBody'

const TableFooter = React.forwardRef<
  HTMLTableSectionElement,
  React.HTMLAttributes<HTMLTableSectionElement>
>(({ className, ...props }, ref) => (
  <tfoot
    ref={ref}
    className={cn(
      'border-t bg-muted/50 font-medium [&>tr]:last:border-b-0',
      className
    )}
    {...props}
  />
))
TableFooter.displayName = 'TableFooter'

const TableRow = React.forwardRef<
  HTMLTableRowElement,
  React.HTMLAttributes<HTMLTableRowElement>
>(({ className, ...props }, ref) => (
  <tr
    ref={ref}
    className={cn(
      'border-b transition-colors hover:bg-muted/50 data-[state=selected]:bg-muted',
      className
    )}
    {...props}
  />
))
TableRow.displayName = 'TableRow'

const TableHead = React.forwardRef<
  HTMLTableCellElement,
  React.ThHTMLAttributes<HTMLTableCellElement>
>(({ className, ...props }, ref) => (
  <th
    ref={ref}
    className={cn(
      'h-12 px-4 text-left align-middle font-medium text-muted-foreground [&:has([role=checkbox])]:pr-0',
      className
    )}
    {...props}
  />
))
TableHead.displayName = 'TableHead'

const TableCell = React.forwardRef<
  HTMLTableCellElement,
  React.TdHTMLAttributes<HTMLTableCellElement>
>(({ className, ...props }, ref) => (
  <td
    ref={ref}
    className={cn('p-4 align-middle [&:has([role=checkbox])]:pr-0', className)}
    {...props}
  />
))
TableCell.displayName = 'TableCell'

const TableCaption = React.forwardRef<
  HTMLTableCaptionElement,
  React.HTMLAttributes<HTMLTableCaptionElement>
>(({ className, ...props }, ref) => (
  <caption
    ref={ref}
    className={cn('mt-4 text-sm text-muted-foreground', className)}
    {...props}
  />
))
TableCaption.displayName = 'TableCaption'

export {
  Table,
  TableHeader,
  TableBody,
  TableFooter,
  TableHead,
  TableRow,
  TableCell,
  TableCaption,
}
```

- [ ] **Step 7: 创建 components/ui/skeleton.tsx**

```typescript
import { cn } from '@/lib/utils'

function Skeleton({
  className,
  ...props
}: React.HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className={cn('animate-pulse rounded-md bg-muted', className)}
      {...props}
    />
  )
}

export { Skeleton }
```

- [ ] **Step 8: 创建 components/ui/badge.tsx**

```typescript
import * as React from 'react'
import { cva, type VariantProps } from 'class-variance-authority'

import { cn } from '@/lib/utils'

const badgeVariants = cva(
  'inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-semibold transition-colors focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2',
  {
    variants: {
      variant: {
        default:
          'border-transparent bg-primary text-primary-foreground hover:bg-primary/80',
        secondary:
          'border-transparent bg-secondary text-secondary-foreground hover:bg-secondary/80',
        destructive:
          'border-transparent bg-destructive text-destructive-foreground hover:bg-destructive/80',
        outline: 'text-foreground',
        warning:
          'border-transparent bg-yellow-500/20 text-yellow-500',
        critical:
          'border-transparent bg-red-500/20 text-red-500',
        info:
          'border-transparent bg-blue-500/20 text-blue-500',
      },
    },
    defaultVariants: {
      variant: 'default',
    },
  }
)

export interface BadgeProps
  extends React.HTMLAttributes<HTMLDivElement>,
    VariantProps<typeof badgeVariants> {}

function Badge({ className, variant, ...props }: BadgeProps) {
  return (
    <div className={cn(badgeVariants({ variant }), className)} {...props} />
  )
}

export { Badge, badgeVariants }
```

- [ ] **Step 9: Commit**

```bash
git add web/dashboard/components/ui web/dashboard/package.json web/dashboard/package-lock.json
git commit -m "feat(dashboard): add Shadcn UI components (button, card, select, table, skeleton, badge)"
```

---

## Task 5: 创建Dashboard组件

**Files:**
- Create: `web/dashboard/components/dashboard/header.tsx`
- Create: `web/dashboard/components/dashboard/filter-bar.tsx`
- Create: `web/dashboard/components/dashboard/metric-cards.tsx`
- Create: `web/dashboard/components/dashboard/main-chart.tsx`
- Create: `web/dashboard/components/dashboard/alert-list.tsx`
- Create: `web/dashboard/components/dashboard/stats-panel.tsx`

- [ ] **Step 1: 创建 components/dashboard/header.tsx**

> **Note**: 使用原生`<select>`元素而非Shadcn Select组件，以保持简易性。如需更丰富的交互体验，可后续替换。

```typescript
'use client'

import { TimeRangeOption } from '@/types'

interface HeaderProps {
  timeRange: TimeRangeOption
  onTimeRangeChange: (range: TimeRangeOption) => void
}

const timeRangeOptions: { value: TimeRangeOption; label: string }[] = [
  { value: '1h', label: '最近1小时' },
  { value: '6h', label: '最近6小时' },
  { value: '24h', label: '最近24小时' },
  { value: '7d', label: '最近7天' },
]

export function Header({ timeRange, onTimeRangeChange }: HeaderProps) {
  return (
    <header className="flex items-center justify-between border-b border-border px-6 py-4">
      <h1 className="text-xl font-semibold">监控告警大盘</h1>
      <div className="flex items-center gap-2">
        <span className="text-sm text-muted-foreground">时间范围:</span>
        <select
          value={timeRange}
          onChange={(e) => onTimeRangeChange(e.target.value as TimeRangeOption)}
          className="rounded-md border border-input bg-background px-3 py-1.5 text-sm"
        >
          {timeRangeOptions.map((opt) => (
            <option key={opt.value} value={opt.value}>
              {opt.label}
            </option>
          ))}
        </select>
      </div>
    </header>
  )
}
```

- [ ] **Step 2: 创建 components/dashboard/filter-bar.tsx**

> **Note**: 使用原生`<select>`元素而非Shadcn Select组件，以保持简易性。

```typescript
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
```

- [ ] **Step 3: 创建 components/dashboard/metric-cards.tsx**

```typescript
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
```

- [ ] **Step 4: 创建 components/dashboard/main-chart.tsx**

```typescript
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
  const timeMap = new Map<string, Record<string, number>>()

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
```

- [ ] **Step 5: 创建 components/dashboard/alert-list.tsx**

```typescript
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
```

- [ ] **Step 6: 创建 components/dashboard/stats-panel.tsx**

```typescript
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
```

- [ ] **Step 7: Commit**

```bash
git add web/dashboard/components/dashboard
git commit -m "feat(dashboard): add dashboard components (header, filter-bar, metric-cards, main-chart, alert-list, stats-panel)"
```

---

## Task 6: 创建主页面

**Files:**
- Modify: `web/dashboard/app/page.tsx`

- [ ] **Step 1: 创建完整的 app/page.tsx**

```typescript
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
```

- [ ] **Step 2: 验证TypeScript编译**

```bash
cd web/dashboard && npx tsc --noEmit
```

Expected: 无错误输出

- [ ] **Step 3: Commit**

```bash
git add web/dashboard/app/page.tsx
git commit -m "feat(dashboard): implement main dashboard page with data fetching and auto-refresh"
```

---

## Task 7: 测试与验证

- [ ] **Step 1: 启动后端服务（在另一个终端）**

```bash
# 在项目根目录
go run cmd/gateway/main.go &
go run cmd/dataquery/main.go &
```

Expected: Gateway在8080端口启动，DataQuery服务正常运行

- [ ] **Step 2: 启动前端开发服务器**

```bash
cd web/dashboard && npm run dev
```

Expected: Next.js开发服务器在3000端口启动

- [ ] **Step 3: 验证页面功能**

打开浏览器访问 http://localhost:3000

检查项目：
1. 页面显示深色主题
2. Header显示"监控告警大盘"和时间选择器
3. FilterBar显示Endpoint/Metric下拉框（数据来自后端）
4. 指标卡片显示统计数据
5. 时序图表正确渲染
6. 告警列表显示模拟数据
7. 切换时间范围后数据刷新
8. 自动刷新功能正常

- [ ] **Step 4: 最终Commit**

```bash
git add -A
git commit -m "feat(dashboard): complete monitoring dashboard implementation"
```

---

## 完成检查清单

- [ ] 项目结构正确
- [ ] TypeScript编译无错误
- [ ] 所有组件渲染正常
- [ ] GraphQL查询正常工作
- [ ] 深色主题正确应用
- [ ] 自动刷新功能正常
- [ ] 告警列表显示模拟数据
- [ ] 统计计算正确