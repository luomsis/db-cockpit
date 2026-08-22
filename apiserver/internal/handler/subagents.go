package handler

import (
	"encoding/json"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"

	"db-cockpit/apiserver/internal/agent"
	"db-cockpit/apiserver/internal/envelope"
	"db-cockpit/apiserver/internal/model"
)

/* ================= 管理面：动态 subagent / workflow 定义 CRUD =================
 * 这些表是 agent 运行时的装配输入（主 agent 直读 PG 按定义实例化 subagent，
 * docs《Agent集群开发规格》§4）；apiserver 管理页面经此 CRUD 写入。
 * 变更按轮次边界生效（agent 侧缓存 + config_version 校验）；重要变更走“新建版本”，历史版本保留供回放。
 */

/* ---------- subagent_defs ---------- */

type subagentBody struct {
	SubagentID   string          `json:"subagentId"`
	Status       string          `json:"status"` // active | shadow | deprecated（空默认 active）
	SysPrompt    string          `json:"sysPrompt"`
	Toolset      json.RawMessage `json:"toolset"`
	WorkflowRef  string          `json:"workflowRef"`
	ModelProfile json.RawMessage `json:"modelProfile"`
	Budget       json.RawMessage `json:"budget"`
	OutputCards  json.RawMessage `json:"outputCards"`
	RoutingHints json.RawMessage `json:"routingHints"`
	Remark       string          `json:"remark"`
}

func normalizeSubagentStatus(s string) (string, bool) {
	switch s {
	case "":
		return "active", true
	case "active", "shadow", "deprecated":
		return s, true
	}
	return "", false
}

type subagentJSONFields struct {
	toolset      json.RawMessage
	modelProfile json.RawMessage
	budget       json.RawMessage
	outputCards  json.RawMessage
	routingHints json.RawMessage
}

func parseSubagentFields(f subagentJSONFields) (toolset, profile, budget, cards, hints datatypes.JSON, ok bool) {
	if toolset, ok = paramsJSON(f.toolset); !ok {
		return
	}
	if profile, ok = paramsJSON(f.modelProfile); !ok {
		return
	}
	if budget, ok = paramsJSON(f.budget); !ok {
		return
	}
	if cards, ok = paramsJSON(f.outputCards); !ok {
		return
	}
	hints, ok = paramsJSON(f.routingHints)
	return
}

func (h *H) ListSubagentDefs(c *gin.Context) {
	var rows []model.SubagentDef
	h.DB.Order("subagent_id asc, version desc").Find(&rows)
	if rows == nil {
		rows = []model.SubagentDef{}
	}
	envelope.OK(c, rows)
}

func (h *H) CreateSubagentDef(c *gin.Context) {
	var body subagentBody
	if err := c.ShouldBindJSON(&body); err != nil || body.SubagentID == "" || body.SysPrompt == "" {
		envelope.BadRequest(c, "subagentId and sysPrompt required")
		return
	}
	status, ok := normalizeSubagentStatus(body.Status)
	if !ok {
		envelope.BadRequest(c, "status must be active, shadow or deprecated")
		return
	}
	toolset, profile, budget, cards, hints, ok := parseSubagentFields(subagentJSONFields{
		toolset: body.Toolset, modelProfile: body.ModelProfile, budget: body.Budget,
		outputCards: body.OutputCards, routingHints: body.RoutingHints,
	})
	if !ok {
		envelope.BadRequest(c, "toolset/modelProfile/budget/outputCards/routingHints must be valid JSON")
		return
	}
	// 版本号按 subagent_id 自增（新建即新版本）
	var maxVer int
	h.DB.Model(&model.SubagentDef{}).Where("subagent_id = ?", body.SubagentID).
		Select("COALESCE(MAX(version), 0)").Scan(&maxVer)
	now := time.Now().UnixMilli()
	m := model.SubagentDef{
		ID: agent.NewID("sa"), SubagentID: body.SubagentID, Version: maxVer + 1, Status: status,
		SysPrompt: body.SysPrompt, Toolset: toolset, WorkflowRef: body.WorkflowRef,
		ModelProfile: profile, Budget: budget, OutputCards: cards, RoutingHints: hints,
		Remark: body.Remark, CreatedAt: now, UpdatedAt: now,
	}
	if err := h.DB.Create(&m).Error; err != nil {
		envelope.Internal(c, err)
		return
	}
	h.audit("subagent.create", "subagent_def", m.ID, map[string]interface{}{
		"subagentId": m.SubagentID, "version": m.Version})
	envelope.OK(c, m)
}

func (h *H) UpdateSubagentDef(c *gin.Context) {
	var row model.SubagentDef
	if err := h.DB.Where("id = ?", c.Param("id")).First(&row).Error; err != nil {
		envelope.NotFound(c, "subagent def not found")
		return
	}
	var body subagentBody
	if err := c.ShouldBindJSON(&body); err != nil {
		envelope.BadRequest(c, "invalid body")
		return
	}
	updates := map[string]interface{}{"updated_at": time.Now().UnixMilli()}
	if body.Status != "" {
		status, ok := normalizeSubagentStatus(body.Status)
		if !ok {
			envelope.BadRequest(c, "status must be active, shadow or deprecated")
			return
		}
		updates["status"] = status
	}
	if body.SysPrompt != "" {
		updates["sys_prompt"] = body.SysPrompt
	}
	if body.WorkflowRef != "" {
		updates["workflow_ref"] = body.WorkflowRef
	}
	if body.Remark != "" {
		updates["remark"] = body.Remark
	}
	if len(body.Toolset) > 0 {
		if v, ok := paramsJSON(body.Toolset); ok {
			updates["toolset"] = v
		}
	}
	if len(body.ModelProfile) > 0 {
		if v, ok := paramsJSON(body.ModelProfile); ok {
			updates["model_profile"] = v
		}
	}
	if len(body.Budget) > 0 {
		if v, ok := paramsJSON(body.Budget); ok {
			updates["budget"] = v
		}
	}
	if len(body.OutputCards) > 0 {
		if v, ok := paramsJSON(body.OutputCards); ok {
			updates["output_cards"] = v
		}
	}
	if len(body.RoutingHints) > 0 {
		if v, ok := paramsJSON(body.RoutingHints); ok {
			updates["routing_hints"] = v
		}
	}
	// 身份不可改：subagent_id / version
	if err := h.DB.Model(&row).Updates(updates).Error; err != nil {
		envelope.Internal(c, err)
		return
	}
	h.audit("subagent.update", "subagent_def", row.ID, map[string]interface{}{"fields": body})
	envelope.OK(c, gin.H{"ok": true})
}

func (h *H) DeleteSubagentDef(c *gin.Context) {
	var row model.SubagentDef
	if err := h.DB.Where("id = ?", c.Param("id")).First(&row).Error; err != nil {
		envelope.NotFound(c, "subagent def not found")
		return
	}
	if err := h.DB.Delete(&row).Error; err != nil {
		envelope.Internal(c, err)
		return
	}
	h.audit("subagent.delete", "subagent_def", row.ID, map[string]interface{}{
		"subagentId": row.SubagentID, "version": row.Version})
	envelope.OK(c, gin.H{"ok": true})
}

/* ---------- workflow_defs ---------- */

type workflowBody struct {
	WorkflowID string          `json:"workflowId"`
	Name       string          `json:"name"`
	Level      string          `json:"level"` // L1_prompt | L2_graph（空默认 L1_prompt）
	Status     string          `json:"status"` // active | deprecated（空默认 active）
	Definition json.RawMessage `json:"definition"`
	Remark     string          `json:"remark"`
}

func normalizeWorkflowLevel(l string) (string, bool) {
	switch l {
	case "":
		return "L1_prompt", true
	case "L1_prompt", "L2_graph":
		return l, true
	}
	return "", false
}

func normalizeWorkflowStatus(s string) (string, bool) {
	switch s {
	case "":
		return "active", true
	case "active", "deprecated":
		return s, true
	}
	return "", false
}

func (h *H) ListWorkflowDefs(c *gin.Context) {
	var rows []model.WorkflowDef
	h.DB.Order("workflow_id asc, version desc").Find(&rows)
	if rows == nil {
		rows = []model.WorkflowDef{}
	}
	envelope.OK(c, rows)
}

func (h *H) CreateWorkflowDef(c *gin.Context) {
	var body workflowBody
	if err := c.ShouldBindJSON(&body); err != nil || body.WorkflowID == "" || body.Name == "" {
		envelope.BadRequest(c, "workflowId and name required")
		return
	}
	level, ok := normalizeWorkflowLevel(body.Level)
	if !ok {
		envelope.BadRequest(c, "level must be L1_prompt or L2_graph")
		return
	}
	status, ok := normalizeWorkflowStatus(body.Status)
	if !ok {
		envelope.BadRequest(c, "status must be active or deprecated")
		return
	}
	definition, ok := paramsJSON(body.Definition)
	if !ok {
		envelope.BadRequest(c, "definition is not valid JSON")
		return
	}
	var maxVer int
	h.DB.Model(&model.WorkflowDef{}).Where("workflow_id = ?", body.WorkflowID).
		Select("COALESCE(MAX(version), 0)").Scan(&maxVer)
	now := time.Now().UnixMilli()
	m := model.WorkflowDef{
		ID: agent.NewID("wf"), WorkflowID: body.WorkflowID, Version: maxVer + 1,
		Name: body.Name, Level: level, Status: status, Definition: definition,
		Remark: body.Remark, CreatedAt: now, UpdatedAt: now,
	}
	if err := h.DB.Create(&m).Error; err != nil {
		envelope.Internal(c, err)
		return
	}
	h.audit("workflow.create", "workflow_def", m.ID, map[string]interface{}{
		"workflowId": m.WorkflowID, "version": m.Version})
	envelope.OK(c, m)
}

func (h *H) UpdateWorkflowDef(c *gin.Context) {
	var row model.WorkflowDef
	if err := h.DB.Where("id = ?", c.Param("id")).First(&row).Error; err != nil {
		envelope.NotFound(c, "workflow def not found")
		return
	}
	var body workflowBody
	if err := c.ShouldBindJSON(&body); err != nil {
		envelope.BadRequest(c, "invalid body")
		return
	}
	updates := map[string]interface{}{"updated_at": time.Now().UnixMilli()}
	if body.Name != "" {
		updates["name"] = body.Name
	}
	if body.Level != "" {
		level, ok := normalizeWorkflowLevel(body.Level)
		if !ok {
			envelope.BadRequest(c, "level must be L1_prompt or L2_graph")
			return
		}
		updates["level"] = level
	}
	if body.Status != "" {
		status, ok := normalizeWorkflowStatus(body.Status)
		if !ok {
			envelope.BadRequest(c, "status must be active or deprecated")
			return
		}
		updates["status"] = status
	}
	if body.Remark != "" {
		updates["remark"] = body.Remark
	}
	if len(body.Definition) > 0 {
		if v, ok := paramsJSON(body.Definition); ok {
			updates["definition"] = v
		}
	}
	// 身份不可改：workflow_id / version
	if err := h.DB.Model(&row).Updates(updates).Error; err != nil {
		envelope.Internal(c, err)
		return
	}
	h.audit("workflow.update", "workflow_def", row.ID, map[string]interface{}{"fields": body})
	envelope.OK(c, gin.H{"ok": true})
}

func (h *H) DeleteWorkflowDef(c *gin.Context) {
	var row model.WorkflowDef
	if err := h.DB.Where("id = ?", c.Param("id")).First(&row).Error; err != nil {
		envelope.NotFound(c, "workflow def not found")
		return
	}
	if err := h.DB.Delete(&row).Error; err != nil {
		envelope.Internal(c, err)
		return
	}
	h.audit("workflow.delete", "workflow_def", row.ID, map[string]interface{}{
		"workflowId": row.WorkflowID, "version": row.Version})
	envelope.OK(c, gin.H{"ok": true})
}
