package server

import (
	"net/http"
	"net/http/httputil"
	"net/url"
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
	ch := &handler.ChatHandler{H: *h, RT: rt}

	r.GET("/healthz", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	api := r.Group("/api")
	{
		api.GET("/version", h.GetVersion)
		api.GET("/overview", h.GetOverview)
		api.GET("/alerts", h.GetAlerts)

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

		api.GET("/metrics", h.ListMetrics)
		api.GET("/dash/series", h.DashSeries)
		api.GET("/dash/annotations", h.DashAnnotations)
	}

	// chat：builtin 本地实现 / upstream 透明代理（docs §3.5，未来热换 Python）
	if cfg.AgentMode == "upstream" && cfg.AgentUpstreamURL != "" {
		mountChatProxy(r, cfg.AgentUpstreamURL)
	} else {
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
	}

	handler.RegisterInternal(r)
	return r
}

// mountChatProxy：upstream 模式把 /api/chat/* 整组反代到 Python agentcluster。
func mountChatProxy(r *gin.Engine, upstream string) {
	target, _ := url.Parse(upstream)
	proxy := httputil.NewSingleHostReverseProxy(target)
	r.Any("/api/chat/*path", func(c *gin.Context) {
		proxy.ServeHTTP(c.Writer, c.Request)
	})
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
