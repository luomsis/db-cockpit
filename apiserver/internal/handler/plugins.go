package handler

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"db-cockpit/apiserver/internal/agent"
	"db-cockpit/apiserver/internal/envelope"
	"db-cockpit/apiserver/internal/model"
)

/* ================= 插件中心：MCP 服务 / Skills CRUD ================= */

/* ---------- MCP 服务 ---------- */

type mcpServerBody struct {
	Name      string          `json:"name"`
	Transport string          `json:"transport"` // http（MVP 仅 http，stdio 校验拒绝）
	BaseURL   string          `json:"baseUrl"`   // http 端点地址
	Command   string          `json:"command"`   // stdio 预留（MVP 不支持）
	Args      json.RawMessage `json:"args"`
	Env       json.RawMessage `json:"env"`
	Status    *string         `json:"status"` // active | deprecated
	Remark    string          `json:"remark"`
	Enabled   *bool           `json:"enabled"`
}

// normalizeTransport MVP 仅支持 http（D15）；stdio 拒绝
func normalizeTransport(t string) (string, bool) {
	switch t {
	case "":
		return "http", true
	case "http":
		return t, true
	default:
		return "", false
	}
}

// validateBaseURL http 端点必填且须为 http(s) 地址
func validateBaseURL(u string) bool {
	return strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://")
}

func mcpStatusOrDefault(s *string) string {
	if s == nil || *s == "" {
		return "active"
	}
	return *s
}

func (h *H) ListMcpServers(c *gin.Context) {
	var rows []model.McpServerConfig
	h.DB.Order("created_at asc").Find(&rows)
	if rows == nil {
		rows = []model.McpServerConfig{}
	}
	envelope.OK(c, rows)
}

func (h *H) CreateMcpServer(c *gin.Context) {
	var body mcpServerBody
	if err := c.ShouldBindJSON(&body); err != nil || body.Name == "" {
		envelope.BadRequest(c, "name required")
		return
	}
	transport, ok := normalizeTransport(body.Transport)
	if !ok {
		envelope.BadRequest(c, "transport not supported (MVP: http only)")
		return
	}
	if !validateBaseURL(body.BaseURL) {
		envelope.BadRequest(c, "base_url required (http:// or https://)")
		return
	}
	if body.Status != nil && *body.Status != "active" && *body.Status != "deprecated" {
		envelope.BadRequest(c, "status must be active/deprecated")
		return
	}
	args, ok := paramsJSON(body.Args)
	if !ok {
		envelope.BadRequest(c, "args is not valid JSON")
		return
	}
	env, ok := paramsJSON(body.Env)
	if !ok {
		envelope.BadRequest(c, "env is not valid JSON")
		return
	}
	now := time.Now().UnixMilli()
	m := model.McpServerConfig{
		ID: agent.NewID("mcp"), Name: body.Name, Transport: transport, BaseURL: body.BaseURL,
		Command: body.Command, Args: args, Env: env, Version: 1, Status: mcpStatusOrDefault(body.Status),
		Remark: body.Remark,
		Enabled: body.Enabled == nil || *body.Enabled,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := h.DB.Create(&m).Error; err != nil {
		envelope.Internal(c, err)
		return
	}
	h.bumpPluginVersion()
	h.audit("mcp_server.create", "mcp_server", m.ID, map[string]interface{}{"name": m.Name})
	envelope.OK(c, m)
}

func (h *H) UpdateMcpServer(c *gin.Context) {
	var m model.McpServerConfig
	if err := h.DB.Where("id = ?", c.Param("id")).First(&m).Error; err != nil {
		envelope.NotFound(c, "mcp server not found")
		return
	}
	var body mcpServerBody
	if err := c.ShouldBindJSON(&body); err != nil {
		envelope.BadRequest(c, "invalid body")
		return
	}
	transport, ok := normalizeTransport(body.Transport)
	if !ok {
		envelope.BadRequest(c, "transport not supported (MVP: http only)")
		return
	}
	if !validateBaseURL(body.BaseURL) {
		envelope.BadRequest(c, "base_url required (http:// or https://)")
		return
	}
	if body.Status != nil && *body.Status != "active" && *body.Status != "deprecated" {
		envelope.BadRequest(c, "status must be active/deprecated")
		return
	}
	args, ok := paramsJSON(body.Args)
	if !ok {
		envelope.BadRequest(c, "args is not valid JSON")
		return
	}
	env, ok := paramsJSON(body.Env)
	if !ok {
		envelope.BadRequest(c, "env is not valid JSON")
		return
	}
	m.Name = body.Name
	m.Transport = transport
	m.BaseURL = body.BaseURL
	m.Command = body.Command
	m.Args = args
	m.Env = env
	m.Status = mcpStatusOrDefault(body.Status)
	m.Remark = body.Remark
	if body.Enabled != nil {
		m.Enabled = *body.Enabled
	}
	m.Version++
	m.UpdatedAt = time.Now().UnixMilli()
	if err := h.DB.Save(&m).Error; err != nil {
		envelope.Internal(c, err)
		return
	}
	h.bumpPluginVersion()
	h.audit("mcp_server.update", "mcp_server", m.ID, map[string]interface{}{"name": m.Name, "version": m.Version})
	envelope.OK(c, m)
}

func (h *H) DeleteMcpServer(c *gin.Context) {
	res := h.DB.Where("id = ?", c.Param("id")).Delete(&model.McpServerConfig{})
	if res.Error != nil || res.RowsAffected == 0 {
		envelope.NotFound(c, "mcp server not found")
		return
	}
	// 级联清理该 server 的注册表行（表结构即契约：不留孤儿工具）
	h.DB.Where("server_id = ?", c.Param("id")).Delete(&model.ToolDefinition{})
	h.bumpPluginVersion()
	h.audit("mcp_server.delete", "mcp_server", c.Param("id"), nil)
	envelope.OK(c, gin.H{"ok": true})
}

/* ---------- Skills ---------- */

type skillBody struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Content     string `json:"content"`
	Enabled     *bool  `json:"enabled"`
}

func (h *H) ListSkills(c *gin.Context) {
	var rows []model.SkillConfig
	h.DB.Order("created_at asc").Find(&rows)
	if rows == nil {
		rows = []model.SkillConfig{}
	}
	envelope.OK(c, rows)
}

func (h *H) CreateSkill(c *gin.Context) {
	var body skillBody
	if err := c.ShouldBindJSON(&body); err != nil || body.Name == "" {
		envelope.BadRequest(c, "name required")
		return
	}
	now := time.Now().UnixMilli()
	s := model.SkillConfig{
		ID: agent.NewID("sk"), Name: body.Name, Description: body.Description, Content: body.Content,
		Enabled: body.Enabled == nil || *body.Enabled,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := h.DB.Create(&s).Error; err != nil {
		envelope.Internal(c, err)
		return
	}
	h.audit("skill.create", "skill", s.ID, map[string]interface{}{"name": s.Name})
	envelope.OK(c, s)
}

func (h *H) UpdateSkill(c *gin.Context) {
	var s model.SkillConfig
	if err := h.DB.Where("id = ?", c.Param("id")).First(&s).Error; err != nil {
		envelope.NotFound(c, "skill not found")
		return
	}
	var body skillBody
	if err := c.ShouldBindJSON(&body); err != nil {
		envelope.BadRequest(c, "invalid body")
		return
	}
	s.Name = body.Name
	s.Description = body.Description
	s.Content = body.Content
	if body.Enabled != nil {
		s.Enabled = *body.Enabled
	}
	s.UpdatedAt = time.Now().UnixMilli()
	if err := h.DB.Save(&s).Error; err != nil {
		envelope.Internal(c, err)
		return
	}
	h.audit("skill.update", "skill", s.ID, map[string]interface{}{"name": s.Name})
	envelope.OK(c, s)
}

func (h *H) DeleteSkill(c *gin.Context) {
	res := h.DB.Where("id = ?", c.Param("id")).Delete(&model.SkillConfig{})
	if res.Error != nil || res.RowsAffected == 0 {
		envelope.NotFound(c, "skill not found")
		return
	}
	h.audit("skill.delete", "skill", c.Param("id"), nil)
	envelope.OK(c, gin.H{"ok": true})
}
