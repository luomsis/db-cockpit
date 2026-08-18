package handler

import (
	"github.com/gin-gonic/gin"

	"db-cockpit/apiserver/internal/envelope"
	"db-cockpit/apiserver/internal/model"
	"db-cockpit/apiserver/internal/version"
)

// GetOverview：概览页五卡片聚合
func (h *H) GetOverview(c *gin.Context) {
	load := func(key string) interface{} {
		var row model.MetaStat
		if err := h.DB.Where("key = ?", key).First(&row).Error; err != nil {
			return nil
		}
		return row.Value
	}
	var slowSqls []model.SlowSql
	h.DB.Where("instance_id = ''").Order("id asc").Find(&slowSqls)
	var alerts []model.AlertRecord
	h.DB.Order("id asc").Find(&alerts)

	envelope.OK(c, gin.H{
		"dbTypes":    load("db_types"),
		"topAnomaly": load("top_anomaly"),
		"sqlIssues":  load("sql_issues"),
		"lock":       load("lock_summary"),
		"slowSqls":   slowSqls,
		"alerts":     alerts,
	})
}

// GetAlerts：顶栏铃铛 / 告警列表
func (h *H) GetAlerts(c *gin.Context) {
	var alerts []model.AlertRecord
	h.DB.Order("id asc").Find(&alerts)
	envelope.OK(c, gin.H{"items": alerts, "total": len(alerts)})
}

// GetVersion：apiserver 构建信息（git 短哈希 + 构建时间，供前端左下角展示）
func (h *H) GetVersion(c *gin.Context) {
	envelope.OK(c, version.Get())
}
