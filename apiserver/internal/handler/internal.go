package handler

import (
	"github.com/gin-gonic/gin"

	"db-cockpit/apiserver/internal/envelope"
)

/* ================= /internal/*：Go↔Python 边界契约（MVP 只注册不实现） =================
 * 路由清单对齐 docs《架构设计文档》§3.5 与《Agent执行框架》§15；
 * Python agentcluster 就绪后在此挂真实实现。
 */

func RegisterInternal(r gin.IRouter) {
	g := r.Group("/internal")
	notImpl := func(c *gin.Context) { envelope.NotImplemented(c) }

	g.POST("/tools/data", notImpl)

	g.GET("/sessions", notImpl)
	g.POST("/sessions", notImpl)
	g.GET("/sessions/:id/turns", notImpl)
	g.GET("/turns/:id/trace", notImpl)
	g.GET("/sessions/:id/checkpoints", notImpl)

	g.POST("/tasks", notImpl)
	g.GET("/tasks/:id", notImpl)

	r.POST("/agentcluster/wake", notImpl)
}
