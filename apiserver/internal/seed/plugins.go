package seed

import (
	"log"
	"time"

	"gorm.io/gorm"

	"db-cockpit/apiserver/internal/model"
)

/* ================= 插件域演示种子（D15 · 表空才导入，独立于主守卫） ================= */

func nowMs() int64 { return time.Now().UnixMilli() }

func pluginSeedRows() ([]*model.McpServerConfig, []*model.ToolDefinition) {
	now := nowMs()
	metrics := &model.McpServerConfig{ID: "mcp_metrics", Name: "metrics-mcp", Transport: "http",
		BaseURL: "http://metrics-mcp:8080/mcp", Version: 1, Status: "active", Enabled: true,
		Remark: "指标数据域 MCP Server（演示种子）", CreatedAt: now, UpdatedAt: now}
	alert := &model.McpServerConfig{ID: "mcp_alert", Name: "alert-mcp", Transport: "http",
		BaseURL: "http://alert-mcp:8080/mcp", Version: 1, Status: "active", Enabled: true,
		Remark: "告警数据域 MCP Server（演示种子）", CreatedAt: now, UpdatedAt: now}

	td := func(id, toolName, serverID, origin, status, category, risk, card string, dbTypes string) *model.ToolDefinition {
		return &model.ToolDefinition{ID: id, ToolName: toolName, ServerID: serverID, OriginToolName: origin,
			DisplayName: origin, Description: "演示种子：" + toolName + "（对齐工具注册表 §8 MVP 清单）",
			Category: category, DbTypes: J(dbTypes), RiskLevel: risk, AuditLevel: "summary",
			InputSchema: J(`{"type":"object","properties":{"instance_id":{"type":"string"},"range":{"type":"string"}},"required":["instance_id"]}`),
			OutputCard: card, ExecutionMode: "sync", TimeoutMs: 10000, Status: status,
			Version: 1, CreatedAt: now, UpdatedAt: now}
	}
	tools := []*model.ToolDefinition{
		td("td_seed_1", "metrics-mcp.get_metrics", "mcp_metrics", "get_metrics", "active", "metrics", "L0", "metric_chart", `["*"]`),
		td("td_seed_2", "metrics-mcp.get_replica_lag", "mcp_metrics", "get_replica_lag", "active", "metrics", "L0", "metric_chart", `["pg","oceanbase"]`),
		td("td_seed_3", "metrics-mcp.scan_metric_anomaly", "mcp_metrics", "scan_metric_anomaly", "active", "diagnosis", "L0", "", `["*"]`),
		td("td_seed_4", "alert-mcp.list_alerts", "mcp_alert", "list_alerts", "active", "alert", "L0", "data_table", `["*"]`),
		td("td_seed_5", "metrics-mcp.get_slow_sql_stats", "mcp_metrics", "get_slow_sql_stats", "draft", "slow_sql", "", "data_table", `["*"]`),
		td("td_seed_6", "alert-mcp.get_alert_stats", "mcp_alert", "get_alert_stats", "draft", "alert", "", "", `["*"]`),
		{ID: "td_seed_7", ToolName: "alert-mcp.push_alert", ServerID: "mcp_alert", OriginToolName: "push_alert",
			DisplayName: "push_alert", Description: "演示种子：已废弃的外推工具（保留供回放）",
			Category: "alert", RiskLevel: "L1", AuditLevel: "full", ExecutionMode: "sync",
			Status: "deprecated", Version: 1, CreatedAt: now, UpdatedAt: now},
	}
	return []*model.McpServerConfig{metrics, alert}, tools
}

// RunPlugins 插件域种子：mcp_server_configs / tool_definitions 各自表空才导入；
// config_versions 初始化 plugin_domain 版本。
func RunPlugins(gdb *gorm.DB) error {
	var n int64
	if err := gdb.Model(&model.McpServerConfig{}).Count(&n).Error; err != nil {
		return err
	}
	if n == 0 {
		servers, tools := pluginSeedRows()
		if err := gdb.Transaction(func(tx *gorm.DB) error {
			if err := tx.Create(&servers).Error; err != nil {
				return err
			}
			return tx.CreateInBatches(tools, 100).Error
		}); err != nil {
			return err
		}
		log.Printf("[seed] plugin domain imported (mcp_servers=2, tool_definitions=7)")
	}
	if err := gdb.Model(&model.ConfigVersion{}).Count(&n).Error; err != nil {
		return err
	}
	if n == 0 {
		if err := gdb.Create(&model.ConfigVersion{Scope: "plugin_domain", Version: 1, UpdatedAt: nowMs()}).Error; err != nil {
			return err
		}
	}
	return nil
}
