package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"

	"db-cockpit/apiserver/internal/agent"
	"db-cockpit/apiserver/internal/envelope"
	"db-cockpit/apiserver/internal/model"
)

/* ================= 模型设置：大模型 / 嵌入模型服务 CRUD + 测试连接 ================= */

// maskKey：API Key 脱敏展示（前 4 + *** + 后 4）
func maskKey(k string) string {
	if k == "" {
		return ""
	}
	r := []rune(k)
	if len(r) <= 8 {
		return "******"
	}
	return string(r[:4]) + "***" + string(r[len(r)-4:])
}

// paramsJSON：校验用户输入的 params 为合法 JSON（空则存 null）
func paramsJSON(raw json.RawMessage) (datatypes.JSON, bool) {
	s := strings.TrimSpace(string(raw))
	if s == "" || s == "null" {
		return nil, true
	}
	if !json.Valid([]byte(s)) {
		return nil, false
	}
	return datatypes.JSON(s), true
}

/* ---------- 大模型配置 ---------- */

type modelConfigBody struct {
	Name     string          `json:"name"`
	Provider string          `json:"provider"`
	BaseURL  string          `json:"baseUrl"`
	APIKey   string          `json:"apiKey"` // 更新时留空 = 保持原值
	Model    string          `json:"model"`
	Params   json.RawMessage `json:"params"`
	Remark   string          `json:"remark"`
	Enabled  *bool           `json:"enabled"`
}

func (h *H) ListModelConfigs(c *gin.Context) {
	var rows []model.ModelConfig
	h.DB.Order("created_at asc").Find(&rows)
	if rows == nil {
		rows = []model.ModelConfig{}
	}
	for i := range rows {
		rows[i].APIKeyMask = maskKey(rows[i].APIKey)
	}
	envelope.OK(c, rows)
}

func (h *H) CreateModelConfig(c *gin.Context) {
	var body modelConfigBody
	if err := c.ShouldBindJSON(&body); err != nil || body.Name == "" {
		envelope.BadRequest(c, "name required")
		return
	}
	params, ok := paramsJSON(body.Params)
	if !ok {
		envelope.BadRequest(c, "params is not valid JSON")
		return
	}
	now := time.Now().UnixMilli()
	m := model.ModelConfig{
		ID: agent.NewID("mc"), Name: body.Name, Provider: body.Provider,
		BaseURL: strings.TrimRight(body.BaseURL, "/"), APIKey: body.APIKey,
		Model: body.Model, Params: params, Remark: body.Remark,
		Enabled: body.Enabled == nil || *body.Enabled,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := h.DB.Create(&m).Error; err != nil {
		envelope.Internal(c, err)
		return
	}
	h.audit("model_config.create", "model_config", m.ID, map[string]interface{}{"name": m.Name})
	m.APIKeyMask = maskKey(m.APIKey)
	envelope.OK(c, m)
}

func (h *H) UpdateModelConfig(c *gin.Context) {
	var m model.ModelConfig
	if err := h.DB.Where("id = ?", c.Param("id")).First(&m).Error; err != nil {
		envelope.NotFound(c, "model config not found")
		return
	}
	var body modelConfigBody
	if err := c.ShouldBindJSON(&body); err != nil {
		envelope.BadRequest(c, "invalid body")
		return
	}
	params, ok := paramsJSON(body.Params)
	if !ok {
		envelope.BadRequest(c, "params is not valid JSON")
		return
	}
	m.Name = body.Name
	m.Provider = body.Provider
	m.BaseURL = strings.TrimRight(body.BaseURL, "/")
	m.Model = body.Model
	m.Params = params
	m.Remark = body.Remark
	if body.Enabled != nil {
		m.Enabled = *body.Enabled
	}
	if body.APIKey != "" { // 留空 = 保持原密钥
		m.APIKey = body.APIKey
	}
	m.UpdatedAt = time.Now().UnixMilli()
	if err := h.DB.Save(&m).Error; err != nil {
		envelope.Internal(c, err)
		return
	}
	h.audit("model_config.update", "model_config", m.ID, map[string]interface{}{"name": m.Name})
	m.APIKeyMask = maskKey(m.APIKey)
	envelope.OK(c, m)
}

func (h *H) DeleteModelConfig(c *gin.Context) {
	res := h.DB.Where("id = ?", c.Param("id")).Delete(&model.ModelConfig{})
	if res.Error != nil || res.RowsAffected == 0 {
		envelope.NotFound(c, "model config not found")
		return
	}
	h.audit("model_config.delete", "model_config", c.Param("id"), nil)
	envelope.OK(c, gin.H{"ok": true})
}

/* ---------- 嵌入模型服务配置 ---------- */

type embeddingConfigBody struct {
	Name      string          `json:"name"`
	BaseURL   string          `json:"baseUrl"`
	APIKey    string          `json:"apiKey"` // 更新时留空 = 保持原值
	Model     string          `json:"model"`
	Dimension int             `json:"dimension"`
	Params    json.RawMessage `json:"params"`
	Remark    string          `json:"remark"`
	Enabled   *bool           `json:"enabled"`
}

func (h *H) ListEmbeddingConfigs(c *gin.Context) {
	var rows []model.EmbeddingConfig
	h.DB.Order("created_at asc").Find(&rows)
	if rows == nil {
		rows = []model.EmbeddingConfig{}
	}
	for i := range rows {
		rows[i].APIKeyMask = maskKey(rows[i].APIKey)
	}
	envelope.OK(c, rows)
}

func (h *H) CreateEmbeddingConfig(c *gin.Context) {
	var body embeddingConfigBody
	if err := c.ShouldBindJSON(&body); err != nil || body.Name == "" {
		envelope.BadRequest(c, "name required")
		return
	}
	params, ok := paramsJSON(body.Params)
	if !ok {
		envelope.BadRequest(c, "params is not valid JSON")
		return
	}
	now := time.Now().UnixMilli()
	e := model.EmbeddingConfig{
		ID: agent.NewID("ec"), Name: body.Name,
		BaseURL: strings.TrimRight(body.BaseURL, "/"), APIKey: body.APIKey,
		Model: body.Model, Dimension: body.Dimension, Params: params, Remark: body.Remark,
		Enabled: body.Enabled == nil || *body.Enabled,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := h.DB.Create(&e).Error; err != nil {
		envelope.Internal(c, err)
		return
	}
	h.audit("embedding_config.create", "embedding_config", e.ID, map[string]interface{}{"name": e.Name})
	e.APIKeyMask = maskKey(e.APIKey)
	envelope.OK(c, e)
}

func (h *H) UpdateEmbeddingConfig(c *gin.Context) {
	var e model.EmbeddingConfig
	if err := h.DB.Where("id = ?", c.Param("id")).First(&e).Error; err != nil {
		envelope.NotFound(c, "embedding config not found")
		return
	}
	var body embeddingConfigBody
	if err := c.ShouldBindJSON(&body); err != nil {
		envelope.BadRequest(c, "invalid body")
		return
	}
	params, ok := paramsJSON(body.Params)
	if !ok {
		envelope.BadRequest(c, "params is not valid JSON")
		return
	}
	e.Name = body.Name
	e.BaseURL = strings.TrimRight(body.BaseURL, "/")
	e.Model = body.Model
	e.Dimension = body.Dimension
	e.Params = params
	e.Remark = body.Remark
	if body.Enabled != nil {
		e.Enabled = *body.Enabled
	}
	if body.APIKey != "" {
		e.APIKey = body.APIKey
	}
	e.UpdatedAt = time.Now().UnixMilli()
	if err := h.DB.Save(&e).Error; err != nil {
		envelope.Internal(c, err)
		return
	}
	h.audit("embedding_config.update", "embedding_config", e.ID, map[string]interface{}{"name": e.Name})
	e.APIKeyMask = maskKey(e.APIKey)
	envelope.OK(c, e)
}

func (h *H) DeleteEmbeddingConfig(c *gin.Context) {
	res := h.DB.Where("id = ?", c.Param("id")).Delete(&model.EmbeddingConfig{})
	if res.Error != nil || res.RowsAffected == 0 {
		envelope.NotFound(c, "embedding config not found")
		return
	}
	h.audit("embedding_config.delete", "embedding_config", c.Param("id"), nil)
	envelope.OK(c, gin.H{"ok": true})
}

/* ---------- 测试连接（真实调用远端，密钥不出后端） ---------- */

type TestResult struct {
	OK        bool   `json:"ok"`
	LatencyMs int64  `json:"latencyMs"`
	Message   string `json:"message"`
	Dimension int    `json:"dimension,omitempty"` // 嵌入服务实测维度
}

var testHTTPClient = &http.Client{Timeout: 10 * time.Second}

// postJSON：向远端发 POST，返回状态码与响应体（最多 1MB）
func postJSON(url, apiKey string, payload interface{}) (int, []byte, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return 0, nil, err
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	resp, err := testHTTPClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	return resp.StatusCode, body, err
}

func truncateStr(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

// TestModelConfig：OpenAI 兼容最小 chat 请求，验证 baseUrl / 密钥 / 模型名
func (h *H) TestModelConfig(c *gin.Context) {
	var m model.ModelConfig
	if err := h.DB.Where("id = ?", c.Param("id")).First(&m).Error; err != nil {
		envelope.NotFound(c, "model config not found")
		return
	}
	if m.BaseURL == "" || m.Model == "" {
		envelope.OK(c, TestResult{OK: false, Message: "缺少 baseUrl 或模型名，无法测试"})
		return
	}
	start := time.Now()
	status, body, err := postJSON(m.BaseURL+"/chat/completions", m.APIKey, map[string]interface{}{
		"model":      m.Model,
		"messages":   []map[string]string{{"role": "user", "content": "ping"}},
		"max_tokens": 1,
		"stream":     false,
	})
	latency := time.Since(start).Milliseconds()
	if err != nil {
		envelope.OK(c, TestResult{OK: false, LatencyMs: latency, Message: "无法连接：" + err.Error()})
		return
	}
	if status != http.StatusOK {
		envelope.OK(c, TestResult{OK: false, LatencyMs: latency, Message: truncateStr("远端返回 "+http.StatusText(status)+"："+string(body), 300)})
		return
	}
	envelope.OK(c, TestResult{OK: true, LatencyMs: latency, Message: "连接成功，模型响应正常"})
}

// TestEmbeddingConfig：向 /embeddings 发一条最小请求，并解析实测向量维度
func (h *H) TestEmbeddingConfig(c *gin.Context) {
	var e model.EmbeddingConfig
	if err := h.DB.Where("id = ?", c.Param("id")).First(&e).Error; err != nil {
		envelope.NotFound(c, "embedding config not found")
		return
	}
	if e.BaseURL == "" || e.Model == "" {
		envelope.OK(c, TestResult{OK: false, Message: "缺少 baseUrl 或模型名，无法测试"})
		return
	}
	start := time.Now()
	status, body, err := postJSON(e.BaseURL+"/embeddings", e.APIKey, map[string]interface{}{
		"model": e.Model,
		"input": "ping",
	})
	latency := time.Since(start).Milliseconds()
	if err != nil {
		envelope.OK(c, TestResult{OK: false, LatencyMs: latency, Message: "无法连接：" + err.Error()})
		return
	}
	if status != http.StatusOK {
		envelope.OK(c, TestResult{OK: false, LatencyMs: latency, Message: truncateStr("远端返回 "+http.StatusText(status)+"："+string(body), 300)})
		return
	}
	var parsed struct {
		Data []struct {
			Embedding []float64 `json:"embedding"`
		} `json:"data"`
	}
	dim := 0
	if json.Unmarshal(body, &parsed) == nil && len(parsed.Data) > 0 {
		dim = len(parsed.Data[0].Embedding)
	}
	envelope.OK(c, TestResult{OK: true, LatencyMs: latency, Message: "连接成功，服务响应正常", Dimension: dim})
}
