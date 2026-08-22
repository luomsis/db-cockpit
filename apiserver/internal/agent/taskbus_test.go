package agent

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"db-cockpit/apiserver/internal/model"
)

// 场景 1：任务 done 且未消费 → 创建 system_resume 轮（resume_of 正确）并经 exec 续跑到终态，notified 置位。
func TestTaskBusResumeOnDone(t *testing.T) {
	gdb := testDB(t)
	up := httptest.NewServer(sseHandler(func(w http.ResponseWriter, flush func()) {
		fmt.Fprint(w, "data: {\"type\":\"token\",\"text_delta\":\"结论：根因已确认\"}\n\n")
		fmt.Fprint(w, "data: {\"type\":\"done\",\"usage\":{\"prompt_tokens\":9,\"completion_tokens\":7}}\n\n")
		flush()
	}))
	defer up.Close()

	sessID, turnID := newSessionTurn(t, gdb)
	gdb.Model(&model.ChatTurn{}).Where("id = ?", turnID).Update("status", "done") // 原轮已完成
	task := model.AgentTask{
		ID: NewID("atask"), SessionID: sessID, TurnID: turnID, SubagentID: "diag_subagent",
		CallID: NewID("call"), ToolName: "builtin_metric_deep_scan",
		Status: "done", ResultRef: "ref://scan/1", Progress: 100,
		CreatedAt: nowMillis(), UpdatedAt: nowMillis(),
	}
	if err := gdb.Create(&task).Error; err != nil {
		t.Fatalf("建任务失败: %v", err)
	}

	rt := NewRuntime(gdb, NewUpstreamSource(up.URL, 0))
	NewTaskBus(gdb, rt).PollOnce()

	// notified 已置位（终态被消费）
	var after model.AgentTask
	gdb.Where("id = ?", task.ID).First(&after)
	if !after.Notified {
		t.Fatalf("notified 未置位")
	}
	// system_resume 轮已创建并到达 done，resume_of 指向任务
	var resume model.ChatTurn
	if err := gdb.Where("session_id = ? AND kind = ?", sessID, "system_resume").First(&resume).Error; err != nil {
		t.Fatalf("续跑轮未创建: %v", err)
	}
	if resume.ResumeOf != task.ID {
		t.Fatalf("resume_of = %s, want %s", resume.ResumeOf, task.ID)
	}
	final := waitTurnFinal(t, gdb, resume.ID, 5*time.Second)
	if final.Status != "done" {
		t.Fatalf("续跑轮终态 = %s, want done", final.Status)
	}
	var msg model.ChatMessage
	if err := gdb.Where("turn_id = ? AND role = ?", resume.ID, "assistant").First(&msg).Error; err != nil || msg.Text != "结论：根因已确认" {
		t.Fatalf("续跑轮 assistant 装配错误: err=%v text=%q", err, msg.Text)
	}
}

// 场景 2：running 任务进度变化 → progress 事件注入会话总线；进度未变不重发（轮询幂等）。
func TestTaskBusProgressInject(t *testing.T) {
	gdb := testDB(t)
	sessID, turnID := newSessionTurn(t, gdb)
	task := model.AgentTask{
		ID: NewID("atask"), SessionID: sessID, TurnID: turnID, SubagentID: "diag_subagent",
		CallID: NewID("call"), ToolName: "builtin_metric_deep_scan",
		Status: "running", Progress: 40, Stage: "多指标长时窗扫描…",
		CreatedAt: nowMillis(), UpdatedAt: nowMillis(),
	}
	gdb.Create(&task)

	rt := NewRuntime(gdb, NewBuiltinSource())
	bus := NewTaskBus(gdb, rt)
	bus.PollOnce()
	var cnt int64
	gdb.Model(&model.ChatTurnEvent{}).Where("session_id = ?", sessID).Count(&cnt)
	if cnt != 1 {
		t.Fatalf("进度事件数 = %d, want 1", cnt)
	}
	// 进度未变化：不重发
	bus.PollOnce()
	gdb.Model(&model.ChatTurnEvent{}).Where("session_id = ?", sessID).Count(&cnt)
	if cnt != 1 {
		t.Fatalf("进度未变仍重发: %d", cnt)
	}
	// 进度推进：再发一条
	gdb.Model(&model.AgentTask{}).Where("id = ?", task.ID).
		Updates(map[string]interface{}{"progress": 80.0, "stage": "根因假设生成…", "updated_at": nowMillis()})
	bus.PollOnce()
	gdb.Model(&model.ChatTurnEvent{}).Where("session_id = ?", sessID).Count(&cnt)
	if cnt != 2 {
		t.Fatalf("进度推进后事件数 = %d, want 2", cnt)
	}
}

// 场景 3：任务 failed → progress 失败通知（不续跑），notified 置位。
func TestTaskBusFailedNotify(t *testing.T) {
	gdb := testDB(t)
	sessID, turnID := newSessionTurn(t, gdb)
	task := model.AgentTask{
		ID: NewID("atask"), SessionID: sessID, TurnID: turnID, SubagentID: "diag_subagent",
		CallID: NewID("call"), ToolName: "builtin_metric_deep_scan",
		Status: "failed", Error: "扫描超时",
		CreatedAt: nowMillis(), UpdatedAt: nowMillis(),
	}
	gdb.Create(&task)

	rt := NewRuntime(gdb, NewBuiltinSource())
	NewTaskBus(gdb, rt).PollOnce()

	var after model.AgentTask
	gdb.Where("id = ?", task.ID).First(&after)
	if !after.Notified {
		t.Fatalf("notified 未置位")
	}
	var cnt int64
	gdb.Model(&model.ChatTurn{}).Where("session_id = ? AND kind = ?", sessID, "system_resume").Count(&cnt)
	if cnt != 0 {
		t.Fatalf("失败任务不应触发续跑轮")
	}
	var evs []model.ChatTurnEvent
	gdb.Where("session_id = ?", sessID).Find(&evs)
	if len(evs) != 1 {
		t.Fatalf("失败通知事件数 = %d, want 1", len(evs))
	}
}
