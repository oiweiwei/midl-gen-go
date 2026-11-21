// Package expr implements the constant/symbolic expressions of IDL attributes
// and constant declarations, e.g. size_is(cb * 2 + 1) or const long X = 1 << 5.
//
// An Expr is an immutable tree. It is either constant (all leaves are
// literals; the tree is folded to a single literal when it is built) or
// symbolic (it refers to identifiers, typically sibling struct fields or
// parameters, that are bound later with Eval or inverted with Solve).
//
// The package knows nothing about the IDL grammar or about Go: how a symbolic
// expression is rendered in a target language is up to the Format passed to
// Expr.Format.
package expr

// Op is an operator of a unary or binary expression node.
type Op uint8

// Binary operators.
const (
	OpInvalid Op = iota

	OpAdd // x + y
	OpSub // x - y
	OpMul // x * y
	OpDiv // x / y, truncated toward zero as in C
	OpRem // x % y, sign follows x as in C
	OpAnd // x & y
	OpOr  // x | y
	OpXor // x ^ y
	OpShl // x << y
	OpShr // x >> y

	OpEq // x == y
	OpNe // x != y
	OpLt // x < y
	OpLe // x <= y
	OpGt // x > y
	OpGe // x >= y

	OpLogicalAnd // x && y
	OpLogicalOr  // x || y

	// Unary operators.

	OpPlus   // +x
	OpNeg    // -x
	OpBitNot // ~x
	OpNot    // !x
	OpDeref  // *x; the operand is a pointer-typed identifier, the value is the same.
)

var opNames = [...]string{
	OpAdd: "+", OpSub: "-", OpMul: "*", OpDiv: "/", OpRem: "%",
	OpAnd: "&", OpOr: "|", OpXor: "^", OpShl: "<<", OpShr: ">>",
	OpEq: "==", OpNe: "!=", OpLt: "<", OpLe: "<=", OpGt: ">", OpGe: ">=",
	OpLogicalAnd: "&&", OpLogicalOr: "||",
	OpPlus: "+", OpNeg: "-", OpBitNot: "~", OpNot: "!", OpDeref: "*",
}

// String returns the C spelling of the operator.
func (op Op) String() string {
	if int(op) < len(opNames) && opNames[op] != "" {
		return opNames[op]
	}
	return "?"
}

// IsUnary reports whether op is a unary operator.
func (op Op) IsUnary() bool { return op >= OpPlus && op <= OpDeref }

// IsBinary reports whether op is a binary operator.
func (op Op) IsBinary() bool { return op >= OpAdd && op <= OpLogicalOr }

// IsBool reports whether the result of the binary operator is a boolean
// (comparisons and logical operators).
func (op Op) IsBool() bool { return op >= OpEq && op <= OpLogicalOr }
