package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Test configuration
const (
	gatewayPort    = 18080
	dataQueryPort  = 18084
	testTenantID   = "test-tenant"
	testUserID     = "test-user"
	testDatabaseID = "test-db"
)

// curlResponse represents the response from curl command
type curlResponse struct {
	StatusCode int
	Body       string
	Headers    map[string]string
}

// curl executes a curl command and returns the response
func curl(t *testing.T, method, url string, headers map[string]string, body string) *curlResponse {
	args := []string{
		"-s",
		"-w", "\n%{http_code}",
		"-X", method,
	}

	// Add headers
	for k, v := range headers {
		args = append(args, "-H", fmt.Sprintf("%s: %s", k, v))
	}

	// Add body if present
	if body != "" {
		args = append(args, "-d", body)
	}

	args = append(args, url)

	cmd := exec.Command("curl", args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		t.Logf("curl command failed: %v", err)
		return &curlResponse{StatusCode: 0, Body: "", Headers: make(map[string]string)}
	}

	output := out.String()
	// Split status code from body
	parts := strings.Split(output, "\n")
	if len(parts) < 2 {
		return &curlResponse{StatusCode: 0, Body: output, Headers: make(map[string]string)}
	}

	bodyPart := strings.Join(parts[:len(parts)-1], "\n")
	statusPart := parts[len(parts)-1]

	var statusCode int
	fmt.Sscanf(statusPart, "%d", &statusCode)

	return &curlResponse{
		StatusCode: statusCode,
		Body:       bodyPart,
		Headers:    make(map[string]string),
	}
}

// TestDatabaseConnection tests database connectivity
func TestDatabaseConnection(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Skipf("Skipping: cannot connect to database: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		t.Skipf("Skipping: cannot ping database: %v", err)
	}

	t.Log("Database connection successful")
}

// TestGatewayHealth tests the gateway health endpoint
func TestGatewayHealth(t *testing.T) {
	gatewayURL := os.Getenv("GATEWAY_URL")
	if gatewayURL == "" {
		gatewayURL = fmt.Sprintf("http://localhost:%d", gatewayPort)
	}

	// Test health endpoint
	t.Run("HealthEndpoint", func(t *testing.T) {
		resp := curl(t, "GET", gatewayURL+"/health", nil, "")
		if resp.StatusCode != 200 {
			t.Logf("Note: Gateway may not be running. Response: %s", resp.Body)
			t.Skip("Gateway not running")
		}

		var result map[string]interface{}
		if err := json.Unmarshal([]byte(resp.Body), &result); err != nil {
			t.Errorf("Failed to parse health response: %v", err)
			return
		}

		if result["status"] != "ok" {
			t.Errorf("Health status = %v, want ok", result["status"])
		}
		t.Logf("Health check passed: %s", resp.Body)
	})

	// Test API health endpoint
	t.Run("APIHealthEndpoint", func(t *testing.T) {
		resp := curl(t, "GET", gatewayURL+"/api/v1/health", nil, "")
		if resp.StatusCode != 200 {
			t.Logf("API health endpoint returned status %d", resp.StatusCode)
		}
	})
}

// TestSQLGovernanceAPI tests the SQL Governance REST API through gateway
func TestSQLGovernanceAPI(t *testing.T) {
	gatewayURL := os.Getenv("GATEWAY_URL")
	if gatewayURL == "" {
		gatewayURL = fmt.Sprintf("http://localhost:%d", gatewayPort)
	}

	// Check if gateway is running
	healthResp := curl(t, "GET", gatewayURL+"/health", nil, "")
	if healthResp.StatusCode != 200 {
		t.Skip("Gateway not running, skipping API tests")
	}

	authHeader := fmt.Sprintf("Bearer %s:%s:user", testTenantID, testUserID)

	t.Run("ReviewSQL", func(t *testing.T) {
		body := `{"database_id":"test-db","sql":"SELECT * FROM users WHERE id = 1"}`
		headers := map[string]string{
			"Authorization": authHeader,
			"Content-Type":  "application/json",
		}

		resp := curl(t, "POST", gatewayURL+"/api/v1/sql/review", headers, body)
		t.Logf("ReviewSQL response: status=%d body=%s", resp.StatusCode, resp.Body)

		if resp.StatusCode != 200 {
			t.Logf("ReviewSQL returned status %d, body: %s", resp.StatusCode, resp.Body)
		}
	})

	t.Run("ExecuteSQL", func(t *testing.T) {
		body := `{"database_id":"test-db","sql":"SELECT 1","timeout_seconds":30,"max_rows":100}`
		headers := map[string]string{
			"Authorization": authHeader,
			"Content-Type":  "application/json",
		}

		resp := curl(t, "POST", gatewayURL+"/api/v1/sql/execute", headers, body)
		t.Logf("ExecuteSQL response: status=%d body=%s", resp.StatusCode, resp.Body)

		if resp.StatusCode != 200 {
			t.Logf("ExecuteSQL returned status %d", resp.StatusCode)
		}
	})

	t.Run("GetSQLAudit", func(t *testing.T) {
		headers := map[string]string{
			"Authorization": authHeader,
		}

		resp := curl(t, "GET", gatewayURL+"/api/v1/sql/audit", headers, "")
		t.Logf("GetSQLAudit response: status=%d body=%s", resp.StatusCode, resp.Body)

		if resp.StatusCode != 200 {
			t.Logf("GetSQLAudit returned status %d", resp.StatusCode)
		}
	})

	t.Run("Unauthorized", func(t *testing.T) {
		body := `{"database_id":"test-db","sql":"SELECT 1"}`
		headers := map[string]string{
			"Content-Type": "application/json",
		}

		resp := curl(t, "POST", gatewayURL+"/api/v1/sql/review", headers, body)
		if resp.StatusCode != 401 {
			t.Errorf("Expected 401 Unauthorized, got status %d", resp.StatusCode)
		}
		t.Logf("Unauthorized test passed: status=%d", resp.StatusCode)
	})
}

// TestPerformanceAPI tests the Performance REST API through gateway
func TestPerformanceAPI(t *testing.T) {
	gatewayURL := os.Getenv("GATEWAY_URL")
	if gatewayURL == "" {
		gatewayURL = fmt.Sprintf("http://localhost:%d", gatewayPort)
	}

	// Check if gateway is running
	healthResp := curl(t, "GET", gatewayURL+"/health", nil, "")
	if healthResp.StatusCode != 200 {
		t.Skip("Gateway not running, skipping API tests")
	}

	authHeader := fmt.Sprintf("Bearer %s:%s:user", testTenantID, testUserID)
	now := time.Now()

	t.Run("Diagnose", func(t *testing.T) {
		body := fmt.Sprintf(`{"database_id":"test-db","scope":"full","start_time":%d,"end_time":%d,"deep_analysis":false}`,
			now.Add(-24*time.Hour).Unix(), now.Unix())
		headers := map[string]string{
			"Authorization": authHeader,
			"Content-Type":  "application/json",
		}

		resp := curl(t, "POST", gatewayURL+"/api/v1/performance/diagnose", headers, body)
		t.Logf("Diagnose response: status=%d body=%s", resp.StatusCode, resp.Body)
	})

	t.Run("GetMetrics", func(t *testing.T) {
		body := fmt.Sprintf(`{"database_id":"test-db","metric_names":["cpu_usage","memory"],"start_time":%d,"end_time":%d}`,
			now.Add(-1*time.Hour).Unix(), now.Unix())
		headers := map[string]string{
			"Authorization": authHeader,
			"Content-Type":  "application/json",
		}

		resp := curl(t, "POST", gatewayURL+"/api/v1/performance/metrics", headers, body)
		t.Logf("GetMetrics response: status=%d body=%s", resp.StatusCode, resp.Body)
	})

	t.Run("GetSlowQueries", func(t *testing.T) {
		body := fmt.Sprintf(`{"database_id":"test-db","start_time":%d,"end_time":%d,"min_duration_ms":100,"limit":10}`,
			now.Add(-24*time.Hour).Unix(), now.Unix())
		headers := map[string]string{
			"Authorization": authHeader,
			"Content-Type":  "application/json",
		}

		resp := curl(t, "POST", gatewayURL+"/api/v1/performance/slow-queries", headers, body)
		t.Logf("GetSlowQueries response: status=%d body=%s", resp.StatusCode, resp.Body)
	})
}

// TestThresholdAPI tests the Threshold REST API through gateway
func TestThresholdAPI(t *testing.T) {
	gatewayURL := os.Getenv("GATEWAY_URL")
	if gatewayURL == "" {
		gatewayURL = fmt.Sprintf("http://localhost:%d", gatewayPort)
	}

	// Check if gateway is running
	healthResp := curl(t, "GET", gatewayURL+"/health", nil, "")
	if healthResp.StatusCode != 200 {
		t.Skip("Gateway not running, skipping API tests")
	}

	authHeader := fmt.Sprintf("Bearer %s:%s:user", testTenantID, testUserID)

	t.Run("GetThresholds", func(t *testing.T) {
		body := `{"database_id":"test-db","metric_names":["cpu_usage","memory_usage"]}`
		headers := map[string]string{
			"Authorization": authHeader,
			"Content-Type":  "application/json",
		}

		resp := curl(t, "GET", gatewayURL+"/api/v1/thresholds", headers, body)
		t.Logf("GetThresholds response: status=%d body=%s", resp.StatusCode, resp.Body)
	})

	t.Run("UpdateThreshold", func(t *testing.T) {
		body := `{"threshold_id":"thresh-1","value":90.0,"type":"static"}`
		headers := map[string]string{
			"Authorization": authHeader,
			"Content-Type":  "application/json",
		}

		resp := curl(t, "PUT", gatewayURL+"/api/v1/thresholds", headers, body)
		t.Logf("UpdateThreshold response: status=%d body=%s", resp.StatusCode, resp.Body)
	})
}

// TestLLMAPI tests the LLM REST API through gateway
func TestLLMAPI(t *testing.T) {
	gatewayURL := os.Getenv("GATEWAY_URL")
	if gatewayURL == "" {
		gatewayURL = fmt.Sprintf("http://localhost:%d", gatewayPort)
	}

	// Check if gateway is running
	healthResp := curl(t, "GET", gatewayURL+"/health", nil, "")
	if healthResp.StatusCode != 200 {
		t.Skip("Gateway not running, skipping API tests")
	}

	authHeader := fmt.Sprintf("Bearer %s:%s:user", testTenantID, testUserID)

	t.Run("Chat", func(t *testing.T) {
		body := `{"session_id":"session-123","message":"How do I optimize this query?"}`
		headers := map[string]string{
			"Authorization": authHeader,
			"Content-Type":  "application/json",
		}

		resp := curl(t, "POST", gatewayURL+"/api/v1/llm/chat", headers, body)
		t.Logf("Chat response: status=%d body=%s", resp.StatusCode, resp.Body)
	})

	t.Run("GenerateSQL", func(t *testing.T) {
		body := `{"database_id":"test-db","natural_language":"Get all users created last month","schema_context":"users(id, name, created_at)"}`
		headers := map[string]string{
			"Authorization": authHeader,
			"Content-Type":  "application/json",
		}

		resp := curl(t, "POST", gatewayURL+"/api/v1/llm/generate-sql", headers, body)
		t.Logf("GenerateSQL response: status=%d body=%s", resp.StatusCode, resp.Body)
	})

	t.Run("GetRecommendations", func(t *testing.T) {
		body := `{"database_id":"test-db","category":"performance","limit":5}`
		headers := map[string]string{
			"Authorization": authHeader,
			"Content-Type":  "application/json",
		}

		resp := curl(t, "GET", gatewayURL+"/api/v1/llm/recommendations", headers, body)
		t.Logf("GetRecommendations response: status=%d body=%s", resp.StatusCode, resp.Body)
	})
}

// TestGraphQLProxy tests the GraphQL proxy through gateway
func TestGraphQLProxy(t *testing.T) {
	gatewayURL := os.Getenv("GATEWAY_URL")
	if gatewayURL == "" {
		gatewayURL = fmt.Sprintf("http://localhost:%d", gatewayPort)
	}

	// Check if gateway is running
	healthResp := curl(t, "GET", gatewayURL+"/health", nil, "")
	if healthResp.StatusCode != 200 {
		t.Skip("Gateway not running, skipping GraphQL tests")
	}

	t.Run("Introspection", func(t *testing.T) {
		query := `{"query":"{ __schema { types { name } } }"}`
		headers := map[string]string{
			"Content-Type": "application/json",
		}

		resp := curl(t, "POST", gatewayURL+"/graphql", headers, query)
		t.Logf("Introspection response: status=%d body=%s", resp.StatusCode, resp.Body)

		if resp.StatusCode == 200 {
			t.Log("GraphQL proxy is working")
		} else if resp.StatusCode == 503 {
			t.Log("Data Query Service not available (expected if not running)")
		}
	})

	t.Run("EndpointsQuery", func(t *testing.T) {
		query := `{"query":"{ endpoints }"}`
		headers := map[string]string{
			"Content-Type": "application/json",
		}

		resp := curl(t, "POST", gatewayURL+"/graphql", headers, query)
		t.Logf("Endpoints query response: status=%d body=%s", resp.StatusCode, resp.Body)
	})

	t.Run("MetricsQuery", func(t *testing.T) {
		query := `{"query":"{ metrics(endpoint: \"/api/metrics\") }"}`
		headers := map[string]string{
			"Content-Type": "application/json",
		}

		resp := curl(t, "POST", gatewayURL+"/graphql", headers, query)
		t.Logf("Metrics query response: status=%d body=%s", resp.StatusCode, resp.Body)
	})

	t.Run("SeriesQuery", func(t *testing.T) {
		query := `{
			"query": "query($tr: TimeRangeInput!) { series(endpoint: \"/api/metrics\", metric: \"cpu_usage\", timeRange: $tr, limit: 5) { meta { id metric } points { time value } } }",
			"variables": {
				"tr": {
					"start": time.Now().Add(-1*time.Hour).Format(time.RFC3339),
					"end": time.Now().Format(time.RFC3339)
				}
			}
		}`
		headers := map[string]string{
			"Content-Type": "application/json",
		}

		resp := curl(t, "POST", gatewayURL+"/graphql", headers, query)
		t.Logf("Series query response: status=%d body=%s", resp.StatusCode, resp.Body)
	})
}

// TestEndToEndFlow tests the complete flow: frontend -> gateway -> dataquery -> db
func TestEndToEndFlow(t *testing.T) {
	gatewayURL := os.Getenv("GATEWAY_URL")
	if gatewayURL == "" {
		gatewayURL = fmt.Sprintf("http://localhost:%d", gatewayPort)
	}

	// Check if gateway is running
	healthResp := curl(t, "GET", gatewayURL+"/health", nil, "")
	if healthResp.StatusCode != 200 {
		t.Skip("Gateway not running, skipping end-to-end tests")
	}

	authHeader := fmt.Sprintf("Bearer %s:%s:user", testTenantID, testUserID)

	t.Run("CompleteAPIFlow", func(t *testing.T) {
		// 1. Check health
		healthResp := curl(t, "GET", gatewayURL+"/health", nil, "")
		if healthResp.StatusCode != 200 {
			t.Fatalf("Health check failed: status %d", healthResp.StatusCode)
		}
		t.Log("Step 1: Health check passed")

		// 2. Review SQL
		reviewBody := `{"database_id":"test-db","sql":"SELECT * FROM users"}`
		reviewResp := curl(t, "POST", gatewayURL+"/api/v1/sql/review", map[string]string{
			"Authorization": authHeader,
			"Content-Type":  "application/json",
		}, reviewBody)
		t.Logf("Step 2: SQL Review status=%d", reviewResp.StatusCode)

		// 3. Execute SQL (dry run)
		execBody := `{"database_id":"test-db","sql":"SELECT 1","dry_run":true}`
		execResp := curl(t, "POST", gatewayURL+"/api/v1/sql/execute", map[string]string{
			"Authorization": authHeader,
			"Content-Type":  "application/json",
		}, execBody)
		t.Logf("Step 3: SQL Execute status=%d", execResp.StatusCode)

		// 4. Check audit trail
		auditResp := curl(t, "GET", gatewayURL+"/api/v1/sql/audit", map[string]string{
			"Authorization": authHeader,
		}, "")
		t.Logf("Step 4: SQL Audit status=%d", auditResp.StatusCode)

		// 5. Query GraphQL
		graphqlQuery := `{"query":"{ endpoints }"}`
		graphqlResp := curl(t, "POST", gatewayURL+"/graphql", map[string]string{
			"Content-Type": "application/json",
		}, graphqlQuery)
		t.Logf("Step 5: GraphQL status=%d", graphqlResp.StatusCode)

		t.Log("Complete flow test finished")
	})
}

// TestCORSPolicy tests CORS headers
func TestCORSPolicy(t *testing.T) {
	gatewayURL := os.Getenv("GATEWAY_URL")
	if gatewayURL == "" {
		gatewayURL = fmt.Sprintf("http://localhost:%d", gatewayPort)
	}

	// Check if gateway is running
	healthResp := curl(t, "GET", gatewayURL+"/health", nil, "")
	if healthResp.StatusCode != 200 {
		t.Skip("Gateway not running, skipping CORS tests")
	}

	t.Run("PreflightRequest", func(t *testing.T) {
		// Use http client for more control
		req, err := http.NewRequest("OPTIONS", gatewayURL+"/api/v1/sql/review", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		req.Header.Set("Origin", "http://localhost:3000")
		req.Header.Set("Access-Control-Request-Method", "POST")

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		allowOrigin := resp.Header.Get("Access-Control-Allow-Origin")
		allowMethods := resp.Header.Get("Access-Control-Allow-Methods")

		t.Logf("CORS headers: Origin=%s, Methods=%s", allowOrigin, allowMethods)

		if resp.StatusCode != 204 {
			t.Logf("Preflight status=%d (expected 204)", resp.StatusCode)
		}
	})

	t.Run("ActualRequest", func(t *testing.T) {
		req, err := http.NewRequest("GET", gatewayURL+"/health", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		req.Header.Set("Origin", "http://localhost:3000")

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		allowOrigin := resp.Header.Get("Access-Control-Allow-Origin")
		t.Logf("CORS Origin header: %s", allowOrigin)
	})
}

// TestAuthentication tests authentication middleware
func TestAuthentication(t *testing.T) {
	gatewayURL := os.Getenv("GATEWAY_URL")
	if gatewayURL == "" {
		gatewayURL = fmt.Sprintf("http://localhost:%d", gatewayPort)
	}

	// Check if gateway is running
	healthResp := curl(t, "GET", gatewayURL+"/health", nil, "")
	if healthResp.StatusCode != 200 {
		t.Skip("Gateway not running, skipping auth tests")
	}

	t.Run("MissingAuth", func(t *testing.T) {
		body := `{"database_id":"test-db","sql":"SELECT 1"}`
		resp := curl(t, "POST", gatewayURL+"/api/v1/sql/review", map[string]string{
			"Content-Type": "application/json",
		}, body)

		if resp.StatusCode != 401 {
			t.Errorf("Expected 401, got status %d", resp.StatusCode)
		}
		t.Logf("Missing auth: status=%d (expected 401)", resp.StatusCode)
	})

	t.Run("InvalidAuthFormat", func(t *testing.T) {
		body := `{"database_id":"test-db","sql":"SELECT 1"}`
		resp := curl(t, "POST", gatewayURL+"/api/v1/sql/review", map[string]string{
			"Authorization": "InvalidToken",
			"Content-Type":  "application/json",
		}, body)

		if resp.StatusCode != 401 {
			t.Errorf("Expected 401, got status %d", resp.StatusCode)
		}
		t.Logf("Invalid auth format: status=%d (expected 401)", resp.StatusCode)
	})

	t.Run("ValidAuth", func(t *testing.T) {
		authHeader := fmt.Sprintf("Bearer %s:%s:user", testTenantID, testUserID)
		body := `{"database_id":"test-db","sql":"SELECT 1"}`
		resp := curl(t, "POST", gatewayURL+"/api/v1/sql/review", map[string]string{
			"Authorization": authHeader,
			"Content-Type":  "application/json",
		}, body)

		if resp.StatusCode == 401 {
			t.Error("Valid auth should not return 401")
		}
		t.Logf("Valid auth: status=%d", resp.StatusCode)
	})
}

// TestRequestIDPropagation tests that request ID is properly propagated
func TestRequestIDPropagation(t *testing.T) {
	gatewayURL := os.Getenv("GATEWAY_URL")
	if gatewayURL == "" {
		gatewayURL = fmt.Sprintf("http://localhost:%d", gatewayPort)
	}

	// Check if gateway is running
	healthResp := curl(t, "GET", gatewayURL+"/health", nil, "")
	if healthResp.StatusCode != 200 {
		t.Skip("Gateway not running, skipping request ID tests")
	}

	t.Run("CustomRequestID", func(t *testing.T) {
		customReqID := "test-request-12345"

		req, err := http.NewRequest("GET", gatewayURL+"/health", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		req.Header.Set("X-Request-ID", customReqID)

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		// Read and discard body
		io.Copy(io.Discard, resp.Body)

		respReqID := resp.Header.Get("X-Request-ID")
		t.Logf("Request ID: sent=%s, received=%s", customReqID, respReqID)

		if respReqID != customReqID {
			t.Logf("Warning: Request ID not preserved (sent=%s, got=%s)", customReqID, respReqID)
		}
	})

	t.Run("GeneratedRequestID", func(t *testing.T) {
		req, err := http.NewRequest("GET", gatewayURL+"/health", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		// Read and discard body
		io.Copy(io.Discard, resp.Body)

		respReqID := resp.Header.Get("X-Request-ID")
		t.Logf("Generated Request ID: %s", respReqID)

		if respReqID == "" {
			t.Error("Request ID should be generated if not provided")
		}
	})
}

// TestErrorResponse tests error handling
func TestErrorResponse(t *testing.T) {
	gatewayURL := os.Getenv("GATEWAY_URL")
	if gatewayURL == "" {
		gatewayURL = fmt.Sprintf("http://localhost:%d", gatewayPort)
	}

	// Check if gateway is running
	healthResp := curl(t, "GET", gatewayURL+"/health", nil, "")
	if healthResp.StatusCode != 200 {
		t.Skip("Gateway not running, skipping error tests")
	}

	authHeader := fmt.Sprintf("Bearer %s:%s:user", testTenantID, testUserID)

	t.Run("InvalidJSON", func(t *testing.T) {
		resp := curl(t, "POST", gatewayURL+"/api/v1/sql/review", map[string]string{
			"Authorization": authHeader,
			"Content-Type":  "application/json",
		}, "invalid json")

		if resp.StatusCode != 400 {
			t.Errorf("Expected 400 for invalid JSON, got status %d", resp.StatusCode)
		}
		t.Logf("Invalid JSON: status=%d body=%s", resp.StatusCode, resp.Body)
	})

	t.Run("NotFound", func(t *testing.T) {
		resp := curl(t, "GET", gatewayURL+"/api/v1/nonexistent", nil, "")

		if resp.StatusCode != 404 {
			t.Errorf("Expected 404, got status %d", resp.StatusCode)
		}
		t.Logf("Not found: status=%d", resp.StatusCode)
	})
}

// printTestInfo prints test information
func TestMain(m *testing.M) {
	fmt.Println("\n========================================")
	fmt.Println("Gateway Integration Tests")
	fmt.Println("========================================")
	fmt.Println("\nPrerequisites:")
	fmt.Println("  1. Gateway running on port 8080 (or set GATEWAY_URL)")
	fmt.Println("  2. Data Query Service running on port 8084")
	fmt.Println("  3. Database (TimescaleDB) running (optional)")
	fmt.Println("\nTo start services:")
	fmt.Println("  go run cmd/gateway/main.go &")
	fmt.Println("  go run cmd/dataquery/main.go &")
	fmt.Println("\n========================================")

	os.Exit(m.Run())
}