package agent

import (
	"fmt"
	"regexp"
	"time"

	"db-cockpit/apiserver/internal/model"
)

/* ================= builtin 事件源：内置场景脚本（演示 / 上游不可用回退） =================
 * 事件形态与 upstream 完全一致（六类事件），前端与落库链路无差别。
 */

type builtinSource struct{}

// NewBuiltinSource 返回内置场景执行源（AGENT_MODE=builtin，或 upstream 未配地址时的回退）。
func NewBuiltinSource() *builtinSource { return &builtinSource{} }

func (s *builtinSource) Run(t *turn) { pickScenario(t.input)(t) }

/* ---------- 场景脚本（移植 mockAgent.ts） ---------- */

var reGreeting = mustRegexp(`你好|hi|hello|帮助|你能`)
var reAlert = mustRegexp(`告警`)
var reDiag = mustRegexp(`诊断|变慢|排查|根因|为什么.*(慢|卡)|分析.*实例`)
var reDataqa = mustRegexp(`QPS|qps|指标|趋势|多少|统计|查询`)

func mustRegexp(expr string) *regexp.Regexp {
	return regexp.MustCompile(expr)
}

func (t *turn) alertCount() int {
	var n int64
	t.db.Model(&model.AlertRecord{}).Count(&n)
	return int(n)
}

func (t *turn) scenarioDiagnosis() {
	t.say("好的，我将对 prod-ob-core-01 的 trade_tenant 租户发起诊断，先采集指标与告警证据：\n\n", 24)
	t.thought("builtin_get_metrics", 800)
	if t.guard() {
		return
	}
	t.emitCard("create", metricChartCard("CPU 使用率（近 24h）", "cpu", 55, 14, "%", []chartThreshold{{Value: 85, Label: "告警阈值", Severity: "warning"}}))
	t.thought("builtin_list_alerts", 600)
	if t.guard() {
		return
	}
	t.say("指标显示 CPU 在 14:00 后异常飙升，继续采集会话快照与慢 SQL 证据：\n\n", 24)
	t.thought("builtin_session_snapshot", 900)
	if t.guard() {
		return
	}
	var sessions []model.RuntimeSession
	t.db.Order("id desc").Find(&sessions)
	t.emitCard("create", sessionsTableCard(sessions))
	t.sleep(300)

	taskID := fmt.Sprintf("task_%x", time.Now().UnixMilli())
	t.emitCard("create", taskProgressCard(taskID))
	t.say("\n\n初步证据已确认异常，已提交异步深度诊断任务（多指标长时窗扫描），完成后我将汇总交叉验证结论。", 24)
	t.sleep(400)

	stages := []string{"多指标长时窗扫描", "异常区间关联分析", "根因假设生成"}
	progressSeq := [][2]int{{15, 0}, {45, 0}, {70, 1}, {95, 2}}
	for _, p := range progressSeq {
		if t.guard() {
			return
		}
		t.emit(ProgressEvent{Type: "progress", TaskID: taskID, Progress: float64(p[0]), Stage: stages[p[1]] + "…"})
		t.sleep(900)
	}
	if t.guard() {
		return
	}
	t.emit(ProgressEvent{Type: "progress", TaskID: taskID, Progress: 100, Stage: "完成"})
	t.sleep(500)
	if t.guard() {
		return
	}
	t.emitCard("update", diagnosisReportCard())
	t.sleep(200)
	t.say("\n\n诊断报告已生成：根因是 trade_order.status 缺索引引发全表扫描（置信度 92%），锁等待为次生影响。可点击卡片查看完整证据链，或直接追问。", 24)
	t.emit(DoneEvent{Type: "done"})
}

func (t *turn) scenarioAlerts() {
	t.thought("builtin_list_alerts", 700)
	if t.guard() {
		return
	}
	var alerts []model.AlertRecord
	t.db.Order("id asc").Find(&alerts)
	t.emitCard("create", alertsTableCard(alerts))
	// 按级别统计，P1 取首条作为重点提示
	cnt := map[string]int{}
	var topP1 *model.AlertRecord
	for i := range alerts {
		cnt[alerts[i].Severity]++
		if alerts[i].Severity == "P1" && topP1 == nil {
			topP1 = &alerts[i]
		}
	}
	var text string
	if topP1 != nil {
		text = fmt.Sprintf("当前共 %d 条活跃告警（P1 %d / P2 %d / P3 %d）。最高优先级：%s — %s。\n\n建议优先处理 P1，可直接对我说「诊断 %s」。",
			len(alerts), cnt["P1"], cnt["P2"], cnt["P3"], topP1.Name, topP1.Title, topP1.Name)
	} else {
		text = fmt.Sprintf("当前共 %d 条活跃告警（P1 0 / P2 %d / P3 %d），无 P1 级紧急告警。", len(alerts), cnt["P2"], cnt["P3"])
	}
	t.say(text, 24)
	t.emit(DoneEvent{Type: "done"})
}

func (t *turn) scenarioDataqa() {
	t.thought("builtin_get_metrics", 700)
	if t.guard() {
		return
	}
	t.emitCard("create", metricChartCard("全局 QPS 趋势（近 24h）", "qps", 18500, 4200, "", nil))
	t.say("近 24h 全局 QPS 均值约 18.5k，14:00 起出现约 35% 的流量高峰并伴随慢 SQL 增多。需要看集群维度对比或切时间窗，可以直接告诉我。", 24)
	t.emit(DoneEvent{Type: "done"})
}

func (t *turn) scenarioGreeting() {
	t.say("你好！我是 DB Cockpit 智能运维助手 🤖\n\n我可以帮你：\n• 诊断实例性能异常（指标 → 会话 → 慢 SQL → 深度扫描全链路）\n• 查询平台运维数据（QPS / 告警 / 慢 SQL 统计）\n• 分析锁等待与长事务根因\n\n试着问我：「当前有哪些告警实例？」或「诊断 trade_tenant」", 20)
	t.emit(DoneEvent{Type: "done"})
}

func (t *turn) scenarioFallback() {
	t.say("已收到问题。当前为前端演示模式（数据为 mock），我支持的演示场景：\n\n1. 「当前有哪些告警实例」— 告警问数\n2. 「诊断 trade_tenant」— 完整诊断链路（含异步深度扫描）\n3. 「QPS 趋势」— 指标问数\n\n正式版将接入 agent 集群（路由 / 诊断 / 问数专家），通过工具注册表调用真实数据。", 20)
	t.emit(DoneEvent{Type: "done"})
}

func pickScenario(text string) func(t *turn) {
	if reGreeting.MatchString(text) {
		return (*turn).scenarioGreeting
	}
	if reAlert.MatchString(text) && !reDiag.MatchString(text) {
		return (*turn).scenarioAlerts
	}
	if reDiag.MatchString(text) {
		return (*turn).scenarioDiagnosis
	}
	if reDataqa.MatchString(text) {
		return (*turn).scenarioDataqa
	}
	return (*turn).scenarioFallback
}
