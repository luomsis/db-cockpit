package dataquery

import (
	"time"
)

// TimeRange represents a time range for queries
type TimeRange struct {
	Start time.Time
	End   time.Time
}

// SeriesMeta represents metadata for a time series
type SeriesMeta struct {
	ID         int64
	Endpoint   string
	Metric     string
	Labels     map[string]string
	LabelsHash string
	CreatedAt  time.Time
}

// DataPoint represents a single data point in a time series
type DataPoint struct {
	Time  time.Time
	Value float64
}

// SeriesStatistics represents statistical summary of a series
type SeriesStatistics struct {
	Min   float64
	Max   float64
	Avg   float64
	Sum   float64
	Count int
}

// SeriesData represents a complete series with metadata and data points
type SeriesData struct {
	Meta       SeriesMeta
	Points     []DataPoint
	Statistics *SeriesStatistics
}

// SeriesQuery represents a query for series
type SeriesQuery struct {
	Endpoint    string
	Metric      string
	LabelFilter string
	TimeRange   TimeRange
	Limit       int
}

// MultiSeriesQuery represents a query for multiple series
type MultiSeriesQuery struct {
	Endpoints   []string
	Metrics     []string
	LabelFilter string
	TimeRange   TimeRange
}

// SeriesQueryRequest is the repository request for querying series
type SeriesQueryRequest struct {
	Endpoint    string
	Metric      string
	LabelFilter string // raw expression
	TimeRange   TimeRange
	Limit       int
}

// PointsQueryRequest is the repository request for querying data points
type PointsQueryRequest struct {
	SeriesIDs []int64
	TimeRange TimeRange
}

// StatsRequest is the repository request for series statistics
type StatsRequest struct {
	SeriesIDs []int64
	TimeRange TimeRange
}

// InstanceMeta represents metadata for a database instance
type InstanceMeta struct {
	ID               int64     `json:"id"`
	DbType           string    `json:"db_type"`
	EntityName       string    `json:"entity_name"`
	ChineseDesc      string    `json:"chinese_desc"`
	OrgCode          string    `json:"org_code"`
	ServiceUser      string    `json:"service_user"`
	OprDba           string    `json:"opr_dba"`
	BusinessOwner    string    `json:"business_owner"`
	AlertSubscriber  string    `json:"alert_subscriber"`
	InfraType        string    `json:"infra_type"`
	ReqCPU           float64   `json:"req_cpu"`
	ReqMemoryGB      float64   `json:"req_memory_gb"`
	ReqStorageGB     float64   `json:"req_storage_gb"`
	CreatedDate      time.Time `json:"created_date"`
	Environment      string    `json:"environment"`
	OprDbaII         string    `json:"opr_dba_ii"`
	InsCreatedDate   time.Time `json:"ins_created_date"`
	InsUpdatedDate   time.Time `json:"ins_updated_date"`
	HostEnvironment1 string    `json:"host_environment1"`
	HostEnvironment2 string    `json:"host_environment2"`
	LeName           string    `json:"le_name"`
	InstanceEndpoint string    `json:"instance_endpoint"`
	SubsysCode       string    `json:"subsys_code"`
	SourceSys        string    `json:"source_sys"`
	AttachDb         string    `json:"attach_db"`
	HostNamel        string    `json:"host_namel"`
	HostName2        string    `json:"host_name2"`
	DefaultRole      string    `json:"default_role"`
	Role             string    `json:"role"`
	Status           string    `json:"status"`
	VersionDetail    string    `json:"version_detail"`
	InstanceName     string    `json:"instance_name"`
	IsCreatedByCloud string    `json:"is_created_by_cloud"`
	CharacterSet     string    `json:"character_set"`
	InstanceVip      string    `json:"instance_vip"`
	InstancePort     int64     `json:"instance_port"`
	UserName         string    `json:"user_name"`
	HostIP1          string    `json:"host_ip1"`
	HostInfraType1   string    `json:"host_infra_type1"`
	OsName           string    `json:"os_name"`
	HostIP2          string    `json:"host_ip2"`
	HostInfraType2   string    `json:"host_infra_type2"`
	HaType           string    `json:"ha_type"`
	BackupMethod     string    `json:"backup_method"`
	FailoverType     string    `json:"failover_type"`
	InsUUID          string    `json:"ins_uuid"`
	CcmName          string    `json:"ccm_name"`
}
