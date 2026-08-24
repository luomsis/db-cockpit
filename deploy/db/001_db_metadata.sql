-- =====================================================================
-- 001_db_metadata.sql —— 元数据域 v2（架构文档 §6.1.1 v2 · D16 定稿）
-- =====================================================================
-- 来源：ddl.sql 的 public.instance_meta（ucmdb 导出，46 列扁平结构）。
-- v2 模型（3 域表 + 水位表，零关系表）：
--   db_cluster         集群层（业务归属 + 环境HA + 服务入口端点）
--   db_component       组件成员统一表（行 = 成员/逻辑单元：引擎成员、代理进程、租户）
--   db_host            独立全局主机表（位置三级 region/AZ/主机集群 唯一存储点）
--   db_sync_watermark  同步水位
-- 关系表达（D16，双自引用字段，无任何关系表）：
--   traffic_upstream_id      数据流上游（纵向·实线）：存储成员 → 其前面的 proxy/access 组件
--   replication_upstream_id  复制上游（横向·虚线）：备 → 主/级联；Paxos/多主置空按 zone+role 渲染
--   租户 N:M 落位            extensions.units（[{instance_id → db_component.id, zone}]）
-- 三条显式约定（v2 定稿）：
--   ① 纵向按层渲染：数据流默认按 kind 层级画组间连线，成员级 traffic 字段为可选精化
--   ② upstream 允许跨集群指向（component id 全局唯一——支持 HBase→HDFS 依赖与跨机房 DR 复制）
--   ③ 规模边界：千级分区级实体（Kafka partition/单 shard 行）不建组件行，建模到 broker/节点/代理层
-- extensions 提升规则：某键成为高频过滤/强约束需求时提升为显式列（character_set 先例），
--   而非类型子表；「独立生命周期 + 跨行 JOIN + 报表级聚合」三条件齐备才评估类型子表。
-- 迁移映射（v1 → v2）：旧 db_instance（逻辑服务）→ cluster（端点）+ db_component（成员）；
--   旧 db_instance_node → db_host（host 字段）+ db_component（副本语义）；「sys 租户挂节点」
--   约定废止（成员挂组件+主机）。v1 环境需 DROP db_instance / db_instance_node 后重建。
-- 边界：运行时表结构由 apiserver GORM AutoMigrate 统一建模（model/metadata.go）；本文件为
--   评审/部署基准；collector 直写（数据面白名单表），apiserver 只读消费；逻辑外键不建硬约束。
-- =====================================================================


-- ---------------------------------------------------------------------
-- 图 1: 表 db_cluster —— 集群层（业务归属 + 服务入口）
-- ---------------------------------------------------------------------
CREATE TABLE db_cluster (
    id                  bigserial PRIMARY KEY,
    name                text NOT NULL,             -- ← instance_meta.entity_name（实体/集群名）
    description         text,                      -- ← instance_meta.chinese_desc
    db_type             text NOT NULL,             -- ← instance_meta.db_type（主库类型）
    environment         text,                      -- ← instance_meta.environment
    org_code            text,                      -- ← instance_meta.org_code（组织）
    service_user        text,                      -- ← instance_meta.service_user（服务负责人）
    opr_dba             text,                      -- ← instance_meta.opr_dba（主管 DBA）
    opr_dba_ii          text,                      -- ← instance_meta.opr_dba_ii（备选 DBA）
    business_owner      text,                      -- ← instance_meta.business_owner
    alert_subscriber    text,                      -- ← instance_meta.alert_subscriber
    subsys_code         text,                      -- ← instance_meta.subsys_code（子系统）
    source_sys          text,                      -- ← instance_meta.source_sys（来源系统）
    ccm_name            text,                      -- ← instance_meta.ccm_name
    le_name             text,                      -- ← instance_meta.le_name
    ha_type             text,                      -- ← instance_meta.ha_type（高可用架构）
    backup_method       text,                      -- ← instance_meta.backup_method
    failover_type       text,                      -- ← instance_meta.failover_type
    is_created_by_cloud boolean,                   -- ← instance_meta.is_created_by_cloud
    source_id           text,                      -- 外部集群级唯一 ID（去重）
    -- 服务入口（v2 端点上移：入口属于集群/服务，不属于单个成员）
    endpoint            text,                      -- ← instance_meta.instance_endpoint（访问端点）
    vip                 text,                      -- ← instance_meta.instance_vip（浮动 IP）
    port                int,                       -- ← instance_meta.instance_port
    username            text,                      -- ← instance_meta.user_name（访问账号）
    role_selector       text,                      -- ← instance_meta.default_role（入口路由角色：primary/any）
    created_at          timestamptz,               -- ← instance_meta.created_date
    synced_at           timestamptz,               -- collector 最近同步时间（行级水位）
    extensions          jsonb DEFAULT '{}'::jsonb  -- 各库类型差异扩展（集群级）
);

CREATE INDEX idx_db_cluster_db_type     ON db_cluster (db_type);
CREATE INDEX idx_db_cluster_environment ON db_cluster (environment);
CREATE INDEX idx_db_cluster_name        ON db_cluster (name);


-- ---------------------------------------------------------------------
-- 图 2: 表 db_component —— 组件成员统一表（v2 核心 · D16）
-- 一行 = 集群的一个成员/逻辑单元：引擎成员（pg 节点/observer）、代理成员（obproxy/mongos/
-- haproxy）、租户逻辑单元（OB tenant）。关系零关系表，双自引用字段串联（见文件头）。
-- ---------------------------------------------------------------------
CREATE TABLE db_component (
    id                     bigserial PRIMARY KEY,
    cluster_id             bigint NOT NULL,           -- 逻辑外键 → db_cluster.id
    name                   text NOT NULL,             -- ← instance_meta.instance_name（成员命名沿用）
    kind                   text NOT NULL,             -- storage/proxy/compute/tenant/access/arbiter
    group_name             text,                      -- 分组：shard-1 / ZONE1 / config…（可空）
    role                   text,                      -- primary/secondary/observer/arbiter/active…
    version                text,                      -- ← instance_meta.version_detail（成员级，滚动升级可不同）
    status                text,                       -- ok/warn/err
    port                   int,                       -- 成员进程端口
    host_ip                text,                      -- 逻辑外键 → db_host.host_ip（逻辑单元可空）
    traffic_upstream_id    bigint,                    -- 数据流上游（纵向）→ db_component.id
    replication_upstream_id bigint,                   -- 复制上游（横向）→ db_component.id（备→主/级联）
    extensions             jsonb DEFAULT '{}'::jsonb, -- 租户 mode/unit/whitelist/units 落位、delay_ms、paxos
    source_id              text,                      -- 外部成员级唯一 ID（去重；原 ins_uuid 语义）
    created_at             timestamptz,               -- ← instance_meta.ins_created_date
    updated_at             timestamptz                -- ← instance_meta.ins_updated_date
);

CREATE INDEX idx_db_component_cluster ON db_component (cluster_id);  -- 集群 → 成员下钻
CREATE INDEX idx_db_component_host    ON db_component (host_ip);    -- 主机 → 成员反查


-- ---------------------------------------------------------------------
-- 图 3: 表 db_host —— 独立全局主机表（D16）
-- 位置三级（region/AZ/主机集群）唯一存储点；物理拓扑按三列分组；主机跨集群共享。
-- ---------------------------------------------------------------------
CREATE TABLE db_host (
    host_ip          text PRIMARY KEY,             -- ← instance_meta.host_ip1 / host_ip2（全局唯一）
    host_name        text,                         -- ← instance_meta.host_name1 / host_name2
    region           text,                         -- 三级位置：region
    az               text,                         -- 三级位置：可用区
    host_cluster     text,                         -- 三级位置：主机集群/机房分组
    os_name          text,                         -- ← instance_meta.os_name
    host_infra_type  text,                         -- ← instance_meta.host_infra_type1/2（物理机/虚拟机）
    host_environment text,                         -- ← instance_meta.host_environment1/2
    status           text,
    extensions       jsonb DEFAULT '{}'::jsonb
);

CREATE INDEX idx_db_host_az ON db_host (az);
CREATE INDEX idx_db_host_hc ON db_host (host_cluster);


-- ---------------------------------------------------------------------
-- 图 4: 表 db_sync_watermark —— collector 同步水位（不变）
-- ---------------------------------------------------------------------
CREATE TABLE db_sync_watermark (
    id             bigserial PRIMARY KEY,
    source_sys     text NOT NULL,
    last_synced_at timestamptz NOT NULL DEFAULT now(),
    cursor         jsonb DEFAULT '{}'::jsonb,
    CONSTRAINT uq_db_sync_watermark_source UNIQUE (source_sys)
);


-- =====================================================================
-- 多库类型覆盖示例（v2 定稿验证）：
--   PG            storage 成员 ×3（两备 replication→主），VIP 在 db_cluster 端点
--   MySQL+proxy   proxy 成员 + storage 成员（traffic→proxy 成员，备 replication→主）
--   MongoDB 分片  mongos(proxy) / shard-N(storage, group_name=shard-N) / configsvr(arbiter)
--   Redis Cluster storage 成员 ×6（group_name=shard-1..3 主备；去中心化 traffic 置空）
--   OceanBase     obproxy(proxy) + tenant 逻辑单元（traffic→obproxy，units 落位）+
--                 observer 成员（group_name=ZONEx，Paxos 置空复制字段）
--   跨机房 DR     replication_upstream_id 跨集群指向对端成员（约定 ②）
-- =====================================================================
