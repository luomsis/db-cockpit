package mcpclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestToolsListJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"jsonrpc":"2.0","id":1,"result":{"tools":[
			{"name":"get_cpu","title":"CPU 查询","description":"查询实例 CPU","inputSchema":{"type":"object","properties":{"instance_id":{"type":"string"}}}},
			{"name":"get_qps"}
		]}}`)
	}))
	defer srv.Close()
	tools, err := ToolsList(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("ToolsList: %v", err)
	}
	if len(tools) != 2 {
		t.Fatalf("tools len = %d, want 2", len(tools))
	}
	if tools[0].Name != "get_cpu" || tools[0].Title != "CPU 查询" {
		t.Errorf("tool[0] = %+v", tools[0])
	}
	if !strings.Contains(string(tools[0].InputSchema), "instance_id") {
		t.Errorf("inputSchema 未透传: %s", tools[0].InputSchema)
	}
}

// streamable-http 兼容：text/event-stream 帧承载 JSON-RPC 响应
func TestToolsListSSE(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "event: message\ndata: {\"jsonrpc\":\"2.0\",\"id\":1,\"result\":{\"tools\":[{\"name\":\"get_mem\"}]}}\n\n")
	}))
	defer srv.Close()
	tools, err := ToolsList(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("ToolsList(SSE): %v", err)
	}
	if len(tools) != 1 || tools[0].Name != "get_mem" {
		t.Fatalf("tools = %+v", tools)
	}
}

func TestRPCError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"jsonrpc":"2.0","id":1,"error":{"code":-32601,"message":"method not found"}}`)
	}))
	defer srv.Close()
	if _, err := ToolsList(context.Background(), srv.URL); err == nil || !strings.Contains(err.Error(), "-32601") {
		t.Fatalf("应返回 JSON-RPC 错误，got %v", err)
	}
}

func TestHTTPStatusError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	if _, err := ToolsList(context.Background(), srv.URL); err == nil || !strings.Contains(err.Error(), "500") {
		t.Fatalf("应返回 http 状态错误，got %v", err)
	}
}

// 握手宽松：initialize 失败（404）不阻断 tools/list（无状态 server 场景）
func TestInitializeLenient(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req struct {
			Method string `json:"method"`
		}
		_ = json.Unmarshal(body, &req)
		w.Header().Set("Content-Type", "application/json")
		if req.Method == "initialize" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		fmt.Fprint(w, `{"jsonrpc":"2.0","id":1,"result":{"tools":[]}}`)
	}))
	defer srv.Close()
	if err := Initialize(context.Background(), srv.URL); err == nil {
		t.Fatal("initialize 应失败（404）")
	}
	if tools, err := ToolsList(context.Background(), srv.URL); err != nil || len(tools) != 0 {
		t.Fatalf("tools/list 不应被握手失败阻断: tools=%v err=%v", tools, err)
	}
}

func TestTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(300 * time.Millisecond)
		fmt.Fprint(w, `{"jsonrpc":"2.0","id":1,"result":{"tools":[]}}`)
	}))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if _, err := ToolsList(ctx, srv.URL); err == nil {
		t.Fatal("超时应返回错误")
	}
}

func TestEmptyBaseURL(t *testing.T) {
	if _, err := ToolsList(context.Background(), ""); err == nil {
		t.Fatal("空 base url 应返回错误")
	}
}
