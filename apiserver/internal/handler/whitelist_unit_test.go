package handler

import (
	"testing"
	"time"

	"db-cockpit/apiserver/internal/model"
)

/* 数据面白名单消费的纯函数单测（无 PG 依赖；聚合/映射/格式化/回退转换） */

func TestMapSeverity(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"Critical", "P1"}, {"critical", "P1"}, {"CRITICAL", "P1"}, {" Critical ", "P1"},
		{"Major", "P2"}, {"major", "P2"},
		{"Minor", "P3"}, {"Warning", "P3"}, {"Info", "P3"}, {"", "P3"}, {"Fatalish", "P3"},
	}
	for _, c := range cases {
		if got := mapSeverity(c.in); got != c.want {
			t.Errorf("mapSeverity(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestFmtMs(t *testing.T) {
	cases := []struct {
		in   float64
		want string
	}{
		{12800, "12.8s"}, {6400, "6.4s"}, {1000, "1.0s"},
		{950, "950ms"}, {0, "0ms"}, {999.9, "999ms"},
	}
	for _, c := range cases {
		if got := fmtMs(c.in); got != c.want {
			t.Errorf("fmtMs(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestFmtRows(t *testing.T) {
	nilv := (*int64)(nil)
	v := func(i int64) *int64 { return &i }
	cases := []struct {
		in   *int64
		want string
	}{
		{nilv, "—"}, {v(0), "0"}, {v(12), "12"}, {v(123), "123"},
		{v(1234), "1,234"}, {v(123456), "123,456"}, {v(4380012), "4,380,012"},
		{v(-1234), "-1,234"},
	}
	for _, c := range cases {
		if got := fmtRows(c.in); got != c.want {
			t.Errorf("fmtRows(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestBuildAlertItems(t *testing.T) {
	base := time.Date(2026, 8, 18, 10, 0, 0, 0, time.Local)
	aggs := []alertAgg{
		{ObjectName: "b", AlertName: "T2", AlertLevel: "Major", FirstFired: base.Add(-2 * time.Hour), Cnt: 2},
		{ObjectName: "c", AlertName: "T3", AlertLevel: "Fatalish", FirstFired: base, Cnt: 1},
		{ObjectName: "a", AlertName: "T1", AlertLevel: "Critical", FirstFired: base.Add(-1 * time.Hour), Cnt: 3},
		{ObjectName: "a2", AlertName: "T1", AlertLevel: "critical", FirstFired: base.Add(-3 * time.Hour), Cnt: 9},
	}
	items := buildAlertItems(aggs)
	if len(items) != 4 {
		t.Fatalf("items len = %d, want 4", len(items))
	}
	// 排序：P1 在前（首触时间倒序），未知级别映射 P3 殿后
	wantOrder := []string{"a", "a2", "b", "c"}
	for i, w := range wantOrder {
		if items[i].Name != w {
			t.Errorf("items[%d].Name = %q, want %q", i, items[i].Name, w)
		}
	}
	if items[0].Time != base.Add(-1*time.Hour).Format("01-02 15:04") {
		t.Errorf("items[0].Time = %q, want %q", items[0].Time, base.Add(-1*time.Hour).Format("01-02 15:04"))
	}
	if items[3].Severity != "P3" {
		t.Errorf("未知级别应映射 P3，got %q", items[3].Severity)
	}
}

func TestBuildSlowItems(t *testing.T) {
	rows := func(i int64) *int64 { return &i }
	aggs := []slowAgg{
		{SqlText: "S1", Db: "d1", AvgMs: 2000, MaxRows: rows(1234567), Cnt: 3},
		{SqlText: "S2", Db: "d2", AvgMs: 500, MaxRows: nil, Cnt: 1},
	}
	items := buildSlowItems(aggs)
	if items[0].Time != "2.0s" || items[0].Rows != "1,234,567" || items[0].Count != 3 {
		t.Errorf("items[0] = %+v", items[0])
	}
	if items[1].Time != "500ms" || items[1].Rows != "—" || items[1].Count != 1 {
		t.Errorf("items[1] = %+v", items[1])
	}
}

// 回退路径：白名单空表时旧演示表行 → 同一前端形状
func TestLegacyItems(t *testing.T) {
	alerts := legacyAlertItems([]model.AlertRecord{{Name: "n", Severity: "P1", Title: "t", Time: "08-18 14:32", Count: 6}})
	if len(alerts) != 1 || alerts[0].Name != "n" || alerts[0].Count != 6 || alerts[0].Time != "08-18 14:32" {
		t.Errorf("legacyAlertItems = %+v", alerts)
	}
	slow := legacySlowItems([]model.SlowSql{{Sql: "s", Db: "d", Time: "12.8s", Rows: "4,380,012", Count: 342}})
	if len(slow) != 1 || slow[0].Sql != "s" || slow[0].Count != 342 {
		t.Errorf("legacySlowItems = %+v", slow)
	}
}

func TestStatusLabel(t *testing.T) {
	cases := map[int]string{0: "待提交", 1: "审批中", 2: "执行中", 3: "已完成", 4: "已取消", 99: "状态99"}
	for code, want := range cases {
		if got := statusLabel(code); got != want {
			t.Errorf("statusLabel(%d) = %q, want %q", code, got, want)
		}
	}
}

func TestParseTimeParam(t *testing.T) {
	// 三种合法布局
	for _, v := range []string{"2026-08-18T10:00:00+08:00", "2026-08-18 10:00", "2026-08-18"} {
		if _, err := parseTimeParam(v); err != nil {
			t.Errorf("parseTimeParam(%q) 不应报错: %v", v, err)
		}
	}
	// 容错：query 未编码的 +08:00 被解析为空格
	if _, err := parseTimeParam("2026-08-18T02:00:00 08:00"); err != nil {
		t.Errorf("parseTimeParam 对空格腐蚀的时区应容错: %v", err)
	}
	for _, v := range []string{"", "abc", "2026-13-45", "18/08/2026"} {
		if _, err := parseTimeParam(v); err == nil {
			t.Errorf("parseTimeParam(%q) 应报错", v)
		}
	}
}
