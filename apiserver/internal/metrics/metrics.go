package metrics

import (
	"math"
	"sort"

	"db-cockpit/apiserver/internal/model"
)

// ClusterLite：resolveGroups 只需要集群名、类型与实例名列表
type ClusterLite struct {
	Name          string
	Type          string
	InstanceNames []string
}

/* ================= 与前端 query.ts MockProvider 算法逐行对齐 =================
 * 目标：同一 (metric, range, shift, agg, groupBy, dbType) 请求，
 * apiserver 返回与原前端 mock 完全一致的数据，切换 Provider 后视觉零变化。
 */

var groupPalette = []string{"#006aff", "#ff9500", "#00b365", "#7a5af8", "#f53f3f", "#00a3e0", "#d46b08", "#0091c9", "#6a3fd4", "#00897b", "#d4380d", "#8a97ad"}

func rangeLen(rng string) int {
	switch rng {
	case "1h", "6h":
		return 12
	case "7d":
		return 7
	default:
		return 24 // 24h
	}
}

func rangeSpike(rng string) int {
	if rng == "7d" {
		return 3
	}
	return 14
}

func shiftSeed(shift string) uint32 {
	if shift != "" {
		return 20260809
	}
	return 20260811
}

// mulberry32：对应 query.ts 的 seededRandom
func seededRandom(seed uint32) func() float64 {
	a := seed
	return func() float64 {
		a += 0x6D2B79F5
		t := imul32(a^(a>>15), a|1)
		t = (t + imul32(t^(t>>7), t|61)) ^ t
		return float64(t^(t>>14)) / 4294967296.0
	}
}

func imul32(x, y uint32) uint32 { return x * y }

// jsRound：JS Math.round 向 +Infinity 取整
func jsRound(v float64) float64 { return math.Floor(v + 0.5) }

func genSeries(base, jitter float64, spikeAt int, seed uint32) []float64 {
	rnd := seededRandom(seed)
	n := 24
	arr := make([]float64, 0, n)
	for i := 0; i < n; i++ {
		v := base + math.Sin(float64(i)/3.2)*jitter + (rnd()-0.5)*jitter
		if spikeAt != 0 && abs(i-spikeAt) <= 1 {
			v += jitter * 2.6
		}
		arr = append(arr, math.Max(1, jsRound(v)))
	}
	return arr
}

// GenSeries 导出给 agent 包生成卡片演示序列（与前端 mock 同算法）
func GenSeries(base, jitter float64, spikeAt int, seed uint32) []float64 {
	return genSeries(base, jitter, spikeAt, seed)
}

func abs(i int) int { if i < 0 { return -i }; return i }

func applyAgg(data []float64, agg string) []float64 {
	if agg == "" || agg == "avg" {
		return data
	}
	out := make([]float64, len(data))
	switch agg {
	case "max":
		for i, v := range data {
			out[i] = jsRound(v*1.35 + 5)
		}
	case "min":
		for i, v := range data {
			out[i] = math.Max(1, jsRound(v*0.65-3))
		}
	case "last":
		lv := 1.0
		if len(data) > 0 {
			lv = data[len(data)-1]
		}
		for i := range data {
			out[i] = lv
		}
	case "p95":
		sorted := append([]float64(nil), data...)
		sort.Float64s(sorted)
		idx := int(math.Floor(float64(len(sorted)) * 0.95))
		p95v := 1.0
		if idx < len(sorted) {
			p95v = sorted[idx]
		} else if len(sorted) > 0 {
			p95v = sorted[len(sorted)-1]
		}
		for i, v := range data {
			out[i] = jsRound((v + p95v) / 2)
		}
	default:
		return data
	}
	return out
}

type group struct {
	Name     string
	SeedBase int
}

func resolveGroups(q Query, clusters []ClusterLite) []group {
	filtered := make([]ClusterLite, 0, len(clusters))
	for _, c := range clusters {
		if q.DbType == "" || c.Type == q.DbType {
			filtered = append(filtered, c)
		}
	}
	if q.GroupBy == "cluster" {
		gs := make([]group, 0, len(filtered))
		for i, c := range filtered {
			gs = append(gs, group{Name: c.Name, SeedBase: i + 1})
		}
		return gs
	}
	if q.GroupBy == "instance" {
		gs := make([]group, 0, 32)
		for ci, c := range filtered {
			for ii, name := range c.InstanceNames {
				gs = append(gs, group{Name: name, SeedBase: ci*100 + ii + 1})
			}
		}
		if len(gs) > 12 {
			gs = gs[:12]
		}
		return gs
	}
	return []group{{Name: "", SeedBase: 0}}
}

// Query 对应前端 QueryRequest 中真正影响服务端计算的字段
type Query struct {
	Metric  string
	Scope   string
	Range   string
	Shift   string
	Name    string
	Color   string
	Axis    string
	Type    string
	Unit    string
	Agg     string
	GroupBy string
	DbType  string
}

type ResolvedTarget struct {
	Name   string    `json:"name"`
	Data   []float64 `json:"data"`
	Color  string    `json:"color"`
	Unit   string    `json:"unit"`
	Axis   string    `json:"axis"`
	Type   string    `json:"type"`
	Dashed bool      `json:"dashed,omitempty"`
}

// BuildSeries：clusters 由调用方从库中读出（与前端 CLUSTERS 常量同源同序）
func BuildSeries(q Query, defs []model.MetricDef, clusters []ClusterLite) []ResolvedTarget {
	m := model.MetricDef{ID: q.Metric, Name: q.Metric, Unit: "", Base: 50, Jitter: 10}
	for _, d := range defs {
		if d.ID == q.Metric {
			m = d
			break
		}
	}
	n := rangeLen(q.Range)
	spike := rangeSpike(q.Range)
	ss := shiftSeed(q.Shift)
	groups := resolveGroups(q, clusters)
	useGroup := q.GroupBy != "" && q.GroupBy != "none" && len(groups) > 1

	series := make([]ResolvedTarget, 0, len(groups))
	for gi, g := range groups {
		seed := uint32(int64(ss) + int64(g.SeedBase)*7919)
		data := genSeries(m.Base, m.Jitter, spike, seed)
		if len(data) > n {
			data = data[:n]
		}
		data = applyAgg(data, q.Agg)

		name := q.Name
		if name == "" {
			name = m.Name
		}
		color := q.Color
		if useGroup {
			color = groupPalette[gi%len(groupPalette)]
			if q.Name != "" {
				name = q.Name + " · " + g.Name
			} else {
				name = g.Name
			}
		}
		unit := q.Unit
		if unit == "" {
			unit = m.Unit
		}
		axis := "left"
		if q.Axis == "right" {
			axis = "right"
		}
		typ := q.Type
		if typ == "" {
			typ = "line"
		}
		dashed := q.Shift != ""
		series = append(series, ResolvedTarget{Name: name, Data: data, Color: color, Unit: unit, Axis: axis, Type: typ, Dashed: dashed})
	}
	return series
}

/* ================= annotations（对应 MockProvider.fetchAnnotations） ================= */

type Annotation struct {
	Time  string `json:"time"`
	Title string `json:"title"`
	Type  string `json:"type"`
}

func BuildAnnotations(rng string) []Annotation {
	seedMap := map[string]uint32{"1h": 1, "6h": 6, "7d": 7}
	seed, ok := seedMap[rng]
	if !ok {
		seed = 24
	}
	rnd := seededRandom(seed)
	n := rangeLen(rng)
	pool := []Annotation{
		{Type: "release", Title: "版本发布 v8.0.36"},
		{Type: "switch", Title: "主备切换"},
		{Type: "alert", Title: "CPU 使用率告警"},
		{Type: "alert", Title: "慢 SQL 告警"},
		{Type: "release", Title: "参数配置变更"},
	}
	count := n / 8
	if count < 1 {
		count = 1
	}
	anns := make([]Annotation, 0, count)
	for i := 0; i < count; i++ {
		idx := int(float64(n) * rnd())
		if idx >= n {
			idx = n - 1
		}
		pi := int(float64(len(pool)) * rnd())
		if pi >= len(pool) {
			pi = len(pool) - 1
		}
		anns = append(anns, Annotation{Time: tickLabel(rng, idx), Title: pool[pi].Title, Type: pool[pi].Type})
	}
	return anns
}

// tickLabel：与前端 rangeTicks 生成的刻度文案一致
func tickLabel(rng string, idx int) string {
	switch rng {
	case "1h":
		return itoa(idx*5) + "m"
	case "6h":
		return itoa(idx*30) + "m"
	case "7d":
		names := []string{"周一", "周二", "周三", "周四", "周五", "周六", "周日"}
		if idx < len(names) {
			return names[idx]
		}
		return names[len(names)-1]
	default:
		h := idx
		if h > 23 {
			h = 23
		}
		s := itoa(h)
		if len(s) < 2 {
			s = "0" + s
		}
		return s + ":00"
	}
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	neg := i < 0
	if neg {
		i = -i
	}
	b := [20]byte{}
	p := len(b)
	for i > 0 {
		p--
		b[p] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		p--
		b[p] = '-'
	}
	return string(b[p:])
}
