package threshold

import (
	"context"
	"sync"
	"time"

	"github.com/db-cockpit/pkg/common/utils"
	"github.com/db-cockpit/pkg/domain"
)

// Service implements the ThresholdService interface
type Service struct {
	repo            Repository
	alertSubs       []chan *ThresholdAlert
	alertMutex      sync.RWMutex
	thresholdsCache map[string][]Threshold
	cacheMutex      sync.RWMutex
}

// NewService creates a new threshold service
func NewService(repo Repository) *Service {
	return &Service{
		repo:            repo,
		alertSubs:       make([]chan *ThresholdAlert, 0),
		thresholdsCache: make(map[string][]Threshold),
	}
}

// Name returns the service name
func (s *Service) Name() string {
	return "threshold"
}

// Initialize initializes the service
func (s *Service) Initialize(ctx context.Context) error {
	return nil
}

// Shutdown shuts down the service
func (s *Service) Shutdown(ctx context.Context) error {
	s.alertMutex.Lock()
	for _, ch := range s.alertSubs {
		close(ch)
	}
	s.alertSubs = nil
	s.alertMutex.Unlock()
	return nil
}

// Health returns the health status
func (s *Service) Health(ctx context.Context) error {
	return nil
}

// GetThresholds retrieves thresholds for metrics
func (s *Service) GetThresholds(ctx *domain.DomainContext, databaseID string, metricNames []string) ([]Threshold, error) {
	// Check cache first
	cacheKey := ctx.TenantID + ":" + databaseID
	s.cacheMutex.RLock()
	if thresholds, ok := s.thresholdsCache[cacheKey]; ok {
		s.cacheMutex.RUnlock()
		return s.filterThresholds(thresholds, metricNames), nil
	}
	s.cacheMutex.RUnlock()

	// Get from repository
	thresholds, err := s.repo.GetThresholds(ctx.Context(), ctx.TenantID, databaseID, metricNames)
	if err != nil {
		return nil, err
	}

	// Update cache
	s.cacheMutex.Lock()
	s.thresholdsCache[cacheKey] = thresholds
	s.cacheMutex.Unlock()

	return thresholds, nil
}

// filterThresholds filters thresholds by metric names
func (s *Service) filterThresholds(thresholds []Threshold, metricNames []string) []Threshold {
	if len(metricNames) == 0 {
		return thresholds
	}

	result := []Threshold{}
	metricSet := make(map[string]bool)
	for _, m := range metricNames {
		metricSet[m] = true
	}

	for _, t := range thresholds {
		if metricSet[t.MetricName] {
			result = append(result, t)
		}
	}
	return result
}

// UpdateThreshold updates a threshold
func (s *Service) UpdateThreshold(ctx *domain.DomainContext, thresholdID string, value float64, thresholdType ThresholdType) error {
	// Get existing threshold
	thresholds, err := s.repo.GetThresholds(ctx.Context(), ctx.TenantID, "", nil)
	if err != nil {
		return err
	}

	var existing *Threshold
	for _, t := range thresholds {
		if t.ThresholdID == thresholdID {
			existing = &t
			break
		}
	}

	if existing == nil {
		return ErrThresholdNotFound
	}

	// Update threshold
	existing.Value = value
	existing.Type = thresholdType
	existing.UpdatedAt = time.Now()

	if err := s.repo.SaveThreshold(ctx.Context(), existing); err != nil {
		return err
	}

	// Save history
	history := &ThresholdHistoryEntry{
		EntryID:   utils.GenerateID(),
		Value:     value,
		Timestamp: time.Now(),
		Reason:    "Manual update",
	}
	_ = s.repo.SaveThresholdHistory(ctx.Context(), history)

	// Invalidate cache
	s.cacheMutex.Lock()
	delete(s.thresholdsCache, ctx.TenantID+":"+existing.MetricName)
	s.cacheMutex.Unlock()

	return nil
}

// CalculateThresholds calculates dynamic thresholds
func (s *Service) CalculateThresholds(ctx *domain.DomainContext, databaseID, metricName string, startTime, endTime time.Time, method CalculationMethod) ([]CalculatedThreshold, error) {
	// TODO: Implement actual threshold calculation based on method
	// This would involve:
	// 1. Querying historical metric data
	// 2. Applying statistical/ML algorithms
	// 3. Calculating bounds

	// Placeholder implementation
	calculated := []CalculatedThreshold{
		{
			MetricName: metricName,
			LowerBound: 0,
			UpperBound: 100,
			Confidence: 0.95,
			Method:     method,
		},
	}

	return calculated, nil
}

// CheckThreshold checks if a value breaches threshold
func (s *Service) CheckThreshold(ctx context.Context, tenantID, databaseID, metricName string, value float64) (bool, string, error) {
	thresholds, err := s.repo.GetThresholds(ctx, tenantID, databaseID, []string{metricName})
	if err != nil {
		return false, "", err
	}

	if len(thresholds) == 0 {
		return false, "", nil
	}

	threshold := thresholds[0]
	var breached bool
	var severity string

	// Check threshold condition
	switch threshold.Condition {
	case ConditionGreaterThan:
		breached = value > threshold.Value
		if breached {
			severity = s.determineSeverity(value, threshold.Value)
		}
	case ConditionLessThan:
		breached = value < threshold.Value
		if breached {
			severity = s.determineSeverity(value, threshold.Value)
		}
	case ConditionBetween:
		breached = value < threshold.DynamicLower || value > threshold.DynamicUpper
		if breached {
			severity = "warning"
		}
	}

	// Generate alert if breached
	if breached {
		alert := &ThresholdAlert{
			AlertID:        utils.GenerateID(),
			ThresholdID:    threshold.ThresholdID,
			MetricName:     metricName,
			DatabaseID:     databaseID,
			Severity:       severity,
			Value:          value,
			ThresholdValue: threshold.Value,
			Timestamp:      time.Now(),
			Message:        metricName + " breached threshold",
		}

		// Save alert
		_ = s.repo.SaveAlert(ctx, alert)

		// Notify subscribers
		s.notifyAlertSubscribers(alert)
	}

	return breached, severity, nil
}

// determineSeverity determines alert severity based on deviation
func (s *Service) determineSeverity(value, threshold float64) string {
	deviation := (value - threshold) / threshold * 100
	if deviation > 50 {
		return "critical"
	} else if deviation > 25 {
		return "warning"
	}
	return "info"
}

// GetThresholdHistory retrieves threshold history
func (s *Service) GetThresholdHistory(ctx *domain.DomainContext, thresholdID string, startTime, endTime time.Time, limit int) ([]ThresholdHistoryEntry, error) {
	if limit <= 0 {
		limit = 100
	}
	return s.repo.GetThresholdHistory(ctx.Context(), thresholdID, startTime, endTime, limit)
}

// CreateThresholdRule creates a threshold rule
func (s *Service) CreateThresholdRule(ctx *domain.DomainContext, rule *ThresholdRule) (string, error) {
	rule.RuleID = utils.GenerateID()

	if err := s.repo.SaveThresholdRule(ctx.Context(), rule); err != nil {
		return "", err
	}

	return rule.RuleID, nil
}

// SubscribeAlerts subscribes to threshold alerts
func (s *Service) SubscribeAlerts(ctx *domain.DomainContext, callback func(alert *ThresholdAlert) error) error {
	ch := make(chan *ThresholdAlert, 100)

	s.alertMutex.Lock()
	s.alertSubs = append(s.alertSubs, ch)
	s.alertMutex.Unlock()

	// Handle alerts
	go func() {
		for alert := range ch {
			if err := callback(alert); err != nil {
				// Log error
			}
		}
	}()

	return nil
}

// notifyAlertSubscribers notifies all alert subscribers
func (s *Service) notifyAlertSubscribers(alert *ThresholdAlert) {
	s.alertMutex.RLock()
	defer s.alertMutex.RUnlock()

	for _, ch := range s.alertSubs {
		select {
		case ch <- alert:
		default:
			// Channel full, skip
		}
	}
}

// Error definitions
var (
	ErrThresholdNotFound = &ThresholdError{Code: "THRESHOLD_NOT_FOUND", Message: "Threshold not found"}
)

// ThresholdError represents a threshold error
type ThresholdError struct {
	Code    string
	Message string
}

func (e *ThresholdError) Error() string {
	return e.Message
}
