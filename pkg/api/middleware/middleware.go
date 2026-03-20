package middleware

import (
	"context"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/db-cockpit/pkg/common/logger"
	"go.uber.org/zap"
)

// AuditLogger handles audit logging
type AuditLogger interface {
	Log(ctx context.Context, entry *AuditEntry) error
}

// AuditEntry represents an audit log entry
type AuditEntry struct {
	Timestamp    time.Time
	RequestID    string
	TenantID     string
	UserID       string
	Method       string
	Path         string
	Query        string
	StatusCode   int
	LatencyMs    int64
	RequestBody  string
	ResponseBody string
	ClientIP     string
	UserAgent    string
	Resource     string
	Action       string
	ResourceID   string
}

// AuditMiddleware logs requests for audit purposes
func AuditMiddleware(auditLogger AuditLogger) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		start := time.Now()

		// Generate request ID
		requestID := string(c.GetHeader("X-Request-ID"))
		if requestID == "" {
			requestID = generateRequestID()
			c.Header("X-Request-ID", requestID)
		}

		// Create audit entry
		entry := &AuditEntry{
			Timestamp: start,
			RequestID: requestID,
			Method:    string(c.Method()),
			Path:      string(c.Path()),
			Query:     string(c.URI().QueryString()),
			ClientIP:  c.ClientIP(),
			UserAgent: string(c.GetHeader("User-Agent")),
		}

		// Get user info from context
		if userID, exists := c.Get("user_id"); exists {
			entry.UserID = userID.(string)
		}
		if tenantID, exists := c.Get("tenant_id"); exists {
			entry.TenantID = tenantID.(string)
		}

		// Store entry in context for handlers to update
		c.Set("audit_entry", entry)

		// Process request
		c.Next(ctx)

		// Complete audit entry
		entry.StatusCode = c.Response.StatusCode()
		entry.LatencyMs = time.Since(start).Milliseconds()

		// Log audit entry
		if auditLogger != nil {
			if err := auditLogger.Log(ctx, entry); err != nil {
				logger.Error("Failed to log audit entry", zap.Error(err))
			}
		}

		// Log to application logs
		logger.Info("Request completed",
			zap.String("request_id", requestID),
			zap.String("method", entry.Method),
			zap.String("path", entry.Path),
			zap.Int("status", entry.StatusCode),
			zap.Int64("latency_ms", entry.LatencyMs),
			zap.String("user_id", entry.UserID),
			zap.String("tenant_id", entry.TenantID),
		)
	}
}

// MultiTenantMiddleware handles multi-tenancy
func MultiTenantMiddleware() app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		// Get tenant ID from header or context
		tenantID := string(c.GetHeader("X-Tenant-ID"))
		if tenantID == "" {
			// Try to get from context (set by auth middleware)
			if tid, exists := c.Get("tenant_id"); exists {
				tenantID = tid.(string)
			}
		}

		if tenantID == "" {
			c.AbortWithStatusJSON(400, map[string]string{
				"error": "Missing tenant ID",
			})
			return
		}

		// Set tenant ID in context
		c.Set("tenant_id", tenantID)

		c.Next(ctx)
	}
}

// RequestIDMiddleware adds a unique request ID to each request
func RequestIDMiddleware() app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		requestID := string(c.GetHeader("X-Request-ID"))
		if requestID == "" {
			requestID = generateRequestID()
		}

		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)

		c.Next(ctx)
	}
}

// CORSMiddleware handles CORS
func CORSMiddleware(allowedOrigins []string) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		origin := string(c.GetHeader("Origin"))

		// Check if origin is allowed
		allowed := false
		for _, o := range allowedOrigins {
			if o == "*" || o == origin {
				allowed = true
				break
			}
		}

		if allowed {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Tenant-ID, X-Request-ID")
			c.Header("Access-Control-Expose-Headers", "X-Request-ID")
		}

		if string(c.Method()) == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next(ctx)
	}
}

// RateLimitMiddleware handles rate limiting
func RateLimitMiddleware(requestsPerMinute int) app.HandlerFunc {
	// TODO: Implement actual rate limiting with Redis
	return func(ctx context.Context, c *app.RequestContext) {
		c.Next(ctx)
	}
}

// RecoveryMiddleware recovers from panics
func RecoveryMiddleware() app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		defer func() {
			if err := recover(); err != nil {
				logger.Error("Panic recovered",
					zap.Any("error", err),
					zap.String("path", string(c.Path())),
				)

				c.AbortWithStatusJSON(500, map[string]string{
					"error": "Internal server error",
				})
			}
		}()

		c.Next(ctx)
	}
}

// generateRequestID generates a unique request ID
func generateRequestID() string {
	return "req_" + time.Now().Format("20060102150405") + "_" + randomString(8)
}

// randomString generates a random string of given length
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[i%len(charset)]
	}
	return string(b)
}
