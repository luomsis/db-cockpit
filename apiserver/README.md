# db-cockpit apiserver

数据库智能驾驶仓的 Go 后端：为 `frontend/` 全部页面提供 REST + Chat SSE 接口，数据落在本地既有 PostgreSQL 实例的独立库 `db_cockpit`。

## 技术栈

- Go 1.22+ / Gin / GORM
- PostgreSQL（复用本地既有实例，启动时自动建库 `db_cockpit` + AutoMigrate + 空库幂等导入演示种子）
- Chat：内置模拟 agent（移植前端 `mockAgent.ts` 的六事件协议：thought/token/card/progress/done/error），预留 `AGENT_MODE=upstream` 透明代理 Python agentcluster

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
| `AGENT_MODE` | `builtin` | `upstream` 时 `/api/chat/*` 整组反代到 `AGENT_UPSTREAM_URL` |

## API 约定

- 统一包裹 `{code, message, data}`，成功 `code=0`；错误保留 HTTP 4xx/5xx，`code` 镜像错误码（`1001` = MVP 未实现）
- 例外：`GET /healthz` 返回裸 `ok`；Chat SSE 为事件流（`id:` 序号支持 `last_event_id`/`Last-Event-ID` 断线重放）
- 契约金样见仓库根 `contracts/api/`（与服务实测响应一致，前端联调以此为准）
- `/internal/*`（Go↔Python 边界）已按 docs §3.5 注册，MVP 一律返回 501

## 测试

```bash
go test ./...   # 指标确定性（与前端 mock 算法对拍黄金向量）、种子幂等（需本地 PG，不可达自动跳过）
```

## 写操作语义（模拟）

创建库/租户/账号、参数修改（置 pending + 历史）、Kill 会话、切换演练、Unit 扩缩容等均为对 `db_cockpit` 库的真实状态变更；危险操作记录 `audit_logs`。
