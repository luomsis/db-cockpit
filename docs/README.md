# docs/ 文档索引

文档分三类：**设计类**（架构与模块设计，描述目标态）、**约定与契约类**（跨组件对接的合同与基准）、**路线图**（已实现功能与待办）。各文档头部的「实现状态」标明与代码的对应关系；功能级实现进度统一见 [ROADMAP.md](ROADMAP.md)。

## 一、设计类（design/）

| 文档 | 一句话定位 | 实现状态 |
|---|---|---|
| [数据库AI智能运维平台架构设计文档](design/数据库AI智能运维平台架构设计文档.md) | 总体架构：分层、控制面/数据面、存储三域分治、核心数据模型（§6）、分期路线 | 总体定稿；已实现部分见 §6.1.1/§6.1.2 注记与 ROADMAP |
| [交互时序与生命周期](design/交互时序与生命周期.md) | chat 全生命周期 / 插件 / 任务 / collector / remote 五组时序与不变量 | §2-§5 已实现（apiserver + agentcluster-mock 联调）；§6/§7（collector/remote）设计态 |
| [Agent执行框架详细设计](design/Agent执行框架详细设计.md) | AI 层运行时：四端口、ReAct 循环、护栏、轨迹与 LangGraph 落地映射 | 设计定稿；运行时待 agentcluster 落地（当前 builtin/mock 事件源） |
| [统一工具注册表详细设计](design/统一工具注册表详细设计.md) | Tool Schema、路由/风险分级、MVP 工具清单、vendor_agent 适配器规范 | 设计定稿；**插件域（apiserver 侧）已实施**（§10：注册表/发现/定级/健康，D15）；agent 消费侧待 agentcluster |
| [Generative UI卡片协议详细设计](design/Generative%20UI卡片协议详细设计.md) | card-protocol/1.0：卡片信封、5 种 MVP 卡片、渲染器注册表约定 | 协议定稿；前端渲染器已实现（ChatPanel 卡片流），Python 侧生成器待 agentcluster |

## 二、约定与契约类（contract/ 与仓库契约工件）

| 文档 / 目录 | 一句话定位 | 实现状态 |
|---|---|---|
| [contract/Agent集群开发规格](contract/Agent集群开发规格.md) | agentcluster（Python）服务对接合同：exec 契约、PG 五域分治、subagent 装配、M1-M8 模块 | 规格定稿；服务未实现，`agentcluster-mock/` 为契约参考实现 |
| `contracts/api/`（仓库根） | API 金样：各端点实测响应快照，前后端联调与回归基线 | 与实现同步维护（见 `contracts/README.md`） |
| `deploy/db/`（仓库根） | 建表基准 SQL：001 元数据域、002 数据面白名单（告警/变更/慢日志） | 与 GORM AutoMigrate 一致；运行时以后者为准 |
| `apiserver/README.md` · `deploy/README-部署.md` | 模块 README：接口约定、本地运行、部署与迁移 | 随代码更新 |

## 三、路线图

| 文档 | 一句话定位 |
|---|---|
| [ROADMAP.md](ROADMAP.md) | 功能路线图：**已实现**（严格对照代码）/ MVP 待办 / 二期 / 北极星验收场景 / 决策记录 |

## 历史文档说明

- 原《MVP任务拆解与分工》（负责人/周排期的任务分派）已删除；其契约优先、集成关卡的实质内容并入 [ROADMAP.md](ROADMAP.md) 与各契约文档。
- 原《MVP实施架构与交付计划》改造为 [ROADMAP.md](ROADMAP.md)（实施架构部分由架构文档承载，里程碑改按实现状态组织）。
