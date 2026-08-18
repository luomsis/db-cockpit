package model

import (
	"gorm.io/datatypes"
)

/* ================= 集群 / 实例 / 参数 ================= */

type Cluster struct {
	ID       string `gorm:"primaryKey;size:32" json:"id"`
	Name     string `gorm:"size:128" json:"name"`
	Type     string `gorm:"size:32;index" json:"type"`
	Version  string `gorm:"size:64" json:"version"`
	Desc     string `json:"desc"`
	AZ       string `gorm:"size:64" json:"az"`
	Biz      string `gorm:"size:64" json:"biz"`
	Nodes    int    `json:"nodes"`
	Mode     string `gorm:"size:128" json:"mode"`
	CPU      int    `json:"cpu"`
	Mem      int    `json:"mem"`
	Conn     int    `json:"conn"`
	QPS      int    `json:"qps"`
	SyncMode *string `gorm:"size:128" json:"syncMode,omitempty"`
	Zones    datatypes.JSON `json:"zones,omitempty"`
}

type Instance struct {
	ID        string  `gorm:"primaryKey;size:32" json:"id"`
	ClusterID string  `gorm:"size:32;index" json:"-"`
	Name      string  `gorm:"size:128" json:"name"`
	Role      string  `gorm:"size:64" json:"role"`
	IP        string  `gorm:"size:64" json:"ip"`
	Port      int     `json:"port"`
	Status    string  `gorm:"size:16" json:"status"`
	CPU       int     `json:"cpu"`
	Mem       int     `json:"mem"`
	Conn      int     `json:"conn"`
	Ver       string  `gorm:"size:32" json:"ver"`
	Zone      *string `gorm:"size:32" json:"zone,omitempty"`
	HostIP    string  `gorm:"size:64;index" json:"-"`
}

type ClusterParam struct {
	ID        uint    `gorm:"primaryKey" json:"-"`
	Scope     string  `gorm:"size:16;uniqueIndex:idx_param_scope,priority:1" json:"-"` // cluster | tenant
	ClusterID string  `gorm:"size:32;uniqueIndex:idx_param_scope,priority:2" json:"-"`
	TenantID  *string `gorm:"size:32;uniqueIndex:idx_param_scope,priority:3" json:"-"`
	Name      string  `gorm:"size:128;uniqueIndex:idx_param_scope,priority:4" json:"name"`
	Value     string  `gorm:"size:256" json:"value"`
	Range     string  `gorm:"column:param_range;size:128" json:"range"`
	Desc      string  `gorm:"column:param_desc;size:256" json:"desc"`
	Status    string  `gorm:"size:16" json:"status"`
}

type ParamHistory struct {
	ID        uint   `gorm:"primaryKey" json:"-"`
	ParamID   uint   `gorm:"index" json:"-"`
	OldValue  string `json:"oldValue"`
	NewValue  string `json:"newValue"`
	ChangedAt int64  `json:"changedAt"`
}

/* ================= PostgreSQL 扩展 ================= */

type PgDatabase struct {
	ID        uint   `gorm:"primaryKey" json:"-"`
	ClusterID string `gorm:"size:32;uniqueIndex:idx_pgdb_cluster_name,priority:1" json:"-"`
	Name      string `gorm:"size:128;uniqueIndex:idx_pgdb_cluster_name,priority:2" json:"name"`
	Owner     string `gorm:"size:128" json:"owner"`
	Size      string `gorm:"size:32" json:"size"`
	Tables    int    `json:"tables"`
	Conn      int    `json:"conn"`
	ConnLimit int    `json:"connLimit"`
	Status    string `gorm:"size:16" json:"status"`
}

type PgReplica struct {
	ID        uint   `gorm:"primaryKey" json:"-"`
	ClusterID string `gorm:"size:32;index" json:"-"`
	Instance  string `gorm:"size:128" json:"instance"`
	Role      string `gorm:"size:64" json:"role"`
	DelayMs   int    `json:"delayMs"`
	WalLag    string `gorm:"size:32" json:"walLag"`
	Status    string `gorm:"size:16" json:"status"`
}

/* ================= OceanBase 租户体系 ================= */

type ObTenant struct {
	ID           string          `gorm:"primaryKey;size:32" json:"id"`
	ClusterID    string          `gorm:"size:32;index" json:"-"`
	Name         string          `gorm:"size:128" json:"name"`
	Kind         string          `gorm:"size:16" json:"kind"`
	Mode         string          `gorm:"column:mode;size:16" json:"mode"`
	PrimaryZone  string          `gorm:"size:64" json:"primaryZone"`
	Locality     string          `gorm:"size:128" json:"locality"`
	UnitNum      int             `json:"unitNum"`
	MaxCpu       float64         `json:"maxCpu"`
	UsedCpu      float64         `json:"usedCpu"`
	MaxMemGb     float64         `json:"maxMemGb"`
	UsedMemGb    float64         `json:"usedMemGb"`
	StorageUsed  string          `gorm:"size:32" json:"storageUsed"`
	StorageTotal string          `gorm:"size:32" json:"storageTotal"`
	Units        datatypes.JSON  `json:"units"`
	Whitelist    datatypes.JSON  `json:"whitelist"`
	ConnHint     string          `gorm:"size:256" json:"connHint"`
	Status       string          `gorm:"size:16" json:"status"`
}

type ObTenantDb struct {
	ID       uint   `gorm:"primaryKey" json:"-"`
	TenantID string `gorm:"size:32;uniqueIndex:idx_obdb_tenant_name,priority:1" json:"-"`
	Name     string `gorm:"size:128;uniqueIndex:idx_obdb_tenant_name,priority:2" json:"name"`
	Tables   int    `json:"tables"`
	Size     string `gorm:"size:32" json:"size"`
	Conn     int    `json:"conn"`
	Status   string `gorm:"size:16" json:"status"`
}

/* ================= 主机 / 账号 / 会话 / 事务 / 慢SQL ================= */

type Host struct {
	IP        string `gorm:"primaryKey;size:64" json:"ip"`
	Zone      string `gorm:"size:64" json:"zone"`
	Spec      string `gorm:"size:64" json:"spec"`
	OS        string `gorm:"column:os;size:64" json:"os"`
	CPU       int    `json:"cpu"`
	Mem       int    `json:"mem"`
	Disk      int    `json:"disk"`
	DiskTotal int    `json:"diskTotal"`
	ClusterID string `gorm:"size:32" json:"-"`
	Status    string `gorm:"size:16" json:"status"`
}

type InstanceUser struct {
	ID         uint   `gorm:"primaryKey" json:"-"`
	InstanceID string `gorm:"size:32;index" json:"-"` // '' = 全局演示数据
	Username   string `gorm:"size:128" json:"user"`
	Host       string `gorm:"size:64" json:"host"`
	Priv       string `gorm:"size:256" json:"priv"`
	LastLogin  string `gorm:"size:32" json:"lastLogin"`
	Status     string `gorm:"size:16" json:"status"`
}

type RuntimeSession struct {
	SessionID  int64  `gorm:"column:id;primaryKey" json:"id"`
	InstanceID string `gorm:"size:32;index" json:"-"`
	Username   string `gorm:"size:128" json:"user"`
	Host       string `gorm:"size:64" json:"host"`
	Db         string `gorm:"size:128" json:"db"`
	Cmd        string `gorm:"size:32" json:"cmd"`
	Time       string `gorm:"size:32" json:"time"`
	State      string `gorm:"size:64" json:"state"`
	LockInfo   string `gorm:"column:lock_info;size:64" json:"lock"`
	Status     string `gorm:"size:16" json:"status"`
}

type Trx struct {
	ID         string `gorm:"primaryKey;size:32" json:"id"`
	InstanceID string `gorm:"size:32;index" json:"-"`
	Session    int64  `json:"session"`
	Username   string `gorm:"size:128" json:"user"`
	Dur        string `gorm:"size:32" json:"dur"`
	Undo       string `gorm:"size:32" json:"undo"`
	LockRows   string `gorm:"size:32" json:"lockRows"`
	Waiting    string `gorm:"size:8" json:"waiting"`
	Sql        string `gorm:"column:sql_text" json:"sql"`
	Status     string `gorm:"size:16" json:"status"`
}

type SlowSql struct {
	ID         uint   `gorm:"primaryKey" json:"-"`
	InstanceID string `gorm:"size:32;index" json:"-"` // '' = 全局
	Sql        string `gorm:"column:sql_text" json:"sql"`
	Db         string `gorm:"size:128" json:"db"`
	Time       string `gorm:"size:16" json:"time"`
	Rows       string `gorm:"column:rows_scanned;size:32" json:"rows"`
	Count      int    `json:"count"`
}

/* ================= 报告 / 告警 / 指标 / 元统计 ================= */

type Report struct {
	ID        uint   `gorm:"primaryKey" json:"-"`
	ClusterID string `gorm:"size:32;index" json:"-"` // '' = 平台级
	Ico       string `gorm:"size:8" json:"ico"`
	Title     string `gorm:"size:128" json:"title"`
	Desc      string `gorm:"size:256" json:"desc"`
	Date      string `gorm:"size:32" json:"date"`
	Size      string `gorm:"size:16" json:"size"`
}

type AlertRecord struct {
	ID       uint   `gorm:"primaryKey" json:"-"`
	Name     string `gorm:"size:128" json:"name"`
	Severity string `gorm:"size:8" json:"severity"`
	Title    string `gorm:"size:256" json:"title"`
	Time     string `gorm:"size:32" json:"time"`
	Count    int    `json:"count"`
}

type MetricDef struct {
	ID     string  `gorm:"primaryKey;size:32" json:"id"`
	Name   string  `gorm:"size:64" json:"name"`
	Unit   string  `gorm:"size:16" json:"unit"`
	Base   float64 `json:"base"`
	Jitter float64 `json:"jitter"`
}

// meta_stats：概览页的非表格型 mock 常量（db_types / sql_issues / lock_summary / top_anomaly）
type MetaStat struct {
	Key   string         `gorm:"primaryKey;size:32" json:"key"`
	Value datatypes.JSON `json:"value"`
}

/* ================= 监控大盘 ================= */

type Dashboard struct {
	ID          string         `gorm:"primaryKey;size:64" json:"id"`
	Title       string         `gorm:"size:128" json:"title"`
	Description string         `gorm:"size:256" json:"description"`
	Cfg         datatypes.JSON `json:"cfg"`
	Panels      datatypes.JSON `json:"panels"`
	CreatedAt   int64          `json:"createdAt"`
	UpdatedAt   int64          `json:"updatedAt"`
}

/* ================= Chat ================= */

type ChatSession struct {
	ID        string `gorm:"primaryKey;size:64" json:"id"`
	Title     string `gorm:"size:128" json:"title"`
	CreatedAt int64  `json:"createdAt"`
	UpdatedAt int64  `json:"updatedAt"`
}

type ChatMessage struct {
	ID        string         `gorm:"primaryKey;size:64" json:"id"`
	SessionID string         `gorm:"size:64;index" json:"-"`
	Seq       int            `gorm:"index" json:"-"`
	Role      string         `gorm:"size:16" json:"role"`
	Text      string         `json:"text"`
	Thoughts  datatypes.JSON `json:"thoughts"`
	Cards     datatypes.JSON `json:"cards"`
	Status    string         `gorm:"size:16" json:"status"`
}

type ChatTurnEvent struct {
	Seq       int64          `gorm:"primaryKey" json:"seq"` // 会话内单调递增
	SessionID string         `gorm:"size:64;index" json:"-"`
	TurnID    string         `gorm:"size:64" json:"turnId"`
	Event     datatypes.JSON `json:"event"`
}

/* ================= 审计 ================= */

type AuditLog struct {
	ID         uint           `gorm:"primaryKey" json:"id"`
	Ts         int64          `json:"ts"`
	Actor      string         `gorm:"size:64" json:"actor"`
	Action     string         `gorm:"size:64" json:"action"`
	TargetType string         `gorm:"size:32" json:"targetType"`
	TargetID   string         `gorm:"size:64" json:"targetId"`
	Detail     datatypes.JSON `json:"detail"`
}

func AllModels() []interface{} {
	return []interface{}{
		&Cluster{}, &Instance{}, &ClusterParam{}, &ParamHistory{},
		&PgDatabase{}, &PgReplica{}, &ObTenant{}, &ObTenantDb{},
		&Host{}, &InstanceUser{}, &RuntimeSession{}, &Trx{}, &SlowSql{},
		&Report{}, &AlertRecord{}, &MetricDef{}, &MetaStat{},
		&Dashboard{}, &ChatSession{}, &ChatMessage{}, &ChatTurnEvent{}, &AuditLog{},
	}
}
