# contracts/ — API 契约金样（fixtures-first）

每个 JSON 文件是 apiserver 对应端点的**实测响应快照**，作为前后端联调契约与回归基线：

- 前端按此形状解包（`{code, message, data}` 统一包裹）
- 修改任一端点响应结构时，须重新抓取快照并同步更新前端类型
- 抓取方式：apiserver 本地起服后 `curl` 落盘（见各文件名对应的路径）

| 文件 | 端点 |
|---|---|
| overview.json | GET /api/overview（dbTypes 活计数 + 告警/慢SQL 白名单聚合视图） |
| clusters.json | GET /api/clusters |
| cluster-pg.json | GET /api/clusters/c1 |
| cluster-ob.json | GET /api/clusters/c3 |
| instance.json | GET /api/clusters/c1/instances/in-1e7f3 |
| tenant.json | GET /api/tenants/t-trade |
| hosts.json | GET /api/hosts |
| alerts.json | GET /api/alerts（alert_raw 聚合：对象+标题+级别 → 次数/首触时间，Critical/Major→P1/P2） |
| changes.json | GET /api/changes?limit=5（变更工单，支持 risk_level/cluster_id/时间窗过滤） |
| meta-clusters.json | GET /api/meta/clusters（元数据域集群列表 + 实例/节点计数） |
| meta-cluster.json | GET /api/meta/clusters/7（集群→实例→节点三级下钻，OB 租户=实例） |
| reports.json | GET /api/clusters/c1/reports |
| metrics.json | GET /api/metrics |
| dash-series.json | GET /api/dash/series?metric=cpu&range=24h&agg=max |
| dash-annotations.json | GET /api/dash/annotations?range=24h |
| dashboards.json | GET /api/dashboards |
| chat-sessions.json | GET /api/chat/sessions |
| tool-definitions.json | GET /api/tool-definitions（插件域注册表，server_id/status/category 过滤，响应含 configVersion） |
| error-404.json | 404 错误体样例 |

注意：
- Chat SSE 事件流不走包裹协议，事件形状见 `frontend/src/lib/types.ts` 的 `AgentEvent`（六事件）；执行流金样见 `exec-turns-*.sse`（apiserver↔agentcluster 契约）。
- 数据面白名单表结构基准见 `deploy/db/`；架构语义见 docs/design/《数据库AI智能运维平台架构设计文档》§6.1。
