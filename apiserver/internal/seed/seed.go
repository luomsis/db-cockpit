package seed

import (
	"log"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	"db-cockpit/apiserver/internal/model"
)

func J(s string) datatypes.JSON { return datatypes.JSON([]byte(s)) }

func sp(s string) *string { return &s }

/* ================= mockData.ts 全量转写（演示种子，与前端原 mock 逐值一致） ================= */

func clusterRows() []*model.Cluster {
	return []*model.Cluster{
		{ID: "c1", Name: "prod-pg-order-01", Type: "pg", Version: "PostgreSQL 15.6",
			Desc: "交易核心集群 · 华东-可用区B", AZ: "华东-AZ-B", Biz: "核心交易", Nodes: 3, Mode: "Patroni 流复制（1主2备）",
			CPU: 52, Mem: 67, Conn: 926, QPS: 8400, SyncMode: sp("Patroni · quorum（ANY 1 (pg2, pg3)）")},
		{ID: "c2", Name: "prod-pg-report-02", Type: "pg", Version: "PostgreSQL 15.6",
			Desc: "报表分析集群 · 华东-可用区B", AZ: "华东-AZ-B", Biz: "报表分析", Nodes: 2, Mode: "流复制（1主1备）",
			CPU: 38, Mem: 55, Conn: 460, QPS: 3100, SyncMode: sp("异步流复制")},
		{ID: "c3", Name: "prod-ob-core-01", Type: "oceanbase", Version: "OceanBase 4.2.1",
			Desc: "核心账务集群 · 华东可用区A/B/C", AZ: "华东-AZ-A/B/C", Biz: "核心账务", Nodes: 6, Mode: "3 Zone × 2 OBServer · Paxos",
			CPU: 68, Mem: 72, Conn: 1240, QPS: 18600, Zones: J(`["ZONE1","ZONE2","ZONE3"]`)},
		{ID: "c4", Name: "prod-ob-log-01", Type: "oceanbase", Version: "OceanBase 4.2.1",
			Desc: "日志分析集群 · 华东可用区A/B/C", AZ: "华东-AZ-A/B/C", Biz: "日志分析", Nodes: 3, Mode: "3 Zone × 1 OBServer · Paxos",
			CPU: 34, Mem: 48, Conn: 380, QPS: 6200, Zones: J(`["ZONE1","ZONE2","ZONE3"]`)},
	}
}

func instanceRows() []*model.Instance {
	return []*model.Instance{
		// c1 prod-pg-order-01
		{ID: "in-1e7f3", ClusterID: "c1", Name: "pg-order-01", Role: "主库 Primary", IP: "10.20.2.11", Port: 5432, Status: "ok", CPU: 45, Mem: 62, Conn: 320, Ver: "15.6", HostIP: "10.20.2.11"},
		{ID: "in-9c3d7", ClusterID: "c1", Name: "pg-order-02", Role: "备库 Standby", IP: "10.20.2.12", Port: 5432, Status: "warn", CPU: 78, Mem: 81, Conn: 410, Ver: "15.6", HostIP: "10.20.2.12"},
		{ID: "in-6b1a8", ClusterID: "c1", Name: "pg-order-03", Role: "备库 Standby", IP: "10.20.2.13", Port: 5432, Status: "ok", CPU: 33, Mem: 58, Conn: 196, Ver: "15.6", HostIP: "10.20.2.13"},
		// c2 prod-pg-report-02
		{ID: "in-3f0a1", ClusterID: "c2", Name: "pg-report-01", Role: "主库 Primary", IP: "10.20.2.21", Port: 5432, Status: "ok", CPU: 35, Mem: 52, Conn: 260, Ver: "15.6", HostIP: "10.20.2.21"},
		{ID: "in-7d2c4", ClusterID: "c2", Name: "pg-report-02", Role: "备库 Standby", IP: "10.20.2.22", Port: 5432, Status: "ok", CPU: 29, Mem: 48, Conn: 110, Ver: "15.6"},
		// c3 prod-ob-core-01
		{ID: "obc-z1-1", ClusterID: "c3", Name: "observer-zone1-01", Role: "OBServer", Zone: sp("ZONE1"), IP: "10.40.1.11", Port: 2881, Status: "ok", CPU: 71, Mem: 74, Conn: 420, Ver: "4.2.1", HostIP: "10.40.1.11"},
		{ID: "obc-z1-2", ClusterID: "c3", Name: "observer-zone1-02", Role: "OBServer", Zone: sp("ZONE1"), IP: "10.40.1.12", Port: 2881, Status: "ok", CPU: 55, Mem: 66, Conn: 380, Ver: "4.2.1", HostIP: "10.40.1.12"},
		{ID: "obc-z2-1", ClusterID: "c3", Name: "observer-zone2-01", Role: "OBServer", Zone: sp("ZONE2"), IP: "10.40.2.11", Port: 2881, Status: "warn", CPU: 82, Mem: 85, Conn: 300, Ver: "4.2.1", HostIP: "10.40.2.11"},
		{ID: "obc-z2-2", ClusterID: "c3", Name: "observer-zone2-02", Role: "OBServer", Zone: sp("ZONE2"), IP: "10.40.2.12", Port: 2881, Status: "ok", CPU: 49, Mem: 63, Conn: 90, Ver: "4.2.1"},
		{ID: "obc-z3-1", ClusterID: "c3", Name: "observer-zone3-01", Role: "OBServer", Zone: sp("ZONE3"), IP: "10.40.3.11", Port: 2881, Status: "ok", CPU: 52, Mem: 68, Conn: 30, Ver: "4.2.1"},
		{ID: "obc-z3-2", ClusterID: "c3", Name: "observer-zone3-02", Role: "OBServer", Zone: sp("ZONE3"), IP: "10.40.3.12", Port: 2881, Status: "ok", CPU: 47, Mem: 61, Conn: 20, Ver: "4.2.1"},
		// c4 prod-ob-log-01
		{ID: "obl-z1-1", ClusterID: "c4", Name: "obs-log-zone1-01", Role: "OBServer", Zone: sp("ZONE1"), IP: "10.41.1.11", Port: 2881, Status: "ok", CPU: 36, Mem: 49, Conn: 160, Ver: "4.2.1", HostIP: "10.41.1.11"},
		{ID: "obl-z2-1", ClusterID: "c4", Name: "obs-log-zone2-01", Role: "OBServer", Zone: sp("ZONE2"), IP: "10.41.2.11", Port: 2881, Status: "ok", CPU: 31, Mem: 45, Conn: 120, Ver: "4.2.1"},
		{ID: "obl-z3-1", ClusterID: "c4", Name: "obs-log-zone3-01", Role: "OBServer", Zone: sp("ZONE3"), IP: "10.41.3.11", Port: 2881, Status: "ok", CPU: 29, Mem: 44, Conn: 100, Ver: "4.2.1"},
	}
}

func pgDatabaseRows() []*model.PgDatabase {
	return []*model.PgDatabase{
		{ClusterID: "c1", Name: "trade_order", Owner: "app_rw", Size: "1.8 TB", Tables: 326, Conn: 320, ConnLimit: 400, Status: "ok"},
		{ClusterID: "c1", Name: "user_center", Owner: "app_rw", Size: "680 GB", Tables: 214, Conn: 190, ConnLimit: 300, Status: "ok"},
		{ClusterID: "c1", Name: "payment", Owner: "app_rw", Size: "420 GB", Tables: 158, Conn: 120, ConnLimit: 200, Status: "ok"},
		{ClusterID: "c1", Name: "analytics", Owner: "report_etl", Size: "2.9 TB", Tables: 502, Conn: 410, ConnLimit: 500, Status: "warn"},
		{ClusterID: "c2", Name: "bi_report", Owner: "report_rw", Size: "1.1 TB", Tables: 388, Conn: 210, ConnLimit: 300, Status: "ok"},
		{ClusterID: "c2", Name: "metrics_cache", Owner: "report_ro", Size: "240 GB", Tables: 96, Conn: 60, ConnLimit: 150, Status: "ok"},
	}
}

func pgReplicaRows() []*model.PgReplica {
	return []*model.PgReplica{
		{ClusterID: "c1", Instance: "pg-order-02", Role: "Standby（quorum）", DelayMs: 850, WalLag: "1.2 GB", Status: "warn"},
		{ClusterID: "c1", Instance: "pg-order-03", Role: "Standby（quorum）", DelayMs: 120, WalLag: "210 MB", Status: "ok"},
		{ClusterID: "c2", Instance: "pg-report-02", Role: "Standby（异步）", DelayMs: 240, WalLag: "480 MB", Status: "ok"},
	}
}

func clusterParamRows() []*model.ClusterParam {
	return []*model.ClusterParam{
		{Scope: "cluster", ClusterID: "c1", Name: "shared_buffers", Value: "32G", Range: "128MB - 128G", Desc: "共享缓冲区大小", Status: "ok"},
		{Scope: "cluster", ClusterID: "c1", Name: "max_connections", Value: "1000", Range: "1 - 262143", Desc: "最大连接数", Status: "ok"},
		{Scope: "cluster", ClusterID: "c1", Name: "work_mem", Value: "64MB", Range: "64kB - 2G", Desc: "排序/哈希操作内存", Status: "pending"},
		{Scope: "cluster", ClusterID: "c1", Name: "max_wal_size", Value: "16GB", Range: "2GB - 64GB", Desc: "WAL 上限（两次检查点间）", Status: "ok"},
		{Scope: "cluster", ClusterID: "c1", Name: "autovacuum", Value: "on", Range: "on / off", Desc: "自动清理开关", Status: "ok"},
		{Scope: "cluster", ClusterID: "c2", Name: "shared_buffers", Value: "16G", Range: "128MB - 64G", Desc: "共享缓冲区大小", Status: "ok"},
		{Scope: "cluster", ClusterID: "c2", Name: "max_connections", Value: "600", Range: "1 - 262143", Desc: "最大连接数", Status: "ok"},
		{Scope: "cluster", ClusterID: "c2", Name: "max_parallel_workers_per_gather", Value: "4", Range: "0 - 64", Desc: "单查询并行度", Status: "pending"},
		{Scope: "cluster", ClusterID: "c3", Name: "memory_limit", Value: "96G", Range: "16G - 物理内存", Desc: "OBServer 内存上限", Status: "ok"},
		{Scope: "cluster", ClusterID: "c3", Name: "system_memory", Value: "10G", Range: "2G - memory_limit", Desc: "系统预留内存（500 租户）", Status: "ok"},
		{Scope: "cluster", ClusterID: "c3", Name: "datafile_max_size", Value: "512G", Range: "磁盘容量内", Desc: "数据文件上限", Status: "ok"},
		{Scope: "cluster", ClusterID: "c3", Name: "enable_syslog_recycle", Value: "true", Range: "true / false", Desc: "系统日志自动回收", Status: "pending"},
		{Scope: "cluster", ClusterID: "c3", Name: "merger_check_interval", Value: "20m", Range: "1m - 24h", Desc: "合并检查间隔", Status: "ok"},
		{Scope: "cluster", ClusterID: "c4", Name: "memory_limit", Value: "48G", Range: "16G - 物理内存", Desc: "OBServer 内存上限", Status: "ok"},
		{Scope: "cluster", ClusterID: "c4", Name: "datafile_max_size", Value: "1T", Range: "磁盘容量内", Desc: "数据文件上限", Status: "ok"},
	}
}

func obTenantRows() []*model.ObTenant {
	return []*model.ObTenant{
		{ID: "t-sys", ClusterID: "c3", Name: "sys", Kind: "sys", Mode: "mysql", PrimaryZone: "RANDOM",
			Locality: "F@ZONE1,F@ZONE2,F@ZONE3", UnitNum: 1, MaxCpu: 6, UsedCpu: 1.4, MaxMemGb: 48, UsedMemGb: 11,
			StorageUsed: "40 GB", StorageTotal: "100 GB", Status: "ok",
			Units: J(`[{"zone":"ZONE1","observer":"observer-zone1-01","maxCpu":2,"usedCpu":0.5,"maxMemGb":16,"usedMemGb":3.8},{"zone":"ZONE2","observer":"observer-zone2-01","maxCpu":2,"usedCpu":0.4,"maxMemGb":16,"usedMemGb":3.6},{"zone":"ZONE3","observer":"observer-zone3-01","maxCpu":2,"usedCpu":0.5,"maxMemGb":16,"usedMemGb":3.6}]`),
			Whitelist: J(`["%"]`), ConnHint: "mysql -h10.40.1.11 -P2881 -uroot@sys -p***"},
		{ID: "t-trade", ClusterID: "c3", Name: "trade_tenant", Kind: "user", Mode: "mysql", PrimaryZone: "ZONE1",
			Locality: "F@ZONE1,F@ZONE2,F@ZONE3", UnitNum: 1, MaxCpu: 14, UsedCpu: 13.1, MaxMemGb: 72, UsedMemGb: 51,
			StorageUsed: "2.1 TB", StorageTotal: "4 TB", Status: "warn",
			Units: J(`[{"zone":"ZONE1","observer":"observer-zone1-01","maxCpu":14,"usedCpu":12.6,"maxMemGb":72,"usedMemGb":49},{"zone":"ZONE2","observer":"observer-zone2-01","maxCpu":14,"usedCpu":13.4,"maxMemGb":72,"usedMemGb":52},{"zone":"ZONE3","observer":"observer-zone3-01","maxCpu":14,"usedCpu":13.3,"maxMemGb":72,"usedMemGb":52}]`),
			Whitelist: J(`["10.40.10.%","10.40.11.%"]`), ConnHint: "mysql -h10.40.1.21 -P2881 -utrade_rw@trade_tenant -p***"},
		{ID: "t-pay", ClusterID: "c3", Name: "pay_tenant", Kind: "user", Mode: "mysql", PrimaryZone: "ZONE2",
			Locality: "F@ZONE1,F@ZONE2,F@ZONE3", UnitNum: 1, MaxCpu: 12, UsedCpu: 7.4, MaxMemGb: 48, UsedMemGb: 43.7,
			StorageUsed: "860 GB", StorageTotal: "1.5 TB", Status: "warn",
			Units: J(`[{"zone":"ZONE1","observer":"observer-zone1-02","maxCpu":12,"usedCpu":7.1,"maxMemGb":48,"usedMemGb":43.2},{"zone":"ZONE2","observer":"observer-zone2-02","maxCpu":12,"usedCpu":7.6,"maxMemGb":48,"usedMemGb":44.1},{"zone":"ZONE3","observer":"observer-zone3-02","maxCpu":12,"usedCpu":7.5,"maxMemGb":48,"usedMemGb":43.8}]`),
			Whitelist: J(`["10.40.20.%"]`), ConnHint: "mysql -h10.40.2.12 -P2881 -upay_rw@pay_tenant -p***"},
		{ID: "t-log", ClusterID: "c4", Name: "log_tenant", Kind: "user", Mode: "mysql", PrimaryZone: "ZONE1",
			Locality: "F@ZONE1,F@ZONE2,F@ZONE3", UnitNum: 1, MaxCpu: 6, UsedCpu: 2.1, MaxMemGb: 32, UsedMemGb: 12,
			StorageUsed: "3.2 TB", StorageTotal: "8 TB", Status: "ok",
			Units: J(`[{"zone":"ZONE1","observer":"obs-log-zone1-01","maxCpu":6,"usedCpu":2.2,"maxMemGb":32,"usedMemGb":12.4},{"zone":"ZONE2","observer":"obs-log-zone2-01","maxCpu":6,"usedCpu":2.0,"maxMemGb":32,"usedMemGb":11.8},{"zone":"ZONE3","observer":"obs-log-zone3-01","maxCpu":6,"usedCpu":2.1,"maxMemGb":32,"usedMemGb":11.8}]`),
			Whitelist: J(`["10.41.0.%"]`), ConnHint: "mysql -h10.41.1.11 -P2881 -ulog_rw@log_tenant -p***"},
		{ID: "t-logsys", ClusterID: "c4", Name: "sys", Kind: "sys", Mode: "mysql", PrimaryZone: "RANDOM",
			Locality: "F@ZONE1,F@ZONE2,F@ZONE3", UnitNum: 1, MaxCpu: 2, UsedCpu: 0.4, MaxMemGb: 16, UsedMemGb: 4,
			StorageUsed: "20 GB", StorageTotal: "60 GB", Status: "ok",
			Units: J(`[{"zone":"ZONE1","observer":"obs-log-zone1-01","maxCpu":2,"usedCpu":0.4,"maxMemGb":16,"usedMemGb":4.1},{"zone":"ZONE2","observer":"obs-log-zone2-01","maxCpu":2,"usedCpu":0.4,"maxMemGb":16,"usedMemGb":4.0},{"zone":"ZONE3","observer":"obs-log-zone3-01","maxCpu":2,"usedCpu":0.4,"maxMemGb":16,"usedMemGb":4.0}]`),
			Whitelist: J(`["%"]`), ConnHint: "mysql -h10.41.1.11 -P2881 -uroot@sys -p***"},
	}
}

func obTenantDbRows() []*model.ObTenantDb {
	return []*model.ObTenantDb{
		{TenantID: "t-sys", Name: "oceanbase", Tables: 0, Size: "38 GB", Conn: 12, Status: "ok"},
		{TenantID: "t-trade", Name: "trade_order", Tables: 326, Size: "1.2 TB", Conn: 180, Status: "ok"},
		{TenantID: "t-trade", Name: "inventory", Tables: 154, Size: "640 GB", Conn: 96, Status: "ok"},
		{TenantID: "t-trade", Name: "seckill", Tables: 48, Size: "88 GB", Conn: 210, Status: "warn"},
		{TenantID: "t-pay", Name: "payment", Tables: 96, Size: "620 GB", Conn: 140, Status: "ok"},
		{TenantID: "t-pay", Name: "billing", Tables: 61, Size: "240 GB", Conn: 52, Status: "ok"},
		{TenantID: "t-log", Name: "access_log", Tables: 12, Size: "2.9 TB", Conn: 40, Status: "ok"},
		{TenantID: "t-log", Name: "audit_log", Tables: 8, Size: "260 GB", Conn: 12, Status: "ok"},
		{TenantID: "t-logsys", Name: "oceanbase", Tables: 0, Size: "18 GB", Conn: 4, Status: "ok"},
	}
}

func hostRows() []*model.Host {
	return []*model.Host{
		{IP: "10.20.2.11", Zone: "华东-AZ-B", Spec: "16C / 64G", OS: "CentOS 7.9", CPU: 45, Mem: 62, Disk: 58, DiskTotal: 1800, ClusterID: "c1", Status: "ok"},
		{IP: "10.20.2.12", Zone: "华东-AZ-B", Spec: "16C / 64G", OS: "CentOS 7.9", CPU: 78, Mem: 81, Disk: 72, DiskTotal: 1800, ClusterID: "c1", Status: "warn"},
		{IP: "10.20.2.13", Zone: "华东-AZ-B", Spec: "16C / 64G", OS: "CentOS 7.9", CPU: 33, Mem: 58, Disk: 55, DiskTotal: 1800, ClusterID: "c1", Status: "ok"},
		{IP: "10.40.1.11", Zone: "华东-AZ-A", Spec: "16C / 96G", OS: "CentOS 7.9", CPU: 71, Mem: 74, Disk: 66, DiskTotal: 2000, ClusterID: "c3", Status: "ok"},
		{IP: "10.40.2.11", Zone: "华东-AZ-B", Spec: "16C / 96G", OS: "CentOS 7.9", CPU: 82, Mem: 85, Disk: 69, DiskTotal: 2000, ClusterID: "c3", Status: "warn"},
		{IP: "10.40.1.12", Zone: "华东-AZ-A", Spec: "16C / 96G", OS: "CentOS 7.9", CPU: 55, Mem: 66, Disk: 61, DiskTotal: 2000, ClusterID: "c3", Status: "ok"},
		{IP: "10.41.1.11", Zone: "华东-AZ-A", Spec: "8C / 48G", OS: "Ubuntu 22.04", CPU: 36, Mem: 49, Disk: 84, DiskTotal: 1200, ClusterID: "c4", Status: "ok"},
		{IP: "10.20.2.21", Zone: "华东-AZ-B", Spec: "16C / 64G", OS: "CentOS 7.9", CPU: 35, Mem: 52, Disk: 52, DiskTotal: 1800, ClusterID: "c2", Status: "ok"},
	}
}

func instanceUserRows() []*model.InstanceUser {
	return []*model.InstanceUser{
		{Username: "trade_rw@trade_tenant", Host: "10.40.10.%", Priv: "SELECT, INSERT, UPDATE", LastLogin: "2026-08-16 15:42", Status: "ok"},
		{Username: "trade_ro@trade_tenant", Host: "10.40.10.%", Priv: "SELECT", LastLogin: "2026-08-16 15:39", Status: "ok"},
		{Username: "root@sys", Host: "%", Priv: "ALL PRIVILEGES", LastLogin: "2026-08-15 22:10", Status: "ok"},
		{Username: "app_rw", Host: "10.20.%.%", Priv: "SELECT, INSERT, UPDATE", LastLogin: "2026-08-16 14:20", Status: "ok"},
		{Username: "report_etl", Host: "10.21.0.%", Priv: "SELECT, COPY", LastLogin: "2026-08-14 03:00", Status: "warn"},
		{Username: "tmp_debug", Host: "10.99.1.5", Priv: "SELECT", LastLogin: "2026-07-28 11:20", Status: "err"},
	}
}

func sessionRows() []*model.RuntimeSession {
	return []*model.RuntimeSession{
		{SessionID: 88231, Username: "trade_rw@trade_tenant", Host: "10.40.10.21:40218", Db: "trade_order", Cmd: "Query", Time: "28s", State: "Sending data", LockInfo: "行锁等待", Status: "err"},
		{SessionID: 88190, Username: "trade_rw@trade_tenant", Host: "10.40.10.22:40871", Db: "trade_order", Cmd: "Query", Time: "12s", State: "update", LockInfo: "—", Status: "warn"},
		{SessionID: 88102, Username: "report_etl", Host: "10.21.0.8:38452", Db: "analytics", Cmd: "Query", Time: "96s", State: "Copying to tmp table", LockInfo: "—", Status: "warn"},
		{SessionID: 87955, Username: "app_ro", Host: "10.20.5.10:41002", Db: "user_center", Cmd: "Sleep", Time: "300s", State: "—", LockInfo: "—", Status: "ok"},
		{SessionID: 87901, Username: "pay_rw@pay_tenant", Host: "10.40.20.9:40233", Db: "payment", Cmd: "Query", Time: "3s", State: "executing", LockInfo: "—", Status: "ok"},
	}
}

func trxRows() []*model.Trx {
	return []*model.Trx{
		{ID: "TRX-998231", Session: 88231, Username: "trade_rw@trade_tenant", Dur: "1m 28s", Undo: "14.2 MB", LockRows: "38,201", Waiting: "是", Sql: "UPDATE stock_record SET qty = qty - ? …", Status: "err"},
		{ID: "TRX-998190", Session: 88190, Username: "trade_rw@trade_tenant", Dur: "42s", Undo: "6.8 MB", LockRows: "12,044", Waiting: "否", Sql: "INSERT INTO trade_order (…) VALUES (…)", Status: "warn"},
		{ID: "TRX-998102", Session: 88102, Username: "report_etl", Dur: "96s", Undo: "0.2 MB", LockRows: "0", Waiting: "否", Sql: "SELECT COUNT(*) FROM access_log …", Status: "ok"},
	}
}

func slowSqlRows() []*model.SlowSql {
	return []*model.SlowSql{
		{Sql: "SELECT o.*, u.name FROM trade_order o JOIN user u ON o.uid = u.id WHERE o.status = ?", Db: "trade_tenant/trade_order", Time: "12.8s", Rows: "4,380,012", Count: 342},
		{Sql: "UPDATE stock_record SET qty = qty - ? WHERE sku_id = ? AND warehouse_id = ?", Db: "trade_tenant/inventory", Time: "9.6s", Rows: "1,203,550", Count: 187},
		{Sql: "SELECT COUNT(*) FROM access_log WHERE create_time BETWEEN ? AND ? GROUP BY path", Db: "log_tenant/access_log", Time: "8.2s", Rows: "9,881,204", Count: 96},
		{Sql: "SELECT * FROM payment_bill WHERE bill_no LIKE ? ORDER BY ctime DESC LIMIT ?", Db: "pay_tenant/payment", Time: "6.4s", Rows: "760,332", Count: 64},
		{Sql: "DELETE FROM session_token WHERE expire_at < ? AND app_id IN (?, ?, ?)", Db: "pg-order-01/auth", Time: "5.1s", Rows: "2,310,778", Count: 41},
	}
}

func reportRows() []*model.Report {
	return []*model.Report{
		{Ico: "📊", Title: "性能周报 · 第 33 周", Desc: "PG/OB 集群负载、TOP SQL、容量趋势", Date: "2026-08-17 08:00", Size: "2.4 MB"},
		{Ico: "🔍", Title: "慢 SQL 专项治理报告", Desc: "本周新增 23 条慢 SQL，已优化 15 条", Date: "2026-08-16 18:00", Size: "1.1 MB"},
		{Ico: "📈", Title: "OB 租户容量预测（8月）", Desc: "trade_tenant 存储 47 天后触达 85% 水位", Date: "2026-08-15 08:00", Size: "3.2 MB"},
		{Ico: "🛡️", Title: "巡检报告 · 每日", Desc: "参数基线、备份校验、账号审计", Date: "2026-08-17 06:00", Size: "860 KB"},
	}
}

func alertRows() []*model.AlertRecord {
	return []*model.AlertRecord{
		/* ---- P1 紧急 ---- */
		{Name: "trade_tenant @ prod-ob-core-01", Severity: "P1", Title: "租户 CPU 13.1/14C（阈值 90%）", Time: "08-18 14:32", Count: 6},
		{Name: "prod-pg-order-01（主库）", Severity: "P1", Title: "连接数 962/1000 · 接近上限", Time: "08-18 11:20", Count: 4},
		/* ---- P2 重要 ---- */
		{Name: "pg-order-01（主库）", Severity: "P2", Title: "备库复制延迟 850ms（阈值 300ms）", Time: "08-18 14:18", Count: 3},
		{Name: "pay_tenant @ prod-ob-core-01", Severity: "P2", Title: "租户内存水位 91%", Time: "08-18 13:55", Count: 2},
		{Name: "analytics @ prod-pg-order-01", Severity: "P2", Title: "连接数 410/500 · 慢查询堆积", Time: "08-18 12:40", Count: 1},
		{Name: "observer-zone2-01", Severity: "P2", Title: "OBServer CPU 82% · 建议检查 Unit 均衡", Time: "08-18 11:02", Count: 1},
		{Name: "host-10.20.2.12", Severity: "P2", Title: "磁盘使用率 72% · 持续上升趋势", Time: "08-17 22:10", Count: 2},
		{Name: "prod-ob-log-01", Severity: "P2", Title: "major 合并耗时超过 2h", Time: "08-17 03:40", Count: 1},
		/* ---- P3 关注 ---- */
		{Name: "seckill @ t-trade", Severity: "P3", Title: "慢 SQL 数量突增（+35%/小时）", Time: "08-18 10:15", Count: 5},
		{Name: "user_center @ prod-pg-order-01", Severity: "P3", Title: "今日死锁 2 次（锁等待连锁）", Time: "08-18 09:40", Count: 2},
		{Name: "observer-zone1-02", Severity: "P3", Title: "RPC 超时率 0.8%（阈值 1%）", Time: "08-17 18:22", Count: 1},
		{Name: "host-10.40.1.12", Severity: "P3", Title: "内存使用率 66% · 周环比 +9%", Time: "08-17 14:30", Count: 1},
		{Name: "log_tenant @ prod-ob-log-01", Severity: "P3", Title: "存储 3.2/8 TB · 40% 水位提醒", Time: "08-17 09:00", Count: 1},
		{Name: "pg-report-02（备库）", Severity: "P3", Title: "autovacuum 积压 · 磁盘 78%", Time: "08-16 20:30", Count: 2},
		{Name: "bi_report @ prod-pg-report-02", Severity: "P3", Title: "表膨胀率 38%（建议 VACUUM）", Time: "08-16 16:45", Count: 1},
		{Name: "prod-ob-core-01", Severity: "P3", Title: "系统租户内存 11/48G · 预留水位提示", Time: "08-16 08:15", Count: 1},
		{Name: "metrics_cache @ prod-pg-report-02", Severity: "P3", Title: "空闲连接占比偏高（60/150）", Time: "08-16 11:05", Count: 1},
		{Name: "observer-zone3-01", Severity: "P3", Title: "副本追赶延迟 1.2s · 观察中", Time: "08-17 07:55", Count: 1},
	}
}

func metricRows() []*model.MetricDef {
	return []*model.MetricDef{
		{ID: "qps", Name: "QPS", Unit: "", Base: 210, Jitter: 60},
		{ID: "cpu", Name: "CPU 使用率", Unit: "%", Base: 55, Jitter: 14},
		{ID: "mem", Name: "内存使用率", Unit: "%", Base: 68, Jitter: 10},
		{ID: "sessions", Name: "活跃会话数", Unit: "", Base: 120, Jitter: 40},
		{ID: "slow_sql", Name: "慢 SQL 趋势", Unit: "", Base: 18, Jitter: 8},
		{ID: "lock_wait", Name: "锁等待会话", Unit: "", Base: 6, Jitter: 4},
		{ID: "disk", Name: "磁盘使用率", Unit: "%", Base: 66, Jitter: 3},
		{ID: "repl_delay", Name: "复制/同步延迟", Unit: "s", Base: 2, Jitter: 3},
	}
}

func tenantParamRows() []*model.ClusterParam {
	tp := func(clusterID, tenantID, name, value, rng, desc, status string) *model.ClusterParam {
		return &model.ClusterParam{Scope: "tenant", ClusterID: clusterID, TenantID: sp(tenantID),
			Name: name, Value: value, Range: rng, Desc: desc, Status: status}
	}
	rows := []*model.ClusterParam{}
	for _, t := range obTenantRows() {
		ver := "8.0.x"
		if t.Mode != "mysql" {
			ver = "11.x"
		}
		rows = append(rows,
			tp(t.ClusterID, t.ID, "ob_max_compatible_version", ver, "—", "租户兼容模式版本", "ok"),
			tp(t.ClusterID, t.ID, "max_connections", itoa(500*t.UnitNum), "1 - 100000", "租户最大连接数", "ok"),
			tp(t.ClusterID, t.ID, "ob_sql_work_area_percentage", "5", "0 - 80", "SQL 工作区内存占比（%）", "pending"),
			tp(t.ClusterID, t.ID, "parallel_servers_target", itoa(int(8*t.MaxCpu)), "1 - 999999", "并行查询目标队列", "ok"),
		)
	}
	return rows
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	neg := i < 0
	if neg {
		i = -i
	}
	b := [20]byte{}
	p := len(b)
	for i > 0 {
		p--
		b[p] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		p--
		b[p] = '-'
	}
	return string(b[p:])
}

func metaStatRows() []*model.MetaStat {
	return []*model.MetaStat{
		{Key: "db_types", Value: J(`[{"type":"pg","name":"PostgreSQL","icon":"🐘","total":42,"alert":5},{"type":"oceanbase","name":"OceanBase","icon":"🌊","total":36,"alert":4}]`)},
		{Key: "top_anomaly", Value: J(`[
			{"name":"trade_tenant @ prod-ob-core-01","cluster":"prod-ob-core-01","score":96,"issue":"租户 CPU 13.1/14C · 慢SQL激增","inst":"trade_tenant"},
			{"name":"pg-order-01（主库）","cluster":"prod-pg-order-01","score":88,"issue":"复制延迟 850ms · WAL 积压 1.2GB","inst":"in-1e7f3"},
			{"name":"pay_tenant @ prod-ob-core-01","cluster":"prod-ob-core-01","score":82,"issue":"租户内存水位 91%","inst":"pay_tenant"},
			{"name":"analytics @ prod-pg-order-01","cluster":"prod-pg-order-01","score":74,"issue":"连接数 410/500 · 慢查询堆积","inst":"in-1e7f3"},
			{"name":"pg-report-02（备库）","cluster":"prod-pg-report-02","score":68,"issue":"autovacuum 积压 · 磁盘 78%","inst":"in-7d2c4"}]`)},
		{Key: "sql_issues", Value: J(`[{"name":"全表扫描","cnt":23},{"name":"缺失索引","cnt":18},{"name":"隐式类型转换","cnt":12},{"name":"过度排序","cnt":9},{"name":"深分页","cnt":7},{"name":"冗余 JOIN","cnt":5}]`)},
		// 概览页「锁分析」卡片静态值（Overview.tsx）
		{Key: "lock_summary", Value: J(`{"lockWaitRate":38,"lockWaitSessions":7,"deadlockToday":2,"longestWait":"1m 28s · TRX-998231","mdlBlocked":1,"hotTable":"stock_record"}`)},
	}
}

/* ================= 幂等导入 ================= */

func Run(gdb *gorm.DB) error {
	// 指标目录：始终幂等 upsert
	for _, m := range metricRows() {
		if err := gdb.Save(m).Error; err != nil {
			return err
		}
	}
	// 元数据域 + 数据面白名单：各表独立「表空才导入」（与主演示种子互不影响）
	if err := RunWhitelist(gdb); err != nil {
		return err
	}
	var clusterCount int64
	if err := gdb.Model(&model.Cluster{}).Count(&clusterCount).Error; err != nil {
		return err
	}
	if clusterCount > 0 {
		log.Printf("[seed] already seeded (clusters=%d), skip", clusterCount)
		return nil
	}

	type batch struct {
		name string
		run  func(tx *gorm.DB) error
	}
	groups := []batch{
		{"clusters", func(tx *gorm.DB) error { return tx.CreateInBatches(clusterRows(), 100).Error }},
		{"instances", func(tx *gorm.DB) error { return tx.CreateInBatches(instanceRows(), 100).Error }},
		{"cluster_params", func(tx *gorm.DB) error { return tx.CreateInBatches(clusterParamRows(), 100).Error }},
		{"pg_databases", func(tx *gorm.DB) error { return tx.CreateInBatches(pgDatabaseRows(), 100).Error }},
		{"pg_replicas", func(tx *gorm.DB) error { return tx.CreateInBatches(pgReplicaRows(), 100).Error }},
		{"ob_tenants", func(tx *gorm.DB) error { return tx.CreateInBatches(obTenantRows(), 100).Error }},
		{"ob_tenant_dbs", func(tx *gorm.DB) error { return tx.CreateInBatches(obTenantDbRows(), 100).Error }},
		{"tenant_params", func(tx *gorm.DB) error { return tx.CreateInBatches(tenantParamRows(), 100).Error }},
		{"hosts", func(tx *gorm.DB) error { return tx.CreateInBatches(hostRows(), 100).Error }},
		{"instance_users", func(tx *gorm.DB) error { return tx.CreateInBatches(instanceUserRows(), 100).Error }},
		{"runtime_sessions", func(tx *gorm.DB) error { return tx.CreateInBatches(sessionRows(), 100).Error }},
		{"transactions", func(tx *gorm.DB) error { return tx.CreateInBatches(trxRows(), 100).Error }},
		{"slow_sqls", func(tx *gorm.DB) error { return tx.CreateInBatches(slowSqlRows(), 100).Error }},
		{"reports", func(tx *gorm.DB) error { return tx.CreateInBatches(reportRows(), 100).Error }},
		{"alert_records", func(tx *gorm.DB) error { return tx.CreateInBatches(alertRows(), 100).Error }},
		{"meta_stats", func(tx *gorm.DB) error { return tx.CreateInBatches(metaStatRows(), 100).Error }},
	}
	if err := gdb.Transaction(func(tx *gorm.DB) error {
		for _, g := range groups {
			if err := g.run(tx); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}
	log.Printf("[seed] imported demo fixtures (%d groups)", len(groups))
	return nil
}
