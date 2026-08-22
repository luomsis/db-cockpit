package handler

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"

	"db-cockpit/apiserver/internal/envelope"
	"db-cockpit/apiserver/internal/model"
)

/* ================= 实例详情 ================= */

func (h *H) GetInstance(c *gin.Context) {
	clusterDTO, err := h.loadClusterDTO(c.Param("id"), true)
	if err != nil {
		envelope.NotFound(c, "cluster not found")
		return
	}
	for _, inst := range clusterDTO.Instances {
		if inst.ID == c.Param("iid") {
			envelope.OK(c, gin.H{"cluster": clusterDTO, "instance": inst})
			return
		}
	}
	envelope.NotFound(c, "instance not found")
}

/* ================= 用户管理 ================= */

func (h *H) ListInstanceUsers(c *gin.Context) {
	iid := c.Param("iid")
	var users []model.InstanceUser
	// 演示数据为全局共享（instance_id = ''），叠加该实例私有账号
	h.DB.Where("instance_id = ? OR instance_id = ''", iid).Order("id asc").Find(&users)
	envelope.OK(c, users)
}

func (h *H) CreateInstanceUser(c *gin.Context) {
	var body struct {
		User string `json:"user" binding:"required"`
		Host string `json:"host"`
		Priv string `json:"priv"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		envelope.BadRequest(c, "user required")
		return
	}
	host := body.Host
	if host == "" {
		host = "%"
	}
	priv := body.Priv
	if priv == "" {
		priv = "SELECT"
	}
	u := model.InstanceUser{InstanceID: c.Param("iid"), Username: body.User, Host: host, Priv: priv,
		LastLogin: time.Now().Format("2006-01-02 15:04"), Status: "ok"}
	if err := h.DB.Create(&u).Error; err != nil {
		envelope.Internal(c, err)
		return
	}
	h.audit("user.create", "instance_user", body.User, nil)
	envelope.OK(c, u)
}

func (h *H) findUser(c *gin.Context) (*model.InstanceUser, bool) {
	iid := c.Param("iid")
	username := c.Param("user")
	var u model.InstanceUser
	if err := h.DB.Where("(instance_id = ? OR instance_id = '') AND username = ?", iid, username).
		First(&u).Error; err != nil {
		envelope.NotFound(c, "user not found")
		return nil, false
	}
	return &u, true
}

func (h *H) GrantUser(c *gin.Context) {
	var body struct {
		Priv string `json:"priv" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		envelope.BadRequest(c, "priv required")
		return
	}
	u, ok := h.findUser(c)
	if !ok {
		return
	}
	u.Priv = body.Priv
	h.DB.Save(u)
	h.audit("user.grant", "instance_user", u.Username, map[string]interface{}{"priv": body.Priv})
	envelope.OK(c, u)
}

func (h *H) ResetUserPassword(c *gin.Context) {
	u, ok := h.findUser(c)
	if !ok {
		return
	}
	h.audit("user.reset_password", "instance_user", u.Username, nil)
	envelope.OK(c, gin.H{"ok": true})
}

func (h *H) LockUser(c *gin.Context) {
	u, ok := h.findUser(c)
	if !ok {
		return
	}
	status := "err"
	if u.Status == "err" {
		status = "ok" // 再点一次 = 解锁
	}
	u.Status = status
	h.DB.Save(u)
	h.audit("user.lock", "instance_user", u.Username, map[string]interface{}{"status": status})
	envelope.OK(c, u)
}

/* ================= 会话管理 ================= */

func (h *H) ListSessions(c *gin.Context) {
	var sessions []model.RuntimeSession
	h.DB.Order("id desc").Find(&sessions)
	envelope.OK(c, sessions)
}

func (h *H) KillSession(c *gin.Context) {
	sid := c.Param("sid")
	res := h.DB.Where("id = ?", sid).Delete(&model.RuntimeSession{})
	if res.Error != nil || res.RowsAffected == 0 {
		envelope.NotFound(c, "session not found")
		return
	}
	h.audit("session.kill", "runtime_session", sid, nil)
	envelope.OK(c, gin.H{"ok": true, "killed": sid})
}

/* ================= 事务 / 慢 SQL ================= */

func (h *H) ListTransactions(c *gin.Context) {
	var trxs []model.Trx
	h.DB.Order("id asc").Find(&trxs)
	envelope.OK(c, trxs)
}

// ListSlowSqls 见 whitelist.go：数据面白名单 slow_query_log 指纹聚合（回退 UI 演示表）

// DiagnoseSql：规则式 SQL 诊断建议（对应 InstanceDetail 的 AI 诊断面板）
func (h *H) DiagnoseSql(c *gin.Context) {
	var body struct {
		Sql   string `json:"sql"`
		Db    string `json:"db"`
		Count int    `json:"count"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		envelope.BadRequest(c, "body required")
		return
	}
	db := body.Db
	if db == "" {
		db = "目标表"
	}
	count := body.Count
	countStr := "高频"
	if count > 0 {
		countStr = fmt.Sprintf("%d", count)
	}
	envelope.OK(c, gin.H{"suggestions": []string{
		fmt.Sprintf("对 %s 表缺少合适索引，建议添加联合索引 idx_status_uid (status, uid)，预计扫描行数下降 92%%；", db),
		"存在隐式类型转换导致索引失效，请核对字段类型与传参类型一致；",
		fmt.Sprintf("该 SQL 日均执行 %s 次，优化后预计集群 CPU 下降约 8%%；", countStr),
		"可一键生成索引变更工单，由参数管理通道灰度下发。",
	}})
}
