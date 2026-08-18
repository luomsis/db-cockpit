package main

import (
	"log"

	"db-cockpit/apiserver/internal/agent"
	"db-cockpit/apiserver/internal/config"
	"db-cockpit/apiserver/internal/db"
	"db-cockpit/apiserver/internal/seed"
	"db-cockpit/apiserver/internal/server"
)

func main() {
	cfg := config.Load()

	gdb, err := db.Open(cfg.DBDSN)
	if err != nil {
		log.Fatalf("[apiserver] init db: %v", err)
	}
	if err := seed.Run(gdb); err != nil {
		log.Fatalf("[apiserver] seed: %v", err)
	}

	rt := agent.NewRuntime(gdb)
	engine := server.New(cfg, gdb, rt)

	log.Printf("[apiserver] listening on :%s (agent.mode=%s)", cfg.Port, cfg.AgentMode)
	if err := engine.Run(":" + cfg.Port); err != nil {
		log.Fatalf("[apiserver] serve: %v", err)
	}
}
