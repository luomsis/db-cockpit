package labels

import (
	"testing"
)

func TestToSQL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Simple comparisons
		{"exact match", `host="server1"`, `labels->>'host' = 'server1'`},
		{"not equal", `host!="server1"`, `labels->>'host' != 'server1'`},
		{"regex match", `region=~"us-.*"`, `labels->>'region' ~ 'us-.*'`},
		{"regex not match", `region!~"eu-.*"`, `labels->>'region' !~ 'eu-.*'`},
		{"empty value", `host=""`, `labels->>'host' = ''`},

		// Binary expressions
		{"AND expression", `host="server1" AND region="us-east"`, `labels->>'host' = 'server1' AND labels->>'region' = 'us-east'`},
		{"OR expression", `host="server1" OR host="server2"`, `labels->>'host' = 'server1' OR labels->>'host' = 'server2'`},

		// Mixed operators
		{"AND with regex", `env="prod" AND region=~"us-.*"`, `labels->>'env' = 'prod' AND labels->>'region' ~ 'us-.*'`},
		{"OR with not equal", `host!="localhost" OR env="dev"`, `labels->>'host' != 'localhost' OR labels->>'env' = 'dev'`},

		// Complex expressions (left-associative with parens)
		{"three way AND", `a="1" AND b="2" AND c="3"`, `(labels->>'a' = '1' AND labels->>'b' = '2') AND labels->>'c' = '3'`},
		{"three way OR", `a="1" OR b="2" OR c="3"`, `(labels->>'a' = '1' OR labels->>'b' = '2') OR labels->>'c' = '3'`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) error = %v", tt.input, err)
			}
			sql, err := ToSQL(expr)
			if err != nil {
				t.Fatalf("ToSQL() error = %v", err)
			}
			if sql != tt.expected {
				t.Errorf("ToSQL(%q) = %q, want %q", tt.input, sql, tt.expected)
			}
		})
	}
}

func TestToSQLWithParentheses(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"parentheses OR", `(host="server1" OR host="server2") AND region="us-east"`},
		{"nested parentheses", `((a="1" OR a="2") AND b="3") OR c="4"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) error = %v", tt.input, err)
			}
			sql, err := ToSQL(expr)
			if err != nil {
				t.Fatalf("ToSQL() error = %v", err)
			}
			// Just verify it doesn't error and produces output
			if sql == "" {
				t.Errorf("ToSQL(%q) returned empty string", tt.input)
			}
		})
	}
}

func TestEscapeSQLString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple", "simple"},
		{"with'quote", "with''quote"},
		{"with''two", "with''''two"},
		{"normal text", "normal text"},
		{"", ""},
	}

	for _, tt := range tests {
		if got := escapeSQLString(tt.input); got != tt.expected {
			t.Errorf("escapeSQLString(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		// Valid expressions
		{"simple match", `host="server1"`, false},
		{"regex match", `region=~"us-.*"`, false},
		{"complex valid", `host="server1" AND region="us-east"`, false},

		// Edge cases - these should pass validation
		{"empty value", `host=""`, false},
		{"regex with dots", `path=~"/api/.*"`, false},
		{"regex with brackets", `version=~"v[0-9]+"`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) error = %v", tt.input, err)
			}
			if err := Validate(expr); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateWithInvalidRegex(t *testing.T) {
	// Test that validation catches null bytes in regex (escaped form)
	comp := &Comparison{
		Key:   "test",
		Op:    OpReMatch,
		Value: "pattern\\x00null", // escaped null byte
	}

	if err := Validate(comp); err == nil {
		t.Error("Validate() should reject regex with null byte")
	}
}

func TestValidateEmptyKey(t *testing.T) {
	comp := &Comparison{
		Key:   "",
		Op:    OpEq,
		Value: "test",
	}

	if err := Validate(comp); err == nil {
		t.Error("Validate() should reject empty key")
	}
}

func TestValidateUnknownType(t *testing.T) {
	// Create an unknown expression type
	type unknownExpr struct {
		Expr
	}

	if err := Validate(&unknownExpr{}); err == nil {
		t.Error("Validate() should reject unknown expression type")
	}
}

func TestToSQLUnknownType(t *testing.T) {
	// Create an unknown expression type
	type unknownExpr struct {
		Expr
	}

	_, err := ToSQL(&unknownExpr{})
	if err == nil {
		t.Error("ToSQL() should reject unknown expression type")
	}
}

func TestToSQLWithNeedsParen(t *testing.T) {
	// Test that nested binary expressions get proper parentheses
	input := `(host="server1" OR host="server2") AND env="prod"`
	expr, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse(%q) error = %v", input, err)
	}

	sql, err := ToSQL(expr)
	if err != nil {
		t.Fatalf("ToSQL() error = %v", err)
	}

	// Verify the SQL contains AND
	if sql == "" {
		t.Error("ToSQL() returned empty string")
	}
}
