package agent

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"db-cockpit/apiserver/internal/model"
)

// testDB：联调环境 PG 可达则真库测试，否则跳过（DSN 与 apiserver 缺省一致）。
func testDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		dsn = "host=localhost port=55432 user=graphiti password=graphiti dbname=db_cockpit sslmode=disable"
	}
	gdb, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Skipf("PG 不可达，跳过：%v", err)
	}
	if err := gdb.AutoMigrate(&model.ChatSession{}, &model.ChatTurn{}, &model.ChatMessage{}, &model.ChatTurnEvent{}, &model.AgentTask{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return gdb
}

func newSessionTurn(t *testing.T, gdb *gorm.DB) (sessID, turnID string) {
	t.Helper()
	sessID = NewID("sess")
	turnID = NewID("turn")
	now := nowMillis()
	if err := gdb.Create(&model.ChatSession{ID: sessID, UserID: "anonymous", Title: "t", CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatalf("建会话失败: %v", err)
	}
	if err := gdb.Create(&model.ChatTurn{ID: turnID, SessionID: sessID, Seq: 1, Kind: "user", Status: "running",
		UserMsg: "测试", CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatalf("建轮次失败: %v", err)
	}
	return sessID, turnID
}

func waitTurnFinal(t *testing.T, gdb *gorm.DB, turnID string, timeout time.Duration) model.ChatTurn {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		var turn model.ChatTurn
		if err := gdb.Where("id = ?", turnID).First(&turn).Error; err == nil && turn.Status != "running" {
			return turn
		}
		time.Sleep(30 * time.Millisecond)
	}
	t.Fatalf("轮次 %s 未在 %s 内到达终态", turnID, timeout)
	return model.ChatTurn{}
}

func waitEventCount(t *testing.T, gdb *gorm.DB, sessID string, n int64, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		var cnt int64
		gdb.Model(&model.ChatTurnEvent{}).Where("session_id = ?", sessID).Count(&cnt)
		if cnt >= n {
			return
		}
		time.Sleep(30 * time.Millisecond)
	}
	t.Fatalf("会话 %s 事件数未达到 %d", sessID, n)
}

func sseHandler(fn func(w http.ResponseWriter, flush func())) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flush := w.(http.Flusher).Flush
		flush()
		fn(w, flush)
	})
}

// 场景 1：happy path——事件落库 seq 单调、assistant 信封级装配、turn=done+usage。
func TestUpstreamHappyPath(t *testing.T) {
	gdb := testDB(t)
	up := httptest.NewServer(sseHandler(func(w http.ResponseWriter, flush func()) {
		fmt.Fprint(w, ": ping\n\n")
		fmt.Fprint(w, "data: {\"type\":\"thought\",\"step\":1,\"tool_name\":\"builtin_get_metrics\",\"status\":\"success\"}\n\n")
		fmt.Fprint(w, "data: {\"type\":\"token\",\"text_delta\":\"你好\"}\n\n")
		fmt.Fprint(w, "data: {\"type\":\"token\",\"text_delta\":\"世界\"}\n\n")
		fmt.Fprint(w, "data: {\"type\":\"card\",\"mode\":\"create\",\"card\":{\"card_id\":\"card_1\",\"card_type\":\"metric_chart\",\"protocol_version\":\"1.0\",\"title\":\"CPU\",\"status\":\"final\",\"context\":{},\"payload\":{},\"fallback_text\":\"\"}}\n\n")
		fmt.Fprint(w, "data: {\"type\":\"custom_x\",\"foo\":1}\n\n")
		fmt.Fprint(w, "data: {\"type\":\"done\",\"usage\":{\"prompt_tokens\":10,\"completion_tokens\":5}}\n\n")
		flush()
	}))
	defer up.Close()

	sessID, turnID := newSessionTurn(t, gdb)
	rt := NewRuntime(gdb, NewUpstreamSource(up.URL, 0))
	rt.StartTurn(sessID, turnID, "测试")

	turn := waitTurnFinal(t, gdb, turnID, 5*time.Second)
	if turn.Status != "done" {
		t.Fatalf("status = %s, want done", turn.Status)
	}
	var usage struct {
		PromptTokens int `json:"prompt_tokens"`
	}
	if err := json.Unmarshal(turn.Usage, &usage); err != nil || usage.PromptTokens != 10 {
		t.Fatalf("usage 未落库或不对: %s err=%v", turn.Usage, err)
	}

	var msg model.ChatMessage
	if err := gdb.Where("session_id = ? AND turn_id = ? AND role = ?", sessID, turnID, "assistant").
		First(&msg).Error; err != nil {
		t.Fatalf("assistant 消息未落库: %v", err)
	}
	if msg.Text != "你好世界" {
		t.Fatalf("text = %q, want 你好世界", msg.Text)
	}
	var cards []CardEnvelope
	if err := json.Unmarshal(msg.Cards, &cards); err != nil || len(cards) != 1 || cards[0].CardID != "card_1" {
		t.Fatalf("cards 装配错误: %s err=%v", msg.Cards, err)
	}
	var thoughts []ThoughtEvent
	if err := json.Unmarshal(msg.Thoughts, &thoughts); err != nil || len(thoughts) != 1 {
		t.Fatalf("thoughts 装配错误: %s err=%v", msg.Thoughts, err)
	}

	var evs []model.ChatTurnEvent
	gdb.Where("session_id = ?", sessID).Order("seq asc").Find(&evs)
	if len(evs) < 6 {
		t.Fatalf("事件数 = %d, want >= 6", len(evs))
	}
	for i := 1; i < len(evs); i++ {
		if evs[i].Seq <= evs[i-1].Seq {
			t.Fatalf("seq 非单调: %d 后跟 %d", evs[i-1].Seq, evs[i].Seq)
		}
	}
	var probe struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(evs[len(evs)-1].Event, &probe); err != nil || probe.Type != "done" {
		t.Fatalf("末事件应为 done: %s", evs[len(evs)-1].Event)
	}
	// 未知类型原样转发
	found := false
	for _, e := range evs {
		if json.Unmarshal(e.Event, &probe) == nil && probe.Type == "custom_x" {
			found = true
		}
	}
	if !found {
		t.Fatalf("未知类型事件未被原样转发")
	}
}

// 场景 2：上游假死（无事件无心跳）→ 看门狗 upstream_idle。
func TestUpstreamIdleWatchdog(t *testing.T) {
	gdb := testDB(t)
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.(http.Flusher).Flush()
		<-r.Context().Done() // 假死：什么都不发，等客户端断开
	}))
	defer up.Close()

	sessID, turnID := newSessionTurn(t, gdb)
	rt := NewRuntime(gdb, NewUpstreamSource(up.URL, 300*time.Millisecond))
	rt.StartTurn(sessID, turnID, "测试")

	turn := waitTurnFinal(t, gdb, turnID, 5*time.Second)
	if turn.Status != "failed" || turn.ErrorCode != "upstream_idle" {
		t.Fatalf("status=%s error_code=%s, want failed/upstream_idle", turn.Status, turn.ErrorCode)
	}
	var evs []model.ChatTurnEvent
	gdb.Where("session_id = ?", sessID).Find(&evs)
	found := false
	var probe struct {
		Code string `json:"code"`
		Type string `json:"type"`
	}
	for _, e := range evs {
		if json.Unmarshal(e.Event, &probe) == nil && probe.Type == "error" && probe.Code == "upstream_idle" {
			found = true
		}
	}
	if !found {
		t.Fatalf("事件流中缺少 upstream_idle error 事件")
	}
}

// 场景 3：断连即取消——上游感知断开、turn=cancelled、截断消息保留。
func TestUpstreamCancelByDisconnect(t *testing.T) {
	gdb := testDB(t)
	disconnected := make(chan struct{})
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flush := w.(http.Flusher).Flush
		flush()
		for i := 0; ; i++ {
			select {
			case <-r.Context().Done():
				close(disconnected)
				return
			case <-time.After(60 * time.Millisecond):
				fmt.Fprintf(w, "data: {\"type\":\"token\",\"text_delta\":\"段%d\"}\n\n", i)
				flush()
			}
		}
	}))
	defer up.Close()

	sessID, turnID := newSessionTurn(t, gdb)
	rt := NewRuntime(gdb, NewUpstreamSource(up.URL, 0))
	rt.StartTurn(sessID, turnID, "测试")
	waitEventCount(t, gdb, sessID, 1, 5*time.Second)

	rt.Cancel(sessID)
	select {
	case <-disconnected:
	case <-time.After(3 * time.Second):
		t.Fatalf("上游未感知断连（断连即取消未生效）")
	}

	turn := waitTurnFinal(t, gdb, turnID, 5*time.Second)
	if turn.Status != "cancelled" {
		t.Fatalf("status = %s, want cancelled", turn.Status)
	}
	var msg model.ChatMessage
	if err := gdb.Where("session_id = ? AND turn_id = ? AND role = ?", sessID, turnID, "assistant").
		First(&msg).Error; err != nil {
		t.Fatalf("截断 assistant 消息未落库: %v", err)
	}
	if msg.Text == "" {
		t.Fatalf("截断消息应保留已产出文本")
	}
}

// 场景 4：上游主动业务错误 → turn=failed 记上游 code。
func TestUpstreamErrorEvent(t *testing.T) {
	gdb := testDB(t)
	up := httptest.NewServer(sseHandler(func(w http.ResponseWriter, flush func()) {
		fmt.Fprint(w, "data: {\"type\":\"token\",\"text_delta\":\"已获证据\"}\n\n")
		fmt.Fprint(w, "data: {\"type\":\"error\",\"code\":\"budget_exceeded\",\"message\":\"步数预算超限\",\"fallback_text\":\"基于已有证据给初步结论\"}\n\n")
		flush()
	}))
	defer up.Close()

	sessID, turnID := newSessionTurn(t, gdb)
	rt := NewRuntime(gdb, NewUpstreamSource(up.URL, 0))
	rt.StartTurn(sessID, turnID, "测试")

	turn := waitTurnFinal(t, gdb, turnID, 5*time.Second)
	if turn.Status != "failed" || turn.ErrorCode != "budget_exceeded" {
		t.Fatalf("status=%s error_code=%s, want failed/budget_exceeded", turn.Status, turn.ErrorCode)
	}
}

// 场景 5：流正常结束但无终态事件 → unexpected_end 兜底。
func TestUpstreamNoTerminal(t *testing.T) {
	gdb := testDB(t)
	up := httptest.NewServer(sseHandler(func(w http.ResponseWriter, flush func()) {
		fmt.Fprint(w, "data: {\"type\":\"token\",\"text_delta\":\"半句\"}\n\n")
		flush() // 直接结束，无 done/error
	}))
	defer up.Close()

	sessID, turnID := newSessionTurn(t, gdb)
	rt := NewRuntime(gdb, NewUpstreamSource(up.URL, 0))
	rt.StartTurn(sessID, turnID, "测试")

	turn := waitTurnFinal(t, gdb, turnID, 5*time.Second)
	if turn.Status != "failed" || turn.ErrorCode != "unexpected_end" {
		t.Fatalf("status=%s error_code=%s, want failed/unexpected_end", turn.Status, turn.ErrorCode)
	}
}
