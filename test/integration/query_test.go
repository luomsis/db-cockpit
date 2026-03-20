package integration

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/db-cockpit/pkg/domain"
	"github.com/db-cockpit/pkg/domain/dataquery"
	"github.com/db-cockpit/pkg/domain/dataquery/labels"
)

// Test database connection
func getTestDB(t *testing.T) *pgxpool.Pool {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/dbcockpit?sslmode=disable"
	}

	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		t.Skipf("Skipping test: cannot connect to database: %v", err)
		return nil
	}

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("Skipping test: cannot ping database: %v", err)
		return nil
	}

	return pool
}

// TestDomainContext tests domain context creation
func TestDomainContext(t *testing.T) {
	ctx := domain.NewDomainContext(context.Background(), "tenant-1", "user-1")

	if ctx.TenantID != "tenant-1" {
		t.Errorf("Expected TenantID tenant-1, got %s", ctx.TenantID)
	}
	if ctx.UserID != "user-1" {
		t.Errorf("Expected UserID user-1, got %s", ctx.UserID)
	}
}

// TestPGRepositoryEnsureTables tests table creation
func TestPGRepositoryEnsureTables(t *testing.T) {
	pool := getTestDB(t)
	if pool == nil {
		return
	}
	defer pool.Close()

	repo := dataquery.NewPGRepository(pool)

	if err := repo.EnsureTables(context.Background()); err != nil {
		t.Fatalf("EnsureTables() error = %v", err)
	}
}

// TestPGRepositoryInsertAndQuery tests inserting and querying data
func TestPGRepositoryInsertAndQuery(t *testing.T) {
	pool := getTestDB(t)
	if pool == nil {
		return
	}
	defer pool.Close()

	repo := dataquery.NewPGRepository(pool)
	ctx := context.Background()

	// Ensure tables exist
	if err := repo.EnsureTables(ctx); err != nil {
		t.Fatalf("EnsureTables() error = %v", err)
	}

	// Insert test series
	labels := map[string]string{
		"host":   fmt.Sprintf("test-server-%d", time.Now().UnixNano()),
		"region": "test-region",
		"env":    "test",
	}

	meta, err := repo.InsertSeriesMeta(ctx, "/test/endpoint", "test_metric", labels)
	if err != nil {
		t.Fatalf("InsertSeriesMeta() error = %v", err)
	}

	if meta.ID == 0 {
		t.Error("InsertSeriesMeta() returned zero ID")
	}
	if meta.Endpoint != "/test/endpoint" {
		t.Errorf("Endpoint = %q, want %q", meta.Endpoint, "/test/endpoint")
	}

	// Insert test points
	now := time.Now()
	points := []dataquery.DataPoint{
		{Time: now.Add(-10 * time.Minute), Value: 10.0},
		{Time: now.Add(-5 * time.Minute), Value: 20.0},
		{Time: now, Value: 30.0},
	}

	if err := repo.InsertPoints(ctx, meta.ID, points); err != nil {
		t.Fatalf("InsertPoints() error = %v", err)
	}

	// Query the series back
	queried, err := repo.GetSeriesByID(ctx, meta.ID)
	if err != nil {
		t.Fatalf("GetSeriesByID() error = %v", err)
	}

	if queried == nil {
		t.Fatal("GetSeriesByID() returned nil")
	}

	if queried.Endpoint != "/test/endpoint" {
		t.Errorf("Queried endpoint = %q, want %q", queried.Endpoint, "/test/endpoint")
	}
}

// TestPGRepositoryGetEndpoints tests getting all endpoints
func TestPGRepositoryGetEndpoints(t *testing.T) {
	pool := getTestDB(t)
	if pool == nil {
		return
	}
	defer pool.Close()

	repo := dataquery.NewPGRepository(pool)
	ctx := context.Background()

	endpoints, err := repo.GetEndpoints(ctx)
	if err != nil {
		t.Fatalf("GetEndpoints() error = %v", err)
	}

	t.Logf("Found %d endpoints", len(endpoints))
}

// TestPGRepositoryGetMetrics tests getting metrics for an endpoint
func TestPGRepositoryGetMetrics(t *testing.T) {
	pool := getTestDB(t)
	if pool == nil {
		return
	}
	defer pool.Close()

	repo := dataquery.NewPGRepository(pool)
	ctx := context.Background()

	// First get endpoints
	endpoints, err := repo.GetEndpoints(ctx)
	if err != nil {
		t.Fatalf("GetEndpoints() error = %v", err)
	}

	if len(endpoints) == 0 {
		t.Skip("No endpoints found in database")
	}

	// Get metrics for first endpoint
	metrics, err := repo.GetMetrics(ctx, endpoints[0])
	if err != nil {
		t.Fatalf("GetMetrics() error = %v", err)
	}

	t.Logf("Found %d metrics for endpoint %s", len(metrics), endpoints[0])
}

// TestPGRepositoryQuerySeries tests querying series with filters
func TestPGRepositoryQuerySeries(t *testing.T) {
	pool := getTestDB(t)
	if pool == nil {
		return
	}
	defer pool.Close()

	repo := dataquery.NewPGRepository(pool)
	ctx := context.Background()

	now := time.Now()
	timeRange := dataquery.TimeRange{
		Start: now.Add(-24 * time.Hour),
		End:   now,
	}

	// Query all series
	series, err := repo.QuerySeries(ctx, &dataquery.SeriesQueryRequest{
		TimeRange: timeRange,
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("QuerySeries() error = %v", err)
	}

	t.Logf("Found %d series", len(series))
}

// TestPGRepositoryQuerySeriesWithLabelFilter tests querying with label filter
func TestPGRepositoryQuerySeriesWithLabelFilter(t *testing.T) {
	pool := getTestDB(t)
	if pool == nil {
		return
	}
	defer pool.Close()

	repo := dataquery.NewPGRepository(pool)
	ctx := context.Background()

	now := time.Now()
	timeRange := dataquery.TimeRange{
		Start: now.Add(-24 * time.Hour),
		End:   now,
	}

	// Query with label filter
	series, err := repo.QuerySeries(ctx, &dataquery.SeriesQueryRequest{
		LabelFilter: `env="prod"`,
		TimeRange:   timeRange,
		Limit:       10,
	})
	if err != nil {
		t.Fatalf("QuerySeries() with label filter error = %v", err)
	}

	t.Logf("Found %d series with label filter", len(series))
}

// TestPGRepositoryGetSeriesPoints tests getting data points
func TestPGRepositoryGetSeriesPoints(t *testing.T) {
	pool := getTestDB(t)
	if pool == nil {
		return
	}
	defer pool.Close()

	repo := dataquery.NewPGRepository(pool)
	ctx := context.Background()

	now := time.Now()
	timeRange := dataquery.TimeRange{
		Start: now.Add(-24 * time.Hour),
		End:   now,
	}

	// First get series
	series, err := repo.QuerySeries(ctx, &dataquery.SeriesQueryRequest{
		TimeRange: timeRange,
		Limit:     5,
	})
	if err != nil {
		t.Fatalf("QuerySeries() error = %v", err)
	}

	if len(series) == 0 {
		t.Skip("No series found in database")
	}

	// Get points for found series
	seriesIDs := make([]int64, len(series))
	for i, s := range series {
		seriesIDs[i] = s.ID
	}

	points, err := repo.GetSeriesPoints(ctx, &dataquery.PointsQueryRequest{
		SeriesIDs: seriesIDs,
		TimeRange: timeRange,
	})
	if err != nil {
		t.Fatalf("GetSeriesPoints() error = %v", err)
	}

	totalPoints := 0
	for _, p := range points {
		totalPoints += len(p)
	}

	t.Logf("Found %d total points across %d series", totalPoints, len(points))
}

// TestPGRepositoryGetAggregatedPoints tests aggregation
func TestPGRepositoryGetAggregatedPoints(t *testing.T) {
	pool := getTestDB(t)
	if pool == nil {
		return
	}
	defer pool.Close()

	repo := dataquery.NewPGRepository(pool)
	ctx := context.Background()

	now := time.Now()
	timeRange := dataquery.TimeRange{
		Start: now.Add(-24 * time.Hour),
		End:   now,
	}

	// First get series
	series, err := repo.QuerySeries(ctx, &dataquery.SeriesQueryRequest{
		TimeRange: timeRange,
		Limit:     5,
	})
	if err != nil {
		t.Fatalf("QuerySeries() error = %v", err)
	}

	if len(series) == 0 {
		t.Skip("No series found in database")
	}

	seriesIDs := make([]int64, len(series))
	for i, s := range series {
		seriesIDs[i] = s.ID
	}

	// Test each aggregation function
	aggFunctions := []string{"AVG", "MIN", "MAX", "SUM", "COUNT"}

	for _, fn := range aggFunctions {
		t.Run(fn, func(t *testing.T) {
			aggPoints, err := repo.GetAggregatedPoints(ctx, &dataquery.AggregationRequest{
				SeriesIDs: seriesIDs,
				TimeRange: timeRange,
				Interval:  "1h",
				Function:  fn,
			})
			if err != nil {
				t.Fatalf("GetAggregatedPoints() error = %v", err)
			}

			t.Logf("%s aggregation: %d buckets", fn, len(aggPoints))
		})
	}
}

// TestPGRepositoryGetSeriesStatistics tests statistics calculation
func TestPGRepositoryGetSeriesStatistics(t *testing.T) {
	pool := getTestDB(t)
	if pool == nil {
		return
	}
	defer pool.Close()

	repo := dataquery.NewPGRepository(pool)
	ctx := context.Background()

	now := time.Now()
	timeRange := dataquery.TimeRange{
		Start: now.Add(-24 * time.Hour),
		End:   now,
	}

	// First get series
	series, err := repo.QuerySeries(ctx, &dataquery.SeriesQueryRequest{
		TimeRange: timeRange,
		Limit:     5,
	})
	if err != nil {
		t.Fatalf("QuerySeries() error = %v", err)
	}

	if len(series) == 0 {
		t.Skip("No series found in database")
	}

	seriesIDs := make([]int64, len(series))
	for i, s := range series {
		seriesIDs[i] = s.ID
	}

	stats, err := repo.GetSeriesStatistics(ctx, &dataquery.StatsRequest{
		SeriesIDs: seriesIDs,
		TimeRange: timeRange,
	})
	if err != nil {
		t.Fatalf("GetSeriesStatistics() error = %v", err)
	}

	for id, s := range stats {
		t.Logf("Series %d: min=%.2f, max=%.2f, avg=%.2f, count=%d",
			id, s.Min, s.Max, s.Avg, s.Count)
	}
}

// TestLabelParserWithRealData tests label parser integration
func TestLabelParserWithRealData(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"simple match", `host="server1"`},
		{"AND expression", `host="server1" AND region="us-east"`},
		{"OR expression", `host="server1" OR host="server2"`},
		{"regex match", `region=~"us-.*"`},
		{"complex", `(host="server1" OR host="server2") AND env="prod"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := labels.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			sql, err := labels.ToSQL(expr)
			if err != nil {
				t.Fatalf("ToSQL() error = %v", err)
			}

			t.Logf("Input: %s -> SQL: %s", tt.input, sql)
		})
	}
}

// TestServiceIntegration tests the full service layer
func TestServiceIntegration(t *testing.T) {
	pool := getTestDB(t)
	if pool == nil {
		return
	}
	defer pool.Close()

	repo := dataquery.NewPGRepository(pool)
	svc := dataquery.NewService(repo)
	ctx := context.Background()

	// Initialize
	if err := svc.Initialize(ctx); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	// Test GetEndpoints
	endpoints, err := svc.GetEndpoints(ctx)
	if err != nil {
		t.Fatalf("GetEndpoints() error = %v", err)
	}
	t.Logf("Endpoints: %v", endpoints)

	// Test GetMetrics
	if len(endpoints) > 0 {
		metrics, err := svc.GetMetrics(ctx, endpoints[0])
		if err != nil {
			t.Fatalf("GetMetrics() error = %v", err)
		}
		t.Logf("Metrics for %s: %v", endpoints[0], metrics)
	}

	// Test QuerySeries
	now := time.Now()
	timeRange := dataquery.TimeRange{
		Start: now.Add(-24 * time.Hour),
		End:   now,
	}

	series, err := svc.QuerySeries(ctx, &dataquery.SeriesQuery{
		TimeRange: timeRange,
		Limit:     5,
	})
	if err != nil {
		t.Fatalf("QuerySeries() error = %v", err)
	}
	t.Logf("Found %d series", len(series))

	// Test QuerySeriesMulti
	multiSeries, err := svc.QuerySeriesMulti(ctx, &dataquery.MultiSeriesQuery{
		TimeRange: timeRange,
		Aggregation: &dataquery.Aggregation{
			Interval: "1h",
			Function: dataquery.AggAvg,
		},
	})
	if err != nil {
		t.Fatalf("QuerySeriesMulti() error = %v", err)
	}
	t.Logf("Found %d series in multi query", len(multiSeries))

	// Test GetSeriesByID
	if len(series) > 0 {
		byID, err := svc.GetSeriesByID(ctx, series[0].Meta.ID, &timeRange)
		if err != nil {
			t.Fatalf("GetSeriesByID() error = %v", err)
		}
		if byID != nil {
			t.Logf("Series %d has %d points and statistics: %+v",
				byID.Meta.ID, len(byID.Points), byID.Statistics)
		}
	}

	// Shutdown
	if err := svc.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
}

// TestComplexLabelFilterIntegration tests complex label filtering with real data
func TestComplexLabelFilterIntegration(t *testing.T) {
	pool := getTestDB(t)
	if pool == nil {
		return
	}
	defer pool.Close()

	repo := dataquery.NewPGRepository(pool)
	svc := dataquery.NewService(repo)
	ctx := context.Background()

	now := time.Now()
	timeRange := dataquery.TimeRange{
		Start: now.Add(-24 * time.Hour),
		End:   now,
	}

	tests := []struct {
		name   string
		filter string
	}{
		{"exact match", `host="server1"`},
		{"regex match", `region=~"us-.*"`},
		{"AND filter", `env="prod" AND region="us-east"`},
		{"OR filter", `host="server1" OR host="server2"`},
		{"complex filter", `(host="server1" OR host="server2") AND env="prod"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			series, err := svc.QuerySeries(ctx, &dataquery.SeriesQuery{
				LabelFilter: tt.filter,
				TimeRange:   timeRange,
				Limit:       10,
			})
			if err != nil {
				t.Fatalf("QuerySeries() error = %v", err)
			}

			t.Logf("Filter '%s' matched %d series", tt.filter, len(series))
		})
	}
}