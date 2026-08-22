package model

import (
	"time"

	"gorm.io/datatypes"
)

/* ================= 元数据域（§6.1 · 数据面白名单表：collector 直写 / apiserver 只读） ================= */

// DbCluster 集群/实体层（KubeBlocks Cluster / §6.1 CLUSTER）。
// 由 instance_meta 的集群级字段拆出：环境、组织归属、HA/备份/切换模式。
// 建表基准见 deploy/db/001_db_metadata.sql。
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
	CreatedAt        time.Time      `json:"createdAt"` // instance_meta.created_date
	SyncedAt         *time.Time     `json:"syncedAt"`  // collector 最近同步时间（行级水位）
	Extensions       datatypes.JSON `json:"extensions,omitempty"` // 各库类型差异扩展
}

func (DbCluster) TableName() string { return "db_cluster" }

// DbInstance 逻辑实例层（KubeBlocks Component+InstanceSet / §6.1 INSTANCE）。
// 回答「这是什么库、什么版本、从哪连」；应用连接 vip/endpoint，与所在主机解耦。
type DbInstance struct {
	ID           int64          `gorm:"primaryKey" json:"id"`
	ClusterID    int64          `gorm:"column:cluster_id;index:idx_db_instance_cluster" json:"clusterId"` // 逻辑外键 → db_cluster.id
	DbType       string         `gorm:"column:db_type;size:32;index:idx_db_instance_db_type" json:"dbType"` // 实例实际引擎，诊断工具按此路由
	Name         string         `gorm:"size:128" json:"name"`       // instance_meta.instance_name
	Version      string         `gorm:"size:64" json:"version"`     // instance_meta.version_detail
	Status       string         `gorm:"size:32" json:"status"`
	Role         string         `gorm:"size:32" json:"role"`        // 组件角色：storage/proxy…
	CharacterSet string         `gorm:"size:64" json:"characterSet"`
	InfraType    string         `gorm:"size:32" json:"infraType"`
	ReqCPU       float64        `gorm:"column:req_cpu" json:"reqCpu"`
	ReqMemoryGb  float64        `gorm:"column:req_memory_gb" json:"reqMemoryGb"`
	ReqStorageGb float64        `gorm:"column:req_storage_gb" json:"reqStorageGb"`
	AttachDB     string         `gorm:"column:attach_db;size:256" json:"attachDb"`
	Endpoint     string         `gorm:"size:256" json:"endpoint"`    // instance_meta.instance_endpoint
	Vip          string         `gorm:"size:64" json:"vip"`          // instance_meta.instance_vip
	Port         int            `json:"port"`                        // instance_meta.instance_port
	Username     string         `gorm:"size:64" json:"username"`     // instance_meta.user_name
	RoleSelector string         `gorm:"size:32" json:"roleSelector"` // 端点路由角色：主/任意副本
	SourceID     string         `gorm:"column:source_id;size:128;uniqueIndex" json:"sourceId"` // instance_meta.ins_uuid，collector 幂等去重键
	CreatedAt    time.Time      `json:"createdAt"`                   // instance_meta.ins_created_date
	UpdatedAt    time.Time      `json:"updatedAt"`                   // instance_meta.ins_updated_date
	Extensions   datatypes.JSON `json:"extensions,omitempty"`        // 各库类型差异字段
}

func (DbInstance) TableName() string { return "db_instance" }

// DbInstanceNode 物理副本节点层（KubeBlocks Instance / §6.1 INSTANCE_NODE）。
// 回答「实例跑在哪些主机、各自角色」；instance_meta 的 host_*1/host_*2 成对列拆成 N 行，
// 主备切换只改本表 role，实例身份不变。
type DbInstanceNode struct {
	ID              int64  `gorm:"primaryKey" json:"id"`
	InstanceID      int64  `gorm:"column:instance_id;uniqueIndex:idx_db_instance_node_ord,priority:1" json:"instanceId"` // 逻辑外键 → db_instance.id
	Ordinal         int    `gorm:"uniqueIndex:idx_db_instance_node_ord,priority:2" json:"ordinal"` // host_*1→0、host_*2→1、…
	Role            string `gorm:"size:32" json:"role"`             // primary/secondary/arbiter
	HostName        string `gorm:"size:128" json:"hostName"`        // instance_meta.host_name1/2
	HostIP          string `gorm:"column:host_ip;size:64;index:idx_db_instance_node_host_ip" json:"hostIp"` // instance_meta.host_ip1/2
	Port            int    `json:"port"`                            // 节点端口（缺省取实例端口）
	HostEnvironment string `gorm:"size:32" json:"hostEnvironment"`  // instance_meta.host_environment1/2
	HostInfraType   string `gorm:"size:32" json:"hostInfraType"`    // instance_meta.host_infra_type1/2
	OSName          string `gorm:"column:os_name;size:64" json:"osName"` // instance_meta.os_name
}

func (DbInstanceNode) TableName() string { return "db_instance_node" }

// DbSyncWatermark collector 同步水位（§6.1 SYNC_WATERMARK）：按来源系统记录断点，幂等续传。
type DbSyncWatermark struct {
	ID           int64          `gorm:"primaryKey" json:"id"`
	SourceSys    string         `gorm:"column:source_sys;size:64;uniqueIndex" json:"sourceSys"`
	LastSyncedAt time.Time      `gorm:"column:last_synced_at" json:"lastSyncedAt"`
	Cursor       datatypes.JSON `json:"cursor"` // 断点游标（各来源自定义：时间戳/分页/位点）
}

func (DbSyncWatermark) TableName() string { return "db_sync_watermark" }

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
