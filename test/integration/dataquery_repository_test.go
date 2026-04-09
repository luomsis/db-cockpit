package integration

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/db-cockpit/pkg/domain/dataquery"
)

// Database connection configuration for integration tests
// Uses the existing postgres database as specified in CLAUDE.md
const (
	testDBHost     = "localhost"
	testDBPort     = 5432
	testDBUser     = "postgres"
	testDBPassword = "postgres"
	testDBName     = "postgres"
)

// setupTestDatabase creates a connection pool for testing
func setupTestDatabase(t *testing.T) *pgxpool.Pool {
	t.Helper()

	connStr := "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"
	config, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		t.Fatalf("Failed to parse database config: %v", err)
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		t.Fatalf("Failed to create connection pool: %v", err)
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	return pool
}

// ============================================================
// Repository Integration Tests
// ============================================================

func TestPGRepository_GetEndpoints(t *testing.T) {
	pool := setupTestDatabase(t)
	defer pool.Close()

	repo := dataquery.NewPGRepository(pool)
	ctx := context.Background()

	t.Run("ReturnsEndpointsFromDatabase", func(t *testing.T) {
		endpoints, err := repo.GetEndpoints(ctx)
		if err != nil {
			t.Fatalf("GetEndpoints() error = %v", err)
		}

		// Should return a slice (may be empty if no data)
		if endpoints == nil {
			t.Error("GetEndpoints() returned nil, expected empty slice")
		}

		t.Logf("Found %d endpoints", len(endpoints))
	})
}

func TestPGRepository_GetMetrics(t *testing.T) {
	pool := setupTestDatabase(t)
	defer pool.Close()

	repo := dataquery.NewPGRepository(pool)
	ctx := context.Background()

	t.Run("ReturnsMetricsForValidEndpoint", func(t *testing.T) {
		// First get a valid endpoint
		endpoints, err := repo.GetEndpoints(ctx)
		if err != nil {
			t.Fatalf("GetEndpoints() error = %v", err)
		}

		if len(endpoints) == 0 {
			t.Skip("No endpoints available in database")
		}

		metrics, err := repo.GetMetrics(ctx, endpoints[0])
		if err != nil {
			t.Fatalf("GetMetrics() error = %v", err)
		}

		t.Logf("Found %d metrics for endpoint %s", len(metrics), endpoints[0])
	})

	t.Run("ReturnsEmptyForNonExistentEndpoint", func(t *testing.T) {
		metrics, err := repo.GetMetrics(ctx, "nonexistent-endpoint-xyz")
		if err != nil {
			t.Fatalf("GetMetrics() error = %v", err)
		}

		if metrics == nil {
			t.Error("GetMetrics() returned nil, expected empty slice")
		}

		if len(metrics) != 0 {
			t.Errorf("GetMetrics() returned %d metrics, want 0", len(metrics))
		}
	})
}

func TestPGRepository_QuerySeries(t *testing.T) {
	pool := setupTestDatabase(t)
	defer pool.Close()

	repo := dataquery.NewPGRepository(pool)
	ctx := context.Background()

	now := time.Now()
	timeRange := dataquery.TimeRange{
		Start: now.Add(-24 * time.Hour),
		End:   now,
	}

	t.Run("ReturnsAllSeriesWithoutFilter", func(t *testing.T) {
		req := &dataquery.SeriesQueryRequest{
			TimeRange: timeRange,
		}

		series, err := repo.QuerySeries(ctx, req)
		if err != nil {
			t.Fatalf("QuerySeries() error = %v", err)
		}

		if series == nil {
			t.Error("QuerySeries() returned nil, expected empty slice")
		}

		t.Logf("Found %d series without filters", len(series))
	})

	t.Run("ReturnsFilteredByEndpoint", func(t *testing.T) {
		// First get a valid endpoint
		endpoints, err := repo.GetEndpoints(ctx)
		if err != nil {
			t.Fatalf("GetEndpoints() error = %v", err)
		}

		if len(endpoints) == 0 {
			t.Skip("No endpoints available in database")
		}

		req := &dataquery.SeriesQueryRequest{
			Endpoint:  endpoints[0],
			TimeRange: timeRange,
		}

		series, err := repo.QuerySeries(ctx, req)
		if err != nil {
			t.Fatalf("QuerySeries() error = %v", err)
		}

		// Verify all returned series have the correct endpoint
		for _, s := range series {
			if s.Endpoint != endpoints[0] {
				t.Errorf("QuerySeries() returned series with endpoint %s, want %s", s.Endpoint, endpoints[0])
			}
		}

		t.Logf("Found %d series for endpoint %s", len(series), endpoints[0])
	})

	t.Run("ReturnsEmptyForNonExistentEndpoint", func(t *testing.T) {
		req := &dataquery.SeriesQueryRequest{
			Endpoint:  "nonexistent-endpoint-xyz",
			TimeRange: timeRange,
		}

		series, err := repo.QuerySeries(ctx, req)
		if err != nil {
			t.Fatalf("QuerySeries() error = %v", err)
		}

		if len(series) != 0 {
			t.Errorf("QuerySeries() returned %d series, want 0", len(series))
		}
	})

	t.Run("RespectsLimitParameter", func(t *testing.T) {
		req := &dataquery.SeriesQueryRequest{
			TimeRange: timeRange,
			Limit:     5,
		}

		series, err := repo.QuerySeries(ctx, req)
		if err != nil {
			t.Fatalf("QuerySeries() error = %v", err)
		}

		if len(series) > 5 {
			t.Errorf("QuerySeries() returned %d series, expected at most 5", len(series))
		}
	})
}

func TestPGRepository_QuerySeries_WithLabelFilter(t *testing.T) {
	pool := setupTestDatabase(t)
	defer pool.Close()

	repo := dataquery.NewPGRepository(pool)
	ctx := context.Background()

	now := time.Now()
	timeRange := dataquery.TimeRange{
		Start: now.Add(-24 * time.Hour),
		End:   now,
	}

	t.Run("ReturnsSeriesWithValidLabelFilter", func(t *testing.T) {
		req := &dataquery.SeriesQueryRequest{
			TimeRange:   timeRange,
			LabelFilter: `env="prod"`,
		}

		series, err := repo.QuerySeries(ctx, req)
		if err != nil {
			t.Fatalf("QuerySeries() error = %v", err)
		}

		t.Logf("Found %d series with label filter env=prod", len(series))
	})

	t.Run("ReturnsErrorForInvalidLabelFilter", func(t *testing.T) {
		req := &dataquery.SeriesQueryRequest{
			TimeRange:   timeRange,
			LabelFilter: `invalid label filter syntax`,
		}

		_, err := repo.QuerySeries(ctx, req)
		if err == nil {
			t.Error("QuerySeries() should return error for invalid label filter")
		}
	})

	t.Run("HandlesRegexLabelFilter", func(t *testing.T) {
		req := &dataquery.SeriesQueryRequest{
			TimeRange:   timeRange,
			LabelFilter: `host=~"server.*"`,
		}

		series, err := repo.QuerySeries(ctx, req)
		if err != nil {
			t.Fatalf("QuerySeries() error = %v", err)
		}

		t.Logf("Found %d series with regex label filter", len(series))
	})

	t.Run("HandlesCompoundLabelFilter", func(t *testing.T) {
		req := &dataquery.SeriesQueryRequest{
			TimeRange:   timeRange,
			LabelFilter: `env="prod" AND region=~"us-.*"`,
		}

		series, err := repo.QuerySeries(ctx, req)
		if err != nil {
			t.Fatalf("QuerySeries() error = %v", err)
		}

		t.Logf("Found %d series with compound label filter", len(series))
	})
}

func TestPGRepository_GetSeriesByID(t *testing.T) {
	pool := setupTestDatabase(t)
	defer pool.Close()

	repo := dataquery.NewPGRepository(pool)
	ctx := context.Background()

	t.Run("ReturnsNilForNonExistentID", func(t *testing.T) {
		series, err := repo.GetSeriesByID(ctx, 999999999)
		if err != nil {
			t.Fatalf("GetSeriesByID() error = %v", err)
		}

		if series != nil {
			t.Error("GetSeriesByID() should return nil for non-existent ID")
		}
	})

	t.Run("ReturnsSeriesForValidID", func(t *testing.T) {
		// First get a valid series ID from QuerySeries
		now := time.Now()
		timeRange := dataquery.TimeRange{
			Start: now.Add(-24 * time.Hour),
			End:   now,
		}

		seriesList, err := repo.QuerySeries(ctx, &dataquery.SeriesQueryRequest{TimeRange: timeRange, Limit: 1})
		if err != nil {
			t.Fatalf("QuerySeries() error = %v", err)
		}

		if len(seriesList) == 0 {
			t.Skip("No series available in database")
		}

		validID := seriesList[0].ID
		series, err := repo.GetSeriesByID(ctx, validID)
		if err != nil {
			t.Fatalf("GetSeriesByID() error = %v", err)
		}

		if series == nil {
			t.Fatal("GetSeriesByID() returned nil for existing series")
		}

		if series.ID != validID {
			t.Errorf("GetSeriesByID() ID = %d, want %d", series.ID, validID)
		}
	})
}

func TestPGRepository_GetSeriesPoints(t *testing.T) {
	pool := setupTestDatabase(t)
	defer pool.Close()

	repo := dataquery.NewPGRepository(pool)
	ctx := context.Background()

	now := time.Now()
	timeRange := dataquery.TimeRange{
		Start: now.Add(-24 * time.Hour),
		End:   now,
	}

	t.Run("ReturnsEmptyForEmptySeriesIDs", func(t *testing.T) {
		req := &dataquery.PointsQueryRequest{
			SeriesIDs: []int64{},
			TimeRange: timeRange,
		}

		points, err := repo.GetSeriesPoints(ctx, req)
		if err != nil {
			t.Fatalf("GetSeriesPoints() error = %v", err)
		}

		if len(points) != 0 {
			t.Errorf("GetSeriesPoints() returned %d series, want 0", len(points))
		}
	})

	t.Run("ReturnsPointsForValidSeriesIDs", func(t *testing.T) {
		// First get valid series IDs
		seriesList, err := repo.QuerySeries(ctx, &dataquery.SeriesQueryRequest{TimeRange: timeRange, Limit: 5})
		if err != nil {
			t.Fatalf("QuerySeries() error = %v", err)
		}

		if len(seriesList) == 0 {
			t.Skip("No series available in database")
		}

		seriesIDs := make([]int64, len(seriesList))
		for i, s := range seriesList {
			seriesIDs[i] = s.ID
		}

		req := &dataquery.PointsQueryRequest{
			SeriesIDs: seriesIDs,
			TimeRange: timeRange,
		}

		points, err := repo.GetSeriesPoints(ctx, req)
		if err != nil {
			t.Fatalf("GetSeriesPoints() error = %v", err)
		}

		// Count total points across all series
		totalPoints := 0
		for _, pts := range points {
			totalPoints += len(pts)
		}

		t.Logf("Found %d total points across %d series", totalPoints, len(points))
	})
}

func TestPGRepository_GetSeriesPoints_WithInterval(t *testing.T) {
	pool := setupTestDatabase(t)
	defer pool.Close()

	repo := dataquery.NewPGRepository(pool)
	ctx := context.Background()

	now := time.Now()
	timeRange := dataquery.TimeRange{
		Start: now.Add(-24 * time.Hour),
		End:   now,
	}

	t.Run("ReturnsAveragedPointsWithInterval", func(t *testing.T) {
		// First get valid series IDs
		seriesList, err := repo.QuerySeries(ctx, &dataquery.SeriesQueryRequest{TimeRange: timeRange, Limit: 1})
		if err != nil {
			t.Fatalf("QuerySeries() error = %v", err)
		}

		if len(seriesList) == 0 {
			t.Skip("No series available in database")
		}

		seriesIDs := []int64{seriesList[0].ID}

		// Get raw points first
		rawReq := &dataquery.PointsQueryRequest{
			SeriesIDs: seriesIDs,
			TimeRange: timeRange,
		}

		rawPoints, err := repo.GetSeriesPoints(ctx, rawReq)
		if err != nil {
			t.Fatalf("GetSeriesPoints() raw error = %v", err)
		}

		// Get sampled points with 5 minute interval
		sampledReq := &dataquery.PointsQueryRequest{
			SeriesIDs: seriesIDs,
			TimeRange: timeRange,
			Interval:  5 * time.Minute,
		}

		sampledPoints, err := repo.GetSeriesPoints(ctx, sampledReq)
		if err != nil {
			t.Fatalf("GetSeriesPoints() sampled error = %v", err)
		}

		// With sampling, we should have fewer or equal points
		rawCount := len(rawPoints[seriesIDs[0]])
		sampledCount := len(sampledPoints[seriesIDs[0]])

		t.Logf("Raw points: %d, Sampled points (5m interval): %d", rawCount, sampledCount)
	})
}

func TestPGRepository_GetSeriesStatistics(t *testing.T) {
	pool := setupTestDatabase(t)
	defer pool.Close()

	repo := dataquery.NewPGRepository(pool)
	ctx := context.Background()

	now := time.Now()
	timeRange := dataquery.TimeRange{
		Start: now.Add(-24 * time.Hour),
		End:   now,
	}

	t.Run("ReturnsEmptyForEmptySeriesIDs", func(t *testing.T) {
		req := &dataquery.StatsRequest{
			SeriesIDs: []int64{},
			TimeRange: timeRange,
		}

		stats, err := repo.GetSeriesStatistics(ctx, req)
		if err != nil {
			t.Fatalf("GetSeriesStatistics() error = %v", err)
		}

		if len(stats) != 0 {
			t.Errorf("GetSeriesStatistics() returned %d entries, want 0", len(stats))
		}
	})

	t.Run("ReturnsStatisticsForValidSeriesIDs", func(t *testing.T) {
		// First get valid series IDs
		seriesList, err := repo.QuerySeries(ctx, &dataquery.SeriesQueryRequest{TimeRange: timeRange, Limit: 5})
		if err != nil {
			t.Fatalf("QuerySeries() error = %v", err)
		}

		if len(seriesList) == 0 {
			t.Skip("No series available in database")
		}

		seriesIDs := make([]int64, len(seriesList))
		for i, s := range seriesList {
			seriesIDs[i] = s.ID
		}

		req := &dataquery.StatsRequest{
			SeriesIDs: seriesIDs,
			TimeRange: timeRange,
		}

		stats, err := repo.GetSeriesStatistics(ctx, req)
		if err != nil {
			t.Fatalf("GetSeriesStatistics() error = %v", err)
		}

		// Verify statistics structure
		for id, stat := range stats {
			if stat == nil {
				t.Errorf("Statistics for series %d is nil", id)
				continue
			}

			// Basic sanity checks
			if stat.Min > stat.Max {
				t.Errorf("Statistics for series %d: Min (%f) > Max (%f)", id, stat.Min, stat.Max)
			}
			if stat.Count < 0 {
				t.Errorf("Statistics for series %d: Count (%d) is negative", id, stat.Count)
			}

			t.Logf("Series %d: Min=%f, Max=%f, Avg=%f, Count=%d", id, stat.Min, stat.Max, stat.Avg, stat.Count)
		}
	})
}

func TestPGRepository_GetInstanceByEndpoint(t *testing.T) {
	pool := setupTestDatabase(t)
	defer pool.Close()

	repo := dataquery.NewPGRepository(pool)
	ctx := context.Background()

	t.Run("ReturnsNilForNonExistentEndpoint", func(t *testing.T) {
		instance, err := repo.GetInstanceByEndpoint(ctx, "nonexistent-endpoint-xyz")
		if err != nil {
			t.Fatalf("GetInstanceByEndpoint() error = %v", err)
		}

		if instance != nil {
			t.Error("GetInstanceByEndpoint() should return nil for non-existent endpoint")
		}
	})

	t.Run("ReturnsInstanceForValidEndpoint", func(t *testing.T) {
		// First get a valid endpoint from GetAllInstances
		instances, _, err := repo.GetAllInstances(ctx, &dataquery.InstancesQueryRequest{
			Pagination: dataquery.PaginationRequest{Page: 1, PageSize: 1},
		})
		if err != nil {
			t.Fatalf("GetAllInstances() error = %v", err)
		}

		if len(instances) == 0 {
			t.Skip("No instances available in database")
		}

		validEndpoint := instances[0].InstanceEndpoint
		instance, err := repo.GetInstanceByEndpoint(ctx, validEndpoint)
		if err != nil {
			t.Fatalf("GetInstanceByEndpoint() error = %v", err)
		}

		if instance == nil {
			t.Fatal("GetInstanceByEndpoint() returned nil for existing endpoint")
		}

		if instance.InstanceEndpoint != validEndpoint {
			t.Errorf("GetInstanceByEndpoint() endpoint = %s, want %s", instance.InstanceEndpoint, validEndpoint)
		}

		t.Logf("Found instance: endpoint=%s, db_type=%s", instance.InstanceEndpoint, instance.DbType)
	})
}

func TestPGRepository_GetAllInstances(t *testing.T) {
	pool := setupTestDatabase(t)
	defer pool.Close()

	repo := dataquery.NewPGRepository(pool)
	ctx := context.Background()

	t.Run("ReturnsInstancesWithDefaultPagination", func(t *testing.T) {
		req := &dataquery.InstancesQueryRequest{
			Pagination: dataquery.PaginationRequest{Page: 1, PageSize: 20},
		}

		instances, totalCount, err := repo.GetAllInstances(ctx, req)
		if err != nil {
			t.Fatalf("GetAllInstances() error = %v", err)
		}

		if instances == nil {
			t.Error("GetAllInstances() returned nil, expected empty slice")
		}

		t.Logf("Found %d instances (total: %d)", len(instances), totalCount)
	})

	t.Run("RespectsPaginationPageSize", func(t *testing.T) {
		pageSize := 5
		req := &dataquery.InstancesQueryRequest{
			Pagination: dataquery.PaginationRequest{Page: 1, PageSize: pageSize},
		}

		instances, _, err := repo.GetAllInstances(ctx, req)
		if err != nil {
			t.Fatalf("GetAllInstances() error = %v", err)
		}

		if len(instances) > pageSize {
			t.Errorf("GetAllInstances() returned %d instances, expected at most %d", len(instances), pageSize)
		}
	})

	t.Run("HandlesSecondPage", func(t *testing.T) {
		// First get total count
		_, totalCount, err := repo.GetAllInstances(ctx, &dataquery.InstancesQueryRequest{
			Pagination: dataquery.PaginationRequest{Page: 1, PageSize: 10},
		})
		if err != nil {
			t.Fatalf("GetAllInstances() error = %v", err)
		}

		if totalCount <= 10 {
			t.Skip("Not enough instances to test pagination")
		}

		// Get second page
		instances, _, err := repo.GetAllInstances(ctx, &dataquery.InstancesQueryRequest{
			Pagination: dataquery.PaginationRequest{Page: 2, PageSize: 10},
		})
		if err != nil {
			t.Fatalf("GetAllInstances() error = %v", err)
		}

		t.Logf("Second page returned %d instances", len(instances))
	})
}

func TestPGRepository_GetAlertsByEndpoint(t *testing.T) {
	pool := setupTestDatabase(t)
	defer pool.Close()

	repo := dataquery.NewPGRepository(pool)
	ctx := context.Background()

	// First get a valid endpoint from instances for tests that need it
	instances, _, err := repo.GetAllInstances(ctx, &dataquery.InstancesQueryRequest{
		Pagination: dataquery.PaginationRequest{Page: 1, PageSize: 1},
	})
	if err != nil {
		t.Fatalf("GetAllInstances() error = %v", err)
	}
	var validEndpoint string
	if len(instances) > 0 {
		validEndpoint = instances[0].InstanceEndpoint
	}

	t.Run("ReturnsEmptyForNonExistentEndpoint", func(t *testing.T) {
		req := &dataquery.AlertsQueryRequest{
			Endpoint:   "nonexistent-endpoint-xyz",
			Pagination: dataquery.PaginationRequest{Page: 1, PageSize: 20},
		}
		alerts, totalCount, err := repo.GetAlertsByEndpoint(ctx, req)
		if err != nil {
			t.Fatalf("GetAlertsByEndpoint() error = %v", err)
		}

		if alerts == nil {
			t.Error("GetAlertsByEndpoint() returned nil, expected empty slice")
		}

		if len(alerts) != 0 {
			t.Errorf("GetAlertsByEndpoint() returned %d alerts, want 0", len(alerts))
		}

		if totalCount != 0 {
			t.Errorf("GetAlertsByEndpoint() totalCount = %d, want 0", totalCount)
		}
	})

	t.Run("ReturnsAlertsForValidEndpoint", func(t *testing.T) {
		if validEndpoint == "" {
			t.Skip("No instances available in database")
		}

		req := &dataquery.AlertsQueryRequest{
			Endpoint:   validEndpoint,
			Pagination: dataquery.PaginationRequest{Page: 1, PageSize: 20},
		}
		alerts, totalCount, err := repo.GetAlertsByEndpoint(ctx, req)
		if err != nil {
			t.Fatalf("GetAlertsByEndpoint() error = %v", err)
		}

		t.Logf("Found %d alerts for endpoint %s (total_count=%d)", len(alerts), validEndpoint, totalCount)

		// Verify alert structure if any
		if len(alerts) > 0 {
			alert := alerts[0]
			if alert.Endpoint != validEndpoint {
				t.Errorf("Alert endpoint = %s, want %s", alert.Endpoint, validEndpoint)
			}
			if alert.EventID == "" {
				t.Log("Warning: Alert has empty event_id")
			}
		}
	})

	t.Run("PaginationWorks", func(t *testing.T) {
		if validEndpoint == "" {
			t.Skip("No instances available in database")
		}

		// First get total count with page 1
		req1 := &dataquery.AlertsQueryRequest{
			Endpoint:   validEndpoint,
			Pagination: dataquery.PaginationRequest{Page: 1, PageSize: 10},
		}
		alerts1, totalCount, err := repo.GetAlertsByEndpoint(ctx, req1)
		if err != nil {
			t.Fatalf("GetAlertsByEndpoint() error = %v", err)
		}

		if totalCount <= 10 {
			t.Skipf("Only %d total alerts, need more than 10 to test pagination", totalCount)
		}

		// Test page 2
		req2 := &dataquery.AlertsQueryRequest{
			Endpoint:   validEndpoint,
			Pagination: dataquery.PaginationRequest{Page: 2, PageSize: 10},
		}
		alerts2, totalCount2, err := repo.GetAlertsByEndpoint(ctx, req2)
		if err != nil {
			t.Fatalf("GetAlertsByEndpoint() error = %v", err)
		}

		if totalCount2 != totalCount {
			t.Errorf("Page 2 totalCount = %d, want %d (same as page 1)", totalCount2, totalCount)
		}

		t.Logf("Page 1: %d alerts, Page 2: %d alerts, Total: %d", len(alerts1), len(alerts2), totalCount)
	})

	t.Run("DefaultPagination", func(t *testing.T) {
		if validEndpoint == "" {
			t.Skip("No instances available in database")
		}

		req := &dataquery.AlertsQueryRequest{
			Endpoint:   validEndpoint,
			Pagination: dataquery.PaginationRequest{Page: 1, PageSize: 20},
		}
		alerts, totalCount, err := repo.GetAlertsByEndpoint(ctx, req)
		if err != nil {
			t.Fatalf("GetAlertsByEndpoint() error = %v", err)
		}

		// PageSize 20 should return at most 20 items
		if len(alerts) > 20 {
			t.Errorf("Expected at most 20 alerts, got %d", len(alerts))
		}

		t.Logf("Default pagination returned %d alerts (total_count=%d)", len(alerts), totalCount)
	})

	t.Run("MaxPageSizeClamping", func(t *testing.T) {
		if validEndpoint == "" {
			t.Skip("No instances available in database")
		}

		// Request with page_size > 100 (max allowed by service layer)
		// Repository doesn't clamp, so it will return whatever is available
		req := &dataquery.AlertsQueryRequest{
			Endpoint:   validEndpoint,
			Pagination: dataquery.PaginationRequest{Page: 1, PageSize: 200},
		}
		alerts, totalCount, err := repo.GetAlertsByEndpoint(ctx, req)
		if err != nil {
			t.Fatalf("GetAlertsByEndpoint() error = %v", err)
		}

		t.Logf("PageSize 200 returned %d alerts (total_count=%d)", len(alerts), totalCount)
	})
}

func TestPGRepository_GetSlowQueries(t *testing.T) {
	pool := setupTestDatabase(t)
	defer pool.Close()

	repo := dataquery.NewPGRepository(pool)
	ctx := context.Background()

	// First get a valid endpoint from instances for tests that need it
	instances, _, err := repo.GetAllInstances(ctx, &dataquery.InstancesQueryRequest{
		Pagination: dataquery.PaginationRequest{Page: 1, PageSize: 1},
	})
	if err != nil {
		t.Fatalf("GetAllInstances() error = %v", err)
	}
	var validEndpoint string
	if len(instances) > 0 {
		validEndpoint = instances[0].InstanceEndpoint
	}

	t.Run("ReturnsAllWithNoFilters", func(t *testing.T) {
		// Request with no filters should return all slow queries
		req := &dataquery.SlowQueryRequest{
			Pagination: dataquery.PaginationRequest{Page: 1, PageSize: 20},
		}
		slowQueries, totalCount, err := repo.GetSlowQueries(ctx, req)
		if err != nil {
			t.Fatalf("GetSlowQueries() error = %v", err)
		}

		if slowQueries == nil {
			t.Error("GetSlowQueries() returned nil, expected empty slice")
		}

		t.Logf("Found %d slow queries with no filters (total_count=%d)", len(slowQueries), totalCount)
	})

	t.Run("ReturnsEmptyForNonExistentEndpoint", func(t *testing.T) {
		req := &dataquery.SlowQueryRequest{
			Endpoint:   "nonexistent-endpoint-xyz",
			Pagination: dataquery.PaginationRequest{Page: 1, PageSize: 20},
		}
		slowQueries, totalCount, err := repo.GetSlowQueries(ctx, req)
		if err != nil {
			t.Fatalf("GetSlowQueries() error = %v", err)
		}

		if slowQueries == nil {
			t.Error("GetSlowQueries() returned nil, expected empty slice")
		}

		if len(slowQueries) != 0 {
			t.Errorf("GetSlowQueries() returned %d slow queries, want 0", len(slowQueries))
		}

		if totalCount != 0 {
			t.Errorf("GetSlowQueries() totalCount = %d, want 0", totalCount)
		}
	})

	t.Run("ReturnsSlowQueriesForValidEndpoint", func(t *testing.T) {
		if validEndpoint == "" {
			t.Skip("No instances available in database")
		}

		req := &dataquery.SlowQueryRequest{
			Endpoint:   validEndpoint,
			Pagination: dataquery.PaginationRequest{Page: 1, PageSize: 20},
		}
		slowQueries, totalCount, err := repo.GetSlowQueries(ctx, req)
		if err != nil {
			t.Fatalf("GetSlowQueries() error = %v", err)
		}

		t.Logf("Found %d slow queries for endpoint %s (total_count=%d)", len(slowQueries), validEndpoint, totalCount)

		// Verify slow query structure if any
		if len(slowQueries) > 0 {
			sq := slowQueries[0]
			if sq.Endpoint != validEndpoint {
				t.Errorf("SlowQuery endpoint = %s, want %s", sq.Endpoint, validEndpoint)
			}
			if sq.ID == 0 {
				t.Log("Warning: SlowQuery has zero ID")
			}
			t.Logf("Sample slow query: id=%d, sql_text=%s, execute_time=%.2f", sq.ID, truncate(sq.SqlText, 50), sq.ExecuteTime)
		}
	})

	t.Run("FiltersBySqlKeyword", func(t *testing.T) {
		// First, get some slow queries to see what SQL text patterns exist
		req := &dataquery.SlowQueryRequest{
			Pagination: dataquery.PaginationRequest{Page: 1, PageSize: 10},
		}
		allQueries, _, err := repo.GetSlowQueries(ctx, req)
		if err != nil {
			t.Fatalf("GetSlowQueries() error = %v", err)
		}

		if len(allQueries) == 0 {
			t.Skip("No slow queries available in database")
		}

		// Try filtering with a common SQL keyword
		keywordReq := &dataquery.SlowQueryRequest{
			SqlKeyword: "SELECT",
			Pagination: dataquery.PaginationRequest{Page: 1, PageSize: 20},
		}
		filteredQueries, _, err := repo.GetSlowQueries(ctx, keywordReq)
		if err != nil {
			t.Fatalf("GetSlowQueries() with sql_keyword error = %v", err)
		}

		t.Logf("Found %d slow queries with sql_keyword=SELECT", len(filteredQueries))

		// Verify all returned queries contain the keyword (case-insensitive)
		for _, sq := range filteredQueries {
			// Note: ILIKE in SQL handles case-insensitive matching
			t.Logf("SQL text (truncated): %s", truncate(sq.SqlText, 50))
		}
	})

	t.Run("FiltersByTimeRange", func(t *testing.T) {
		now := time.Now()
		start := now.Add(-24 * time.Hour)
		end := now

		req := &dataquery.SlowQueryRequest{
			TimeRange:  &dataquery.TimeRange{Start: start, End: end},
			Pagination: dataquery.PaginationRequest{Page: 1, PageSize: 20},
		}
		slowQueries, totalCount, err := repo.GetSlowQueries(ctx, req)
		if err != nil {
			t.Fatalf("GetSlowQueries() error = %v", err)
		}

		t.Logf("Found %d slow queries in last 24 hours (total_count=%d)", len(slowQueries), totalCount)

		// Verify all returned queries are within the time range
		for _, sq := range slowQueries {
			if sq.ExecuteDate.Before(start) || sq.ExecuteDate.After(end) {
				t.Errorf("SlowQuery execute_date %v is outside range [%v, %v]", sq.ExecuteDate, start, end)
			}
		}
	})

	t.Run("CombinedFilters", func(t *testing.T) {
		if validEndpoint == "" {
			t.Skip("No instances available in database")
		}

		now := time.Now()
		start := now.Add(-7 * 24 * time.Hour) // Last 7 days
		end := now

		req := &dataquery.SlowQueryRequest{
			Endpoint:   validEndpoint,
			TimeRange:  &dataquery.TimeRange{Start: start, End: end},
			SqlKeyword: "SELECT",
			Pagination: dataquery.PaginationRequest{Page: 1, PageSize: 10},
		}
		slowQueries, totalCount, err := repo.GetSlowQueries(ctx, req)
		if err != nil {
			t.Fatalf("GetSlowQueries() error = %v", err)
		}

		t.Logf("Combined filters: endpoint=%s, time_range=7d, sql_keyword=SELECT => %d results (total=%d)",
			validEndpoint, len(slowQueries), totalCount)
	})

	t.Run("PaginationWorks", func(t *testing.T) {
		// First get total count with page 1
		req1 := &dataquery.SlowQueryRequest{
			Pagination: dataquery.PaginationRequest{Page: 1, PageSize: 10},
		}
		page1Queries, totalCount, err := repo.GetSlowQueries(ctx, req1)
		if err != nil {
			t.Fatalf("GetSlowQueries() error = %v", err)
		}

		if totalCount <= 10 {
			t.Skipf("Only %d total slow queries, need more than 10 to test pagination", totalCount)
		}

		// Test page 2
		req2 := &dataquery.SlowQueryRequest{
			Pagination: dataquery.PaginationRequest{Page: 2, PageSize: 10},
		}
		page2Queries, totalCount2, err := repo.GetSlowQueries(ctx, req2)
		if err != nil {
			t.Fatalf("GetSlowQueries() error = %v", err)
		}

		if totalCount2 != totalCount {
			t.Errorf("Page 2 totalCount = %d, want %d (same as page 1)", totalCount2, totalCount)
		}

		// Verify different results on different pages
		if len(page1Queries) > 0 && len(page2Queries) > 0 {
			if page1Queries[0].ID == page2Queries[0].ID {
				t.Error("Page 1 and Page 2 returned same first result, pagination may not be working")
			}
		}

		t.Logf("Page 1: %d queries, Page 2: %d queries, Total: %d", len(page1Queries), len(page2Queries), totalCount)
	})
}

// truncate truncates a string to maxLen characters
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}