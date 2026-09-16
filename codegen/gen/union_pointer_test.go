package gen

import (
	"bytes"
	"context"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"testing"
)

func TestPrimitiveUnionPointerClearsReusedValue(t *testing.T) {
	const idl = `typedef [switch_type(unsigned long)] union {
    [case(1)] long* number;
    [case(2)] long scalar;
    [default] small* fallback;
} Optional;`
	dir := t.TempDir()
	t.Setenv("MSIDLPATH", dir)
	source := filepath.Join(dir, "optional.idl")
	if err := os.WriteFile(source, []byte(idl), 0600); err != nil {
		t.Fatal(err)
	}
	p := &Generator{Dir: dir, Format: true}
	if err := p.Gen(context.Background(), source); err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filepath.Join(dir, "optional", "optional.go"), nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	checked := 0
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "UnmarshalUnionNDR" {
			continue
		}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			arm, ok := n.(*ast.CaseClause)
			if !ok {
				return true
			}
			var first bytes.Buffer
			if err := format.Node(&first, fset, arm.Body[0]); err != nil {
				t.Fatal(err)
			}
			var label bytes.Buffer
			if len(arm.List) > 0 {
				if err := format.Node(&label, fset, arm.List[0]); err != nil {
					t.Fatal(err)
				}
			}
			// A null pointer skips its deferred callback. Both the numbered
			// and default pointer arms must clear the old union first.
			if label.String() == "uint32(1)" || len(arm.List) == 0 {
				if first.String() != "o.Value = nil" {
					t.Errorf("pointer arm %q starts with %q, want o.Value = nil", label.String(), first.String())
				}
				checked++
			}
			return true
		})
	}
	if checked != 2 {
		t.Fatalf("checked %d pointer arms, want 2", checked)
	}
}
