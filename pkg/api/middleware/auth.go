package middleware

import (
	"context"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/db-cockpit/pkg/common/errors"
)

// AuthMiddleware handles authentication
func AuthMiddleware(jwtSecret string) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		// Get Authorization header
		authHeader := string(c.GetHeader("Authorization"))
		if authHeader == "" {
			c.AbortWithStatusJSON(401, map[string]string{
				"error": "Missing authorization header",
			})
			return
		}

		// Extract token
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.AbortWithStatusJSON(401, map[string]string{
				"error": "Invalid authorization header format",
			})
			return
		}

		token := parts[1]

		// Validate token
		claims, err := validateToken(token, jwtSecret)
		if err != nil {
			c.AbortWithStatusJSON(401, map[string]string{
				"error": "Invalid token",
			})
			return
		}

		// Set user info in context
		c.Set("user_id", claims.UserID)
		c.Set("tenant_id", claims.TenantID)
		c.Set("roles", claims.Roles)

		c.Next(ctx)
	}
}

// TokenClaims represents JWT token claims
type TokenClaims struct {
	UserID   string
	TenantID string
	Roles    []string
}

// validateToken validates a JWT token
func validateToken(token, secret string) (*TokenClaims, error) {
	// TODO: Implement actual JWT validation
	// This is a placeholder implementation

	if token == "" {
		return nil, errors.Unauthorized("Empty token", nil)
	}

	// For now, parse token as "tenantID:userID:roles"
	parts := strings.Split(token, ":")
	if len(parts) < 2 {
		return nil, errors.Unauthorized("Invalid token format", nil)
	}

	claims := &TokenClaims{
		TenantID: parts[0],
		UserID:   parts[1],
		Roles:    []string{"user"},
	}

	if len(parts) > 2 {
		claims.Roles = strings.Split(parts[2], ",")
	}

	return claims, nil
}

// GenerateToken generates a JWT token (for testing)
func GenerateToken(userID, tenantID string, roles []string, secret string) (string, error) {
	// TODO: Implement actual JWT generation
	// This is a placeholder implementation
	return tenantID + ":" + userID + ":" + strings.Join(roles, ","), nil
}

// OptionalAuthMiddleware allows optional authentication
func OptionalAuthMiddleware(jwtSecret string) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		authHeader := string(c.GetHeader("Authorization"))
		if authHeader == "" {
			c.Next(ctx)
			return
		}

		AuthMiddleware(jwtSecret)(ctx, c)
	}
}
