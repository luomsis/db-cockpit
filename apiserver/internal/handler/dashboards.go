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

/* ================= 监控大盘 CRUD（cfg/panels 原样 JSONB） ================= */

func (h *H) ListDashboards(c *gin.Context) {
	var dashboards []model.Dashboard
	h.DB.Order("created_at asc").Find(&dashboards)
	if dashboards == nil {
		dashboards = []model.Dashboard{}
	}
	envelope.OK(c, dashboards)
}

type dashboardBody struct {
	ID          string          `json:"id"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Cfg         json.RawMessage `json:"cfg"`
	Panels      json.RawMessage `json:"panels"`
	CreatedAt   int64           `json:"createdAt"`
	UpdatedAt   int64           `json:"updatedAt"`
}

func (h *H) CreateDashboard(c *gin.Context) {
	var body dashboardBody
	if err := c.ShouldBindJSON(&body); err != nil || body.Title == "" {
		envelope.BadRequest(c, "title required")
		return
	}
	now := time.Now().UnixMilli()
	panels := body.Panels
	if len(panels) == 0 {
		panels = json.RawMessage("[]")
	}
	d := model.Dashboard{
		ID: agent.NewID("d"), Title: body.Title, Description: body.Description,
		Cfg: datatypes.JSON(body.Cfg), Panels: datatypes.JSON(panels),
		CreatedAt: now, UpdatedAt: now,
	}
	if err := h.DB.Create(&d).Error; err != nil {
		envelope.Internal(c, err)
		return
	}
	envelope.OK(c, d)
}

func (h *H) GetDashboard(c *gin.Context) {
	var d model.Dashboard
	if err := h.DB.Where("id = ?", c.Param("id")).First(&d).Error; err != nil {
		envelope.NotFound(c, "dashboard not found")
		return
	}
	envelope.OK(c, d)
}

func (h *H) UpdateDashboard(c *gin.Context) {
	var d model.Dashboard
	if err := h.DB.Where("id = ?", c.Param("id")).First(&d).Error; err != nil {
		envelope.NotFound(c, "dashboard not found")
		return
	}
	var body dashboardBody
	if err := c.ShouldBindJSON(&body); err != nil {
		envelope.BadRequest(c, "invalid body")
		return
	}
	if body.Title != "" {
		d.Title = body.Title
	}
	d.Description = body.Description
	if len(body.Cfg) > 0 {
		d.Cfg = datatypes.JSON(body.Cfg)
	}
	if len(body.Panels) > 0 {
		d.Panels = datatypes.JSON(body.Panels)
	}
	d.UpdatedAt = time.Now().UnixMilli()
	if err := h.DB.Save(&d).Error; err != nil {
		envelope.Internal(c, err)
		return
	}
	envelope.OK(c, d)
}

func (h *H) DeleteDashboard(c *gin.Context) {
	res := h.DB.Where("id = ?", c.Param("id")).Delete(&model.Dashboard{})
	if res.Error != nil || res.RowsAffected == 0 {
		envelope.NotFound(c, "dashboard not found")
		return
	}
	envelope.OK(c, gin.H{"ok": true})
}

// ImportDashboards：前端 localStorage 一次性迁移
func (h *H) ImportDashboards(c *gin.Context) {
	var body struct {
		Dashboards []dashboardBody `json:"dashboards"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		envelope.BadRequest(c, "invalid body")
		return
	}
	imported := 0
	for _, in := range body.Dashboards {
		if in.ID == "" {
			continue
		}
		panels := in.Panels
		if len(panels) == 0 {
			panels = json.RawMessage("[]")
		}
		createdAt := in.CreatedAt
		if createdAt == 0 {
			createdAt = time.Now().UnixMilli()
		}
		d := model.Dashboard{
			ID: in.ID, Title: in.Title, Description: in.Description,
			Cfg: datatypes.JSON(in.Cfg), Panels: datatypes.JSON(panels),
			CreatedAt: createdAt, UpdatedAt: time.Now().UnixMilli(),
		}
		// upsert：保留原 id 与时间戳
		if err := h.DB.Save(&d).Error; err != nil {
			continue
		}
		imported++
	}
	envelope.OK(c, gin.H{"imported": imported})
}
