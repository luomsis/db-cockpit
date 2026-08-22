# db-cockpit apiserver

数据库智能驾驶仓的 Go 后端：为 `frontend/` 全部页面提供 REST + Chat SSE 接口，数据落在本地既有 PostgreSQL 实例的独立库 `db_cockpit`。

> 设计文档见 `docs/design/`，对接契约见 `docs/contract/` 与 `contracts/`，功能实现进度见 `docs/ROADMAP.md`（文档索引：`docs/README.md`）。

## 技术栈

- Go 1.22+ / Gin / GORM
- PostgreSQL（复用本地既有实例，启动时自动建库 `db_cockpit` + AutoMigrate + 空库幂等导入演示种子）
- Chat：Go 终结 SSE（会话事件总线 + `chat_turn_events` 落库重放 + 轮次状态机 `chat_turns` + `client_request_id` 幂等 + 取消替换）；`AGENT_MODE` 切换事件源——builtin 内置场景（移植前端 `mockAgent.ts` 六事件协议）/ upstream 消费 agentcluster 执行流 `POST /internal/exec/turns`（契约见 docs《交互时序与生命周期》§5.1，联调用 `agentcluster-mock/`）

## 本地运行

```bash
cd apiserver
go run ./cmd/apiserver
# 监听 :8090，依赖本地 PG localhost:55432（postgres-age 容器）
```

配置（环境变量，均有默认值）：

| 变量 | 默认 | 说明 |
|---|---|---|
| `APISERVER_PORT` | `8090` | 监听端口（宿主机 8080 已被占用，勿改回） |
| `DB_DSN` | `host=localhost port=55432 user=graphiti password=graphiti dbname=db_cockpit sslmode=disable` | 容器内改为 `host.docker.internal` |
| `AUTH_ENABLED` | `false` | MVP 免登录；true 时要求 `Authorization: Bearer <token>`（占位，接 OIDC 时替换校验逻辑） |
| `AGENT_MODE` | `builtin` | 事件源选择（Go 始终终结 SSE）：`upstream` 时消费 `AGENT_EXEC_URL` 执行流（agentcluster / agentcluster-mock） |
| `AGENT_EXEC_URL` | `""` | 完整 exec 端点；缺省由 `AGENT_UPSTREAM_URL + /internal/exec/turns` 推导 |

## API 约定

- 统一包裹 `{code, message, data}`，成功 `code=0`；错误保留 HTTP 4xx/5xx，`code` 镜像错误码（`1001` = MVP 未实现）
- 例外：`GET /healthz` 返回裸 `ok`；Chat SSE 为事件流（`id:` 序号支持 `last_event_id`/`Last-Event-ID` 断线重放）
- 契约金样见仓库根 `contracts/api/`（与服务实测响应一致，前端联调以此为准）
- POST /api/chat/sessions 为**惰性初始化**：前端在会话首条消息发送时才调用（新建界面不落库，docs 交互时序 D39）
- `/internal/*`（Go↔agent 边界）已按 docs §3.5 注册，MVP 一律返回 501

### 数据面白名单消费（架构 §6.1.2，表基准 `deploy/db/`）

- `GET /api/alerts`、`GET /api/overview`、`GET /api/clusters/:id/instances/:iid/slow-sqls`：消费 `alert_raw` / `slow_query_log` 白名单表，聚合出前端视图（级别映射、指纹聚合、按实例过滤）；白名单空表时回退 UI 演示表
- `GET /api/changes`：变更工单（`risk_level` / `cluster_id` / `instance_id` / `from,to` 执行时间窗闭区间 / `limit` 过滤，时间参数支持 RFC3339、`2006-01-02 15:04`、`2006-01-02`）
- `GET /api/meta/clusters`（`db_type` / `environment` 过滤 + 实例/节点计数）、`GET /api/meta/clusters/:id`（集群→实例→节点三级下钻，OB 租户 extensions 透出）

## 测试

```bash
go test ./...   # 白名单聚合单测+集成（需本地 PG，不可达自动跳过）、指标确定性（与前端 mock 算法对拍黄金向量）、种子幂等
```

## 写操作语义（模拟）

创建库/租户/账号、参数修改（置 pending + 历史）、Kill 会话、切换演练、Unit 扩缩容等均为对 `db_cockpit` 库的真实状态变更；危险操作记录 `audit_logs`。
