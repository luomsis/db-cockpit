package agent

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync/atomic"
	"time"

	"db-cockpit/apiserver/internal/model"
)

/* ================= upstream 事件源：agentcluster exec 执行流 =================
 * 契约（docs《交互时序与生命周期》§5.1）：
 *   - POST {execURL}，body = {turn_id, session_id, user_msg, config_version}，响应为 SSE；
 *   - 上游只发 data 事件（六类信封，不带序号；seq 由本侧分配）与 `: ping` 注释行心跳；
 *   - 断连即取消（ctx 取消 → 连接关闭，上游感知断连须中止执行）；
 *   - 90s（可配）无事件且无心跳 → 看门狗发 upstream_idle 并断连；
 *   - 每轮必发 done 或 error，唯一例外是被取消。
 */

type upstreamSource struct {
	execURL     string
	idleTimeout time.Duration
	client      *http.Client
}

// NewUpstreamSource：execURL 为完整执行端点；idleTimeout<=0 时取默认 90s。
func NewUpstreamSource(execURL string, idleTimeout time.Duration) *upstreamSource {
	if idleTimeout <= 0 {
		idleTimeout = 90 * time.Second
	}
	return &upstreamSource{execURL: execURL, idleTimeout: idleTimeout, client: &http.Client{}}
}

type execRequest struct {
	TurnID        string       `json:"turn_id"`
	SessionID     string       `json:"session_id"`
	UserMsg       string       `json:"user_msg"`
	ConfigVersion string       `json:"config_version,omitempty"`
	Kind          string       `json:"kind,omitempty"`      // user | system_resume
	ResumeOf      string       `json:"resume_of,omitempty"` // wake 续跑时指向 task_id
	AuthContext   *AuthContext `json:"auth_context,omitempty"`
}

// AuthContext：apiserver 统一鉴权后的授权上下文，随 exec 请求下发（依赖规则④：权限只在 Go 校验一次）。
// MVP 权限桩：InstanceScope/ToolAllowlist 为空 = 全量；二期接 SSO/RBAC 后按请求计算真实范围。
type AuthContext struct {
	UserID        string   `json:"user_id"`
	InstanceScope []string `json:"instance_scope,omitempty"` // 可见实例 ID；空 = 全量
	ToolAllowlist []string `json:"tool_allowlist,omitempty"` // 工具白名单；空 = 全量
}

// wireEvent：上游六类事件信封合一解析；未识别 type 原样转发不参与装配。
type wireEvent struct {
	Type      string          `json:"type"`
	Step      int             `json:"step"`
	ToolName  string          `json:"tool_name"`
	Status    string          `json:"status"`
	Agent     string          `json:"agent"`
	TextDelta string          `json:"text_delta"`
	Card      *CardEnvelope   `json:"card"`
	Mode      string          `json:"mode"`
	TaskID    string          `json:"task_id"`
	Progress  float64         `json:"progress"`
	Stage     string          `json:"stage"`
	Usage     json.RawMessage `json:"usage"`
	Code      string          `json:"code"`
	Message   string          `json:"message"`
}

var dataPrefix = []byte("data:")

func (s *upstreamSource) Run(t *turn) {
	// 从呈现域补全 exec 请求：kind/resume_of/config_version（ChatTurn 行）与
	// auth_context（会话归属用户，MVP 权限桩=全量范围）——依赖规则②/④
	body := execRequest{
		TurnID:      t.turnID,
		SessionID:   t.sessionID,
		UserMsg:     t.input,
		AuthContext: &AuthContext{UserID: "anonymous"},
	}
	var turnRow model.ChatTurn
	if err := t.db.Where("id = ?", t.turnID).First(&turnRow).Error; err == nil {
		body.Kind = turnRow.Kind
		body.ResumeOf = turnRow.ResumeOf
		body.ConfigVersion = turnRow.ConfigVersion
	}
	var sessRow model.ChatSession
	if err := t.db.Where("id = ?", t.sessionID).First(&sessRow).Error; err == nil && sessRow.UserID != "" {
		body.AuthContext.UserID = sessRow.UserID
	}
	payload, err := jsonMarshal(body)
	if err != nil {
		s.fail(t, "upstream_error", err.Error())
		return
	}
	req, err := http.NewRequestWithContext(t.ctx, http.MethodPost, s.execURL, bytes.NewReader(payload))
	if err != nil {
		s.fail(t, "upstream_error", err.Error())
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	resp, err := s.client.Do(req)
	if err != nil {
		if t.guard() {
			return // 用户已取消（含取消替换），不算失败
		}
		s.fail(t, "upstream_unreachable", "无法连接执行服务："+err.Error())
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		s.fail(t, "upstream_error", fmt.Sprintf("执行服务返回 %d: %s", resp.StatusCode, string(b)))
		return
	}

	// 空闲看门狗：无事件且无心跳超过 idleTimeout → 先发 upstream_idle 再断连（docs §3.3-E3）
	var lastActivity atomic.Int64
	lastActivity.Store(time.Now().UnixNano())
	stopWatch := make(chan struct{})
	defer close(stopWatch)
	tick := s.idleTimeout / 3
	if tick > 5*time.Second {
		tick = 5 * time.Second
	}
	if tick < 100*time.Millisecond {
		tick = 100 * time.Millisecond
	}
	go func() {
		tk := time.NewTicker(tick)
		defer tk.Stop()
		for {
			select {
			case <-stopWatch:
				return
			case <-t.ctx.Done():
				return
			case <-tk.C:
				if time.Since(time.Unix(0, lastActivity.Load())) > s.idleTimeout {
					t.markFailed("upstream_idle", "执行流空闲超时（无事件且无心跳）")
					t.emit(ErrorEvent{Type: "error", Code: "upstream_idle", Message: "执行服务长时间无响应，本轮已终止；已产出内容已保留"})
					t.r.Cancel(t.sessionID) // 断开 exec 连接
					return
				}
			}
		}
	}()

	sc := bufio.NewScanner(resp.Body)
	sc.Buffer(make([]byte, 0, 64*1024), 8*1024*1024) // 大卡片事件
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		if line[0] == ':' {
			// `: ping` 注释行心跳：重置看门狗，不转发（前端心跳由 Go 自己发）
			lastActivity.Store(time.Now().UnixNano())
			continue
		}
		if !bytes.HasPrefix(line, dataPrefix) {
			continue
		}
		lastActivity.Store(time.Now().UnixNano())
		raw := bytes.TrimSpace(line[len(dataPrefix):])

		var ev wireEvent
		if err := json.Unmarshal(raw, &ev); err != nil {
			log.Printf("[agent] upstream %s: 非法事件 %s: %v", t.turnID, truncateStr(string(raw), 128), err)
			continue
		}
		switch ev.Type {
		case "token":
			t.text.WriteString(ev.TextDelta)
			t.emit(TokenEvent{Type: "token", TextDelta: ev.TextDelta, Agent: ev.Agent})
		case "thought":
			te := ThoughtEvent{Type: "thought", Step: ev.Step, ToolName: ev.ToolName, Status: ev.Status, Agent: ev.Agent}
			t.emit(te)
			if ev.Status == "success" {
				t.thoughts = append(t.thoughts, te)
			}
		case "card":
			if ev.Card != nil {
				t.upsertCard(*ev.Card) // 按 card_id 就地合并（create/update 幂等）
				t.emit(CardEvent{Type: "card", Card: *ev.Card, Mode: ev.Mode})
			}
		case "progress":
			t.emit(ProgressEvent{Type: "progress", TaskID: ev.TaskID, Progress: ev.Progress, Stage: ev.Stage})
		case "done":
			t.usage = ev.Usage
			t.done = true
			t.emit(DoneEvent{Type: "done", Usage: ev.Usage})
			return
		case "error":
			code := ev.Code
			if code == "" {
				code = "upstream_error"
			}
			t.markFailed(code, ev.Message)
			t.emit(ErrorEvent{Type: "error", Code: code, Message: ev.Message})
			return
		default:
			// 未识别类型：信封级兼容，原样转发不参与装配
			t.emit(json.RawMessage(raw))
		}
	}
	if err := sc.Err(); err != nil {
		if t.guard() || t.errCode != "" {
			return // 用户取消（静默，docs §3.3-D1）或看门狗已定终态
		}
		s.fail(t, "upstream_broken", "执行流中断："+err.Error())
		return
	}
	// 流正常结束但没有 done/error：交给 StartTurn 的 unexpected_end 兜底（docs §3.3-E5）
}

func (s *upstreamSource) fail(t *turn, code, msg string) {
	t.markFailed(code, msg)
	t.emit(ErrorEvent{Type: "error", Code: code, Message: msg})
}

func truncateStr(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}
