package handler

import (
	"encoding/json"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"

	"db-cockpit/apiserver/internal/envelope"
	"db-cockpit/apiserver/internal/model"
)

/* ================= OB 租户详情 ================= */

func (h *H) findTenant(c *gin.Context) (*model.ObTenant, bool) {
	var t model.ObTenant
	if err := h.DB.Where("id = ?", c.Param("tid")).First(&t).Error; err != nil {
		envelope.NotFound(c, "tenant not found")
		return nil, false
	}
	return &t, true
}

func (h *H) GetTenant(c *gin.Context) {
	t, ok := h.findTenant(c)
	if !ok {
		return
	}
	var dbs []model.ObTenantDb
	h.DB.Where("tenant_id = ?", t.ID).Order("id asc").Find(&dbs)
	var params []model.ClusterParam
	h.DB.Where("scope = ? AND tenant_id = ?", "tenant", t.ID).Order("id asc").Find(&params)
	envelope.OK(c, gin.H{
		"tenant":    t,
		"databases": dbs,
		"params":    params,
	})
}

// ResizeTenant：Unit 扩缩容（maxCpu/maxMemGb 等比下发到每个 Unit）
func (h *H) ResizeTenant(c *gin.Context) {
	t, ok := h.findTenant(c)
	if !ok {
		return
	}
	var body struct {
		MaxCpu   *float64 `json:"maxCpu"`
		MaxMemGb *float64 `json:"maxMemGb"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		envelope.BadRequest(c, "body required")
		return
	}
	if body.MaxCpu == nil && body.MaxMemGb == nil {
		envelope.BadRequest(c, "maxCpu or maxMemGb required")
		return
	}
	type unitRow struct {
		Zone      string  `json:"zone"`
		Observer  string  `json:"observer"`
		MaxCpu    float64 `json:"maxCpu"`
		UsedCpu   float64 `json:"usedCpu"`
		MaxMemGb  float64 `json:"maxMemGb"`
		UsedMemGb float64 `json:"usedMemGb"`
	}
	var units []unitRow
	_ = json.Unmarshal([]byte(t.Units), &units)
	newCpu, newMem := t.MaxCpu, t.MaxMemGb
	if body.MaxCpu != nil {
		newCpu = *body.MaxCpu
	}
	if body.MaxMemGb != nil {
		newMem = *body.MaxMemGb
	}
	for i := range units {
		units[i].MaxCpu = newCpu
		units[i].MaxMemGb = newMem
	}
	t.MaxCpu, t.MaxMemGb = newCpu, newMem
	t.Units = datatypes.JSON(jsonMarshal(units))
	if err := h.DB.Save(t).Error; err != nil {
		envelope.Internal(c, err)
		return
	}
	h.audit("tenant.resize", "ob_tenant", t.ID, map[string]interface{}{"maxCpu": newCpu, "maxMemGb": newMem})
	envelope.OK(c, t)
}

/* ================= 租户参数 ================= */

func (h *H) ListTenantParams(c *gin.Context) {
	var params []model.ClusterParam
	h.DB.Where("scope = ? AND tenant_id = ?", "tenant", c.Param("tid")).Order("id asc").Find(&params)
	envelope.OK(c, params)
}

func (h *H) UpdateTenantParam(c *gin.Context) {
	var body struct {
		Value string `json:"value" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		envelope.BadRequest(c, "value required")
		return
	}
	var p model.ClusterParam
	if err := h.DB.Where("scope = ? AND tenant_id = ? AND name = ?", "tenant", c.Param("tid"), c.Param("name")).First(&p).Error; err != nil {
		envelope.NotFound(c, "param not found")
		return
	}
	old := p.Value
	p.Value = body.Value
	p.Status = "pending"
	h.DB.Save(&p)
	h.DB.Create(&model.ParamHistory{ParamID: p.ID, OldValue: old, NewValue: p.Value, ChangedAt: time.Now().UnixMilli()})
	h.audit("param.update", "tenant_param", c.Param("tid")+"/"+p.Name, map[string]interface{}{"old": old, "new": p.Value})
	envelope.OK(c, p)
}

/* ================= 租户数据库 / 会话 / 慢SQL ================= */

func (h *H) ListTenantDatabases(c *gin.Context) {
	var dbs []model.ObTenantDb
	h.DB.Where("tenant_id = ?", c.Param("tid")).Order("id asc").Find(&dbs)
	envelope.OK(c, dbs)
}

func (h *H) CreateTenantDatabase(c *gin.Context) {
	var body struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		envelope.BadRequest(c, "name required")
		return
	}
	db := model.ObTenantDb{TenantID: c.Param("tid"), Name: body.Name, Tables: 0, Size: "0 GB", Conn: 0, Status: "ok"}
	if err := h.DB.Create(&db).Error; err != nil {
		envelope.BadRequest(c, "create failed (duplicate name?)")
		return
	}
	h.audit("database.create", "ob_tenant_db", c.Param("tid")+"/"+body.Name, nil)
	envelope.OK(c, db)
}
