# 前端 Mock 版 · Docker 部署与内网迁移指南

| 项 | 内容 |
|---|---|
| 适用版本 | db-cockpit-frontend **v0.1.0**（linux/amd64） |
| 服务范围 | 仅前端（mock 数据，离线可用，无外部依赖） |
| 相关文件 | `docker-compose.yml`（仓库根目录）、`frontend/Dockerfile`、`frontend/nginx.conf` |

---

## 一、本机构建与验证（开发机，含 arm64 Mac）

```bash
cd /path/to/db-cockpit
docker compose up -d --build     # 构建并启动 → http://localhost:8080
docker image inspect db-cockpit-frontend:v0.1.0 --format '{{.Os}}/{{.Architecture}}'
# 期望输出：linux/amd64
```

说明：Dockerfile 两个构建阶段均固定 `--platform=linux/amd64`，compose 中构建与运行
platform 也已锁定——在任何架构的机器上构建，产物都是 amd64（前端无原生模块，
arm64 Mac 上交叉构建安全）。

## 二、迁移到内网 Linux（三步）

### 1. 导出镜像（本机）

```bash
mkdir -p deploy
docker save db-cockpit-frontend:v0.1.0 | gzip > deploy/db-cockpit-frontend-amd64-v0.1.0.tar.gz
```

### 2. 拷贝两个文件到内网机器

- `deploy/db-cockpit-frontend-amd64-v0.1.0.tar.gz`（镜像）
- `docker-compose.yml`（编排文件）

### 3. 内网启动（只需 Docker，无需 node/npm）

```bash
docker load < db-cockpit-frontend-amd64-v0.1.0.tar.gz
docker compose up -d                       # 使用已加载镜像，不会触发构建
# 访问：http://<内网机器IP>:8080
```

> 内网 Linux 需为 x86_64（amd64）架构；`docker compose` 为 v2 语法
> （`docker-compose` v1 用户执行 `docker-compose up -d` 等价）。

## 三、常见操作

```bash
docker compose ps                          # 查看状态
docker compose logs -f frontend            # 查看日志
docker compose down                        # 停止并移除容器
docker compose up -d --build               # 代码更新后重新构建启动
```

## 四、apiserver（已接线）

Go apiserver（`apiserver/`，Gin + GORM）已随 compose 一并部署，前端全部页面走真实接口：

- **存储**：复用宿主机既有 PostgreSQL（postgres-age 容器，宿主机 `55432`，账号 `graphiti/graphiti`），独立建库 `db_cockpit`，启动时自动建库 + 建表 + 幂等导入演示种子；**不新增 postgres 服务**。
- **容器网络**：apiserver 容器经 `host.docker.internal:55432` 访问宿主机 PG（compose 已配 `extra_hosts`）。
- **端口**：容器间走 `apiserver:8090`（宿主机 8080 已被其它服务占用）；nginx `/api/` 反代已启用（SSE 不缓冲）。
- **降级**：apiserver 不可达时前端自动回退本地 mock，页面仍可演示。

常用验证：

```bash
curl http://localhost:8848/api/overview          # 经 nginx 反代（compose 部署）
curl http://localhost:8090/healthz               # 本地直连 apiserver
docker compose logs -f apiserver
```

接口契约金样见仓库根 `contracts/api/`；apiserver 细节见 `apiserver/README.md`；建表基准见 `deploy/db/`（001 元数据域、002 数据面白名单——运行时以 apiserver GORM AutoMigrate 为准）。

## 五、后续演进（agentcluster / collector / remote）

其余三个后端组件就绪后（目标形态：**五容器 + 复用宿主机 PG**，组件清单见架构文档 §3.4.1）：

1. **agentcluster（Python · LangGraph）**：追加服务后 apiserver 切 `AGENT_MODE=upstream`——v1.3 起为**事件源切换**：Go 经 `POST /internal/exec/turns` 消费 Python 执行流并在 Go 终结 SSE（不再整组反代，前端零改动）；`/internal/*` 从 501 stub 换成真实实现；
2. **collector（既有独立程序 · 数据面）**：追加服务，定时拉取 DBaaS/告警/日志并直写 PG 白名单表（六类；**表结构已定稿**——元数据 4 表 + 告警/变更/慢日志/水位，基准 SQL 见 `deploy/db/`，详见架构文档 §6.1.1/§6.1.2；事件类待定，lock 不建表）；apiserver 定时消费告警原始表做 Issue 化；
3. **remote（Go · 数据面 ×1）**：追加服务，承接 Probe「实例直连」通道——仅 SQL、按需建连即用即毁、SQL 白名单双保险（账号 + 语句解析）、执行审计直写 PG、实例熔断。

时序与契约详见 docs《交互时序与生命周期》§6-§8。
