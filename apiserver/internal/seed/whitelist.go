package seed

import (
	"crypto/md5"
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"

	"db-cockpit/apiserver/internal/model"
)

/* ================= 元数据域 v2 + 数据面白名单演示种子 =================
与主种子（clusters 演示域）独立：各表「表空才导入」。
v2 结构（D16）：db_cluster（含端点）→ db_component（成员，双上游字段）→ db_host（全局主机）。
形状与前端拓扑原型（topoMock.ts）同构，正式接线零转换。 */

func tp(t time.Time) *time.Time { return &t }

func ipid(v int64) *int64 { return &v }

// refOf 映射取指针；键缺失返回 nil（关联列可空，缺失不该落 0 这种伪 ID）
func refOf(m map[string]int64, key string) *int64 {
	if v, ok := m[key]; ok {
		return &v
	}
	return nil
}

func whitelistHosts() []*model.DbHost {
	h := func(ip, name, az, hc, os, infra string) *model.DbHost {
		return &model.DbHost{HostIP: ip, HostName: name, Region: "华东", Az: az, HostCluster: hc,
			OsName: os, HostInfraType: infra, HostEnvironment: "生产", Status: "ok"}
	}
	return []*model.DbHost{
		// prod-pg-order-01（az-b/hc-1 主，az-c/hc-2 两备）
		h("10.20.2.11", "pg-order-01", "az-b", "hc-1", "CentOS 7.9", "虚拟机"),
		h("10.20.2.12", "pg-order-02", "az-c", "hc-2", "CentOS 7.9", "虚拟机"),
		h("10.20.2.13", "pg-order-03", "az-c", "hc-2", "CentOS 7.9", "虚拟机"),
		// prod-pg-report-02
		h("10.20.2.21", "pg-report-01", "az-b", "hc-1", "CentOS 7.9", "虚拟机"),
		h("10.20.2.22", "pg-report-02", "az-b", "hc-1", "CentOS 7.9", "虚拟机"),
		// prod-ob-core-01：obproxy（az-a/b）+ observer 三 Zone
		h("10.40.5.11", "obproxy-1", "az-a", "hc-1", "CentOS 7.9", "虚拟机"),
		h("10.40.5.12", "obproxy-2", "az-b", "hc-2", "CentOS 7.9", "虚拟机"),
		h("10.40.1.11", "observer-z1-1", "az-a", "hc-1", "CentOS 7.9", "物理机"),
		h("10.40.1.12", "observer-z1-2", "az-a", "hc-1", "CentOS 7.9", "物理机"),
		h("10.40.2.11", "observer-z2-1", "az-b", "hc-2", "CentOS 7.9", "物理机"),
		h("10.40.2.12", "observer-z2-2", "az-b", "hc-2", "CentOS 7.9", "物理机"),
		h("10.40.3.11", "observer-z3-1", "az-c", "hc-3", "CentOS 7.9", "物理机"),
		h("10.40.3.12", "observer-z3-2", "az-c", "hc-3", "CentOS 7.9", "物理机"),
		// prod-ob-log-01
		h("10.41.1.10", "obproxy-log", "az-a", "hc-3", "Ubuntu 22.04", "虚拟机"),
		h("10.41.1.11", "observer-lz1", "az-a", "hc-3", "Ubuntu 22.04", "虚拟机"),
		h("10.41.2.11", "observer-lz2", "az-b", "hc-3", "Ubuntu 22.04", "虚拟机"),
		h("10.41.3.11", "observer-lz3", "az-c", "hc-3", "Ubuntu 22.04", "虚拟机"),
	}
}

func whitelistClusters() []*model.DbCluster {
	now := time.Now()
	base := time.Date(2024, 3, 12, 10, 0, 0, 0, time.Local)
	return []*model.DbCluster{
		{Name: "prod-pg-order-01", Description: "交易核心集群 · 华东-可用区B", DbType: "pg", Environment: "生产",
			OrgCode: "ORG-CORE-TRADE", ServiceUser: "李明", OprDba: "王强", OprDbaIi: "赵磊", BusinessOwner: "陈锋",
			AlertSubscriber: "王强,赵磊", SubsysCode: "trade-order", SourceSys: "ucmdb", CcmName: "ccm-pg-order", LeName: "le-sh-east",
			HaType: "Patroni 流复制（1主2备）", BackupMethod: "pg_basebackup 每日全量 + WAL 归档", FailoverType: "Patroni 自动切换",
			IsCreatedByCloud: false, SourceID: "ucmdb-ent-001",
			Endpoint: "10.20.2.10:5432", Vip: "10.20.2.10", Port: 5432, Username: "app_rw", RoleSelector: "primary",
			CreatedAt: base, SyncedAt: tp(now), Extensions: J(`{"az":"华东-AZ-B","biz":"核心交易"}`)},
		{Name: "prod-pg-report-02", Description: "报表分析集群 · 华东-可用区B", DbType: "pg", Environment: "生产",
			OrgCode: "ORG-BI", ServiceUser: "周婷", OprDba: "赵磊", OprDbaIi: "王强", BusinessOwner: "吴楠",
			AlertSubscriber: "赵磊", SubsysCode: "bi-report", SourceSys: "ucmdb", CcmName: "ccm-pg-report", LeName: "le-sh-east",
			HaType: "流复制（1主1备）", BackupMethod: "pg_basebackup 每日全量", FailoverType: "手动提升",
			IsCreatedByCloud: false, SourceID: "ucmdb-ent-002",
			Endpoint: "10.20.2.20:5432", Vip: "10.20.2.20", Port: 5432, Username: "report_rw", RoleSelector: "primary",
			CreatedAt: base, SyncedAt: tp(now), Extensions: J(`{"az":"华东-AZ-B","biz":"报表分析"}`)},
		{Name: "prod-ob-core-01", Description: "核心账务集群 · 华东可用区A/B/C", DbType: "oceanbase", Environment: "生产",
			OrgCode: "ORG-CORE-TRADE", ServiceUser: "李明", OprDba: "王强", OprDbaIi: "赵磊", BusinessOwner: "陈锋",
			AlertSubscriber: "王强,赵磊", SubsysCode: "core-ledger", SourceSys: "ucmdb", CcmName: "ccm-ob-core", LeName: "le-sh-east",
			HaType: "3 Zone × 2 OBServer · Paxos", BackupMethod: "每日全量备份 + 日志归档", FailoverType: "Paxos 多数派自动切换",
			IsCreatedByCloud: true, SourceID: "ucmdb-ent-003",
			Endpoint: "10.40.5.100:2883", Vip: "10.40.5.100", Port: 2883, Username: "root", RoleSelector: "any",
			CreatedAt: base, SyncedAt: tp(now), Extensions: J(`{"zones":["ZONE1","ZONE2","ZONE3"],"biz":"核心账务"}`)},
		{Name: "prod-ob-log-01", Description: "日志分析集群 · 华东可用区A/B/C", DbType: "oceanbase", Environment: "生产",
			OrgCode: "ORG-BI", ServiceUser: "周婷", OprDba: "赵磊", OprDbaIi: "王强", BusinessOwner: "吴楠",
			AlertSubscriber: "赵磊", SubsysCode: "log-analytics", SourceSys: "ucmdb", CcmName: "ccm-ob-log", LeName: "le-sh-east",
			HaType: "3 Zone × 1 OBServer · Paxos", BackupMethod: "每日全量备份", FailoverType: "Paxos 多数派自动切换",
			IsCreatedByCloud: true, SourceID: "ucmdb-ent-004",
			Endpoint: "10.41.1.100:2883", Vip: "10.41.1.100", Port: 2883, Username: "root", RoleSelector: "any",
			CreatedAt: base, SyncedAt: tp(now), Extensions: J(`{"zones":["ZONE1","ZONE2","ZONE3"],"biz":"日志分析"}`)},
	}
}

// compSeed 成员种子：trafficKey/replKey 为上游成员键（clusterName/name），建行后回填引用
type compSeed struct {
	key      string
	row      *model.DbComponent
	trafficKey string
	replKey    string
}

// whitelistBaseComponents 引擎/代理成员（租户单独两阶段构建，units 需引用 observer id）
func whitelistBaseComponents() []compSeed {
	obCore, obLog := "prod-ob-core-01", "prod-ob-log-01"
	comp := func(key string, row *model.DbComponent) compSeed { return compSeed{key: key, row: row} }
	pg := func(cluster, name, role, host string, replTo string) compSeed {
		return compSeed{key: cluster + "/" + name, replKey: replTo,
			row: &model.DbComponent{Name: name, Kind: "storage", Role: role,
				Version: "PostgreSQL 15.6", Status: "ok", Port: 5432, HostIP: host,
				Extensions: J(`{"character_set":"UTF8","sync_mode":"quorum"}`), SourceID: "ucmdb-comp-" + name}}
	}
	seeds := []compSeed{
		// PG 复制链：两备 → 主（replKey 指向主成员）
		pg("prod-pg-order-01", "pg-order-01", "primary", "10.20.2.11", ""),
		pg("prod-pg-order-01", "pg-order-02", "secondary", "10.20.2.12", "prod-pg-order-01/pg-order-01"),
		pg("prod-pg-order-01", "pg-order-03", "secondary", "10.20.2.13", "prod-pg-order-01/pg-order-01"),
		pg("prod-pg-report-02", "pg-report-01", "primary", "10.20.2.21", ""),
		pg("prod-pg-report-02", "pg-report-02", "secondary", "10.20.2.22", "prod-pg-report-02/pg-report-01"),
		comp(obCore+"/obproxy-1", &model.DbComponent{Name: "obproxy-1", Kind: "proxy", Role: "active",
			Version: "OceanBase 4.2.1", Status: "ok", Port: 2883, HostIP: "10.40.5.11", SourceID: "ucmdb-comp-obp1"}),
		comp(obCore+"/obproxy-2", &model.DbComponent{Name: "obproxy-2", Kind: "proxy", Role: "active",
			Version: "OceanBase 4.2.1", Status: "ok", Port: 2883, HostIP: "10.40.5.12", SourceID: "ucmdb-comp-obp2"}),
		comp(obLog+"/obproxy-log", &model.DbComponent{Name: "obproxy-log", Kind: "proxy", Role: "active",
			Version: "OceanBase 4.2.1", Status: "ok", Port: 2883, HostIP: "10.41.1.10", SourceID: "ucmdb-comp-obpl"}),
	}
	// OB observer 成员：Paxos 多主，复制字段置空（按 Zone+role 渲染）
	ob := func(cluster, name, zone, host, status string) compSeed {
		return comp(cluster+"/"+name, &model.DbComponent{Name: name, Kind: "storage", GroupName: zone, Role: "observer",
			Version: "OceanBase 4.2.1", Status: status, Port: 2881, HostIP: host,
			Extensions: J(`{"paxos":true}`), SourceID: "ucmdb-comp-" + name})
	}
	seeds = append(seeds,
		ob(obCore, "observer-z1-1", "ZONE1", "10.40.1.11", "ok"), ob(obCore, "observer-z1-2", "ZONE1", "10.40.1.12", "ok"),
		ob(obCore, "observer-z2-1", "ZONE2", "10.40.2.11", "warn"), ob(obCore, "observer-z2-2", "ZONE2", "10.40.2.12", "ok"),
		ob(obCore, "observer-z3-1", "ZONE3", "10.40.3.11", "ok"), ob(obCore, "observer-z3-2", "ZONE3", "10.40.3.12", "ok"),
		ob(obLog, "observer-lz1", "ZONE1", "10.41.1.11", "ok"), ob(obLog, "observer-lz2", "ZONE2", "10.41.2.11", "ok"),
		ob(obLog, "observer-lz3", "ZONE3", "10.41.3.11", "ok"),
	)
	return seeds
}

// whitelistTenants 已由 buildTenants（两阶段引用 observer/obproxy id）取代
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
	cluster, instance           string // 元数据域键（instance → db_component）；空串 = 不关联
}

func alertRawRows(cl map[string]int64, comp map[string]int64) []*model.AlertRaw {
	specs := []alertSpec{
		{"trade_tenant @ prod-ob-core-01", "tenant", "Critical", "租户 CPU 13.1/14C（阈值 90%）", 6, 14, 32, 18, "prod-ob-core-01", "prod-ob-core-01/trade_tenant"},
		{"prod-pg-order-01（主库）", "cluster", "Critical", "连接数 962/1000 · 接近上限", 4, 11, 20, 18, "prod-pg-order-01", "prod-pg-order-01/pg-order-01"},
		{"pg-order-01（主库）", "member", "Major", "备库复制延迟 850ms（阈值 300ms）", 3, 14, 18, 18, "prod-pg-order-01", "prod-pg-order-01/pg-order-01"},
		{"pay_tenant @ prod-ob-core-01", "tenant", "Major", "租户内存水位 91%", 2, 13, 55, 18, "prod-ob-core-01", "prod-ob-core-01/pay_tenant"},
		{"analytics @ prod-pg-order-01", "database", "Major", "连接数 410/500 · 慢查询堆积", 1, 12, 40, 18, "prod-pg-order-01", "prod-pg-order-01/pg-order-01"},
		{"observer-z2-1", "host", "Major", "OBServer CPU 82% · 建议检查 Unit 均衡", 1, 11, 2, 18, "prod-ob-core-01", "prod-ob-core-01/observer-z2-1"},
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
			rows = append(rows, &model.AlertRaw{
				SourceSys: "alert-sys", EventID: fmt.Sprintf("alt-2026-%04d", seq),
				ObjectName: sp.object, ObjectType: sp.otype,
				ClusterID:  refOf(cl, sp.cluster),
				InstanceID: refOf(comp, sp.instance),
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

func changeTicketRows(cl map[string]int64, comp map[string]int64) []*model.ChangeTicket {
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
			ClusterID: refOf(cl, "prod-ob-core-01"), InstanceID: refOf(comp, "prod-ob-core-01/trade_tenant"),
			ProjectID: "PRJ-1024", CreateTime: at(15, 9, 0), UpdateTime: at(20, 14, 0),
			Raw: J(`{"ticket_id":88210,"change_time":"high"}`)},
		{SourceSys: "change-sys", TicketNo: "CHG-2026-0816-003", Title: "prod-ob-log-01 新建租户 log_tenant_etl（MySQL 模式）",
			StatusCode: 3, RiskLevel: "低", OwnerName: "赵磊",
			PlanStartAt: at(16, 14, 0), PlanEndAt: at(16, 15, 0), ExecuteStartAt: at(16, 14, 10), ExecuteEndAt: at(16, 14, 35),
			ClusterID: refOf(cl, "prod-ob-log-01"), InstanceID: refOf(comp, "prod-ob-log-01/log_tenant"),
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
			ClusterID: refOf(cl, "prod-ob-core-01"), InstanceID: refOf(comp, "prod-ob-core-01/pay_tenant"),
			ProjectID: "PRJ-1102", CreateTime: at(20, 11, 0), UpdateTime: at(21, 10, 20),
			Raw: J(`{"ticket_id":88520,"change_time":"low"}`)},
	}
}

/* ---- 慢查询事件：每指纹多条执行记录，均值与主种子 slow_sqls 展示值一致 ---- */

type slowSpec struct {
	sqlText, db            string
	cnt                    int
	avgMs                  float64
	rowsExamined           int64
	instance, endpoint     string
	hostIP, hostname, user string
	port                   int
}

func slowQueryLogRows(comp map[string]int64) []*model.SlowQueryLog {
	specs := []slowSpec{
		{"SELECT o.*, u.name FROM trade_order o JOIN user u ON o.uid = u.id WHERE o.status = ?", "trade_tenant/trade_order",
			342, 12800, 4380012, "prod-ob-core-01/trade_tenant", "10.40.5.100:2883", "10.40.1.11", "observer-z1-1", "trade_rw@trade_tenant", 2881},
		{"UPDATE stock_record SET qty = qty - ? WHERE sku_id = ? AND warehouse_id = ?", "trade_tenant/inventory",
			187, 9600, 1203550, "prod-ob-core-01/trade_tenant", "10.40.5.100:2883", "10.40.1.11", "observer-z1-1", "trade_rw@trade_tenant", 2881},
		{"SELECT COUNT(*) FROM access_log WHERE create_time BETWEEN ? AND ? GROUP BY path", "log_tenant/access_log",
			96, 8200, 9881204, "prod-ob-log-01/log_tenant", "10.41.1.100:2883", "10.41.1.11", "observer-lz1", "log_rw@log_tenant", 2881},
		{"SELECT * FROM payment_bill WHERE bill_no LIKE ? ORDER BY ctime DESC LIMIT ?", "pay_tenant/payment",
			64, 6400, 760332, "prod-ob-core-01/pay_tenant", "10.40.5.100:2883", "10.40.2.12", "observer-z2-2", "pay_rw@pay_tenant", 2881},
		{"DELETE FROM session_token WHERE expire_at < ? AND app_id IN (?, ?, ?)", "pg-order-01/auth",
			41, 5100, 2310778, "prod-pg-order-01/pg-order-01", "10.20.2.10:5432", "10.20.2.11", "pg-order-01", "app_rw", 5432},
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
				SourceSys: "log-sys", InstanceID: refOf(comp, sp.instance),
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

// loadComponentKeys 组件键：clusterName/name（跨集群同名成员需带集群名区分）
func loadComponentKeys(gdb *gorm.DB, cl map[string]int64) map[string]int64 {
	nameByCluster := map[int64]string{}
	for name, id := range cl {
		nameByCluster[id] = name
	}
	var rows []*model.DbComponent
	gdb.Find(&rows)
	m := map[string]int64{}
	for _, r := range rows {
		m[nameByCluster[r.ClusterID]+"/"+r.Name] = r.ID
	}
	return m
}

func RunWhitelist(gdb *gorm.DB) error {
	var n int64
	// 元数据三表：cluster → host → component（租户两阶段引用 observer/obproxy id）
	if err := gdb.Model(&model.DbCluster{}).Count(&n).Error; err != nil {
		return err
	}
	if n == 0 {
		if err := gdb.Transaction(func(tx *gorm.DB) error {
			clusters := whitelistClusters()
			if err := tx.Create(&clusters).Error; err != nil {
				return err
			}
			if err := tx.CreateInBatches(whitelistHosts(), 100).Error; err != nil {
				return err
			}
			cl := map[string]int64{}
			for _, c := range clusters {
				cl[c.Name] = c.ID
			}
			// 阶段 A：建基础成员行，收集键 → id
			seeds := whitelistBaseComponents()
			ids := map[string]int64{}
			for i := range seeds {
				row := seeds[i].row
				row.ClusterID = cl[keyCluster(seeds[i].key)]
				if err := tx.Create(row).Error; err != nil {
					return err
				}
				ids[seeds[i].key] = row.ID
			}
			// 阶段 B：回填双上游引用（复制链 / 数据流）
			for i := range seeds {
				if seeds[i].replKey != "" {
					if err := tx.Model(seeds[i].row).Update("replication_upstream_id", refOf(ids, seeds[i].replKey)).Error; err != nil {
						return err
					}
				}
			}
			// 阶段 C：租户逻辑单元（traffic→obproxy，units→observer）
			tenants := buildTenants(func(key string) (int64, bool) {
				v, ok := ids[key]
				return v, ok
			})
			for _, t := range tenants {
				t.row.ClusterID = cl[t.clusterName]
				if err := tx.Create(t.row).Error; err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			return err
		}
		log.Printf("[seed] whitelist metadata v2 imported")
	}
	cl := loadClusterIDs(gdb)
	comp := loadComponentKeys(gdb, cl)

	groups := []struct {
		name  string
		model interface{}
		rows  func() interface{}
	}{
		{"db_sync_watermark", &model.DbSyncWatermark{}, func() interface{} { return watermarkRows() }},
		{"alert_raw", &model.AlertRaw{}, func() interface{} { return alertRawRows(cl, comp) }},
		{"change_ticket", &model.ChangeTicket{}, func() interface{} { return changeTicketRows(cl, comp) }},
		{"slow_query_log", &model.SlowQueryLog{}, func() interface{} { return slowQueryLogRows(comp) }},
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

func keyCluster(key string) string {
	for i := len(key) - 1; i >= 0; i-- {
		if key[i] == '/' {
			return key[:i]
		}
	}
	return key
}

type tenantSeed struct {
	clusterName string
	row         *model.DbComponent
}

func buildTenants(idOf func(string) (int64, bool)) []*tenantSeed {
	tenant := func(cluster, name, trafficKey string, maxCpu int, units ...string) *tenantSeed {
		unitJSON := "["
		first := true
		for i, u := range units {
			if id, ok := idOf(u); ok {
				if !first {
					unitJSON += ","
				}
				first = false
				unitJSON += fmt.Sprintf(`{"instance_id":%d,"zone":"ZONE%d"}`, id, i+1)
			}
		}
		unitJSON += "]"
		var traffic *int64
		if id, ok := idOf(trafficKey); ok {
			traffic = &id
		}
		return &tenantSeed{clusterName: cluster, row: &model.DbComponent{Name: name, Kind: "tenant", Role: "user",
			Version: "OceanBase 4.2.1", Status: "ok", Port: 2881, TrafficUpstreamID: traffic,
			Extensions: J(fmt.Sprintf(`{"mode":"mysql","tenant_kind":"user","unit_num":1,"max_cpu":%d,"max_mem_gb":%d,"whitelist":["10.40.10.%%"],"units":%s}`, maxCpu, maxCpu*5+2, unitJSON)),
			SourceID: "ucmdb-comp-" + name}}
	}
	return []*tenantSeed{
		tenant("prod-ob-core-01", "trade_tenant", "prod-ob-core-01/obproxy-1", 14,
			"prod-ob-core-01/observer-z1-1", "prod-ob-core-01/observer-z2-1", "prod-ob-core-01/observer-z3-1"),
		tenant("prod-ob-core-01", "pay_tenant", "prod-ob-core-01/obproxy-2", 12,
			"prod-ob-core-01/observer-z1-2", "prod-ob-core-01/observer-z2-2", "prod-ob-core-01/observer-z3-2"),
		tenant("prod-ob-log-01", "log_tenant", "prod-ob-log-01/obproxy-log", 6,
			"prod-ob-log-01/observer-lz1", "prod-ob-log-01/observer-lz2", "prod-ob-log-01/observer-lz3"),
	}
}
