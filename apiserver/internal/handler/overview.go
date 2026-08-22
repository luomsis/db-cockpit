package handler

import (
	"github.com/gin-gonic/gin"

	"db-cockpit/apiserver/internal/envelope"
	"db-cockpit/apiserver/internal/model"
	"db-cockpit/apiserver/internal/version"
)

// GetOverview：概览页五卡片聚合。
// dbTypes 优先元数据域活计数（db_cluster/db_instance），空表回退 meta_stats 静态值；
// 告警/慢SQL 走数据面白名单聚合（whitelist.go），空表回退 UI 演示表。
func (h *H) GetOverview(c *gin.Context) {
	load := func(key string) interface{} {
		var row model.MetaStat
		if err := h.DB.Where("key = ?", key).First(&row).Error; err != nil {
			return nil
		}
		return row.Value
	}
	var dbTypes interface{}
	if live := h.dbTypes(); live != nil {
		dbTypes = live
	} else {
		dbTypes = load("db_types")
	}
	envelope.OK(c, gin.H{
		"dbTypes":    dbTypes,
		"topAnomaly": load("top_anomaly"),
		"sqlIssues":  load("sql_issues"),
		"lock":       load("lock_summary"),
		"slowSqls":   h.slowItems(nil),
		"alerts":     h.alertItems(),
	})
}

// GetVersion：apiserver 构建信息（git 短哈希 + 构建时间，供前端左下角展示）
func (h *H) GetVersion(c *gin.Context) {
	envelope.OK(c, version.Get())
}
