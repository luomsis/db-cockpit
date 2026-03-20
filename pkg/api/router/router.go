package router

import (
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/db-cockpit/pkg/api/handler"
	"github.com/db-cockpit/pkg/api/middleware"
)

// RegisterRoutes registers all routes
// Note: Data Query operations use GraphQL endpoint at /graphql
func RegisterRoutes(h *server.Hertz, gatewayHandler *handler.GatewayHandler, jwtSecret string) {
	// Global middleware
	h.Use(middleware.RecoveryMiddleware())
	h.Use(middleware.RequestIDMiddleware())
	h.Use(middleware.CORSMiddleware([]string{"*"}))

	// Health check (no auth required)
	h.GET("/health", gatewayHandler.Health)
	h.GET("/api/v1/health", gatewayHandler.Health)

	// API v1 routes
	v1 := h.Group("/api/v1")
	{
		// Auth-protected routes
		authGroup := v1.Group("")
		authGroup.Use(middleware.AuthMiddleware(jwtSecret))
		authGroup.Use(middleware.MultiTenantMiddleware())
		authGroup.Use(middleware.AuditMiddleware(nil))
		{
			// SQL Governance endpoints
			sqlGroup := authGroup.Group("/sql")
			{
				sqlGroup.POST("/review", gatewayHandler.ReviewSQL)
				sqlGroup.POST("/execute", gatewayHandler.ExecuteSQL)
				sqlGroup.GET("/audit", gatewayHandler.GetSQLAudit)
			}

			// Performance endpoints
			perfGroup := authGroup.Group("/performance")
			{
				perfGroup.POST("/diagnose", gatewayHandler.Diagnose)
				perfGroup.POST("/metrics", gatewayHandler.GetMetrics)
				perfGroup.POST("/slow-queries", gatewayHandler.GetSlowQueries)
			}

			// Threshold endpoints
			thresholdGroup := authGroup.Group("/thresholds")
			{
				thresholdGroup.GET("", gatewayHandler.GetThresholds)
				thresholdGroup.PUT("", gatewayHandler.UpdateThreshold)
			}

			// LLM endpoints
			llmGroup := authGroup.Group("/llm")
			{
				llmGroup.POST("/chat", gatewayHandler.Chat)
				llmGroup.POST("/generate-sql", gatewayHandler.GenerateSQL)
				llmGroup.GET("/recommendations", gatewayHandler.GetRecommendations)
			}
		}
	}
}
