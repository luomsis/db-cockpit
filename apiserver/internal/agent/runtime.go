package agent

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"gorm.io/datatypes"
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
	Agent    string `json:"agent,omitempty"` // 产出事件的 subagent 标识（动态装配对用户可见可追溯）
}

type TokenEvent struct {
	Type      string `json:"type"`
	TextDelta string `json:"text_delta"`
	Agent     string `json:"agent,omitempty"`
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
	Type  string          `json:"type"`
	Usage json.RawMessage `json:"usage,omitempty"` // token 用量（可缺省）
}

type ErrorEvent struct {
	Type    string `json:"type"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

/* ================= 事件源（AGENT_MODE 选择：builtin 场景 / agentcluster exec 执行流） ================= */

// TurnSource 产出一次轮次的执行事件。Run 返回即本轮执行结束；
// 正常路径应已发出 done 或 error 终态事件（未发且未被取消时由 StartTurn 兜底 unexpected_end）。
type TurnSource interface {
	Run(t *turn)
}

/* ================= 会话运行时：事件发布 / 订阅 / 取消 ================= */

type sessionRT struct {
	mu      sync.Mutex
	seq     int64             // 已发布的最后一个事件序号
	events  [][]byte          // 本次进程生命周期内的事件（json）
	subs    map[chan eventMsg]struct{}
	cancelC chan struct{}     // 当前 turn 的取消信号（nil = 无活跃轮次）
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
	source   TurnSource
}

func NewRuntime(gdb *gorm.DB, source TurnSource) *Runtime {
	if source == nil {
		source = NewBuiltinSource()
	}
	return &Runtime{sessions: map[string]*sessionRT{}, db: gdb, source: source}
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
	if err := r.db.Create(&model.ChatTurnEvent{Seq: seq, SessionID: sessionID, TurnID: turnID, Event: raw}).Error; err != nil {
		// 落库失败只影响断线重放，不影响实时流；但必须可见，否则会静默丢失重放数据
		log.Printf("[agent] persist event %s#%d: %v", sessionID, seq, err)
	}
}

// Cancel 取消会话当前 turn（执行源经 ctx 感知；upstream 表现为断开 exec 连接）。
// 取 nil 防重复 close，保证 Cancel/StartTurn 并发安全。
func (r *Runtime) Cancel(sessionID string) {
	rt := r.rt(sessionID)
	rt.mu.Lock()
	c := rt.cancelC
	rt.cancelC = nil
	rt.mu.Unlock()
	if c != nil {
		close(c)
	}
}

/* ================= turn 执行 ================= */

type turn struct {
	r         *Runtime
	db        *gorm.DB
	sessionID string
	turnID    string
	input     string         // 用户输入原文（builtin 选场景 / upstream 作为 user_msg）
	cancelC   chan struct{}  // 取消信号（用户取消 / 取消替换 / 会话删除）
	ctx       context.Context // 随 cancelC 取消；upstream 据此断开 exec 连接（断连即取消）

	text     strings.Builder
	thoughts []ThoughtEvent
	cards    []CardEnvelope

	done     bool
	usage    json.RawMessage
	failOnce sync.Once
	errCode  string
	errMsg   string
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

// say：3 字符一片、tickMs 间隔流式输出（与前端 mock 节奏一致）
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

// markFailed：记录首个失败原因（once 保护，看门狗与读循环并发触发时取先到者）。
func (t *turn) markFailed(code, msg string) {
	t.failOnce.Do(func() { t.errCode, t.errMsg = code, msg })
}

// upsertCard：card 事件按 card_id 就地合并（create/update 幂等，装配最终态卡片列表）。
func (t *turn) upsertCard(card CardEnvelope) {
	for i := range t.cards {
		if t.cards[i].CardID == card.CardID {
			t.cards[i] = card
			return
		}
	}
	t.cards = append(t.cards, card)
}

// StartTurn：后台执行一轮；无论正常/取消/出错，最终落 assistant 消息并推进轮次状态机。
func (r *Runtime) StartTurn(sessionID, turnID, text string) {
	rt := r.rt(sessionID)
	rt.mu.Lock()
	if rt.cancelC != nil {
		// 上一 turn 仍在跑：先取消（同会话串行化 = 取消替换，docs §3.3-D2）
		old := rt.cancelC
		rt.cancelC = nil
		rt.mu.Unlock()
		close(old)
		time.Sleep(50 * time.Millisecond)
		rt.mu.Lock()
	}
	c := make(chan struct{})
	rt.cancelC = c
	rt.mu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	t := &turn{r: r, db: r.db, sessionID: sessionID, turnID: turnID, input: text, cancelC: c, ctx: ctx}
	go func() { <-c; cancel() }() // 取消信号传导为 ctx 取消（upstream 断开 exec 连接）

	go func() {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("[agent] turn %s panic: %v", turnID, rec)
				t.emit(ErrorEvent{Type: "error", Code: "internal", Message: "执行中断"})
				t.markFailed("internal", "执行中断")
			}
			// 契约兜底：源未产出终态且非用户取消 → unexpected_end（docs §3.3-E5）
			if !t.done && t.errCode == "" && !t.guard() {
				t.emit(ErrorEvent{Type: "error", Code: "unexpected_end", Message: "执行源未产出终止事件"})
				t.markFailed("unexpected_end", "执行源未产出终止事件")
			}
			r.finalize(t)
			cancel()
			rt.mu.Lock()
			if rt.cancelC == c {
				rt.cancelC = nil
			}
			rt.mu.Unlock()
		}()
		r.source.Run(t)
	}()
}

// finalize：把本轮累积的文本/思考/卡片固化为 assistant 消息（含异常截断），并推进 chat_turns 状态机。
func (r *Runtime) finalize(t *turn) {
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
		ID: NewID("msg"), SessionID: t.sessionID, TurnID: t.turnID, Seq: maxSeq + 1,
		Role: "assistant", Text: t.text.String(),
		Thoughts: jraw(thoughts), Cards: jraw(cards), Status: "final",
	}
	if err := r.db.Create(&msg).Error; err != nil {
		log.Printf("[agent] persist assistant message: %v", err)
	}

	status := "done"
	switch {
	case t.errCode != "":
		status = "failed"
	case t.done:
		status = "done"
	case t.guard():
		status = "cancelled"
	default:
		status = "failed"
	}
	updates := map[string]interface{}{"status": status, "updated_at": nowMillis()}
	if len(t.usage) > 0 {
		updates["usage"] = datatypes.JSON(t.usage)
	}
	if t.errCode != "" {
		updates["error_code"] = t.errCode
		updates["error_msg"] = t.errMsg
	}
	r.db.Model(&model.ChatTurn{}).Where("id = ?", t.turnID).Updates(updates)
	r.db.Model(&model.ChatSession{}).Where("id = ?", t.sessionID).
		Update("updated_at", nowMillis())
}

// RecoverInterruptedTurns：启动恢复——重启后所有 running 轮次不可能仍在执行
// （exec 连接已随进程消失，agentcluster 断连即取消），统一改判 failed/restart（docs §3.3-F2）。
func RecoverInterruptedTurns(gdb *gorm.DB) {
	res := gdb.Model(&model.ChatTurn{}).Where("status = ?", "running").
		Updates(map[string]interface{}{
			"status": "failed", "error_code": "restart",
			"error_msg": "apiserver 重启，轮次中断", "updated_at": nowMillis(),
		})
	if res.Error != nil {
		log.Printf("[agent] 启动恢复 running 轮次失败: %v", res.Error)
		return
	}
	if res.RowsAffected > 0 {
		log.Printf("[agent] 启动恢复：%d 个 running 轮次改判 failed/restart", res.RowsAffected)
	}
}
