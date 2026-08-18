package handler

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"db-cockpit/apiserver/internal/agent"
	"db-cockpit/apiserver/internal/envelope"
	"db-cockpit/apiserver/internal/model"
)

/* ================= DTO：与前端 Cluster 类型逐字段一致 ================= */

type tenantDTO struct {
	model.ObTenant
	Databases []model.ObTenantDb `json:"databases"`
}

type clusterDTO struct {
	model.Cluster
	Instances  []model.Instance    `json:"instances"`
	Params     []model.ClusterParam `json:"params"`
	Tenants    []tenantDTO         `json:"tenants,omitempty"`
	Databases  []model.PgDatabase  `json:"databases,omitempty"`
	Replicas   []model.PgReplica   `json:"replicas,omitempty"`
}

func (h *H) loadClusterDTO(id string, full bool) (*clusterDTO, error) {
	var cl model.Cluster
	if err := h.DB.Where("id = ?", id).First(&cl).Error; err != nil {
		return nil, err
	}
	dto := &clusterDTO{Cluster: cl}
	h.DB.Where("cluster_id = ?", id).Order("id asc").Find(&dto.Instances)
	if !full {
		return dto, nil
	}
	h.DB.Where("scope = ? AND cluster_id = ?", "cluster", id).Order("id asc").Find(&dto.Params)
	switch cl.Type {
	case "pg":
		h.DB.Where("cluster_id = ?", id).Order("id asc").Find(&dto.Databases)
		h.DB.Where("cluster_id = ?", id).Order("id asc").Find(&dto.Replicas)
	case "oceanbase":
		var tenants []model.ObTenant
		h.DB.Where("cluster_id = ?", id).Order("id asc").Find(&tenants)
		for _, t := range tenants {
			td := tenantDTO{ObTenant: t}
			h.DB.Where("tenant_id = ?", t.ID).Order("id asc").Find(&td.Databases)
			dto.Tenants = append(dto.Tenants, td)
		}
	}
	return dto, nil
}

/* ================= 列表 / 详情 ================= */

func (h *H) ListClusters(c *gin.Context) {
	kw := c.Query("kw")
	typ := c.Query("type")
	ver := c.Query("ver")
	az := c.Query("az")
	page, pageSize := pageParams(c)

	q := h.DB.Model(&model.Cluster{})
	if kw != "" {
		q = q.Where("name ILIKE ?", "%"+kw+"%")
	}
	if typ != "" {
		q = q.Where("type = ?", typ)
	}
	if ver != "" {
		q = q.Where("version = ?", ver)
	}
	if az != "" {
		q = q.Where("az = ?", az)
	}
	var total int64
	q.Count(&total)
	var clusters []model.Cluster
	q.Order("id asc").Limit(pageSize).Offset((page - 1) * pageSize).Find(&clusters)

	items := make([]*clusterDTO, 0, len(clusters))
	for _, cl := range clusters {
		dto := &clusterDTO{Cluster: cl}
		h.DB.Where("cluster_id = ?", cl.ID).Order("id asc").Find(&dto.Instances)
		dto.Params = []model.ClusterParam{}
		items = append(items, dto)
	}
	envelope.OK(c, gin.H{"items": items, "total": total})
}

func (h *H) ClusterFilterOptions(c *gin.Context) {
	var clusters []model.Cluster
	h.DB.Order("id asc").Find(&clusters)
	types := map[string]bool{}
	versions := map[string]bool{}
	azs := map[string]bool{}
	for _, cl := range clusters {
		types[cl.Type] = true
		versions[cl.Version] = true
		azs[cl.AZ] = true
	}
	envelope.OK(c, gin.H{"types": keys(types), "versions": keys(versions), "azs": keys(azs)})
}

func keys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func (h *H) GetCluster(c *gin.Context) {
	dto, err := h.loadClusterDTO(c.Param("id"), true)
	if err != nil {
		envelope.NotFound(c, "cluster not found")
		return
	}
	envelope.OK(c, dto)
}

/* ================= 参数管理 ================= */

func (h *H) ListClusterParams(c *gin.Context) {
	var params []model.ClusterParam
	h.DB.Where("scope = ? AND cluster_id = ?", "cluster", c.Param("id")).Order("id asc").Find(&params)
	envelope.OK(c, params)
}

func (h *H) UpdateClusterParam(c *gin.Context) {
	var body struct {
		Value string `json:"value" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		envelope.BadRequest(c, "value required")
		return
	}
	var p model.ClusterParam
	if err := h.DB.Where("scope = ? AND cluster_id = ? AND name = ?", "cluster", c.Param("id"), c.Param("name")).First(&p).Error; err != nil {
		envelope.NotFound(c, "param not found")
		return
	}
	old := p.Value
	p.Value = body.Value
	p.Status = "pending"
	if err := h.DB.Save(&p).Error; err != nil {
		envelope.Internal(c, err)
		return
	}
	h.DB.Create(&model.ParamHistory{ParamID: p.ID, OldValue: old, NewValue: p.Value, ChangedAt: time.Now().UnixMilli()})
	h.audit("param.update", "cluster_param", c.Param("id")+"/"+p.Name, map[string]interface{}{"old": old, "new": p.Value})
	envelope.OK(c, p)
}

func (h *H) ClusterParamHistory(c *gin.Context) {
	var p model.ClusterParam
	if err := h.DB.Where("scope = ? AND cluster_id = ? AND name = ?", "cluster", c.Param("id"), c.Param("name")).First(&p).Error; err != nil {
		envelope.NotFound(c, "param not found")
		return
	}
	var hist []model.ParamHistory
	h.DB.Where("param_id = ?", p.ID).Order("changed_at desc").Find(&hist)
	envelope.OK(c, hist)
}

/* ================= PG 数据库管理 ================= */

func (h *H) ListPgDatabases(c *gin.Context) {
	var dbs []model.PgDatabase
	h.DB.Where("cluster_id = ?", c.Param("id")).Order("id asc").Find(&dbs)
	envelope.OK(c, dbs)
}

func (h *H) CreatePgDatabase(c *gin.Context) {
	var body struct {
		Name  string `json:"name" binding:"required"`
		Owner string `json:"owner"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		envelope.BadRequest(c, "name required")
		return
	}
	owner := body.Owner
	if owner == "" {
		owner = "app_rw"
	}
	db := model.PgDatabase{ClusterID: c.Param("id"), Name: body.Name, Owner: owner,
		Size: "0 GB", Tables: 0, Conn: 0, ConnLimit: 100, Status: "ok"}
	if err := h.DB.Create(&db).Error; err != nil {
		envelope.BadRequest(c, "create failed (duplicate name?)")
		return
	}
	h.audit("database.create", "pg_database", c.Param("id")+"/"+body.Name, nil)
	envelope.OK(c, db)
}

/* ================= PG 复制与高可用 ================= */

func (h *H) ListPgReplicas(c *gin.Context) {
	var reps []model.PgReplica
	h.DB.Where("cluster_id = ?", c.Param("id")).Order("id asc").Find(&reps)
	envelope.OK(c, reps)
}

// SwitchDrill：切换演练——目标备库升主，原主降备（模拟状态变更 + 审计）
func (h *H) SwitchDrill(c *gin.Context) {
	clusterID := c.Param("id")
	instName := c.Param("inst")
	var cl model.Cluster
	if err := h.DB.Where("id = ? AND type = ?", clusterID, "pg").First(&cl).Error; err != nil {
		envelope.NotFound(c, "pg cluster not found")
		return
	}
	var instances []model.Instance
	h.DB.Where("cluster_id = ?", clusterID).Order("id asc").Find(&instances)
	var target, primary *model.Instance
	for i := range instances {
		if instances[i].Name == instName {
			target = &instances[i]
		}
		if instances[i].Role == "主库 Primary" {
			primary = &instances[i]
		}
	}
	if target == nil || target.Role == "主库 Primary" {
		envelope.BadRequest(c, "target standby instance not found")
		return
	}
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		target.Role = "主库 Primary"
		if err := tx.Save(target).Error; err != nil {
			return err
		}
		if primary != nil {
			primary.Role = "备库 Standby"
			if err := tx.Save(primary).Error; err != nil {
				return err
			}
		}
		// 复制表：目标备库行删除，原主新增一行
		if err := tx.Where("cluster_id = ? AND instance = ?", clusterID, instName).
			Delete(&model.PgReplica{}).Error; err != nil {
			return err
		}
		if primary != nil {
			row := model.PgReplica{ClusterID: clusterID, Instance: primary.Name, Role: "Standby（quorum）", DelayMs: 0, WalLag: "0 MB", Status: "ok"}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		envelope.Internal(c, err)
		return
	}
	h.audit("replica.switch_drill", "cluster", clusterID, map[string]interface{}{"newPrimary": instName})
	envelope.OK(c, gin.H{"ok": true, "newPrimary": instName})
}

func (h *H) RebuildReplica(c *gin.Context) {
	clusterID := c.Param("id")
	instName := c.Param("inst")
	res := h.DB.Model(&model.PgReplica{}).
		Where("cluster_id = ? AND instance = ?", clusterID, instName).
		Updates(map[string]interface{}{"delay_ms": 0, "wal_lag": "0 MB", "status": "ok"})
	if res.Error != nil || res.RowsAffected == 0 {
		envelope.NotFound(c, "replica not found")
		return
	}
	h.audit("replica.rebuild", "cluster", clusterID, map[string]interface{}{"instance": instName})
	envelope.OK(c, gin.H{"ok": true})
}

/* ================= OB 租户 ================= */

func (h *H) ListObTenants(c *gin.Context) {
	dto, err := h.loadClusterDTO(c.Param("id"), true)
	if err != nil {
		envelope.NotFound(c, "cluster not found")
		return
	}
	envelope.OK(c, dto.Tenants)
}

func (h *H) CreateObTenant(c *gin.Context) {
	clusterID := c.Param("id")
	var cl model.Cluster
	if err := h.DB.Where("id = ? AND type = ?", clusterID, "oceanbase").First(&cl).Error; err != nil {
		envelope.NotFound(c, "oceanbase cluster not found")
		return
	}
	var body struct {
		Name        string  `json:"name" binding:"required"`
		Mode        string  `json:"mode"`
		PrimaryZone string  `json:"primaryZone"`
		MaxCpu      float64 `json:"maxCpu"`
		MaxMemGb    float64 `json:"maxMemGb"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		envelope.BadRequest(c, "name required")
		return
	}
	mode := body.Mode
	if mode == "" {
		mode = "mysql"
	}
	pz := body.PrimaryZone
	if pz == "" {
		pz = "RANDOM"
	}
	maxCpu := body.MaxCpu
	if maxCpu <= 0 {
		maxCpu = 4
	}
	maxMem := body.MaxMemGb
	if maxMem <= 0 {
		maxMem = 16
	}
	// Unit 分布：每个 Zone 取该 Zone 第一个 OBServer
	type unitRow struct {
		Zone     string  `json:"zone"`
		Observer string  `json:"observer"`
		MaxCpu   float64 `json:"maxCpu"`
		UsedCpu  float64 `json:"usedCpu"`
		MaxMemGb float64 `json:"maxMemGb"`
		UsedMemGb float64 `json:"usedMemGb"`
	}
	var zones []string
	_ = json.Unmarshal([]byte(cl.Zones), &zones)
	seenZone := map[string]string{}
	var instances []model.Instance
	h.DB.Where("cluster_id = ?", clusterID).Order("id asc").Find(&instances)
	for _, inst := range instances {
		if inst.Zone != nil {
			if _, ok := seenZone[*inst.Zone]; !ok {
				seenZone[*inst.Zone] = inst.Name
			}
		}
	}
	units := make([]unitRow, 0, len(zones))
	for _, z := range zones {
		if ob, ok := seenZone[z]; ok {
			units = append(units, unitRow{Zone: z, Observer: ob, MaxCpu: maxCpu, UsedCpu: 0, MaxMemGb: maxMem, UsedMemGb: 0})
		}
	}
	raw := jsonMarshal(units)

	t := model.ObTenant{
		ID: agent.NewID("t"), ClusterID: clusterID, Name: body.Name, Kind: "user", Mode: mode,
		PrimaryZone: pz, Locality: "F@ZONE1,F@ZONE2,F@ZONE3", UnitNum: 1,
		MaxCpu: maxCpu, UsedCpu: 0, MaxMemGb: maxMem, UsedMemGb: 0,
		StorageUsed: "10 GB", StorageTotal: "100 GB",
		Units: raw, Whitelist: datatypes.JSON([]byte(`["%"]`)), ConnHint: "-- created by db-cockpit", Status: "ok",
	}
	if err := h.DB.Create(&t).Error; err != nil {
		envelope.Internal(c, err)
		return
	}
	h.audit("tenant.create", "ob_tenant", t.ID, map[string]interface{}{"name": body.Name, "maxCpu": maxCpu})
	envelope.OK(c, t)
}

/* ================= 报告 ================= */

func (h *H) ListReports(c *gin.Context) {
	var reports []model.Report
	h.DB.Where("cluster_id = ? OR cluster_id = ''", c.Param("id")).Order("id asc").Find(&reports)
	envelope.OK(c, reports)
}

func (h *H) DownloadReport(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("rid"))
	var r model.Report
	if err := h.DB.Where("id = ?", id).First(&r).Error; err != nil {
		envelope.NotFound(c, "report not found")
		return
	}
	body := fmt.Sprintf("# %s %s\n\n%s\n\n- 日期：%s\n- 大小：%s\n- 集群：%s\n\n> 由 db-cockpit apiserver 生成的演示报告。\n",
		r.Ico, r.Title, r.Desc, r.Date, r.Size, c.Param("id"))
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="report-%d.md"`, r.ID))
	c.Data(200, "text/markdown; charset=utf-8", []byte(body))
}
