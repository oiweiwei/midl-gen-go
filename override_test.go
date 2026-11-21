package main

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/oiweiwei/midl-gen-go/codegen/golang"
	"github.com/oiweiwei/midl-gen-go/midl"
)

// An override renames one member of one owner, and the new name must be used
// everywhere the member is referred to: in the declaration and in the size
// expression of another member (which names it by its IDL name).
func TestScopedOverride(t *testing.T) {
	dir, err := filepath.Abs("testdata/override")
	if err != nil {
		t.Fatal(err)
	}

	sess := midl.NewSession(midl.WithInclude(dir))
	out := t.TempDir()

	p, err := golang.New(golang.Options{Session: sess, ImportsPath: "x/", Format: true})
	if err != nil {
		t.Fatal(err)
	}
	files, err := p.Gen(context.Background(), filepath.Join(dir, "ov.idl"))
	if err != nil {
		t.Fatal(err)
	}
	if err := golang.WriteFiles(nil, out, files); err != nil {
		t.Fatal(err)
	}

	b, err := os.ReadFile(filepath.Join(out, "ov", "ov", "v1", "v1.go"))
	if err != nil {
		t.Fatal(err)
	}
	src := string(b)

	// the operation parameter: renamed to MaxCount, and no use of the old name remains in
	// the code of the operation (OTHER has a Count of its own, which stays).
	if !regexp.MustCompile(`(?m)^\s+MaxCount\s+int32`).MatchString(src) {
		t.Errorf("the overridden parameter must be MaxCount:\n%s", src)
	}
	opCode := 0
	for _, decl := range regexp.MustCompile(`\n(?:func|type) `).Split(src, -1) {
		header, _, _ := strings.Cut(decl, "\n")
		if !strings.Contains(header, "Stats") {
			continue
		}
		opCode++
		if regexp.MustCompile(`\bo\.Count\b|\bop\.Count\b`).MatchString(decl) {
			t.Errorf("the old name Count is still used in:\n%s", decl)
		}
	}
	if opCode == 0 || !strings.Contains(src, "o.MaxCount") {
		t.Errorf("the size expression must use the new name (checked %d declarations)", opCode)
	}

	// the struct field of BUF is renamed; the field of the same name in OTHER is not.
	structOf := func(name string) string {
		m := regexp.MustCompile(`(?s)type ` + name + ` struct \{(.*?)\n\}`).FindStringSubmatch(src)
		if m == nil {
			t.Fatalf("no struct %s in:\n%s", name, src)
		}
		return m[1]
	}
	if s := structOf("Buf"); !strings.Contains(s, "Length ") || strings.Contains(s, "Count ") {
		t.Errorf("Buf.count must be Length:%s", s)
	}
	if s := structOf("Other"); !strings.Contains(s, "Count ") || strings.Contains(s, "Length ") {
		t.Errorf("Other.count must stay Count:%s", s)
	}
}

// The symbols listing is the generator's own naming, so for every field it lists
// the Go name must be the one in the generated code.
func TestSymbolsAgreeWithGeneratedCode(t *testing.T) {
	dir, err := filepath.Abs("testdata/override")
	if err != nil {
		t.Fatal(err)
	}

	sess := midl.NewSession(midl.WithInclude(dir))
	f, err := sess.Load(filepath.Join(dir, "ov.idl"))
	if err != nil {
		t.Fatal(err)
	}

	p, err := golang.New(golang.Options{Session: sess})
	if err != nil {
		t.Fatal(err)
	}
	symbols := p.Symbols(context.Background(), f)

	type key struct{ kind, owner, idl string }
	got := map[key]golang.Symbol{}
	for _, s := range symbols {
		got[key{s.Kind, s.Owner, s.IDL}] = s
	}

	for k, want := range map[key]struct {
		goName string
		rules  []string
	}{
		{"param", "stats", "count"}: {"MaxCount", []string{"override"}},
		{"field", "BUF", "count"}:   {"Length", []string{"override"}},
		{"field", "OTHER", "count"}: {"Count", nil}, // the same IDL name of another owner is not touched.
		{"field", "OTHER", "other"}: {"Other", nil},
		{"type", "", "BUF"}:         {"Buf", nil},
	} {
		s, ok := got[k]
		if !ok {
			t.Errorf("no symbol %v in %v", k, symbols)
			continue
		}
		if s.Go != want.goName || strings.Join(s.Trace.Rules(), ",") != strings.Join(want.rules, ",") {
			t.Errorf("%v: got %s %v, want %s %v", k, s.Go, s.Trace.Rules(), want.goName, want.rules)
		}
	}

	// every field of every struct listed must be in the generated code under the name listed.
	g, err := golang.New(golang.Options{Session: sess, ImportsPath: "x/", Format: true})
	if err != nil {
		t.Fatal(err)
	}
	files, err := g.Gen(context.Background(), filepath.Join(dir, "ov.idl"))
	if err != nil {
		t.Fatal(err)
	}
	var src strings.Builder
	for _, file := range files {
		src.Write(file.Content)
	}
	for _, s := range symbols {
		if s.Kind != "field" {
			continue
		}
		if !regexp.MustCompile(`(?m)^\s+` + s.Go + `\s`).MatchString(src.String()) {
			t.Errorf("field %s.%s is listed as %s but the generated code has no such field", s.Owner, s.IDL, s.Go)
		}
	}
}
