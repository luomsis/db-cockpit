package labels

import (
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		// Valid expressions
		{"simple match", `host="server1"`, false},
		{"not equal", `host!="server1"`, false},
		{"regex match", `region=~"us-.*"`, false},
		{"regex not match", `region!~"eu-.*"`, false},
		{"and expression", `host="server1" AND region="us-east"`, false},
		{"or expression", `host="server1" OR host="server2"`, false},
		{"parentheses", `(host="server1" OR host="server2") AND region="us-east"`, false},
		{"complex nested", `host!="localhost" AND (region=~"us-.*" OR region=~"eu-.*")`, false},
		{"multiple AND", `a="1" AND b="2" AND c="3"`, false},
		{"multiple OR", `a="1" OR b="2" OR c="3"`, false},
		{"nested parentheses", `((a="1" OR a="2") AND b="3") OR c="4"`, false},
		{"single quotes", `host='server1'`, false},
		{"underscore in key", `host_name="server1"`, false},
		{"numbers in key", `host1="server1"`, false},
		{"regex with special chars", `path=~"/api/.*"`, false},
		{"empty string value", `host=""`, false},

		// Invalid expressions
		{"empty input", "", true},
		{"missing value", `host=`, true},
		{"missing operator", `host"server1"`, true},
		{"unclosed string", `host="server1`, true},
		{"unbalanced parens", `(host="server1"`, true},
		{"invalid operator", `host=="server1"`, true},
		{"missing key", `="server1"`, true},
		{"double AND", `a="1" AND AND b="2"`, true},
		{"trailing AND", `a="1" AND`, true},
		{"leading AND", `AND a="1"`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := Parse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && expr == nil {
				t.Errorf("Parse(%q) returned nil expression", tt.input)
			}
		})
	}
}

func TestParseComparison(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantKey   string
		wantOp    ComparisonOp
		wantValue string
	}{
		{"exact match", `host="server1"`, "host", OpEq, "server1"},
		{"not equal", `host!="server1"`, "host", OpNeq, "server1"},
		{"regex match", `region=~"us-.*"`, "region", OpReMatch, "us-.*"},
		{"regex not match", `region!~"eu-.*"`, "region", OpReNotMatch, "eu-.*"},
		{"empty value", `host=""`, "host", OpEq, ""},
		{"special chars in value", `path="/api/v1/users"`, "path", OpEq, "/api/v1/users"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) error = %v", tt.input, err)
			}

			comp, ok := expr.(*Comparison)
			if !ok {
				t.Fatalf("Expected *Comparison, got %T", expr)
			}

			if comp.Key != tt.wantKey {
				t.Errorf("Key = %q, want %q", comp.Key, tt.wantKey)
			}
			if comp.Op != tt.wantOp {
				t.Errorf("Op = %v, want %v", comp.Op, tt.wantOp)
			}
			if comp.Value != tt.wantValue {
				t.Errorf("Value = %q, want %q", comp.Value, tt.wantValue)
			}
		})
	}
}

func TestParseBinaryExpr(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		wantOp BinaryOp
	}{
		{"AND expression", `a="1" AND b="2"`, OpAnd},
		{"OR expression", `a="1" OR b="2"`, OpOr},
		{"multiple AND", `a="1" AND b="2" AND c="3"`, OpAnd},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) error = %v", tt.input, err)
			}

			binary, ok := expr.(*BinaryExpr)
			if !ok {
				t.Fatalf("Expected *BinaryExpr, got %T", expr)
			}

			if binary.Op != tt.wantOp {
				t.Errorf("Op = %v, want %v", binary.Op, tt.wantOp)
			}
		})
	}
}

func TestParsePrecedence(t *testing.T) {
	// Test operator precedence: AND binds tighter than OR
	tests := []struct {
		name  string
		input string
	}{
		{"AND before OR", `a="1" AND b="2" OR c="3"`},
		{"OR before AND with parens", `(a="1" OR b="2") AND c="3"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) error = %v", tt.input, err)
			}
			if expr == nil {
				t.Errorf("Parse(%q) returned nil", tt.input)
			}
		})
	}
}

func TestBinaryOpString(t *testing.T) {
	tests := []struct {
		op   BinaryOp
		want string
	}{
		{OpAnd, "AND"},
		{OpOr, "OR"},
		{BinaryOp(99), "UNKNOWN"},
	}

	for _, tt := range tests {
		if got := tt.op.String(); got != tt.want {
			t.Errorf("BinaryOp(%d).String() = %q, want %q", tt.op, got, tt.want)
		}
	}
}

func TestComparisonOpString(t *testing.T) {
	tests := []struct {
		op   ComparisonOp
		want string
	}{
		{OpEq, "="},
		{OpNeq, "!="},
		{OpReMatch, "=~"},
		{OpReNotMatch, "!~"},
		{ComparisonOp(99), "UNKNOWN"},
	}

	for _, tt := range tests {
		if got := tt.op.String(); got != tt.want {
			t.Errorf("ComparisonOp(%d).String() = %q, want %q", tt.op, got, tt.want)
		}
	}
}
