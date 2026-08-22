-- =====================================================================
-- 002_whitelist.sql —— 数据面白名单表：告警 / 变更 / 慢日志
-- =====================================================================
-- 来源：ddl.sql 的 public.alerts / public.changes / public.slow_query /
--       public."lock"（旧系统导出结构，review 定稿见架构文档 §6.1.2）。
-- 定位：collector 定时拉取旧系统后直写的原始事实表（无业务语义）；
--       apiserver 只读消费（聚合/映射在消费端完成，如告警 Issue 化）。
-- review 结论：
--   · alerts 40 列近半为存储设备/运维流程长尾 → 收进 raw jsonb，不逐列建结构化列；
--   · 时间口径统一 timestamptz，fired_at 为主口径（原 alert_push_to_alert_time）；
--   · 级别保持源值（Critical/Major/…），P1/P2/P3 映射由消费端完成；
--   · changes 的 varchar 日期 → timestamptz，补对象关联（诊断关联「当时有没有变更」）；
--   · slow_query 为每次执行一行的事件流，前端需要按指纹聚合 → 补 digest 列；
--   · "lock" 不建表：锁状态秒级时效，按架构 §4.1 走 remote 实时采集，结论落诊断档案；
--     将来若需死锁历史审计，另立 deadlock_event 类事件表。
-- 指标表（series_meta/series_points）不在本文件范围，保持 mock（二期本地时序库另定）。
-- 运行时表结构由 apiserver GORM AutoMigrate 统一建模（model/metadata.go），
-- 本文件为评审/部署基准；逻辑外键不建硬约束。
-- =====================================================================


-- ---------------------------------------------------------------------
-- 图 1: 表 alert_raw —— 原始告警（旧告警系统 → collector 直写）
-- 每行一条告警事件；对象/级别/标题维度的聚合并发生次数由消费端 GROUP BY，
-- Issue 化（fingerprint、状态机、P1 映射）在控制面 Issue 域完成，不在本表加工
-- ---------------------------------------------------------------------
CREATE TABLE alert_raw (
    id           bigserial PRIMARY KEY,
    source_sys   text NOT NULL,                   -- 来源系统标识
    event_id     text NOT NULL,                   -- ← alerts.event_id（外部唯一 ID）
    object_name  text,                            -- ← alerts.resource / event_name（告警对象名）
    object_type  text,                            -- host/tenant/instance/cluster（结构化对象类型，替代前端解析对象名）
    instance_id  bigint,                          -- 逻辑外键 → db_instance.id（可空，关联回填）
    cluster_id   bigint,                          -- 逻辑外键 → db_cluster.id（可空，关联回填）
    alert_level  text,                            -- ← alerts.alert_level（原值：Critical/Major/Minor…；Critical=P1，Major=P2）
    alert_name   text,                            -- ← alerts.event_name（告警标题，聚合展示用）
    alert_desc   text,                            -- ← alerts.alert_description
    fired_at     timestamptz NOT NULL,            -- ← alerts.alert_push_to_alert_time（主时间口径）
    start_time   timestamptz,                     -- ← alerts.start_time
    end_time     timestamptz,                     -- ← alerts.end_time（恢复时间）
    environment  text,                            -- ← alerts.environment
    create_time  timestamptz,                     -- ← alerts.create_time
    update_time  timestamptz,                     -- ← alerts.update_time
    raw          jsonb DEFAULT '{}'::jsonb,       -- 源系统整行原样（nas_*/device_*/exemption_*/时长分钟数等长尾字段）

    CONSTRAINT uq_alert_raw_event_id UNIQUE (event_id)
);

CREATE INDEX idx_alert_raw_fired_at       ON alert_raw (fired_at DESC);        -- 告警列表按时间
CREATE INDEX idx_alert_raw_level_time     ON alert_raw (alert_level, fired_at); -- 按级别+时间过滤
CREATE INDEX idx_alert_raw_instance       ON alert_raw (instance_id);          -- 按实例关联


-- ---------------------------------------------------------------------
-- 图 2: 表 change_ticket —— 变更工单（旧日志/变更系统 → collector 直写）
-- 供诊断关联（某实例某时间窗内是否有变更）与问数统计
-- ---------------------------------------------------------------------
CREATE TABLE change_ticket (
    id               bigserial PRIMARY KEY,
    source_sys       text NOT NULL,               -- 来源系统标识
    ticket_no        text NOT NULL,               -- ← changes.ticket_no（外部唯一 ID）
    title            text,                        -- ← changes.title
    status_code      int,                         -- ← changes.status（源系统数字枚举，语义映射由消费端完成）
    risk_level       text,                        -- ← changes.risk_level
    owner_name       text,                        -- ← changes."owner"（改名避开保留字）
    plan_start_at    timestamptz,                 -- ← changes.plan_execute_date_start（原 varchar → timestamptz）
    plan_end_at      timestamptz,                 -- ← changes.plan_execute_date_end
    execute_start_at timestamptz,                 -- ← changes.execute_date_start
    execute_end_at   timestamptz,                 -- ← changes.execute_date_end
    expected_stop_at timestamptz,                 -- ← changes.expected_stop_time
    instance_id      bigint,                      -- 逻辑外键 → db_instance.id（可空，best-effort 回填）
    cluster_id       bigint,                      -- 逻辑外键 → db_cluster.id（可空）
    project_id       text,                        -- ← changes.project_id
    create_time      timestamptz,                 -- ← changes.create_time
    update_time      timestamptz,                 -- ← changes.update_time
    raw              jsonb DEFAULT '{}'::jsonb,   -- 源系统整行原样

    CONSTRAINT uq_change_ticket_no UNIQUE (ticket_no)
);

CREATE INDEX idx_change_ticket_execute   ON change_ticket (execute_start_at);          -- 时间窗关联查询
CREATE INDEX idx_change_ticket_instance  ON change_ticket (instance_id, execute_start_at);


-- ---------------------------------------------------------------------
-- 图 3: 表 slow_query_log —— 慢查询原始事件（旧日志系统 → collector 直写）
-- 每行一次执行；前端「SQL 诊断」面板的指纹级聚合（平均耗时/扫描行数/执行次数）
-- 由消费端 GROUP BY digest 完成；实时慢日志快照走 remote 通道，不落本表
-- ---------------------------------------------------------------------
CREATE TABLE slow_query_log (
    id             bigserial PRIMARY KEY,
    source_sys     text NOT NULL,                 -- 来源系统标识
    instance_id    bigint,                        -- 逻辑外键 → db_instance.id（可空，采集时按元数据匹配回填）
    endpoint       text,                          -- ← slow_query.endpoint（源系统访问端点）
    hostname       text,                          -- ← slow_query.hostname
    host_ip        text,                          -- ← slow_query.host_ip
    port           int,                           -- ← slow_query.port（int8 → int）
    database_name  text,                          -- ← slow_query.database_name（OB 为 tenant/db 复合）
    username       text,                          -- ← slow_query.username
    sql_text       text,                          -- ← slow_query.sql_text
    digest         text,                          -- SQL 指纹（归一化 SQL 的散列，聚合键）
    execute_ms     numeric(14, 2),                -- ← slow_query.execute_time（统一毫秒）
    rows_examined  bigint,                        -- 扫描行数（源缺失时为 NULL，展示为 —）
    execute_date   timestamptz,                   -- ← slow_query.execute_date
    create_time    timestamptz DEFAULT now()
);

CREATE INDEX idx_slow_query_log_inst_time  ON slow_query_log (instance_id, execute_date DESC); -- 实例+时间窗
CREATE INDEX idx_slow_query_log_digest     ON slow_query_log (digest);                         -- 指纹聚合
