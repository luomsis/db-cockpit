package labels

import (
	"fmt"
	"strings"
)

// ToSQL translates an expression to a SQL WHERE clause fragment
func ToSQL(expr Expr) (string, error) {
	return toSQL(expr, false)
}

func toSQL(expr Expr, needsParen bool) (string, error) {
	switch e := expr.(type) {
	case *BinaryExpr:
		left, err := toSQL(e.Left, true)
		if err != nil {
			return "", err
		}
		right, err := toSQL(e.Right, true)
		if err != nil {
			return "", err
		}
		result := fmt.Sprintf("%s %s %s", left, e.Op.String(), right)
		if needsParen {
			result = fmt.Sprintf("(%s)", result)
		}
		return result, nil

	case *Comparison:
		return comparisonToSQL(e), nil

	default:
		return "", fmt.Errorf("unknown expression type: %T", expr)
	}
}

func comparisonToSQL(c *Comparison) string {
	keySQL := fmt.Sprintf("labels->>'%s'", escapeSQLString(c.Key))
	valueSQL := fmt.Sprintf("'%s'", escapeSQLString(c.Value))

	switch c.Op {
	case OpEq:
		return fmt.Sprintf("%s = %s", keySQL, valueSQL)
	case OpNeq:
		return fmt.Sprintf("%s != %s", keySQL, valueSQL)
	case OpReMatch:
		return fmt.Sprintf("%s ~ %s", keySQL, valueSQL)
	case OpReNotMatch:
		return fmt.Sprintf("%s !~ %s", keySQL, valueSQL)
	default:
		return ""
	}
}

func escapeSQLString(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

// Validate checks if the expression is valid and safe
func Validate(expr Expr) error {
	return validateExpr(expr)
}

func validateExpr(expr Expr) error {
	switch e := expr.(type) {
	case *BinaryExpr:
		if err := validateExpr(e.Left); err != nil {
			return err
		}
		return validateExpr(e.Right)

	case *Comparison:
		// Validate the key (should be a valid identifier)
		if e.Key == "" {
			return fmt.Errorf("empty label key")
		}
		// Validate the value for regex operations
		if e.Op == OpReMatch || e.Op == OpReNotMatch {
			// Basic validation - could add more sophisticated regex validation
			if strings.Contains(e.Value, `\x00`) {
				return fmt.Errorf("invalid regex pattern: contains null byte")
			}
		}
		return nil

	default:
		return fmt.Errorf("unknown expression type: %T", expr)
	}
}
