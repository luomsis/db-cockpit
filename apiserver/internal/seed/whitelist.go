package seed

import (
	"crypto/md5"
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"

	"db-cockpit/apiserver/internal/model"
)

/* ================= 元数据域 + 数据面白名单演示种子 =================
与主种子（clusters 演示域）独立：各表「表空才导入」，已种子过的库可增量补种。
三值复用主种子同款对象（集群/实例/租户同名同 IP），前端切到白名单聚合后视图一致。 */

func tp(t time.Time) *time.Time { return &t }

func ipid(v int64) *int64 { return &v }

// refOf 映射取指针；键缺失返回 nil（关联列可空，缺失不该落 0 这种伪 ID）
func refOf(m map[string]int64, key string) *int64 {
	if v, ok := m[key]; ok {
		return &v
	}
	return nil
}

// instanceKey 元数据域实例键：clusterName/instanceName（OB 两个集群都有 sys 租户，需带集群名区分）
func instanceKey(clusterName, name string) string { return clusterName + "/" + name }

type instSeed struct {
	key string
	row *model.DbInstance
}

func whitelistClusters() []*model.DbCluster {
	now := time.Now()
	base := func() time.Time { return time.Date(2024, 3, 12, 10, 0, 0, 0, time.Local) }
	return []*model.DbCluster{
		{Name: "prod-pg-order-01", Description: "交易核心集群 · 华东-可用区B", DbType: "pg", Environment: "生产",
			OrgCode: "ORG-CORE-TRADE", ServiceUser: "李明", OprDba: "王强", OprDbaIi: "赵磊", BusinessOwner: "陈锋",
			AlertSubscriber: "王强,赵磊", SubsysCode: "trade-order", SourceSys: "ucmdb", CcmName: "ccm-pg-order", LeName: "le-sh-east",
			HaType: "Patroni 流复制（1主2备）", BackupMethod: "pg_basebackup 每日全量 + WAL 归档", FailoverType: "Patroni 自动切换",
			IsCreatedByCloud: false, SourceID: "ucmdb-ent-001", CreatedAt: base(), SyncedAt: tp(now),
			Extensions: J(`{"az":"华东-AZ-B","biz":"核心交易"}`)},
		{Name: "prod-pg-report-02", Description: "报表分析集群 · 华东-可用区B", DbType: "pg", Environment: "生产",
			OrgCode: "ORG-BI", ServiceUser: "周婷", OprDba: "赵磊", OprDbaIi: "王强", BusinessOwner: "吴楠",
			AlertSubscriber: "赵磊", SubsysCode: "bi-report", SourceSys: "ucmdb", CcmName: "ccm-pg-report", LeName: "le-sh-east",
			HaType: "流复制（1主1备）", BackupMethod: "pg_basebackup 每日全量", FailoverType: "手动提升",
			IsCreatedByCloud: false, SourceID: "ucmdb-ent-002", CreatedAt: base(), SyncedAt: tp(now),
			Extensions: J(`{"az":"华东-AZ-B","biz":"报表分析"}`)},
		{Name: "prod-ob-core-01", Description: "核心账务集群 · 华东可用区A/B/C", DbType: "oceanbase", Environment: "生产",
			OrgCode: "ORG-CORE-TRADE", ServiceUser: "李明", OprDba: "王强", OprDbaIi: "赵磊", BusinessOwner: "陈锋",
			AlertSubscriber: "王强,赵磊", SubsysCode: "core-ledger", SourceSys: "ucmdb", CcmName: "ccm-ob-core", LeName: "le-sh-east",
			HaType: "3 Zone × 2 OBServer · Paxos", BackupMethod: "每日全量备份 + 日志归档", FailoverType: "Paxos 多数派自动切换",
			IsCreatedByCloud: true, SourceID: "ucmdb-ent-003", CreatedAt: base(), SyncedAt: tp(now),
			Extensions: J(`{"zones":["ZONE1","ZONE2","ZONE3"],"biz":"核心账务"}`)},
		{Name: "prod-ob-log-01", Description: "日志分析集群 · 华东可用区A/B/C", DbType: "oceanbase", Environment: "生产",
			OrgCode: "ORG-BI", ServiceUser: "周婷", OprDba: "赵磊", OprDbaIi: "王强", BusinessOwner: "吴楠",
			AlertSubscriber: "赵磊", SubsysCode: "log-analytics", SourceSys: "ucmdb", CcmName: "ccm-ob-log", LeName: "le-sh-east",
			HaType: "3 Zone × 1 OBServer · Paxos", BackupMethod: "每日全量备份", FailoverType: "Paxos 多数派自动切换",
			IsCreatedByCloud: true, SourceID: "ucmdb-ent-004", CreatedAt: base(), SyncedAt: tp(now),
			Extensions: J(`{"zones":["ZONE1","ZONE2","ZONE3"],"biz":"日志分析"}`)},
	}
}

// whitelistInstances OB 按「租户=实例」约定：sys 租户实例（role=sys）承载集群物理拓扑，
// 业务租户实例 extensions 携带租户规格（与主种子 ob_tenants 同值）。
func whitelistInstances(cl map[string]int64) []instSeed {
	obCore, obLog := cl["prod-ob-core-01"], cl["prod-ob-log-01"]
	return []instSeed{
		{"prod-pg-order-01/pg-order-01", &model.DbInstance{
			ClusterID: cl["prod-pg-order-01"], DbType: "pg", Name: "pg-order-01", Version: "PostgreSQL 15.6", Status: "ok",
			Role: "storage", CharacterSet: "UTF8", InfraType: "虚拟机",
			ReqCPU: 16, ReqMemoryGb: 64, ReqStorageGb: 2000, AttachDB: "trade_order;user_center;payment;analytics",
			Endpoint: "10.20.2.11:5432", Vip: "10.20.2.10", Port: 5432, Username: "app_rw", RoleSelector: "primary",
			SourceID: "ucmdb-ins-101", CreatedAt: time.Date(2024, 3, 12, 10, 0, 0, 0, time.Local), UpdatedAt: time.Now(),
			Extensions: J(`{"sync_mode":"quorum","nodes":3}`)}},
		{"prod-pg-report-02/pg-report-01", &model.DbInstance{
			ClusterID: cl["prod-pg-report-02"], DbType: "pg", Name: "pg-report-01", Version: "PostgreSQL 15.6", Status: "ok",
			Role: "storage", CharacterSet: "UTF8", InfraType: "虚拟机",
			ReqCPU: 16, ReqMemoryGb: 64, ReqStorageGb: 1800, AttachDB: "bi_report;metrics_cache",
			Endpoint: "10.20.2.21:5432", Vip: "10.20.2.20", Port: 5432, Username: "report_rw", RoleSelector: "primary",
			SourceID: "ucmdb-ins-102", CreatedAt: time.Date(2024, 6, 1, 9, 0, 0, 0, time.Local), UpdatedAt: time.Now(),
			Extensions: J(`{"sync_mode":"async","nodes":2}`)}},
		{"prod-ob-core-01/sys", &model.DbInstance{
			ClusterID: obCore, DbType: "oceanbase", Name: "sys", Version: "OceanBase 4.2.1", Status: "ok",
			Role: "sys", CharacterSet: "utf8mb4", InfraType: "物理机",
			Endpoint: "10.40.1.11:2881", Port: 2881, Username: "root@sys", RoleSelector: "any",
			SourceID: "ucmdb-ins-201", CreatedAt: time.Date(2024, 3, 12, 10, 0, 0, 0, time.Local), UpdatedAt: time.Now(),
			Extensions: J(`{"mode":"mysql","tenant_kind":"sys","primary_zone":"RANDOM","locality":"F@ZONE1,F@ZONE2,F@ZONE3","unit_num":1,"max_cpu":6,"max_mem_gb":48,"whitelist":["%"]}`)}},
		{"prod-ob-core-01/trade_tenant", &model.DbInstance{
			ClusterID: obCore, DbType: "oceanbase", Name: "trade_tenant", Version: "OceanBase 4.2.1", Status: "warn",
			Role: "user", CharacterSet: "utf8mb4", InfraType: "物理机",
			ReqCPU: 14, ReqMemoryGb: 72, ReqStorageGb: 4096, AttachDB: "trade_order;inventory;seckill",
			Endpoint: "10.40.1.21:2881", Port: 2881, Username: "trade_rw@trade_tenant", RoleSelector: "primary",
			SourceID: "ucmdb-ins-202", CreatedAt: time.Date(2024, 3, 20, 14, 0, 0, 0, time.Local), UpdatedAt: time.Now(),
			Extensions: J(`{"mode":"mysql","tenant_kind":"user","primary_zone":"ZONE1","locality":"F@ZONE1,F@ZONE2,F@ZONE3","unit_num":1,"max_cpu":14,"max_mem_gb":72,"whitelist":["10.40.10.%","10.40.11.%"],"units":[{"zone":"ZONE1","server_ip":"10.40.1.11"},{"zone":"ZONE2","server_ip":"10.40.2.11"},{"zone":"ZONE3","server_ip":"10.40.3.11"}]}`)}},
		{"prod-ob-core-01/pay_tenant", &model.DbInstance{
			ClusterID: obCore, DbType: "oceanbase", Name: "pay_tenant", Version: "OceanBase 4.2.1", Status: "warn",
			Role: "user", CharacterSet: "utf8mb4", InfraType: "物理机",
			ReqCPU: 12, ReqMemoryGb: 48, ReqStorageGb: 1536, AttachDB: "payment;billing",
			Endpoint: "10.40.2.12:2881", Port: 2881, Username: "pay_rw@pay_tenant", RoleSelector: "primary",
			SourceID: "ucmdb-ins-203", CreatedAt: time.Date(2024, 8, 5, 16, 0, 0, 0, time.Local), UpdatedAt: time.Now(),
			Extensions: J(`{"mode":"mysql","tenant_kind":"user","primary_zone":"ZONE2","locality":"F@ZONE1,F@ZONE2,F@ZONE3","unit_num":1,"max_cpu":12,"max_mem_gb":48,"whitelist":["10.40.20.%"],"units":[{"zone":"ZONE1","server_ip":"10.40.1.12"},{"zone":"ZONE2","server_ip":"10.40.2.12"},{"zone":"ZONE3","server_ip":"10.40.3.12"}]}`)}},
		{"prod-ob-log-01/sys", &model.DbInstance{
			ClusterID: obLog, DbType: "oceanbase", Name: "sys", Version: "OceanBase 4.2.1", Status: "ok",
			Role: "sys", CharacterSet: "utf8mb4", InfraType: "虚拟机",
			Endpoint: "10.41.1.11:2881", Port: 2881, Username: "root@sys", RoleSelector: "any",
			SourceID: "ucmdb-ins-301", CreatedAt: time.Date(2025, 1, 15, 11, 0, 0, 0, time.Local), UpdatedAt: time.Now(),
			Extensions: J(`{"mode":"mysql","tenant_kind":"sys","primary_zone":"RANDOM","locality":"F@ZONE1,F@ZONE2,F@ZONE3","unit_num":1,"max_cpu":2,"max_mem_gb":16,"whitelist":["%"]}`)}},
		{"prod-ob-log-01/log_tenant", &model.DbInstance{
			ClusterID: obLog, DbType: "oceanbase", Name: "log_tenant", Version: "OceanBase 4.2.1", Status: "ok",
			Role: "user", CharacterSet: "utf8mb4", InfraType: "虚拟机",
			ReqCPU: 6, ReqMemoryGb: 32, ReqStorageGb: 8192, AttachDB: "access_log;audit_log",
			Endpoint: "10.41.1.11:2881", Port: 2881, Username: "log_rw@log_tenant", RoleSelector: "primary",
			SourceID: "ucmdb-ins-302", CreatedAt: time.Date(2025, 1, 15, 11, 0, 0, 0, time.Local), UpdatedAt: time.Now(),
			Extensions: J(`{"mode":"mysql","tenant_kind":"user","primary_zone":"ZONE1","locality":"F@ZONE1,F@ZONE2,F@ZONE3","unit_num":1,"max_cpu":6,"max_mem_gb":32,"whitelist":["10.41.0.%"],"units":[{"zone":"ZONE1","server_ip":"10.41.1.11"},{"zone":"ZONE2","server_ip":"10.41.2.11"},{"zone":"ZONE3","server_ip":"10.41.3.11"}]}`)}},
	}
}

// whitelistNodes 物理副本节点：PG 挂逻辑实例；OB observer 全部挂 sys 租户实例（横跨全部 server 的集群拓扑锚点）。
func whitelistNodes(inst map[string]int64) []*model.DbInstanceNode {
	node := func(key, role, hostName, hostIP string, port int, infra, os string) *model.DbInstanceNode {
		return &model.DbInstanceNode{InstanceID: inst[key], Ordinal: 0, Role: role, HostName: hostName,
			HostIP: hostIP, Port: port, HostEnvironment: "生产", HostInfraType: infra, OSName: os}
	}
	pg := []struct{ role, name, ip string }{
		{"primary", "pg-order-01", "10.20.2.11"}, {"secondary", "pg-order-02", "10.20.2.12"}, {"secondary", "pg-order-03", "10.20.2.13"},
	}
	pgr := []struct{ role, name, ip string }{
		{"primary", "pg-report-01", "10.20.2.21"}, {"secondary", "pg-report-02", "10.20.2.22"},
	}
	obc := []string{"10.40.1.11", "10.40.1.12", "10.40.2.11", "10.40.2.12", "10.40.3.11", "10.40.3.12"}
	obl := []string{"10.41.1.11", "10.41.2.11", "10.41.3.11"}

	var rows []*model.DbInstanceNode
	for i, h := range pg {
		n := node("prod-pg-order-01/pg-order-01", h.role, h.name, h.ip, 5432, "虚拟机", "CentOS 7.9")
		n.Ordinal = i
		rows = append(rows, n)
	}
	for i, h := range pgr {
		n := node("prod-pg-report-02/pg-report-01", h.role, h.name, h.ip, 5432, "虚拟机", "CentOS 7.9")
		n.Ordinal = i
		rows = append(rows, n)
	}
	for i, ip := range obc {
		n := node("prod-ob-core-01/sys", "observer", fmt.Sprintf("observer-zone%d-%02d", i/2+1, i%2+1), ip, 2881, "物理机", "CentOS 7.9")
		n.Ordinal = i
		rows = append(rows, n)
	}
	for i, ip := range obl {
		n := node("prod-ob-log-01/sys", "observer", fmt.Sprintf("obs-log-zone%d-01", i+1), ip, 2881, "虚拟机", "Ubuntu 22.04")
		n.Ordinal = i
		rows = append(rows, n)
	}
	return rows
}

func watermarkRows() []*model.DbSyncWatermark {
	now := time.Now()
	return []*model.DbSyncWatermark{
		{SourceSys: "ucmdb", LastSyncedAt: now, Cursor: J(`{"last_entity":"ucmdb-ent-004","page":3}`)},
		{SourceSys: "alert-sys", LastSyncedAt: now, Cursor: J(`{"last_event_time":"` + now.Add(-5*time.Minute).Format(time.RFC3339) + `"}`)},
		{SourceSys: "log-sys", LastSyncedAt: now, Cursor: J(`{"last_slow_query_id":98231}`)},
		{SourceSys: "change-sys", LastSyncedAt: now, Cursor: J(`{"last_ticket_no":"CHG-2026-0821-005"}`)},
	}
}

/* ---- 告警原始事件：复刻主种子 alert_records（同对象/标题/次数），源级别 Critical/Major/Minor ---- */

type alertSpec struct {
	object, otype, level, title string
	count                       int
	hour, min                   int
	day                         int
	cluster, instance          string // 元数据域键；空串 = 不关联
}

func alertRawRows(cl map[string]int64, inst map[string]int64) []*model.AlertRaw {
	specs := []alertSpec{
		{"trade_tenant @ prod-ob-core-01", "tenant", "Critical", "租户 CPU 13.1/14C（阈值 90%）", 6, 14, 32, 18, "prod-ob-core-01", "prod-ob-core-01/trade_tenant"},
		{"prod-pg-order-01（主库）", "cluster", "Critical", "连接数 962/1000 · 接近上限", 4, 11, 20, 18, "prod-pg-order-01", "prod-pg-order-01/pg-order-01"},
		{"pg-order-01（主库）", "instance", "Major", "备库复制延迟 850ms（阈值 300ms）", 3, 14, 18, 18, "prod-pg-order-01", "prod-pg-order-01/pg-order-01"},
		{"pay_tenant @ prod-ob-core-01", "tenant", "Major", "租户内存水位 91%", 2, 13, 55, 18, "prod-ob-core-01", "prod-ob-core-01/pay_tenant"},
		{"analytics @ prod-pg-order-01", "database", "Major", "连接数 410/500 · 慢查询堆积", 1, 12, 40, 18, "prod-pg-order-01", "prod-pg-order-01/pg-order-01"},
		{"observer-zone2-01", "host", "Major", "OBServer CPU 82% · 建议检查 Unit 均衡", 1, 11, 2, 18, "prod-ob-core-01", ""},
		{"host-10.20.2.12", "host", "Major", "磁盘使用率 72% · 持续上升趋势", 2, 22, 10, 17, "prod-pg-order-01", ""},
		{"prod-ob-log-01", "cluster", "Major", "major 合并耗时超过 2h", 1, 3, 40, 17, "prod-ob-log-01", ""},
		{"seckill @ t-trade", "database", "Minor", "慢 SQL 数量突增（+35%/小时）", 5, 10, 15, 18, "prod-ob-core-01", "prod-ob-core-01/trade_tenant"},
	}
	var rows []*model.AlertRaw
	seq := 0
	for _, sp := range specs {
		first := time.Date(2026, 8, sp.day, sp.hour, sp.min, 0, 0, time.Local)
		for k := 0; k < sp.count; k++ {
			seq++
			fired := first.Add(time.Duration(k*7) * time.Minute)
			var cid, iid *int64
			if sp.cluster != "" {
				cid = refOf(cl, sp.cluster)
			}
			if sp.instance != "" {
				iid = refOf(inst, sp.instance)
			}
			rows = append(rows, &model.AlertRaw{
				SourceSys: "alert-sys", EventID: fmt.Sprintf("alt-2026-%04d", seq),
				ObjectName: sp.object, ObjectType: sp.otype, ClusterID: cid, InstanceID: iid,
				AlertLevel: sp.level, AlertName: sp.title, AlertDesc: sp.title + "（来源：旧告警系统）",
				FiredAt: fired, StartTime: tp(fired.Add(-time.Minute)), Environment: "生产",
				CreateTime: tp(fired), UpdateTime: tp(fired),
				Raw: J(fmt.Sprintf(`{"bu":"核心交易","subsystem":"trade","device_type":"server","alert_recovery_time_minutes":%d,"shield_duration_minutes":0,"exemption_status":"none"}`, 5+k)),
			})
		}
	}
	return rows
}

/* ---- 变更工单 ---- */

func changeTicketRows(cl map[string]int64, inst map[string]int64) []*model.ChangeTicket {
	at := func(day, hour, min int) *time.Time { return tp(time.Date(2026, 8, day, hour, min, 0, 0, time.Local)) }
	return []*model.ChangeTicket{
		{SourceSys: "change-sys", TicketNo: "CHG-2026-0812-001", Title: "prod-pg-order-01 参数变更：shared_buffers 32G → 48G",
			StatusCode: 3, RiskLevel: "中", OwnerName: "王强",
			PlanStartAt: at(12, 1, 30), PlanEndAt: at(12, 3, 0), ExecuteStartAt: at(12, 2, 0), ExecuteEndAt: at(12, 2, 40), ExpectedStopAt: at(12, 2, 40),
			ClusterID: refOf(cl, "prod-pg-order-01"), ProjectID: "PRJ-1024", CreateTime: at(11, 10, 0), UpdateTime: at(12, 2, 45),
			Raw: J(`{"ticket_id":88101,"change_time":"low"}`)},
		{SourceSys: "change-sys", TicketNo: "CHG-2026-0815-002", Title: "prod-ob-core-01 trade_tenant Unit 扩容：14C → 20C",
			StatusCode: 1, RiskLevel: "高", OwnerName: "赵磊",
			PlanStartAt: at(22, 1, 0), PlanEndAt: at(22, 5, 0), ExpectedStopAt: at(22, 5, 0),
			ClusterID: refOf(cl, "prod-ob-core-01"), InstanceID: refOf(inst, "prod-ob-core-01/trade_tenant"),
			ProjectID: "PRJ-1024", CreateTime: at(15, 9, 0), UpdateTime: at(20, 14, 0),
			Raw: J(`{"ticket_id":88210,"change_time":"high"}`)},
		{SourceSys: "change-sys", TicketNo: "CHG-2026-0816-003", Title: "prod-ob-log-01 新建租户 log_tenant_etl（MySQL 模式）",
			StatusCode: 3, RiskLevel: "低", OwnerName: "赵磊",
			PlanStartAt: at(16, 14, 0), PlanEndAt: at(16, 15, 0), ExecuteStartAt: at(16, 14, 10), ExecuteEndAt: at(16, 14, 35),
			ClusterID: refOf(cl, "prod-ob-log-01"), InstanceID: refOf(inst, "prod-ob-log-01/log_tenant"),
			ProjectID: "PRJ-1088", CreateTime: at(15, 16, 0), UpdateTime: at(16, 14, 40),
			Raw: J(`{"ticket_id":88315,"change_time":"low"}`)},
		{SourceSys: "change-sys", TicketNo: "CHG-2026-0818-004", Title: "prod-pg-report-02 备库 pg-report-02 重建（WAL 积压超限）",
			StatusCode: 3, RiskLevel: "中", OwnerName: "王强",
			PlanStartAt: at(18, 2, 30), PlanEndAt: at(18, 5, 0), ExecuteStartAt: at(18, 3, 0), ExecuteEndAt: at(18, 4, 12),
			ClusterID: refOf(cl, "prod-pg-report-02"), ProjectID: "PRJ-1024", CreateTime: at(17, 20, 0), UpdateTime: at(18, 4, 20),
			Raw: J(`{"ticket_id":88402,"change_time":"mid"}`)},
		{SourceSys: "change-sys", TicketNo: "CHG-2026-0821-005", Title: "prod-ob-core-01 pay_tenant 连接白名单变更（新增 10.40.30.%）",
			StatusCode: 3, RiskLevel: "低", OwnerName: "陈静",
			PlanStartAt: at(21, 9, 45), PlanEndAt: at(21, 10, 30), ExecuteStartAt: at(21, 10, 0), ExecuteEndAt: at(21, 10, 15),
			ClusterID: refOf(cl, "prod-ob-core-01"), InstanceID: refOf(inst, "prod-ob-core-01/pay_tenant"),
			ProjectID: "PRJ-1102", CreateTime: at(20, 11, 0), UpdateTime: at(21, 10, 20),
			Raw: J(`{"ticket_id":88520,"change_time":"low"}`)},
	}
}

/* ---- 慢查询事件：每指纹多条执行记录，均值与主种子 slow_sqls 展示值一致 ---- */

type slowSpec struct {
	sqlText, db           string
	cnt                   int
	avgMs                 float64
	rowsExamined          int64
	instance, endpoint    string
	hostIP, hostname, user string
	port                  int
}

func slowQueryLogRows(inst map[string]int64) []*model.SlowQueryLog {
	specs := []slowSpec{
		{"SELECT o.*, u.name FROM trade_order o JOIN user u ON o.uid = u.id WHERE o.status = ?", "trade_tenant/trade_order",
			342, 12800, 4380012, "prod-ob-core-01/trade_tenant", "10.40.1.21:2881", "10.40.1.21", "observer-zone1-01", "trade_rw@trade_tenant", 2881},
		{"UPDATE stock_record SET qty = qty - ? WHERE sku_id = ? AND warehouse_id = ?", "trade_tenant/inventory",
			187, 9600, 1203550, "prod-ob-core-01/trade_tenant", "10.40.1.21:2881", "10.40.1.21", "observer-zone1-01", "trade_rw@trade_tenant", 2881},
		{"SELECT COUNT(*) FROM access_log WHERE create_time BETWEEN ? AND ? GROUP BY path", "log_tenant/access_log",
			96, 8200, 9881204, "prod-ob-log-01/log_tenant", "10.41.1.11:2881", "10.41.1.11", "obs-log-zone1-01", "log_rw@log_tenant", 2881},
		{"SELECT * FROM payment_bill WHERE bill_no LIKE ? ORDER BY ctime DESC LIMIT ?", "pay_tenant/payment",
			64, 6400, 760332, "prod-ob-core-01/pay_tenant", "10.40.2.12:2881", "10.40.2.12", "observer-zone2-02", "pay_rw@pay_tenant", 2881},
		{"DELETE FROM session_token WHERE expire_at < ? AND app_id IN (?, ?, ?)", "pg-order-01/auth",
			41, 5100, 2310778, "prod-pg-order-01/pg-order-01", "10.20.2.11:5432", "10.20.2.11", "pg-order-01", "app_rw", 5432},
	}
	base := time.Now().Add(-24 * time.Hour)
	var rows []*model.SlowQueryLog
	for i, sp := range specs {
		digest := fmt.Sprintf("%x", md5.Sum([]byte(sp.sqlText)))[:16]
		for k := 0; k < sp.cnt; k++ {
			ms := sp.avgMs
			switch {
			case k == sp.cnt-1 && sp.cnt%2 == 1: // 奇数条最后一条取均值，保证 avg 精确
			case k%2 == 0:
				ms += sp.avgMs * 0.18
			default:
				ms -= sp.avgMs * 0.18
			}
			rows = append(rows, &model.SlowQueryLog{
				SourceSys: "log-sys", InstanceID: refOf(inst, sp.instance),
				Endpoint: sp.endpoint, Hostname: sp.hostname, HostIP: sp.hostIP, Port: sp.port,
				DatabaseName: sp.db, Username: sp.user, SqlText: sp.sqlText, Digest: digest,
				ExecuteMs: ms, RowsExamined: ipid(sp.rowsExamined),
				ExecuteDate: base.Add(time.Duration(i*151+k*2) * time.Minute), CreateTime: time.Now(),
			})
		}
	}
	return rows
}

/* ---- 导入入口：各表独立「表空才导入」 ---- */

func loadClusterIDs(gdb *gorm.DB) map[string]int64 {
	var rows []*model.DbCluster
	gdb.Find(&rows)
	m := map[string]int64{}
	for _, r := range rows {
		m[r.Name] = r.ID
	}
	return m
}

// loadInstanceKeys 键与 whitelistInstances 的可读键一致（clusterName/instanceName）
func loadInstanceKeys(gdb *gorm.DB, cl map[string]int64) map[string]int64 {
	nameByCluster := map[int64]string{}
	for name, id := range cl {
		nameByCluster[id] = name
	}
	var rows []*model.DbInstance
	gdb.Find(&rows)
	m := map[string]int64{}
	for _, r := range rows {
		m[instanceKey(nameByCluster[r.ClusterID], r.Name)] = r.ID
	}
	return m
}

func RunWhitelist(gdb *gorm.DB) error {
	var n int64
	// 元数据三表：cluster → instance → node 依赖自增 ID，一次性导入
	if err := gdb.Model(&model.DbCluster{}).Count(&n).Error; err != nil {
		return err
	}
	if n == 0 {
		if err := gdb.Transaction(func(tx *gorm.DB) error {
			clusters := whitelistClusters()
			if err := tx.Create(&clusters).Error; err != nil {
				return err
			}
			cl := map[string]int64{}
			for _, c := range clusters {
				cl[c.Name] = c.ID
			}
			seeds := whitelistInstances(cl)
			inst := map[string]int64{}
			for _, s := range seeds {
				if err := tx.Create(s.row).Error; err != nil {
					return err
				}
				inst[s.key] = s.row.ID
			}
			return tx.CreateInBatches(whitelistNodes(inst), 100).Error
		}); err != nil {
			return err
		}
		log.Printf("[seed] whitelist metadata imported")
	}
	cl := loadClusterIDs(gdb)
	inst := loadInstanceKeys(gdb, cl)

	groups := []struct {
		name  string
		model interface{}
		rows  func() interface{}
	}{
		{"db_sync_watermark", &model.DbSyncWatermark{}, func() interface{} { return watermarkRows() }},
		{"alert_raw", &model.AlertRaw{}, func() interface{} { return alertRawRows(cl, inst) }},
		{"change_ticket", &model.ChangeTicket{}, func() interface{} { return changeTicketRows(cl, inst) }},
		{"slow_query_log", &model.SlowQueryLog{}, func() interface{} { return slowQueryLogRows(inst) }},
	}
	for _, g := range groups {
		if err := gdb.Model(g.model).Count(&n).Error; err != nil {
			return err
		}
		if n > 0 {
			continue
		}
		if err := gdb.CreateInBatches(g.rows(), 100).Error; err != nil {
			return err
		}
		log.Printf("[seed] whitelist %s imported", g.name)
	}
	return nil
}
