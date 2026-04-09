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
	Interval    time.Duration // Sampling interval (0 = no sampling)
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
	Interval  time.Duration // Sampling interval (0 = no sampling)
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
	HostName1        string    `json:"host_name1"`
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

// Alert represents an alert event from public.alert table
type Alert struct {
	ID        int64     `json:"id"`
	EventID   string    `json:"event_id"`
	Endpoint  string    `json:"endpoint"`
	AlertText string    `json:"alert_text"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Metric    string    `json:"metric"`
	Status    string    `json:"status"`
}

// PaginationRequest represents pagination parameters
type PaginationRequest struct {
	Page     int
	PageSize int
}

// PaginationMeta contains pagination information for responses
type PaginationMeta struct {
	TotalCount  int64 `json:"total_count"`
	TotalPages  int   `json:"total_pages"`
	CurrentPage int   `json:"current_page"`
	PageSize    int   `json:"page_size"`
}

// InstancesQueryRequest for querying instances with pagination
type InstancesQueryRequest struct {
	Pagination PaginationRequest
}

// SlowQuery represents a slow SQL query record
type SlowQuery struct {
	ID           int64     `json:"id"`
	Endpoint     string    `json:"endpoint"`
	Hostname     string    `json:"hostname"`
	Port         int64     `json:"port"`
	DatabaseName string    `json:"database_name"`
	Username     string    `json:"username"`
	SqlText      string    `json:"sql_text"`
	ExecuteTime  float64   `json:"execute_time"`
	ExecuteDate  time.Time `json:"execute_date"`
}

// SlowQueryRequest is the repository request for querying slow queries
type SlowQueryRequest struct {
	Endpoint    string     // Optional - instance endpoint filter
	SqlKeyword  string     // Optional, LIKE filter on sql_text (case-insensitive)
	TimeRange   *TimeRange // Optional, nil means no time filter
	Pagination  PaginationRequest
}

// AlertsQueryRequest is the request for querying alerts with optional filters and pagination
type AlertsQueryRequest struct {
	Endpoint    string     // Optional, empty string means no filter
	AlertText   string     // Optional, ILIKE match on alert_text
	StartTime   *time.Time // Optional, time overlap filter (alert.end_time >= start)
	EndTime     *time.Time // Optional, time overlap filter (alert.start_time <= end)
	Metric      string     // Optional, exact match on metric
	Status      string     // Optional, exact match on status
	Pagination  PaginationRequest
}
