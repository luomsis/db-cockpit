package config

import "os"

type Config struct {
	Port             string
	DBDSN            string
	AuthEnabled      bool
	AgentMode        string // builtin | upstream（事件源选择，非路由切换：Go 始终终结 SSE）
	AgentUpstreamURL string
	AgentExecURL     string // 完整 exec 端点；缺省由 AgentUpstreamURL + /internal/exec/turns 推导
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func Load() Config {
	return Config{
		Port:             env("APISERVER_PORT", "8090"),
		DBDSN:            env("DB_DSN", "host=localhost port=55432 user=graphiti password=graphiti dbname=db_cockpit sslmode=disable"),
		AuthEnabled:      env("AUTH_ENABLED", "false") == "true",
		AgentMode:        env("AGENT_MODE", "builtin"),
		AgentUpstreamURL: env("AGENT_UPSTREAM_URL", ""),
		AgentExecURL:     env("AGENT_EXEC_URL", ""),
	}
}
