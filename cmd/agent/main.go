package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/db-cockpit/pkg/agent"
	"github.com/db-cockpit/pkg/common/config"
	"github.com/db-cockpit/pkg/common/logger"
	"github.com/db-cockpit/pkg/data"
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

	logger.Info("Starting Execution Agent Service")

	// Initialize data layer
	dataLayer, err := data.InitializeDataLayer(cfg)
	if err != nil {
		logger.Fatal("Failed to initialize data layer", zap.Error(err))
	}
	defer dataLayer.Close()

	// Create execution agent
	execAgent := agent.NewExecutionAgent(dataLayer, nil)

	// Start agent
	ctx := context.Background()
	if err := execAgent.Start(ctx); err != nil {
		logger.Fatal("Failed to start execution agent", zap.Error(err))
	}

	logger.Info("Execution Agent started",
		zap.String("host", cfg.Server.Agent.Host),
		zap.Int("port", cfg.Server.Agent.Port),
	)

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down Execution Agent...")

	// Stop agent
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := execAgent.Stop(shutdownCtx); err != nil {
		logger.Error("Error stopping agent", zap.Error(err))
	}

	logger.Info("Execution Agent stopped")
}