package goprint

import (
	"bytes"
	"strings"
	"testing"
)

func write(f func(w *Writer)) string {
	var b bytes.Buffer
	f(NewWriter(&b))
	return b.String()
}

func TestAssembleKeepsOnlyUsedImports(t *testing.T) {
	imps := NewImports(
		Import{Path: "context"},
		Import{Path: "fmt"},
		Import{Path: "github.com/x/ndr"},
		Import{Path: "github.com/x/dcerpc", Name: "rpc"},
		Import{Path: "github.com/x/unused"},
	)
	body := `
func f(ctx context.Context, r ndr.Reader) error {
	return rpc.Do(ctx, r)
}
`
	got, err := Assemble([]byte("// Code generated. DO NOT EDIT.\n\npackage p\n"), []byte(body), imps, true)
	if err != nil {
		t.Fatal(err)
	}
	want := `// Code generated. DO NOT EDIT.

package p

import (
	"context"

	"github.com/x/ndr"
	rpc "github.com/x/dcerpc"
)
`
	if !strings.HasPrefix(string(got), want[:strings.Index(want, "import")]) {
		t.Fatalf("header lost:\n%s", got)
	}
	for _, must := range []string{`"context"`, `"github.com/x/ndr"`, `rpc "github.com/x/dcerpc"`} {
		if !strings.Contains(string(got), must) {
			t.Errorf("missing import %s in:\n%s", must, got)
		}
	}
	for _, mustNot := range []string{`"fmt"`, `unused`} {
		if strings.Contains(string(got), mustNot) {
			t.Errorf("unused import %s present in:\n%s", mustNot, got)
		}
	}
	// the standard library comes first, separated from the rest.
	if strings.Index(string(got), `"context"`) > strings.Index(string(got), `"github.com/x/ndr"`) {
		t.Errorf("std imports must come first:\n%s", got)
	}
}

func TestLocalVariableIsNotAPackageQualifier(t *testing.T) {
	// "strings" is a parameter of type S here, not the package.
	imps := NewImports(Import{Path: "strings"})
	body := "type S struct{}\nfunc (S) Len() int { return 0 }\nfunc f(strings S) int { return strings.Len() }\n"
	got, err := Assemble([]byte("package p\n"), []byte(body), imps, false)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(got, []byte(`"strings"`)) {
		t.Errorf("a parameter named like a package must not import it:\n%s", got)
	}
}

func TestAssembleReportsSyntaxErrors(t *testing.T) {
	_, err := Assemble([]byte("package p\n"), []byte("func f( {\n"), NewImports(), false)
	if err == nil {
		t.Fatal("want an error for code that does not parse")
	}
}

func TestImportConflict(t *testing.T) {
	m := NewImports()
	if id := m.Add(Import{Path: "a/dtyp", Name: "dtyp"}); id != "dtyp" {
		t.Fatalf("got %s", id)
	}
	m.Add(Import{Path: "a/dtyp", Name: "dtyp"}) // the same import twice is fine.
	if len(m.Conflicts()) != 0 {
		t.Fatalf("unexpected conflicts: %v", m.Conflicts())
	}
	m.Add(Import{Path: "b/dtyp", Name: "dtyp"})
	if c := m.Conflicts(); len(c) != 1 || c[0].Kept.Path != "a/dtyp" || c[0].Dropped.Path != "b/dtyp" {
		t.Fatalf("conflicts: %v", c)
	}
}

func TestCond(t *testing.T) {
	got := write(func(w *Writer) {
		w.Cond(If("a", func() { w.Line("x()") }), ElseIf("b", func() { w.Line("y()") }), Else(func() { w.Line("z()") }))
	})
	want := "if a {\nx()\n} else if b {\ny()\n} else {\nz()\n}\n"
	if got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
}

func TestCondRejectsInvalidChains(t *testing.T) {
	for name, f := range map[string]func(){
		"else first":    func() { write(func(w *Writer) { w.Cond(Else(nil)) }) },
		"else not last": func() { write(func(w *Writer) { w.Cond(If("a", nil), Else(nil), ElseIf("b", nil)) }) },
		"if after if":   func() { write(func(w *Writer) { w.Cond(If("a", nil), If("b", nil)) }) },
		"elseif first":  func() { write(func(w *Writer) { w.Cond(ElseIf("a", nil)) }) },
	} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("%s: want a panic", name)
				}
			}()
			f()
		}()
	}
}

func TestFuncSignature(t *testing.T) {
	for _, tc := range []struct {
		f    Func
		want string
	}{
		{Func{Name: "F"}, "func F()"},
		{Func{Name: "F", Params: []Param{{"a", "int"}, {"b", "string"}}, Results: []Param{{"", "error"}}}, "func F(a int, b string) error"},
		{Func{Name: "F", Results: []Param{{"", "int"}, {"", "error"}}}, "func F() (int, error)"},
		{Func{Name: "F", Results: []Param{{"n", "int"}}}, "func F() (n int)"},
		{Func{Recv: &Param{"o", "*T"}, Name: "M", Params: []Param{{"ctx", "context.Context"}}}, "func (o *T) M(ctx context.Context)"},
		{Func{Params: []Param{{"", "int"}}, Results: []Param{{"", "bool"}}}, "func(int) bool"},
	} {
		if got := tc.f.Signature(); got != tc.want {
			t.Errorf("got %q, want %q", got, tc.want)
		}
	}
}

func TestDeclarations(t *testing.T) {
	got := write(func(w *Writer) {
		w.Struct("T", func() {
			w.Field("A", "int", `json:"a"`)
			w.Field("", "Embedded", "")
		})
		w.Const("X", "", "1")
		w.Var("v", "int", "")
		w.Func(Func{Name: "F"}, func() { w.Return(Call("g", "1", Quote(`a"b`)), "nil") })
	})
	for _, must := range []string{"type T struct {", "A int `json:\"a\"`", "Embedded\n", "const X = 1", "var v int", "func F() {", `return g(1, "a\"b"), nil`} {
		if !strings.Contains(got, must) {
			t.Errorf("missing %q in\n%s", must, got)
		}
	}
}
