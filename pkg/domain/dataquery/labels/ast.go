package labels

// Expr is the interface for all expression nodes
type Expr interface {
	exprNode()
}

// BinaryExpr represents a binary expression (AND, OR)
type BinaryExpr struct {
	Op    BinaryOp
	Left  Expr
	Right Expr
}

func (e *BinaryExpr) exprNode() {}

// Comparison represents a comparison expression (=, !=, =~, !~)
type Comparison struct {
	Key   string
	Op    ComparisonOp
	Value string
}

func (e *Comparison) exprNode() {}

// BinaryOp represents binary operators
type BinaryOp int

const (
	OpAnd BinaryOp = iota
	OpOr
)

// ComparisonOp represents comparison operators
type ComparisonOp int

const (
	OpEq         ComparisonOp = iota // =
	OpNeq                            // !=
	OpReMatch                        // =~
	OpReNotMatch                     // !~
)

// String returns the string representation of BinaryOp
func (op BinaryOp) String() string {
	switch op {
	case OpAnd:
		return "AND"
	case OpOr:
		return "OR"
	default:
		return "UNKNOWN"
	}
}

// String returns the string representation of ComparisonOp
func (op ComparisonOp) String() string {
	switch op {
	case OpEq:
		return "="
	case OpNeq:
		return "!="
	case OpReMatch:
		return "=~"
	case OpReNotMatch:
		return "!~"
	default:
		return "UNKNOWN"
	}
}
