package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/db-cockpit/pkg/domain/dataquery"
)

func main() {
	// Get database URL from environment
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/dbcockpit?sslmode=disable"
	}

	// Connect to database
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	// Create repository
	repo := dataquery.NewPGRepository(pool)

	// Ensure tables exist
	if err := repo.EnsureTables(ctx); err != nil {
		log.Fatalf("Failed to ensure tables: %v", err)
	}

	// Seed random
	rand.Seed(time.Now().UnixNano())

	// Define test data
	endpoints := []string{"/api/metrics", "/api/health", "/api/query"}
	metricsByEndpoint := map[string][]string{
		"/api/metrics": {"cpu_usage", "memory_usage", "disk_io", "network_bytes"},
		"/api/health":  {"response_time", "status_code"},
		"/api/query":   {"query_count", "query_latency"},
	}

	labelCombinations := []map[string]string{
		{"host": "server1", "region": "us-east", "env": "prod"},
		{"host": "server2", "region": "us-east", "env": "prod"},
		{"host": "server3", "region": "eu-west", "env": "prod"},
		{"host": "server1", "region": "us-east", "env": "staging"},
	}

	// Insert test data
	now := time.Now()
	startTime := now.Add(-24 * time.Hour)

	fmt.Println("Inserting test data...")
	totalSeries := 0
	totalPoints := 0

	for _, endpoint := range endpoints {
		metrics := metricsByEndpoint[endpoint]
		for _, metric := range metrics {
			for _, labels := range labelCombinations {
				// Insert series metadata
				seriesMeta, err := repo.InsertSeriesMeta(ctx, endpoint, metric, labels)
				if err != nil {
					log.Printf("Failed to insert series meta: %v", err)
					continue
				}
				totalSeries++

				// Generate data points (every 5 minutes for 24 hours = 288 points)
				var points []dataquery.DataPoint
				for t := startTime; t.Before(now); t = t.Add(5 * time.Minute) {
					// Generate realistic-ish values based on metric
					var value float64
					switch metric {
					case "cpu_usage":
						value = 30 + rand.Float64()*50 + 10*float64(t.Hour()%6) // 30-80% with hourly variation
					case "memory_usage":
						value = 40 + rand.Float64()*30 // 40-70%
					case "disk_io":
						value = 50 + rand.Float64()*100 // 50-150 MB/s
					case "network_bytes":
						value = 1000 + rand.Float64()*5000 // 1-6 KB/s
					case "response_time":
						value = 10 + rand.Float64()*100 // 10-110 ms
					case "status_code":
						if rand.Float64() > 0.95 {
							value = 500 // 5% errors
						} else {
							value = 200 // 95% success
						}
					case "query_count":
						value = 10 + rand.Float64()*50 // 10-60 queries per 5 min
					case "query_latency":
						value = 5 + rand.Float64()*50 // 5-55 ms
					}

					points = append(points, dataquery.DataPoint{
						Time:  t,
						Value: value,
					})
				}

				// Insert data points
				if err := repo.InsertPoints(ctx, seriesMeta.ID, points); err != nil {
					log.Printf("Failed to insert points for series %d: %v", seriesMeta.ID, err)
					continue
				}
				totalPoints += len(points)

				fmt.Printf("  Inserted series %d: %s/%s with labels %v (%d points)\n",
					seriesMeta.ID, endpoint, metric, labels, len(points))
			}
		}
	}

	fmt.Printf("\nDone! Inserted %d series with %d total data points.\n", totalSeries, totalPoints)
	fmt.Println("\nSample queries to try:")
	fmt.Print(`
# Query all endpoints
curl -s -X POST http://localhost:8080/graphql \
  -H "Content-Type: application/json" \
  -d '{"query": "{ endpoints }"}' | jq

# Query metrics for an endpoint
curl -s -X POST http://localhost:8080/graphql \
  -H "Content-Type: application/json" \
  -d '{"query": "{ metrics(endpoint: \"/api/metrics\") }"}' | jq

# Query series with time range
curl -s -X POST http://localhost:8080/graphql \
  -H "Content-Type: application/json" \
  -d '{
    "query": "query($tr: TimeRangeInput!) { series(endpoint: \"/api/metrics\", metric: \"cpu_usage\", timeRange: $tr) { meta { id endpoint metric labels { entries { key value } } } points { time value } } }",
    "variables": {
      "tr": {
        "start": "' + startTime.Format(time.RFC3339) + '",
        "end": "' + now.Format(time.RFC3339) + '"
      }
    }
  }' | jq

# Query series with label filter
curl -s -X POST http://localhost:8080/graphql \
  -H "Content-Type: application/json" \
  -d '{
    "query": "query($tr: TimeRangeInput!) { series(labels: {expression: \"host=\\\"server1\\\"\"}, timeRange: $tr) { meta { id endpoint metric labels { entries { key value } } } } }",
    "variables": {
      "tr": {
        "start": "' + startTime.Format(time.RFC3339) + '",
        "end": "' + now.Format(time.RFC3339) + '"
      }
    }
  }' | jq
`)
}