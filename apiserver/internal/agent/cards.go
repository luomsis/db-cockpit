package agent

import (
	"fmt"

	"gorm.io/datatypes"

	"db-cockpit/apiserver/internal/metrics"
	"db-cockpit/apiserver/internal/model"
)

/* ================= card-protocol/1.0 卡片封装（对齐前端 types.ts CardEnvelope） ================= */

type CardSource struct {
	SessionID  string  `json:"session_id,omitempty"`
	TurnID     string  `json:"turn_id,omitempty"`
	Agent      string  `json:"agent,omitempty"`
	ToolCallID *string `json:"tool_call_id"`
}

type CardTimeRange struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

type CardContext struct {
	InstanceID interface{}    `json:"instance_id,omitempty"`
	ClusterID  interface{}    `json:"cluster_id,omitempty"`
	TimeRange  *CardTimeRange `json:"time_range,omitempty"`
}

type CardInteraction struct {
	ID      string                 `json:"id"`
	Label   string                 `json:"label"`
	Kind    string                 `json:"kind"`
	Payload map[string]interface{} `json:"payload,omitempty"`
}

type CardEnvelope struct {
	CardID          string            `json:"card_id"`
	CardType        string            `json:"card_type"`
	ProtocolVersion string            `json:"protocol_version"`
	Title           string            `json:"title"`
	Status          string            `json:"status"`
	Source          *CardSource       `json:"source,omitempty"`
	Context         *CardContext      `json:"context"`
	Payload         interface{}       `json:"payload"`
	Interactions    []CardInteraction `json:"interactions,omitempty"`
	FallbackText    string            `json:"fallback_text"`
}

var cardSeq int64

func newCardID() string {
	cardSeq++
	return fmt.Sprintf("card_%x_%x", nowMillis(), cardSeq)
}

/* ---------- metric_chart ---------- */

type chartMetric struct {
	Name  string `json:"name"`
	Label string `json:"label,omitempty"`
	Unit  string `json:"unit,omitempty"`
}

type chartThreshold struct {
	Value    float64 `json:"value"`
	Label    string  `json:"label,omitempty"`
	Severity string  `json:"severity,omitempty"`
}

type metricChartPayload struct {
	ChartType  string          `json:"chart_type"`
	Metrics    []chartMetric   `json:"metrics"`
	Data       chartData       `json:"data"`
	Thresholds []chartThreshold `json:"thresholds"`
}

type chartData struct {
	Points [][2]interface{} `json:"points"`
}

func metricChartCard(title, metric string, base, jitter float64, unit string, thresholds []chartThreshold) CardEnvelope {
	data := metrics.GenSeries(base, jitter, 14, 20260816)
	points := make([][2]interface{}, 0, len(data))
	for i, v := range data {
		points = append(points, [2]interface{}{hourLabel(i), v})
	}
	avg := 0.0
	for _, v := range data {
		avg += v
	}
	avg = avg / float64(len(data))
	return CardEnvelope{
		CardID: newCardID(), CardType: "metric_chart", ProtocolVersion: "1.0",
		Title: title, Status: "final",
		Source: &CardSource{Agent: "diagnosis_expert", ToolCallID: nil},
		Context: &CardContext{},
		Payload: metricChartPayload{
			ChartType: "line",
			Metrics:   []chartMetric{{Name: metric, Label: title, Unit: unit}},
			Data:      chartData{Points: points},
			Thresholds: thresholds,
		},
		FallbackText: fmt.Sprintf("%s：近 24h 均值 %d%s", title, int64(avg+0.5), unit),
	}
}

func hourLabel(i int) string {
	s := fmt.Sprintf("%02d", i)
	return s + ":00"
}

/* ---------- data_table ---------- */

type dtColumn struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Type  string `json:"type,omitempty"`
}

type dataTablePayload struct {
	Columns    []dtColumn            `json:"columns"`
	Rows       []map[string]interface{} `json:"rows"`
	Total      int                   `json:"total"`
	RowActions []string              `json:"row_actions"`
}

func sessionsTableCard(sessions []model.RuntimeSession) CardEnvelope {
	rows := make([]map[string]interface{}, 0, len(sessions))
	for _, s := range sessions {
		rows = append(rows, map[string]interface{}{
			"id": s.SessionID, "user": s.Username, "db": s.Db,
			"time": s.Time, "state": s.State, "lock": s.LockInfo,
		})
	}
	abn := 0
	for _, s := range sessions {
		if s.Status != "ok" {
			abn++
		}
	}
	return CardEnvelope{
		CardID: newCardID(), CardType: "data_table", ProtocolVersion: "1.0",
		Title: "租户会话快照（trade_tenant @ prod-ob-core-01）", Status: "final",
		Source: &CardSource{Agent: "diagnosis_expert", ToolCallID: nil},
		Context: &CardContext{},
		Payload: dataTablePayload{
			Columns: []dtColumn{
				{Key: "id", Label: "会话 ID", Type: "number"},
				{Key: "user", Label: "用户"},
				{Key: "db", Label: "库"},
				{Key: "time", Label: "时长", Type: "duration"},
				{Key: "state", Label: "状态"},
				{Key: "lock", Label: "锁信息"},
			},
			Rows: rows, Total: len(rows), RowActions: []string{"ask"},
		},
		FallbackText: fmt.Sprintf("会话快照：共 %d 个会话，%d 个异常（行锁等待 / 长事务）", len(rows), abn),
	}
}

func alertsTableCard(alerts []model.AlertRecord) CardEnvelope {
	rows := make([]map[string]interface{}, 0, len(alerts))
	for _, a := range alerts {
		rows = append(rows, map[string]interface{}{
			"name": a.Name, "severity": a.Severity, "title": a.Title, "time": a.Time, "count": a.Count,
		})
	}
	top := "P3"
	if len(alerts) > 0 {
		top = alerts[0].Severity
		for _, a := range alerts {
			if a.Severity < top {
				top = a.Severity
			}
		}
	}
	return CardEnvelope{
		CardID: newCardID(), CardType: "data_table", ProtocolVersion: "1.0",
		Title: "当前告警实例（近 24h）", Status: "final",
		Source: &CardSource{Agent: "router", ToolCallID: nil},
		Context: &CardContext{},
		Payload: dataTablePayload{
			Columns: []dtColumn{
				{Key: "name", Label: "实例", Type: "string"},
				{Key: "severity", Label: "级别", Type: "status"},
				{Key: "title", Label: "告警内容"},
				{Key: "time", Label: "首次触发", Type: "time"},
				{Key: "count", Label: "次数", Type: "number"},
			},
			Rows: rows, Total: len(rows), RowActions: []string{"ask"},
		},
		FallbackText: fmt.Sprintf("当前 %d 个实例处于告警状态，最高 %s", len(rows), top),
	}
}

/* ---------- task_progress ---------- */

type progressStage struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

type taskProgressPayload struct {
	TaskID       string          `json:"task_id"`
	TaskType     string          `json:"task_type"`
	Status       string          `json:"status"`
	Stages       []progressStage `json:"stages"`
	Cancellable  bool            `json:"cancellable"`
}

func taskProgressCard(taskID string) CardEnvelope {
	return CardEnvelope{
		CardID: newCardID(), CardType: "task_progress", ProtocolVersion: "1.0",
		Title: "深度诊断任务", Status: "streaming",
		Source: &CardSource{Agent: "task_bus"},
		Context: &CardContext{},
		Payload: taskProgressPayload{
			TaskID: taskID, TaskType: "builtin_metric_deep_scan", Status: "pending",
			Stages: []progressStage{
				{Name: "多指标长时窗扫描", Status: "pending"},
				{Name: "异常区间关联分析", Status: "pending"},
				{Name: "根因假设生成", Status: "pending"},
			},
			Cancellable: true,
		},
		FallbackText: "深度诊断任务已提交",
	}
}

/* ---------- diagnosis_report ---------- */

type rootCause struct {
	Hypothesis   string   `json:"hypothesis"`
	Confidence   float64  `json:"confidence"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
}

type finding struct {
	Category string `json:"category"`
	Detail   string `json:"detail"`
}

type diagnosisPayload struct {
	Summary     string      `json:"summary"`
	Severity    string      `json:"severity"`
	RootCauses  []rootCause `json:"root_causes"`
	Findings    []finding   `json:"findings"`
	Suggestions []string    `json:"suggestions"`
	Provider    string      `json:"provider"`
}

func diagnosisReportCard() CardEnvelope {
	return CardEnvelope{
		CardID: newCardID(), CardType: "diagnosis_report", ProtocolVersion: "1.0",
		Title: "租户 trade_tenant @ prod-ob-core-01 性能诊断报告", Status: "final",
		Source: &CardSource{Agent: "diagnosis_expert", ToolCallID: nil},
		Context: &CardContext{
			InstanceID: "trade_tenant", ClusterID: "prod-ob-core-01",
			TimeRange: &CardTimeRange{Start: "2026-08-16T13:00:00+08:00", End: "2026-08-16T14:30:00+08:00"},
		},
		Payload: diagnosisPayload{
			Summary:  "CPU 飙升由全表扫描型慢 SQL 放大引发，锁等待为次生影响",
			Severity: "critical",
			RootCauses: []rootCause{
				{Hypothesis: "trade_order.status 无索引导致全表扫描，QPS 高峰放大 CPU", Confidence: 0.92},
				{Hypothesis: "长事务 TRX-998231 持锁未提交引发会话堆积", Confidence: 0.74},
			},
			Findings: []finding{
				{Category: "metric_anomaly", Detail: "CPU 使用率 14:00 后从 55% 升至 98%，与 QPS 激增同步"},
				{Category: "slow_sql", Detail: "TOP1 慢 SQL 扫描 438 万行，日均执行 342 次"},
				{Category: "lock", Detail: "会话 88231 持有 stock_record 行锁 1m28s，阻塞 3 个会话"},
				{Category: "session", Detail: "活跃会话 512（阈值 300），堆积持续 40 分钟"},
			},
			Suggestions: []string{
				"添加联合索引 idx_trade_order_status_uid (status, uid)，预计扫描行数下降 92%",
				"联系 app_rw 业务方确认事务边界，避免长事务持锁",
				"如影响持续扩大，可在会话管理中 Kill 会话 88231 止血",
			},
			Provider: "builtin",
		},
		FallbackText: "诊断结论：全表扫描慢 SQL 引发 CPU 飙升，锁等待为次生影响。建议添加联合索引。",
	}
}

func jraw(v interface{}) datatypes.JSON {
	b, _ := jsonMarshal(v)
	return datatypes.JSON(b)
}
