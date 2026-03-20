package threshold

import (
	"context"
	"time"

	"github.com/db-cockpit/pkg/domain"
)

// ThresholdType represents the type of threshold
type ThresholdType string

const (
	ThresholdTypeStatic   ThresholdType = "static"
	ThresholdTypeDynamic  ThresholdType = "dynamic"
	ThresholdTypeAdaptive ThresholdType = "adaptive"
)

// ThresholdCondition represents the condition for threshold
type ThresholdCondition string

const (
	ConditionGreaterThan ThresholdCondition = "greater_than"
	ConditionLessThan    ThresholdCondition = "less_than"
	ConditionBetween     ThresholdCondition = "between"
	ConditionOutside     ThresholdCondition = "outside"
)

// CalculationMethod represents the method for calculating dynamic thresholds
type CalculationMethod string

const (
	MethodStatistical CalculationMethod = "statistical"
	MethodPercentile  CalculationMethod = "percentile"
	MethodML          CalculationMethod = "ml"
	MethodSeasonal    CalculationMethod = "seasonal"
)

// Threshold represents a threshold configuration
type Threshold struct {
	ThresholdID  string
	MetricName   string
	Type         ThresholdType
	Condition    ThresholdCondition
	Value        float64
	DynamicLower float64
	DynamicUpper float64
	IsDynamic    bool
	Unit         string
	UpdatedAt    time.Time
}

// ThresholdRule represents a threshold rule
type ThresholdRule struct {
	RuleID            string
	DatabaseID        string
	MetricName        string
	Name              string
	Description       string
	Type              ThresholdType
	Condition         ThresholdCondition
	StaticValue       float64
	DynamicParameters map[string]string
	AlertConfig       AlertConfig
}

// AlertConfig represents alert configuration
type AlertConfig struct {
	Enabled         bool
	Channels        []string
	CooldownSeconds int
	Recipients      []string
}

// ThresholdAlert represents a threshold alert
type ThresholdAlert struct {
	AlertID        string
	ThresholdID    string
	MetricName     string
	DatabaseID     string
	Severity       string
	Value          float64
	ThresholdValue float64
	Timestamp      time.Time
	Message        string
	Labels         map[string]string
}

// CalculatedThreshold represents a calculated threshold
type CalculatedThreshold struct {
	MetricName string
	LowerBound float64
	UpperBound float64
	Confidence float64
	Method     CalculationMethod
	Parameters map[string]float64
}

// ThresholdService defines the interface for threshold domain
type ThresholdService interface {
	domain.DomainService

	// GetThresholds retrieves thresholds for metrics
	GetThresholds(ctx *domain.DomainContext, databaseID string, metricNames []string) ([]Threshold, error)

	// UpdateThreshold updates a threshold
	UpdateThreshold(ctx *domain.DomainContext, thresholdID string, value float64, thresholdType ThresholdType) error

	// CalculateThresholds calculates dynamic thresholds
	CalculateThresholds(ctx *domain.DomainContext, databaseID, metricName string, startTime, endTime time.Time, method CalculationMethod) ([]CalculatedThreshold, error)

	// CheckThreshold checks if a value breaches threshold
	CheckThreshold(ctx context.Context, tenantID, databaseID, metricName string, value float64) (bool, string, error)

	// GetThresholdHistory retrieves threshold history
	GetThresholdHistory(ctx *domain.DomainContext, thresholdID string, startTime, endTime time.Time, limit int) ([]ThresholdHistoryEntry, error)

	// CreateThresholdRule creates a threshold rule
	CreateThresholdRule(ctx *domain.DomainContext, rule *ThresholdRule) (string, error)

	// SubscribeAlerts subscribes to threshold alerts
	SubscribeAlerts(ctx *domain.DomainContext, callback func(alert *ThresholdAlert) error) error
}

// ThresholdHistoryEntry represents a threshold history entry
type ThresholdHistoryEntry struct {
	EntryID      string
	Value        float64
	DynamicLower float64
	DynamicUpper float64
	Timestamp    time.Time
	Reason       string
	Metrics      map[string]float64
}

// Repository defines the data access interface for threshold domain
type Repository interface {
	// GetThresholds retrieves thresholds
	GetThresholds(ctx context.Context, tenantID, databaseID string, metricNames []string) ([]Threshold, error)

	// SaveThreshold saves a threshold
	SaveThreshold(ctx context.Context, threshold *Threshold) error

	// GetThresholdHistory retrieves threshold history
	GetThresholdHistory(ctx context.Context, thresholdID string, startTime, endTime time.Time, limit int) ([]ThresholdHistoryEntry, error)

	// SaveThresholdHistory saves threshold history
	SaveThresholdHistory(ctx context.Context, entry *ThresholdHistoryEntry) error

	// SaveThresholdRule saves a threshold rule
	SaveThresholdRule(ctx context.Context, rule *ThresholdRule) error

	// GetThresholdRules retrieves threshold rules
	GetThresholdRules(ctx context.Context, tenantID, databaseID string) ([]ThresholdRule, error)

	// SaveAlert saves an alert
	SaveAlert(ctx context.Context, alert *ThresholdAlert) error

	// GetRecentAlerts retrieves recent alerts
	GetRecentAlerts(ctx context.Context, tenantID string, limit int) ([]ThresholdAlert, error)
}
