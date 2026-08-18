package agent

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"

	"db-cockpit/apiserver/internal/model"
)

func nowMillis() int64 { return time.Now().UnixMilli() }

func jsonMarshal(v interface{}) ([]byte, error) { return json.Marshal(v) }

// NewID 生成短随机 ID（sess_xxx / msg_xxx / turn_xxx）
func NewID(prefix string) string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%s_%x", prefix, b) + fmt.Sprintf("%x", time.Now().UnixNano()%100000)
}

/* ================= SSE 事件类型（六事件，对齐执行框架 §9 / 前端 AgentEvent） ================= */

type ThoughtEvent struct {
	Type     string `json:"type"`
	Step     int    `json:"step"`
	ToolName string `json:"tool_name"`
	Status   string `json:"status"`
}

type TokenEvent struct {
	Type      string `json:"type"`
	TextDelta string `json:"text_delta"`
}

type CardEvent struct {
	Type string       `json:"type"`
	Card CardEnvelope `json:"card"`
	Mode string       `json:"mode"`
}

type ProgressEvent struct {
	Type     string  `json:"type"`
	TaskID   string  `json:"task_id"`
	Progress float64 `json:"progress"`
	Stage    string  `json:"stage"`
}

type DoneEvent struct {
	Type string `json:"type"`
}

type ErrorEvent struct {
	Type    string `json:"type"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

/* ================= 会话运行时：事件发布 / 订阅 / 取消 ================= */

type sessionRT struct {
	mu      sync.Mutex
	seq     int64          // 已发布的最后一个事件序号
	events  [][]byte       // 本次进程生命周期内的事件（json）
	subs    map[chan eventMsg]struct{}
	cancelC chan struct{} // 当前 turn 的取消信号
}

type eventMsg struct {
	Seq int64
	Raw []byte
}

func (rt *sessionRT) subscribe() chan eventMsg {
	ch := make(chan eventMsg, 512)
	rt.mu.Lock()
	rt.subs[ch] = struct{}{}
	rt.mu.Unlock()
	return ch
}

func (rt *sessionRT) unsubscribe(ch chan eventMsg) {
	rt.mu.Lock()
	delete(rt.subs, ch)
	rt.mu.Unlock()
}

// Runtime 管理全部会话的事件总线；事件同时落库（chat_turn_events）以支持断线重放。
type Runtime struct {
	mu       sync.Mutex
	sessions map[string]*sessionRT
	db       *gorm.DB
}

func NewRuntime(gdb *gorm.DB) *Runtime {
	return &Runtime{sessions: map[string]*sessionRT{}, db: gdb}
}

func (r *Runtime) rt(sessionID string) *sessionRT {
	r.mu.Lock()
	defer r.mu.Unlock()
	if rt, ok := r.sessions[sessionID]; ok {
		return rt
	}
	// seq 从库里最大值继续，保证重启后不冲突
	var maxSeq int64
	r.db.Model(&model.ChatTurnEvent{}).Where("session_id = ?", sessionID).
		Select("COALESCE(MAX(seq), 0)").Scan(&maxSeq)
	rt := &sessionRT{seq: maxSeq, subs: map[chan eventMsg]struct{}{}}
	if rt.seq > 0 {
		// 进程刚启动：把历史事件载入缓冲，供 stream 重放
		var evs []model.ChatTurnEvent
		r.db.Where("session_id = ?", sessionID).Order("seq asc").Limit(2000).Find(&evs)
		for _, e := range evs {
			rt.events = append(rt.events, e.Event)
		}
	}
	r.sessions[sessionID] = rt
	return rt
}

// Subscribe / Unsubscribe / Snapshot：供 SSE handler 使用。
func (r *Runtime) Subscribe(sessionID string) chan eventMsg { return r.rt(sessionID).subscribe() }
func (r *Runtime) Unsubscribe(sessionID string, ch chan eventMsg) {
	r.rt(sessionID).unsubscribe(ch)
}

// Drop：会话删除时清理内存态。
func (r *Runtime) Drop(sessionID string) {
	r.mu.Lock()
	delete(r.sessions, sessionID)
	r.mu.Unlock()
}

// Snapshot 返回 seq > after 的历史事件及其序号。
func (r *Runtime) Snapshot(sessionID string, after int64) (msgs []eventMsg, lastSeq int64) {
	rt := r.rt(sessionID)
	rt.mu.Lock()
	defer rt.mu.Unlock()
	for i, raw := range rt.events {
		seq := rt.seq - int64(len(rt.events)) + int64(i) + 1
		if seq > after {
			msgs = append(msgs, eventMsg{Seq: seq, Raw: raw})
		}
	}
	return msgs, rt.seq
}

// publish：分配 seq、落库、广播。
func (r *Runtime) publish(sessionID, turnID string, ev interface{}) {
	raw, err := jsonMarshal(ev)
	if err != nil {
		log.Printf("[agent] marshal event: %v", err)
		return
	}
	rt := r.rt(sessionID)
	rt.mu.Lock()
	rt.seq++
	seq := rt.seq
	rt.events = append(rt.events, raw)
	for ch := range rt.subs {
		select {
		case ch <- eventMsg{Seq: seq, Raw: raw}:
		default: // 慢订阅者丢给重放机制兜底
		}
	}
	rt.mu.Unlock()
	_ = r.db.Create(&model.ChatTurnEvent{Seq: seq, SessionID: sessionID, TurnID: turnID, Event: raw}).Error
}

// Cancel 取消会话当前 turn（场景通过 guard 感知）。
func (r *Runtime) Cancel(sessionID string) {
	rt := r.rt(sessionID)
	rt.mu.Lock()
	c := rt.cancelC
	rt.mu.Unlock()
	if c != nil {
		close(c)
	}
}

/* ================= turn 执行 ================= */

type scenarioFn func(t *turn)

type turn struct {
	r         *Runtime
	db        *gorm.DB
	sessionID string
	turnID    string
	cancelC   chan struct{}

	text      strings.Builder
	thoughts  []ThoughtEvent
	cards     []CardEnvelope
}

func (t *turn) guard() bool {
	select {
	case <-t.cancelC:
		return true
	default:
		return false
	}
}

func (t *turn) emit(ev interface{}) { t.r.publish(t.sessionID, t.turnID, ev) }

func (t *turn) emitToken(delta string) {
	t.text.WriteString(delta)
	t.emit(TokenEvent{Type: "token", TextDelta: delta})
}

// say：3 字符一片、24ms 间隔流式输出（与前端 mock 节奏一致）
func (t *turn) say(text string, tickMs int) {
	t.emitToken("")
	runes := []rune(text)
	for i := 0; i < len(runes) && !t.guard(); i += 3 {
		end := i + 3
		if end > len(runes) {
			end = len(runes)
		}
		t.emitToken(string(runes[i:end]))
		time.Sleep(time.Duration(tickMs) * time.Millisecond)
	}
}

func (t *turn) thought(tool string, ms int) {
	if t.guard() {
		return
	}
	te := ThoughtEvent{Type: "thought", Step: 0, ToolName: tool, Status: "running"}
	t.emit(te)
	time.Sleep(time.Duration(ms) * time.Millisecond)
	if t.guard() {
		return
	}
	te.Status = "success"
	t.emit(te)
	t.thoughts = append(t.thoughts, te)
}

func (t *turn) emitCard(mode string, card CardEnvelope) {
	t.cards = append(t.cards, card)
	t.emit(CardEvent{Type: "card", Card: card, Mode: mode})
}

func (t *turn) sleep(ms int) {
	select {
	case <-time.After(time.Duration(ms) * time.Millisecond):
	case <-t.cancelC:
	}
}

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

// StartTurn：后台执行场景；无论正常/取消/出错，最终落 assistant 消息并发布 done。
func (r *Runtime) StartTurn(sessionID, turnID, text string) {
	rt := r.rt(sessionID)
	rt.mu.Lock()
	if rt.cancelC != nil {
		// 上一 turn 仍在跑：先取消（MVP 串行化会话）
		old := rt.cancelC
		rt.mu.Unlock()
		close(old)
		time.Sleep(50 * time.Millisecond)
		rt.mu.Lock()
	}
	c := make(chan struct{})
	rt.cancelC = c
	rt.mu.Unlock()

	t := &turn{r: r, db: r.db, sessionID: sessionID, turnID: turnID, cancelC: c}

	go func() {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("[agent] scenario panic: %v", rec)
				t.emit(ErrorEvent{Type: "error", Code: "internal", Message: "执行中断"})
			}
			r.finalize(t)
			rt.mu.Lock()
			if rt.cancelC == c {
				rt.cancelC = nil
			}
			rt.mu.Unlock()
		}()
		pickScenario(text)(t)
	}()
}

// finalize：把本轮累积的文本/思考/卡片固化为 assistant 消息。
func (r *Runtime) finalize(t *turn) {
	// 已有 done 事件则消息状态 final；被取消时同样落库（截断的文本）
	var maxSeq int
	r.db.Model(&model.ChatMessage{}).Where("session_id = ?", t.sessionID).
		Select("COALESCE(MAX(seq), 0)").Scan(&maxSeq)
	// nil slice 序列化为 null 会让前端崩溃，统一落 "[]"
	thoughts := t.thoughts
	if thoughts == nil {
		thoughts = []ThoughtEvent{}
	}
	cards := t.cards
	if cards == nil {
		cards = []CardEnvelope{}
	}
	msg := model.ChatMessage{
		ID: NewID("msg"), SessionID: t.sessionID, Seq: maxSeq + 1,
		Role: "assistant", Text: t.text.String(),
		Thoughts: jraw(thoughts), Cards: jraw(cards), Status: "final",
	}
	if err := r.db.Create(&msg).Error; err != nil {
		log.Printf("[agent] persist assistant message: %v", err)
	}
	r.db.Model(&model.ChatSession{}).Where("id = ?", t.sessionID).
		Update("updated_at", nowMillis())
}
