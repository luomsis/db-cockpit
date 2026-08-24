package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"db-cockpit/apiserver/internal/agent"
	"db-cockpit/apiserver/internal/envelope"
	"db-cockpit/apiserver/internal/mcpclient"
	"db-cockpit/apiserver/internal/model"
)

/* ================= 插件域：tool_definitions 注册表 + MCP 发现/健康（D15 · 工具注册表 §10） ================= */

const pluginVersionScope = "plugin_domain"

// validToolTransition 状态机：draft→active（需先定级 risk）、draft/active→deprecated；
// deprecated 不可复活（重新发现会生成新草案）。
func validToolTransition(from, to string) bool {
	switch {
	case from == "draft" && to == "active":
		return true
	case from == "draft" && to == "deprecated":
		return true
	case from == "active" && to == "deprecated":
		return true
	default:
		return false
	}
}

// prefixToolName 跨 server 冲突隔离：<server>.<tool>（subagent toolset 前缀解析依据）
func prefixToolName(server, tool string) (string, bool) {
	if server == "" || tool == "" || server == "." || tool == "." {
		return "", false
	}
	return server + "." + tool, true
}

// ListToolDefinitions GET /api/tool-definitions?server_id=&status=&category=
// 响应携带 configVersion（agentcluster 轮询锚点，轮次边界生效）
func (h *H) ListToolDefinitions(c *gin.Context) {
	q := h.DB.Model(&model.ToolDefinition{})
	if v := c.Query("server_id"); v != "" {
		q = q.Where("server_id = ?", v)
	}
	if v := c.Query("status"); v != "" {
		q = q.Where("status = ?", v)
	}
	if v := c.Query("category"); v != "" {
		q = q.Where("category = ?", v)
	}
	var rows []model.ToolDefinition
	if err := q.Order("tool_name asc").Find(&rows).Error; err != nil {
		envelope.Internal(c, err)
		return
	}
	if rows == nil {
		rows = []model.ToolDefinition{}
	}
	envelope.OK(c, gin.H{"items": rows, "total": len(rows), "configVersion": h.pluginConfigVersion()})
}

type toolDefBody struct {
	DisplayName     *string         `json:"displayName"`
	Description     *string         `json:"description"`
	UsageHints      json.RawMessage `json:"usageHints"`
	Category        *string         `json:"category"`
	DbTypes         json.RawMessage `json:"dbTypes"`
	RoutingPriority *int            `json:"routingPriority"`
	RiskLevel       *string         `json:"riskLevel"`
	AuditLevel      *string         `json:"auditLevel"`
	RateLimit       json.RawMessage `json:"rateLimit"`
	Idempotent      *bool           `json:"idempotent"`
	OutputSchema    json.RawMessage `json:"outputSchema"`
	OutputCard      *string         `json:"outputCard"`
	ExecutionMode   *string         `json:"executionMode"`
	TimeoutMs       *int            `json:"timeoutMs"`
	Status          *string         `json:"status"`
}

// UpdateToolDefinition PUT /api/tool-definitions/:id —— 人工定级编辑与状态流转
func (h *H) UpdateToolDefinition(c *gin.Context) {
	var td model.ToolDefinition
	if err := h.DB.Where("id = ?", c.Param("id")).First(&td).Error; err != nil {
		envelope.NotFound(c, "tool definition not found")
		return
	}
	var body toolDefBody
	if err := c.ShouldBindJSON(&body); err != nil {
		envelope.BadRequest(c, "invalid body")
		return
	}
	inEnum := func(v string, allowed ...string) bool {
		for _, a := range allowed {
			if v == a {
				return true
			}
		}
		return false
	}
	if body.RiskLevel != nil && !inEnum(*body.RiskLevel, "L0", "L1", "L2") {
		envelope.BadRequest(c, "risk_level must be L0/L1/L2")
		return
	}
	if body.AuditLevel != nil && !inEnum(*body.AuditLevel, "full", "summary", "none") {
		envelope.BadRequest(c, "audit_level must be full/summary/none")
		return
	}
	if body.ExecutionMode != nil && !inEnum(*body.ExecutionMode, "sync", "async") {
		envelope.BadRequest(c, "execution_mode must be sync/async")
		return
	}
	if body.Status != nil && !inEnum(*body.Status, "draft", "active", "deprecated") {
		envelope.BadRequest(c, "status must be draft/active/deprecated")
		return
	}
	if body.Status != nil && *body.Status != td.Status && !validToolTransition(td.Status, *body.Status) {
		envelope.BadRequest(c, "invalid status transition: "+td.Status+" → "+*body.Status)
		return
	}
	applyToolDefBody(&td, body)
	if td.Status == "active" && td.RiskLevel == "" {
		envelope.BadRequest(c, "risk_level required before active（人工定级前置）")
		return
	}
	td.Version++
	td.UpdatedAt = time.Now().UnixMilli()
	if err := h.DB.Save(&td).Error; err != nil {
		envelope.Internal(c, err)
		return
	}
	h.bumpPluginVersion()
	h.audit("tool_def.update", "tool_definition", td.ID, map[string]interface{}{"status": td.Status, "version": td.Version})
	envelope.OK(c, td)
}

func applyToolDefBody(td *model.ToolDefinition, body toolDefBody) {
	if body.DisplayName != nil {
		td.DisplayName = *body.DisplayName
	}
	if body.Description != nil {
		td.Description = *body.Description
	}
	if v, ok := paramsJSON(body.UsageHints); ok {
		td.UsageHints = v
	}
	if body.Category != nil {
		td.Category = *body.Category
	}
	if v, ok := paramsJSON(body.DbTypes); ok {
		td.DbTypes = v
	}
	if body.RoutingPriority != nil {
		td.RoutingPriority = *body.RoutingPriority
	}
	if body.RiskLevel != nil {
		td.RiskLevel = *body.RiskLevel
	}
	if body.AuditLevel != nil {
		td.AuditLevel = *body.AuditLevel
	}
	if v, ok := paramsJSON(body.RateLimit); ok {
		td.RateLimit = v
	}
	if body.Idempotent != nil {
		td.Idempotent = *body.Idempotent
	}
	if v, ok := paramsJSON(body.OutputSchema); ok {
		td.OutputSchema = v
	}
	if body.OutputCard != nil {
		td.OutputCard = *body.OutputCard
	}
	if body.ExecutionMode != nil {
		td.ExecutionMode = *body.ExecutionMode
	}
	if body.TimeoutMs != nil {
		td.TimeoutMs = *body.TimeoutMs
	}
	if body.Status != nil {
		td.Status = *body.Status
	}
}

// DiscoverMcpServer POST /api/mcp-servers/:id/discover —— apiserver 直连 tools/list 生成草案。
// 语义：按 origin_tool_name upsert——draft 刷新发现侧字段（schema/描述），
// active/deprecated 的定级结果不覆盖；本次消失的工具置 deprecated；在场工具 health=ok。
func (h *H) DiscoverMcpServer(c *gin.Context) {
	var m model.McpServerConfig
	if err := h.DB.Where("id = ?", c.Param("id")).First(&m).Error; err != nil {
		envelope.NotFound(c, "mcp server not found")
		return
	}
	if err := validateMcpEndpoint(m); err != "" {
		envelope.BadRequest(c, err)
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()
	_ = mcpclient.Initialize(ctx, m.BaseURL) // 宽松：无状态 server 可不握手，以 tools/list 为准
	tools, err := mcpclient.ToolsList(ctx, m.BaseURL)
	if err != nil {
		envelope.Fail(c, http.StatusBadGateway, http.StatusBadGateway, "mcp server unreachable: "+err.Error())
		return
	}

	var existing []model.ToolDefinition
	h.DB.Where("server_id = ?", m.ID).Find(&existing)
	byOrigin := make(map[string]*model.ToolDefinition, len(existing))
	for i := range existing {
		byOrigin[existing[i].OriginToolName] = &existing[i]
	}
	now := time.Now()
	ms := now.UnixMilli()
	created, updated := 0, 0
	seen := map[string]bool{}
	for _, t := range tools {
		if t.Name == "" {
			continue
		}
		seen[t.Name] = true
		if ex := byOrigin[t.Name]; ex != nil {
			if ex.Status == "draft" {
				if v, ok := paramsJSON(t.InputSchema); ok {
					ex.InputSchema = v
				}
				if t.Description != "" {
					ex.Description = t.Description
				}
				if t.Title != "" {
					ex.DisplayName = t.Title
				}
				ex.Health, ex.LastCheckedAt, ex.UpdatedAt = "ok", &now, ms
				h.DB.Save(ex)
				updated++
			} else {
				// 定级行不覆盖，仅刷新运行态健康
				h.DB.Model(ex).Updates(map[string]interface{}{"health": "ok", "last_checked_at": now, "updated_at": ms})
			}
			continue
		}
		name, ok := prefixToolName(m.Name, t.Name)
		if !ok {
			continue
		}
		display := t.Title
		if display == "" {
			display = t.Name
		}
		schema, ok := paramsJSON(t.InputSchema)
		if !ok {
			schema = datatypes.JSON([]byte("{}"))
		}
		td := model.ToolDefinition{
			ID: agent.NewID("td"), ToolName: name, ServerID: m.ID, OriginToolName: t.Name,
			DisplayName: display, Description: t.Description, InputSchema: schema,
			ExecutionMode: "sync", Status: "draft", Health: "ok", LastCheckedAt: &now,
			Version: 1, CreatedAt: ms, UpdatedAt: ms,
		}
		if err := h.DB.Create(&td).Error; err != nil {
			envelope.Internal(c, err)
			return
		}
		created++
	}
	deprecated := 0
	for i := range existing {
		ex := &existing[i]
		if !seen[ex.OriginToolName] && ex.Status != "deprecated" {
			ex.Status, ex.UpdatedAt = "deprecated", ms
			h.DB.Save(ex)
			deprecated++
		}
	}
	h.bumpPluginVersion()
	h.audit("mcp_server.discover", "mcp_server", m.ID,
		map[string]interface{}{"discovered": len(tools), "created": created, "deprecated": deprecated})
	envelope.OK(c, gin.H{"discovered": len(tools), "created": created, "updated": updated, "deprecated": deprecated})
}

// HealthMcpServer POST /api/mcp-servers/:id/health —— 连通性检查，刷新该 server 全部工具的
// health/last_checked_at（status 不动）。健康是运行态信号，不 bump config_version。
func (h *H) HealthMcpServer(c *gin.Context) {
	var m model.McpServerConfig
	if err := h.DB.Where("id = ?", c.Param("id")).First(&m).Error; err != nil {
		envelope.NotFound(c, "mcp server not found")
		return
	}
	if err := validateMcpEndpoint(m); err != "" {
		envelope.BadRequest(c, err)
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 8*time.Second)
	defer cancel()
	health, errMsg := "ok", ""
	if _, err := mcpclient.ToolsList(ctx, m.BaseURL); err != nil {
		health, errMsg = "unreachable", err.Error()
	}
	now := time.Now()
	h.DB.Model(&model.ToolDefinition{}).Where("server_id = ?", m.ID).
		Updates(map[string]interface{}{"health": health, "last_checked_at": now})
	var n int64
	h.DB.Model(&model.ToolDefinition{}).Where("server_id = ?", m.ID).Count(&n)
	envelope.OK(c, gin.H{"health": health, "tools": n, "error": errMsg})
}

func validateMcpEndpoint(m model.McpServerConfig) string {
	if m.Transport != "http" {
		return "stdio transport not supported (MVP: http only)"
	}
	if m.BaseURL == "" {
		return "base_url required"
	}
	return ""
}

/* ---- 管理面变更版本（agentcluster 直读 config_versions，轮次边界生效） ---- */

func (h *H) bumpPluginVersion() {
	now := time.Now().UnixMilli()
	res := h.DB.Model(&model.ConfigVersion{}).Where("scope = ?", pluginVersionScope).
		Updates(map[string]interface{}{"version": gorm.Expr("version + 1"), "updated_at": now})
	if res.Error == nil && res.RowsAffected == 0 {
		h.DB.Create(&model.ConfigVersion{Scope: pluginVersionScope, Version: 1, UpdatedAt: now})
	}
}

func (h *H) pluginConfigVersion() int64 {
	var cv model.ConfigVersion
	if err := h.DB.Where("scope = ?", pluginVersionScope).First(&cv).Error; err != nil {
		return 0
	}
	return cv.Version
}
