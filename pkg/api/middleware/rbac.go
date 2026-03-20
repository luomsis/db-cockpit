package middleware

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
)

// Permission represents a permission
type Permission struct {
	Resource string
	Action   string
}

// Role represents a role with permissions
type Role struct {
	Name        string
	Permissions []Permission
}

// RBACConfig represents RBAC configuration
type RBACConfig struct {
	Roles map[string]Role
}

// DefaultRBACConfig returns default RBAC configuration
func DefaultRBACConfig() *RBACConfig {
	return &RBACConfig{
		Roles: map[string]Role{
			"admin": {
				Name: "admin",
				Permissions: []Permission{
					{Resource: "*", Action: "*"},
				},
			},
			"analyst": {
				Name: "analyst",
				Permissions: []Permission{
					{Resource: "queries", Action: "read"},
					{Resource: "queries", Action: "write"},
					{Resource: "reports", Action: "read"},
					{Resource: "diagnosis", Action: "read"},
				},
			},
			"viewer": {
				Name: "viewer",
				Permissions: []Permission{
					{Resource: "queries", Action: "read"},
					{Resource: "reports", Action: "read"},
				},
			},
			"developer": {
				Name: "developer",
				Permissions: []Permission{
					{Resource: "queries", Action: "read"},
					{Resource: "queries", Action: "write"},
					{Resource: "sql", Action: "execute"},
					{Resource: "diagnosis", Action: "read"},
				},
			},
		},
	}
}

// RBACMiddleware handles role-based access control
func RBACMiddleware(config *RBACConfig) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		// Get user roles from context
		rolesVal, exists := c.Get("roles")
		if !exists {
			c.AbortWithStatusJSON(403, map[string]string{
				"error": "No roles found",
			})
			return
		}

		roles, ok := rolesVal.([]string)
		if !ok {
			c.AbortWithStatusJSON(403, map[string]string{
				"error": "Invalid roles format",
			})
			return
		}

		// Get required permission from route metadata
		requiredPerm, exists := c.Get("required_permission")
		if !exists {
			// No permission required, allow access
			c.Next(ctx)
			return
		}

		perm, ok := requiredPerm.(Permission)
		if !ok {
			c.Next(ctx)
			return
		}

		// Check if any role has the required permission
		hasPermission := false
		for _, roleName := range roles {
			role, exists := config.Roles[roleName]
			if !exists {
				continue
			}

			if roleHasPermission(role, perm) {
				hasPermission = true
				break
			}
		}

		if !hasPermission {
			c.AbortWithStatusJSON(403, map[string]string{
				"error": "Permission denied",
			})
			return
		}

		c.Next(ctx)
	}
}

// roleHasPermission checks if a role has a specific permission
func roleHasPermission(role Role, perm Permission) bool {
	for _, p := range role.Permissions {
		if p.Resource == "*" || p.Resource == perm.Resource {
			if p.Action == "*" || p.Action == perm.Action {
				return true
			}
		}
	}
	return false
}

// RequirePermission creates a middleware that requires a specific permission
func RequirePermission(resource, action string) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		c.Set("required_permission", Permission{
			Resource: resource,
			Action:   action,
		})
		c.Next(ctx)
	}
}

// RequireRole creates a middleware that requires a specific role
func RequireRole(requiredRoles ...string) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		rolesVal, exists := c.Get("roles")
		if !exists {
			c.AbortWithStatusJSON(403, map[string]string{
				"error": "No roles found",
			})
			return
		}

		userRoles, ok := rolesVal.([]string)
		if !ok {
			c.AbortWithStatusJSON(403, map[string]string{
				"error": "Invalid roles format",
			})
			return
		}

		hasRole := false
		for _, userRole := range userRoles {
			for _, requiredRole := range requiredRoles {
				if userRole == requiredRole {
					hasRole = true
					break
				}
			}
		}

		if !hasRole {
			c.AbortWithStatusJSON(403, map[string]string{
				"error": "Role required",
			})
			return
		}

		c.Next(ctx)
	}
}
