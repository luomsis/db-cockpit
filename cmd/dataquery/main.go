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

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/db-cockpit/pkg/common/config"
	"github.com/db-cockpit/pkg/common/logger"
	"github.com/db-cockpit/pkg/domain/dataquery"
	_ "github.com/db-cockpit/docs" // swagger docs
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

var (
	configPath = flag.String("config", "configs/config.yaml", "Path to configuration file")
)

// @title Data Query Service API
// @version 1.0
// @description RESTful API for querying time series data from TimescaleDB
// @BasePath /api/v1
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
		logger.Fatal("Failed to connect to PostgreSQL", zap.Error(err))
	}
	logger.Info("PostgreSQL connection pool initialized")

	// Initialize Data Query Service
	repo := dataquery.NewPGRepository(pool)
	dataQueryService := dataquery.NewService(repo)
	logger.Info("Data Query Service initialized with PostgreSQL")

	// Create REST handler
	handler := dataquery.NewHandler(dataQueryService)

	// Create Hertz server
	addr := fmt.Sprintf("%s:%d", cfg.Server.DataQuery.Host, cfg.Server.DataQuery.Port)
	h := server.Default(
		server.WithHostPorts(addr),
		server.WithDisablePrintRoute(false),
	)

	// Add request logging middleware
	h.Use(func(ctx context.Context, c *app.RequestContext) {
		start := time.Now()

		// Log incoming request
		logger.Info("Request received",
			zap.String("method", string(c.Method())),
			zap.String("path", string(c.Path())),
			zap.String("query", string(c.URI().QueryString())),
			zap.String("client_ip", c.ClientIP()),
		)

		// Process request
		c.Next(ctx)

		// Log response
		latency := time.Since(start)
		logger.Info("Request completed",
			zap.String("method", string(c.Method())),
			zap.String("path", string(c.Path())),
			zap.Int("status", c.Response.StatusCode()),
			zap.Duration("latency", latency),
			zap.Int("response_size", len(c.Response.Body())),
		)
	})

	// Register REST routes
	api := h.Group("/api/v1")
	{
		api.GET("/endpoints", func(c context.Context, ctx *app.RequestContext) {
			handler.GetEndpoints(c, ctx)
		})
		api.GET("/metrics", func(c context.Context, ctx *app.RequestContext) {
			handler.GetMetrics(c, ctx)
		})
		api.GET("/series", func(c context.Context, ctx *app.RequestContext) {
			handler.GetSeries(c, ctx)
		})
		api.GET("/series/:id", func(c context.Context, ctx *app.RequestContext) {
			handler.GetSeriesByID(c, ctx)
		})
		api.POST("/series/query", func(c context.Context, ctx *app.RequestContext) {
			handler.QuerySeries(c, ctx)
		})
		api.GET("/instances", func(c context.Context, ctx *app.RequestContext) {
			handler.GetInstances(c, ctx)
		})
		api.GET("/instances/:endpoint", func(c context.Context, ctx *app.RequestContext) {
			handler.GetInstance(c, ctx)
		})
		api.GET("/alerts", func(c context.Context, ctx *app.RequestContext) {
			handler.GetAlerts(c, ctx)
		})
		api.GET("/slow-queries", func(c context.Context, ctx *app.RequestContext) {
			handler.GetSlowQueries(c, ctx)
		})
	}

	// Health check endpoint
	h.GET("/health", func(c context.Context, ctx *app.RequestContext) {
		ctx.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	// Swagger UI (using local static files)
	swaggerHandler := dataquery.SwaggerUIHandler()
	h.GET("/swagger/*any", func(c context.Context, ctx *app.RequestContext) {
		swaggerHandler(c, ctx)
	})

	// Start server in goroutine
	go func() {
		logger.Info("Data Query Service started", zap.String("addr", addr))
		printEndpoints(addr)
		if err := h.Run(); err != nil && err != http.ErrServerClosed {
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

	h.Shutdown(shutdownCtx)

	logger.Info("Data Query Service stopped")
}

func printEndpoints(addr string) {
	fmt.Println("\n========================================")
	fmt.Println("Data Query Service (REST API)")
	fmt.Println("========================================")
	fmt.Println("\n📡 REST API Endpoints:")
	fmt.Printf("  GET  http://%s/api/v1/endpoints\n", addr)
	fmt.Printf("  GET  http://%s/api/v1/metrics?endpoint=<endpoint>\n", addr)
	fmt.Printf("  GET  http://%s/api/v1/series?endpoint=<ep>&metric=<m>&start=<t>&end=<t>\n", addr)
	fmt.Printf("  GET  http://%s/api/v1/series/:id\n", addr)
	fmt.Printf("  POST http://%s/api/v1/series/query\n", addr)
	fmt.Printf("  GET  http://%s/api/v1/instances\n", addr)
	fmt.Printf("  GET  http://%s/api/v1/instances/:endpoint\n", addr)
	fmt.Printf("  GET  http://%s/api/v1/alerts\n", addr)
	fmt.Printf("  GET  http://%s/api/v1/slow-queries?hostname=<host>&port=<port>\n", addr)
	fmt.Printf("  GET  http://%s/health\n", addr)
	fmt.Printf("\n📖 Swagger UI: http://%s/swagger/index.html\n", addr)
	fmt.Println("\n📝 Example REST API Requests:")
	fmt.Print(`
  # Get all endpoints
  curl http://localhost:8084/api/v1/endpoints

  # Get metrics for an endpoint
  curl "http://localhost:8084/api/v1/metrics?endpoint=mysql-cn-east-1-finance-order-01"

  # Query series with filters
  curl "http://localhost:8084/api/v1/series?endpoint=mysql-cn-east-1-finance-order-01&metric=cpu_usage_percent&start=2024-01-01T00:00:00Z&end=2024-12-31T00:00:00Z&limit=10"

  # Get series by ID
  curl http://localhost:8084/api/v1/series/1

  # Complex query with POST
  curl -X POST http://localhost:8084/api/v1/series/query \
    -H "Content-Type: application/json" \
    -d '{
      "endpoints": ["mysql-cn-east-1-finance-order-01"],
      "metrics": ["cpu_usage_percent"],
      "start": "2024-01-01T00:00:00Z",
      "end": "2024-12-31T00:00:00Z"
    }'

  # Get all instances
  curl http://localhost:8084/api/v1/instances

  # Get instance metadata by endpoint
  curl http://localhost:8084/api/v1/instances/mysql-cn-east-1-finance-order-01

  # Get alerts with filters
  curl "http://localhost:8084/api/v1/alerts?endpoint=pg-cn-north-2-ecom-user-01"
  curl "http://localhost:8084/api/v1/alerts?metric=cpu_usage&status=firing"
`)
	fmt.Println("\n========================================")
}