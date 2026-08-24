package model

import (
	"time"

	"gorm.io/datatypes"
)

/* ================= 元数据域 v2（§6.1.1 · 数据面白名单表：collector 直写 / apiserver 只读） ================= */

// DbCluster 集群层（逻辑包含顶层 + 服务入口）。端点上移自 v1 的 db_instance：
// 服务入口（VIP/接入串）属于集群/服务，不属于单个成员——与 KubeBlocks Service on Component 同构。
type DbCluster struct {
	ID               int64          `gorm:"primaryKey" json:"id"`
	Name             string         `gorm:"size:128;index:idx_db_cluster_name" json:"name"` // instance_meta.entity_name
	Description      string         `gorm:"size:512" json:"description"`                    // instance_meta.chinese_desc
	DbType           string         `gorm:"column:db_type;size:32;index:idx_db_cluster_db_type" json:"dbType"` // 主库类型，大盘按类型过滤
	Environment      string         `gorm:"size:32;index:idx_db_cluster_environment" json:"environment"`
	OrgCode          string         `gorm:"size:64" json:"orgCode"`
	ServiceUser      string         `gorm:"size:128" json:"serviceUser"`
	OprDba           string         `gorm:"size:64" json:"oprDba"`
	OprDbaIi         string         `gorm:"size:64" json:"oprDbaIi"`
	BusinessOwner    string         `gorm:"size:128" json:"businessOwner"`
	AlertSubscriber  string         `gorm:"size:256" json:"alertSubscriber"`
	SubsysCode       string         `gorm:"size:64" json:"subsysCode"`
	SourceSys        string         `gorm:"size:64" json:"sourceSys"`
	CcmName          string         `gorm:"size:128" json:"ccmName"`
	LeName           string         `gorm:"size:128" json:"leName"`
	HaType           string         `gorm:"size:64" json:"haType"`
	BackupMethod     string         `gorm:"size:64" json:"backupMethod"`
	FailoverType     string         `gorm:"size:64" json:"failoverType"`
	IsCreatedByCloud bool           `json:"isCreatedByCloud"`
	SourceID         string         `gorm:"column:source_id;size:128" json:"sourceId"` // 外部集群级唯一 ID（去重）
	Endpoint         string         `gorm:"size:256" json:"endpoint"`                  // 服务入口（数据流拓扑起点）
	Vip              string         `gorm:"size:64" json:"vip"`
	Port             int            `json:"port"`
	Username         string         `gorm:"size:64" json:"username"`
	RoleSelector     string         `gorm:"size:32" json:"roleSelector"` // 入口路由角色：primary/any
	CreatedAt        time.Time      `json:"createdAt"` // instance_meta.created_date
	SyncedAt         *time.Time     `json:"syncedAt"`  // collector 最近同步时间（行级水位）
	Extensions       datatypes.JSON `json:"extensions,omitempty"` // 各库类型差异扩展（集群级）
}

func (DbCluster) TableName() string { return "db_cluster" }

// DbComponent 组件成员统一表（v2 核心，D16）：一行 = 集群的一个成员/逻辑单元——
// 引擎成员（pg 节点/observer）、代理成员（obproxy/mongos/haproxy）、租户逻辑单元。
// 关系零关系表、双自引用字段串联：
//   - traffic_upstream_id     数据流上游（纵向·实线）：存储成员指向其前面的 proxy/access 组件
//   - replication_upstream_id 复制上游（横向·虚线）：备→主/级联；Paxos/多主置空（按 zone+role 渲染）
//   - 租户 N:M 落位走 extensions.units（[{instance_id → db_component.id, zone}]）
// 语义约束（应用层校验）：traffic 边只指向 proxy/access/compute 类；replication 边仅在引擎成员间。
type DbComponent struct {
	ID                   int64          `gorm:"primaryKey" json:"id"`
	ClusterID            int64          `gorm:"column:cluster_id;index:idx_db_component_cluster" json:"clusterId"`
	Name                 string         `gorm:"size:128" json:"name"`
	Kind                 string         `gorm:"size:16" json:"kind"`      // storage/proxy/compute/tenant/access/arbiter
	GroupName            string         `gorm:"column:group_name;size:64" json:"groupName"` // 分组：shard-1/ZONE1…（可空）
	Role                 string         `gorm:"size:32" json:"role"`                         // primary/secondary/observer/active…
	Version              string         `gorm:"size:64" json:"version"`
	Status               string         `gorm:"size:16" json:"status"`
	Port                 int            `json:"port"`
	HostIP               string         `gorm:"column:host_ip;size:64;index:idx_db_component_host" json:"hostIp"` // → db_host（逻辑单元可空）
	TrafficUpstreamID    *int64         `gorm:"column:traffic_upstream_id" json:"trafficUpstreamId,omitempty"`
	ReplicationUpstreamID *int64        `gorm:"column:replication_upstream_id" json:"replicationUpstreamId,omitempty"`
	Extensions           datatypes.JSON `json:"extensions,omitempty"` // 租户 mode/unit/whitelist/units 落位、delay_ms、paxos、路由规则
	SourceID             string         `gorm:"column:source_id;size:128" json:"sourceId"`
	CreatedAt            time.Time      `json:"createdAt"`
	UpdatedAt            time.Time      `json:"updatedAt"`
}

func (DbComponent) TableName() string { return "db_component" }

// DbHost 独立全局主机表（D16）：位置（region/AZ/主机集群）唯一存储点；
// 组件成员经 host_ip 挂载（同机多进程天然表达），物理拓扑直接按三列分组。
type DbHost struct {
	HostIP         string         `gorm:"column:host_ip;primaryKey;size:64" json:"hostIp"`
	HostName       string         `gorm:"size:128" json:"hostName"`
	Region         string         `gorm:"size:32" json:"region"`
	Az             string         `gorm:"column:az;size:32;index:idx_db_host_az" json:"az"`
	HostCluster    string         `gorm:"column:host_cluster;size:32;index:idx_db_host_hc" json:"hostCluster"`
	OsName         string         `gorm:"column:os_name;size:64" json:"osName"`
	HostInfraType  string         `gorm:"size:32" json:"hostInfraType"`  // 物理机/虚拟机/容器
	HostEnvironment string        `gorm:"size:32" json:"hostEnvironment"`
	Status         string         `gorm:"size:16" json:"status"`
	Extensions     datatypes.JSON `json:"extensions,omitempty"`
}

func (DbHost) TableName() string { return "db_host" }

// DbSyncWatermark collector 同步水位（§6.1 SYNC_WATERMARK）：按来源系统记录断点，幂等续传。
type DbSyncWatermark struct {
	ID           int64          `gorm:"primaryKey" json:"id"`
	SourceSys    string         `gorm:"column:source_sys;size:64;uniqueIndex" json:"sourceSys"`
	LastSyncedAt time.Time      `gorm:"column:last_synced_at" json:"lastSyncedAt"`
	Cursor       datatypes.JSON `json:"cursor"` // 断点游标（各来源自定义：时间戳/分页/位点）
}

/* ================= 数据面白名单表（§6.1.2 · collector 直写 / apiserver 只读聚合） ================= */

// AlertRaw 原始告警事件（旧告警系统）：每行一条事件；聚合（对象+级别+标题 → 次数/首触时间）
// 与 P1/P2/P3 映射由消费端完成，Issue 化在控制面 Issue 域。建表基准 deploy/db/002_whitelist.sql。
type AlertRaw struct {
	ID          int64          `gorm:"primaryKey" json:"id"`
	SourceSys   string         `gorm:"column:source_sys;size:64" json:"sourceSys"`
	EventID     string         `gorm:"column:event_id;size:128;uniqueIndex" json:"eventId"` // 外部唯一 ID
	ObjectName  string         `gorm:"size:256" json:"objectName"`                          // alerts.resource/event_name
	ObjectType  string         `gorm:"size:32" json:"objectType"`                           // host/tenant/instance/cluster
	InstanceID  *int64         `gorm:"column:instance_id;index:idx_alert_raw_instance" json:"instanceId,omitempty"`
	ClusterID   *int64         `gorm:"column:cluster_id" json:"clusterId,omitempty"`
	AlertLevel  string         `gorm:"size:32" json:"alertLevel"` // 原值 Critical/Major/Minor…
	AlertName   string         `gorm:"size:256" json:"alertName"` // 告警标题（聚合展示）
	AlertDesc   string         `json:"alertDesc"`
	FiredAt     time.Time      `gorm:"column:fired_at;index:idx_alert_raw_fired" json:"firedAt"` // 主时间口径
	StartTime   *time.Time     `json:"startTime,omitempty"`
	EndTime     *time.Time     `json:"endTime,omitempty"` // 恢复时间
	Environment string         `gorm:"size:32" json:"environment"`
	CreateTime  *time.Time     `json:"createTime,omitempty"`
	UpdateTime  *time.Time     `json:"updateTime,omitempty"`
	Raw         datatypes.JSON `json:"raw,omitempty"` // 源系统整行原样
}

func (AlertRaw) TableName() string { return "alert_raw" }

// ChangeTicket 变更工单（旧变更系统）：供诊断关联（某实例某时间窗内是否有变更）。
type ChangeTicket struct {
	ID             int64          `gorm:"primaryKey" json:"id"`
	SourceSys      string         `gorm:"column:source_sys;size:64" json:"sourceSys"`
	TicketNo       string         `gorm:"size:128;uniqueIndex" json:"ticketNo"` // 外部唯一 ID
	Title          string         `gorm:"size:512" json:"title"`
	StatusCode     int            `gorm:"column:status_code" json:"statusCode"` // 源系统数字枚举，语义映射由消费端完成
	RiskLevel      string         `gorm:"column:risk_level;size:32" json:"riskLevel"`
	OwnerName      string         `gorm:"size:128" json:"ownerName"`
	PlanStartAt    *time.Time     `gorm:"column:plan_start_at" json:"planStartAt,omitempty"`
	PlanEndAt      *time.Time     `gorm:"column:plan_end_at" json:"planEndAt,omitempty"`
	ExecuteStartAt *time.Time     `gorm:"column:execute_start_at;index:idx_change_ticket_execute" json:"executeStartAt,omitempty"`
	ExecuteEndAt   *time.Time     `gorm:"column:execute_end_at" json:"executeEndAt,omitempty"`
	ExpectedStopAt *time.Time     `gorm:"column:expected_stop_at" json:"expectedStopAt,omitempty"`
	InstanceID     *int64         `gorm:"column:instance_id;index:idx_change_ticket_instance" json:"instanceId,omitempty"`
	ClusterID      *int64         `gorm:"column:cluster_id" json:"clusterId,omitempty"`
	ProjectID      string         `gorm:"column:project_id;size:128" json:"projectId"`
	CreateTime     *time.Time     `json:"createTime,omitempty"`
	UpdateTime     *time.Time     `json:"updateTime,omitempty"`
	Raw            datatypes.JSON `json:"raw,omitempty"`
}

func (ChangeTicket) TableName() string { return "change_ticket" }

// SlowQueryLog 慢查询原始事件（旧日志系统）：每行一次执行；指纹级聚合（平均耗时/扫描行数/执行次数）
// 由消费端 GROUP BY digest 完成；实时慢日志快照走 remote 通道，不落本表。
type SlowQueryLog struct {
	ID           int64         `gorm:"primaryKey" json:"id"`
	SourceSys    string        `gorm:"column:source_sys;size:64" json:"sourceSys"`
	InstanceID   *int64        `gorm:"column:instance_id;index:idx_slow_query_log_inst_time" json:"instanceId,omitempty"`
	Endpoint     string        `gorm:"size:256" json:"endpoint"`
	Hostname     string        `gorm:"size:128" json:"hostname"`
	HostIP       string        `gorm:"column:host_ip;size:64" json:"hostIp"`
	Port         int           `json:"port"`
	DatabaseName string        `gorm:"size:128" json:"databaseName"` // OB 为 tenant/db 复合
	Username     string        `gorm:"size:64" json:"username"`
	SqlText      string        `gorm:"column:sql_text" json:"sqlText"`
	Digest       string        `gorm:"size:64;index:idx_slow_query_log_digest" json:"digest"` // 指纹（聚合键）
	ExecuteMs    float64       `gorm:"column:execute_ms" json:"executeMs"`
	RowsExamined *int64        `gorm:"column:rows_examined" json:"rowsExamined,omitempty"` // 源缺失为 NULL → 展示 —
	ExecuteDate  time.Time     `gorm:"column:execute_date;index:idx_slow_query_log_inst_time,priority:2" json:"executeDate"`
	CreateTime   time.Time     `gorm:"column:create_time" json:"createTime"`
}

func (SlowQueryLog) TableName() string { return "slow_query_log" }
