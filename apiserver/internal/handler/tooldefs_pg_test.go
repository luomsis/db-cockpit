package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"db-cockpit/apiserver/internal/model"
)

/* 插件域集成测试：真实 PG（不可达跳过）+ httptest mock MCP server；marker 自建自清。 */

func pluginTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		dsn = "host=localhost port=55432 user=graphiti password=graphiti dbname=db_cockpit sslmode=disable"
	}
	gdb, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Skipf("PG 不可达，跳过：%v", err)
	}
	if err := gdb.AutoMigrate(&model.McpServerConfig{}, &model.SkillConfig{},
		&model.ToolDefinition{}, &model.ConfigVersion{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return gdb
}

func pluginRouter(gdb *gorm.DB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := &H{DB: gdb}
	r := gin.New()
	mp := r.Group("/api/mcp-servers")
	{
		mp.POST("", h.CreateMcpServer)
		mp.PUT("/:id", h.UpdateMcpServer)
		mp.DELETE("/:id", h.DeleteMcpServer)
		mp.POST("/:id/discover", h.DiscoverMcpServer)
		mp.POST("/:id/health", h.HealthMcpServer)
	}
	td := r.Group("/api/tool-definitions")
	{
		td.GET("", h.ListToolDefinitions)
		td.PUT("/:id", h.UpdateToolDefinition)
	}
	return r
}

// mockMCP 可变工具清单的 MCP http server（JSON-RPC；initialize 可选 404 模式）
type mockMCP struct {
	*httptest.Server
	mu       sync.Mutex
	tools    string
	initFail bool
}

func newMockMCP(t *testing.T, tools string) *mockMCP {
	m := &mockMCP{tools: tools}
	m.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req struct {
			Method string `json:"method"`
		}
		_ = json.Unmarshal(body, &req)
		w.Header().Set("Content-Type", "application/json")
		m.mu.Lock()
		initFail, tools := m.initFail, m.tools
		m.mu.Unlock()
		if req.Method == "initialize" {
			if initFail {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			fmt.Fprint(w, `{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2025-03-26","capabilities":{},"serverInfo":{"name":"mock"}}}`)
			return
		}
		if req.Method == "tools/list" {
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":1,"result":{"tools":%s}}`, tools)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(m.Close)
	return m
}

func (m *mockMCP) setTools(s string) { m.mu.Lock(); m.tools = s; m.mu.Unlock() }

func postBody(t *testing.T, r *gin.Engine, path, body string) (int, envBody) {
	t.Helper()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	var b envBody
	if err := json.Unmarshal(w.Body.Bytes(), &b); err != nil {
		t.Fatalf("解析响应 %s: %v", w.Body.String(), err)
	}
	return w.Code, b
}

func putBody(t *testing.T, r *gin.Engine, path, body string) (int, envBody) {
	t.Helper()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	var b envBody
	if err := json.Unmarshal(w.Body.Bytes(), &b); err != nil {
		t.Fatalf("解析响应 %s: %v", w.Body.String(), err)
	}
	return w.Code, b
}

type tdRow struct {
	ID          string          `json:"id"`
	ToolName    string          `json:"toolName"`
	Status      string          `json:"status"`
	Health      string          `json:"health"`
	RiskLevel   string          `json:"riskLevel"`
	Description string          `json:"description"`
	Version     int             `json:"version"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

func listTools(t *testing.T, r *gin.Engine, query string) ([]tdRow, int64) {
	t.Helper()
	code, body := doGet(t, r, "/api/tool-definitions"+query)
	if code != http.StatusOK {
		t.Fatalf("list tool-definitions: %d", code)
	}
	var resp struct {
		Items         []tdRow `json:"items"`
		ConfigVersion int64   `json:"configVersion"`
	}
	decodeData(t, body.Data, &resp)
	return resp.Items, resp.ConfigVersion
}

// createMarkerServer 经 API 创建 marker MCP server，返回 id；cleanup 自动清理注册表行
func createMarkerServer(t *testing.T, r *gin.Engine, gdb *gorm.DB, name, baseURL string) string {
	t.Helper()
	code, body := postBody(t, r, "/api/mcp-servers",
		fmt.Sprintf(`{"name":%q,"transport":"http","baseUrl":%q}`, name, baseURL))
	if code != http.StatusOK {
		t.Fatalf("创建 server: %d %s", code, body.Message)
	}
	var resp struct {
		ID string `json:"id"`
	}
	decodeData(t, body.Data, &resp)
	t.Cleanup(func() {
		gdb.Where("server_id = ?", resp.ID).Delete(&model.ToolDefinition{})
		gdb.Where("id = ?", resp.ID).Delete(&model.McpServerConfig{})
	})
	return resp.ID
}

/* ---- B1 发现正常路径 + B7 列表过滤/版本 ---- */

func TestDiscoverFlowAndFilters(t *testing.T) {
	gdb := pluginTestDB(t)
	r := pluginRouter(gdb)
	uniq := fmt.Sprintf("pm%d", time.Now().UnixNano())
	mock := newMockMCP(t, `[
		{"name":"get_cpu","title":"CPU 查询","description":"查 CPU","inputSchema":{"type":"object","properties":{"instance_id":{"type":"string"}}}},
		{"name":"get_qps","description":"查 QPS"}
	]`)
	sid := createMarkerServer(t, r, gdb, "metrics-"+uniq, mock.URL)

	code, body := postBody(t, r, "/api/mcp-servers/"+sid+"/discover", "{}")
	if code != http.StatusOK {
		t.Fatalf("discover: %d %s", code, body.Message)
	}
	var disc struct {
		Discovered int `json:"discovered"`
		Created    int `json:"created"`
		Updated    int `json:"updated"`
	}
	decodeData(t, body.Data, &disc)
	if disc.Discovered != 2 || disc.Created != 2 || disc.Updated != 0 {
		t.Fatalf("discover 计数不符: %+v", disc)
	}

	items, v1 := listTools(t, r, "?server_id="+sid)
	if len(items) != 2 {
		t.Fatalf("应聚出 2 个工具, got %d", len(items))
	}
	if items[0].ToolName != "metrics-"+uniq+".get_cpu" || items[0].Status != "draft" || items[0].Health != "ok" {
		t.Errorf("草案不符: %+v", items[0])
	}
	if !strings.Contains(string(items[0].InputSchema), "instance_id") {
		t.Errorf("input_schema 未带入: %s", items[0].InputSchema)
	}
	if v1 < 1 {
		t.Fatalf("configVersion 应 ≥1, got %d", v1)
	}

	// B7：过滤无命中
	none, _ := listTools(t, r, "?server_id="+sid+"&status=active")
	if len(none) != 0 {
		t.Errorf("draft 工具不应命中 status=active 过滤: %+v", none)
	}
}

/* ---- B2 发现幂等与定级保护 ---- */

func TestDiscoverIdempotentAndProtection(t *testing.T) {
	gdb := pluginTestDB(t)
	r := pluginRouter(gdb)
	uniq := fmt.Sprintf("pm%d", time.Now().UnixNano())
	mock := newMockMCP(t, `[
		{"name":"get_cpu","description":"v1 描述","inputSchema":{"type":"object","properties":{"a":{"type":"string"}}}},
		{"name":"get_qps","description":"qps"}
	]`)
	sid := createMarkerServer(t, r, gdb, "metrics-"+uniq, mock.URL)
	postBody(t, r, "/api/mcp-servers/"+sid+"/discover", "{}")

	// get_cpu 定级转 active（自定义描述 = 定级产物）
	items, _ := listTools(t, r, "?server_id="+sid)
	var cpuID string
	for _, it := range items {
		if strings.HasSuffix(it.ToolName, ".get_cpu") {
			cpuID = it.ID
		}
	}
	if cpuID == "" {
		t.Fatal("找不到 get_cpu")
	}
	code, body := putBody(t, r, "/api/tool-definitions/"+cpuID, `{"riskLevel":"L0","status":"active","description":"已定级描述"}`)
	if code != http.StatusOK {
		t.Fatalf("定级: %d %s", code, body.Message)
	}

	// 二次发现：schema 变更 + 描述变更 —— draft 刷新、active 不覆盖
	mock.setTools(`[
		{"name":"get_cpu","description":"v2 描述","inputSchema":{"type":"object","properties":{"b":{"type":"integer"}}}},
		{"name":"get_qps","description":"qps"}
	]`)
	code, body = postBody(t, r, "/api/mcp-servers/"+sid+"/discover", "{}")
	if code != http.StatusOK {
		t.Fatalf("re-discover: %d", code)
	}
	var disc struct {
		Created int `json:"created"`
		Updated int `json:"updated"`
	}
	decodeData(t, body.Data, &disc)
	if disc.Created != 0 || disc.Updated != 1 {
		t.Fatalf("重复发现应 0 建 1 更: %+v", disc)
	}
	items, _ = listTools(t, r, "?server_id="+sid)
	if len(items) != 2 {
		t.Fatalf("不应重复建行: %d", len(items))
	}
	for _, it := range items {
		if it.ID == cpuID {
			if it.Description != "已定级描述" {
				t.Errorf("active 行定级描述被覆盖: %q", it.Description)
			}
			if strings.Contains(string(it.InputSchema), `"b"`) {
				t.Errorf("active 行 schema 被覆盖: %s", it.InputSchema)
			}
			if it.Status != "active" {
				t.Errorf("active 状态被改: %q", it.Status)
			}
		} else {
			if it.Description != "qps" {
				t.Logf("draft 行描述=%q", it.Description)
			}
		}
	}
}

/* ---- B3 消失工具置 deprecated ---- */

func TestDiscoverDeprecateMissing(t *testing.T) {
	gdb := pluginTestDB(t)
	r := pluginRouter(gdb)
	uniq := fmt.Sprintf("pm%d", time.Now().UnixNano())
	mock := newMockMCP(t, `[{"name":"tool_a"},{"name":"tool_b"}]`)
	sid := createMarkerServer(t, r, gdb, "srv-"+uniq, mock.URL)
	postBody(t, r, "/api/mcp-servers/"+sid+"/discover", "{}")

	mock.setTools(`[{"name":"tool_a"}]`)
	code, body := postBody(t, r, "/api/mcp-servers/"+sid+"/discover", "{}")
	if code != http.StatusOK {
		t.Fatalf("re-discover: %d", code)
	}
	var disc struct {
		Deprecated int `json:"deprecated"`
	}
	decodeData(t, body.Data, &disc)
	if disc.Deprecated != 1 {
		t.Fatalf("消失工具应置 deprecated=1: %+v", disc)
	}
	dep, _ := listTools(t, r, "?server_id="+sid+"&status=deprecated")
	if len(dep) != 1 || !strings.HasSuffix(dep[0].ToolName, ".tool_b") {
		t.Fatalf("deprecated 过滤不符: %+v", dep)
	}
	// 再发现缺席者不重复计数
	postBody(t, r, "/api/mcp-servers/"+sid+"/discover", "{}")
	items, _ := listTools(t, r, "?server_id="+sid)
	if len(items) != 2 {
		t.Fatalf("行数不应变化: %d", len(items))
	}
}

/* ---- B4 发现错误路径 ---- */

func TestDiscoverErrorPaths(t *testing.T) {
	gdb := pluginTestDB(t)
	r := pluginRouter(gdb)
	uniq := fmt.Sprintf("pm%d", time.Now().UnixNano())

	if code, _ := postBody(t, r, "/api/mcp-servers/no-such/discover", "{}"); code != http.StatusNotFound {
		t.Errorf("无此 server 应 404, got %d", code)
	}

	// stdio server（直插 DB 绕过 API 校验）→ discover 400
	stdio := &model.McpServerConfig{ID: "mcp_stdio_" + uniq, Name: "stdio-" + uniq, Transport: "stdio",
		BaseURL: "http://127.0.0.1:1/mcp", Version: 1, Status: "active", Enabled: true}
	gdb.Create(stdio)
	t.Cleanup(func() { gdb.Where("id = ?", stdio.ID).Delete(&model.McpServerConfig{}) })
	if code, _ := postBody(t, r, "/api/mcp-servers/"+stdio.ID+"/discover", "{}"); code != http.StatusBadRequest {
		t.Errorf("stdio 应 400, got %d", code)
	}

	// 缺 base_url → 400
	noURL := &model.McpServerConfig{ID: "mcp_nourl_" + uniq, Name: "nourl-" + uniq, Transport: "http", Version: 1, Status: "active", Enabled: true}
	gdb.Create(noURL)
	t.Cleanup(func() { gdb.Where("id = ?", noURL.ID).Delete(&model.McpServerConfig{}) })
	if code, _ := postBody(t, r, "/api/mcp-servers/"+noURL.ID+"/discover", "{}"); code != http.StatusBadRequest {
		t.Errorf("缺 base_url 应 400, got %d", code)
	}

	// 目标宕机 → 502 且错误信息明确
	mock := newMockMCP(t, `[{"name":"x"}]`)
	sid := createMarkerServer(t, r, gdb, "down-"+uniq, mock.URL)
	mock.Server.Close() // 关停目标
	code, body := postBody(t, r, "/api/mcp-servers/"+sid+"/discover", "{}")
	if code != http.StatusBadGateway {
		t.Fatalf("宕机应 502, got %d", code)
	}
	if !strings.Contains(body.Message, "unreachable") {
		t.Errorf("错误信息应含 unreachable: %s", body.Message)
	}
}

/* ---- B5 健康检查 ---- */

func TestHealthCheck(t *testing.T) {
	gdb := pluginTestDB(t)
	r := pluginRouter(gdb)
	uniq := fmt.Sprintf("pm%d", time.Now().UnixNano())
	mock := newMockMCP(t, `[{"name":"tool_a"}]`)
	sid := createMarkerServer(t, r, gdb, "health-"+uniq, mock.URL)
	postBody(t, r, "/api/mcp-servers/"+sid+"/discover", "{}")

	code, body := postBody(t, r, "/api/mcp-servers/"+sid+"/health", "{}")
	if code != http.StatusOK {
		t.Fatalf("health: %d", code)
	}
	var hc struct {
		Health string `json:"health"`
		Tools  int64  `json:"tools"`
	}
	decodeData(t, body.Data, &hc)
	if hc.Health != "ok" || hc.Tools != 1 {
		t.Fatalf("健康结果不符: %+v", hc)
	}

	// 宕机 → unreachable，status 不受影响
	mock.Server.Close()
	code, body = postBody(t, r, "/api/mcp-servers/"+sid+"/health", "{}")
	decodeData(t, body.Data, &hc)
	if hc.Health != "unreachable" {
		t.Fatalf("宕机应 unreachable: %+v", hc)
	}
	items, _ := listTools(t, r, "?server_id="+sid)
	if len(items) != 1 || items[0].Status != "draft" || items[0].Health != "unreachable" {
		t.Fatalf("健康检查不应改 status: %+v", items)
	}
}

/* ---- B6 定级 PUT 生命周期 ---- */

func TestToolDefUpdateLifecycle(t *testing.T) {
	gdb := pluginTestDB(t)
	r := pluginRouter(gdb)
	uniq := fmt.Sprintf("pm%d", time.Now().UnixNano())
	mock := newMockMCP(t, `[{"name":"tool_x","description":"x"}]`)
	sid := createMarkerServer(t, r, gdb, "lc-"+uniq, mock.URL)
	postBody(t, r, "/api/mcp-servers/"+sid+"/discover", "{}")
	items, vBefore := listTools(t, r, "?server_id="+sid)
	if len(items) != 1 {
		t.Fatal("应有一个草案")
	}
	id := items[0].ID

	// 转 active 缺 risk → 400
	if code, body := putBody(t, r, "/api/tool-definitions/"+id, `{"status":"active"}`); code != http.StatusBadRequest || !strings.Contains(body.Message, "risk") {
		t.Fatalf("缺 risk 转 active 应 400: %d %s", code, body.Message)
	}
	// 带 risk → 200，version+1，configVersion 递增
	code, body := putBody(t, r, "/api/tool-definitions/"+id, `{"riskLevel":"L0","status":"active","category":"metrics"}`)
	if code != http.StatusOK {
		t.Fatalf("定级转 active: %d %s", code, body.Message)
	}
	var updated tdRow
	decodeData(t, body.Data, &updated)
	if updated.Status != "active" || updated.RiskLevel != "L0" || updated.Version != 2 {
		t.Fatalf("active 行不符: %+v", updated)
	}
	if _, vAfter := listTools(t, r, "?server_id="+sid); vAfter <= vBefore {
		t.Fatalf("configVersion 应递增: %d → %d", vBefore, vAfter)
	}
	// active → deprecated ✓
	if code, _ := putBody(t, r, "/api/tool-definitions/"+id, `{"status":"deprecated"}`); code != http.StatusOK {
		t.Fatalf("active→deprecated 应 200, got %d", code)
	}
	// deprecated → active ✗（不可复活）
	if code, _ := putBody(t, r, "/api/tool-definitions/"+id, `{"status":"active"}`); code != http.StatusBadRequest {
		t.Fatalf("deprecated→active 应 400, got %d", code)
	}
	// 非法枚举 / 404
	if code, _ := putBody(t, r, "/api/tool-definitions/"+id, `{"riskLevel":"L9"}`); code != http.StatusBadRequest {
		t.Errorf("非法 risk 应 400, got %d", code)
	}
	if code, _ := putBody(t, r, "/api/tool-definitions/td_unknown", `{"riskLevel":"L0"}`); code != http.StatusNotFound {
		t.Errorf("未知 id 应 404, got %d", code)
	}
}

/* ---- B8 MCP CRUD 校验回归 + 删除级联 ---- */

func TestMcpCrudValidationAndCascade(t *testing.T) {
	gdb := pluginTestDB(t)
	r := pluginRouter(gdb)
	uniq := fmt.Sprintf("pm%d", time.Now().UnixNano())

	if code, _ := postBody(t, r, "/api/mcp-servers",
		fmt.Sprintf(`{"name":"s-%s","transport":"stdio","command":"x"}`, uniq)); code != http.StatusBadRequest {
		t.Errorf("stdio 创建应 400, got %d", code)
	}
	if code, _ := postBody(t, r, "/api/mcp-servers",
		fmt.Sprintf(`{"name":"s-%s","transport":"http"}`, uniq)); code != http.StatusBadRequest {
		t.Errorf("缺 base_url 应 400, got %d", code)
	}
	mock := newMockMCP(t, `[{"name":"tool_a"}]`)
	code, body := postBody(t, r, "/api/mcp-servers",
		fmt.Sprintf(`{"name":"ok-%s","baseUrl":%q}`, uniq, mock.URL)) // transport 缺省 http
	if code != http.StatusOK {
		t.Fatalf("默认 http 创建应 200: %d", code)
	}
	var created struct {
		ID       string `json:"id"`
		BaseURL  string `json:"baseUrl"`
		Transport string `json:"transport"`
		Status   string `json:"status"`
	}
	decodeData(t, body.Data, &created)
	if created.Transport != "http" || created.BaseURL != mock.URL || created.Status != "active" {
		t.Fatalf("创建结果不符: %+v", created)
	}
	t.Cleanup(func() { gdb.Where("id = ?", created.ID).Delete(&model.McpServerConfig{}) })

	// 删除级联清理注册表行
	postBody(t, r, "/api/mcp-servers/"+created.ID+"/discover", "{}")
	var n int64
	gdb.Model(&model.ToolDefinition{}).Where("server_id = ?", created.ID).Count(&n)
	if n != 1 {
		t.Fatalf("发现后应有 1 行, got %d", n)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/api/mcp-servers/"+created.ID, nil))
	if w.Code != http.StatusOK {
		t.Fatalf("删除: %d", w.Code)
	}
	gdb.Model(&model.ToolDefinition{}).Where("server_id = ?", created.ID).Count(&n)
	if n != 0 {
		t.Fatalf("删除 server 应级联清理注册表行, 剩 %d", n)
	}
}
