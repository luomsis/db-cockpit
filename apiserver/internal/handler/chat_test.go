package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"db-cockpit/apiserver/internal/agent"
	"db-cockpit/apiserver/internal/model"
)

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
	if err := gdb.AutoMigrate(&model.ChatSession{}, &model.ChatTurn{}, &model.ChatMessage{}, &model.ChatTurnEvent{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return gdb
}

func submitBody(text, crid string) []byte {
	b, _ := json.Marshal(map[string]string{"text": text, "client_request_id": crid})
	return b
}

// 慢上游：发 1 个 token 后挂住等断连或 2s 超时再 done（用于取消替换场景；2s < 断言窗口 5s）。
func slowUpstream() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flush := w.(http.Flusher).Flush
		fmt.Fprint(w, "data: {\"type\":\"token\",\"text_delta\":\"开头\"}\n\n")
		flush()
		select {
		case <-r.Context().Done():
			return
		case <-time.After(2 * time.Second):
			fmt.Fprint(w, "data: {\"type\":\"done\",\"usage\":{\"prompt_tokens\":1}}\n\n")
			flush()
		}
	}))
}

// 提交幂等（同 client_request_id 返回原轮次）+ 取消替换（新提交隐含取消旧轮）。
func TestSubmitTurnIdempotencyAndCancelReplace(t *testing.T) {
	gdb := testDB(t)
	up := slowUpstream()
	defer up.Close()

	gin.SetMode(gin.TestMode)
	rt := agent.NewRuntime(gdb, agent.NewUpstreamSource(up.URL, 0))
	ch := &ChatHandler{H: H{DB: gdb}, RT: rt, ConfigVersion: "test-v0"}
	r := gin.New()
	r.POST("/api/chat/sessions/:id/turns", ch.SubmitTurn)

	now := time.Now().UnixMilli()
	sess := model.ChatSession{ID: agent.NewID("sess"), UserID: "anonymous", Title: "t", CreatedAt: now, UpdatedAt: now}
	gdb.Create(&sess)
	post := func(body []byte) (turnID string, duplicated bool) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/chat/sessions/"+sess.ID+"/turns", bytes.NewReader(body))
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("submit 状态码 %d: %s", w.Code, w.Body.String())
		}
		var resp struct {
			Data struct {
				TurnID     string `json:"turnId"`
				Duplicated bool   `json:"duplicated"`
			} `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("解析响应: %v", err)
		}
		return resp.Data.TurnID, resp.Data.Duplicated
	}

	// 幂等键每次运行唯一（dev 库不清理历史，避免命中上轮残留）
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	crid1, crid2 := "crid_x_"+suffix, "crid_y_"+suffix

	// ① 首次提交
	turn1, dup := post(submitBody("第一问", crid1))
	if dup {
		t.Fatalf("首次提交不应标记 duplicated")
	}
	// 等首轮至少落一个事件（确认执行中）
	deadline := time.Now().Add(5 * time.Second)
	for {
		var cnt int64
		gdb.Model(&model.ChatTurnEvent{}).Where("session_id = ?", sess.ID).Count(&cnt)
		if cnt >= 1 || time.Now().After(deadline) {
			break
		}
		time.Sleep(30 * time.Millisecond)
	}
	// ② 重复提交同 crid → 幂等返回原轮次
	turn1b, dup := post(submitBody("第一问（重试）", crid1))
	if !dup || turn1b != turn1 {
		t.Fatalf("幂等失效: turn1b=%s dup=%v, want %s true", turn1b, dup, turn1)
	}
	// ③ 追问取消替换：新 crid → 新轮次，旧轮最终 cancelled
	turn2, _ := post(submitBody("第二问", crid2))
	if turn2 == turn1 {
		t.Fatalf("新提交应产生新轮次")
	}
	waitStatus := func(turnID, want string) {
		t.Helper()
		dl := time.Now().Add(5 * time.Second)
		for time.Now().Before(dl) {
			var turn model.ChatTurn
			if gdb.Where("id = ?", turnID).First(&turn).Error == nil && turn.Status == want {
				return
			}
			time.Sleep(30 * time.Millisecond)
		}
		t.Fatalf("轮次 %s 未到达终态 %s", turnID, want)
	}
	waitStatus(turn1, "cancelled")
	waitStatus(turn2, "done")

	// 两轮的用户消息与 assistant 消息成对落库
	var userCnt, assistantCnt int64
	gdb.Model(&model.ChatMessage{}).Where("session_id = ? AND role = ?", sess.ID, "user").Count(&userCnt)
	gdb.Model(&model.ChatMessage{}).Where("session_id = ? AND role = ?", sess.ID, "assistant").Count(&assistantCnt)
	if userCnt != 2 || assistantCnt != 2 {
		t.Fatalf("消息数不对: user=%d assistant=%d, want 2/2", userCnt, assistantCnt)
	}
}
