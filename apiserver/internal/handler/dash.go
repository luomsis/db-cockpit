package handler

import (
	"github.com/gin-gonic/gin"

	"db-cockpit/apiserver/internal/envelope"
	"db-cockpit/apiserver/internal/metrics"
	"db-cockpit/apiserver/internal/model"
)

/* ================= 统一查询协议：/api/dash/* ================= */

func (h *H) clusterLites() []metrics.ClusterLite {
	var clusters []model.Cluster
	h.DB.Order("id asc").Find(&clusters)
	var instances []model.Instance
	h.DB.Order("id asc").Find(&instances)
	namesByCluster := map[string][]string{}
	for _, inst := range instances {
		namesByCluster[inst.ClusterID] = append(namesByCluster[inst.ClusterID], inst.Name)
	}
	lites := make([]metrics.ClusterLite, 0, len(clusters))
	for _, cl := range clusters {
		lites = append(lites, metrics.ClusterLite{Name: cl.Name, Type: cl.Type, InstanceNames: namesByCluster[cl.ID]})
	}
	return lites
}

func (h *H) ListMetrics(c *gin.Context) {
	var defs []model.MetricDef
	h.DB.Order("id asc").Find(&defs)
	envelope.OK(c, defs)
}

func (h *H) DashSeries(c *gin.Context) {
	q := metrics.Query{
		Metric:  c.Query("metric"),
		Scope:   c.DefaultQuery("scope", "global"),
		Range:   c.DefaultQuery("range", "24h"),
		Shift:   c.Query("shift"),
		Name:    c.Query("name"),
		Color:   c.Query("color"),
		Axis:    c.Query("axis"),
		Type:    c.Query("type"),
		Unit:    c.Query("unit"),
		Agg:     c.DefaultQuery("agg", "avg"),
		GroupBy: c.DefaultQuery("groupBy", "none"),
		DbType:  c.Query("dbType"),
	}
	if q.Metric == "" {
		envelope.BadRequest(c, "metric required")
		return
	}
	var defs []model.MetricDef
	h.DB.Find(&defs)
	series := metrics.BuildSeries(q, defs, h.clusterLites())
	envelope.OK(c, gin.H{"series": series})
}

func (h *H) DashAnnotations(c *gin.Context) {
	rng := c.DefaultQuery("range", "24h")
	envelope.OK(c, metrics.BuildAnnotations(rng))
}
