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

func TestStringUnmarshalTermination(t *testing.T) {
	// Exercise the IDL parser and generator together. Counted wchar_t arrays
	// are strings in Go, but NUL code units belong to their contents.
	const idl = `
typedef struct { unsigned long count; [size_is(count)] wchar_t data[]; } Counted;
typedef struct { unsigned long count; [string, size_is(count)] wchar_t data[]; } Terminated;
typedef struct { [size_is(*)] wchar_t data[]; } Unbounded;
typedef struct { [string, size_is(*)] wchar_t data[]; } UnboundedTerminated;
typedef struct { [string, size_is(*)] char data[]; } UnboundedChar;
typedef struct { unsigned long count; [format(null_terminated), size_is(count)] wchar_t data[]; } FormatTerminated;
typedef struct { [format(null_terminated), size_is(*)] wchar_t data[]; } UnboundedFormatTerminated;
`
	dir := t.TempDir()
	t.Setenv("MSIDLPATH", dir)
	source := filepath.Join(dir, "strings.idl")
	if err := os.WriteFile(source, []byte(idl), 0600); err != nil {
		t.Fatal(err)
	}
	p := &Generator{Dir: dir, Format: true}
	if err := p.Gen(context.Background(), source); err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"Counted":                   "string(utf16.Decode(_Data_buf))",
		"Terminated":                "strings.TrimRight(string(utf16.Decode(_Data_buf)), ndr.ZeroString)",
		"Unbounded":                 "string(utf16.Decode(_Data_buf))",
		"UnboundedTerminated":       "strings.TrimRight(string(utf16.Decode(_Data_buf)), ndr.ZeroString)",
		"UnboundedChar":             "strings.TrimRight(string(_Data_buf), ndr.ZeroString)",
		"FormatTerminated":          "strings.TrimRight(string(utf16.Decode(_Data_buf)), ndr.ZeroString)",
		"UnboundedFormatTerminated": "strings.TrimRight(string(utf16.Decode(_Data_buf)), ndr.ZeroString)",
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filepath.Join(dir, "strings", "strings.go"), nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "UnmarshalNDR" {
			continue
		}
		name := fn.Recv.List[0].Type.(*ast.StarExpr).X.(*ast.Ident).Name
		expected, ok := want[name]
		if !ok {
			continue
		}
		found := false
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			assignment, ok := n.(*ast.AssignStmt)
			if !ok || len(assignment.Lhs) != 1 {
				return true
			}
			field, ok := assignment.Lhs[0].(*ast.SelectorExpr)
			if !ok || field.Sel.Name != "Data" {
				return true
			}
			var expr bytes.Buffer
			if err := format.Node(&expr, fset, assignment.Rhs[0]); err != nil {
				t.Fatal(err)
			}
			if expr.String() != expected {
				t.Errorf("%s.Data = %s, want %s", name, expr.String(), expected)
			}
			found = true
			return true
		})
		if !found {
			t.Errorf("%s: missing Data assignment", name)
		}
		delete(want, name)
	}
	for name := range want {
		t.Errorf("missing decoder for %s", name)
	}
}
