# Data Query Service GraphQL API 使用指南

## 概述

Data Query Service 提供 GraphQL 接口用于查询时序数据。支持以下功能：

- 查询所有端点和指标
- 基于标签过滤的时序查询
- 按 ID 查询单个时序
- 批量查询多个时序
- 时间聚合（AVG/MIN/MAX/SUM/COUNT）

## 服务端点

| 服务 | 地址 | 说明 |
|------|------|------|
| Data Query Service | `http://localhost:8084/graphql` | 直接访问 |
| Gateway Proxy | `http://localhost:8080/graphql` | 通过 Gateway 代理访问 |
| GraphQL Playground | `http://localhost:8080/graphql/playground` | 交互式查询界面 |

---

## 查询示例

### 1. 查询所有端点

获取系统中所有的数据采集端点列表。

```bash
curl -s -X POST http://localhost:8080/graphql \
  -H "Content-Type: application/json" \
  -d '{"query": "{ endpoints }"}'
```

**响应示例：**
```json
{
  "data": {
    "endpoints": ["/api/health", "/api/metrics", "/test/endpoint"]
  }
}
```

---

### 2. 查询指定端点的指标

获取指定端点下的所有指标名称。

```bash
curl -s -X POST http://localhost:8080/graphql \
  -H "Content-Type: application/json" \
  -d '{"query": "{ metrics(endpoint: \"/api/metrics\") }"}'
```

**响应示例：**
```json
{
  "data": {
    "metrics": ["cpu_usage", "disk_io", "memory_usage", "network_bytes"]
  }
}
```

---

### 3. 查询时序数据（基本）

根据端点和指标查询时序数据，必须指定时间范围。

```bash
curl -s -X POST http://localhost:8080/graphql \
  -H "Content-Type: application/json" \
  -d '{
    "query": "query($tr: TimeRangeInput!) { series(endpoint: \"/api/metrics\", metric: \"cpu_usage\", timeRange: $tr, limit: 2) { meta { id endpoint metric labels { entries { key value } } } points { time value } } }",
    "variables": {
      "tr": {
        "start": "2026-03-19T00:00:00Z",
        "end": "2026-03-20T23:59:59Z"
      }
    }
  }'
```

**响应示例：**
```json
{
  "data": {
    "series": [
      {
        "meta": {
          "id": "1",
          "endpoint": "/api/metrics",
          "metric": "cpu_usage",
          "labels": {
            "entries": [
              {"key": "env", "value": "prod"},
              {"key": "host", "value": "server1"},
              {"key": "region", "value": "us-east"}
            ]
          }
        },
        "points": [
          {"time": "2026-03-19T14:56:34+08:00", "value": 65.51},
          {"time": "2026-03-19T15:01:34+08:00", "value": 95.03}
        ]
      }
    ]
  }
}
```

---

### 4. 查询时序数据（标签过滤）

使用标签表达式过滤时序数据。

```bash
curl -s -X POST http://localhost:8080/graphql \
  -H "Content-Type: application/json" \
  -d '{
    "query": "query($tr: TimeRangeInput!) { series(labels: {expression: \"host=\\\"server1\\\"\"}, timeRange: $tr, limit: 3) { meta { id endpoint metric labels { entries { key value } } } } }",
    "variables": {
      "tr": {
        "start": "2026-03-19T00:00:00Z",
        "end": "2026-03-20T23:59:59Z"
      }
    }
  }'
```

**标签过滤语法：**

| 操作符 | 说明 | 示例 |
|--------|------|------|
| `=` | 精确匹配 | `host="server1"` |
| `!=` | 不等于 | `env!="localhost"` |
| `=~` | 正则匹配 | `region=~"us-.*"` |
| `!~` | 正则不匹配 | `region!~"eu-.*"` |
| `AND` | 逻辑与 | `env="prod" AND region="us-east"` |
| `OR` | 逻辑或 | `host="server1" OR host="server2"` |
| `()` | 分组 | `(host="s1" OR host="s2") AND env="prod"` |

---

### 5. 查询时序数据（复杂标签过滤）

```bash
curl -s -X POST http://localhost:8080/graphql \
  -H "Content-Type: application/json" \
  -d '{
    "query": "query($tr: TimeRangeInput!) { series(labels: {expression: \"env=\\\"prod\\\" AND region=\\\"us-east\\\"\"}, timeRange: $tr, limit: 3) { meta { id endpoint metric labels { entries { key value } } } } }",
    "variables": {
      "tr": {
        "start": "2026-03-19T00:00:00Z",
        "end": "2026-03-20T23:59:59Z"
      }
    }
  }'
```

---

### 6. 查询时序数据（正则标签过滤）

```bash
curl -s -X POST http://localhost:8080/graphql \
  -H "Content-Type: application/json" \
  -d '{
    "query": "query($tr: TimeRangeInput!) { series(labels: {expression: \"region=~\\\"us-.*\\\"\"}, timeRange: $tr, limit: 3) { meta { id endpoint metric labels { entries { key value } } } } }",
    "variables": {
      "tr": {
        "start": "2026-03-19T00:00:00Z",
        "end": "2026-03-20T23:59:59Z"
      }
    }
  }'
```

---

### 7. 根据 ID 查询单个时序

查询指定 ID 的时序数据，包含统计信息。

```bash
curl -s -X POST http://localhost:8080/graphql \
  -H "Content-Type: application/json" \
  -d '{
    "query": "query($tr: TimeRangeInput!) { seriesById(id: \"1\", timeRange: $tr) { meta { id endpoint metric labels { entries { key value } } } statistics { min max avg count } points { time value } } }",
    "variables": {
      "tr": {
        "start": "2026-03-19T00:00:00Z",
        "end": "2026-03-20T23:59:59Z"
      }
    }
  }'
```

**响应示例：**
```json
{
  "data": {
    "seriesById": {
      "meta": {
        "id": "1",
        "endpoint": "/api/metrics",
        "metric": "cpu_usage",
        "labels": {"entries": [...]}
      },
      "statistics": {
        "min": 30.04,
        "max": 128.30,
        "avg": 81.69,
        "count": 288
      },
      "points": [...]
    }
  }
}
```

---

### 8. 批量查询多个指标 (seriesMulti)

一次查询多个端点和指标的时序数据。

```bash
curl -s -X POST http://localhost:8080/graphql \
  -H "Content-Type: application/json" \
  -d '{
    "query": "query($tr: TimeRangeInput!) { seriesMulti(endpoints: [\"/api/metrics\"], metrics: [\"cpu_usage\", \"memory_usage\"], timeRange: $tr) { meta { id metric } } }",
    "variables": {
      "tr": {
        "start": "2026-03-19T00:00:00Z",
        "end": "2026-03-20T23:59:59Z"
      }
    }
  }'
```

---

### 9. 批量查询带聚合 (AVG)

按时间间隔聚合数据。

```bash
curl -s -X POST http://localhost:8080/graphql \
  -H "Content-Type: application/json" \
  -d '{
    "query": "query($tr: TimeRangeInput!) { seriesMulti(endpoints: [\"/api/metrics\"], metrics: [\"cpu_usage\"], timeRange: $tr, aggregation: {interval: \"1h\", function: AVG}) { meta { id metric labels { entries { key value } } } aggregatedPoints { time value count } } }",
    "variables": {
      "tr": {
        "start": "2026-03-19T00:00:00Z",
        "end": "2026-03-20T23:59:59Z"
      }
    }
  }'
```

---

### 10-13. 聚合函数

支持的聚合函数：

| 函数 | 说明 |
|------|------|
| `AVG` | 平均值 |
| `MIN` | 最小值 |
| `MAX` | 最大值 |
| `SUM` | 求和 |
| `COUNT` | 计数 |

**MIN 示例：**
```bash
curl -s -X POST http://localhost:8080/graphql \
  -H "Content-Type: application/json" \
  -d '{
    "query": "query($tr: TimeRangeInput!) { seriesMulti(endpoints: [\"/api/metrics\"], metrics: [\"cpu_usage\"], timeRange: $tr, aggregation: {interval: \"6h\", function: MIN}) { meta { id metric } aggregatedPoints { time value } } }",
    "variables": {
      "tr": {
        "start": "2026-03-19T00:00:00Z",
        "end": "2026-03-20T23:59:59Z"
      }
    }
  }'
```

**时间间隔格式：** 支持 PostgreSQL interval 格式
- `1m` - 1 分钟
- `5m` - 5 分钟
- `1h` - 1 小时
- `6h` - 6 小时
- `1d` - 1 天

---

## GraphQL Schema

```graphql
type Query {
  endpoints: [String!]!
  metrics(endpoint: String!): [String!]!
  series(
    endpoint: String
    metric: String
    labels: LabelFilter
    timeRange: TimeRangeInput!
    limit: Int
  ): [Series!]!
  seriesById(id: ID!, timeRange: TimeRangeInput!): Series
  seriesMulti(
    endpoints: [String!]
    metrics: [String!]
    labels: LabelFilter
    timeRange: TimeRangeInput!
    aggregation: AggregationInput
  ): [Series!]!
}

type Series {
  meta: SeriesMeta!
  points: [DataPoint!]!
  aggregatedPoints: [AggregatedPoint!]!
  statistics: SeriesStatistics
}

type SeriesMeta {
  id: ID!
  endpoint: String!
  metric: String!
  labels: Labels!
  labelsHash: String!
  createdAt: Time!
}

type SeriesStatistics {
  min: Float!
  max: Float!
  avg: Float!
  sum: Float!
  count: Int!
}

input TimeRangeInput {
  start: Time!
  end: Time!
}

input AggregationInput {
  interval: String!
  function: AggFunction!
}

enum AggFunction {
  AVG
  MIN
  MAX
  SUM
  COUNT
}
```

---

## 常用查询模板

### 监控面板 - 获取最近1小时数据

```bash
# 获取最近1小时的 CPU 使用率
curl -s -X POST http://localhost:8080/graphql \
  -H "Content-Type: application/json" \
  -d '{
    "query": "query($tr: TimeRangeInput!) { series(endpoint: \"/api/metrics\", metric: \"cpu_usage\", timeRange: $tr) { meta { id labels { entries { key value } } } statistics { max avg } } }",
    "variables": {
      "tr": {
        "start": "'$(date -u -v-1H +%Y-%m-%dT%H:%M:%SZ)'",
        "end": "'$(date -u +%Y-%m-%dT%H:%M:%SZ)'"
      }
    }
  }'
```

### 告警 - 检查阈值

```bash
# 获取指定指标的最大值
curl -s -X POST http://localhost:8080/graphql \
  -H "Content-Type: application/json" \
  -d '{
    "query": "query($tr: TimeRangeInput!) { seriesById(id: \"1\", timeRange: $tr) { meta { metric } statistics { max } } }",
    "variables": {
      "tr": {
        "start": "'$(date -u -v-5M +%Y-%m-%dT%H:%M:%SZ)'",
        "end": "'$(date -u +%Y-%m-%dT%H:%M:%SZ)'"
      }
    }
  }'
```

### 趋势分析 - 小时级聚合

```bash
# 获取24小时内的小时级平均值
curl -s -X POST http://localhost:8080/graphql \
  -H "Content-Type: application/json" \
  -d '{
    "query": "query($tr: TimeRangeInput!) { seriesMulti(endpoints: [\"/api/metrics\"], metrics: [\"cpu_usage\", \"memory_usage\"], timeRange: $tr, aggregation: {interval: \"1h\", function: AVG}) { meta { metric labels { entries { key value } } } aggregatedPoints { time value } } }",
    "variables": {
      "tr": {
        "start": "'$(date -u -v-24H +%Y-%m-%dT%H:%M:%SZ)'",
        "end": "'$(date -u +%Y-%m-%dT%H:%M:%SZ)'"
      }
    }
  }'
```

---

## 错误处理

当查询出错时，响应包含 `errors` 字段：

```json
{
  "errors": [
    {
      "message": "Cannot query field \"endpoints\" on type \"Query\".",
      "locations": [{"line": 1, "column": 3}],
      "extensions": {"code": "GRAPHQL_VALIDATION_FAILED"}
    }
  ],
  "data": null
}
```

常见错误：
- `GRAPHQL_VALIDATION_FAILED` - 查询语法错误
- 数据库连接错误 - 检查 TimescaleDB 是否运行
- 标签过滤语法错误 - 检查标签表达式格式