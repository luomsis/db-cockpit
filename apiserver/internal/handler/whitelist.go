package handler

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"db-cockpit/apiserver/internal/envelope"
	"db-cockpit/apiserver/internal/model"
)

/* ================= 数据面白名单表消费（§6.1.2 · collector 直写 / apiserver 只读聚合） =================
告警/慢SQL 在消费端聚合出前端视图；P1 映射、Issue 化（fingerprint/状态机）后续在控制面 Issue 域实现。 */

/* ---- 告警：原始事件 → 聚合视图（告警中心 / 大盘共用） ---- */

type alertItem struct {
	Name     string `json:"name"`
	Severity string `json:"severity"`
	Title    string `json:"title"`
	Time     string `json:"time"`
	Count    int64  `json:"count"`
}

// mapSeverity 源级别 → 平台级别：Critical→P1、Major→P2、其余（Minor/Warning/未知/空）→P3
func mapSeverity(level string) string {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "critical":
		return "P1"
	case "major":
		return "P2"
	default:
		return "P3"
	}
}

type alertAgg struct {
	ObjectName string    `gorm:"column:object_name"`
	AlertName  string    `gorm:"column:alert_name"`
	AlertLevel string    `gorm:"column:alert_level"`
	FirstFired time.Time `gorm:"column:first_fired"`
	Cnt        int64     `gorm:"column:cnt"`
}

// buildAlertItems 聚合行 → 前端形状；按级别升序（P1<P2<P3）、首触时间倒序
func buildAlertItems(aggs []alertAgg) []alertItem {
	sort.SliceStable(aggs, func(i, j int) bool {
		si, sj := mapSeverity(aggs[i].AlertLevel), mapSeverity(aggs[j].AlertLevel)
		if si != sj {
			return si < sj
		}
		return aggs[i].FirstFired.After(aggs[j].FirstFired)
	})
	items := make([]alertItem, 0, len(aggs))
	for _, a := range aggs {
		items = append(items, alertItem{
			Name: a.ObjectName, Severity: mapSeverity(a.AlertLevel), Title: a.AlertName,
			Time: a.FirstFired.Format("01-02 15:04"), Count: a.Cnt,
		})
	}
	return items
}

// legacyAlertItems 白名单空表时回退 UI 演示表（老部署兼容）
func legacyAlertItems(rows []model.AlertRecord) []alertItem {
	items := make([]alertItem, 0, len(rows))
	for _, r := range rows {
		items = append(items, alertItem{Name: r.Name, Severity: r.Severity, Title: r.Title, Time: r.Time, Count: int64(r.Count)})
	}
	return items
}

func (h *H) alertItems() []alertItem {
	var n int64
	if err := h.DB.Model(&model.AlertRaw{}).Count(&n).Error; err != nil || n == 0 {
		var rows []model.AlertRecord
		h.DB.Order("id asc").Find(&rows)
		return legacyAlertItems(rows)
	}
	var aggs []alertAgg
	h.DB.Model(&model.AlertRaw{}).
		Select("object_name, alert_name, alert_level, min(fired_at) as first_fired, count(*) as cnt").
		Group("object_name, alert_name, alert_level").
		Find(&aggs)
	return buildAlertItems(aggs)
}

// GetAlerts 告警中心列表（顶栏铃铛同源）
func (h *H) GetAlerts(c *gin.Context) {
	items := h.alertItems()
	envelope.OK(c, gin.H{"items": items, "total": len(items)})
}

/* ---- 慢SQL：原始执行事件 → 指纹聚合视图（实例详情 SQL 诊断 / 大盘共用） ---- */

type slowItem struct {
	Sql   string `json:"sql"`
	Db    string `json:"db"`
	Time  string `json:"time"`
	Rows  string `json:"rows"`
	Count int64  `json:"count"`
}

type slowAgg struct {
	SqlText string  `gorm:"column:sql_text"`
	Db      string  `gorm:"column:database_name"`
	AvgMs   float64 `gorm:"column:avg_ms"`
	MaxRows *int64  `gorm:"column:max_rows"`
	Cnt     int64   `gorm:"column:cnt"`
}

// fmtMs 毫秒 → 展示：≥1s 记 "12.8s"，否则 "950ms"
func fmtMs(ms float64) string {
	if ms >= 1000 {
		return fmt.Sprintf("%.1fs", ms/1000)
	}
	return fmt.Sprintf("%dms", int64(ms))
}

// fmtRows 扫描行数千分位展示；NULL（源系统缺失）→ "—"
func fmtRows(v *int64) string {
	if v == nil {
		return "—"
	}
	s := strconv.FormatInt(*v, 10)
	sign := ""
	if *v < 0 {
		sign, s = "-", s[1:]
	}
	if len(s) <= 3 {
		return sign + s
	}
	var out []byte
	for i := 0; i < len(s); i++ {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, s[i])
	}
	return sign + string(out)
}

// buildSlowItems 聚合行（按平均耗时倒序来自 SQL）→ 前端形状
func buildSlowItems(aggs []slowAgg) []slowItem {
	items := make([]slowItem, 0, len(aggs))
	for _, a := range aggs {
		items = append(items, slowItem{Sql: a.SqlText, Db: a.Db, Time: fmtMs(a.AvgMs), Rows: fmtRows(a.MaxRows), Count: a.Cnt})
	}
	return items
}

func legacySlowItems(rows []model.SlowSql) []slowItem {
	items := make([]slowItem, 0, len(rows))
	for _, r := range rows {
		items = append(items, slowItem{Sql: r.Sql, Db: r.Db, Time: r.Time, Rows: r.Rows, Count: int64(r.Count)})
	}
	return items
}

func (h *H) slowItems(instanceID *int64) []slowItem {
	var n int64
	if err := h.DB.Model(&model.SlowQueryLog{}).Count(&n).Error; err != nil || n == 0 {
		var rows []model.SlowSql
		h.DB.Order("id asc").Find(&rows)
		return legacySlowItems(rows)
	}
	q := h.DB.Model(&model.SlowQueryLog{}).
		Select("min(sql_text) as sql_text, min(database_name) as database_name, avg(execute_ms)::float8 as avg_ms, max(rows_examined) as max_rows, count(*) as cnt").
		Group("digest").Order("avg_ms desc")
	if instanceID != nil {
		q = q.Where("instance_id = ?", *instanceID)
	}
	var aggs []slowAgg
	if err := q.Find(&aggs).Error; err != nil {
		return nil
	}
	return buildSlowItems(aggs)
}

// resolveMetaInstance UI 演示实例 → 元数据域 db_component（v2 三级链）：
// ① 名称精确匹配（引擎/代理成员）；② host_ip 匹配（成员）；
// ③ OB 链式：按 host_ip 定位 observer 成员 → extensions.units 含该成员的租户逻辑单元。
// 解析失败返回 nil（调用方回退全局）。
func (h *H) resolveMetaInstance(iid string) *int64 {
	var inst model.Instance
	if err := h.DB.Where("id = ?", iid).First(&inst).Error; err != nil {
		return nil
	}
	var m model.DbComponent
	err := h.DB.Where("name = ?", inst.Name).Order("id asc").First(&m).Error
	if err != nil {
		err = h.DB.Where("host_ip = ?", inst.IP).Order("id asc").First(&m).Error
	}
	if err != nil {
		var obs model.DbComponent
		if h.DB.Where("host_ip = ?", inst.IP).Order("id asc").First(&obs).Error != nil {
			return nil
		}
		var tenants []model.DbComponent
		h.DB.Where("kind = 'tenant'").Find(&tenants)
		for _, t := range tenants {
			var ext struct {
				Units []struct {
					InstanceID int64 `json:"instance_id"`
				} `json:"units"`
			}
			if json.Unmarshal(t.Extensions, &ext) != nil {
				continue
			}
			for _, u := range ext.Units {
				if u.InstanceID == obs.ID {
					id := t.ID
					return &id
				}
			}
		}
		return nil
	}
	id := m.ID
	return &id
}

// ListSlowSqls 实例详情「SQL 诊断」：按实例过滤的指纹聚合；实例解析失败回退全局
func (h *H) ListSlowSqls(c *gin.Context) {
	envelope.OK(c, h.slowItems(h.resolveMetaInstance(c.Param("iid"))))
}

/* ---- 大盘：库类型活计数（db_cluster / db_instance） ---- */

type dbTypeRow struct {
	Type  string `json:"type"`
	Name  string `json:"name"`
	Icon  string `json:"icon"`
	Total int64  `json:"total"`
	Alert int64  `json:"alert"`
}

var dbTypeMeta = map[string][2]string{
	"pg":        {"PostgreSQL", "🐘"},
	"oceanbase": {"OceanBase", "🌊"},
	"mysql":     {"MySQL", "🐬"},
	"redis":     {"Redis", "🔴"},
	"mongodb":   {"MongoDB", "🍃"},
}

// dbTypes 元数据域活计数（v2：db_cluster/db_component，storage 成员数即「实例数」）；db_cluster 为空返回 nil（调用方回退 meta_stats 静态值）
func (h *H) dbTypes() []dbTypeRow {
	var clusterN int64
	if err := h.DB.Model(&model.DbCluster{}).Count(&clusterN).Error; err != nil || clusterN == 0 {
		return nil
	}
	var totals []struct {
		DbType string `gorm:"column:db_type"`
		Cnt    int64  `gorm:"column:cnt"`
	}
	h.DB.Model(&model.DbComponent{}).
		Select("c.db_type AS db_type, count(*) AS cnt").
		Joins("JOIN db_cluster c ON c.id = db_component.cluster_id").
		Where("db_component.kind = ?", "storage").
		Group("c.db_type").Find(&totals)

	var alerts []struct {
		DbType string `gorm:"column:db_type"`
		Cnt    int64  `gorm:"column:cnt"`
	}
	// 活跃告警数：聚合行（对象+标题+级别去重）按 cluster 关联到库类型；未关联对象不计入
	h.DB.Raw(`SELECT c.db_type AS db_type, count(*) AS cnt
		FROM (SELECT DISTINCT cluster_id, object_name, alert_name, alert_level FROM alert_raw WHERE cluster_id IS NOT NULL) a
		JOIN db_cluster c ON c.id = a.cluster_id
		GROUP BY c.db_type`).Scan(&alerts)

	alertByType := map[string]int64{}
	for _, a := range alerts {
		alertByType[a.DbType] = a.Cnt
	}
	rows := make([]dbTypeRow, 0, len(totals))
	for _, t := range totals {
		meta := dbTypeMeta[t.DbType]
		name, icon := meta[0], meta[1]
		if name == "" {
			name, icon = t.DbType, "🗄️"
		}
		rows = append(rows, dbTypeRow{Type: t.DbType, Name: name, Icon: icon, Total: t.Cnt, Alert: alertByType[t.DbType]})
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].Total != rows[j].Total {
			return rows[i].Total > rows[j].Total
		}
		return rows[i].Type < rows[j].Type
	})
	return rows
}

/* ---- 变更工单 ---- */

// changeStatusLabels 源系统数字枚举 → 展示语义（消费端映射，白名单表存原值）
var changeStatusLabels = map[int]string{0: "待提交", 1: "审批中", 2: "执行中", 3: "已完成", 4: "已取消"}

func statusLabel(code int) string {
	if s, ok := changeStatusLabels[code]; ok {
		return s
	}
	return "状态" + strconv.Itoa(code)
}

type changeItem struct {
	model.ChangeTicket
	StatusLabel string `json:"statusLabel"`
}

// parseTimeParam 时间参数：RFC3339 / "2006-01-02 15:04" / "2006-01-02"。
// 容错：query 未编码的 RFC3339 +08:00 时区，"+" 会被解析为空格，补一次替换重试。
func parseTimeParam(v string) (time.Time, error) {
	layouts := []string{time.RFC3339, "2006-01-02 15:04", "2006-01-02"}
	try := func(s string) (time.Time, bool) {
		for _, layout := range layouts {
			if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
				return t, true
			}
		}
		return time.Time{}, false
	}
	if t, ok := try(v); ok {
		return t, nil
	}
	if t, ok := try(strings.Replace(v, " ", "+", 1)); ok {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("bad time %q", v)
}

// ListChanges 变更工单列表：risk_level / cluster_id / instance_id / from,to（执行开始时间闭区间）/ limit
func (h *H) ListChanges(c *gin.Context) {
	q := h.DB.Model(&model.ChangeTicket{})
	if v := c.Query("risk_level"); v != "" {
		q = q.Where("risk_level = ?", v)
	}
	for _, p := range []struct{ key, col string }{
		{"cluster_id", "cluster_id"}, {"instance_id", "instance_id"},
	} {
		if v := c.Query(p.key); v != "" {
			id, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				envelope.BadRequest(c, "invalid "+p.key)
				return
			}
			q = q.Where(p.col+" = ?", id)
		}
	}
	for _, p := range []struct{ key, cond string }{
		{"from", "execute_start_at >= ?"}, {"to", "execute_start_at <= ?"},
	} {
		if v := c.Query(p.key); v != "" {
			t, err := parseTimeParam(v)
			if err != nil {
				envelope.BadRequest(c, "invalid "+p.key+": expect RFC3339 / 2006-01-02 15:04 / 2006-01-02")
				return
			}
			q = q.Where(p.cond, t)
		}
	}
	limit := 100
	if v := c.Query("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 500 {
			envelope.BadRequest(c, "invalid limit: 1-500")
			return
		}
		limit = n
	}
	var rows []model.ChangeTicket
	if err := q.Order("execute_start_at DESC NULLS LAST").Limit(limit).Find(&rows).Error; err != nil {
		envelope.Internal(c, err)
		return
	}
	items := make([]changeItem, 0, len(rows))
	for i := range rows {
		items = append(items, changeItem{ChangeTicket: rows[i], StatusLabel: statusLabel(rows[i].StatusCode)})
	}
	envelope.OK(c, gin.H{"items": items, "total": len(items)})
}

/* ---- 元数据域下钻（§6.1.1 v2：集群 → 组件成员 → 主机） ---- */

type metaClusterItem struct {
	*model.DbCluster
	ComponentCount int64 `json:"componentCount"` // 成员数（含 proxy/租户逻辑单元）
	HostCount      int64 `json:"hostCount"`      // 去重主机数
}

// ListMetaClusters 元数据域集群列表（db_type / environment 过滤 + 成员/主机计数）
func (h *H) ListMetaClusters(c *gin.Context) {
	q := h.DB.Model(&model.DbCluster{})
	if v := c.Query("db_type"); v != "" {
		q = q.Where("db_type = ?", v)
	}
	if v := c.Query("environment"); v != "" {
		q = q.Where("environment = ?", v)
	}
	var clusters []*model.DbCluster
	if err := q.Order("id asc").Find(&clusters).Error; err != nil {
		envelope.Internal(c, err)
		return
	}
	var compCnt []struct {
		ClusterID int64 `gorm:"column:cluster_id"`
		Cnt       int64 `gorm:"column:cnt"`
	}
	h.DB.Model(&model.DbComponent{}).Select("cluster_id, count(*) as cnt").Group("cluster_id").Find(&compCnt)
	var hostCnt []struct {
		ClusterID int64 `gorm:"column:cluster_id"`
		Cnt       int64 `gorm:"column:cnt"`
	}
	h.DB.Raw(`SELECT cluster_id, count(DISTINCT host_ip) AS cnt
		FROM db_component WHERE host_ip IS NOT NULL AND host_ip <> ''
		GROUP BY cluster_id`).Scan(&hostCnt)
	compBy, hostBy := map[int64]int64{}, map[int64]int64{}
	for _, r := range compCnt {
		compBy[r.ClusterID] = r.Cnt
	}
	for _, r := range hostCnt {
		hostBy[r.ClusterID] = r.Cnt
	}
	items := make([]metaClusterItem, 0, len(clusters))
	for _, cl := range clusters {
		items = append(items, metaClusterItem{DbCluster: cl, ComponentCount: compBy[cl.ID], HostCount: hostBy[cl.ID]})
	}
	envelope.OK(c, gin.H{"items": items, "total": len(items)})
}

// GetMetaCluster 集群详情下钻（v2 形状，对齐前端拓扑 TopoScenario）：
// {cluster（含端点）, components[]（kind/group_name/双上游字段/extensions.units）, hosts[]（三级位置）}
// 三类拓扑由此推导：逻辑包含 = cluster→components(kind/group_name 分组)；数据流 = traffic 链 + cluster.endpoint；
// 复制 = replication 链；物理位置 = components JOIN hosts 按 region/az/host_cluster 分组。
func (h *H) GetMetaCluster(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		envelope.NotFound(c, "cluster not found")
		return
	}
	var cl model.DbCluster
	if err := h.DB.First(&cl, id).Error; err != nil {
		envelope.NotFound(c, "cluster not found")
		return
	}
	var components []*model.DbComponent
	if err := h.DB.Where("cluster_id = ?", id).Order("kind asc, id asc").Find(&components).Error; err != nil {
		envelope.Internal(c, err)
		return
	}
	hostIPs := make([]string, 0, len(components))
	seen := map[string]bool{}
	for _, m := range components {
		if m.HostIP != "" && !seen[m.HostIP] {
			seen[m.HostIP] = true
			hostIPs = append(hostIPs, m.HostIP)
		}
	}
	hosts := []model.DbHost{}
	if len(hostIPs) > 0 {
		if err := h.DB.Where("host_ip IN ?", hostIPs).Order("host_ip asc").Find(&hosts).Error; err != nil {
			envelope.Internal(c, err)
			return
		}
	}
	envelope.OK(c, gin.H{"cluster": cl, "components": components, "hosts": hosts})
}
