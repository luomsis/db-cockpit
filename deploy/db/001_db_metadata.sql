-- =====================================================================
-- 001_db_metadata.sql —— 元数据域（架构文档 §6.1）落地：instance_meta 拆分建表
-- =====================================================================
-- 来源：ddl.sql 的 public.instance_meta（公司 ucmdb 导出，46 列扁平结构，一条记录
--       = 1 集群 + 1 实例 + 最多 2 台主机）。
-- 建模：参考 KubeBlocks「Cluster → Component → InstanceSet → Instance」分层思想：
--         db_cluster         集群/实体层      （KubeBlocks Cluster                 / §6.1 CLUSTER）
--         db_instance        逻辑实例层      （KubeBlocks Component + InstanceSet / §6.1 INSTANCE）
--         db_instance_node   物理副本节点层  （KubeBlocks Instance(Pod)           / §6.1 INSTANCE_NODE）
--         db_sync_watermark  同步水位        （§6.1 SYNC_WATERMARK，collector 断点续传）
--       Component 与 InstanceSet 在 ucmdb 扁平数据中合并为 db_instance（一条记录
--       天然是「一个实例 = 一个副本组」）；host_*1/host_*2 成对列拆为节点表多行，
--       支持一主两从、副本集等任意规模拓扑。
-- 库类型兼容：db_type 文本判别符（KubeBlocks serviceKind 思路）+ extensions jsonb
--       承载各类型差异字段，不为每种库单独建表（与现有 GORM Cluster.Type +
--       datatypes.JSON 的做法一致）。
-- 边界：本文件为评审/部署基准，运行时表结构由 apiserver GORM AutoMigrate 统一
--       建模（apiserver/internal/model/metadata.go）；collector 直写（数据面白名单
--       表），apiserver 只读消费。逻辑外键不建硬约束（collector upsert 顺序不保证）。
-- =====================================================================


-- ---------------------------------------------------------------------
-- 图 1: 表 db_cluster —— 集群/实体层
-- 回答「这是哪套库、归谁管、什么环境、什么高可用模式」
-- ---------------------------------------------------------------------
CREATE TABLE db_cluster (
    id                  bigserial PRIMARY KEY,
    name                text NOT NULL,             -- ← instance_meta.entity_name（实体/集群名）
    description         text,                      -- ← instance_meta.chinese_desc（中文描述）
    db_type             text NOT NULL,             -- ← instance_meta.db_type（主库类型；大盘按类型过滤/计数）
    environment         text,                      -- ← instance_meta.environment
    org_code            text,                      -- ← instance_meta.org_code（组织）
    service_user        text,                      -- ← instance_meta.service_user（服务负责人）
    opr_dba             text,                      -- ← instance_meta.opr_dba（主管 DBA）
    opr_dba_ii          text,                      -- ← instance_meta.opr_dba_ii（备选 DBA）
    business_owner      text,                      -- ← instance_meta.business_owner（业务负责人）
    alert_subscriber    text,                      -- ← instance_meta.alert_subscriber（告警订阅人）
    subsys_code         text,                      -- ← instance_meta.subsys_code（子系统）
    source_sys          text,                      -- ← instance_meta.source_sys（来源系统，如 ucmdb）
    ccm_name            text,                      -- ← instance_meta.ccm_name
    le_name             text,                      -- ← instance_meta.le_name
    ha_type             text,                      -- ← instance_meta.ha_type（高可用架构）
    backup_method       text,                      -- ← instance_meta.backup_method（备份方式）
    failover_type       text,                      -- ← instance_meta.failover_type（故障切换方式）
    is_created_by_cloud boolean,                   -- ← instance_meta.is_created_by_cloud（text → bool）
    source_id           text,                      -- 外部集群级唯一 ID（ucmdb 未提供时按 name+org_code 归并）
    created_at          timestamptz,               -- ← instance_meta.created_date
    synced_at           timestamptz,               -- collector 最近同步时间（行级水位）
    extensions          jsonb DEFAULT '{}'::jsonb  -- 各库类型差异扩展（集群级）
);

-- 前端查询支撑：大盘/集群列表按类型+环境过滤、名称搜索
CREATE INDEX idx_db_cluster_db_type     ON db_cluster (db_type);
CREATE INDEX idx_db_cluster_environment ON db_cluster (environment);
CREATE INDEX idx_db_cluster_name        ON db_cluster (name);


-- ---------------------------------------------------------------------
-- 图 2: 表 db_instance —— 逻辑实例层
-- 回答「这是什么库、什么版本、从哪连」；应用连接的是 vip/endpoint，
-- 与所在主机解耦（换机器、主备切换不影响本表身份字段）
-- ---------------------------------------------------------------------
CREATE TABLE db_instance (
    id              bigserial PRIMARY KEY,
    cluster_id      bigint NOT NULL,               -- 逻辑外键 → db_cluster.id（集群详情下钻）
    db_type         text NOT NULL,                 -- ← instance_meta.db_type（实例实际引擎；诊断工具按此路由）
    name            text NOT NULL,                 -- ← instance_meta.instance_name
    version         text,                          -- ← instance_meta.version_detail
    status          text,                          -- ← instance_meta.status
    role            text,                          -- ← instance_meta.default_role/role（组件角色：storage/proxy…）
    character_set   text,                          -- ← instance_meta.character_set
    infra_type      text,                          -- ← instance_meta.infra_type（基础设施类型）
    req_cpu         numeric(10, 2),                -- ← instance_meta.req_cpu
    req_memory_gb   numeric(10, 2),                -- ← instance_meta.req_memory_gb
    req_storage_gb  numeric(10, 2),                -- ← instance_meta.req_storage_gb
    attach_db       text,                          -- ← instance_meta.attach_db（挂载数据库）
    endpoint        text,                          -- ← instance_meta.instance_endpoint（访问端点）
    vip             text,                          -- ← instance_meta.instance_vip（浮动 IP）
    port            int,                           -- ← instance_meta.instance_port
    username        text,                          -- ← instance_meta.user_name（访问账号）
    role_selector   text,                          -- ← instance_meta.default_role（端点路由角色：主/任意副本）
    source_id       text NOT NULL,                 -- ← instance_meta.ins_uuid（外部唯一 ID，collector 幂等去重键）
    created_at      timestamptz,                   -- ← instance_meta.ins_created_date
    updated_at      timestamptz,                   -- ← instance_meta.ins_updated_date
    extensions      jsonb DEFAULT '{}'::jsonb,     -- 各库类型差异字段（实例级，如 OB 的 tenant 模式等）

    CONSTRAINT uq_db_instance_source_id UNIQUE (source_id)
);

CREATE INDEX idx_db_instance_cluster ON db_instance (cluster_id);  -- 集群详情 → 实例列表
CREATE INDEX idx_db_instance_db_type ON db_instance (db_type);    -- 按库类型统计/路由


-- ---------------------------------------------------------------------
-- 图 3: 表 db_instance_node —— 物理副本节点层
-- 回答「这套库跑在哪些主机上、各自什么角色」；
-- instance_meta 的 host_*1/host_*2 成对列拆成 N 行（每台主机一行），
-- 主备切换只改本表 role，实例身份不变
-- ---------------------------------------------------------------------
CREATE TABLE db_instance_node (
    id               bigserial PRIMARY KEY,
    instance_id      bigint NOT NULL,              -- 逻辑外键 → db_instance.id
    ordinal          int NOT NULL DEFAULT 0,       -- 副本序号：host_*1→0、host_*2→1、…（KubeBlocks 实例序号）
    role             text,                         -- ← instance_meta.default_role/role（primary/secondary/arbiter）
    host_name        text,                         -- ← instance_meta.host_name1 / host_name2
    host_ip          text,                         -- ← instance_meta.host_ip1 / host_ip2
    port             int,                          -- 节点端口（ucmdb 仅有实例级端口时取 instance_port）
    host_environment text,                         -- ← instance_meta.host_environment1 / host_environment2
    host_infra_type  text,                         -- ← instance_meta.host_infra_type1 / host_infra_type2
    os_name          text,                         -- ← instance_meta.os_name

    CONSTRAINT uq_db_instance_node_ord UNIQUE (instance_id, ordinal)  -- 兼作 instance_id 前缀查询索引
);

CREATE INDEX idx_db_instance_node_host_ip ON db_instance_node (host_ip);  -- 主机列表 / 按 IP 反查实例


-- ---------------------------------------------------------------------
-- 图 4: 表 db_sync_watermark —— collector 同步水位
-- 按来源系统记录断点游标，支撑幂等 upsert 与断点续传（§6.1 SYNC_WATERMARK）
-- ---------------------------------------------------------------------
CREATE TABLE db_sync_watermark (
    id             bigserial PRIMARY KEY,
    source_sys     text NOT NULL,                  -- 来源系统标识（如 ucmdb / 告警系统 / 日志系统）
    last_synced_at timestamptz NOT NULL DEFAULT now(),
    cursor         jsonb DEFAULT '{}'::jsonb,      -- 断点游标（各来源自定义：时间戳/分页/位点）

    CONSTRAINT uq_db_sync_watermark_source UNIQUE (source_sys)
);


-- =====================================================================
-- 拆分示例：一条主从 MySQL 的 instance_meta 记录
--   db_cluster        1 行：name=pay-db、environment=prod、ha_type=主从、org_code=…
--   db_instance       1 行：name=pay-mysql、db_type=mysql、version=8.0.30、
--                          vip=10.0.0.5、port=3306、source_id=ins_uuid
--   db_instance_node  2 行：ordinal=0, role=primary,   host_ip=10.0.1.11, host_infra_type=物理机
--                          ordinal=1, role=secondary, host_ip=10.0.1.12, host_infra_type=虚拟机
--
-- 与现有模型对照（并存，不删除）：
--   db_cluster / db_instance / db_instance_node 为 §6.1 元数据域（数据面白名单表，
--   collector 直写、apiserver 只读）；现有 GORM Cluster(clusters)/Instance(instances)
--   为 UI 演示模型。后续接入真实数据时由 db_* 表替代演示数据供前端消费。
-- =====================================================================
