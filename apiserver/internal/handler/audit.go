package handler

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"db-cockpit/apiserver/internal/model"
)

// H：所有 handler 共享的依赖
type H struct {
	DB *gorm.DB
}

func jsonMarshal(v interface{}) []byte {
	b, _ := json.Marshal(v)
	return b
}

// pageParams：分页参数（默认 1 / 20，pageSize 上限 100）
func pageParams(c *gin.Context) (page, pageSize int) {
	page, _ = strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	pageSize, _ = strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

// audit：危险/写操作审计
func (h *H) audit(action, targetType, targetID string, detail map[string]interface{}) {
	raw, _ := json.Marshal(detail)
	_ = h.DB.Create(&model.AuditLog{
		Ts: time.Now().UnixMilli(), Actor: "admin",
		Action: action, TargetType: targetType, TargetID: targetID,
		Detail: raw,
	}).Error
}
