# MVP 实施架构与交付计划

| 项 | 内容 |
|---|---|
| 文档版本 | v1.0 |
| 日期 | 2026-08-16 |
| 上游文档 | 《数据库AI智能运维平台架构设计文档》v1.1；《Agent执行框架详细设计》v1.1；《统一工具注册表详细设计》v1.1；《Generative UI卡片协议详细设计》v1.1 |
| 定位 | MVP（≈3 个月 · 3-5 人）的实施架构、模块清单、里程碑、验收场景与风险清单 |
| 状态 | 待评审 |

---

## 1. 目标与范围

**一句话目标**：以真实数据跑通平台架构的核心假设——统一查询协议、多 Agent 编排（LangGraph + 四端口）、卡片协议、异步任务闭环——交付可试点的最小内核。

### 1.1 做（MVP 范围）

| 能力 | 内容 |
|---|---|
| 监控大盘 | React 重写，完整复刻 vanilla 原型编辑器（多大盘管理/面板编辑/12列布局/序列级样式/阈值/预览）；接真实旧监控 API |
| chat | 多会话管理 + SSE 流式（六类事件）+ 卡片渲染器注册表（text/metric_chart/data_table/task_progress/diagnosis_report） |
| agent 问数 | 简版 NL2Metric：指标/告警/慢SQL统计/实例元数据四类白名单工具，不做自由 NL2SQL |
| agent 诊断 | LangGraph ReAct + 路由/诊断专家 + 自建工具集（指标异常/会话快照/慢SQL/锁分析）+ 异步任务闭环 |
| 真实接入 | 公司 AI 平台（OpenAI 兼容 + tool calling + 流式）、旧监控 TSDB API（指标代理）、DBaaS 元数据同步、告警同步 Issue 化 |
| 外采诊断 | 仅落地 vendor_agent 适配器规范与 shadow 注册位（工具注册表 §9）；契约到手后实现适配器即接入，不阻塞交付 |

### 1.2 不做（防范围蔓延清单）

鉴权/SSO（无鉴权 + 权限桩）、页面上下文注入与元素选中诊断、自治服务页、知识库问答（说明书 RAG + URL 跳转）、L1/L2 动作、Redis/MQ/对象存储、MCP/CLI/Skills 插件生态。

## 2. 技术栈与部署（摘要）

| 层 | 选型 |
|---|---|
| 前端 | React 18 + TypeScript + Vite + ECharts |
| apiserver | Go（对外 REST + chat SSE 透传 + 内部数据 API + 任务总线调度 + 全部旧系统适配层） |
| Agent 集群 | Python + LangGraph（四端口封装，无状态 ≥2 副本，不直连 DB/不对外暴露） |
| 存储 | PostgreSQL 单库（JSONB 承载半结构化数据） |
| LLM | 公司统一 AI 平台，OpenAI 兼容 + tool calling + 流式；模型名配置化 |
| 部署 | Docker Compose（frontend / apiserver / agentcluster / postgres） |

边界三原则与 Go↔Python 内部契约详见架构文档 §3.4/§3.5；LangGraph 落地映射详见执行框架文档 §14-§16。

## 3. 仓库结构（monorepo）

```
db-copilot/
├── frontend/        # React 18 + TS + Vite + ECharts
├── apiserver/       # Go：REST / SSE 透传 / 内部 API / 任务调度 / 旧系统适配层
├── agentcluster/    # Python + LangGraph：四端口执行框架 + 路由/诊断/问数专家
├── deploy/          # docker-compose.yml、PG schema 迁移、配置样例（.env.example）
└── docs/            # 架构文档集（本目录）
```

## 4. 模块清单与归属

| 模块 | 归属服务 | 核心内容 |
|---|---|---|
| 大盘与查询协议 | frontend + apiserver | 面板模型复刻、fetchSeries/fetchAnnotations 双 Provider（Mock/Api）、指标查询代理 + 缓存 + 白名单 |
| 元数据/告警同步 | apiserver | DBaaS/告警适配器、同步水位、Issue 化 |
| chat 会话 | frontend + apiserver + agentcluster | 多会话 UI、SSE 透传、卡片渲染器注册表、轨迹折叠视图 |
| Agent 执行框架 | agentcluster | LangGraph 四端口封装、AgentDefinition 加载器、护栏与流式事件转换 |
| 诊断专家 + 工具集 | agentcluster + apiserver | 自建诊断工具（含直连通道）、诊断报告卡、追问 |
| 问数专家 | agentcluster + apiserver | 四类白名单工具 NL2Metric |
| 异步任务总线 | apiserver | DIAG_TASK 状态机、轮询调度、wake 回调、任务续聊闭环 |
| 实例接入网关 | apiserver | JDBC 只读直连、审计、超时熔断（安全底线，不降级） |

## 5. 里程碑（3 个月）

### M1（第 1 个月）：数据底座 + 前端大盘

- PG schema 与迁移框架；Go apiserver 骨架（配置/日志/REST 框架）；
- DBaaS 元数据同步适配器 + 告警同步 Issue 化（真实 API 联调）；
- 指标查询代理接真实旧监控 API + 最小指标白名单（架构文档 §4.4）+ 缓存（进程内/PG 表）；
- React 工程搭建；大盘完整复刻（从 vanilla 原型移植交互与数据模型）；ApiProvider 接通真实数据；
- chat UI 骨架：多会话列表 + SSE 通道（先用脚本化假流验证前端渲染与断线重连）；
- **验收**：真实数据大盘可浏览可编辑；chat UI 与 SSE 链路就绪。

### M2（第 2 个月）：Agent 集群 + 诊断闭环

- agentcluster：LangGraph 四端口封装（含 CheckpointSaver 经 Go API）、AgentDefinition 加载器、护栏、流式事件转换；
- 路由专家 + 诊断专家上线；自建诊断工具集（Go 内部数据 API + 实例接入网关只读直连）；
- 卡片渲染器注册表（5 种卡片）；诊断报告卡 + 行级追问；
- 异步任务总线（DIAG_TASK 状态机 + 轮询 + wake 回调）；以 `builtin_metric_deep_scan` 验证任务续聊闭环；
- vendor_agent 适配器规范落地 + shadow 注册位；
- **验收**：北极星场景前半段真实跑通（chat 诊断 → 证据采集 → 报告卡 → 追问）。

### M3（第 3 个月）：问数 + 集成收尾

- 问数专家（四类白名单工具）+ metric_chart/data_table 输出打磨；
- 若外采契约到手：适配器实现 + 真实深度诊断接入验收；未到手则按风险预案保持自建兜底；
- Compose 编排 + 配置样例 + 部署文档；北极星场景全链路串联与体验打磨；
- **验收**：北极星场景全链路 + Compose 一键拉起。

## 6. 北极星验收场景

> 实例变慢 → 大盘发现异常指标 → 打开 chat 发起诊断 → 诊断专家采集真实证据（指标/告警/会话快照/慢SQL）→（异步深度任务：进度卡 → 完成唤醒 → 二次推理交叉验证）→ 诊断报告卡（结论/根因/证据/建议）→ 用户追问。

全链路涉及：统一查询协议 ✚ 多 Agent 编排 ✚ 工具注册表 ✚ 卡片协议 ✚ 异步任务闭环——五项架构核心假设一次验证。

## 7. 风险与行动项

| # | 风险 | 影响 | 缓解 / 行动项 |
|---|---|---|---|
| R1 | 外采诊断契约未拿到 | 深度诊断接入延期 | 适配器抽象（submit/poll/normalize）+ 自建工具兜底 + `builtin_metric_deep_scan` 验证闭环；**行动项：本周内向供应商索取 API 文档** |
| R2 | 网络连通性未确认 | 真实接入无法联调 | **行动项：M1 第 1 周确认部署位置与旧监控/DBaaS/AI 平台/外采的网络策略并打通** |
| R3 | AI 平台模型未定名 | LLM 评估/调参延后 | model_profile 配置化；M1 先用任一 OpenAI 兼容端点开发，模型定了即切 |
| R4 | 大盘完整复刻工作量 | 挤压 chat/卡片开发时间 | 前端专人专注大盘（vanilla 原型逻辑可移植）；若延期，序列级样式/事件标注降级至 M3 尾部 |
| R5 | 实例只读账号批期 | 直连工具（会话/锁）延期 | Probe Executor 先走旧系统 API 通道兜底，工具 Schema 不变 |
| R6 | 无鉴权窗口 | 安全暴露面 | 仅部署内网开发环境；实例网关只读/审计/熔断不降级；对外演示前完成敏感过滤回归 |

## 8. 决策记录（2026-08-16 架构评审）

| # | 决策 | 结论 |
|---|---|---|
| D1 | 后端技术栈 | Go apiserver（入口+存储收口）+ Python agent 集群（LangGraph，无状态） |
| D2 | 前端技术栈 | React 18 + TS + Vite + ECharts；大盘完整复刻原型编辑器 |
| D3 | MVP AI 能力 | chat + 问数（简版 NL2Metric）+ 诊断（自建工具集）；知识库问答后置二期 |
| D4 | MVP 依赖 | 全部真实接入（LLM/旧监控/DBaaS/告警）；外采仅落地适配器规范 |
| D5 | 存储简化 | PostgreSQL 单库；不引入 Redis/MQ/对象存储（接口抽象保留） |
| D6 | 任务总线 | DIAG_TASK 表为唯一事实源；Go 侧调度 + wake 回调 Python 续跑 |
| D7 | MVP 砍掉项 | 鉴权/SSO、页面上下文、自治页、L1/L2 动作、插件生态 |
| D8 | 部署 | Docker Compose 起步，生产 K8s 后置 |
| D9 | 人力与周期 | 3-5 人 × 3 个月（前端 1 + Go 2 + Python 1~2 建议配比） |
