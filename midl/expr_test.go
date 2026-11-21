package midl

import (
	"fmt"
	"strings"
	"testing"

	"github.com/oiweiwei/midl-gen-go/midl/expr"
)

func parseConsts(t *testing.T, body string) *File {
	t.Helper()
	f, err := NewSession(WithoutBuiltins()).Parse(`[uuid(12345678-1234-1234-1234-123456789012), version(1.0)] interface consts { ` + body + ` }`)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func TestConstExpressions(t *testing.T) {
	f := parseConsts(t, `
		const long A = 1 << 5;
		const long B = A + 3 * 2;
		const long C = -7 / 2;
		const long D = B > 30 ? 100 : 200;
		const long E = ~0;
	`)
	body := f.Interfaces[0].Body
	for name, want := range map[string]int64{"A": 32, "B": 38, "C": -3, "D": 100, "E": -1} {
		c := body.Export[name].Const
		if got, ok := c.Value.Int64(); !ok || got != want {
			t.Errorf("%s: got %v, want %d", name, c.Value, want)
		}
	}
}

func TestConstDivisionByZeroIsAParseError(t *testing.T) {
	_, err := NewSession(WithoutBuiltins()).Parse(`[uuid(12345678-1234-1234-1234-123456789012), version(1.0)] interface c { const long A = 1 / 0; }`)
	if err == nil || !strings.Contains(err.Error(), "division by zero") {
		t.Fatalf("want a division by zero error, got %v", err)
	}
}

func TestCoerce(t *testing.T) {
	for _, tc := range []struct {
		name string
		e    Expr
		kind Kind
		ok   bool
	}{
		{"fits short", expr.Int64(1000), TypeInt16, true},
		{"overflows short", expr.Int64(1 << 16), TypeInt16, false},
		{"unsigned fits", expr.Int64(0xFFFFFFFF), TypeUint32, true},
		{"unsigned overflow", expr.Int64(1 << 32), TypeUint32, false},
		{"string to int", expr.String("a"), TypeInt32, false},
		{"symbolic", expr.Ident("x"), TypeInt32, false},
		{"string", expr.String("a"), TypeString, true},
		{"null to void", expr.Null(), TypeVoid, true},
		{"unsupported", expr.Int64(1), TypeStruct, false},
	} {
		_, err := Coerce(tc.e, tc.kind)
		if (err == nil) != tc.ok {
			t.Errorf("%s: err = %v, want ok = %v", tc.name, err, tc.ok)
		}
	}
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
			f, err := NewSession(WithoutBuiltins()).Parse(`[uuid(12345678-1234-1234-1234-123456789012), version(1.0)] interface cbtest { ` + tc.body + ` }`)
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
