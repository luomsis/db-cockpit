package server

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"db-cockpit/apiserver/internal/agent"
	"db-cockpit/apiserver/internal/config"
	"db-cockpit/apiserver/internal/envelope"
	"db-cockpit/apiserver/internal/handler"
)

func New(cfg config.Config, gdb *gorm.DB, rt *agent.Runtime) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery(), cors(), authStub(cfg))

	h := &handler.H{DB: gdb}
	// chat_turns.config_version 快照：标识本轮事件源（配置中心接入后细化为真实配置版本）
	agentCV := "builtin-v0"
	if cfg.AgentMode == "upstream" {
		agentCV = "upstream-v0"
	}
	ch := &handler.ChatHandler{H: *h, RT: rt, ConfigVersion: agentCV}

	r.GET("/healthz", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	api := r.Group("/api")
	{
		api.GET("/version", h.GetVersion)
		api.GET("/overview", h.GetOverview)
		api.GET("/alerts", h.GetAlerts)
		api.GET("/changes", h.ListChanges) // 数据面白名单：变更工单（诊断时间窗关联）

		// 元数据域下钻（§6.1.1：集群 → 实例 → 节点）
		meta := api.Group("/meta")
		{
			meta.GET("/clusters", h.ListMetaClusters)
			meta.GET("/clusters/:id", h.GetMetaCluster)
		}

		cl := api.Group("/clusters")
		{
			cl.GET("", h.ListClusters)
			cl.GET("/filter-options", h.ClusterFilterOptions)
			cl.GET("/:id", h.GetCluster)
			cl.GET("/:id/params", h.ListClusterParams)
			cl.PUT("/:id/params/:name", h.UpdateClusterParam)
			cl.GET("/:id/params/:name/history", h.ClusterParamHistory)
			cl.GET("/:id/databases", h.ListPgDatabases)
			cl.POST("/:id/databases", h.CreatePgDatabase)
			cl.GET("/:id/replicas", h.ListPgReplicas)
			cl.POST("/:id/replicas/:inst/switch-drill", h.SwitchDrill)
			cl.POST("/:id/replicas/:inst/rebuild", h.RebuildReplica)
			cl.GET("/:id/tenants", h.ListObTenants)
			cl.POST("/:id/tenants", h.CreateObTenant)
			cl.GET("/:id/reports", h.ListReports)
			cl.GET("/:id/reports/:rid/download", h.DownloadReport)
			cl.GET("/:id/instances/:iid", h.GetInstance)
			cl.GET("/:id/instances/:iid/users", h.ListInstanceUsers)
			cl.POST("/:id/instances/:iid/users", h.CreateInstanceUser)
			cl.POST("/:id/instances/:iid/users/:user/grant", h.GrantUser)
			cl.POST("/:id/instances/:iid/users/:user/reset-password", h.ResetUserPassword)
			cl.POST("/:id/instances/:iid/users/:user/lock", h.LockUser)
			cl.GET("/:id/instances/:iid/sessions", h.ListSessions)
			cl.POST("/:id/instances/:iid/sessions/:sid/kill", h.KillSession)
			cl.GET("/:id/instances/:iid/transactions", h.ListTransactions)
			cl.GET("/:id/instances/:iid/slow-sqls", h.ListSlowSqls)
		}

		api.GET("/tenants/:tid", h.GetTenant)
		api.POST("/tenants/:tid/resize", h.ResizeTenant)
		api.GET("/tenants/:tid/params", h.ListTenantParams)
		api.PUT("/tenants/:tid/params/:name", h.UpdateTenantParam)
		api.GET("/tenants/:tid/databases", h.ListTenantDatabases)
		api.POST("/tenants/:tid/databases", h.CreateTenantDatabase)
		api.GET("/tenants/:tid/sessions", h.ListSessions)
		api.POST("/tenants/:tid/sessions/:sid/kill", h.KillSession)
		api.GET("/tenants/:tid/slow-sqls", h.ListSlowSqls)

		api.GET("/hosts", h.ListHosts)

		api.POST("/diagnosis/sql", h.DiagnoseSql)

		dbGroup := api.Group("/dashboards")
		{
			dbGroup.GET("", h.ListDashboards)
			dbGroup.POST("", h.CreateDashboard)
			dbGroup.POST("/import", h.ImportDashboards)
			dbGroup.GET("/:id", h.GetDashboard)
			dbGroup.PUT("/:id", h.UpdateDashboard)
			dbGroup.DELETE("/:id", h.DeleteDashboard)
		}

		// 设置中心：大模型 / 嵌入模型服务配置（含测试连接）
		mc := api.Group("/model-configs")
		{
			mc.GET("", h.ListModelConfigs)
			mc.POST("", h.CreateModelConfig)
			mc.PUT("/:id", h.UpdateModelConfig)
			mc.DELETE("/:id", h.DeleteModelConfig)
			mc.POST("/:id/test", h.TestModelConfig)
		}
		ec := api.Group("/embedding-configs")
		{
			ec.GET("", h.ListEmbeddingConfigs)
			ec.POST("", h.CreateEmbeddingConfig)
			ec.PUT("/:id", h.UpdateEmbeddingConfig)
			ec.DELETE("/:id", h.DeleteEmbeddingConfig)
			ec.POST("/:id/test", h.TestEmbeddingConfig)
		}

		// 插件中心：MCP 服务 / Skills / 工具注册表（插件域 D15）
		mp := api.Group("/mcp-servers")
		{
			mp.GET("", h.ListMcpServers)
			mp.POST("", h.CreateMcpServer)
			mp.PUT("/:id", h.UpdateMcpServer)
			mp.DELETE("/:id", h.DeleteMcpServer)
			mp.POST("/:id/discover", h.DiscoverMcpServer) // 触发式 tools/list → draft 草案
			mp.POST("/:id/health", h.HealthMcpServer)     // 连通性 → 工具 health 标记
		}
		td := api.Group("/tool-definitions")
		{
			td.GET("", h.ListToolDefinitions)
			td.PUT("/:id", h.UpdateToolDefinition)
		}
		sk := api.Group("/skills")
		{
			sk.GET("", h.ListSkills)
			sk.POST("", h.CreateSkill)
			sk.PUT("/:id", h.UpdateSkill)
			sk.DELETE("/:id", h.DeleteSkill)
		}

		// 管理面：动态 subagent / workflow 定义（agent 运行时直读 PG，依赖规则②）
		sa := api.Group("/subagents")
		{
			sa.GET("", h.ListSubagentDefs)
			sa.POST("", h.CreateSubagentDef)
			sa.PUT("/:id", h.UpdateSubagentDef)
			sa.DELETE("/:id", h.DeleteSubagentDef)
		}
		wf := api.Group("/workflows")
		{
			wf.GET("", h.ListWorkflowDefs)
			wf.POST("", h.CreateWorkflowDef)
			wf.PUT("/:id", h.UpdateWorkflowDef)
			wf.DELETE("/:id", h.DeleteWorkflowDef)
		}

		api.GET("/metrics", h.ListMetrics)
		api.GET("/dash/series", h.DashSeries)
		api.GET("/dash/annotations", h.DashAnnotations)
	}

	// chat：Go 始终终结 SSE（v1.2 起退役整组反代；AGENT_MODE 只切换事件源，见 main.go 装配）
	chat := api.Group("/chat")
	{
		chat.GET("/sessions", ch.ListSessions)
		chat.POST("/sessions", ch.CreateSession)
		chat.POST("/sessions/import", ch.ImportSessions)
		chat.GET("/sessions/:id", ch.GetSession)
		chat.DELETE("/sessions/:id", ch.DeleteSession)
		chat.POST("/sessions/:id/turns", ch.SubmitTurn)
		chat.GET("/sessions/:id/stream", ch.Stream)
		chat.POST("/sessions/:id/turns/:tid/cancel", ch.CancelTurn)
	}

	handler.RegisterInternal(r)
	return r
}

func cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, Last-Event-ID")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// authStub：MVP 免登录；AUTH_ENABLED=true 时要求 Bearer 头（占位校验，接 OIDC 时替换）。
func authStub(cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !cfg.AuthEnabled || !strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.Next()
			return
		}
		auth := c.GetHeader("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") || len(auth) <= len("Bearer ") {
			envelope.Fail(c, http.StatusUnauthorized, http.StatusUnauthorized, "missing bearer token")
			return
		}
		c.Next()
	}
}
