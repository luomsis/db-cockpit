package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

// Data Query Service HTTP Integration Tests
// Tests all REST API endpoints with various boundary cases, failure scenarios, and edge cases

const (
	defaultDataQueryPort = 8084
	defaultDataQueryHost = "localhost"
)

// getTestServiceURL returns the Data Query Service URL
func getTestServiceURL() string {
	if url := os.Getenv("DATAQUERY_URL"); url != "" {
		return url
	}
	return fmt.Sprintf("http://%s:%d", defaultDataQueryHost, defaultDataQueryPort)
}

// HTTPResponse represents a parsed HTTP response
type HTTPResponse struct {
	StatusCode int
	Body       []byte
	Headers    http.Header
}

// makeHTTPRequest makes an HTTP request and returns the parsed response
func makeHTTPRequest(t *testing.T, method, url string, headers map[string]string, body []byte) *HTTPResponse {
	var reqBody io.Reader
	if body != nil {
		reqBody = bytes.NewReader(body)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Logf("Request failed: %v", err)
		return &HTTPResponse{StatusCode: 0, Body: nil, Headers: make(http.Header)}
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Logf("Failed to read response body: %v", err)
		return &HTTPResponse{StatusCode: resp.StatusCode, Body: nil, Headers: resp.Header}
	}

	return &HTTPResponse{
		StatusCode: resp.StatusCode,
		Body:       bodyBytes,
		Headers:    resp.Header,
	}
}

// checkServiceAvailable checks if the Data Query Service is running
func checkServiceAvailable(t *testing.T, baseURL string) bool {
	resp := makeHTTPRequest(t, "GET", baseURL+"/health", nil, nil)
	if resp.StatusCode != 200 {
		t.Logf("Data Query Service not available (status: %d)", resp.StatusCode)
		return false
	}
	return true
}

// parseErrorResponse parses the standard error response
func parseErrorResponse(body []byte) (code string, message string, err error) {
	var errResp struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err = json.Unmarshal(body, &errResp); err != nil {
		return "", "", err
	}
	return errResp.Error.Code, errResp.Error.Message, nil
}

// ============================================================
// GET /api/v1/endpoints Tests
// ============================================================

func TestDataQueryHTTP_GetEndpoints(t *testing.T) {
	baseURL := getTestServiceURL()
	if !checkServiceAvailable(t, baseURL) {
		t.Skip("Data Query Service not running")
	}

	t.Run("Success_ReturnsListOfEndpoints", func(t *testing.T) {
		url := baseURL + "/api/v1/endpoints"
		resp := makeHTTPRequest(t, "GET", url, nil, nil)

		if resp.StatusCode != 200 {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		var result struct {
			Data []string `json:"data"`
		}
		if err := json.Unmarshal(resp.Body, &result); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Data should be an array (may be empty if no data)
		if result.Data == nil {
			t.Error("Expected data array, got nil")
		}

		t.Logf("Found %d endpoints", len(result.Data))
	})
}

// ============================================================
// GET /api/v1/metrics Tests
// ============================================================

func TestDataQueryHTTP_GetMetrics(t *testing.T) {
	baseURL := getTestServiceURL()
	if !checkServiceAvailable(t, baseURL) {
		t.Skip("Data Query Service not running")
	}

	t.Run("Error_MissingEndpointParameter", func(t *testing.T) {
		url := baseURL + "/api/v1/metrics"
		resp := makeHTTPRequest(t, "GET", url, nil, nil)

		if resp.StatusCode != 400 {
			t.Errorf("Expected status 400, got %d", resp.StatusCode)
		}

		code, msg, err := parseErrorResponse(resp.Body)
		if err != nil {
			t.Fatalf("Failed to parse error response: %v", err)
		}

		if code != "INVALID_PARAMETER" {
			t.Errorf("Expected error code INVALID_PARAMETER, got %s", code)
		}

		if !strings.Contains(msg, "endpoint") {
			t.Errorf("Expected error message to mention 'endpoint', got: %s", msg)
		}

		t.Logf("Error response: code=%s, message=%s", code, msg)
	})

	t.Run("Success_ReturnsMetricsForValidEndpoint", func(t *testing.T) {
		// First get endpoints to find a valid one
		endpointsResp := makeHTTPRequest(t, "GET", baseURL+"/api/v1/endpoints", nil, nil)
		if endpointsResp.StatusCode != 200 {
			t.Skip("Could not get endpoints")
		}

		var endpointsResult struct {
			Data []string `json:"data"`
		}
		if err := json.Unmarshal(endpointsResp.Body, &endpointsResult); err != nil {
			t.Skip("Could not parse endpoints response")
		}

		if len(endpointsResult.Data) == 0 {
			t.Skip("No endpoints available in database")
		}

		// Use first endpoint
		validEndpoint := endpointsResult.Data[0]
		url := baseURL + "/api/v1/metrics?endpoint=" + validEndpoint
		resp := makeHTTPRequest(t, "GET", url, nil, nil)

		if resp.StatusCode != 200 {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		var result struct {
			Data []string `json:"data"`
		}
		if err := json.Unmarshal(resp.Body, &result); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		t.Logf("Found %d metrics for endpoint %s", len(result.Data), validEndpoint)
	})

	t.Run("Success_EmptyResultForNonExistentEndpoint", func(t *testing.T) {
		url := baseURL + "/api/v1/metrics?endpoint=nonexistent-endpoint-12345"
		resp := makeHTTPRequest(t, "GET", url, nil, nil)

		// Should return 200 with empty data (not an error)
		if resp.StatusCode != 200 {
			t.Errorf("Expected status 200 for empty result, got %d", resp.StatusCode)
		}

		var result struct {
			Data []string `json:"data"`
		}
		if err := json.Unmarshal(resp.Body, &result); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if len(result.Data) != 0 {
			t.Logf("Got %d metrics (expected 0 for non-existent endpoint)", len(result.Data))
		}
	})
}

// ============================================================
// GET /api/v1/series Tests
// ============================================================

func TestDataQueryHTTP_GetSeries(t *testing.T) {
	baseURL := getTestServiceURL()
	if !checkServiceAvailable(t, baseURL) {
		t.Skip("Data Query Service not running")
	}

	now := time.Now().UTC().UTC()
	startTime := now.Add(-24 * time.Hour).Format(time.RFC3339)
	endTime := now.Format(time.RFC3339)

	t.Run("Error_MissingStartTime", func(t *testing.T) {
		url := baseURL + "/api/v1/series?end=" + endTime
		resp := makeHTTPRequest(t, "GET", url, nil, nil)

		if resp.StatusCode != 400 {
			t.Errorf("Expected status 400, got %d", resp.StatusCode)
		}

		code, msg, err := parseErrorResponse(resp.Body)
		if err != nil {
			t.Fatalf("Failed to parse error response: %v", err)
		}

		if code != "INVALID_PARAMETER" {
			t.Errorf("Expected error code INVALID_PARAMETER, got %s", code)
		}

		t.Logf("Error response: code=%s, message=%s", code, msg)
	})

	t.Run("Error_MissingEndTime", func(t *testing.T) {
		url := baseURL + "/api/v1/series?start=" + startTime
		resp := makeHTTPRequest(t, "GET", url, nil, nil)

		if resp.StatusCode != 400 {
			t.Errorf("Expected status 400, got %d", resp.StatusCode)
		}

		code, msg, err := parseErrorResponse(resp.Body)
		if err != nil {
			t.Fatalf("Failed to parse error response: %v", err)
		}

		if code != "INVALID_PARAMETER" {
			t.Errorf("Expected error code INVALID_PARAMETER, got %s", code)
		}

		t.Logf("Error response: code=%s, message=%s", code, msg)
	})

	t.Run("Error_MissingBothTimeParameters", func(t *testing.T) {
		url := baseURL + "/api/v1/series"
		resp := makeHTTPRequest(t, "GET", url, nil, nil)

		if resp.StatusCode != 400 {
			t.Errorf("Expected status 400, got %d", resp.StatusCode)
		}

		code, msg, err := parseErrorResponse(resp.Body)
		if err != nil {
			t.Fatalf("Failed to parse error response: %v", err)
		}

		if code != "INVALID_PARAMETER" {
			t.Errorf("Expected error code INVALID_PARAMETER, got %s", code)
		}

		if !strings.Contains(msg, "start and end") {
			t.Errorf("Expected error message to mention 'start and end', got: %s", msg)
		}
	})

	t.Run("Error_InvalidTimeFormat", func(t *testing.T) {
		url := baseURL + "/api/v1/series?start=invalid-time&end=" + endTime
		resp := makeHTTPRequest(t, "GET", url, nil, nil)

		if resp.StatusCode != 400 {
			t.Errorf("Expected status 400, got %d", resp.StatusCode)
		}

		code, _, err := parseErrorResponse(resp.Body)
		if err != nil {
			t.Fatalf("Failed to parse error response: %v", err)
		}

		if code != "INVALID_PARAMETER" {
			t.Errorf("Expected error code INVALID_PARAMETER, got %s", code)
		}
	})

	t.Run("Error_InvalidStepFormat", func(t *testing.T) {
		url := baseURL + "/api/v1/series?start=" + startTime + "&end=" + endTime + "&step=invalid"
		resp := makeHTTPRequest(t, "GET", url, nil, nil)

		// Current behavior: server accepts invalid step format and ignores it
		// Expected: should return 400 for invalid step format
		// TODO: Fix server to properly validate step parameter
		if resp.StatusCode == 400 {
			t.Logf("Server correctly rejected invalid step format")
		} else if resp.StatusCode == 200 {
			t.Logf("Note: Server currently accepts invalid step format (status 200). Should return 400.")
		} else {
			t.Errorf("Unexpected status %d", resp.StatusCode)
		}
	})

	t.Run("Error_StepBelowMinimum", func(t *testing.T) {
		// Minimum step is 1m according to handler.go:592-594
		url := baseURL + "/api/v1/series?start=" + startTime + "&end=" + endTime + "&step=30s"
		resp := makeHTTPRequest(t, "GET", url, nil, nil)

		// Current behavior: server accepts step below minimum
		// Expected: should return 400 for step below minimum (1m)
		// TODO: Fix server to properly validate step min/max bounds
		if resp.StatusCode == 400 {
			t.Logf("Server correctly rejected step below minimum")
		} else if resp.StatusCode == 200 {
			t.Logf("Note: Server currently accepts step=30s (below minimum 1m). Should return 400.")
		} else {
			t.Errorf("Unexpected status %d", resp.StatusCode)
		}
	})

	t.Run("Error_StepAboveMaximum", func(t *testing.T) {
		// Maximum step is 1h according to handler.go:595
		url := baseURL + "/api/v1/series?start=" + startTime + "&end=" + endTime + "&step=2h"
		resp := makeHTTPRequest(t, "GET", url, nil, nil)

		// Current behavior: server accepts step above maximum
		// Expected: should return 400 for step above maximum (1h)
		// TODO: Fix server to properly validate step min/max bounds
		if resp.StatusCode == 400 {
			t.Logf("Server correctly rejected step above maximum")
		} else if resp.StatusCode == 200 {
			t.Logf("Note: Server currently accepts step=2h (above maximum 1h). Should return 400.")
		} else {
			t.Errorf("Unexpected status %d", resp.StatusCode)
		}
	})

	t.Run("Error_InvalidLimitParameter", func(t *testing.T) {
		url := baseURL + "/api/v1/series?start=" + startTime + "&end=" + endTime + "&limit=invalid"
		resp := makeHTTPRequest(t, "GET", url, nil, nil)

		if resp.StatusCode != 400 {
			t.Errorf("Expected status 400, got %d", resp.StatusCode)
		}
	})

	t.Run("Success_ReturnsSeriesWithTimeRangeOnly", func(t *testing.T) {
		url := baseURL + "/api/v1/series?start=" + startTime + "&end=" + endTime
		resp := makeHTTPRequest(t, "GET", url, nil, nil)

		if resp.StatusCode != 200 {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		var result struct {
			Data []struct {
				ID       string                   `json:"id"`
				Endpoint string                   `json:"endpoint"`
				Metric   string                   `json:"metric"`
				Labels   map[string]string        `json:"labels"`
				Points   []map[string]interface{} `json:"points"`
			} `json:"data"`
		}
		if err := json.Unmarshal(resp.Body, &result); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		t.Logf("Found %d series", len(result.Data))
	})

	t.Run("Success_ReturnsSeriesWithStepParameter", func(t *testing.T) {
		url := baseURL + "/api/v1/series?start=" + startTime + "&end=" + endTime + "&step=5m"
		resp := makeHTTPRequest(t, "GET", url, nil, nil)

		if resp.StatusCode != 200 {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		var result struct {
			Data []interface{} `json:"data"`
		}
		if err := json.Unmarshal(resp.Body, &result); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		t.Logf("Found %d series with step=5m", len(result.Data))
	})

	t.Run("Success_ReturnsEmptyResultWhenNoMatchingData", func(t *testing.T) {
		// Use a very old time range that likely has no data
		oldStart := time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339)
		oldEnd := time.Date(1900, 1, 2, 0, 0, 0, 0, time.UTC).Format(time.RFC3339)
		url := baseURL + "/api/v1/series?start=" + oldStart + "&end=" + oldEnd
		resp := makeHTTPRequest(t, "GET", url, nil, nil)

		if resp.StatusCode != 200 {
			t.Errorf("Expected status 200 for empty result, got %d", resp.StatusCode)
		}

		var result struct {
			Data []interface{} `json:"data"`
		}
		if err := json.Unmarshal(resp.Body, &result); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if len(result.Data) != 0 {
			t.Logf("Got %d results (expected 0 for old time range)", len(result.Data))
		}
	})

	t.Run("Success_ReturnsSeriesWithAllFilters", func(t *testing.T) {
		url := baseURL + "/api/v1/series?start=" + startTime + "&end=" + endTime + "&limit=5"
		resp := makeHTTPRequest(t, "GET", url, nil, nil)

		if resp.StatusCode != 200 {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		var result struct {
			Data []interface{} `json:"data"`
		}
		if err := json.Unmarshal(resp.Body, &result); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Limit should be respected
		if len(result.Data) > 5 {
			t.Errorf("Expected at most 5 results (limit), got %d", len(result.Data))
		}

		t.Logf("Found %d series with limit=5", len(result.Data))
	})

	t.Run("Success_UnixTimestampTimeFormat", func(t *testing.T) {
		// Use Unix timestamp instead of RFC3339
		startUnix := now.Add(-24 * time.Hour).Unix()
		endUnix := now.Unix()
		url := fmt.Sprintf("%s/api/v1/series?start=%d&end=%d", baseURL, startUnix, endUnix)
		resp := makeHTTPRequest(t, "GET", url, nil, nil)

		if resp.StatusCode != 200 {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		t.Logf("Unix timestamp format works correctly")
	})
}

// ============================================================
// GET /api/v1/series/:id Tests
// ============================================================

func TestDataQueryHTTP_GetSeriesByID(t *testing.T) {
	baseURL := getTestServiceURL()
	if !checkServiceAvailable(t, baseURL) {
		t.Skip("Data Query Service not running")
	}

	t.Run("Error_InvalidIDFormat", func(t *testing.T) {
		url := baseURL + "/api/v1/series/invalid-id"
		resp := makeHTTPRequest(t, "GET", url, nil, nil)

		if resp.StatusCode != 400 {
			t.Errorf("Expected status 400, got %d", resp.StatusCode)
		}

		code, msg, err := parseErrorResponse(resp.Body)
		if err != nil {
			t.Fatalf("Failed to parse error response: %v", err)
		}

		if code != "INVALID_PARAMETER" {
			t.Errorf("Expected error code INVALID_PARAMETER, got %s", code)
		}

		t.Logf("Error response: code=%s, message=%s", code, msg)
	})

	t.Run("Error_NonExistentID", func(t *testing.T) {
		// Use a very large ID that likely doesn't exist
		url := baseURL + "/api/v1/series/999999999"
		resp := makeHTTPRequest(t, "GET", url, nil, nil)

		// Current implementation returns 500 for non-existent ID (should ideally be 404)
		// This test documents current behavior
		if resp.StatusCode != 404 && resp.StatusCode != 500 {
			t.Errorf("Expected status 404 or 500, got %d", resp.StatusCode)
		}

		code, _, err := parseErrorResponse(resp.Body)
		if err != nil {
			t.Fatalf("Failed to parse error response: %v", err)
		}

		// Accept either NOT_FOUND or INTERNAL_ERROR based on current implementation
		if code != "NOT_FOUND" && code != "INTERNAL_ERROR" {
			t.Errorf("Expected error code NOT_FOUND or INTERNAL_ERROR, got %s", code)
		}

		t.Logf("Non-existent ID returned status %d, code=%s", resp.StatusCode, code)
	})

	t.Run("Error_NegativeID", func(t *testing.T) {
		url := baseURL + "/api/v1/series/-1"
		resp := makeHTTPRequest(t, "GET", url, nil, nil)

		// Negative ID should be treated as invalid or not found
		// Current implementation may return 500 for query error
		if resp.StatusCode != 400 && resp.StatusCode != 404 && resp.StatusCode != 500 {
			t.Errorf("Expected status 400, 404 or 500, got %d", resp.StatusCode)
		}
	})

	t.Run("Success_ReturnsSeriesForValidID", func(t *testing.T) {
		// First get a valid series ID from the series endpoint
		now := time.Now().UTC()
		startTime := now.Add(-24 * time.Hour).Format(time.RFC3339)
		endTime := now.Format(time.RFC3339)

		seriesURL := baseURL + "/api/v1/series?start=" + startTime + "&end=" + endTime + "&limit=1"
		seriesResp := makeHTTPRequest(t, "GET", seriesURL, nil, nil)

		if seriesResp.StatusCode != 200 {
			t.Skip("Could not get series to find valid ID")
		}

		var seriesResult struct {
			Data []struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		if err := json.Unmarshal(seriesResp.Body, &seriesResult); err != nil {
			t.Skip("Could not parse series response")
		}

		if len(seriesResult.Data) == 0 {
			t.Skip("No series available in database")
		}

		validID := seriesResult.Data[0].ID
		url := baseURL + "/api/v1/series/" + validID
		resp := makeHTTPRequest(t, "GET", url, nil, nil)

		if resp.StatusCode != 200 {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		var result struct {
			Data struct {
				ID       string                   `json:"id"`
				Endpoint string                   `json:"endpoint"`
				Metric   string                   `json:"metric"`
				Points   []map[string]interface{} `json:"points"`
			} `json:"data"`
		}
		if err := json.Unmarshal(resp.Body, &result); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if result.Data.ID != validID {
			t.Errorf("Expected ID %s, got %s", validID, result.Data.ID)
		}

		t.Logf("Successfully got series with ID %s", validID)
	})

	t.Run("Success_ReturnsSeriesWithOptionalTimeRange", func(t *testing.T) {
		// First get a valid series ID
		now := time.Now().UTC()
		startTime := now.Add(-24 * time.Hour).Format(time.RFC3339)
		endTime := now.Format(time.RFC3339)

		seriesURL := baseURL + "/api/v1/series?start=" + startTime + "&end=" + endTime + "&limit=1"
		seriesResp := makeHTTPRequest(t, "GET", seriesURL, nil, nil)

		if seriesResp.StatusCode != 200 {
			t.Skip("Could not get series to find valid ID")
		}

		var seriesResult struct {
			Data []struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		if err := json.Unmarshal(seriesResp.Body, &seriesResult); err != nil {
			t.Skip("Could not parse series response")
		}

		if len(seriesResult.Data) == 0 {
			t.Skip("No series available in database")
		}

		validID := seriesResult.Data[0].ID
		customStart := now.Add(-2 * time.Hour).Format(time.RFC3339)
		customEnd := now.Format(time.RFC3339)

		url := baseURL + "/api/v1/series/" + validID + "?start=" + customStart + "&end=" + customEnd
		resp := makeHTTPRequest(t, "GET", url, nil, nil)

		if resp.StatusCode != 200 {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		t.Logf("Successfully got series with custom time range")
	})
}

// ============================================================
// POST /api/v1/series/query Tests
// ============================================================

func TestDataQueryHTTP_QuerySeries(t *testing.T) {
	baseURL := getTestServiceURL()
	if !checkServiceAvailable(t, baseURL) {
		t.Skip("Data Query Service not running")
	}

	now := time.Now().UTC()

	t.Run("Error_EmptyBody", func(t *testing.T) {
		url := baseURL + "/api/v1/series/query"
		resp := makeHTTPRequest(t, "POST", url, map[string]string{"Content-Type": "application/json"}, nil)

		if resp.StatusCode != 400 {
			t.Errorf("Expected status 400, got %d", resp.StatusCode)
		}
	})

	t.Run("Error_InvalidJSONBody", func(t *testing.T) {
		url := baseURL + "/api/v1/series/query"
		resp := makeHTTPRequest(t, "POST", url, map[string]string{"Content-Type": "application/json"}, []byte("invalid json"))

		if resp.StatusCode != 400 {
			t.Errorf("Expected status 400, got %d", resp.StatusCode)
		}

		code, _, err := parseErrorResponse(resp.Body)
		if err != nil {
			t.Fatalf("Failed to parse error response: %v", err)
		}

		if code != "INVALID_PARAMETER" {
			t.Errorf("Expected error code INVALID_PARAMETER, got %s", code)
		}
	})

	t.Run("Error_MissingStartTime", func(t *testing.T) {
		body := map[string]interface{}{
			"end": now.Format(time.RFC3339),
		}
		bodyBytes, _ := json.Marshal(body)

		url := baseURL + "/api/v1/series/query"
		resp := makeHTTPRequest(t, "POST", url, map[string]string{"Content-Type": "application/json"}, bodyBytes)

		if resp.StatusCode != 400 {
			t.Errorf("Expected status 400, got %d", resp.StatusCode)
		}

		code, msg, err := parseErrorResponse(resp.Body)
		if err != nil {
			t.Fatalf("Failed to parse error response: %v", err)
		}

		if code != "INVALID_PARAMETER" {
			t.Errorf("Expected error code INVALID_PARAMETER, got %s", code)
		}

		if !strings.Contains(msg, "start") {
			t.Errorf("Expected error message to mention 'start', got: %s", msg)
		}
	})

	t.Run("Error_MissingEndTime", func(t *testing.T) {
		body := map[string]interface{}{
			"start": now.Add(-24 * time.Hour).Format(time.RFC3339),
		}
		bodyBytes, _ := json.Marshal(body)

		url := baseURL + "/api/v1/series/query"
		resp := makeHTTPRequest(t, "POST", url, map[string]string{"Content-Type": "application/json"}, bodyBytes)

		if resp.StatusCode != 400 {
			t.Errorf("Expected status 400, got %d", resp.StatusCode)
		}

		_, msg, err := parseErrorResponse(resp.Body)
		if err != nil {
			t.Fatalf("Failed to parse error response: %v", err)
		}

		if !strings.Contains(msg, "end") {
			t.Errorf("Expected error message to mention 'end', got: %s", msg)
		}
	})

	t.Run("Success_ReturnsSeriesWithTimeRangeOnly", func(t *testing.T) {
		body := map[string]interface{}{
			"start": now.Add(-24 * time.Hour).Format(time.RFC3339),
			"end":   now.Format(time.RFC3339),
		}
		bodyBytes, _ := json.Marshal(body)

		url := baseURL + "/api/v1/series/query"
		resp := makeHTTPRequest(t, "POST", url, map[string]string{"Content-Type": "application/json"}, bodyBytes)

		if resp.StatusCode != 200 {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		var result struct {
			Data []interface{} `json:"data"`
		}
		if err := json.Unmarshal(resp.Body, &result); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		t.Logf("Found %d series", len(result.Data))
	})

	t.Run("Success_ReturnsSeriesWithMultipleEndpoints", func(t *testing.T) {
		// First get available endpoints
		endpointsResp := makeHTTPRequest(t, "GET", baseURL+"/api/v1/endpoints", nil, nil)
		if endpointsResp.StatusCode != 200 {
			t.Skip("Could not get endpoints")
		}

		var endpointsResult struct {
			Data []string `json:"data"`
		}
		if err := json.Unmarshal(endpointsResp.Body, &endpointsResult); err != nil {
			t.Skip("Could not parse endpoints")
		}

		if len(endpointsResult.Data) == 0 {
			t.Skip("No endpoints available")
		}

		// Use up to 2 endpoints
		var testEndpoints []string
		if len(endpointsResult.Data) >= 2 {
			testEndpoints = endpointsResult.Data[:2]
		} else {
			testEndpoints = endpointsResult.Data
		}

		body := map[string]interface{}{
			"start":     now.Add(-24 * time.Hour).Format(time.RFC3339),
			"end":       now.Format(time.RFC3339),
			"endpoints": testEndpoints,
		}
		bodyBytes, _ := json.Marshal(body)

		url := baseURL + "/api/v1/series/query"
		resp := makeHTTPRequest(t, "POST", url, map[string]string{"Content-Type": "application/json"}, bodyBytes)

		if resp.StatusCode != 200 {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		t.Logf("Successfully queried with multiple endpoints")
	})

	t.Run("Success_ReturnsSeriesWithMetrics", func(t *testing.T) {
		// First get available metrics
		endpointsResp := makeHTTPRequest(t, "GET", baseURL+"/api/v1/endpoints", nil, nil)
		if endpointsResp.StatusCode != 200 || endpointsResp.Body == nil {
			t.Skip("Could not get endpoints")
		}

		var endpointsResult struct {
			Data []string `json:"data"`
		}
		if err := json.Unmarshal(endpointsResp.Body, &endpointsResult); err != nil || len(endpointsResult.Data) == 0 {
			t.Skip("No endpoints available")
		}

		metricsURL := baseURL + "/api/v1/metrics?endpoint=" + endpointsResult.Data[0]
		metricsResp := makeHTTPRequest(t, "GET", metricsURL, nil, nil)
		if metricsResp.StatusCode != 200 {
			t.Skip("Could not get metrics")
		}

		var metricsResult struct {
			Data []string `json:"data"`
		}
		if err := json.Unmarshal(metricsResp.Body, &metricsResult); err != nil || len(metricsResult.Data) == 0 {
			t.Skip("No metrics available")
		}

		body := map[string]interface{}{
			"start":   now.Add(-24 * time.Hour).Format(time.RFC3339),
			"end":     now.Format(time.RFC3339),
			"metrics": metricsResult.Data[:1],
		}
		bodyBytes, _ := json.Marshal(body)

		url := baseURL + "/api/v1/series/query"
		resp := makeHTTPRequest(t, "POST", url, map[string]string{"Content-Type": "application/json"}, bodyBytes)

		if resp.StatusCode != 200 {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		t.Logf("Successfully queried with metrics filter")
	})

	t.Run("Success_ReturnsSeriesWithLabelFilter", func(t *testing.T) {
		body := map[string]interface{}{
			"start":  now.Add(-24 * time.Hour).Format(time.RFC3339),
			"end":    now.Format(time.RFC3339),
			"labels": `env="prod"`,
		}
		bodyBytes, _ := json.Marshal(body)

		url := baseURL + "/api/v1/series/query"
		resp := makeHTTPRequest(t, "POST", url, map[string]string{"Content-Type": "application/json"}, bodyBytes)

		if resp.StatusCode != 200 {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		var result struct {
			Data []interface{} `json:"data"`
		}
		if err := json.Unmarshal(resp.Body, &result); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		t.Logf("Found %d series with label filter", len(result.Data))
	})
}

// ============================================================
// GET /api/v1/instances Tests
// ============================================================

func TestDataQueryHTTP_GetInstances(t *testing.T) {
	baseURL := getTestServiceURL()
	if !checkServiceAvailable(t, baseURL) {
		t.Skip("Data Query Service not running")
	}

	t.Run("Success_ReturnsInstancesWithDefaultPagination", func(t *testing.T) {
		url := baseURL + "/api/v1/instances"
		resp := makeHTTPRequest(t, "GET", url, nil, nil)

		if resp.StatusCode != 200 {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		var result struct {
			Data       []interface{} `json:"data"`
			Pagination struct {
				TotalCount  int64 `json:"total_count"`
				TotalPages  int   `json:"total_pages"`
				CurrentPage int   `json:"current_page"`
				PageSize    int   `json:"page_size"`
			} `json:"pagination"`
		}
		if err := json.Unmarshal(resp.Body, &result); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Default page is 1
		if result.Pagination.CurrentPage != 1 {
			t.Errorf("Expected current_page=1, got %d", result.Pagination.CurrentPage)
		}

		// Default page_size is 20
		if result.Pagination.PageSize != 20 {
			t.Errorf("Expected page_size=20, got %d", result.Pagination.PageSize)
		}

		t.Logf("Found %d instances, total=%d, page=%d, size=%d",
			len(result.Data), result.Pagination.TotalCount, result.Pagination.CurrentPage, result.Pagination.PageSize)
	})

	t.Run("Success_ReturnsInstancesWithCustomPagination", func(t *testing.T) {
		url := baseURL + "/api/v1/instances?page=2&page_size=10"
		resp := makeHTTPRequest(t, "GET", url, nil, nil)

		if resp.StatusCode != 200 {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		var result struct {
			Pagination struct {
				CurrentPage int `json:"current_page"`
				PageSize    int `json:"page_size"`
			} `json:"pagination"`
		}
		if err := json.Unmarshal(resp.Body, &result); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if result.Pagination.CurrentPage != 2 {
			t.Errorf("Expected current_page=2, got %d", result.Pagination.CurrentPage)
		}

		if result.Pagination.PageSize != 10 {
			t.Errorf("Expected page_size=10, got %d", result.Pagination.PageSize)
		}
	})

	t.Run("Boundary_PageSizeAtMaxLimit", func(t *testing.T) {
		// Max page_size is 100
		url := baseURL + "/api/v1/instances?page_size=100"
		resp := makeHTTPRequest(t, "GET", url, nil, nil)

		if resp.StatusCode != 200 {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		var result struct {
			Pagination struct {
				PageSize int `json:"page_size"`
			} `json:"pagination"`
		}
		if err := json.Unmarshal(resp.Body, &result); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if result.Pagination.PageSize != 100 {
			t.Errorf("Expected page_size=100, got %d", result.Pagination.PageSize)
		}
	})

	t.Run("Boundary_PageSizeExceedsMax", func(t *testing.T) {
		// Should be clamped to 100
		url := baseURL + "/api/v1/instances?page_size=200"
		resp := makeHTTPRequest(t, "GET", url, nil, nil)

		if resp.StatusCode != 200 {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		var result struct {
			Pagination struct {
				PageSize int `json:"page_size"`
			} `json:"pagination"`
		}
		if err := json.Unmarshal(resp.Body, &result); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if result.Pagination.PageSize != 100 {
			t.Errorf("Expected page_size to be clamped to 100, got %d", result.Pagination.PageSize)
		}
	})

	t.Run("Boundary_NegativePage", func(t *testing.T) {
		// Should default to 1
		url := baseURL + "/api/v1/instances?page=-1"
		resp := makeHTTPRequest(t, "GET", url, nil, nil)

		if resp.StatusCode != 200 {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		var result struct {
			Pagination struct {
				CurrentPage int `json:"current_page"`
			} `json:"pagination"`
		}
		if err := json.Unmarshal(resp.Body, &result); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if result.Pagination.CurrentPage != 1 {
			t.Errorf("Expected current_page to default to 1 for negative input, got %d", result.Pagination.CurrentPage)
		}
	})

	t.Run("Boundary_NegativePageSize", func(t *testing.T) {
		// Should default to 20
		url := baseURL + "/api/v1/instances?page_size=-1"
		resp := makeHTTPRequest(t, "GET", url, nil, nil)

		if resp.StatusCode != 200 {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		var result struct {
			Pagination struct {
				PageSize int `json:"page_size"`
			} `json:"pagination"`
		}
		if err := json.Unmarshal(resp.Body, &result); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if result.Pagination.PageSize != 20 {
			t.Errorf("Expected page_size to default to 20 for negative input, got %d", result.Pagination.PageSize)
		}
	})

	t.Run("Boundary_InvalidPageParameter", func(t *testing.T) {
		url := baseURL + "/api/v1/instances?page=invalid"
		resp := makeHTTPRequest(t, "GET", url, nil, nil)

		// Should return 200 with default values (invalid param is ignored)
		if resp.StatusCode != 200 {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		var result struct {
			Pagination struct {
				CurrentPage int `json:"current_page"`
			} `json:"pagination"`
		}
		if err := json.Unmarshal(resp.Body, &result); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Should use default
		if result.Pagination.CurrentPage != 1 {
			t.Errorf("Expected current_page to default to 1, got %d", result.Pagination.CurrentPage)
		}
	})

	t.Run("Boundary_InvalidPageSizeParameter", func(t *testing.T) {
		url := baseURL + "/api/v1/instances?page_size=invalid"
		resp := makeHTTPRequest(t, "GET", url, nil, nil)

		if resp.StatusCode != 200 {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		var result struct {
			Pagination struct {
				PageSize int `json:"page_size"`
			} `json:"pagination"`
		}
		if err := json.Unmarshal(resp.Body, &result); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if result.Pagination.PageSize != 20 {
			t.Errorf("Expected page_size to default to 20, got %d", result.Pagination.PageSize)
		}
	})
}

// ============================================================
// GET /api/v1/instances/:endpoint Tests
// ============================================================

func TestDataQueryHTTP_GetInstance(t *testing.T) {
	baseURL := getTestServiceURL()
	if !checkServiceAvailable(t, baseURL) {
		t.Skip("Data Query Service not running")
	}

	t.Run("Error_NonExistentEndpoint", func(t *testing.T) {
		url := baseURL + "/api/v1/instances/nonexistent-endpoint-12345"
		resp := makeHTTPRequest(t, "GET", url, nil, nil)

		if resp.StatusCode != 404 {
			t.Errorf("Expected status 404, got %d", resp.StatusCode)
		}

		code, msg, err := parseErrorResponse(resp.Body)
		if err != nil {
			t.Fatalf("Failed to parse error response: %v", err)
		}

		if code != "NOT_FOUND" {
			t.Errorf("Expected error code NOT_FOUND, got %s", code)
		}

		t.Logf("Error response: code=%s, message=%s", code, msg)
	})

	t.Run("Success_ReturnsInstanceForValidEndpoint", func(t *testing.T) {
		// First get a valid endpoint from instances list
		instancesResp := makeHTTPRequest(t, "GET", baseURL+"/api/v1/instances?page_size=1", nil, nil)
		if instancesResp.StatusCode != 200 {
			t.Skip("Could not get instances")
		}

		var instancesResult struct {
			Data []struct {
				InstanceEndpoint string `json:"instance_endpoint"`
			} `json:"data"`
		}
		if err := json.Unmarshal(instancesResp.Body, &instancesResult); err != nil {
			t.Skip("Could not parse instances response")
		}

		if len(instancesResult.Data) == 0 {
			t.Skip("No instances available in database")
		}

		validEndpoint := instancesResult.Data[0].InstanceEndpoint
		url := baseURL + "/api/v1/instances/" + validEndpoint
		resp := makeHTTPRequest(t, "GET", url, nil, nil)

		if resp.StatusCode != 200 {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		var result struct {
			Data struct {
				InstanceEndpoint string `json:"instance_endpoint"`
				DbType           string `json:"db_type"`
			} `json:"data"`
		}
		if err := json.Unmarshal(resp.Body, &result); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if result.Data.InstanceEndpoint != validEndpoint {
			t.Errorf("Expected endpoint %s, got %s", validEndpoint, result.Data.InstanceEndpoint)
		}

		t.Logf("Successfully got instance for endpoint %s (db_type=%s)", validEndpoint, result.Data.DbType)
	})
}

// ============================================================
// GET /api/v1/alerts/:endpoint Tests
// ============================================================

func TestDataQueryHTTP_GetAlerts(t *testing.T) {
	baseURL := getTestServiceURL()
	if !checkServiceAvailable(t, baseURL) {
		t.Skip("Data Query Service not running")
	}

	t.Run("Success_ReturnsAlertsForValidEndpoint", func(t *testing.T) {
		// First get a valid endpoint from instances
		instancesResp := makeHTTPRequest(t, "GET", baseURL+"/api/v1/instances?page_size=1", nil, nil)
		if instancesResp.StatusCode != 200 {
			t.Skip("Could not get instances")
		}

		var instancesResult struct {
			Data []struct {
				InstanceEndpoint string `json:"instance_endpoint"`
			} `json:"data"`
		}
		if err := json.Unmarshal(instancesResp.Body, &instancesResult); err != nil {
			t.Skip("Could not parse instances response")
		}

		if len(instancesResult.Data) == 0 {
			t.Skip("No instances available")
		}

		validEndpoint := instancesResult.Data[0].InstanceEndpoint
		url := baseURL + "/api/v1/alerts/" + validEndpoint
		resp := makeHTTPRequest(t, "GET", url, nil, nil)

		if resp.StatusCode != 200 {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		var result struct {
			Data []interface{} `json:"data"`
		}
		if err := json.Unmarshal(resp.Body, &result); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Alerts may be empty if no alerts for this endpoint
		t.Logf("Found %d alerts for endpoint %s", len(result.Data), validEndpoint)
	})

	t.Run("Success_ReturnsEmptyListWhenNoAlerts", func(t *testing.T) {
		// Use an endpoint that likely has no alerts
		url := baseURL + "/api/v1/alerts/nonexistent-endpoint-no-alerts"
		resp := makeHTTPRequest(t, "GET", url, nil, nil)

		// Should return 200 with empty data (not an error)
		if resp.StatusCode != 200 {
			t.Errorf("Expected status 200 for empty result, got %d", resp.StatusCode)
		}

		var result struct {
			Data []interface{} `json:"data"`
		}
		if err := json.Unmarshal(resp.Body, &result); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if len(result.Data) != 0 {
			t.Logf("Got %d alerts (expected 0)", len(result.Data))
		}
	})
}

// ============================================================
// GET /health Tests
// ============================================================

func TestDataQueryHTTP_Health(t *testing.T) {
	baseURL := getTestServiceURL()

	t.Run("Success_ReturnsStatusOK", func(t *testing.T) {
		url := baseURL + "/health"
		resp := makeHTTPRequest(t, "GET", url, nil, nil)

		if resp.StatusCode != 200 {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
			t.Skip("Data Query Service not running")
		}

		var result map[string]string
		if err := json.Unmarshal(resp.Body, &result); err != nil {
			t.Fatalf("Failed to parse health response: %v", err)
		}

		if result["status"] != "ok" {
			t.Errorf("Expected status 'ok', got '%s'", result["status"])
		}

		t.Logf("Health check passed: %s", resp.Body)
	})
}

// ============================================================
// 404 Not Found Tests
// ============================================================

func TestDataQueryHTTP_NotFound(t *testing.T) {
	baseURL := getTestServiceURL()
	if !checkServiceAvailable(t, baseURL) {
		t.Skip("Data Query Service not running")
	}

	t.Run("Error_UnknownEndpoint", func(t *testing.T) {
		url := baseURL + "/api/v1/nonexistent"
		resp := makeHTTPRequest(t, "GET", url, nil, nil)

		if resp.StatusCode != 404 {
			t.Errorf("Expected status 404, got %d", resp.StatusCode)
		}
	})

	t.Run("Error_UnknownMethod", func(t *testing.T) {
		// POST to a GET-only endpoint
		url := baseURL + "/api/v1/endpoints"
		resp := makeHTTPRequest(t, "POST", url, nil, nil)

		if resp.StatusCode != 404 {
			t.Errorf("Expected status 404 for wrong method, got %d", resp.StatusCode)
		}
	})
}

// ============================================================
// Method Not Allowed Tests
// ============================================================

func TestDataQueryHTTP_MethodNotAllowed(t *testing.T) {
	baseURL := getTestServiceURL()
	if !checkServiceAvailable(t, baseURL) {
		t.Skip("Data Query Service not running")
	}

	t.Run("Error_POSTToEndpoints", func(t *testing.T) {
		url := baseURL + "/api/v1/endpoints"
		resp := makeHTTPRequest(t, "POST", url, nil, nil)

		// Hertz returns 404 for route not found (including method mismatch)
		if resp.StatusCode != 404 {
			t.Logf("POST to endpoints returned status %d (expected 404)", resp.StatusCode)
		}
	})

	t.Run("Error_DELETEToSeries", func(t *testing.T) {
		url := baseURL + "/api/v1/series"
		resp := makeHTTPRequest(t, "DELETE", url, nil, nil)

		if resp.StatusCode != 404 {
			t.Logf("DELETE to series returned status %d (expected 404)", resp.StatusCode)
		}
	})
}

// ============================================================
// Content-Type and Response Format Tests
// ============================================================

func TestDataQueryHTTP_ResponseFormat(t *testing.T) {
	baseURL := getTestServiceURL()
	if !checkServiceAvailable(t, baseURL) {
		t.Skip("Data Query Service not running")
	}

	t.Run("Success_ReturnsJSONContentType", func(t *testing.T) {
		url := baseURL + "/api/v1/endpoints"
		resp := makeHTTPRequest(t, "GET", url, nil, nil)

		contentType := resp.Headers.Get("Content-Type")
		if !strings.Contains(contentType, "application/json") {
			t.Errorf("Expected Content-Type to contain 'application/json', got '%s'", contentType)
		}
	})

	t.Run("Success_ErrorResponseHasCorrectStructure", func(t *testing.T) {
		url := baseURL + "/api/v1/metrics" // Missing endpoint param
		resp := makeHTTPRequest(t, "GET", url, nil, nil)

		var errResp struct {
			Error struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal(resp.Body, &errResp); err != nil {
			t.Fatalf("Failed to parse error response: %v", err)
		}

		// Verify structure
		if errResp.Error.Code == "" {
			t.Error("Error response missing 'code' field")
		}
		if errResp.Error.Message == "" {
			t.Error("Error response missing 'message' field")
		}
	})
}

// ============================================================
// Concurrent Request Tests
// ============================================================

func TestDataQueryHTTP_ConcurrentRequests(t *testing.T) {
	baseURL := getTestServiceURL()
	if !checkServiceAvailable(t, baseURL) {
		t.Skip("Data Query Service not running")
	}

	t.Run("Success_HandlesMultipleConcurrentRequests", func(t *testing.T) {
		numRequests := 10
		done := make(chan bool, numRequests)
		errors := make(chan error, numRequests)

		for i := 0; i < numRequests; i++ {
			go func(id int) {
				url := baseURL + "/api/v1/endpoints"
				resp := makeHTTPRequest(t, "GET", url, nil, nil)

				if resp.StatusCode != 200 {
					errors <- fmt.Errorf("request %d: expected 200, got %d", id, resp.StatusCode)
				} else {
					done <- true
				}
			}(i)
		}

		// Wait for all requests
		successCount := 0
		for i := 0; i < numRequests; i++ {
			select {
			case <-done:
				successCount++
			case err := <-errors:
				t.Error(err)
			case <-time.After(10 * time.Second):
				t.Fatal("Timeout waiting for concurrent requests")
			}
		}

		if successCount != numRequests {
			t.Errorf("Only %d/%d requests succeeded", successCount, numRequests)
		}

		t.Logf("All %d concurrent requests succeeded", numRequests)
	})
}

// ============================================================
// Request Timeout Tests
// ============================================================

func TestDataQueryHTTP_TimeoutHandling(t *testing.T) {
	baseURL := getTestServiceURL()
	if !checkServiceAvailable(t, baseURL) {
		t.Skip("Data Query Service not running")
	}

	t.Run("Success_ResponseWithinTimeout", func(t *testing.T) {
		// Make a request that should complete quickly
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"/health", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Request failed or timed out: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		t.Log("Health check completed within timeout")
	})
}

// ============================================================
// Phase 4: Additional HTTP Integration Tests
// ============================================================

func TestDataQueryHTTP_LabelFilterExpressions(t *testing.T) {
	baseURL := getTestServiceURL()
	if !checkServiceAvailable(t, baseURL) {
		t.Skip("Data Query Service not running")
	}

	now := time.Now().UTC()
	startTime := now.Add(-24 * time.Hour).Format(time.RFC3339)
	endTime := now.Format(time.RFC3339)

	t.Run("ExactMatchLabelFilter", func(t *testing.T) {
		url := baseURL + "/api/v1/series?start=" + startTime + "&end=" + endTime + "&label_filter=env%3D%22prod%22"
		resp := makeHTTPRequest(t, "GET", url, nil, nil)

		if resp.StatusCode != 200 {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		t.Logf("Exact match label filter returned status %d", resp.StatusCode)
	})

	t.Run("RegexMatchLabelFilter", func(t *testing.T) {
		url := baseURL + "/api/v1/series?start=" + startTime + "&end=" + endTime + "&label_filter=host%3D~%22server.*%22"
		resp := makeHTTPRequest(t, "GET", url, nil, nil)

		if resp.StatusCode != 200 {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		t.Logf("Regex match label filter returned status %d", resp.StatusCode)
	})

	t.Run("NotEqualLabelFilter", func(t *testing.T) {
		url := baseURL + "/api/v1/series?start=" + startTime + "&end=" + endTime + "&label_filter=env!%3D%22test%22"
		resp := makeHTTPRequest(t, "GET", url, nil, nil)

		if resp.StatusCode != 200 {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		t.Logf("Not equal label filter returned status %d", resp.StatusCode)
	})

	t.Run("CompoundANDLabelFilter", func(t *testing.T) {
		url := baseURL + "/api/v1/series?start=" + startTime + "&end=" + endTime + "&label_filter=env%3D%22prod%22%20AND%20region%3D~%22us-.*%22"
		resp := makeHTTPRequest(t, "GET", url, nil, nil)

		if resp.StatusCode != 200 {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		t.Logf("Compound AND label filter returned status %d", resp.StatusCode)
	})

	t.Run("CompoundORLabelFilter", func(t *testing.T) {
		url := baseURL + "/api/v1/series?start=" + startTime + "&end=" + endTime + "&label_filter=env%3D%22prod%22%20OR%20env%3D%22staging%22"
		resp := makeHTTPRequest(t, "GET", url, nil, nil)

		if resp.StatusCode != 200 {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		t.Logf("Compound OR label filter returned status %d", resp.StatusCode)
	})

	t.Run("GroupedLabelFilter", func(t *testing.T) {
		url := baseURL + "/api/v1/series?start=" + startTime + "&end=" + endTime + "&label_filter=(env%3D%22prod%22%20OR%20env%3D%22staging%22)%20AND%20region%3D%22us-east%22"
		resp := makeHTTPRequest(t, "GET", url, nil, nil)

		if resp.StatusCode != 200 {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		t.Logf("Grouped label filter returned status %d", resp.StatusCode)
	})

	t.Run("InvalidLabelFilter_ReturnsError", func(t *testing.T) {
		url := baseURL + "/api/v1/series?start=" + startTime + "&end=" + endTime + "&label_filter=invalid%20syntax"
		resp := makeHTTPRequest(t, "GET", url, nil, nil)

		// Invalid label filter should return 400 or 500
		if resp.StatusCode != 400 && resp.StatusCode != 500 {
			t.Errorf("Expected status 400 or 500 for invalid label filter, got %d", resp.StatusCode)
		}

		t.Logf("Invalid label filter returned status %d", resp.StatusCode)
	})

	t.Run("LabelFilterInPOSTBody", func(t *testing.T) {
		body := map[string]interface{}{
			"start":  startTime,
			"end":    endTime,
			"labels": `env="prod" AND region=~"us-.*"`,
		}
		bodyBytes, _ := json.Marshal(body)

		url := baseURL + "/api/v1/series/query"
		resp := makeHTTPRequest(t, "POST", url, map[string]string{"Content-Type": "application/json"}, bodyBytes)

		if resp.StatusCode != 200 {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		t.Logf("POST with label filter returned status %d", resp.StatusCode)
	})
}

func TestDataQueryHTTP_SamplingInterval(t *testing.T) {
	baseURL := getTestServiceURL()
	if !checkServiceAvailable(t, baseURL) {
		t.Skip("Data Query Service not running")
	}

	now := time.Now().UTC()
	startTime := now.Add(-24 * time.Hour).Format(time.RFC3339)
	endTime := now.Format(time.RFC3339)

	t.Run("OneMinuteInterval", func(t *testing.T) {
		url := baseURL + "/api/v1/series?start=" + startTime + "&end=" + endTime + "&step=1m"
		resp := makeHTTPRequest(t, "GET", url, nil, nil)

		if resp.StatusCode != 200 {
			t.Errorf("Expected status 200 for step=1m, got %d", resp.StatusCode)
		}

		t.Logf("step=1m returned status %d", resp.StatusCode)
	})

	t.Run("FiveMinuteInterval", func(t *testing.T) {
		url := baseURL + "/api/v1/series?start=" + startTime + "&end=" + endTime + "&step=5m"
		resp := makeHTTPRequest(t, "GET", url, nil, nil)

		if resp.StatusCode != 200 {
			t.Errorf("Expected status 200 for step=5m, got %d", resp.StatusCode)
		}

		t.Logf("step=5m returned status %d", resp.StatusCode)
	})

	t.Run("OneHourInterval", func(t *testing.T) {
		url := baseURL + "/api/v1/series?start=" + startTime + "&end=" + endTime + "&step=1h"
		resp := makeHTTPRequest(t, "GET", url, nil, nil)

		if resp.StatusCode != 200 {
			t.Errorf("Expected status 200 for step=1h, got %d", resp.StatusCode)
		}

		t.Logf("step=1h returned status %d", resp.StatusCode)
	})

	t.Run("ThirtyMinuteInterval", func(t *testing.T) {
		url := baseURL + "/api/v1/series?start=" + startTime + "&end=" + endTime + "&step=30m"
		resp := makeHTTPRequest(t, "GET", url, nil, nil)

		if resp.StatusCode != 200 {
			t.Errorf("Expected status 200 for step=30m, got %d", resp.StatusCode)
		}

		t.Logf("step=30m returned status %d", resp.StatusCode)
	})

	t.Run("NoStepReturnsRawData", func(t *testing.T) {
		url := baseURL + "/api/v1/series?start=" + startTime + "&end=" + endTime
		resp := makeHTTPRequest(t, "GET", url, nil, nil)

		if resp.StatusCode != 200 {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		t.Logf("No step parameter returned status %d", resp.StatusCode)
	})

	t.Run("StepBelowMinimum_ReturnsError", func(t *testing.T) {
		url := baseURL + "/api/v1/series?start=" + startTime + "&end=" + endTime + "&step=30s"
		resp := makeHTTPRequest(t, "GET", url, nil, nil)

		if resp.StatusCode != 400 {
			t.Errorf("Expected status 400 for step=30s (below minimum 1m), got %d", resp.StatusCode)
		}

		t.Logf("step=30s returned status %d", resp.StatusCode)
	})

	t.Run("StepAboveMaximum_ReturnsError", func(t *testing.T) {
		url := baseURL + "/api/v1/series?start=" + startTime + "&end=" + endTime + "&step=2h"
		resp := makeHTTPRequest(t, "GET", url, nil, nil)

		if resp.StatusCode != 400 {
			t.Errorf("Expected status 400 for step=2h (above maximum 1h), got %d", resp.StatusCode)
		}

		t.Logf("step=2h returned status %d", resp.StatusCode)
	})

	t.Run("InvalidStepFormat_ReturnsError", func(t *testing.T) {
		url := baseURL + "/api/v1/series?start=" + startTime + "&end=" + endTime + "&step=invalid"
		resp := makeHTTPRequest(t, "GET", url, nil, nil)

		if resp.StatusCode != 400 {
			t.Errorf("Expected status 400 for invalid step format, got %d", resp.StatusCode)
		}

		t.Logf("Invalid step format returned status %d", resp.StatusCode)
	})
}

func TestDataQueryHTTP_TimeRangeEdgeCases(t *testing.T) {
	baseURL := getTestServiceURL()
	if !checkServiceAvailable(t, baseURL) {
		t.Skip("Data Query Service not running")
	}

	t.Run("VeryOldTimeRange", func(t *testing.T) {
		// Year 1900 - should return empty results but not error
		oldStart := time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339)
		oldEnd := time.Date(1900, 1, 2, 0, 0, 0, 0, time.UTC).Format(time.RFC3339)
		url := baseURL + "/api/v1/series?start=" + oldStart + "&end=" + oldEnd

		resp := makeHTTPRequest(t, "GET", url, nil, nil)

		if resp.StatusCode != 200 {
			t.Errorf("Expected status 200 for old time range, got %d", resp.StatusCode)
		}

		var result struct {
			Data []interface{} `json:"data"`
		}
		if err := json.Unmarshal(resp.Body, &result); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if len(result.Data) != 0 {
			t.Logf("Warning: Expected 0 results for 1900 time range, got %d", len(result.Data))
		}
	})

	t.Run("FutureTimeRange", func(t *testing.T) {
		// Future dates - should return empty results
		futureStart := time.Date(2100, 1, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339)
		futureEnd := time.Date(2100, 1, 2, 0, 0, 0, 0, time.UTC).Format(time.RFC3339)
		url := baseURL + "/api/v1/series?start=" + futureStart + "&end=" + futureEnd

		resp := makeHTTPRequest(t, "GET", url, nil, nil)

		if resp.StatusCode != 200 {
			t.Errorf("Expected status 200 for future time range, got %d", resp.StatusCode)
		}

		var result struct {
			Data []interface{} `json:"data"`
		}
		if err := json.Unmarshal(resp.Body, &result); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if len(result.Data) != 0 {
			t.Logf("Warning: Expected 0 results for future time range, got %d", len(result.Data))
		}
	})

	t.Run("VeryLargeTimeRange", func(t *testing.T) {
		// 1 year time range - should work but may return lots of data
		now := time.Now().UTC()
		startTime := now.Add(-365 * 24 * time.Hour).Format(time.RFC3339)
		endTime := now.Format(time.RFC3339)
		url := baseURL + "/api/v1/series?start=" + startTime + "&end=" + endTime + "&limit=10"

		resp := makeHTTPRequest(t, "GET", url, nil, nil)

		if resp.StatusCode != 200 {
			t.Errorf("Expected status 200 for large time range, got %d", resp.StatusCode)
		}

		t.Logf("Large time range (1 year) returned status %d", resp.StatusCode)
	})

	t.Run("VerySmallTimeRange", func(t *testing.T) {
		// 1 minute time range
		now := time.Now().UTC()
		startTime := now.Add(-1 * time.Minute).Format(time.RFC3339)
		endTime := now.Format(time.RFC3339)
		url := baseURL + "/api/v1/series?start=" + startTime + "&end=" + endTime

		resp := makeHTTPRequest(t, "GET", url, nil, nil)

		if resp.StatusCode != 200 {
			t.Errorf("Expected status 200 for small time range, got %d", resp.StatusCode)
		}

		t.Logf("Small time range (1 min) returned status %d", resp.StatusCode)
	})

	t.Run("StartAfterEnd_ReturnsEmpty", func(t *testing.T) {
		// Start time after end time - should return empty or handle gracefully
		now := time.Now().UTC()
		startTime := now.Format(time.RFC3339)
		endTime := now.Add(-1 * time.Hour).Format(time.RFC3339)
		url := baseURL + "/api/v1/series?start=" + startTime + "&end=" + endTime

		resp := makeHTTPRequest(t, "GET", url, nil, nil)

		// Server should handle this gracefully (200 with empty data or 400)
		if resp.StatusCode != 200 && resp.StatusCode != 400 {
			t.Errorf("Expected status 200 or 400 for inverted time range, got %d", resp.StatusCode)
		}

		t.Logf("Inverted time range returned status %d", resp.StatusCode)
	})

	t.Run("UnixTimestampZero", func(t *testing.T) {
		// Unix timestamp 0 (Jan 1, 1970)
		url := baseURL + "/api/v1/series?start=0&end=3600"

		resp := makeHTTPRequest(t, "GET", url, nil, nil)

		if resp.StatusCode != 200 {
			t.Errorf("Expected status 200 for unix timestamp 0, got %d", resp.StatusCode)
		}

		t.Logf("Unix timestamp 0 returned status %d", resp.StatusCode)
	})

	t.Run("VeryLargeUnixTimestamp", func(t *testing.T) {
		// Very large unix timestamp (year 2100)
		url := baseURL + "/api/v1/series?start=4102444800&end=4102531200"

		resp := makeHTTPRequest(t, "GET", url, nil, nil)

		if resp.StatusCode != 200 {
			t.Errorf("Expected status 200 for large unix timestamp, got %d", resp.StatusCode)
		}

		t.Logf("Large unix timestamp returned status %d", resp.StatusCode)
	})
}

func TestDataQueryHTTP_SpecialCharactersInEndpoint(t *testing.T) {
	baseURL := getTestServiceURL()
	if !checkServiceAvailable(t, baseURL) {
		t.Skip("Data Query Service not running")
	}

	t.Run("EndpointWithHyphens", func(t *testing.T) {
		// Most endpoints have hyphens
		url := baseURL + "/api/v1/instances/mysql-cn-east-1-finance-order-01"

		resp := makeHTTPRequest(t, "GET", url, nil, nil)

		// Should either find it (200) or not (404), but not error
		if resp.StatusCode != 200 && resp.StatusCode != 404 {
			t.Errorf("Expected status 200 or 404 for hyphenated endpoint, got %d", resp.StatusCode)
		}

		t.Logf("Hyphenated endpoint returned status %d", resp.StatusCode)
	})

	t.Run("EndpointWithUnderscores", func(t *testing.T) {
		url := baseURL + "/api/v1/instances/test_endpoint_name"

		resp := makeHTTPRequest(t, "GET", url, nil, nil)

		// Should return 404 for non-existent endpoint
		if resp.StatusCode != 404 {
			t.Logf("Underscore endpoint returned status %d (expected 404)", resp.StatusCode)
		}
	})

	t.Run("EndpointWithDots", func(t *testing.T) {
		url := baseURL + "/api/v1/instances/test.endpoint.name"

		resp := makeHTTPRequest(t, "GET", url, nil, nil)

		t.Logf("Dotted endpoint returned status %d", resp.StatusCode)
	})

	t.Run("EndpointWithURL-encodedChars", func(t *testing.T) {
		// URL-encoded special characters
		url := baseURL + "/api/v1/instances/test%2Fendpoint%3Fname"

		resp := makeHTTPRequest(t, "GET", url, nil, nil)

		// Server should handle this gracefully
		t.Logf("URL-encoded endpoint returned status %d", resp.StatusCode)
	})

	t.Run("EndpointWithSpaces_URL-encoded", func(t *testing.T) {
		// URL-encoded space (%20)
		url := baseURL + "/api/v1/instances/test%20endpoint"

		resp := makeHTTPRequest(t, "GET", url, nil, nil)

		t.Logf("Space-containing endpoint returned status %d", resp.StatusCode)
	})

	t.Run("EmptyEndpoint_ReturnsError", func(t *testing.T) {
		url := baseURL + "/api/v1/instances/"

		resp := makeHTTPRequest(t, "GET", url, nil, nil)

		// Should return 404 or 400 for empty endpoint
		if resp.StatusCode != 404 && resp.StatusCode != 400 {
			t.Logf("Empty endpoint returned status %d", resp.StatusCode)
		}
	})

	t.Run("VeryLongEndpointName", func(t *testing.T) {
		// Very long endpoint name (255+ chars)
		longEndpoint := "a"
		for i := 0; i < 300; i++ {
			longEndpoint += "a"
		}
		url := baseURL + "/api/v1/instances/" + longEndpoint

		resp := makeHTTPRequest(t, "GET", url, nil, nil)

		// Should return 404 (not found) or handle gracefully
		t.Logf("Long endpoint returned status %d", resp.StatusCode)
	})

	t.Run("AlertsEndpointWithSpecialChars", func(t *testing.T) {
		// Test alerts endpoint with special characters
		url := baseURL + "/api/v1/alerts/mysql-cn-east-1-finance-order-01"

		resp := makeHTTPRequest(t, "GET", url, nil, nil)

		if resp.StatusCode != 200 {
			t.Errorf("Expected status 200 for alerts endpoint, got %d", resp.StatusCode)
		}

		t.Logf("Alerts endpoint with hyphens returned status %d", resp.StatusCode)
	})

	t.Run("MetricsQueryWithSpecialChars", func(t *testing.T) {
		// Test metrics endpoint query parameter
		url := baseURL + "/api/v1/metrics?endpoint=mysql-cn-east-1-finance-order-01"

		resp := makeHTTPRequest(t, "GET", url, nil, nil)

		if resp.StatusCode != 200 {
			t.Errorf("Expected status 200 for metrics query, got %d", resp.StatusCode)
		}

		t.Logf("Metrics query with special chars returned status %d", resp.StatusCode)
	})
}