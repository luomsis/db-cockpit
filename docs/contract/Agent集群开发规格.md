# Agent 集群（agentcluster）开发规格

| 项 | 内容 |
|---|---|
| 文档版本 | v2.1 |
| 日期 | 2026-08-22 |
| 上游文档 | 《数据库AI智能运维平台架构设计文档》v2.0；《Agent执行框架详细设计》v2.0；《统一工具注册表详细设计》v1.4；《交互时序与生命周期》v2.0 |
| 定位 | agentcluster（Python）服务的**对接开发规格**：照此实现 agent 服务，可与 apiserver 无缝对接 |
| 参考实现 | `agentcluster-mock/main.py`（可运行的最小契约样例） |
| 实现状态 | **服务未实现**（apiserver 侧契约已就绪，当前以 agentcluster-mock 联调）；实现进度见 docs/ROADMAP.md |

> **变更记录**
> - v2.1（2026-08-22）：文档重组——去除任务分派表述（对接方/周排期）；里程碑节改为模块依赖顺序；头部补实现状态。
> - v2.0（2026-08-21）：依赖规则重构——四条依赖通道（唯一 exec 调用 / PG 表契约 / MCP / 权限注入）；固定专家改为**主 agent + 动态 subagent**（SubagentDef 直读装配 + workflow 两级）；配置拉取 API 退役改 **PG 直读**；任务改为 **agent_tasks 表契约**（wake 退役）；exec 请求新增 `auth_context`，事件新增 `agent` 字段。
> - v1.0（2026-08-21）：首版。

---

## 0. 定位与四条依赖规则

agent 服务是**无状态常驻推理服务**（Python + LangGraph，≥2 副本），架构 = **主 agent（内置编排）+ 动态派发 subagent**（由管理面的 sys_prompt + 工具集 + workflow 定义装配）。

**依赖规则（验收红线，来源：架构文档 v2.0 §3.4）**：

| # | 规则 | 对你的含义 |
|---|---|---|
| ① | **唯一运行时调用** = apiserver→agent 的 exec 执行流 | 你**不调用 apiserver 的任何接口**（tools/data 仅 MCP 未就绪期的过渡通道）；`POST /agentcluster/wake` 已退役 |
| ② | **PG 表契约** | 与 apiserver 的一切共享状态都是表 + 读写权属（§3 五域分治），表结构即契约 |
| ③ | **能力经 MCP** | 工具执行连接 MCP Servers；MCP 未就绪的 builtin 工具暂经 `POST /internal/tools/data` 过渡 |
| ④ | **权限注入** | 权限只在 apiserver 校验一次，`auth_context` 随 exec 请求下发；你按其裁剪工具与实例范围，**不自行鉴权** |

其余硬约束不变：不对外暴露端口；**断连即取消**；**每轮必有终态**（done/error，唯一例外被取消）；同 session 至多一条活跃流；不编序号、不写呈现域表。

---

## 1. 服务对外契约（必须实现）

### 1.1 `POST /internal/exec/turns`（轮次执行流，SSE 响应）

**请求**（JSON，字段名不可改）：

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `turn_id` / `session_id` | string | ✓ | apiserver 预创建；内核域/任务域表以此关联 |
| `user_msg` | string | ✓ | 用户输入原文；system_resume 轮为任务结果注入的系统输入（结果本体从 agent_tasks.result_ref 解析） |
| `config_version` | string | | 本轮快照的配置版本（缓存失效校验用） |
| `kind` | string | | `user`（默认）/ `system_resume`（任务完成续跑的新轮） |
| `resume_of` | string | | 续跑时指向 agent_tasks.id |
| `auth_context` | object | ✓ | **授权上下文（依赖规则④）**：`{user_id, instance_scope[], tool_allowlist[]}`；空数组 = 全量（MVP 权限桩）。你必须在派发 subagent 与调用工具前按它裁剪候选 |

**响应**：`text/event-stream`。事件行 `data: {信封 JSON}\n\n`（**不带序号**）；心跳为注释行 `: ping`（默认 15s；**长任务必须持续心跳**，apiserver 侧 90s 无事件且无心跳即判 `upstream_idle` 断连）。

**六类事件信封**（与 `contracts/api/exec-turns-*.sse` 金样严格一致）：

```json
{"type":"thought","step":1,"tool_name":"builtin_get_metrics","status":"running","agent":"diag_subagent"}
{"type":"token","text_delta":"指标显示 CPU 异常飙升，","agent":"diag_subagent"}
{"type":"card","mode":"create","card":{"card_id":"card_01","card_type":"metric_chart","protocol_version":"1.0","title":"...","status":"final","context":{},"payload":{},"fallback_text":"..."}}
{"type":"progress","task_id":"atask_x","progress":45.0,"stage":"多指标长时窗扫描…"}
{"type":"done","usage":{"prompt_tokens":2100,"completion_tokens":380}}
{"type":"error","code":"budget_exceeded","message":"...","fallback_text":"已基于已获证据给出初步结论…"}
```

- `agent` 字段（thought/token 可选）：当前产出的 subagent 标识（主 agent 汇总阶段填 `main_agent`）——动态装配对用户可见可追溯；
- `progress.task_id` = `agent_tasks.id`；**注意 apiserver 任务轮询器也会以同型事件直发进度**，你不得对同一任务重复推送；
- `card.mode`=create/update，同 `card_id` 由 apiserver 就地合并；`done`/`error` 为终态，发出后关流。

**断连即取消**：客户端断开即中止本轮（asyncio cancel / `request.is_disconnected()`），不需要任何显式 cancel 端点。

### 1.2 `GET /healthz`

`200 {"ok": true, ...}`，容器就绪 ≤30s。

---

## 2. 能力与任务通道

### 2.1 工具执行（依赖规则③）

- **目标形态**：一切工具经 **MCP Server** 执行（每类数据域一个 Server：metrics-mcp / instance-mcp / alert-mcp…），你持有 MCP 连接池（stdio/http），`tools/list` 发现 → 注册表准入后使用；
- **MVP 过渡**：MCP 未就绪的 builtin 工具经 `POST /internal/tools/data`（apiserver 过渡通道，`{tool_name, input}` → 标准化输出；output_schema 已由 Go 校验）；按工具逐个 MCP 化后该通道退役。

### 2.2 异步任务（agent_tasks 表契约，取代 tasks API 与 wake）

**你写入并执行，apiserver 只读轮询**：

| 时机 | 动作 |
|---|---|
| 专家决定异步工具 | `INSERT agent_tasks(pending)`（call_id 幂等防重）→ 本轮收尾 done + 进度卡 |
| 后台 worker | `SELECT ... FOR UPDATE SKIP LOCKED` 认领 pending（写 `claimed_by` + `lease_until` 租约，心跳续期）→ 执行 → `UPDATE status/progress/stage` |
| 完成 | `UPDATE status=done, result_ref=...`（大结果自行存储，表内只放引用） |
| 观察 cancel_requested | apiserver 限列写 `cancel_requested=true`（用户取消轮次级联）→ 你中止执行并置 `cancelled` |
| 续跑 | **无需你发起**：apiserver 轮询检出 done → 创建 system_resume 轮 → 经 exec 流调你（`resume_of` 指向任务，user_msg 含结果引用提示）→ 你从 checkpoint 恢复做二次推理 |

**限列写约定**：`notified` / `cancel_requested` 两列属 apiserver，你不得写；其余列你读写。租约过期（`lease_until` 超时未续期）的 running 任务由你的 worker 重新认领或置 failed。

---

## 3. PG 直连（五域分治，依赖规则②）

连接串 env `PG_DSN`。表由 apiserver GORM AutoMigrate 建，你**只读映射/裸 SQL，不建表不迁移**。

| 域 | 表 | 你的权属 |
|---|---|---|
| 呈现域 | `chat_sessions` / `chat_turns` / `chat_messages` / `chat_turn_events` | **只读**（上下文装配；chat_turns 有 kind/resume_of/config_version） |
| 内核域 | `tool_calls`（call_id 幂等）/ `llm_calls` / `agent_checkpoints`（session+checkpoint 唯一）/ `agent_context_summaries` | **读写** |
| 任务域 | `agent_tasks` | **读写**（除 notified/cancel_requested 两列） |
| 管理面 | `subagent_defs` / `workflow_defs` / `model_configs` / `mcp_server_configs` / `skill_configs` | **只读**（运行时装配输入；model_configs 含明文 api_key，**严禁写日志**；缓存 + config_version 校验，变更轮次边界生效） |
| 其余表 | 审计/大盘/元数据等 | 不可见（目标形态独立 PG 角色） |

---

## 4. 主 agent + 动态 subagent 装配

### 4.1 主 agent（常驻内置，编排逻辑不配置化）

exec 请求到达 → 装配上下文（近 N 轮原文 + 滚动摘要 + 命中 Skill）→ 意图识别 → **从缓存的 active SubagentDef 中选目标**（routing_hints + 意义匹配）→ 按 `auth_context` 裁剪其 toolset → 派发（context_pack：实例上下文/时间窗/目标/前轮摘要）→ 汇总回报（此阶段事件 agent=main_agent）。同 session 单 subagent 活跃；跨 subagent 协作经主 agent 串行中转。

### 4.2 SubagentDef（管理面实体，apiserver `/api/subagents` CRUD，你直读）

```yaml
subagent_id: diag_subagent     # 稳定标识（路由依据）
version: 3                     # 版本化；active | shadow | deprecated
sys_prompt: "你是数据库性能诊断专家，擅长……"
toolset: ["metrics-mcp.*", "instance-mcp.sessions,slow_sql"]   # 工具集绑定
workflow_ref: wf_diag_v2       # 可空：定制化工作流
model_profile: {model: "...", params: {...}}                   # 引用 model_configs
budget: {max_steps: 20, turn_timeout_s: 300, ...}
output_cards: [diagnosis_report, metric_chart]
routing_hints: ["诊断", "变慢", "根因"]
```

### 4.3 workflow 两级

- **L1_prompt**（MVP）：`workflow_defs.definition` 为结构化步骤清单（检查单/强制步骤/输出要求），渲染进 sys_prompt；
- **L2_graph**（二期）：图 DSL（节点/边/条件分支）编译为 LangGraph 强约束执行图；
- 同一 workflow_id 升级 level 不需要改 subagent 定义。

### 4.4 装配流程

启动拉全量 SubagentDef/WorkflowDef/配置 → 编译缓存（config_version）→ 周期校验版本，变更**轮次边界生效**（进行中轮次跑完旧版本，轮内留痕所用版本）。加载失败的 subagent 标 unavailable（主 agent 降级自答并声明能力受限）。

---

## 5. 功能模块（M1-M8，按开发顺序；验收均可执行）

| 模块 | 职责 | 验收要点 |
|---|---|---|
| **M1 运行时基座** | FastAPI 入口（exec + healthz）；主 agent 循环；astream_events→六事件转换（含 agent 字段）；护栏预算（步数/工具数/时长/token/同参循环熔断）；断连即取消 | 事件序列对 `exec-turns-diagnosis.sse` 全序一致；护栏触发即 error；手动断连执行中止；healthz ≤30s |
| **M2 上下文管理** | 装配（近 N 轮 + 摘要 + Skill）；>8 轮生成滚动摘要直写 `agent_context_summaries` | 9 轮会话第 10 轮装配只含近 8 轮 + 摘要；摘要表写入正确 |
| **M3 CheckpointSaver** | 直写 `agent_checkpoints`（thread=session_id）；resume 恢复 | 中断后续跑状态逐项一致；多副本不串扰 |
| **M4 轨迹/计量** | `tool_calls` / `llm_calls` 直写 | 单轮 ≥3 工具调用行数与状态正确；token 计量自洽 |
| **M5 工具执行** | ToolPort 候选解析（toolset + auth_context 裁剪）；MCP 连接池 + tools/list 发现；MVP 经 tools/data 过渡 | auth_context 裁剪生效（越权工具不出现）；MCP 发现→准入→调用链路通 |
| **M6 异步任务** | agent_tasks 写入/认领/租约/推进度/取消观察（§2.2） | call_id 幂等；进度变化被 apiserver 轮询转发（观察 chat_turn_events）；done 后收到 system_resume exec |
| **M7 动态 subagent 装配** | SubagentDef/WorkflowDef 直读缓存 + 版本热载 + L1 workflow 渲染 + 主 agent 路由派发 | 管理页改 sys_prompt 后新一轮生效；路由准确率 ≥80%（20 问抽检）；shadow 双跑比对 |
| **M8 卡片生成器** | card-protocol/1.0；output_cards 映射；schema 校验失败一次修复再降级 fallback_text | 5 种卡片信封合法；非法输出被修复或降级 |

---

## 6. 工程落地

```
agentcluster/
├── app/
│   ├── main.py        # FastAPI：/internal/exec/turns + /healthz
│   ├── main_agent/    # 主 agent 编排（意图→选 subagent→派发→汇总）
│   ├── subagent/      # SubagentDef 加载/装配/热载 + workflow 渲染（M7）
│   ├── context/       # 上下文装配 + 摘要（M2）
│   ├── checkpointer.py # PG CheckpointSaver（M3）
│   ├── trace.py       # tool_calls / llm_calls 直写（M4）
│   ├── tools/         # ToolPort：MCP 连接池 + tools/data 过渡（M5）
│   ├── taskport.py    # agent_tasks 写入/认领/租约（M6）
│   └── cards/         # 卡片生成器（M8）
├── tests/
├── requirements.txt   # fastapi uvicorn langgraph langchain-openai sqlalchemy psycopg pydantic
└── Dockerfile
```

| 环境变量 | 默认 | 说明 |
|---|---|---|
| `PORT` | `8000` | 服务端口 |
| `PG_DSN` | `host=localhost port=55432 ... dbname=db_cockpit` | 五域分治直连（§3） |
| `AI_BASE_URL` / `AI_API_KEY` / `AI_MODEL` | — | 公司 AI 平台（OpenAI 兼容） |
| `APISERVER_URL` | `http://apiserver:8090` | 仅 tools/data 过渡期使用 |
| `HEARTBEAT_SECONDS` | `15` | `: ping` 间隔 |
| `BUDGET_*` | — | 护栏预算参数 |

联调：`AGENT_MODE=upstream AGENT_UPSTREAM_URL=http://localhost:8000` 起 apiserver；先对金样 curl 自检。

---

## 7. 无缝对接 checklist

| # | 项 | 依据 |
|---|---|---|
| 1 | exec 请求字段（含 `auth_context`）与 §1.1 一致 | `exec-turns-request.json` |
| 2 | 六类事件键名与金样逐字一致（含 `agent`、`fallback_text`） | `exec-turns-*.sse` |
| 3 | 事件 `data:` 行 + `: ping` 心跳 + 不带序号 | §1.1 |
| 4 | 每轮 done/error 终态；error 带 fallback_text | 交互时序 §3.2 |
| 5 | 断连即取消 | §1.1 |
| 6 | 不写呈现域；不写 agent_tasks 的 notified/cancel_requested | §3 权属 |
| 7 | agent_tasks 写入/认领/租约语义正确 | §2.2 |
| 8 | auth_context 裁剪工具与实例范围 | 依赖规则④ |
| 9 | 不调用 apiserver 任何接口（tools/data 过渡除外） | 依赖规则① |
| 10 | 对金样回放一致（集成验收标准） | `contracts/api/exec-turns-*.sse` |

## 8. 开发顺序（模块依赖）

契约金样自检（对 `contracts/api/exec-turns-*` curl 回放）→ M1-M4 运行时基座 → M5/M6/M8 工具执行 + 任务闭环 + 卡片生成 → M7 动态 subagent 装配。集成验收：apiserver 切 `AGENT_MODE=upstream`（事件源从 builtin/mock 切到本服务），双端对同一份金样回放结果一致。
