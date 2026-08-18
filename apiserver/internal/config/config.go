package config

import "os"

type Config struct {
	Port             string
	DBDSN            string
	AuthEnabled      bool
	AgentMode        string // builtin | upstream
	AgentUpstreamURL string
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
	}
}
