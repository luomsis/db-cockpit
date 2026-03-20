package middleware

import (
	"context"
	"testing"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
)

// =====================
// Auth Middleware Tests
// =====================

func TestAuthMiddleware_MissingHeader(t *testing.T) {
	middleware := AuthMiddleware("test-secret")
	ctx := context.Background()
	reqCtx := &app.RequestContext{}

	middleware(ctx, reqCtx)

	if reqCtx.Response.StatusCode() != 401 {
		t.Errorf("Status = %d, want 401", reqCtx.Response.StatusCode())
	}
}

func TestAuthMiddleware_InvalidFormat(t *testing.T) {
	tests := []string{
		"InvalidToken",
		"Basic token",
		"Bearer",
		"bearer token",
	}

	for _, token := range tests {
		t.Run(token, func(t *testing.T) {
			middleware := AuthMiddleware("test-secret")
			ctx := context.Background()
			reqCtx := &app.RequestContext{}
			reqCtx.Request.Header.Set("Authorization", token)

			middleware(ctx, reqCtx)

			if reqCtx.Response.StatusCode() != 401 {
				t.Errorf("Status = %d, want 401 for token: %s", reqCtx.Response.StatusCode(), token)
			}
		})
	}
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	middleware := AuthMiddleware("test-secret")
	ctx := context.Background()
	reqCtx := &app.RequestContext{}
	reqCtx.Request.Header.Set("Authorization", "Bearer tenant-1:user-1:admin")

	middleware(ctx, reqCtx)

	// Should not abort (status should be 0 or continue)
	if reqCtx.Response.StatusCode() == 401 {
		t.Error("Valid token should not result in 401")
	}

	// Check context values
	if tid, exists := reqCtx.Get("tenant_id"); !exists || tid != "tenant-1" {
		t.Errorf("tenant_id = %v, want tenant-1", tid)
	}
	if uid, exists := reqCtx.Get("user_id"); !exists || uid != "user-1" {
		t.Errorf("user_id = %v, want user-1", uid)
	}
}

func TestAuthMiddleware_TokenParsing(t *testing.T) {
	tests := []struct {
		token     string
		tenantID  string
		userID    string
		roles     string
	}{
		{"tenant-1:user-1", "tenant-1", "user-1", "user"},
		{"tenant-2:user-2:admin", "tenant-2", "user-2", "admin"},
		{"t:u:admin,editor", "t", "u", "admin,editor"},
	}

	for _, tt := range tests {
		t.Run(tt.token, func(t *testing.T) {
			claims, err := validateToken(tt.token, "secret")
			if err != nil {
				t.Fatalf("validateToken() error = %v", err)
			}

			if claims.TenantID != tt.tenantID {
				t.Errorf("TenantID = %s, want %s", claims.TenantID, tt.tenantID)
			}
			if claims.UserID != tt.userID {
				t.Errorf("UserID = %s, want %s", claims.UserID, tt.userID)
			}
		})
	}
}

func TestValidateToken_EmptyToken(t *testing.T) {
	_, err := validateToken("", "secret")
	if err == nil {
		t.Error("validateToken() should return error for empty token")
	}
}

func TestValidateToken_InvalidFormat(t *testing.T) {
	_, err := validateToken("invalid-token-format", "secret")
	if err == nil {
		t.Error("validateToken() should return error for invalid format")
	}
}

func TestGenerateToken(t *testing.T) {
	token, err := GenerateToken("user-1", "tenant-1", []string{"admin"}, "secret")
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	if token == "" {
		t.Error("GenerateToken() should return non-empty token")
	}

	// Token format is tenantID:userID:roles
	if token != "tenant-1:user-1:admin" {
		t.Errorf("Token = %s, want tenant-1:user-1:admin", token)
	}
}

func TestOptionalAuthMiddleware_NoHeader(t *testing.T) {
	middleware := OptionalAuthMiddleware("test-secret")
	ctx := context.Background()
	reqCtx := &app.RequestContext{}
	called := false

	// Add a next handler
	middleware(ctx, reqCtx)

	// Should continue without error (not abort)
	if reqCtx.Response.StatusCode() == 401 {
		t.Error("OptionalAuth should not return 401 when no header")
	}

	_ = called // Avoid unused variable warning
}

// =====================
// Multi-Tenant Middleware Tests
// =====================

func TestMultiTenantMiddleware_MissingTenantID(t *testing.T) {
	middleware := MultiTenantMiddleware()
	ctx := context.Background()
	reqCtx := &app.RequestContext{}

	middleware(ctx, reqCtx)

	if reqCtx.Response.StatusCode() != 400 {
		t.Errorf("Status = %d, want 400", reqCtx.Response.StatusCode())
	}
}

func TestMultiTenantMiddleware_FromHeader(t *testing.T) {
	middleware := MultiTenantMiddleware()
	ctx := context.Background()
	reqCtx := &app.RequestContext{}
	reqCtx.Request.Header.Set("X-Tenant-ID", "tenant-123")

	middleware(ctx, reqCtx)

	if reqCtx.Response.StatusCode() == 400 {
		t.Error("Should not return 400 when tenant ID is in header")
	}

	tid, exists := reqCtx.Get("tenant_id")
	if !exists || tid != "tenant-123" {
		t.Errorf("tenant_id = %v, want tenant-123", tid)
	}
}

func TestMultiTenantMiddleware_FromContext(t *testing.T) {
	middleware := MultiTenantMiddleware()
	ctx := context.Background()
	reqCtx := &app.RequestContext{}
	reqCtx.Set("tenant_id", "tenant-456")

	middleware(ctx, reqCtx)

	if reqCtx.Response.StatusCode() == 400 {
		t.Error("Should not return 400 when tenant ID is in context")
	}

	tid, exists := reqCtx.Get("tenant_id")
	if !exists || tid != "tenant-456" {
		t.Errorf("tenant_id = %v, want tenant-456", tid)
	}
}

// =====================
// CORS Middleware Tests
// =====================

func TestCORSMiddleware_AllowAll(t *testing.T) {
	middleware := CORSMiddleware([]string{"*"})
	ctx := context.Background()
	reqCtx := &app.RequestContext{}
	reqCtx.Request.Header.Set("Origin", "http://localhost:3000")

	middleware(ctx, reqCtx)

	// Check CORS headers are set
	if reqCtx.Response.Header.Get("Access-Control-Allow-Origin") == "" {
		t.Error("Access-Control-Allow-Origin header should be set")
	}
}

func TestCORSMiddleware_OptionsRequest(t *testing.T) {
	middleware := CORSMiddleware([]string{"*"})
	ctx := context.Background()
	reqCtx := &app.RequestContext{}
	reqCtx.Request.SetMethod("OPTIONS")
	reqCtx.Request.Header.Set("Origin", "http://localhost:3000")

	middleware(ctx, reqCtx)

	if reqCtx.Response.StatusCode() != 204 {
		t.Errorf("OPTIONS request status = %d, want 204", reqCtx.Response.StatusCode())
	}
}

func TestCORSMiddleware_SpecificOrigin(t *testing.T) {
	allowedOrigins := []string{"http://localhost:3000", "http://example.com"}
	middleware := CORSMiddleware(allowedOrigins)

	tests := []struct {
		origin   string
		allowed  bool
	}{
		{"http://localhost:3000", true},
		{"http://example.com", true},
		{"http://notallowed.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.origin, func(t *testing.T) {
			reqCtx := &app.RequestContext{}
			reqCtx.Request.Header.Set("Origin", tt.origin)

			middleware(context.Background(), reqCtx)

			allowOrigin := reqCtx.Response.Header.Get("Access-Control-Allow-Origin")
			if tt.allowed && allowOrigin == "" {
				t.Errorf("Origin %s should be allowed", tt.origin)
			}
		})
	}
}

func TestCORSMiddleware_AllowedMethods(t *testing.T) {
	middleware := CORSMiddleware([]string{"*"})
	ctx := context.Background()
	reqCtx := &app.RequestContext{}
	reqCtx.Request.Header.Set("Origin", "http://localhost:3000")

	middleware(ctx, reqCtx)

	allowMethods := reqCtx.Response.Header.Get("Access-Control-Allow-Methods")
	if allowMethods == "" {
		t.Error("Access-Control-Allow-Methods header should be set")
	}

	expectedMethods := []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	for _, method := range expectedMethods {
		if !contains(allowMethods, method) {
			t.Errorf("Access-Control-Allow-Methods should include %s", method)
		}
	}
}

// =====================
// Request ID Middleware Tests
// =====================

func TestRequestIDMiddleware_Generate(t *testing.T) {
	middleware := RequestIDMiddleware()
	ctx := context.Background()
	reqCtx := &app.RequestContext{}

	middleware(ctx, reqCtx)

	// Check request ID is set
	rid, exists := reqCtx.Get("request_id")
	if !exists || rid == "" {
		t.Error("request_id should be set")
	}

	// Check header is set
	if reqCtx.Response.Header.Get("X-Request-ID") == "" {
		t.Error("X-Request-ID header should be set")
	}
}

func TestRequestIDMiddleware_UseExisting(t *testing.T) {
	middleware := RequestIDMiddleware()
	ctx := context.Background()
	reqCtx := &app.RequestContext{}
	reqCtx.Request.Header.Set("X-Request-ID", "existing-request-id")

	middleware(ctx, reqCtx)

	rid, exists := reqCtx.Get("request_id")
	if !exists || rid != "existing-request-id" {
		t.Errorf("request_id = %v, want existing-request-id", rid)
	}
}

func TestGenerateRequestID(t *testing.T) {
	id1 := generateRequestID()

	// Check format: req_YYYYMMDDHHMMSS_randomstring
	if len(id1) < 4 {
		t.Error("Request ID should have some length")
	}

	if id1[:4] != "req_" {
		t.Error("Request ID should start with 'req_'")
	}

	// Note: randomString uses deterministic generation (i%len(charset))
	// so IDs generated in the same second may be identical
	t.Logf("Generated request ID: %s", id1)
}

func TestRandomString(t *testing.T) {
	tests := []int{8, 16, 32}

	for _, length := range tests {
		t.Run(string(rune(length)), func(t *testing.T) {
			s := randomString(length)
			if len(s) != length {
				t.Errorf("randomString(%d) returned length %d", length, len(s))
			}
		})
	}
}

// =====================
// Recovery Middleware Tests
// =====================

func TestRecoveryMiddleware_NoPanic(t *testing.T) {
	middleware := RecoveryMiddleware()
	ctx := context.Background()
	reqCtx := &app.RequestContext{}

	// Should not panic
	middleware(ctx, reqCtx)

	if reqCtx.Response.StatusCode() == 500 {
		t.Error("Should not return 500 when no panic")
	}
}

func TestRecoveryMiddleware_WithPanic(t *testing.T) {
	middleware := RecoveryMiddleware()
	ctx := context.Background()
	reqCtx := &app.RequestContext{}

	// Create a handler that panics
	panicHandler := func(c context.Context, rc *app.RequestContext) {
		panic("test panic")
	}

	// Set a next handler that panics
	reqCtx.Set("next_handler", panicHandler)

	// Manually trigger the middleware and simulate panic
	defer func() {
		// The middleware should have recovered
	}()

	// Call middleware - it should recover from panic
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Log("Recovered in test")
			}
		}()
		middleware(ctx, reqCtx)
	}()
}

// =====================
// Rate Limit Middleware Tests
// =====================

func TestRateLimitMiddleware(t *testing.T) {
	middleware := RateLimitMiddleware(100)
	ctx := context.Background()
	reqCtx := &app.RequestContext{}

	// Currently a no-op, should just pass through
	middleware(ctx, reqCtx)

	// Should not abort - RateLimitMiddleware is a no-op that calls c.Next(ctx) internally
	// Status code 0 means no response was written (middleware passed through)
	if reqCtx.Response.StatusCode() == 401 || reqCtx.Response.StatusCode() == 429 {
		t.Errorf("RateLimitMiddleware should pass through, got status %d", reqCtx.Response.StatusCode())
	}
}

// =====================
// Audit Middleware Tests
// =====================

func TestAuditMiddleware_Basic(t *testing.T) {
	middleware := AuditMiddleware(nil) // nil logger for testing
	ctx := context.Background()
	reqCtx := &app.RequestContext{}
	reqCtx.Request.SetMethod("GET")
	reqCtx.Request.SetRequestURI("/api/v1/test")

	middleware(ctx, reqCtx)

	// Check audit entry is stored
	_, exists := reqCtx.Get("audit_entry")
	if !exists {
		t.Error("audit_entry should be set in context")
	}
}

func TestAuditMiddleware_WithUserInfo(t *testing.T) {
	middleware := AuditMiddleware(nil)
	ctx := context.Background()
	reqCtx := &app.RequestContext{}
	reqCtx.Set("user_id", "user-123")
	reqCtx.Set("tenant_id", "tenant-456")
	reqCtx.Request.SetMethod("POST")
	reqCtx.Request.SetRequestURI("/api/v1/sql/execute")

	middleware(ctx, reqCtx)

	entry, exists := reqCtx.Get("audit_entry")
	if !exists {
		t.Fatal("audit_entry should be set")
	}

	auditEntry, ok := entry.(*AuditEntry)
	if !ok {
		t.Fatal("audit_entry should be *AuditEntry")
	}

	if auditEntry.UserID != "user-123" {
		t.Errorf("UserID = %s, want user-123", auditEntry.UserID)
	}
	if auditEntry.TenantID != "tenant-456" {
		t.Errorf("TenantID = %s, want tenant-456", auditEntry.TenantID)
	}
}

func TestAuditMiddleware_RequestIDFromHeader(t *testing.T) {
	middleware := AuditMiddleware(nil)
	ctx := context.Background()
	reqCtx := &app.RequestContext{}
	reqCtx.Request.Header.Set("X-Request-ID", "test-req-id")

	middleware(ctx, reqCtx)

	entry, exists := reqCtx.Get("audit_entry")
	if !exists {
		t.Fatal("audit_entry should be set")
	}

	auditEntry, ok := entry.(*AuditEntry)
	if !ok {
		t.Fatal("audit_entry should be *AuditEntry")
	}

	if auditEntry.RequestID != "test-req-id" {
		t.Errorf("RequestID = %s, want test-req-id", auditEntry.RequestID)
	}
}

func TestAuditMiddleware_GenerateRequestID(t *testing.T) {
	middleware := AuditMiddleware(nil)
	ctx := context.Background()
	reqCtx := &app.RequestContext{}
	// No X-Request-ID header

	middleware(ctx, reqCtx)

	entry, exists := reqCtx.Get("audit_entry")
	if !exists {
		t.Fatal("audit_entry should be set")
	}

	auditEntry, ok := entry.(*AuditEntry)
	if !ok {
		t.Fatal("audit_entry should be *AuditEntry")
	}

	if auditEntry.RequestID == "" {
		t.Error("RequestID should be generated if not provided")
	}
}

func TestAuditEntry_Fields(t *testing.T) {
	now := time.Now()
	entry := &AuditEntry{
		Timestamp:  now,
		RequestID:  "req-1",
		TenantID:   "tenant-1",
		UserID:     "user-1",
		Method:     "POST",
		Path:       "/api/v1/sql/execute",
		Query:      "db=test",
		StatusCode: 200,
		LatencyMs:  50,
		ClientIP:   "127.0.0.1",
		UserAgent:  "test-agent",
	}

	if entry.RequestID != "req-1" {
		t.Errorf("RequestID = %s, want req-1", entry.RequestID)
	}
	if entry.Method != "POST" {
		t.Errorf("Method = %s, want POST", entry.Method)
	}
	if entry.LatencyMs != 50 {
		t.Errorf("LatencyMs = %d, want 50", entry.LatencyMs)
	}
}

// =====================
// Mock Audit Logger Tests
// =====================

type mockAuditLogger struct {
	entries []*AuditEntry
}

func (m *mockAuditLogger) Log(ctx context.Context, entry *AuditEntry) error {
	m.entries = append(m.entries, entry)
	return nil
}

func TestAuditMiddleware_WithLogger(t *testing.T) {
	logger := &mockAuditLogger{}
	middleware := AuditMiddleware(logger)
	ctx := context.Background()
	reqCtx := &app.RequestContext{}
	reqCtx.Request.SetMethod("GET")
	reqCtx.Request.SetRequestURI("/api/v1/test")

	middleware(ctx, reqCtx)

	if len(logger.entries) != 1 {
		t.Errorf("Logger entries count = %d, want 1", len(logger.entries))
	}
}

// =====================
// Helper Functions
// =====================

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// =====================
// Benchmark Tests
// =====================

func BenchmarkAuthMiddleware(b *testing.B) {
	middleware := AuthMiddleware("test-secret")
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reqCtx := &app.RequestContext{}
		reqCtx.Request.Header.Set("Authorization", "Bearer tenant-1:user-1:admin")
		middleware(ctx, reqCtx)
	}
}

func BenchmarkCORSMiddleware(b *testing.B) {
	middleware := CORSMiddleware([]string{"*"})
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reqCtx := &app.RequestContext{}
		reqCtx.Request.Header.Set("Origin", "http://localhost:3000")
		middleware(ctx, reqCtx)
	}
}

func BenchmarkRequestIDMiddleware(b *testing.B) {
	middleware := RequestIDMiddleware()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reqCtx := &app.RequestContext{}
		middleware(ctx, reqCtx)
	}
}

func BenchmarkAuditMiddleware(b *testing.B) {
	middleware := AuditMiddleware(nil)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reqCtx := &app.RequestContext{}
		reqCtx.Request.SetMethod("GET")
		reqCtx.Request.SetRequestURI("/api/v1/test")
		middleware(ctx, reqCtx)
	}
}