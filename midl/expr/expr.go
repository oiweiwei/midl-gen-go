package expr

import (
	"encoding/json"
	"errors"
	"math/big"
)

// NodeKind tells what a Node is.
type NodeKind uint8

// Node kinds.
const (
	NodeConst  NodeKind = iota + 1 // a literal: Value
	NodeIdent                      // a reference to a name: Name
	NodeUnary                      // Op X
	NodeBinary                     // X Op Y
	NodeCond                       // X ? Y : Z
)

// Node is a vertex of the expression tree. Nodes are immutable: never modify
// a Node obtained from an Expr.
type Node struct {
	Kind NodeKind
	// Op is the operator of a NodeUnary or NodeBinary.
	Op Op
	// Value is the literal of a NodeConst.
	Value Value
	// Name is the identifier of a NodeIdent.
	Name string
	// X is the operand of a unary node, the left operand of a binary node and
	// the condition of a conditional node. Y is the right operand of a binary
	// node and the "then" branch of a conditional node. Z is the "else" branch.
	X, Y, Z *Node
}

// IsRef reports whether the node is a plain reference to a name, i.e. an
// identifier or a dereference of one (*name).
func (n *Node) IsRef() bool {
	return n != nil && (n.Kind == NodeIdent || n.Kind == NodeUnary && n.Op == OpDeref)
}

// IsBool reports whether the node is a comparison or a logical operation.
func (n *Node) IsBool() bool {
	return n != nil && n.Kind == NodeBinary && n.Op.IsBool()
}

// Expr is an expression. The zero Expr is empty (an absent attribute).
// Expr values are immutable and cheap to copy.
//
// An Expr always keeps the tree it was built from, even when it has no
// identifiers and so has a value (see Const). The value of an IDL integer
// expression has no width of its own, e.g. ~7 is -8 as a number but
// 0xFFFFFFF8 once it is used with a 32-bit operand, so a printer that embeds
// a constant sub-expression in a symbolic one needs the original form.
type Expr struct {
	root *Node
}

// ErrEmpty is returned when an empty Expr is used as an operand.
var ErrEmpty = errors.New("expr: empty expression")

// FromNode wraps a node (typically a sub-tree of another expression).
func FromNode(n *Node) Expr { return Expr{n} }

// Int returns an integer literal.
func Int(v *big.Int) Expr { return lit(Value{kind: ValueInt, i: v}) }

// Int64 returns an integer literal.
func Int64(v int64) Expr { return Int(big.NewInt(v)) }

// Float returns a floating point literal.
func Float(v *big.Float) Expr { return lit(Value{kind: ValueFloat, f: v}) }

// String returns a string literal.
func String(v string) Expr { return lit(Value{kind: ValueString, s: v}) }

// Char returns a character literal.
func Char(v rune) Expr { return lit(Value{kind: ValueChar, r: v}) }

// Bool returns a boolean literal.
func Bool(v bool) Expr { return lit(Value{kind: ValueBool, b: v}) }

// Null returns the NULL literal.
func Null() Expr { return lit(Value{}) }

// Literal returns an expression holding the value.
func Literal(v Value) Expr { return lit(v) }

// Ident returns a reference to the name.
func Ident(name string) Expr { return Expr{&Node{Kind: NodeIdent, Name: name}} }

func lit(v Value) Expr { return Expr{&Node{Kind: NodeConst, Value: v}} }

// validated wraps the node; if it has no identifiers it is evaluated once to
// report an invalid operation (e.g. a division by zero) to the caller.
func validated(n *Node) (Expr, error) {
	if !hasIdent(n) {
		if _, err := fold(n); err != nil {
			return Expr{}, err
		}
	}
	return Expr{n}, nil
}

func hasIdent(n *Node) bool {
	return n != nil && (n.Kind == NodeIdent || hasIdent(n.X) || hasIdent(n.Y) || hasIdent(n.Z))
}

// fold evaluates a tree that has no identifiers.
func fold(n *Node) (Value, error) {
	switch n.Kind {
	case NodeConst:
		return n.Value, nil
	case NodeUnary:
		x, err := fold(n.X)
		if err != nil {
			return Value{}, err
		}
		return evalUnary(n.Op, x)
	case NodeBinary:
		x, err := fold(n.X)
		if err != nil {
			return Value{}, err
		}
		y, err := fold(n.Y)
		if err != nil {
			return Value{}, err
		}
		return evalBinary(n.Op, x, y)
	case NodeCond:
		c, err := fold(n.X)
		if err != nil {
			return Value{}, err
		}
		t, err := fold(n.Y)
		if err != nil {
			return Value{}, err
		}
		o, err := fold(n.Z)
		if err != nil {
			return Value{}, err
		}
		if c.Truth() {
			return t, nil
		}
		return o, nil
	}
	return Value{}, errors.New("expr: not a constant expression")
}

// Unary applies the unary operator. It fails if x is constant and the
// operation is invalid for it.
func Unary(op Op, x Expr) (Expr, error) {
	if !op.IsUnary() {
		return Expr{}, errors.New("expr: not a unary operator: " + op.String())
	}
	if x.IsEmpty() {
		return Expr{}, ErrEmpty
	}
	return validated(&Node{Kind: NodeUnary, Op: op, X: x.root})
}

// Binary applies the binary operator. It fails if both operands are constant
// and the operation is invalid for them, e.g. a division by zero.
func Binary(op Op, x, y Expr) (Expr, error) {
	if !op.IsBinary() {
		return Expr{}, errors.New("expr: not a binary operator: " + op.String())
	}
	if x.IsEmpty() || y.IsEmpty() {
		return Expr{}, ErrEmpty
	}
	return validated(&Node{Kind: NodeBinary, Op: op, X: x.root, Y: y.root})
}

// Ternary builds "cond ? then : otherwise".
func Ternary(cond, then, otherwise Expr) (Expr, error) {
	if cond.IsEmpty() || then.IsEmpty() || otherwise.IsEmpty() {
		return Expr{}, ErrEmpty
	}
	return validated(&Node{Kind: NodeCond, X: cond.root, Y: then.root, Z: otherwise.root})
}

// IsEmpty reports whether the expression is absent.
func (e Expr) IsEmpty() bool { return e.root == nil }

// Root returns the root of the expression tree (nil for an empty expression).
func (e Expr) Root() *Node { return e.root }

// IsConst reports whether the expression has a value, i.e. it does not
// depend on any identifier.
func (e Expr) IsConst() bool { return e.root != nil && !hasIdent(e.root) }

// Const returns the value of a constant expression.
func (e Expr) Const() (Value, bool) {
	if !e.IsConst() {
		return Value{}, false
	}
	v, err := fold(e.root)
	return v, err == nil
}

// Int returns the value of a constant integer expression.
func (e Expr) Int() (*big.Int, bool) {
	v, ok := e.Const()
	if !ok {
		return nil, false
	}
	return v.Int()
}

// Int64 returns the value of a constant integer expression that fits int64.
func (e Expr) Int64() (int64, bool) {
	v, ok := e.Int()
	if !ok || !v.IsInt64() {
		return 0, false
	}
	return v.Int64(), true
}

// Uint64 returns the value of a constant integer expression that fits uint64.
func (e Expr) Uint64() (uint64, bool) {
	v, ok := e.Int()
	if !ok || !v.IsUint64() {
		return 0, false
	}
	return v.Uint64(), true
}

// Str returns the value of a constant string expression.
func (e Expr) Str() (string, bool) {
	v, ok := e.Const()
	if !ok {
		return "", false
	}
	return v.Str()
}

// Char returns the value of a constant character expression.
func (e Expr) Char() (rune, bool) {
	v, ok := e.Const()
	if !ok {
		return 0, false
	}
	return v.Char()
}

// IsNull reports whether the expression is the NULL literal.
func (e Expr) IsNull() bool {
	v, ok := e.Const()
	return ok && v.IsNull()
}

// Truth returns the C truth value of a constant expression (false if the
// expression is not constant).
func (e Expr) Truth() bool {
	v, _ := e.Const()
	return v.Truth()
}

// IdentName returns the name if the expression is exactly one identifier.
func (e Expr) IdentName() (string, bool) {
	if e.root == nil || e.root.Kind != NodeIdent {
		return "", false
	}
	return e.root.Name, true
}

// Idents returns the identifiers the expression refers to, in the order of a
// left-to-right walk; an identifier used twice is listed twice.
func (e Expr) Idents() []string {
	var names []string
	var walk func(*Node)
	walk = func(n *Node) {
		if n == nil {
			return
		}
		if n.Kind == NodeIdent {
			names = append(names, n.Name)
		}
		walk(n.X)
		walk(n.Y)
		walk(n.Z)
	}
	walk(e.root)
	return names
}

// FirstIdent returns the first identifier of the expression in a left-to-right
// walk.
func (e Expr) FirstIdent() (string, bool) {
	if names := e.Idents(); len(names) > 0 {
		return names[0], true
	}
	return "", false
}

// String returns the value of a constant expression (strings quoted), and
// the postfix notation of a symbolic one, e.g. "(x 4 *)". The generator uses
// it where a constant must be spelled out, e.g. for union case labels.
func (e Expr) String() string {
	if e.root == nil {
		return ""
	}
	if v, ok := e.Const(); ok {
		return v.String()
	}
	return postfix(e.root)
}

// MarshalJSON encodes the expression as its String form.
func (e Expr) MarshalJSON() ([]byte, error) { return json.Marshal(e.String()) }
