package expr

import (
	"errors"
	"fmt"
	"math/big"
)

// maxShift bounds the shift distance so a typo cannot allocate gigabytes.
const maxShift = 1 << 16

func intOperands(op Op, x, y Value) (*big.Int, *big.Int, error) {
	xi, ok := x.Int()
	if !ok {
		return nil, nil, fmt.Errorf("expr: left operand of '%s' must be an integer", op)
	}
	yi, ok := y.Int()
	if !ok {
		return nil, nil, fmt.Errorf("expr: right operand of '%s' must be an integer", op)
	}
	return xi, yi, nil
}

func intValue(v *big.Int) Value { return Value{kind: ValueInt, i: v} }
func boolValue(b bool) Value    { return Value{kind: ValueBool, b: b} }

func evalBinary(op Op, x, y Value) (Value, error) {
	switch op {
	case OpLogicalOr:
		return boolValue(x.Truth() || y.Truth()), nil
	case OpLogicalAnd:
		return boolValue(x.Truth() && y.Truth()), nil
	case OpEq, OpNe:
		eq, err := equal(x, y)
		if err != nil {
			return Value{}, err
		}
		return boolValue(eq == (op == OpEq)), nil
	}

	xi, yi, err := intOperands(op, x, y)
	if err != nil {
		return Value{}, err
	}

	z := new(big.Int)
	switch op {
	case OpLt:
		return boolValue(xi.Cmp(yi) < 0), nil
	case OpLe:
		return boolValue(xi.Cmp(yi) <= 0), nil
	case OpGt:
		return boolValue(xi.Cmp(yi) > 0), nil
	case OpGe:
		return boolValue(xi.Cmp(yi) >= 0), nil
	case OpAdd:
		z.Add(xi, yi)
	case OpSub:
		z.Sub(xi, yi)
	case OpMul:
		z.Mul(xi, yi)
	case OpDiv, OpRem:
		if yi.Sign() == 0 {
			return Value{}, errors.New("expr: division by zero")
		}
		if op == OpDiv {
			z.Quo(xi, yi) // truncated, as in C.
		} else {
			z.Rem(xi, yi)
		}
	case OpAnd:
		z.And(xi, yi)
	case OpOr:
		z.Or(xi, yi)
	case OpXor:
		z.Xor(xi, yi)
	case OpShl, OpShr:
		if !yi.IsUint64() || yi.Uint64() > maxShift {
			return Value{}, errors.New("expr: invalid shift distance")
		}
		if op == OpShl {
			z.Lsh(xi, uint(yi.Uint64()))
		} else {
			z.Rsh(xi, uint(yi.Uint64()))
		}
	default:
		return Value{}, fmt.Errorf("expr: unsupported binary operator '%s'", op)
	}
	return intValue(z), nil
}

// equal compares two literals of the same kind; the kind of x decides (a
// boolean x is compared by truth value).
func equal(x, y Value) (bool, error) {
	switch x.kind {
	case ValueInt:
		xi, yi, err := intOperands(OpEq, x, y)
		if err != nil {
			return false, err
		}
		return xi.Cmp(yi) == 0, nil
	case ValueBool:
		return x.Truth() == y.Truth(), nil
	case ValueString:
		ys, ok := y.Str()
		if !ok {
			return false, errors.New("expr: cannot compare a string with a non-string")
		}
		return x.s == ys, nil
	case ValueChar:
		yr, ok := y.Char()
		if !ok {
			return false, errors.New("expr: cannot compare a character with a non-character")
		}
		return x.r == yr, nil
	case ValueNull:
		if !y.IsNull() {
			return false, errors.New("expr: cannot compare NULL with a non-NULL value")
		}
		return true, nil
	}
	return false, errors.New("expr: values are not comparable")
}

func evalUnary(op Op, x Value) (Value, error) {
	switch op {
	case OpNot:
		return boolValue(!x.Truth()), nil
	case OpDeref:
		return x, nil
	}
	xi, ok := x.Int()
	if !ok {
		return Value{}, fmt.Errorf("expr: operand of unary '%s' must be an integer", op)
	}
	switch op {
	case OpPlus:
		return x, nil
	case OpNeg:
		return intValue(new(big.Int).Neg(xi)), nil
	case OpBitNot:
		return intValue(new(big.Int).Not(xi)), nil
	}
	return Value{}, fmt.Errorf("expr: unsupported unary operator '%s'", op)
}

// Env binds identifiers to expressions during Eval.
type Env interface {
	// LookupExpr returns the expression the name is bound to.
	LookupExpr(name string) (Expr, bool)
}

// EnvFunc adapts a function to Env.
type EnvFunc func(name string) (Expr, bool)

// LookupExpr calls f.
func (f EnvFunc) LookupExpr(name string) (Expr, bool) { return f(name) }

// MapEnv is an Env backed by a map.
type MapEnv map[string]Expr

// LookupExpr returns the binding of the name.
func (m MapEnv) LookupExpr(name string) (Expr, bool) { e, ok := m[name]; return e, ok }

// Always returns an Env that binds every identifier to v.
func Always(v Expr) Env {
	return EnvFunc(func(string) (Expr, bool) { return v, true })
}

// Eval replaces every identifier by its binding in env. If all the bindings
// are constant the result is a literal, otherwise it is a symbolic expression. It fails if an
// identifier is unbound (env may be nil) or if an operation is invalid, e.g.
// a division by zero.
func (e Expr) Eval(env Env) (Expr, error) {
	if e.root == nil {
		return Expr{}, ErrEmpty
	}
	r, err := evalNode(e.root, env)
	if err != nil {
		return Expr{}, err
	}
	if v, ok := r.Const(); ok {
		return lit(v), nil
	}
	return r, nil
}

func evalNode(n *Node, env Env) (Expr, error) {
	switch n.Kind {
	case NodeConst:
		return Expr{n}, nil
	case NodeIdent:
		if env != nil {
			if b, ok := env.LookupExpr(n.Name); ok {
				return b, nil
			}
		}
		return Expr{}, fmt.Errorf("expr: unresolved identifier %q", n.Name)
	case NodeUnary:
		x, err := evalNode(n.X, env)
		if err != nil {
			return Expr{}, err
		}
		return Unary(n.Op, x)
	case NodeBinary:
		x, err := evalNode(n.X, env)
		if err != nil {
			return Expr{}, err
		}
		y, err := evalNode(n.Y, env)
		if err != nil {
			return Expr{}, err
		}
		return Binary(n.Op, x, y)
	case NodeCond:
		c, err := evalNode(n.X, env)
		if err != nil {
			return Expr{}, err
		}
		t, err := evalNode(n.Y, env)
		if err != nil {
			return Expr{}, err
		}
		o, err := evalNode(n.Z, env)
		if err != nil {
			return Expr{}, err
		}
		return Ternary(c, t, o)
	}
	return Expr{}, errors.New("expr: invalid node")
}
