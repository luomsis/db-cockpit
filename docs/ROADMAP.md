# ROADMAP · 功能路线图

| 项 | 内容 |
|---|---|
| 文档版本 | v2.1 |
| 日期 | 2026-08-24 |
| 定位 | 功能路线图：**已实现**（严格对照代码）/ MVP 待办 / 二期 / 北极星验收场景 / 决策记录 |
| 上游文档 | 《数据库AI智能运维平台架构设计文档》v2.4；docs/design/ 各模块设计；docs/contract/ 对接规格 |

> **变更记录**
> - v2.1（2026-08-24）：**元数据域 v2 实施定稿（D16）**——四层精简模型（db_cluster/db_component/db_host + 水位）落地，零关系表、双自引用字段（traffic_upstream_id/replication_upstream_id）串联关系，host 独立成全局表；OB 租户=组件逻辑单元（units 落位到 observer 组件 id）；§6.1.1 重写为 v2 定稿；DDL 001/seed/handler/meta API 同步更新。
> - v2.0（2026-08-22）：由《MVP实施架构与交付计划》改造——里程碑从「周排期」改为按**实现状态**组织（已实现 / 待办 / 二期，对照代码逐项核对）；删除人力配比与任务分派内容（原《MVP任务拆解与分工》整体退役）；新增 D14（数据面白名单表定稿）。
> - v1.3 / v1.2 / v1.1：见 git 历史（依赖规则重构对齐、存储三域分治、数据面拆分）。

---

## 1. 已实现（与代码对应）

### 1.1 前端（frontend/ · React 18 + TS + Vite + ECharts）

| 能力 | 内容 |
|---|---|
| 监控大盘 | 完整编辑器复刻（多大盘管理 / 面板编辑 / 12 列布局 / 序列级样式 / 阈值 / 预览 / 标注）；Mock↔Api 双 Provider |
| 集群管理 | 集群列表（类型过滤）；PG 详情（库 / 复制 / 参数变更历史 / 切换演练 / 重建）；OB 详情（租户 / 租户库 / 参数 / Unit 扩缩容） |
| 实例详情 | 拓扑图、账号管理（授权 / 重置 / 锁定）、性能监控、SQL 诊断（慢SQL + AI 建议）、长事务与阻塞链、会话管理（Kill） |
| 其他页面 | 租户详情、主机列表、报告中心（下载）、告警中心（级别 / 对象类型过滤、排序、分页、问 AI） |
| Chat | 多会话管理、SSE 流式（六事件）、卡片渲染器注册表（text / metric_chart / data_table / task_progress / diagnosis_report）、轨迹折叠、草稿惰性创建（D39）、断线重连重放、侧边抽屉与「问 AI」统一入口 |
| 管理页 | 设置中心（大模型 / 嵌入配置 + 测试连接）、插件中心（MCP / Skills） |
| 降级 | apiserver 不可达时全页面自动回退本地 mock |

### 1.2 apiserver（Go · 控制面）

| 能力 | 内容 |
|---|---|
| chat 全链路 | Go 终结 SSE（会话级常开 / 15s 心跳 / `last_event_id` 重放 / 会话内单调 seq 落库）、`client_request_id` 幂等、追问取消替换、restart 恢复（running→failed）、上游错误码族（unreachable/broken/idle/unexpected_end）；`AGENT_MODE` 双事件源（builtin 内置场景 / upstream 消费 exec 执行流，与 agentcluster-mock 联调通过） |
| 任务总线 | `agent_tasks` 轮询（2s）、progress 事件直发会话总线、done→system_resume 续跑、取消级联（限列写 cancel_requested）、租约观察（`agent/taskbus.go`） |
| 管理面 CRUD | subagents / workflows / model-configs / embedding-configs / mcp-servers / skills（启停 + 审计 + api_key 脱敏） |
| 插件域（D15，2026-08-23 实施） | `tool_definitions`/`config_versions` 表 + 4 个管理端点（触发式 `tools/list` 发现→draft 草案、人工定级状态机、连通性健康标记、列表过滤）；mcp/skill 补 version/status/base_url；stdio 校验拒绝；最小 MCP http 客户端（JSON/SSE 双承载）；版本自增 + 删除级联；演示种子与全量测试（工具注册表 §10.8） |
| 数据面白名单消费（架构 §6.1.2） | 告警聚合（Critical/Major→P1/P2、次数、首触时间）；慢 SQL 指纹聚合（digest 分组、按实例三级解析：名称→host_ip→OB `extensions.units` 链式）；`GET /api/changes`（风险 / 对象 / 执行时间窗闭区间过滤）；`GET /api/meta/clusters[/:id]`（v2 形状：集群→组件成员→主机，含 kind/group_name/双上游字段/extensions.units）；空表自动回退 UI 演示表 |
| 大盘与查询 | 指标 mock 确定性（与前端 mock 算法对拍黄金向量）、dash series / annotations、大盘 CRUD 与导入 |
| 其余 REST | 集群 / 实例 / 租户 / 主机 / 报告 / 参数（pending+历史）/ 账号 / 会话 / 事务 / 慢SQL / SQL 诊断建议 / overview 库类型活计数 |
| 工程基座 | 审计（audit_logs）、种子幂等（主演示 + 白名单 7 表）、单测 + PG 集成测试（marker 自清） |

### 1.3 数据底座

| 内容 | 位置 |
|---|---|
| 元数据域 v2（D16，2026-08-24）：3 域表 + 水位表（`db_cluster`/`db_component`/`db_host`/`db_sync_watermark`），零关系表，双自引用字段（`traffic_upstream_id`/`replication_upstream_id`）串联关系，host 独立成全局表；OB 租户=组件逻辑单元（`extensions.units` 落位到 observer 组件 id） | `deploy/db/001_db_metadata.sql` + GORM AutoMigrate（`apiserver/internal/model/metadata.go`） |
| 白名单 3 表（`alert_raw/change_ticket/slow_query_log`，**lock 不建**） | `deploy/db/002_whitelist.sql` + GORM AutoMigrate |
| 部署 | docker-compose（frontend + apiserver 两容器，复用宿主机 PG 55432） |

### 1.4 联调与契约工件

`contracts/api/` 金样（REST 快照 + exec-turns SSE 金样）；`agentcluster-mock/`（可运行的契约参考实现）；`deploy/db/` 建表基准。

## 2. MVP 待办

| 项 | 说明 | 依据 |
|---|---|---|
| agentcluster（Python） | 真服务：LangGraph 四端口 + 主 agent / 动态 subagent + 内核域直写（现为 mock 顶替） | contract/《Agent集群开发规格》M1-M8 |
| collector 对接 | 既有程序直写白名单表 + 水位（**表结构已定稿**，程序未接入 compose） | 架构 §3.4.1、时序 §6 |
| remote 网关 | 仅 SQL 实例访问：凭证表 / 白名单双保险 / 审计直写 / 熔断（`POST /query` 契约已定） | 时序 §7、§8.1 |
| Issue 域 | ISSUE / ALERT_RECORD / ISSUE_EVENT 状态机（告警**消费端聚合已先行**，fingerprint 与状态流转未做） | 架构 §6.2 |
| `/internal/tools/data` | builtin 工具取数过渡通道（现 501；Probe 三通道路由待实现） | 工具注册表 §8 |
| 插件域 agent 消费侧 | agentcluster 直读 `tool_definitions`/`mcp_server_configs` + `config_versions` 轮询 + toolset 前缀解析 + active∧health=ok 过滤（**apiserver 侧已实现**，D15，工具注册表 §10） | 工具注册表 §10.8 |
| 指标真实接入 | 旧监控 TSDB 代理 + 最小指标白名单 + 缓存（现为确定性 mock） | 架构 §4.4 |
| 北极星全链路 | 大盘真实异常 → chat 诊断 → 真实证据（经 remote）→ 异步任务 → 报告卡 → 追问 | §4 |
| 鉴权 | `AUTH_ENABLED` Bearer 占位已有；OIDC / RBAC / 实例范围授权接入 | 架构 §8 |

## 3. 二期

HITL 中途询问（复用中断-恢复机制）；结构化计划卡；知识库问答（说明书 RAG + open_link）；页面上下文注入与行级问 AI；自治服务页（异步任务中心 / 建议展示 / 手动触发）；L1/L2 动作提议与确认执行；外采 vendor_agent 适配器正式接入；evals 回归（fixtures 回放 + 标注集）；subagent 并行取证；跨会话记忆（实例档案）；会话 fork 重放；工具健康检查与降级标记；OTel 全链路；shadow diff 比对；任务死信 Issue 化；独立插件执行体（CLI 沙箱 / stdio MCP / 插件网关化，D15 预留）；AgentDefinition 发布状态机与环境隔离；自建采集 Agent + 本地时序库（指标数据源切换）；Redis / 对象存储按需引入；SSO。

## 4. 北极星验收场景

> 实例变慢 → 大盘发现异常指标 → 打开 chat 发起诊断 → 诊断专家采集真实证据（指标/告警/会话快照/慢SQL）→（异步深度任务：进度卡 → 完成唤醒 → 二次推理交叉验证）→ 诊断报告卡（结论/根因/证据/建议）→ 用户追问。

全链路涉及：统一查询协议 ✚ 多 Agent 编排 ✚ 工具注册表 ✚ 卡片协议 ✚ 异步任务闭环——五项架构核心假设一次验证。当前 builtin/mock 事件源已可演示全形态，真实数据链路待 §2 各项就绪。

## 5. 风险清单

| # | 风险 | 影响 | 缓解 |
|---|---|---|---|
| R1 | 外采诊断契约未拿到 | 深度诊断接入延期 | 适配器抽象（submit/poll/normalize）+ 自建工具兜底；`builtin_metric_deep_scan` 验证异步闭环 |
| R2 | 网络连通性未确认（部署位置 ↔ 旧监控/DBaaS/AI 平台/实例） | 真实接入无法联调 | 接入前统一确认网络策略并打通 |
| R3 | AI 平台模型未定名 | LLM 调参延后 | model_profile 配置化，任一 OpenAI 兼容端点可开发，定名即切 |
| R4 | 实例只读账号批期 | 直连工具（会话/锁）延期 | Probe Executor 降级通道 c（旧系统 API）兜底，工具 Schema 不变 |
| R5 | 无鉴权窗口 | 安全暴露面 | 仅内网部署；remote 只读 / 审计 / 熔断不降级；对外演示前敏感过滤回归 |

## 6. 决策记录

| # | 决策 | 结论 |
|---|---|---|
| D1 | 后端技术栈 | Go apiserver（入口+存储收口）+ Python agent 集群（LangGraph，无状态） |
| D2 | 前端技术栈 | React 18 + TS + Vite + ECharts；大盘完整复刻原型编辑器 |
| D3 | MVP AI 能力 | chat + 问数（简版 NL2Metric）+ 诊断（自建工具集）；知识库问答后置二期 |
| D4 | MVP 依赖 | 全部真实接入（LLM/旧监控/DBaaS/告警）；外采仅落地适配器规范 |
| D5 | 存储简化 | PostgreSQL 单库；不引入 Redis/MQ/对象存储（接口抽象保留） |
| D6 | 任务总线 | agent_tasks 表为唯一事实源；Go 侧轮询 + system_resume 续跑（v2.0 取代 wake） |
| D7 | MVP 砍掉项 | 鉴权/SSO、页面上下文、自治页、L1/L2 动作、插件生态（均归二期） |
| D8 | 部署 | Docker Compose 起步，生产 K8s 后置 |
| D10 | 数据面拆分（2026-08-20） | collector（既有程序）直写白名单表；remote（Go · 仅 SQL）承接实例直连；控制面/数据面分层（架构 §3.4/§3.4.1） |
| D11 | 部署形态 | 目标五容器（frontend/apiserver/agentcluster/collector/remote）+ 复用宿主机 PostgreSQL；当前两容器已交付 |
| D12 | 表结构边界 | 部分定稿：元数据/告警/变更/慢日志/水位已落地（见 D14）；凭证表、执行审计表、事件类待定 |
| D13 | 存储三域分治（2026-08-21） | agentcluster 直连 PG 读写分域（呈现域只读 / 内核域读写，Go 统一建模，受限角色） |
| D14 | 数据面白名单表定稿（2026-08-22） | 元数据 4 表 + 告警/变更/慢日志 3 表建模落地（`deploy/db/` + GORM）；**lock 不建表**（锁走 remote 实时采集，死锁历史另立事件表）；消费端聚合先行，Issue 化后续；指标表继续 mock |
| D15 | agent 插件域（2026-08-22） | **合并进 apiserver，不新增容器**（仅 http MCP + 无 CLI/stdio → 无运行时可托管；低调用量；暂无权限诉求）：apiserver 管理插件（mcp_server_configs/skill_configs/tool_definitions），agentcluster **PG 表直读**使用（规则②延伸，config_version 轮次边界生效）；工具调用 agent→MCP server 直连（规则）；MCP(http) 插件生态由二期提前 MVP；独立插件执行体（CLI/stdio/网关化）留二期。详见工具注册表 §10 |
| D16 | 元数据域 v2（2026-08-24） | **四层精简模型**（db_cluster/db_component/db_host + 水位），零关系表，双自引用字段（traffic_upstream_id/replication_upstream_id）串联关系；host 独立成全局表（region/AZ/主机集群三级位置唯一存储点）；OB 租户=组件逻辑单元（`extensions.units` 落位到 observer 组件 id，N:M 字段内承载）；三条显式约定（纵向按层渲染/upstream 允许跨集群/规模边界到 broker 层）；extensions 提升规则（高频过滤键提升为列，三条件齐备才评估类型子表）。详见架构 §6.1.1 v2 |
