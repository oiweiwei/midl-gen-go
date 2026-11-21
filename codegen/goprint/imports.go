// Package goprint builds Go source text: a typed Writer for statements and
// declarations, and an import manager that emits exactly the imports a file
// uses.
//
// Generated code is written as text, so the generator cannot always tell while
// it emits a line whether a package ends up referenced (a name computed for a
// comparison, a branch that writes nothing). Instead of importing eagerly and
// silencing the compiler with "var _ = pkg.Symbol" guards, the generator
// registers candidate imports and Assemble keeps those whose qualifier appears
// in the code, like goimports does.
package goprint

import (
	"fmt"
	"path"
	"sort"
	"strings"
)

// Import is a Go import.
type Import struct {
	// Path is the import path.
	Path string
	// Name is the identifier the package is referred to by when it differs
	// from the last element of the path; it is the alias of the import.
	Name string
}

// Ident returns the identifier the code uses to refer to the package.
func (i Import) Ident() string {
	if i.Name != "" {
		return i.Name
	}
	return path.Base(i.Path)
}

// IsStd reports whether the import is from the standard library: the first
// element of its path has no dot.
func (i Import) IsStd() bool {
	first, _, _ := strings.Cut(i.Path, "/")
	return !strings.Contains(first, ".")
}

// Conflict is a pair of different packages registered under one identifier.
type Conflict struct {
	Ident string
	// Kept is the import that was registered first and stays in effect;
	// Dropped is the one that was refused.
	Kept, Dropped Import
}

func (c Conflict) String() string {
	return fmt.Sprintf("identifier %q is both %q and %q; %q is used", c.Ident, c.Kept.Path, c.Dropped.Path, c.Kept.Path)
}

// Imports is the set of candidate imports of one file. The zero value is
// ready to use.
type Imports struct {
	byIdent   map[string]Import
	conflicts []Conflict
}

// NewImports returns a set holding the candidates.
func NewImports(candidates ...Import) *Imports {
	m := &Imports{}
	for _, c := range candidates {
		m.Add(c)
	}
	return m
}

// Add registers a candidate import and returns the identifier the code must use
// to refer to the package. If the identifier is already taken by another package
// the first one is kept and the conflict is recorded (see Conflicts).
func (m *Imports) Add(imp Import) string {
	if m.byIdent == nil {
		m.byIdent = make(map[string]Import)
	}
	id := imp.Ident()
	if prev, ok := m.byIdent[id]; ok {
		if prev.Path != imp.Path {
			m.conflicts = append(m.conflicts, Conflict{Ident: id, Kept: prev, Dropped: imp})
		}
		return id
	}
	m.byIdent[id] = imp
	return id
}

// Conflicts returns the identifier conflicts seen so far.
func (m *Imports) Conflicts() []Conflict { return m.conflicts }

// Lookup returns the candidate registered under the identifier.
func (m *Imports) Lookup(ident string) (Import, bool) {
	imp, ok := m.byIdent[ident]
	return imp, ok
}

// Clone returns an independent copy of the set.
func (m *Imports) Clone() *Imports {
	c := &Imports{byIdent: make(map[string]Import, len(m.byIdent))}
	for k, v := range m.byIdent {
		c.byIdent[k] = v
	}
	return c
}

// declaration renders the import declaration for the used imports: the
// standard library first, then the others, each group sorted by path.
func declaration(used []Import) string {
	if len(used) == 0 {
		return ""
	}
	sort.Slice(used, func(i, j int) bool { return used[i].Path < used[j].Path })

	var std, other []Import
	for _, imp := range used {
		if imp.IsStd() {
			std = append(std, imp)
		} else {
			other = append(other, imp)
		}
	}

	var b strings.Builder
	b.WriteString("\nimport (\n")
	line := func(imp Import) {
		if imp.Name != "" && imp.Name != path.Base(imp.Path) {
			fmt.Fprintf(&b, "\t%s %q\n", imp.Name, imp.Path)
		} else {
			fmt.Fprintf(&b, "\t%q\n", imp.Path)
		}
	}
	for _, imp := range std {
		line(imp)
	}
	if len(std) > 0 && len(other) > 0 {
		b.WriteString("\n")
	}
	for _, imp := range other {
		line(imp)
	}
	b.WriteString(")\n")
	return b.String()
}
