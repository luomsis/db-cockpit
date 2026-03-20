package domain

import (
	"context"
)

// DomainService defines the base interface for all domain services
type DomainService interface {
	// Name returns the domain service name
	Name() string

	// Initialize initializes the domain service
	Initialize(ctx context.Context) error

	// Shutdown shuts down the domain service
	Shutdown(ctx context.Context) error

	// Health returns the health status
	Health(ctx context.Context) error
}

// DomainContext holds context information for domain operations
type DomainContext struct {
	TenantID   string
	UserID     string
	RequestID  string
	DatabaseID string
	Ctx        context.Context
}

// NewDomainContext creates a new DomainContext
func NewDomainContext(ctx context.Context, tenantID, userID string) *DomainContext {
	return &DomainContext{
		Ctx:      ctx,
		TenantID: tenantID,
		UserID:   userID,
	}
}

// WithDatabaseID sets the database ID
func (d *DomainContext) WithDatabaseID(dbID string) *DomainContext {
	d.DatabaseID = dbID
	return d
}

// WithRequestID sets the request ID
func (d *DomainContext) WithRequestID(requestID string) *DomainContext {
	d.RequestID = requestID
	return d
}

// Context returns the underlying context
func (d *DomainContext) Context() context.Context {
	return d.Ctx
}
