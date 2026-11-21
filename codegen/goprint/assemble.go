package goprint

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
)

// Assemble joins the header (the file comment and the package clause), the
// import declaration and the body into a file. The import declaration lists the
// candidates of imps that the code refers to as "pkg.Name" and no others.
//
// It fails if header and body do not parse as a Go file, which includes
// the case of a body that refers to a package it did not register. When
// formatted is true the result is gofmt'ed.
func Assemble(header, body []byte, imps *Imports, formatted bool) ([]byte, error) {

	src := append(append([]byte{}, header...), body...)

	file, err := parser.ParseFile(token.NewFileSet(), "", src, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("goprint: generated code does not parse: %w", err)
	}

	used := usedImports(file, imps)

	out := append(append(append([]byte{}, header...), declaration(used)...), body...)

	if formatted {
		if out, err = format.Source(out); err != nil {
			return nil, fmt.Errorf("goprint: format: %w", err)
		}
	}

	return out, nil
}

// usedImports returns the candidates whose identifier is the unresolved
// qualifier of a selector expression in the file.
func usedImports(file *ast.File, imps *Imports) []Import {

	var used []Import
	seen := map[string]bool{}

	ast.Inspect(file, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		id, ok := sel.X.(*ast.Ident)
		// an identifier that resolves to a declaration of the file (a variable,
		// a parameter) is not a package qualifier.
		if !ok || id.Obj != nil || seen[id.Name] {
			return true
		}
		if imp, ok := imps.Lookup(id.Name); ok {
			seen[id.Name] = true
			used = append(used, imp)
		}
		return true
	})

	return used
}
