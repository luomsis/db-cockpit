package handler

import "testing"

/* 插件域纯函数单测（无 PG；聚合/发现/定级流程的集成测试见 tooldefs_pg_test.go） */

func TestValidToolTransition(t *testing.T) {
	cases := []struct {
		from, to string
		want     bool
	}{
		{"draft", "active", true},
		{"draft", "deprecated", true},
		{"active", "deprecated", true},
		{"deprecated", "active", false}, // 不可复活（重新发现生成新草案）
		{"active", "draft", false},
		{"deprecated", "draft", false},
		{"", "active", false},
		{"draft", "", false},
	}
	for _, c := range cases {
		if got := validToolTransition(c.from, c.to); got != c.want {
			t.Errorf("validToolTransition(%q→%q) = %v, want %v", c.from, c.to, got, c.want)
		}
	}
}

// MVP 仅 http：stdio 明确拒绝（D15）
func TestNormalizeTransportRejectsStdio(t *testing.T) {
	cases := []struct {
		in   string
		want string
		ok   bool
	}{
		{"", "http", true},
		{"http", "http", true},
		{"stdio", "", false},
		{"grpc", "", false},
		{"HTTP", "", false},
	}
	for _, c := range cases {
		got, ok := normalizeTransport(c.in)
		if ok != c.ok || (ok && got != c.want) {
			t.Errorf("normalizeTransport(%q) = (%q,%v), want (%q,%v)", c.in, got, ok, c.want, c.ok)
		}
	}
}

func TestPrefixToolName(t *testing.T) {
	if n, ok := prefixToolName("metrics-mcp", "get_cpu"); !ok || n != "metrics-mcp.get_cpu" {
		t.Errorf("prefixToolName = (%q,%v)", n, ok)
	}
	for _, bad := range [][2]string{{"", "x"}, {"s", ""}, {".", "x"}, {"s", "."}} {
		if _, ok := prefixToolName(bad[0], bad[1]); ok {
			t.Errorf("prefixToolName(%q,%q) 不应合法", bad[0], bad[1])
		}
	}
}

func TestValidateBaseURL(t *testing.T) {
	valid := []string{"http://a", "https://a/mcp"}
	invalid := []string{"", "ftp://a", "example.com", "http:/missing"}
	for _, v := range valid {
		if !validateBaseURL(v) {
			t.Errorf("validateBaseURL(%q) 应合法", v)
		}
	}
	for _, v := range invalid {
		if validateBaseURL(v) {
			t.Errorf("validateBaseURL(%q) 应非法", v)
		}
	}
}

func TestMcpStatusOrDefault(t *testing.T) {
	if s := mcpStatusOrDefault(nil); s != "active" {
		t.Errorf("nil → %q, want active", s)
	}
	if s := mcpStatusOrDefault(strp("")); s != "active" {
		t.Errorf("空串 → %q, want active", s)
	}
	if s := mcpStatusOrDefault(strp("deprecated")); s != "deprecated" {
		t.Errorf("deprecated → %q", s)
	}
}

func strp(s string) *string { return &s }
