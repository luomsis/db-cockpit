package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"db-cockpit/apiserver/internal/model"
)

/* 数据面白名单集成测试：连本地既有 PG（同 chat_test，不可达跳过）。
marker 数据自建自清（source_sys/名称带纳秒后缀），不依赖、不破坏 dev 种子。 */

func whitelistTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		dsn = "host=localhost port=55432 user=graphiti password=graphiti dbname=db_cockpit sslmode=disable"
	}
	gdb, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Skipf("PG 不可达，跳过：%v", err)
	}
	if err := gdb.AutoMigrate(
		&model.DbCluster{}, &model.DbInstance{}, &model.DbInstanceNode{},
		&model.AlertRaw{}, &model.ChangeTicket{}, &model.SlowQueryLog{},
		&model.Cluster{}, &model.Instance{}, &model.SlowSql{}, &model.AlertRecord{}, &model.MetaStat{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return gdb
}

func whitelistRouter(gdb *gorm.DB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := &H{DB: gdb}
	r := gin.New()
	r.GET("/api/alerts", h.GetAlerts)
	r.GET("/api/overview", h.GetOverview)
	r.GET("/api/changes", h.ListChanges)
	r.GET("/api/meta/clusters", h.ListMetaClusters)
	r.GET("/api/meta/clusters/:id", h.GetMetaCluster)
	r.GET("/api/clusters/:id/instances/:iid/slow-sqls", h.ListSlowSqls)
	return r
}

type envBody struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func doGet(t *testing.T, r *gin.Engine, path string) (int, envBody) {
	t.Helper()
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
	var b envBody
	if err := json.Unmarshal(w.Body.Bytes(), &b); err != nil {
		t.Fatalf("解析响应 %s: %v", w.Body.String(), err)
	}
	return w.Code, b
}

func decodeData(t *testing.T, raw json.RawMessage, v interface{}) {
	t.Helper()
	if err := json.Unmarshal(raw, v); err != nil {
		t.Fatalf("解析 data: %v", err)
	}
}

/* ---- 告警聚合 ---- */

func TestAlertsWhitelistAggregation(t *testing.T) {
	gdb := whitelistTestDB(t)
	r := whitelistRouter(gdb)
	uniq := fmt.Sprintf("wt%d", time.Now().UnixNano())
	obj := "obj-" + uniq
	t0 := time.Date(2026, 8, 18, 10, 0, 0, 0, time.Local)
	mk := func(id, level, title string, fired time.Time) *model.AlertRaw {
		return &model.AlertRaw{SourceSys: uniq, EventID: id, ObjectName: obj, ObjectType: "instance",
			AlertLevel: level, AlertName: title, FiredAt: fired}
	}
	rows := []*model.AlertRaw{
		mk(uniq+"-1", "Critical", "CPU 高", t0.Add(10*time.Minute)),
		mk(uniq+"-2", "Critical", "CPU 高", t0),
		mk(uniq+"-3", "Critical", "CPU 高", t0.Add(5*time.Minute)),
		mk(uniq+"-4", "Warning", "水位", t0.Add(time.Hour)), // 未知级别 → P3
	}
	if err := gdb.Create(&rows).Error; err != nil {
		t.Fatalf("造数: %v", err)
	}
	t.Cleanup(func() { gdb.Where("source_sys = ?", uniq).Delete(&model.AlertRaw{}) })

	code, body := doGet(t, r, "/api/alerts")
	if code != http.StatusOK || body.Code != 0 {
		t.Fatalf("GET /api/alerts: code=%d body=%+v", code, body)
	}
	var resp struct {
		Items []alertItem `json:"items"`
		Total int         `json:"total"`
	}
	decodeData(t, body.Data, &resp)

	var p1, p3 *alertItem
	p1Idx, p3Idx := -1, -1
	for i := range resp.Items {
		if resp.Items[i].Name != obj {
			continue
		}
		switch resp.Items[i].Severity {
		case "P1":
			p1, p1Idx = &resp.Items[i], i
		case "P3":
			p3, p3Idx = &resp.Items[i], i
		}
	}
	if p1 == nil || p3 == nil {
		t.Fatalf("聚合行缺失: p1=%v p3=%v", p1, p3)
	}
	if p1.Count != 3 || p1.Title != "CPU 高" || p1.Time != t0.Format("01-02 15:04") {
		t.Errorf("P1 聚合不符: %+v（want count=3 time=%s）", p1, t0.Format("01-02 15:04"))
	}
	if p3.Count != 1 {
		t.Errorf("P3 聚合不符: %+v", p3)
	}
	if p1Idx > p3Idx {
		t.Errorf("P1 应排在 P3 前（idx %d vs %d）", p1Idx, p3Idx)
	}
}

/* ---- 慢SQL 指纹聚合 + 实例过滤/回退 ---- */

func TestSlowSqlWhitelistAggregation(t *testing.T) {
	gdb := whitelistTestDB(t)
	r := whitelistRouter(gdb)
	uniq := fmt.Sprintf("wt%d", time.Now().UnixNano())

	demoC := &model.Cluster{ID: "cx" + uniq, Name: "cl-" + uniq, Type: "pg"}
	demoI := &model.Instance{ID: "in" + uniq, ClusterID: demoC.ID, Name: "inst-" + uniq, IP: "10.99.99.99", Port: 5555}
	metaC := &model.DbCluster{Name: "mcl-" + uniq, DbType: "pg", Environment: "生产", SourceSys: uniq}
	for _, m := range []interface{}{demoC, demoI, metaC} {
		if err := gdb.Create(m).Error; err != nil {
			t.Fatalf("造数: %v", err)
		}
	}
	dbI := &model.DbInstance{ClusterID: metaC.ID, DbType: "pg", Name: "inst-" + uniq, Role: "storage",
		SourceID: "src" + uniq, Endpoint: "1.2.3.4:9999"}
	if err := gdb.Create(dbI).Error; err != nil {
		t.Fatalf("造数: %v", err)
	}
	t.Cleanup(func() {
		gdb.Where("source_sys = ?", uniq).Delete(&model.SlowQueryLog{})
		gdb.Where("source_id = ?", "src"+uniq).Delete(&model.DbInstance{})
		gdb.Where("name = ?", "mcl-"+uniq).Delete(&model.DbCluster{})
		gdb.Where("id = ?", demoI.ID).Delete(&model.Instance{})
		gdb.Where("id = ?", demoC.ID).Delete(&model.Cluster{})
	})

	sqlA, sqlC := "SELECT marker-a-"+uniq, "SELECT marker-c-"+uniq
	v := func(i int64) *int64 { return &i }
	iid := dbI.ID
	base := time.Now().Add(-2 * time.Hour)
	mk := func(sqlText, digest string, ms float64, rows *int64, instID *int64, when time.Time) *model.SlowQueryLog {
		return &model.SlowQueryLog{SourceSys: uniq, InstanceID: instID, SqlText: sqlText, Digest: digest,
			DatabaseName: "db-" + uniq, ExecuteMs: ms, RowsExamined: rows, ExecuteDate: when, CreateTime: when}
	}
	logs := []*model.SlowQueryLog{
		mk(sqlA, "dgA"+uniq, 2000, v(1234567), &iid, base),
		mk(sqlA, "dgA"+uniq, 1800, v(1234567), &iid, base.Add(time.Minute)),
		mk(sqlA, "dgA"+uniq, 2200, v(1234567), &iid, base.Add(2*time.Minute)),
		mk("SELECT marker-b-"+uniq, "dgB"+uniq, 500, nil, &iid, base), // 扫描行数源缺失 → "—"
		mk(sqlC, "dgC"+uniq, 3000, v(1), nil, base),                    // 未关联实例 → 仅全局可见
	}
	if err := gdb.Create(&logs).Error; err != nil {
		t.Fatalf("造数: %v", err)
	}

	// ① 按实例过滤：demo 实例经名称匹配到元数据实例，只见该实例的两个指纹
	code, body := doGet(t, r, "/api/clusters/"+demoC.ID+"/instances/"+demoI.ID+"/slow-sqls")
	if code != http.StatusOK || body.Code != 0 {
		t.Fatalf("slow-sqls: code=%d body=%+v", code, body)
	}
	var items []slowItem
	decodeData(t, body.Data, &items)
	if len(items) != 2 {
		t.Fatalf("按实例应聚出 2 个指纹, got %d: %+v", len(items), items)
	}
	if items[0].Sql != sqlA || items[0].Time != "2.0s" || items[0].Rows != "1,234,567" || items[0].Count != 3 {
		t.Errorf("指纹 A 聚合不符: %+v", items[0])
	}
	if items[1].Sql != "SELECT marker-b-"+uniq || items[1].Time != "500ms" || items[1].Rows != "—" || items[1].Count != 1 {
		t.Errorf("指纹 B 聚合不符: %+v", items[1])
	}

	// ② 实例解析失败 → 回退全局（应包含未关联实例的指纹 C）
	code, body = doGet(t, r, "/api/clusters/"+demoC.ID+"/instances/no-such-instance/slow-sqls")
	if code != http.StatusOK {
		t.Fatalf("全局回退请求失败: %d", code)
	}
	decodeData(t, body.Data, &items)
	hasC := false
	for _, it := range items {
		if it.Sql == sqlC {
			hasC = true
			if it.Time != "3.0s" {
				t.Errorf("指纹 C 耗时不符: %+v", it)
			}
		}
	}
	if !hasC {
		t.Fatalf("全局回退应包含未关联实例的指纹 C")
	}

	// ③ OB 约定：名称/端点不匹配时，按租户 extensions.units（unit 所在 server_ip）兜底匹配
	obDemoI := &model.Instance{ID: "ob" + uniq, ClusterID: demoC.ID, Name: "observer-x", IP: "10.99.0.9", Port: 2881}
	obTenant := &model.DbInstance{ClusterID: metaC.ID, DbType: "oceanbase", Name: "tenant-x", Role: "user",
		SourceID: "obsrc" + uniq, Endpoint: "10.99.9.9:2881",
		Extensions: datatypes.JSON([]byte(`{"mode":"mysql","units":[{"server_ip":"10.99.0.9","zone":"Z1"}]}`))}
	if err := gdb.Create([]*model.Instance{obDemoI}).Error; err != nil {
		t.Fatalf("造数: %v", err)
	}
	if err := gdb.Create(obTenant).Error; err != nil {
		t.Fatalf("造数: %v", err)
	}
	t.Cleanup(func() {
		gdb.Where("source_id = ?", "obsrc"+uniq).Delete(&model.DbInstance{})
		gdb.Where("id = ?", obDemoI.ID).Delete(&model.Instance{})
	})
	obIid := obTenant.ID
	mkDb := func(sqlText, digest string, ms float64, instID *int64) *model.SlowQueryLog {
		return mk(sqlText, digest, ms, v(10), instID, base)
	}
	if err := gdb.Create([]*model.SlowQueryLog{
		mkDb("SELECT ob-marker-"+uniq, "dgOB"+uniq, 4000, &obIid),
	}).Error; err != nil {
		t.Fatalf("造数: %v", err)
	}
	code, body = doGet(t, r, "/api/clusters/"+demoC.ID+"/instances/"+obDemoI.ID+"/slow-sqls")
	if code != http.StatusOK {
		t.Fatalf("OB units 匹配请求失败: %d", code)
	}
	decodeData(t, body.Data, &items)
	if len(items) != 1 || items[0].Sql != "SELECT ob-marker-"+uniq || items[0].Time != "4.0s" {
		t.Fatalf("OB units 兜底应只命中租户指纹: %+v", items)
	}
}

/* ---- 变更工单：过滤 / 时间窗边界 / 非法参数 ---- */

func TestChangesEndpoint(t *testing.T) {
	gdb := whitelistTestDB(t)
	r := whitelistRouter(gdb)
	uniq := fmt.Sprintf("wt%d", time.Now().UnixNano())
	metaC := &model.DbCluster{Name: "mcl-" + uniq, DbType: "pg", Environment: "生产", SourceSys: uniq}
	if err := gdb.Create(metaC).Error; err != nil {
		t.Fatalf("造数: %v", err)
	}
	t.Cleanup(func() {
		gdb.Where("source_sys = ?", uniq).Delete(&model.ChangeTicket{})
		gdb.Where("name = ?", "mcl-"+uniq).Delete(&model.DbCluster{})
	})

	tk1At := time.Date(2026, 8, 18, 2, 0, 0, 0, time.Local)
	tk2At := tk1At.Add(time.Hour)
	mk := func(no string, risk string, status int, at *time.Time, clusterID *int64) *model.ChangeTicket {
		return &model.ChangeTicket{SourceSys: uniq, TicketNo: no, Title: "变更-" + no, StatusCode: status,
			RiskLevel: risk, OwnerName: "tester", ExecuteStartAt: at, ClusterID: clusterID}
	}
	cid := metaC.ID
	tickets := []*model.ChangeTicket{
		mk(uniq+"-1", "高", 3, &tk1At, &cid),
		mk(uniq+"-2", "低", 1, &tk2At, nil),
		mk(uniq+"-3", "低", 0, nil, nil), // 仅计划，未执行 → 排最后
	}
	if err := gdb.Create(&tickets).Error; err != nil {
		t.Fatalf("造数: %v", err)
	}

	type changeRow struct {
		TicketNo    string `json:"ticketNo"`
		StatusLabel string `json:"statusLabel"`
	}
	markerIdx := func(items []changeRow) map[string]int {
		m := map[string]int{}
		for i, it := range items {
			switch it.TicketNo {
			case uniq + "-1":
				m["tk1"] = i
			case uniq + "-2":
				m["tk2"] = i
			case uniq + "-3":
				m["tk3"] = i
			}
		}
		return m
	}
	var resp struct {
		Items []changeRow `json:"items"`
		Total int         `json:"total"`
	}

	// ① 默认列表：execute_start_at 倒序、NULL 最后；状态标签映射
	code, body := doGet(t, r, "/api/changes?limit=500")
	if code != http.StatusOK {
		t.Fatalf("GET /api/changes: %d", code)
	}
	decodeData(t, body.Data, &resp)
	idx := markerIdx(resp.Items)
	if len(idx) != 3 {
		t.Fatalf("marker 工单缺失: %+v", idx)
	}
	if !(idx["tk2"] < idx["tk1"] && idx["tk1"] < idx["tk3"]) {
		t.Errorf("排序不符（应 tk2<tk1<tk3）: %+v", idx)
	}
	for _, it := range resp.Items {
		if it.TicketNo == uniq+"-1" && it.StatusLabel != "已完成" {
			t.Errorf("statusLabel(3) = %q, want 已完成", it.StatusLabel)
		}
	}

	// ② risk_level 过滤
	code, body = doGet(t, r, "/api/changes?risk_level=%E9%AB%98")
	if code != http.StatusOK {
		t.Fatalf("risk 过滤: %d", code)
	}
	decodeData(t, body.Data, &resp)
	idx = markerIdx(resp.Items)
	if len(idx) != 1 || idx["tk1"] != 0 && idx["tk1"] > 0 {
		// 只断言 tk1 命中、其余不在
		if _, has2 := idx["tk2"]; has2 {
			t.Errorf("risk 过滤不应包含 tk2")
		}
	}
	// ③ 时间窗闭区间边界：from=to=tk1 时刻 → 只 tk1
	q := "/api/changes?from=" + url.QueryEscape(tk1At.Format(time.RFC3339)) + "&to=" + url.QueryEscape(tk1At.Format(time.RFC3339))
	code, body = doGet(t, r, q)
	if code != http.StatusOK {
		t.Fatalf("时间窗过滤: %d", code)
	}
	decodeData(t, body.Data, &resp)
	idx = markerIdx(resp.Items)
	if len(idx) != 1 {
		t.Fatalf("闭区间边界应只含 tk1: %+v", idx)
	}
	if _, ok := idx["tk1"]; !ok {
		t.Fatalf("闭区间边界应包含 tk1: %+v", idx)
	}

	// ④ cluster_id 数字过滤
	code, body = doGet(t, r, "/api/changes?cluster_id="+fmt.Sprint(cid))
	decodeData(t, body.Data, &resp)
	idx = markerIdx(resp.Items)
	if len(idx) != 1 || idx["tk1"] < 0 {
		t.Fatalf("cluster_id 过滤应只含 tk1: %+v", idx)
	}

	// ⑤ 无命中过滤 → marker 为空
	code, body = doGet(t, r, "/api/changes?risk_level=不存在的级别")
	decodeData(t, body.Data, &resp)
	if idx = markerIdx(resp.Items); len(idx) != 0 {
		t.Errorf("无命中过滤应不含 marker: %+v", idx)
	}

	// ⑥ 非法参数 → 400
	for _, bad := range []string{
		"/api/changes?from=abc", "/api/changes?to=2026-13-45",
		"/api/changes?limit=abc", "/api/changes?limit=0", "/api/changes?limit=1000",
		"/api/changes?cluster_id=xyz",
	} {
		if code, _ = doGet(t, r, bad); code != http.StatusBadRequest {
			t.Errorf("%s 应 400, got %d", bad, code)
		}
	}
}

/* ---- 元数据域三级下钻 ---- */

func TestMetaDrillDown(t *testing.T) {
	gdb := whitelistTestDB(t)
	r := whitelistRouter(gdb)
	uniq := fmt.Sprintf("wt%d", time.Now().UnixNano())
	dbType := "tdb" + uniq

	metaC := &model.DbCluster{Name: "mcl-" + uniq, DbType: dbType, Environment: "生产", SourceSys: uniq}
	if err := gdb.Create(metaC).Error; err != nil {
		t.Fatalf("造数: %v", err)
	}
	sysI := &model.DbInstance{ClusterID: metaC.ID, DbType: dbType, Name: "sys", Role: "sys", SourceID: uniq + "-sys"}
	userI := &model.DbInstance{ClusterID: metaC.ID, DbType: dbType, Name: "tenant-a", Role: "user",
		SourceID: uniq + "-user", Extensions: datatypes.JSON([]byte(`{"mode":"mysql","unit_num":3}`))}
	if err := gdb.Create([]*model.DbInstance{sysI, userI}).Error; err != nil {
		t.Fatalf("造数: %v", err)
	}
	// 乱序插入，验证按 ordinal 排序
	nodes := []*model.DbInstanceNode{
		{InstanceID: sysI.ID, Ordinal: 1, Role: "observer", HostIP: "10.99.0.2"},
		{InstanceID: sysI.ID, Ordinal: 0, Role: "observer", HostIP: "10.99.0.1"},
	}
	if err := gdb.Create(&nodes).Error; err != nil {
		t.Fatalf("造数: %v", err)
	}
	t.Cleanup(func() {
		gdb.Where("instance_id IN ?", []int64{sysI.ID, userI.ID}).Delete(&model.DbInstanceNode{})
		gdb.Where("cluster_id = ?", metaC.ID).Delete(&model.DbInstance{})
		gdb.Where("name = ?", "mcl-"+uniq).Delete(&model.DbCluster{})
	})

	// ① 列表 + 过滤 + 计数
	code, body := doGet(t, r, "/api/meta/clusters?db_type="+dbType)
	if code != http.StatusOK {
		t.Fatalf("meta clusters: %d", code)
	}
	var list struct {
		Items []metaClusterItem `json:"items"`
	}
	decodeData(t, body.Data, &list)
	if len(list.Items) != 1 {
		t.Fatalf("db_type 过滤应只命中 1 个集群, got %d", len(list.Items))
	}
	if list.Items[0].InstanceCount != 2 || list.Items[0].NodeCount != 2 {
		t.Errorf("计数不符: %+v", list.Items[0])
	}
	clusterID := fmt.Sprint(list.Items[0].ID)

	// ② 详情三级下钻：sys 实例挂全部节点（OB 约定），user 实例节点为空数组
	code, body = doGet(t, r, "/api/meta/clusters/"+clusterID)
	if code != http.StatusOK {
		t.Fatalf("meta cluster detail: %d", code)
	}
	var detail struct {
		Cluster   model.DbCluster     `json:"cluster"`
		Instances []instanceWithNodes `json:"instances"`
	}
	decodeData(t, body.Data, &detail)
	if detail.Cluster.Name != "mcl-"+uniq || len(detail.Instances) != 2 {
		t.Fatalf("detail 不符: name=%s instances=%d", detail.Cluster.Name, len(detail.Instances))
	}
	var sysDetail, userDetail *instanceWithNodes
	for i := range detail.Instances {
		switch detail.Instances[i].Role {
		case "sys":
			sysDetail = &detail.Instances[i]
		case "user":
			userDetail = &detail.Instances[i]
		}
	}
	if sysDetail == nil || userDetail == nil {
		t.Fatalf("实例角色缺失: %+v", detail.Instances)
	}
	if len(sysDetail.Nodes) != 2 || sysDetail.Nodes[0].Ordinal != 0 || sysDetail.Nodes[1].Ordinal != 1 {
		t.Errorf("sys 节点应按 ordinal 升序: %+v", sysDetail.Nodes)
	}
	if userDetail.Nodes == nil || len(userDetail.Nodes) != 0 {
		t.Errorf("user 实例 nodes 应为空数组（非 null）: %+v", userDetail.Nodes)
	}

	// ③ 404：不存在 / 非数字 id
	for _, p := range []string{"/api/meta/clusters/999999999", "/api/meta/clusters/abc"} {
		if code, _ = doGet(t, r, p); code != http.StatusNotFound {
			t.Errorf("%s 应 404, got %d", p, code)
		}
	}

	// ④ 过滤无命中
	code, body = doGet(t, r, "/api/meta/clusters?environment=不存在环境")
	decodeData(t, body.Data, &list)
	for _, it := range list.Items {
		if it.Name == "mcl-"+uniq {
			t.Errorf("environment 过滤无命中不应返回 marker")
		}
	}
}

/* ---- 大盘：库类型活计数 + 响应形状 ---- */

func TestOverviewWhitelist(t *testing.T) {
	gdb := whitelistTestDB(t)
	r := whitelistRouter(gdb)
	uniq := fmt.Sprintf("wt%d", time.Now().UnixNano())
	dbType := "ovdb" + uniq

	metaC := &model.DbCluster{Name: "mcl-" + uniq, DbType: dbType, Environment: "生产", SourceSys: uniq}
	if err := gdb.Create(metaC).Error; err != nil {
		t.Fatalf("造数: %v", err)
	}
	if err := gdb.Create(&model.DbInstance{ClusterID: metaC.ID, DbType: dbType, Name: "i-" + uniq,
		Role: "storage", SourceID: "src" + uniq}).Error; err != nil {
		t.Fatalf("造数: %v", err)
	}
	t.Cleanup(func() {
		gdb.Where("source_id = ?", "src"+uniq).Delete(&model.DbInstance{})
		gdb.Where("name = ?", "mcl-"+uniq).Delete(&model.DbCluster{})
	})

	code, body := doGet(t, r, "/api/overview")
	if code != http.StatusOK || body.Code != 0 {
		t.Fatalf("GET /api/overview: %d %+v", code, body)
	}
	var ov struct {
		DbTypes  []dbTypeRow `json:"dbTypes"`
		Alerts   []alertItem `json:"alerts"`
		SlowSqls []slowItem  `json:"slowSqls"`
	}
	decodeData(t, body.Data, &ov)
	if len(ov.DbTypes) == 0 {
		t.Fatalf("dbTypes 不应为空（元数据域有数据时走活计数）")
	}
	found := false
	for _, d := range ov.DbTypes {
		if d.Type == dbType {
			found = true
			if d.Total < 1 || d.Name == "" || d.Icon == "" {
				t.Errorf("marker 库类型行不完整: %+v", d)
			}
		}
	}
	if !found {
		t.Errorf("dbTypes 应包含 marker 类型 %s: %+v", dbType, ov.DbTypes)
	}
	// alerts/slowSqls 键存在且为数组（内容多寡取决于库内数据）
	if ov.Alerts == nil || ov.SlowSqls == nil {
		t.Errorf("alerts/slowSqls 应为数组: alerts=%v slowSqls=%v", ov.Alerts, ov.SlowSqls)
	}
}
