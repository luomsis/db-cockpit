package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/db-cockpit/pkg/common/config"
	"github.com/db-cockpit/pkg/common/logger"
	"github.com/db-cockpit/pkg/domain/dataquery"
	"github.com/db-cockpit/pkg/domain/dataquery/graph"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

var (
	configPath = flag.String("config", "configs/config.yaml", "Path to configuration file")
)

func main() {
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Printf("Failed to load config: %v, using defaults\n", err)
		cfg = config.DefaultConfig()
	}

	// Initialize logger
	if err := logger.Init(&logger.Config{
		Level:  cfg.Logging.Level,
		Format: cfg.Logging.Format,
		Output: cfg.Logging.Output,
	}); err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	logger.Info("Starting Data Query Service")

	ctx := context.Background()

	// Initialize PostgreSQL connection pool for TimescaleDB
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Database.TimescaleDB.Host,
		cfg.Database.TimescaleDB.Port,
		cfg.Database.TimescaleDB.User,
		cfg.Database.TimescaleDB.Password,
		cfg.Database.TimescaleDB.Database,
		cfg.Database.TimescaleDB.SSLMode,
	)

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		logger.Warn("Failed to connect to PostgreSQL, using mock data", zap.Error(err))
		pool = nil
	} else {
		logger.Info("PostgreSQL connection pool initialized")
	}

	// Initialize Data Query Service
	var dataQueryService dataquery.DataQueryService
	if pool != nil {
		repo := dataquery.NewPGRepository(pool)
		dataQueryService = dataquery.NewService(repo)
		logger.Info("Data Query Service initialized with PostgreSQL")
	} else {
		// Use mock service for development
		dataQueryService = dataquery.NewMockService()
		logger.Info("Data Query Service initialized with mock data")
	}

	// Create GraphQL handler
	resolver := graph.NewResolver(dataQueryService)
	graphqlHandler := handler.NewDefaultServer(graph.NewExecutableSchema(graph.Config{Resolvers: resolver}))
	playgroundHandler := playground.Handler("GraphQL Playground", "/graphql")

	// Setup HTTP routes
	mux := http.NewServeMux()
	mux.Handle("/graphql", graphqlHandler)
	mux.Handle("/graphql/playground", playgroundHandler)

	// Create HTTP server
	addr := fmt.Sprintf("%s:%d", cfg.Server.DataQuery.Host, cfg.Server.DataQuery.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	// Start server in goroutine
	go func() {
		logger.Info("Data Query Service started", zap.String("addr", addr))
		printEndpoints(addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down Data Query Service...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_ = dataQueryService.Shutdown(shutdownCtx)

	if pool != nil {
		pool.Close()
	}

	srv.Shutdown(shutdownCtx)

	logger.Info("Data Query Service stopped")
}

func printEndpoints(addr string) {
	fmt.Println("\n========================================")
	fmt.Println("Data Query Service (GraphQL)")
	fmt.Println("========================================")
	fmt.Println("\n📡 GraphQL Endpoint:")
	fmt.Printf("  POST http://%s/graphql\n", addr)
	fmt.Printf("  GET  http://%s/graphql/playground\n", addr)
	fmt.Println("\n📝 Example GraphQL Queries:")
	fmt.Print(`
  # Query all endpoints
  query {
    endpoints
  }

  # Query metrics for an endpoint
  query {
    metrics(endpoint: "/api/metrics")
  }

  # Query series with filters
  query($tr: TimeRangeInput!) {
    series(
      endpoint: "/api/metrics"
      metric: "cpu_usage"
      timeRange: $tr
      limit: 10
    ) {
      meta {
        id
        metric
        labels {
          entries { key value }
        }
      }
      points {
        time
        value
      }
    }
  }
`)
	fmt.Println("\n========================================")
}