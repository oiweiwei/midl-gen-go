package midl

import (
	"context"
	"fmt"
	"math/big"
	"testing"
)

type tExprStore map[string]Expr

func (s tExprStore) LookupExpr(v string) (Expr, bool) {
	e, ok := s[v]
	return e, ok
}

func TestExprTreeResolve(t *testing.T) {

	e := Expr{
		reqParam: true,
		Expr: &ExprTree{
			Op: '*',
			Rval: &ExprTree{
				Value: big.NewInt(4),
			},
			Lval: &ExprTree{
				Op: '+',
				Lval: &ExprTree{
					Op: UMINUS,
					Lval: &ExprTree{
						Op:    IDENT,
						Value: "x",
					},
				},
				Rval: &ExprTree{
					Value: big.NewInt(2),
				},
			},
		}}

	fmt.Println(e.String())

	eval, ok := e.Eval(tExprStore{"x": Expr{Value: big.NewInt(5)}})
	fmt.Println(eval.String(), ok)

	fmt.Println("y", "=", e.Expression(context.Background()))

	// r := &ExprTree{Value: big.NewInt(28)}
	rx, ok := e.Resolve(NewIdent("y"))
	fmt.Println(rx.Value, "=", rx.Expression(context.Background()), ok)

}

// [callback] operations get their own opnum sequence (MIDL numbering) and do not
// consume an operation opnum.
func TestCallbackOpNums(t *testing.T) {

	for _, tc := range []struct {
		name      string
		body      string
		ops       map[string]int
		callbacks map[string]int
	}{
		{"no callbacks", "long A([in] long x); long B([in] long x);", map[string]int{"A": 0, "B": 1}, map[string]int{}},
		{"callback last", "long A([in] long x); long B([in] long x); [callback] long C([in] long x);", map[string]int{"A": 0, "B": 1}, map[string]int{"C": 0}},
		{"callback first", "[callback] long C([in] long x); long A([in] long x); long B([in] long x);", map[string]int{"A": 0, "B": 1}, map[string]int{"C": 0}},
		{"interleaved", "long A([in] long x); [callback] long C([in] long x); long B([in] long x); [callback] long D([in] long x);", map[string]int{"A": 0, "B": 1}, map[string]int{"C": 0, "D": 1}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, err := Parse(`[uuid(12345678-1234-1234-1234-123456789012), version(1.0)] interface cbtest { ` + tc.body + ` }`)
			if err != nil {
				t.Fatal(err)
			}
			if len(f.Interfaces) != 1 {
				t.Fatalf("interfaces: %d", len(f.Interfaces))
			}
			got := func(ops []*Operation) map[string]int {
				m := map[string]int{}
				for _, o := range ops {
					m[o.Name] = o.OpNum
				}
				return m
			}
			if g := got(f.Interfaces[0].Body.Operations); fmt.Sprint(g) != fmt.Sprint(tc.ops) {
				t.Errorf("operations: got %v, want %v", g, tc.ops)
			}
			if g := got(f.Interfaces[0].Body.Callbacks); fmt.Sprint(g) != fmt.Sprint(tc.callbacks) {
				t.Errorf("callbacks: got %v, want %v", g, tc.callbacks)
			}
		})
	}
}
