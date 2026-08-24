package handler

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"db-cockpit/apiserver/internal/model"
)

/* 元数据域 v2 集成测试：真实 PG（不可达跳过）+ 造数验证 meta 下钻 / 慢SQL 解析链 */

func metaTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		dsn = "host=localhost port=55432 user=graphiti password=graphiti dbname=db_cockpit sslmode=disable"
	}
	gdb, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Skipf("PG 不可达，跳过：%v", err)
	}
	if err := gdb.AutoMigrate(&model.DbCluster{}, &model.DbComponent{}, &model.DbHost{},
		&model.SlowQueryLog{}, &model.AlertRaw{}, &model.ChangeTicket{}, &model.DbSyncWatermark{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return gdb
}

func metaRouter(gdb *gorm.DB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := &H{DB: gdb}
	r := gin.New()
	api := r.Group("/api")
	api.GET("/meta/clusters", h.ListMetaClusters)
	api.GET("/meta/clusters/:id", h.GetMetaCluster)
	api.GET("/clusters/:id/instances/:iid/slow-sqls", h.ListSlowSqls)
	return r
}

/* 造数：1 个 PG 集群（1 主 2 备 replication→主）+ 1 个 OB 集群（obproxy + 租户 units 落位 observer + observer 成员） */

type metaMarker struct {
	clusterID int64
	compIDs   map[string]int64 // key: clusterName/name
	hostIPs   []string
}

func seedMetaMarker(t *testing.T, gdb *gorm.DB) metaMarker {
	t.Helper()
	uniq := fmt.Sprintf("mv2_%d", os.Getpid())
	cl := &model.DbCluster{Name: "meta-test-" + uniq, DbType: "pg", Environment: "生产",
		Endpoint: "10.99.0.10:5432", Vip: "10.99.0.10", Port: 5432}
	gdb.Create(cl)
	t.Cleanup(func() {
		gdb.Where("cluster_id = ?", cl.ID).Delete(&model.DbComponent{})
		gdb.Where("id = ?", cl.ID).Delete(&model.DbCluster{})
	})

	hosts := []string{"10.99.1.11", "10.99.1.12", "10.99.1.13"}
	for _, ip := range hosts {
		gdb.Create(&model.DbHost{HostIP: ip, HostName: "meta-host-" + ip, Region: "华东", Az: "az-a", HostCluster: "hc-1", OsName: "CentOS 7.9", HostInfraType: "虚拟机", Status: "ok"})
	}
	t.Cleanup(func() {
		for _, ip := range hosts {
			gdb.Where("host_ip = ?", ip).Delete(&model.DbHost{})
		}
	})

	comp := func(name, kind, role, host string) *model.DbComponent {
		return &model.DbComponent{ClusterID: cl.ID, Name: name, Kind: kind, Role: role, HostIP: host,
			Version: "PG 15.6", Status: "ok", Port: 5432, SourceID: "meta-comp-" + name}
	}
	master := comp("pg-master", "storage", "primary", hosts[0])
	sec1 := comp("pg-sec1", "storage", "secondary", hosts[1])
	sec2 := comp("pg-sec2", "storage", "secondary", hosts[2])
	gdb.Create(master)
	gdb.Create(sec1)
	gdb.Create(sec2)
	// 复制链：两备 → 主
	gdb.Model(sec1).Update("replication_upstream_id", master.ID)
	gdb.Model(sec2).Update("replication_upstream_id", master.ID)

	// 慢SQL：一条指向 master 组件
	gdb.Create(&model.SlowQueryLog{SourceSys: "log-sys", InstanceID: &master.ID,
		Endpoint: "10.99.0.10:5432", Hostname: "meta-host", HostIP: hosts[0], Port: 5432,
		DatabaseName: "test_db", Username: "app_rw", SqlText: "SELECT 1", Digest: "abc123",
		ExecuteMs: 100, RowsExamined: ipid(1000), CreateTime: time.Now()})

	return metaMarker{clusterID: cl.ID, compIDs: map[string]int64{
		"meta-test-" + uniq + "/pg-master": master.ID,
		"meta-test-" + uniq + "/pg-sec1":  sec1.ID,
		"meta-test-" + uniq + "/pg-sec2":  sec2.ID,
	}, hostIPs: hosts}
}

func ipid(v int64) *int64 { return &v }

/* ---- 测试：meta 列表 / 下钻 / 慢SQL 解析 ---- */

func TestMetaClustersV2(t *testing.T) {
	gdb := metaTestDB(t)
	r := metaRouter(gdb)
	_ = seedMetaMarker(t, gdb)

	// 列表：componentCount = 3, hostCount = 3
	code, body := doGet(t, r, "/api/meta/clusters")
	if code != http.StatusOK {
		t.Fatalf("list: %d", code)
	}
	var resp struct {
		Items []struct {
			Name           string `json:"name"`
			DbType         string `json:"dbType"`
			ComponentCount int64  `json:"componentCount"`
			HostCount      int64  `json:"hostCount"`
		} `json:"items"`
	}
	decodeData(t, body.Data, &resp)
	var found bool
	for _, i := range resp.Items {
		if strings.Contains(i.Name, "meta-test-") {
			found = true
			if i.ComponentCount != 3 || i.HostCount != 3 {
				t.Fatalf("计数不符: %+v", i)
			}
		}
	}
	if !found {
		t.Fatalf("marker 集群未出现: %+v", resp.Items)
	}
}

func TestMetaClusterDetailV2(t *testing.T) {
	gdb := metaTestDB(t)
	r := metaRouter(gdb)
	m := seedMetaMarker(t, gdb)

	code, body := doGet(t, r, fmt.Sprintf("/api/meta/clusters/%d", m.clusterID))
	if code != http.StatusOK {
		t.Fatalf("detail: %d", code)
	}
	var resp struct {
		Cluster    model.DbCluster     `json:"cluster"`
		Components []model.DbComponent `json:"components"`
		Hosts      []model.DbHost      `json:"hosts"`
	}
	decodeData(t, body.Data, &resp)
	if len(resp.Components) != 3 {
		t.Fatalf("components = %d, want 3", len(resp.Components))
	}
	if len(resp.Hosts) != 3 {
		t.Fatalf("hosts = %d, want 3", len(resp.Hosts))
	}
	// 复制链：两备 replication_upstream_id → 主
	byName := map[string]*model.DbComponent{}
	for i := range resp.Components {
		byName[resp.Components[i].Name] = &resp.Components[i]
	}
	if byName["pg-sec1"].ReplicationUpstreamID == nil || *byName["pg-sec1"].ReplicationUpstreamID != byName["pg-master"].ID {
		t.Errorf("sec1 复制上游未指向主: %+v", byName["pg-sec1"].ReplicationUpstreamID)
	}
}

func TestSlowSqlResolveV2(t *testing.T) {
	gdb := metaTestDB(t)
	r := metaRouter(gdb)
	m := seedMetaMarker(t, gdb)

	// UI 演示实例 → 元数据域 db_component（名称匹配）
	// 造一个演示 Instance 指向 master 组件
	demoInst := &model.Instance{ID: "demo-inst-v2", Name: "pg-master", IP: "10.99.1.11", Port: 5432, Status: "ok"}
	gdb.Create(demoInst)
	t.Cleanup(func() { gdb.Where("id = ?", demoInst.ID).Delete(&model.Instance{}) })

	code, body := doGet(t, r, fmt.Sprintf("/api/clusters/c1/instances/%s/slow-sqls", demoInst.ID))
	if code != http.StatusOK {
		t.Fatalf("slow-sqls: %d", code)
	}
	var items []slowItem
	decodeData(t, body.Data, &items)
	if len(items) != 1 || items[0].Sql != "SELECT 1" {
		t.Fatalf("慢SQL 解析链失败: %+v", items)
	}
	_ = m
}
