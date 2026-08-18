# Generative UI 结构化卡片协议 · 详细设计

| 项 | 内容 |
|---|---|
| 文档版本 | v1.1（MVP 卡片子集共识稿） |
| 上游文档 | 《数据库AI智能运维平台架构设计文档》第 5 / 7 章 |
| 协议版本 | card-protocol/1.0 |
| 状态 | 待评审 |

> **变更记录**
> - v1.1（2026-08-16）：§4 枚举表按 MVP 共识重标（MVP 子集 5 种）；§5.4 Action 卡 MVP 不产出；新增 §6.1 open_link 规范占位（二期知识库问答）；§7 补充 React 渲染器实现约定。
> - v1.0（2026-08-10）：初版。

---

## 1. 设计目标

1. AI（Agent）输出不再是纯文本流，而是**结构化卡片**，前端按 `card_type` 渲染为对应组件；
2. 协议与渲染解耦：新增卡片类型 = 新增 `card_type` 枚举 + 前端注册渲染器，**不改协议本身**；
3. 卡片可交互：携带上下文回注（追问）、下钻跳转、Action 确认等能力；
4. 向前兼容：未知 `card_type` 有降级渲染策略；`version` 字段支持协议演进。

## 2. 传输通道

| 场景 | 通道 |
|---|---|
| 流式回复中的卡片 | SSE 消息流，`event: card`，卡片可分片更新（以 `card_id` 幂等合并） |
| 异步任务进度 | 任务总线推送（WebSocket/SSE），payload 为 `task_progress` 卡 |
| 历史会话回放 | 从诊断档案（DIAG_TURN / DIAG_REPORT）读取已落库卡片 |

## 3. 卡片信封（Envelope）

所有卡片共享统一信封，类型差异全部收敛在 `payload` 内：

```json
{
  "card_id": "card_01J9XYZ...",
  "card_type": "diagnosis_report",
  "protocol_version": "1.0",
  "title": "实例 order-db-01 性能诊断报告",
  "status": "final",
  "created_at": "2026-08-10T14:32:00+08:00",
  "updated_at": "2026-08-10T14:32:05+08:00",
  "source": {
    "session_id": 10231,
    "turn_id": 102310004,
    "agent": "diagnosis_expert",
    "tool_call_id": null
  },
  "context": {
    "cluster_id": 88,
    "instance_id": 512,
    "time_range": { "start": "2026-08-10T13:30:00+08:00", "end": "2026-08-10T14:30:00+08:00" }
  },
  "payload": { },
  "interactions": [ ],
  "fallback_text": "诊断报告：锁等待导致会话堆积，建议关注慢SQL#2231。"
}
```

信封字段定义：

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `card_id` | string | 是 | 全局唯一，流式更新以此幂等合并 |
| `card_type` | enum | 是 | 见第 4 章枚举 |
| `protocol_version` | string | 是 | 协议语义化版本 |
| `title` | string | 是 | 卡片标题（折叠态也展示） |
| `status` | enum | 是 | `streaming`（生成中）/ `final`（定稿）/ `expired`（数据已过期） |
| `source` | object | 是 | 产生溯源：会话/轮次/Agent/工具调用，用于"查看推理轨迹" |
| `context` | object | 否 | 卡片锚定的运维上下文（实例/时间窗），"追问"与"下钻"按钮据此回注 |
| `payload` | object | 是 | 类型特化数据，见各卡片定义 |
| `interactions` | array | 否 | 交互按钮声明，见第 6 章 |
| `fallback_text` | string | 是 | 降级/复制/纯文本场景（如 IM 通知）使用的文字摘要 |

## 4. card_type 枚举

| card_type | 名称 | 典型产生方 | MVP |
|---|---|---|---|
| `text` | 文本卡 | 普通回复的富文本容器 | ✅ |
| `metric_chart` | 图表卡（时序指标图 + 类目分布图） | 指标查询工具 / 问数专家 | ✅ |
| `data_table` | 数据表卡 | 会话快照/慢SQL/告警列表等工具结果 | ✅ |
| `task_progress` | 进度卡 | 异步任务总线（分钟级诊断） | ✅ |
| `diagnosis_report` | 诊断报告卡 | 诊断专家 Agent 汇总输出 | ✅ |
| `issue_card` | Issue 卡 | 告警关联分析 | 二期（MVP 告警以 data_table 呈现） |
| `action_suggestion` | Action 建议卡 | Agent 动作提议 | 协议占位（MVP 不产出，L1 动作后置时启用） |
| `topology` | 拓扑卡 | 拓扑影响分析 | 二期 |

> 扩展规则：枚举只增不改；废弃类型保留渲染器并标记 `deprecated`。

---

## 5. 各卡片 JSON Schema 与渲染约定

以下 Schema 均为 `payload` 部分（draft-07 风格简写）。

### 5.1 diagnosis_report（诊断报告卡）

```json
{
  "type": "object",
  "required": ["summary", "severity", "findings", "suggestions"],
  "properties": {
    "summary":       { "type": "string", "description": "一句话结论" },
    "severity":      { "enum": ["normal", "notice", "warning", "critical"] },
    "root_causes":   {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["hypothesis", "confidence"],
        "properties": {
          "hypothesis":  { "type": "string" },
          "confidence":  { "type": "number", "minimum": 0, "maximum": 1 },
          "evidence_refs": { "type": "array", "items": { "type": "string" }, "description": "指向证据卡card_id" }
        }
      }
    },
    "findings": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["category", "detail"],
        "properties": {
          "category":    { "enum": ["metric_anomaly", "lock", "slow_sql", "session", "capacity", "config", "other"] },
          "detail":      { "type": "string" },
          "metric":      { "type": "string" },
          "evidence_card": { "type": "string", "description": "证据卡片card_id（图表/表格），支持展开联动" }
        }
      }
    },
    "suggestions": {
      "type": "array",
      "items": { "type": "string", "description": "文字建议；可执行建议另出action_suggestion卡" }
    },
    "provider": { "enum": ["builtin", "vendor_agent"], "description": "诊断能力来源，外采报告标注vendor_agent" },
    "external_report_id": { "type": "string", "description": "外采诊断Agent原始报告ID" }
  }
}
```

**渲染约定**
- 顶部色带按 `severity` 着色（normal 绿 / notice 蓝 / warning 橙 / critical 红）；
- `root_causes` 按 `confidence` 降序展示，附置信度进度条；
- `findings[].evidence_card` 渲染为可展开的证据引用，点击就地展开对应图表/表格卡；
- 外采报告（`provider=vendor_agent`）展示来源徽标"专家诊断引擎"，保留 `external_report_id` 供追溯；
- 卡片定稿后不可变（`status=final` 后内容只读），重新诊断生成新卡。

### 5.2 task_progress（进度卡）

```json
{
  "type": "object",
  "required": ["task_id", "task_type", "status"],
  "properties": {
    "task_id":      { "type": "string" },
    "task_type":    { "type": "string", "description": "如 deep_diagnosis" },
    "status":       { "enum": ["pending", "running", "done", "failed", "cancelled"] },
    "progress":     { "type": "integer", "minimum": 0, "maximum": 100, "description": "无法估算时缺省" },
    "stage":        { "type": "string", "description": "当前阶段文案，如：正在采集会话快照" },
    "stages":       {
      "type": "array",
      "description": "已知步骤清单（可选，用于步骤条渲染）",
      "items": {
        "type": "object",
        "required": ["name", "status"],
        "properties": { "name": {"type":"string"}, "status": {"enum":["pending","running","done","failed","skipped"]} }
      }
    },
    "eta_seconds":  { "type": "integer" },
    "started_at":   { "type": "string", "format": "date-time" },
    "result_card_id": { "type": "string", "description": "任务完成后指向结果卡（如diagnosis_report）" },
    "error":        { "type": "string", "description": "status=failed时的失败原因" },
    "cancellable":  { "type": "boolean", "default": false }
  }
}
```

**渲染约定**
- `streaming` 语义：同一 `card_id` 高频推送，前端**合并替换**而非追加；
- 有 `stages` 渲染步骤条，否则渲染进度条 + `stage` 文案；
- `done` 且存在 `result_card_id` 时，卡片折叠为一行"诊断完成"并可点击跳转结果卡；
- `failed` 红色态展示 `error`，并提供"重试"交互（重新提交任务）；
- 分钟级任务必须配进度卡，禁止对话流阻塞式等待。

### 5.3 metric_chart（图表卡）

支持两类图形：**时序图**（随时间变化的指标曲线）与**类目图**（占比/分布，如饼图）。

```json
{
  "type": "object",
  "required": ["chart_type", "metrics", "data"],
  "properties": {
    "chart_type": { "enum": ["line", "bar", "stacked_bar", "area", "heatmap", "pie", "donut", "table_sparkline"],
      "description": "line/bar/stacked_bar/area/heatmap为时序图；pie/donut/bar可为类目图" },
    "metrics": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["name"],
        "properties": {
          "name":       { "type": "string", "description": "统一指标名（指标抽象层命名）" },
          "label":      { "type": "string" },
          "unit":       { "type": "string" },
          "agg":        { "enum": ["avg", "max", "min", "p95", "p99", "sum", "count"] }
        }
      }
    },
    "data": {
      "oneOf": [
        { "type": "object", "required": ["points"],
          "properties": {
            "points": { "type": "array", "items": { "type": "array", "prefixItems": [{"type":"string","format":"date-time"},{"type":"number"}] } }
          }, "description": "内联时序数据：点数 <= 2000 时直接内嵌" },
        { "type": "object", "required": ["categories"],
          "properties": {
            "categories": {
              "type": "array",
              "items": { "type": "object", "required": ["label", "value"], "properties": {
                "label": { "type": "string" }, "value": { "type": "number" },
                "extra": { "type": "object", "description": "附加字段，如table_name/instance_id，供切片追问回注" } } }
            },
            "value_unit": { "type": "string" }
          }, "description": "内联类目数据：pie/donut/分布类bar使用（如表空间占比、告警级别分布、会话状态分布）" },
        { "type": "object", "required": ["data_ref"],
          "properties": { "data_ref": { "type": "string" }, "data_shape": { "enum": ["timeseries", "categories"] }, "granularity": {"type":"string"} },
          "description": "引用模式：大数据量时给查询句柄，前端经指标查询代理二次拉取" }
      ]
    },
    "time_range": { "type": "object", "required": ["start","end"], "properties": {
      "start": {"type":"string","format":"date-time"}, "end": {"type":"string","format":"date-time"} },
      "description": "时序图必填；类目图可选（表示统计时间窗）" },
    "thresholds": {
      "type": "array",
      "items": { "type": "object", "required": ["value"], "properties": {
        "value": {"type":"number"}, "label": {"type":"string"}, "severity": {"enum":["warning","critical"]} } }
    },
    "anomalies": {
      "type": "array",
      "description": "AI标注的异常区间，渲染为高亮带",
      "items": { "type": "object", "required": ["start","end"], "properties": {
        "start": {"type":"string","format":"date-time"}, "end": {"type":"string","format":"date-time"}, "note": {"type":"string"} } }
    },
    "compare_instances": { "type": "array", "items": {"type":"integer"}, "description": "多实例对比曲线" }
  }
}
```

**渲染约定**

通用：
- 指标名必须使用**指标抽象层**的统一命名，前端不做私有指标映射；
- 数据双模式：小数据内联（对话即看），大数据 `data_ref` 懒加载（避免撑爆消息体）；
- 卡片右上角固定"下钻"按钮：携带 `context.instance_id + time_range` 跳转实例详情对应面板；
- 图表为只读展示，编辑/圈选时间窗追问走交互层（见第 6 章 `ask` 交互）。

时序图（line/bar/stacked_bar/area/heatmap）：
- `anomalies` 渲染为半透明高亮区间，悬浮展示 `note`——这是"AI 看图说话"的视觉锚点。

类目图（pie/donut/分布类 bar）：
- `categories` 按 `value` 降序渲染；超过 8 项时尾部合并为"其他"切片；
- 切片标签展示数值 + 百分比；`extra` 字段携带的实体标识（如表名）在 tooltip 中呈现；
- 切片级追问：点击切片触发 `ask` 交互，将 `label + extra` 注入对话（如点击某表空间切片追问"分析这张表的增长趋势"）；
- 典型场景：表空间/数据量占比、告警级别分布、会话状态分布、慢SQL来源分布（问数专家高频输出）。

### 5.4 action_suggestion（Action 建议卡）

```json
{
  "type": "object",
  "required": ["action_id", "action_type", "target", "risk_level", "state"],
  "properties": {
    "action_id":    { "type": "string" },
    "action_type":  { "type": "string", "description": "注册的动作类型，如 kill_session / flush_slow_query_log" },
    "target": {
      "type": "object", "required": ["instance_id"],
      "properties": {
        "instance_id": { "type": "integer" },
        "session_id":  { "type": "string" },
        "extra":       { "type": "object" }
      }
    },
    "params":       { "type": "object", "description": "动作参数（JSON Schema由动作注册表定义）" },
    "risk_level":   { "enum": ["L0", "L1", "L2"] },
    "rationale":    { "type": "string", "description": "Agent给出的执行理由" },
    "expected_impact": { "type": "string", "description": "预期影响说明" },
    "rollback_plan":   { "type": "string", "description": "回滚方案，L1/L2必填" },
    "state":        { "enum": ["proposed", "confirmed", "rejected", "executing", "succeeded", "failed", "ticketed"], "description": "一期仅出现proposed" },
    "approval": {
      "type": "object",
      "properties": {
        "mode":       { "enum": ["one_click", "ticket_flow"] },
        "ticket_id":  { "type": "string", "description": "L2转旧工单系统后回填" }
      }
    }
  }
}
```

**渲染约定**
- **MVP（v1.1）**：不产出 Action 卡（L1 动作后置）；协议与渲染约定保留，二期启用时首个动作类型走 L1 确认链路；
- 按 `risk_level` 着色：L0 灰、L1 橙、L2 红；L2 卡片明确提示"将转工单审批，平台不直接执行"；
- 按钮可见性受 RBAC 控制：无处置权限的用户仅可见卡片内容，不可见确认按钮；
- `state` 流转全程记录到动作审计表（二期实现），卡片随状态更新渲染。

### 5.5 data_table（数据表卡）

```json
{
  "type": "object",
  "required": ["columns", "rows"],
  "properties": {
    "columns": { "type": "array", "items": { "type": "object", "required": ["key","label"], "properties": {
      "key": {"type":"string"}, "label": {"type":"string"},
      "type": {"enum":["string","number","time","duration","sql","status"]},
      "sortable": {"type":"boolean"} } } },
    "rows":     { "type": "array", "maxItems": 200, "description": "超过200行走data_ref分页" },
    "data_ref": { "type": "string", "description": "分页查询句柄（可选）" },
    "total":    { "type": "integer" },
    "row_actions": { "type": "array", "items": {"enum":["ask","drilldown","explain_sql"]} }
  }
}
```

**渲染约定**
- `type=sql` 列使用等宽字体 + 语法高亮 + 点击展开完整文本；
- 行级"追问"：选中行 → 右键/按钮将行数据注入对话上下文（如"解释这条 SQL 为什么慢"）；
- 会话快照/锁等待表格为诊断高频证据卡，必须支持行级追问。

### 5.6 issue_card（Issue 卡）

```json
{
  "type": "object",
  "required": ["issue_id", "issue_no", "severity", "status", "title"],
  "properties": {
    "issue_id":   { "type": "integer" },
    "issue_no":   { "type": "string" },
    "severity":   { "enum": ["P0","P1","P2","P3","P4"] },
    "status":     { "type": "string" },
    "title":      { "type": "string" },
    "instance_id": { "type": "integer" },
    "first_alert_at": { "type": "string", "format": "date-time" },
    "related_alert_count": { "type": "integer" },
    "ai_note":    { "type": "string", "description": "Agent对本Issue与当前诊断的关联说明" }
  }
}
```

**渲染约定**：徽标展示严重级别；点击跳转 Issue 详情；`ai_note` 是诊断上下文中"告警线索"的呈现位。

### 5.7 text（文本卡）

`payload = { "markdown": string }`。富文本回复的默认容器，支持 Markdown + 内联引用其他 `card_id`（语法 `[[card:card_id]]`，前端就地嵌入被引用卡片）。

---

## 6. 交互层（interactions）

信封 `interactions` 数组声明卡片级按钮，统一 Schema：

```json
{
  "type": "object",
  "required": ["id", "label", "kind"],
  "properties": {
    "id":    { "type": "string" },
    "label": { "type": "string" },
    "kind":  { "enum": ["ask", "drilldown", "confirm_action", "cancel_task", "retry_task", "open_link"] },
    "payload": { "type": "object", "description": "kind特化参数" }
  }
}
```

| kind | 行为 |
|---|---|
| `ask` | 将 `context` + `payload.question` 注入当前会话作为新一轮用户输入（追问） |
| `drilldown` | 携带 `context.instance_id + time_range` 路由到实例详情指定面板 |
| `confirm_action` / `cancel_task` / `retry_task` | 调用后端对应接口后以新卡片更新状态 |
| `open_link` | 跳转内部页面或外部系统（三期工单） |

### 6.1 open_link 规范（二期知识库问答占位，v1.1）

```json
{
  "id": "goto_instance_sessions",
  "label": "前往实例详情 · 会话面板",
  "kind": "open_link",
  "payload": { "url": "#/instance/512/sessions?range=1h", "scope": "internal" }
}
```

| 字段 | 约定 |
|---|---|
| `url` | 前端 hash 路由（`#/path?query`）或外链；内部路由 path 由前端页面路由表定义——该路由表同时是二期知识库问答的知识源（架构文档 §11.1） |
| `scope` | `internal`（前端路由跳转，可携带卡片 `context`） / `external`（新窗口，仅白名单域） |
| MVP 状态 | 仅实现 internal 跳转的渲染与分发（卡片"下钻"按钮已在使用）；知识库问答（help_expert）二期消费此交互回答"怎么操作"并给出跳转 URL |

## 7. 渲染器注册表（前端）

```
CardRendererRegistry
  ├── register(card_type, renderer_component, {min_version, deprecated})
  ├── resolve(card_type, protocol_version) → renderer | FallbackRenderer
  └── FallbackRenderer：展示 title + fallback_text + "卡片类型暂不支持"提示
```

约定：
1. 未知 `card_type` **绝不报错白屏**，一律走 FallbackRenderer；
2. `protocol_version` 不兼容时优先降级为 `fallback_text`；
3. 渲染器不允许私有解析 `payload` 之外的信封字段语义（溯源/上下文由框架统一处理）；
4. 每个渲染器提供"查看推理轨迹"入口（读 `source.tool_call_id`）。

**React 实现约定（v1.1，前端为 React 18 + TS + Vite）**：

1. 注册表 = `Map<card_type, React.ComponentType<CardProps>>`，应用启动时集中 `register()`，卡片类型与渲染器一一对应；
2. 统一**卡片容器组件**处理信封语义（折叠态标题、`status=streaming` 合并更新、`interactions` 分发、trace 入口），业务渲染器只消费 `payload`，不解析信封；
3. `card` SSE 事件按 `card_id` 幂等合并进会话状态（React state/store）；`update` 仅在 `status=streaming` 期间生效，`final` 后只读；
4. 图表卡继续用 ECharts 实例渲染，容器负责生命周期与 resize。

## 8. 安全与合规约束

| 约束 | 说明 |
|---|---|
| 权限后置校验 | 卡片渲染的实例数据仍受权限网关约束：跨实例对比卡中不可见实例显示为"无权限"占位 |
| 敏感数据 | SQL 文本中的字面量经脱敏过滤器处理后再入卡；`data_table` 的 SQL 列默认脱敏展示、授权后可展开原文 |
| 卡片不可伪造 | `source.session_id / turn_id` 由服务端填充，Agent 输出中的对应字段被覆盖 |
