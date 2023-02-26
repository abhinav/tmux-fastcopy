package tmuxfmt

import "fmt"

// Expr is the base interface for expressions accepted by the tmux message
// format.
type Expr interface{ expr() }

// String is a string literal in an expression.
//
//	value
type String string // must not contain tabs

func (String) expr() {}

// Int is an integer literal in an expression.
//
//	42
type Int int

func (Int) expr() {}

// Var is a reference to a variable.
//
//	#{name}
type Var string

func (Var) expr() {}

// Ternary is a conditional operator that evaluates the first expression and
// returns either the second or the third expression based on whether it's
// true.
//
//	#{?cond,then,else}
type Ternary struct {
	Cond Expr
	Then Expr
	Else Expr
}

func (Ternary) expr() {}

// BinaryOp is a binary operation.
type BinaryOp int

// Supported binary operations.
const (
	Equals            BinaryOp = iota // ==
	NotEquals                         // !=
	LessThan                          // <
	GreaterThan                       // >
	LessThanEquals                    // <=
	GreaterThanEquals                 // >=
)

func (op BinaryOp) String() string {
	switch op {
	case Equals:
		return "=="
	case NotEquals:
		return "!="
	case LessThan:
		return "<"
	case GreaterThan:
		return ">"
	case LessThanEquals:
		return "<="
	case GreaterThanEquals:
		return ">="
	default:
		return fmt.Sprintf("BinaryOp(%d)", int(op))
	}
}

// Binary is a binary expression.
//
//	#{op:lhs,rhs}
type Binary struct {
	Op       BinaryOp
	LHS, RHS Expr
}

func (Binary) expr() {}
