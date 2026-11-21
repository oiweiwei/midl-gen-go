package expr

// Solution is the result of Solve: the identifier, expressed through the
// right-hand side.
type Solution struct {
	// Ident is the solved identifier.
	Ident string
	// Value is the expression Ident is equal to.
	Value Expr
}

// Solve inverts the expression: if e == rhs, it returns the first identifier
// of e (left-to-right) and the expression it equals, e.g. for e = "n * 4 - 1"
// and rhs = "len" it returns n = (len + 1) / 4. Other identifiers of e stay
// symbolic in the result.
//
// Only the linear operators +, -, *, / and unary + and - are inverted; Solve
// fails for anything else (shifts, bit operations, comparisons, conditions,
// dereferences) and if the inversion is not defined (division by constant
// zero).
func (e Expr) Solve(rhs Expr) (Solution, bool) {
	if e.root == nil || rhs.IsEmpty() {
		return Solution{}, false
	}
	return solve(e.root, rhs)
}

func solve(n *Node, rhs Expr) (Solution, bool) {
	switch n.Kind {
	case NodeIdent:
		return Solution{Ident: n.Name, Value: rhs}, true
	case NodeUnary:
		switch n.Op {
		case OpPlus:
			return solve(n.X, rhs)
		case OpNeg:
			// -x = r  =>  x = -r
			inv, err := Unary(OpNeg, rhs)
			if err != nil {
				return Solution{}, false
			}
			return solve(n.X, inv)
		}
	case NodeBinary:
		x, y := Expr{n.X}, Expr{n.Y}
		// the operand to solve for is tried first on the left.
		for _, side := range []struct {
			target *Node
			// the inverse expression for the operand "target" when it is on the
			// left (x op y = r) or on the right of the operator.
			inv func() (Expr, error)
		}{
			{n.X, func() (Expr, error) { return inverseLeft(n.Op, rhs, y) }},
			{n.Y, func() (Expr, error) { return inverseRight(n.Op, rhs, x) }},
		} {
			if !hasIdent(side.target) {
				continue
			}
			inv, err := side.inv()
			if err != nil {
				continue
			}
			if s, ok := solve(side.target, inv); ok {
				return s, true
			}
		}
	}
	return Solution{}, false
}

// inverseLeft returns the expression for x given "x op y == r".
func inverseLeft(op Op, r, y Expr) (Expr, error) {
	switch op {
	case OpAdd:
		return Binary(OpSub, r, y)
	case OpSub:
		return Binary(OpAdd, r, y)
	case OpMul:
		return Binary(OpDiv, r, y)
	case OpDiv:
		return Binary(OpMul, r, y)
	}
	return Expr{}, ErrEmpty
}

// inverseRight returns the expression for y given "x op y == r".
func inverseRight(op Op, r, x Expr) (Expr, error) {
	switch op {
	case OpAdd:
		return Binary(OpSub, r, x)
	case OpSub:
		return Binary(OpSub, x, r)
	case OpMul:
		return Binary(OpDiv, r, x)
	case OpDiv:
		return Binary(OpDiv, x, r)
	}
	return Expr{}, ErrEmpty
}
