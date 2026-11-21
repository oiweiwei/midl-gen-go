package ndr_test

import (
	"testing"

	"github.com/oiweiwei/midl-gen-go/desc"
	"github.com/oiweiwei/midl-gen-go/desc/ndr"
	"github.com/oiweiwei/midl-gen-go/midl"
)

func structOf(t *testing.T, decls, name string) *midl.Type {
	t.Helper()
	f, err := midl.NewSession(midl.WithoutBuiltins()).Parse(`[uuid(12345678-1234-1234-1234-123456789012), version(1.0)] interface t { ` + decls + ` }`)
	if err != nil {
		t.Fatal(err)
	}
	typ, _ := f.LookupType(name)
	if typ == nil {
		t.Fatalf("type %s not found", name)
	}
	return typ
}

func fieldCursor(t *testing.T, typ *midl.Type, name string) *desc.Cursor {
	t.Helper()
	for typ != nil && typ.Struct == nil {
		typ = typ.Elem
	}
	for _, f := range typ.Struct.Fields {
		if f.Name == name {
			return desc.Of(f)
		}
	}
	t.Fatalf("no field %s", name)
	return nil
}

// C706 14.2.2: a primitive of size n is aligned at a multiple of n; a structure
// is aligned like its most strictly aligned field; an enum is a 16 bit
// integer.
func TestAlignment(t *testing.T) {
	for _, tc := range []struct {
		name, decls string
		want        int
	}{
		{"char", `typedef struct S { char a; } S;`, 1},
		{"short", `typedef struct S { char a; short b; } S;`, 2},
		{"long", `typedef struct S { char a; long b; } S;`, 4},
		{"hyper", `typedef struct S { char a; hyper b; } S;`, 8},
		{"nested takes the largest", `typedef struct I { short a; hyper b; } I; typedef struct S { char c; I i; } S;`, 8},
		{"enum is 16 bit", `typedef enum { A, B } E; typedef struct S { E e; } S;`, 2},
		{"v1 enum is 32 bit", `typedef [v1_enum] enum { A, B } E; typedef struct S { E e; } S;`, 4},
		// a pointer is symbolic: 4 in NDR20, 8 in NDR64. The data it points to is not
		// part of the alignment, but is added for the opaque case (5 + 1).
		{"pointer", `typedef struct S { long *p; } S;`, ndr.PointerAlign + 1},
		{"pointer beats long", `typedef struct S { long a; long *p; } S;`, ndr.PointerAlign + 4},
	} {
		typ := structOf(t, tc.decls, "S")
		if got := ndr.Alignment(desc.OfType(typ)); got != tc.want {
			t.Errorf("%s: got %d, want %d", tc.name, got, tc.want)
		}
	}
}

func TestPlainAlignmentOfAPointerIsTheSymbol(t *testing.T) {
	typ := structOf(t, `typedef struct S { long *p; } S;`, "S")
	if got := ndr.PlainAlignment(desc.OfType(typ)); got != ndr.PointerAlign {
		t.Errorf("got %d, want %d", got, ndr.PointerAlign)
	}
}

func TestConformantAndVarying(t *testing.T) {
	typ := structOf(t, `typedef struct S {
		long n;
		long fixed[4];
		[size_is(n)] char conf[];
		[size_is(n), length_is(n)] char confvar[];
		[string] wchar_t str[16];
	} S;`, "S")

	for _, tc := range []struct {
		field               string
		conformant, varying bool
	}{
		{"fixed", false, false},
		{"conf", true, false},
		{"confvar", true, true},
		// a string is varying (it carries offset and count) but fixed in size.
		{"str", false, true},
	} {
		c := fieldCursor(t, typ, tc.field)
		if got := ndr.IsConformant(c); got != tc.conformant {
			t.Errorf("%s: conformant = %v, want %v", tc.field, got, tc.conformant)
		}
		if got := ndr.IsVarying(c); got != tc.varying {
			t.Errorf("%s: varying = %v, want %v", tc.field, got, tc.varying)
		}
	}
}

// A structure whose last member is conformant is conformant itself (C706 14.3.8).
func TestConformantStructure(t *testing.T) {
	typ := structOf(t, `typedef struct C { long n; [size_is(n)] char d[]; } C; typedef struct S { long x; C c; } S;`, "S")
	if !ndr.IsConformant(fieldCursor(t, typ, "c")) {
		t.Error("a struct ending with a conformant array must be conformant")
	}
}

// C706 14.3.12: the referents of embedded pointers are deferred.
func TestDeferrable(t *testing.T) {
	typ := structOf(t, `
		typedef struct P { long *p; } P;
		typedef struct V { long a; } V;
		typedef struct S { long a; V v; P p; long *q; long arr[4]; P parr[2]; } S;`, "S")
	for field, want := range map[string]bool{
		"a": false, "v": false, "p": true, "q": true, "arr": false, "parr": true,
	} {
		if got := ndr.IsDeferrable(fieldCursor(t, typ, field)); got != want {
			t.Errorf("%s: got %v, want %v", field, got, want)
		}
	}
}
