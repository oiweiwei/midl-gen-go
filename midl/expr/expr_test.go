package expr

import (
	"math/big"
	"testing"
)

func must(t *testing.T) func(Expr, error) Expr {
	return func(e Expr, err error) Expr {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		return e
	}
}

func bin(t *testing.T, op Op, x, y Expr) Expr { t.Helper(); return must(t)(Binary(op, x, y)) }

func TestConstantFolding(t *testing.T) {
	for _, tc := range []struct {
		name string
		e    Expr
		want int64
	}{
		{"add", bin(t, OpAdd, Int64(2), Int64(3)), 5},
		{"nested", bin(t, OpMul, bin(t, OpAdd, Int64(1), Int64(2)), Int64(4)), 12},
		{"shl", bin(t, OpShl, Int64(1), Int64(5)), 32},
		{"rem sign follows dividend", bin(t, OpRem, Int64(-7), Int64(2)), -1},
		// C truncates toward zero; math/big's Div is Euclidean and gives -4.
		{"div truncates", bin(t, OpDiv, Int64(-7), Int64(2)), -3},
		{"neg", must(t)(Unary(OpNeg, Int64(4))), -4},
		{"bitnot", must(t)(Unary(OpBitNot, Int64(0))), -1},
		{"ternary", must(t)(Ternary(Bool(false), Int64(1), Int64(2))), 2},
		{"compare is bool", bin(t, OpLt, Int64(1), Int64(2)), 0},
	} {
		v, ok := tc.e.Const()
		if !ok {
			t.Errorf("%s: not folded: %s", tc.name, tc.e)
			continue
		}
		if tc.name == "compare is bool" {
			if v.Kind() != ValueBool || !v.Truth() {
				t.Errorf("%s: got %s", tc.name, v)
			}
			continue
		}
		if got, ok := tc.e.Int64(); !ok || got != tc.want {
			t.Errorf("%s: got %v, want %d", tc.name, v, tc.want)
		}
	}
}

func TestSymbolicKeepsConstantPartsAsWritten(t *testing.T) {
	f := Format{BitNot: func(x string) string { return "(^uint32(" + x + "))" }}

	// x & ~7: the width of ~7 comes from x, so it must not be folded to -8.
	e := bin(t, OpAnd, bin(t, OpAdd, Ident("x"), Int64(7)), must(t)(Unary(OpBitNot, Int64(7))))
	if got := e.Format(f); got != "((x+7)&(^uint32(7)))" {
		t.Errorf("got %s", got)
	}
	if got := bin(t, OpAdd, Ident("x"), bin(t, OpShl, Int64(1), Int64(5))).Format(f); got != "(x+(1<<5))" {
		t.Errorf("got %s", got)
	}

	// on its own it is a number.
	if got := must(t)(Unary(OpBitNot, Int64(7))).Format(f); got != "-8" {
		t.Errorf("got %s", got)
	}
	if got := bin(t, OpShl, Int64(1), Int64(5)).Format(f); got != "32" {
		t.Errorf("got %s", got)
	}
}

func TestStringOfConstantIsItsValue(t *testing.T) {
	// the generator spells union case labels with String.
	if got := bin(t, OpOr, Int64(4096), Int64(17)).String(); got != "4113" {
		t.Errorf("got %s", got)
	}
	if got := Char('a').String(); got != "97" {
		t.Errorf("got %s", got)
	}
	if got := String("x").String(); got != `"x"` {
		t.Errorf("got %s", got)
	}
}

func TestConstantErrors(t *testing.T) {
	for name, f := range map[string]func() (Expr, error){
		"div by zero":   func() (Expr, error) { return Binary(OpDiv, Int64(1), Int64(0)) },
		"rem by zero":   func() (Expr, error) { return Binary(OpRem, Int64(1), Int64(0)) },
		"add string":    func() (Expr, error) { return Binary(OpAdd, String("a"), Int64(1)) },
		"huge shift":    func() (Expr, error) { return Binary(OpShl, Int64(1), Int64(1<<40)) },
		"neg shift":     func() (Expr, error) { return Binary(OpShl, Int64(1), Int64(-1)) },
		"empty":         func() (Expr, error) { return Binary(OpAdd, Expr{}, Int64(1)) },
		"not an op":     func() (Expr, error) { return Binary(OpNot, Int64(1), Int64(1)) },
		"neg of bool":   func() (Expr, error) { return Unary(OpNeg, Bool(true)) },
		"string vs int": func() (Expr, error) { return Binary(OpEq, String("a"), Int64(1)) },
	} {
		if _, err := f(); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
}

func TestTruthOfLargeInteger(t *testing.T) {
	// 1<<64 has all-zero low 64 bits: it is true, not false.
	big64 := bin(t, OpShl, Int64(1), Int64(64))
	if !big64.Truth() {
		t.Error("1<<64 must be true")
	}
	if !Int64(-1).Truth() {
		t.Error("-1 must be true")
	}
}

func TestSymbolicStaysSymbolic(t *testing.T) {
	e := bin(t, OpAdd, Ident("n"), Int64(1))
	if e.IsConst() {
		t.Fatal("n+1 must not be constant")
	}
	if got := e.Format(Format{}); got != "(n+1)" {
		t.Errorf("format: %s", got)
	}
	if got := e.String(); got != "(n 1 +)" {
		t.Errorf("string: %s", got)
	}
}

func TestUnaryMinusOfIdent(t *testing.T) {
	// regression: used to be rendered as (+x).
	e := must(t)(Unary(OpNeg, Ident("x")))
	if got := e.Format(Format{}); got != "(-x)" {
		t.Errorf("got %s", got)
	}
}

func TestEval(t *testing.T) {
	// y = (-x + 2) * 4
	e := bin(t, OpMul, bin(t, OpAdd, must(t)(Unary(OpNeg, Ident("x"))), Int64(2)), Int64(4))

	r, err := e.Eval(MapEnv{"x": Int64(5)})
	if err != nil {
		t.Fatal(err)
	}
	if v, ok := r.Int64(); !ok || v != -12 {
		t.Errorf("got %s, want -12", r)
	}

	if _, err := e.Eval(nil); err == nil {
		t.Error("unbound identifier must fail")
	}
	if _, err := e.Eval(MapEnv{}); err == nil {
		t.Error("unbound identifier must fail")
	}

	// binding to a symbolic expression keeps the result symbolic.
	r, err = e.Eval(MapEnv{"x": Ident("z")})
	if err != nil || r.IsConst() {
		t.Fatalf("got %v, %v", r, err)
	}
}

func TestAlwaysBindsEveryOccurrence(t *testing.T) {
	// the old NewArgs store handed out its single value once; the second
	// occurrence failed to resolve.
	e := bin(t, OpSub, bin(t, OpMul, Ident("x"), Ident("x")), Int64(1))
	r, err := e.Eval(Always(Int64(0)))
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := r.Int64(); v != -1 {
		t.Errorf("got %s", r)
	}
}

func TestIdents(t *testing.T) {
	e := bin(t, OpAdd, bin(t, OpMul, Ident("a"), Ident("b")), Ident("a"))
	got := e.Idents()
	if len(got) != 3 || got[0] != "a" || got[1] != "b" || got[2] != "a" {
		t.Errorf("got %v", got)
	}
	if n, ok := e.FirstIdent(); !ok || n != "a" {
		t.Errorf("first: %s %v", n, ok)
	}
	if _, ok := Int64(1).FirstIdent(); ok {
		t.Error("constant has no identifiers")
	}
}

func TestSolve(t *testing.T) {
	r := Ident("len")
	for _, tc := range []struct {
		name string
		e    Expr
		want string // Format of the solution; "" if not solvable
		id   string
	}{
		{"x", Ident("x"), "len", "x"},
		{"x+3", bin(t, OpAdd, Ident("x"), Int64(3)), "(len-3)", "x"},
		{"3+x", bin(t, OpAdd, Int64(3), Ident("x")), "(len-3)", "x"},
		{"x-3", bin(t, OpSub, Ident("x"), Int64(3)), "(len+3)", "x"},
		// regression: used to give len+5.
		{"5-x", bin(t, OpSub, Int64(5), Ident("x")), "(5-len)", "x"},
		{"x*4", bin(t, OpMul, Ident("x"), Int64(4)), "(len/4)", "x"},
		{"x/4", bin(t, OpDiv, Ident("x"), Int64(4)), "(len*4)", "x"},
		// regression: used to give len*6.
		{"6/x", bin(t, OpDiv, Int64(6), Ident("x")), "(6/len)", "x"},
		{"-x", must(t)(Unary(OpNeg, Ident("x"))), "(-len)", "x"},
		{"+x", must(t)(Unary(OpPlus, Ident("x"))), "len", "x"},
		{"(x-1)*4", bin(t, OpMul, bin(t, OpSub, Ident("x"), Int64(1)), Int64(4)), "((len/4)+1)", "x"},
		{"a+b solves a", bin(t, OpAdd, Ident("a"), Ident("b")), "(len-b)", "a"},
		{"shift", bin(t, OpShl, Ident("x"), Int64(1)), "", ""},
		{"deref", must(t)(Unary(OpDeref, Ident("x"))), "", ""},
		{"const", Int64(1), "", ""},
	} {
		s, ok := tc.e.Solve(r)
		if tc.want == "" {
			if ok {
				t.Errorf("%s: must not be solvable, got %s = %s", tc.name, s.Ident, s.Value.Format(Format{}))
			}
			continue
		}
		if !ok {
			t.Errorf("%s: not solvable", tc.name)
			continue
		}
		if got := s.Value.Format(Format{}); got != tc.want || s.Ident != tc.id {
			t.Errorf("%s: got %s = %s, want %s = %s", tc.name, s.Ident, got, tc.id, tc.want)
		}
	}
}

func TestSolveRoundTrip(t *testing.T) {
	// substituting the solution back must give the right-hand side.
	e := bin(t, OpSub, bin(t, OpMul, Int64(3), Ident("x")), Int64(2)) // 3x-2
	s, ok := e.Solve(Ident("r"))
	if !ok {
		t.Fatal("not solvable")
	}
	for _, r := range []int64{4, 10, 100} {
		x, err := s.Value.Eval(MapEnv{"r": Int64(r)})
		if err != nil {
			t.Fatal(err)
		}
		back, err := e.Eval(MapEnv{"x": x})
		if err != nil {
			t.Fatal(err)
		}
		if v, _ := back.Int64(); v != r {
			t.Errorf("r=%d: 3*(%s)-2 = %s", r, x, back)
		}
	}
}

func TestFormat(t *testing.T) {
	f := Format{
		Ident:  func(n string) string { return "o." + n },
		Uint:   func(v uint64) string { return "0x" + big.NewInt(int64(v)).Text(16) },
		BitNot: func(x string) string { return "^uint32(" + x + ")" },
	}
	for _, tc := range []struct {
		name string
		e    Expr
		want string
	}{
		{"hex literal", Int64(255), "0xff"},
		{"negative literal is not passed to Uint", Int64(-1), "-1"},
		{"string", String("a"), `"a"`},
		{"ident", Ident("n"), "o.n"},
		{"deref", must(t)(Unary(OpDeref, Ident("n"))), "o.n"},
		{"bitnot", must(t)(Unary(OpBitNot, Ident("n"))), "^uint32(o.n)"},
		{"not", must(t)(Unary(OpNot, Ident("n"))), "(!o.n)"},
		{"binary", bin(t, OpAdd, Ident("n"), Int64(16)), "(o.n+0x10)"},
		{"cond on ref renders then-branch", must(t)(Ternary(Ident("n"), Ident("a"), Ident("b"))), "o.a"},
		{"cond on comparison", must(t)(Ternary(bin(t, OpGt, Ident("n"), Int64(1)), Ident("a"), Ident("b"))), "((o.n>0x1)?o.a:o.b)"},
	} {
		if got := tc.e.Format(f); got != tc.want {
			t.Errorf("%s: got %s, want %s", tc.name, got, tc.want)
		}
	}
	if got := must(t)(Unary(OpBitNot, Ident("n"))).Format(Format{}); got != "(~n)" {
		t.Errorf("default bitnot: %s", got)
	}
}

func TestEmpty(t *testing.T) {
	var e Expr
	if !e.IsEmpty() || e.String() != "" || e.Format(Format{}) != "" || len(e.Idents()) != 0 {
		t.Error("zero Expr must be empty")
	}
	if _, err := e.Eval(nil); err == nil {
		t.Error("evaluating an empty expression must fail")
	}
	if _, ok := e.Solve(Ident("x")); ok {
		t.Error("solving an empty expression must fail")
	}
}
