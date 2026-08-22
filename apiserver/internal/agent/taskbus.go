package agent

import (
	"context"
	"log"
	"sync"
	"time"

	"gorm.io/gorm"

	"db-cockpit/apiserver/internal/model"
)

/* ================= 任务表契约的 Go 侧：轮询 agent_tasks（依赖规则②，取代 tasks API 与 wake 回调） =================
 * agent 专家 INSERT 提交、agent worker 认领执行并推进度（claimed_by/lease_until）；
 * apiserver 本轮询器：
 *   1) running 进度变化 → ProgressEvent 直发会话总线（D13：不经 agent）；
 *   2) done 且未消费   → 创建 system_resume 轮（resume_of=task_id）并启动 exec 续跑（二次推理）；
 *   3) failed 且未消费  → progress 事件通知失败（不续跑，用户可重新发起）。
 * 取消级联：handler 对 cancel_requested 限列写，agent worker 观察后自行中止。
 * Go 对本表仅有的写操作：cancel_requested / notified / updated_at（限列写约定）。
 */

type TaskBus struct {
	db           *gorm.DB
	rt           *Runtime
	interval     time.Duration
	mu           sync.Mutex
	lastProgress map[string]float64
}

func NewTaskBus(gdb *gorm.DB, rt *Runtime) *TaskBus {
	return &TaskBus{db: gdb, rt: rt, interval: 2 * time.Second, lastProgress: map[string]float64{}}
}

// Run：进程生命周期内轮询（main 挂 goroutine）。
func (b *TaskBus) Run(ctx context.Context) {
	tk := time.NewTicker(b.interval)
	defer tk.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tk.C:
			b.PollOnce()
		}
	}
}

// PollOnce：单次轮询（进度注入 + 终态消费），独立可测。
func (b *TaskBus) PollOnce() {
	b.injectProgress()
	b.consumeFinished()
}

func (b *TaskBus) injectProgress() {
	var running []model.AgentTask
	b.db.Where("status = ?", "running").Find(&running)
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, task := range running {
		if b.lastProgress[task.ID] == task.Progress {
			continue // 进度未变化不重发（轮询侧幂等）
		}
		b.lastProgress[task.ID] = task.Progress
		b.rt.publish(task.SessionID, task.TurnID, ProgressEvent{
			Type: "progress", TaskID: task.ID, Progress: task.Progress, Stage: task.Stage,
		})
	}
}

func (b *TaskBus) consumeFinished() {
	var fin []model.AgentTask
	b.db.Where("status IN ? AND notified = ?", []string{"done", "failed"}, false).Find(&fin)
	for _, task := range fin {
		if task.Status == "done" {
			b.startResumeTurn(task)
		} else {
			b.rt.publish(task.SessionID, task.TurnID, ProgressEvent{
				Type: "progress", TaskID: task.ID, Progress: 100,
				Stage: "任务失败：" + task.Error + "（可重新发起）",
			})
		}
		b.db.Model(&model.AgentTask{}).Where("id = ?", task.ID).
			Updates(map[string]interface{}{"notified": true, "updated_at": nowMillis()})
	}
}

// startResumeTurn：任务完成 → 创建 system_resume 轮并启动 exec 续跑（二次推理，docs《交互时序与生命周期》§4）。
func (b *TaskBus) startResumeTurn(task model.AgentTask) {
	var sess model.ChatSession
	if err := b.db.Where("id = ?", task.SessionID).First(&sess).Error; err != nil {
		log.Printf("[taskbus] 任务 %s 完成但会话不存在: %v", task.ID, err)
		return
	}
	turnID := NewID("turn")
	now := nowMillis()
	var turnSeq int
	b.db.Model(&model.ChatTurn{}).Where("session_id = ?", sess.ID).
		Select("COALESCE(MAX(seq), 0)").Scan(&turnSeq)
	if err := b.db.Create(&model.ChatTurn{
		ID: turnID, SessionID: sess.ID, Seq: turnSeq + 1, Kind: "system_resume",
		ResumeOf: task.ID, Status: "running", UserMsg: task.ToolName,
		CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		log.Printf("[taskbus] 创建续跑轮次失败: %v", err)
		return
	}
	b.rt.publish(sess.ID, turnID, ProgressEvent{
		Type: "progress", TaskID: task.ID, Progress: 100, Stage: "完成，正在汇总结论…",
	})
	b.rt.StartTurn(sess.ID, turnID,
		"任务 "+task.CallID+"（"+task.ToolName+"）已完成，结果引用 "+task.ResultRef+
			"。请基于该任务结果与会话既有证据进行二次推理并输出最终结论。")
}
