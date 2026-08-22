package handler

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"

	"db-cockpit/apiserver/internal/agent"
	"db-cockpit/apiserver/internal/envelope"
	"db-cockpit/apiserver/internal/model"
)

/* ================= Chat 会话 + SSE ================= */

type ChatHandler struct {
	H
	RT            *agent.Runtime
	ConfigVersion string // 落到 chat_turns 的配置版本快照（builtin-v0 / upstream-v0）
}

func (ch *ChatHandler) findSession(c *gin.Context) (*model.ChatSession, bool) {
	var s model.ChatSession
	if err := ch.DB.Where("id = ?", c.Param("id")).First(&s).Error; err != nil {
		envelope.NotFound(c, "session not found")
		return nil, false
	}
	return &s, true
}

// cancelRunningTurns：断开在跑执行流并把遗留 running 轮次置 cancelled
// （活跃轮次由其 finalize 落库；此处的直接更新兜底崩溃后遗留的僵尸行）。
func (ch *ChatHandler) cancelRunningTurns(sessionID string) {
	ch.RT.Cancel(sessionID)
	ch.DB.Model(&model.ChatTurn{}).Where("session_id = ? AND status = ?", sessionID, "running").
		Updates(map[string]interface{}{"status": "cancelled", "updated_at": time.Now().UnixMilli()})
	// 任务取消级联（限列写）：agent worker 观察 cancel_requested 后自行中止（依赖规则②）
	ch.DB.Model(&model.AgentTask{}).
		Where("session_id = ? AND status IN ? AND cancel_requested = ?",
			sessionID, []string{"pending", "running"}, false).
		Update("cancel_requested", true)
}

func (ch *ChatHandler) nextMsgSeq(sessionID string) int {
	var maxSeq int
	ch.DB.Model(&model.ChatMessage{}).Where("session_id = ?", sessionID).
		Select("COALESCE(MAX(seq), 0)").Scan(&maxSeq)
	return maxSeq + 1
}

func (ch *ChatHandler) ListSessions(c *gin.Context) {
	var sessions []model.ChatSession
	ch.DB.Order("created_at desc").Find(&sessions)
	if sessions == nil {
		sessions = []model.ChatSession{}
	}
	type sessionWithMessages struct {
		model.ChatSession
		Messages []model.ChatMessage `json:"messages"`
	}
	out := make([]sessionWithMessages, 0, len(sessions))
	for _, s := range sessions {
		var messages []model.ChatMessage
		ch.DB.Where("session_id = ?", s.ID).Order("seq asc").Find(&messages)
		if messages == nil {
			messages = []model.ChatMessage{}
		}
		out = append(out, sessionWithMessages{ChatSession: s, Messages: messages})
	}
	envelope.OK(c, out)
}

func (ch *ChatHandler) CreateSession(c *gin.Context) {
	var body struct {
		Title string `json:"title"`
	}
	_ = c.ShouldBindJSON(&body)
	title := body.Title
	if title == "" {
		title = "新会话"
	}
	now := time.Now().UnixMilli()
	s := model.ChatSession{ID: agent.NewID("sess"), UserID: "anonymous", Title: title, CreatedAt: now, UpdatedAt: now}
	if err := ch.DB.Create(&s).Error; err != nil {
		envelope.Internal(c, err)
		return
	}
	envelope.OK(c, s)
}

func (ch *ChatHandler) GetSession(c *gin.Context) {
	s, ok := ch.findSession(c)
	if !ok {
		return
	}
	var messages []model.ChatMessage
	ch.DB.Where("session_id = ?", s.ID).Order("seq asc").Find(&messages)
	if messages == nil {
		messages = []model.ChatMessage{}
	}
	envelope.OK(c, gin.H{"id": s.ID, "title": s.Title, "createdAt": s.CreatedAt, "updatedAt": s.UpdatedAt, "messages": messages})
}

func (ch *ChatHandler) DeleteSession(c *gin.Context) {
	s, ok := ch.findSession(c)
	if !ok {
		return
	}
	ch.cancelRunningTurns(s.ID) // 有在跑轮次先取消（docs §3.3-F3）
	ch.DB.Where("session_id = ?", s.ID).Delete(&model.ChatMessage{})
	ch.DB.Where("session_id = ?", s.ID).Delete(&model.ChatTurn{})
	ch.DB.Where("session_id = ?", s.ID).Delete(&model.ChatTurnEvent{})
	ch.DB.Delete(s)
	ch.RT.Drop(s.ID)
	envelope.OK(c, gin.H{"ok": true})
}

type importMessage struct {
	ID       string            `json:"id"`
	Role     string            `json:"role"`
	Text     string            `json:"text"`
	Thoughts json.RawMessage   `json:"thoughts"`
	Cards    json.RawMessage   `json:"cards"`
	Status   string            `json:"status"`
}
type importSession struct {
	ID        string          `json:"id"`
	Title     string          `json:"title"`
	CreatedAt int64           `json:"createdAt"`
	UpdatedAt int64           `json:"updatedAt"`
	Messages  []importMessage `json:"messages"`
}

// ImportSessions：前端 localStorage 会话一次性迁移（保留 id/时间戳/消息）
func (ch *ChatHandler) ImportSessions(c *gin.Context) {
	var body struct {
		Sessions []importSession `json:"sessions"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		envelope.BadRequest(c, "invalid body")
		return
	}
	imported := 0
	for _, in := range body.Sessions {
		if in.ID == "" {
			continue
		}
		s := model.ChatSession{ID: in.ID, Title: in.Title, CreatedAt: in.CreatedAt, UpdatedAt: in.UpdatedAt}
		if err := ch.DB.Save(&s).Error; err != nil {
			continue
		}
		ch.DB.Where("session_id = ?", in.ID).Delete(&model.ChatMessage{})
		for i, m := range in.Messages {
			thoughts := m.Thoughts
			if len(thoughts) == 0 {
				thoughts = json.RawMessage("[]")
			}
			cards := m.Cards
			if len(cards) == 0 {
				cards = json.RawMessage("[]")
			}
			status := m.Status
			if status == "" {
				status = "final"
			}
			ch.DB.Create(&model.ChatMessage{
				ID: m.ID, SessionID: in.ID, Seq: i + 1, Role: m.Role, Text: m.Text,
				Thoughts: datatypes.JSON(thoughts), Cards: datatypes.JSON(cards), Status: status,
			})
		}
		imported++
	}
	envelope.OK(c, gin.H{"imported": imported})
}

// SubmitTurn：幂等 + 取消替换 → 落用户消息与轮次行（running）→ 触发后台执行；返回 turnId。
func (ch *ChatHandler) SubmitTurn(c *gin.Context) {
	s, ok := ch.findSession(c)
	if !ok {
		return
	}
	var body struct {
		Text            string `json:"text" binding:"required"`
		ClientRequestID string `json:"client_request_id"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		envelope.BadRequest(c, "text required")
		return
	}
	// 幂等：同 client_request_id 的重复提交（前端重试/双击）直接返回已创建轮次（docs §3.3-F1）
	if body.ClientRequestID != "" {
		var exist model.ChatTurn
		if err := ch.DB.Where("client_request_id = ?", body.ClientRequestID).First(&exist).Error; err == nil {
			var msg model.ChatMessage
			ch.DB.Where("session_id = ? AND turn_id = ? AND role = ?", exist.SessionID, exist.ID, "user").
				Order("seq desc").First(&msg)
			envelope.OK(c, gin.H{"turnId": exist.ID, "duplicated": true, "message": msg})
			return
		}
	}
	// 取消替换：上一轮仍在 running → 隐含取消（docs §3.3-D2）
	ch.cancelRunningTurns(s.ID)

	turnID := agent.NewID("turn")
	now := time.Now().UnixMilli()
	var turnSeq int
	ch.DB.Model(&model.ChatTurn{}).Where("session_id = ?", s.ID).
		Select("COALESCE(MAX(seq), 0)").Scan(&turnSeq)
	var crid *string // 仅显式携带幂等键时参与唯一去重
	if body.ClientRequestID != "" {
		crid = &body.ClientRequestID
	}
	if err := ch.DB.Create(&model.ChatTurn{
		ID: turnID, SessionID: s.ID, Seq: turnSeq + 1, Kind: "user", Status: "running",
		ClientRequestID: crid, ConfigVersion: ch.ConfigVersion,
		UserMsg: body.Text, CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		envelope.Internal(c, err)
		return
	}
	msg := model.ChatMessage{
		ID: agent.NewID("msg"), SessionID: s.ID, TurnID: turnID, Seq: ch.nextMsgSeq(s.ID),
		Role: "user", Text: body.Text,
		Thoughts: datatypes.JSON([]byte("[]")), Cards: datatypes.JSON([]byte("[]")), Status: "final",
	}
	if err := ch.DB.Create(&msg).Error; err != nil {
		envelope.Internal(c, err)
		return
	}
	// 首条消息时以问题开头作为会话标题（与前端原行为一致）
	var cnt int64
	ch.DB.Model(&model.ChatMessage{}).Where("session_id = ?", s.ID).Count(&cnt)
	if cnt <= 1 {
		title := []rune(body.Text)
		if len(title) > 24 {
			title = title[:24]
		}
		ch.DB.Model(&model.ChatSession{}).Where("id = ?", s.ID).Update("title", string(title))
	}
	ch.DB.Model(&model.ChatSession{}).Where("id = ?", s.ID).Update("updated_at", now)
	ch.RT.StartTurn(s.ID, turnID, body.Text)
	envelope.OK(c, gin.H{"turnId": turnID, "message": msg})
}

func (ch *ChatHandler) CancelTurn(c *gin.Context) {
	s, ok := ch.findSession(c)
	if !ok {
		return
	}
	ch.cancelRunningTurns(s.ID)
	envelope.OK(c, gin.H{"ok": true})
}

// Stream：SSE 六事件流；支持 ?last_event_id=（或 Last-Event-ID 头）断线重放。
func (ch *ChatHandler) Stream(c *gin.Context) {
	_, ok := ch.findSession(c)
	if !ok {
		return
	}
	sessionID := c.Param("id")

	var after int64
	if v := c.Query("last_event_id"); v != "" {
		after, _ = strconv.ParseInt(v, 10, 64)
	} else if v := c.GetHeader("Last-Event-ID"); v != "" {
		after, _ = strconv.ParseInt(v, 10, 64)
	}

	// 先订阅再取快照，事件不丢；channel 内按 seq 去重
	msgCh := ch.RT.Subscribe(sessionID)
	defer ch.RT.Unsubscribe(sessionID, msgCh)
	snap, _ := ch.RT.Snapshot(sessionID, after)

	c.Header("Content-Type", "text/event-stream; charset=utf-8")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Status(200)

	lastSent := after
	write := func(seq int64, raw []byte) {
		fmt.Fprintf(c.Writer, "id: %d\ndata: %s\n\n", seq, raw)
		if f, ok := c.Writer.(interface{ Flush() }); ok {
			f.Flush()
		}
	}
	terminal := func(raw []byte) bool {
		var probe struct {
			Type string `json:"type"`
		}
		if json.Unmarshal(raw, &probe) != nil {
			return false
		}
		return probe.Type == "done" || probe.Type == "error"
	}

	for _, m := range snap {
		if m.Seq <= lastSent {
			continue
		}
		write(m.Seq, m.Raw)
		lastSent = m.Seq
	}

	ctx := c.Request.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case m, ok := <-msgCh:
			if !ok {
				return
			}
			if m.Seq <= lastSent {
				continue
			}
			write(m.Seq, m.Raw)
			lastSent = m.Seq
			if terminal(m.Raw) {
				time.Sleep(20 * time.Millisecond) // 让对端收完
				return
			}
		}
	}
}
