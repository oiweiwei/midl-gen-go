package midl

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func writeIDL(t *testing.T, field string) string {
	t.Helper()
	dir := t.TempDir()
	src := `[uuid(12345678-1234-1234-1234-123456789012), version(1.0)] interface shared {
	typedef struct S { long ` + field + `; } S;
}`
	if err := os.WriteFile(filepath.Join(dir, "shared.idl"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// Two sessions loading a same-named file from different include paths must
// each see their own definition.
func TestSessionsAreIsolated(t *testing.T) {
	a := NewSession(WithoutBuiltins(), WithInclude(writeIDL(t, "A")))
	b := NewSession(WithoutBuiltins(), WithInclude(writeIDL(t, "B")))

	for _, c := range []struct {
		s     *Session
		field string
	}{{a, "A"}, {b, "B"}} {
		if _, err := c.s.Load("shared.idl"); err != nil {
			t.Fatal(err)
		}
		typ := c.s.LookupType("S")
		for typ != nil && typ.Struct == nil {
			typ = typ.Elem
		}
		if typ == nil || typ.Struct.Fields[0].Name != c.field {
			t.Fatalf("session %q sees a foreign or missing type: %+v", c.field, typ)
		}
	}

	if len(a.Files()) != 1 || len(b.Files()) != 1 {
		t.Fatalf("unexpected file counts: %d, %d", len(a.Files()), len(b.Files()))
	}
}

func TestSessionsConcurrent(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		dir := writeIDL(t, "F")
		wg.Add(1)
		go func() {
			defer wg.Done()
			s := NewSession(WithoutBuiltins(), WithInclude(dir))
			if _, err := s.Load("shared.idl"); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
}

func TestFindFileDeterministicPrefix(t *testing.T) {
	dir := t.TempDir()
	for _, n := range []string{"ms-x.idl", "x.idl"} {
		if err := os.WriteFile(filepath.Join(dir, n), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	s := NewSession(WithoutBuiltins(), WithInclude(dir))
	for i := 0; i < 50; i++ {
		p, err := s.FindFile("ms-x.idl")
		if err != nil || filepath.Base(p.File) != "x.idl" {
			t.Fatalf("want x.idl deterministically, got %v %v", p, err)
		}
	}
}
