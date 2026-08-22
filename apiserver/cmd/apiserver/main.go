package main

import (
	"context"
	"log"
	"strings"

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
	agent.RecoverInterruptedTurns(gdb)

	// 事件源装配：AGENT_MODE 只选择 builtin / upstream 事件源，Go 始终终结 SSE（docs §3.5）
	var source agent.TurnSource = agent.NewBuiltinSource()
	switch {
	case cfg.AgentMode == "upstream" && cfg.AgentExecURL != "":
		source = agent.NewUpstreamSource(cfg.AgentExecURL, 0)
		log.Printf("[apiserver] agent.mode=upstream exec=%s", cfg.AgentExecURL)
	case cfg.AgentMode == "upstream" && cfg.AgentUpstreamURL != "":
		execURL := strings.TrimRight(cfg.AgentUpstreamURL, "/") + "/internal/exec/turns"
		source = agent.NewUpstreamSource(execURL, 0)
		log.Printf("[apiserver] agent.mode=upstream exec=%s", execURL)
	case cfg.AgentMode == "upstream":
		log.Printf("[apiserver] AGENT_MODE=upstream 但未配置 AGENT_EXEC_URL/AGENT_UPSTREAM_URL，回退 builtin")
	}
	rt := agent.NewRuntime(gdb, source)
	engine := server.New(cfg, gdb, rt)

	// 任务表契约轮询器：agent_tasks 进度注入 + 完成续跑（依赖规则②，取代 tasks API 与 wake）
	go agent.NewTaskBus(gdb, rt).Run(context.Background())

	log.Printf("[apiserver] listening on :%s (agent.mode=%s)", cfg.Port, cfg.AgentMode)
	if err := engine.Run(":" + cfg.Port); err != nil {
		log.Fatalf("[apiserver] serve: %v", err)
	}
}
