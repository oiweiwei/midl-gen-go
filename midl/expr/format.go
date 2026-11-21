package expr

import (
	"fmt"
	"strings"
)

// Format controls how Expr.Format renders an expression. The zero Format
// renders C-like infix notation with every binary operation parenthesized.
type Format struct {
	// Ident renders a reference to a name; nil renders the name as is.
	Ident func(name string) string
	// Uint renders a non-negative integer literal; nil renders decimal.
	Uint func(v uint64) string
	// BitNot renders the bitwise negation of an already rendered operand; nil
	// renders "(~x)".
	BitNot func(operand string) string
}

// Format renders the expression in the notation described by f. A constant
// expression renders as its value; a symbolic one is rendered as written, so
// a constant part of it keeps its operators (see Expr).
//
// A conditional whose test is a plain reference (a ? b : c, where a is a name
// or *name) renders as its "then" branch only. This reproduces what the Go
// generator has always emitted for such expressions.
func (e Expr) Format(f Format) string {
	if e.root == nil {
		return ""
	}
	// a constant is rendered by its value; its sub-expressions are not
	// spelled out.
	if v, ok := e.Const(); ok && e.root.Kind != NodeConst {
		return f.node(&Node{Kind: NodeConst, Value: v})
	}
	return f.node(e.root)
}

func (f Format) node(n *Node) string {
	switch n.Kind {
	case NodeConst:
		if i, ok := n.Value.Int(); ok {
			if f.Uint != nil && i.IsUint64() {
				return f.Uint(i.Uint64())
			}
			return i.String()
		}
		if n.Value.IsNull() {
			return "nil"
		}
		return n.Value.String()
	case NodeIdent:
		if f.Ident != nil {
			return f.Ident(n.Name)
		}
		return n.Name
	case NodeUnary:
		x := f.node(n.X)
		switch n.Op {
		case OpDeref:
			return x
		case OpBitNot:
			if f.BitNot != nil {
				return f.BitNot(x)
			}
		}
		return "(" + n.Op.String() + x + ")"
	case NodeBinary:
		return "(" + f.node(n.X) + n.Op.String() + f.node(n.Y) + ")"
	case NodeCond:
		if n.X.IsRef() {
			return f.node(n.Y)
		}
		return fmt.Sprintf("(%s?%s:%s)", f.node(n.X), f.node(n.Y), f.node(n.Z))
	}
	return ""
}

// postfix is the debug form used by Expr.String.
func postfix(n *Node) string {
	switch n.Kind {
	case NodeConst:
		return n.Value.String()
	case NodeIdent:
		return n.Name
	case NodeUnary:
		if n.Op == OpDeref {
			return "*" + postfix(n.X)
		}
		return "(" + postfix(n.X) + " " + n.Op.String() + ")"
	case NodeBinary:
		return "(" + postfix(n.X) + " " + postfix(n.Y) + " " + n.Op.String() + ")"
	case NodeCond:
		return "(" + strings.Join([]string{postfix(n.X), postfix(n.Y), postfix(n.Z)}, " ") + " ?:)"
	}
	return ""
}
