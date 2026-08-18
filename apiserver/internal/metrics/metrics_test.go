package metrics

import (
	"encoding/json"
	"testing"

	"db-cockpit/apiserver/internal/model"
)

// 与前端 query.ts 原算法（JS）对拍的黄金向量：metric=cpu range=24h agg=max
func TestBuildSeriesParityWithFrontendMock(t *testing.T) {
	defs := []model.MetricDef{{ID: "cpu", Name: "CPU 使用率", Unit: "%", Base: 55, Jitter: 14}}
	clusters := []ClusterLite{{Name: "prod-pg-order-01", Type: "pg"}}
	got := BuildSeries(Query{Metric: "cpu", Range: "24h", Agg: "max"}, defs, clusters)
	want := []float64{77, 82, 95, 87, 104, 97, 97, 91, 100, 79, 71, 83, 70, 113, 117, 102, 54, 71, 77, 78, 82, 79, 97, 98}
	if len(got) != 1 || len(got[0].Data) != 24 {
		t.Fatalf("series shape: %d series, len=%d", len(got), len(got[0].Data))
	}
	for i := range want {
		if got[0].Data[i] != want[i] {
			t.Fatalf("point %d: got %v want %v", i, got[0].Data[i], want[i])
		}
	}
}

func TestBuildSeriesDeterministic(t *testing.T) {
	defs := []model.MetricDef{{ID: "qps", Name: "QPS", Unit: "", Base: 210, Jitter: 60}}
	clusters := []ClusterLite{
		{Name: "a", Type: "pg", InstanceNames: []string{"a1", "a2"}},
		{Name: "b", Type: "oceanbase", InstanceNames: []string{"b1"}},
	}
	q := Query{Metric: "qps", Range: "1h", GroupBy: "instance", Agg: "p95"}
	s1 := BuildSeries(q, defs, clusters)
	s2 := BuildSeries(q, defs, clusters)
	j1, _ := json.Marshal(s1)
	j2, _ := json.Marshal(s2)
	if string(j1) != string(j2) {
		t.Fatal("same query must produce identical series")
	}
	if len(s1) != 3 {
		t.Fatalf("instance fan-out: got %d series, want 3", len(s1))
	}
	if len(s1[0].Data) != 12 {
		t.Fatalf("1h range: got %d points, want 12", len(s1[0].Data))
	}
}

// 昨日对比（shift）不改变当日序列的确定性
func TestBuildSeriesShiftStable(t *testing.T) {
	defs := []model.MetricDef{{ID: "cpu", Name: "CPU", Unit: "%", Base: 55, Jitter: 14}}
	clusters := []ClusterLite{{Name: "a", Type: "pg"}}
	today := BuildSeries(Query{Metric: "cpu", Range: "24h"}, defs, clusters)
	shifted := BuildSeries(Query{Metric: "cpu", Range: "24h", Shift: "24h"}, defs, clusters)
	if today[0].Dashed || !shifted[0].Dashed {
		t.Fatal("shifted series must be dashed, today must not")
	}
	a, _ := json.Marshal(today)
	b, _ := json.Marshal(shifted)
	if string(a) == string(b) {
		t.Fatal("shift should alter the generated data")
	}
	// 同一 shift 两次请求仍一致
	again := BuildSeries(Query{Metric: "cpu", Range: "24h", Shift: "24h"}, defs, clusters)
	c, _ := json.Marshal(again)
	if string(b) != string(c) {
		t.Fatal("shift must be deterministic")
	}
}

func TestBuildAnnotationsDeterministic(t *testing.T) {
	a1 := BuildAnnotations("24h")
	a2 := BuildAnnotations("24h")
	j1, _ := json.Marshal(a1)
	j2, _ := json.Marshal(a2)
	if string(j1) != string(j2) {
		t.Fatal("annotations must be deterministic")
	}
	if len(a1) != 3 { // 24/8=3
		t.Fatalf("24h annotations: got %d, want 3", len(a1))
	}
	for _, a := range a1 {
		switch a.Type {
		case "release", "switch", "alert":
		default:
			t.Fatalf("bad annotation type %q", a.Type)
		}
	}
}
