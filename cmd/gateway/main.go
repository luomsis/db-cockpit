package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	apihandler "github.com/db-cockpit/pkg/api/handler"
	"github.com/db-cockpit/pkg/api/router"
	"github.com/db-cockpit/pkg/common/config"
	"github.com/db-cockpit/pkg/common/logger"
	"github.com/db-cockpit/pkg/domain/llm"
	"github.com/db-cockpit/pkg/domain/performance"
	"github.com/db-cockpit/pkg/domain/sqlgovernance"
	"github.com/db-cockpit/pkg/domain/threshold"
	agentrpc "github.com/db-cockpit/pkg/rpc/agent"
	"github.com/db-cockpit/pkg/rpc/adapter"
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

	logger.Info("Starting Database Intelligent Cockpit API Gateway")

	// Initialize RPC client for Agent
	agentAddr := fmt.Sprintf("%s:%d", cfg.Server.Agent.Host, cfg.Server.Agent.Port)

	agentRPCClient, err := agentrpc.NewAgentClient(agentAddr)
	if err != nil {
		logger.Warn("Failed to connect to Agent RPC, using local fallback", zap.Error(err))
	}
	logger.Info("Agent RPC client initialized", zap.String("addr", agentAddr))

	// Initialize domain services with RPC clients
	var agentClient sqlgovernance.ExecutionAgentClient
	if agentRPCClient != nil {
		agentClient = adapter.NewAgentClientAdapter(agentRPCClient)
		logger.Info("SQL Governance using RPC Agent client")
	} else {
		agentClient = &localAgentClient{}
		logger.Info("SQL Governance using local Agent client (fallback)")
	}
	sqlGovernanceService := sqlgovernance.NewService(nil, agentClient)

	thresholdService := threshold.NewService(nil)
	thresholdClient := &localThresholdClient{service: thresholdService}
	performanceService := performance.NewService(nil, thresholdClient)
	llmService := llm.NewService(nil, nil)

	gatewayHandler := apihandler.NewGatewayHandler(
		sqlGovernanceService,
		performanceService,
		thresholdService,
		llmService,
	)

	// Create Hertz server
	h := server.Default(
		server.WithHostPorts(fmt.Sprintf("%s:%d", cfg.Server.Gateway.Host, cfg.Server.Gateway.Port)),
		server.WithDisablePrintRoute(false),
	)

	// Register REST routes
	router.RegisterRoutes(h, gatewayHandler, cfg.Auth.JWTSecret)

	// Data Query Service proxy target
	dataQueryAddr := fmt.Sprintf("http://%s:%d", cfg.Server.DataQuery.Host, cfg.Server.DataQuery.Port)

	// Register Data Query REST API proxy
	h.NoRoute(func(c context.Context, ctx *app.RequestContext) {
		path := string(ctx.URI().Path())
		if strings.HasPrefix(path, "/api/v1/endpoints") ||
			strings.HasPrefix(path, "/api/v1/metrics") ||
			strings.HasPrefix(path, "/api/v1/series") {
			proxyToDataQuery(ctx, dataQueryAddr)
		} else {
			ctx.AbortWithStatus(404)
		}
	})

	// Start server in goroutine
	go func() {
		if err := h.Run(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	logger.Info("API Gateway started",
		zap.String("host", cfg.Server.Gateway.Host),
		zap.Int("port", cfg.Server.Gateway.Port),
		zap.String("dataquery", dataQueryAddr),
	)

	printEndpoints(dataQueryAddr)

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down API Gateway...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_ = sqlGovernanceService.Shutdown(shutdownCtx)
	_ = performanceService.Shutdown(shutdownCtx)
	_ = thresholdService.Shutdown(shutdownCtx)
	_ = llmService.Shutdown(shutdownCtx)

	h.Shutdown(shutdownCtx)

	logger.Info("API Gateway stopped")
}

// proxyToDataQuery forwards requests to the Data Query Service
func proxyToDataQuery(ctx *app.RequestContext, targetAddr string) {
	// Build the target URL
	targetURL := targetAddr + string(ctx.URI().Path())
	if len(ctx.URI().QueryString()) > 0 {
		targetURL += "?" + string(ctx.URI().QueryString())
	}

	// Create the proxy request
	var body io.Reader
	if len(ctx.Request.Body()) > 0 {
		body = bytes.NewReader(ctx.Request.Body())
	}

	req, err := http.NewRequest(string(ctx.Method()), targetURL, body)
	if err != nil {
		ctx.JSON(500, map[string]string{"error": "Failed to create proxy request"})
		return
	}

	// Copy headers
	req.Header = make(http.Header)
	ctx.Request.Header.VisitAll(func(k, v []byte) {
		req.Header.Set(string(k), string(v))
	})

	// Send request to Data Query Service
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logger.Error("Failed to connect to Data Query Service", zap.Error(err))
		ctx.JSON(503, map[string]string{"error": "Data Query Service unavailable"})
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	for k, vs := range resp.Header {
		for _, v := range vs {
			ctx.Header(k, v)
		}
	}

	// Read and return response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		ctx.JSON(500, map[string]string{"error": "Failed to read response"})
		return
	}

	ctx.Data(resp.StatusCode, resp.Header.Get("Content-Type"), respBody)
}

func printEndpoints(dataQueryAddr string) {
	fmt.Println("\n========================================")
	fmt.Println("Database Intelligent Cockpit API Gateway")
	fmt.Println("========================================")
	fmt.Println("\n📡 Data Query Service REST API (proxied):")
	fmt.Println("  GET  http://localhost:8080/api/v1/endpoints")
	fmt.Println("  GET  http://localhost:8080/api/v1/metrics")
	fmt.Println("  GET  http://localhost:8080/api/v1/series")
	fmt.Printf("  → Proxied to: %s\n", dataQueryAddr)
	fmt.Println("\n🔐 Authentication:")
	fmt.Println("  Authorization: Bearer tenant_id:user_id:role")
	fmt.Println("  Example: Bearer tenant-001:user-001:admin")
	fmt.Println("\n📝 REST API Endpoints:")
	fmt.Println("  SQL Governance: POST /api/v1/sql/*")
	fmt.Println("  Performance:    POST /api/v1/performance/*")
	fmt.Println("  Thresholds:     GET/PUT /api/v1/thresholds")
	fmt.Println("  LLM:            POST /api/v1/llm/*")
	fmt.Println("\n========================================")
}

type localAgentClient struct{}

func (c *localAgentClient) ExecuteSQL(ctx context.Context, req *sqlgovernance.SQLExecuteRequest) (*sqlgovernance.SQLExecuteResult, error) {
	return &sqlgovernance.SQLExecuteResult{
		ExecutionID:     "local-" + time.Now().Format("20060102150405"),
		Status:          "completed",
		ExecutionTimeMs: 10,
		RowsAffected:    0,
	}, nil
}

func (c *localAgentClient) ExplainSQL(ctx context.Context, databaseID, sql string) (string, error) {
	return "EXPLAIN: " + sql, nil
}

type localThresholdClient struct {
	service threshold.ThresholdService
}

func (c *localThresholdClient) CheckThreshold(ctx context.Context, tenantID, databaseID, metricName string, value float64) (bool, string, error) {
	if c.service != nil {
		return c.service.CheckThreshold(ctx, tenantID, databaseID, metricName, value)
	}
	return false, "", nil
}

func (c *localThresholdClient) GetThresholds(ctx context.Context, tenantID, databaseID string, metricNames []string) (map[string]float64, error) {
	// Simple implementation without domain context for local fallback
	return make(map[string]float64), nil
}