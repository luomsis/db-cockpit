package handler

import (
	"github.com/gin-gonic/gin"

	"db-cockpit/apiserver/internal/envelope"
)

/* ================= /internal/*：Go↔agent 边界（v2.0 依赖规则重构后仅存过渡通道） =================
 * 依赖规则：运行时唯一服务间调用 = Go→agent 的 exec 执行流（见 agent/upstream.go）；
 * 本组路由仅为 MVP 过渡保留：tools/data 在 MCP Server 就绪后退役；
 * wake 已随任务表契约（agent_tasks 轮询，见 agent/taskbus.go）退役删除。
 */

func RegisterInternal(r gin.IRouter) {
	g := r.Group("/internal")
	notImpl := func(c *gin.Context) { envelope.NotImplemented(c) }

	// 过渡通道：同步工具取数（MCP Server 逐工具承接后退役）
	g.POST("/tools/data", notImpl)
}
