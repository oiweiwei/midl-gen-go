package desc_test

import (
	"testing"

	"github.com/oiweiwei/midl-gen-go/desc"
	"github.com/oiweiwei/midl-gen-go/midl"
)

// parseStruct parses the IDL declarations and returns the struct behind the
// typedef of the given name.
func parseStruct(t *testing.T, decls, name string) *midl.Struct {
	t.Helper()
	f, err := midl.NewSession(midl.WithoutBuiltins()).Parse(`[uuid(12345678-1234-1234-1234-123456789012), version(1.0)] interface t { ` + decls + ` }`)
	if err != nil {
		t.Fatal(err)
	}
	typ, _ := f.LookupType(name)
	for typ != nil && typ.Struct == nil {
		typ = typ.Elem
	}
	if typ == nil {
		t.Fatalf("struct %s not found", name)
	}
	return typ.Struct
}

func field(t *testing.T, s *midl.Struct, name string) *midl.Field {
	t.Helper()
	for _, f := range s.Fields {
		if f.Name == name {
			return f
		}
	}
	t.Fatalf("no field %s", name)
	return nil
}

// kinds flattens the chain to the kinds of its nodes.
func kinds(c *desc.Chain) []midl.Kind {
	var ks []midl.Kind
	for _, l := range c.Layers {
		for _, n := range l.Nodes {
			ks = append(ks, n.Kind)
		}
	}
	return ks
}

func assertKinds(t *testing.T, c *desc.Chain, want ...midl.Kind) {
	t.Helper()
	got := kinds(c)
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestPointerDepthIsNumberedFromTheInside(t *testing.T) {
	s := parseStruct(t, `typedef struct S { long **pp; } S;`, "S")
	c := desc.Resolve(field(t, s, "pp"))
	assertKinds(t, c, midl.TypePointer, midl.TypePointer, midl.TypeInt32)
	n := c.Layers[0].Nodes
	if n[0].Pointer != 2 || n[1].Pointer != 1 {
		t.Errorf("pointer depth: got %d, %d; want 2, 1", n[0].Pointer, n[1].Pointer)
	}
}

func TestSizeIsInsertsAConformantArrayAfterThePointer(t *testing.T) {
	// https://learn.microsoft.com/en-us/windows/win32/Midl/size-is: a pointer
	// with size_is is an array of the pointed-to type.
	s := parseStruct(t, `typedef struct S { long n; [size_is(n)] long *p; } S;`, "S")
	c := desc.Resolve(field(t, s, "p"))
	assertKinds(t, c, midl.TypePointer, midl.TypeArray, midl.TypeInt32)
	arr := c.Layers[0].Nodes[1]
	if arr.Array.IsFixed() {
		t.Error("the inserted array must be conformant")
	}
	if names := arr.Dim.SizeIs.Idents(); len(names) != 1 || names[0] != "n" {
		t.Errorf("size_is must be attached to the array, got %v", arr.Dim.SizeIs)
	}
}

func TestSizeIsOnAnExistingArrayIsNotDuplicated(t *testing.T) {
	s := parseStruct(t, `typedef struct S { long n; [size_is(n)] long a[]; } S;`, "S")
	assertKinds(t, desc.Resolve(field(t, s, "a")), midl.TypeArray, midl.TypeInt32)
}

func TestStringPointerBecomesANullTerminatedArray(t *testing.T) {
	s := parseStruct(t, `typedef struct S { [string] wchar_t *p; } S;`, "S")
	c := desc.Resolve(field(t, s, "p"))
	assertKinds(t, c, midl.TypePointer, midl.TypeArray, midl.TypeWChar)
	if d := c.Layers[0].Nodes[1].Dim; !d.IsString || !d.IsNullTerminated {
		t.Errorf("want a null terminated string, got %+v", d)
	}
}

func TestUniqueAttributeOnANonPointerAddsAPointer(t *testing.T) {
	s := parseStruct(t, `typedef struct S { [unique] long x; } S;`, "S")
	assertKinds(t, desc.Resolve(field(t, s, "x")), midl.TypePointer, midl.TypeInt32)
}

func TestSite(t *testing.T) {
	s := parseStruct(t, `typedef struct S { long *p; [ref] long *r; [ptr] long *f; } S;`, "S")
	at := func(name string) *desc.Cursor { return desc.Of(field(t, s, name)) }

	for _, tc := range []struct {
		name string
		site desc.Site
		want midl.PointerType
	}{
		// the use site attribute wins over everything.
		{"r", desc.Site{PointerDefault: midl.PointerTypePtr}, midl.PointerTypeRef},
		{"f", desc.Site{}, midl.PointerTypePtr},
		// unannotated: the interface default, then [unique].
		{"p", desc.Site{}, midl.PointerTypeUnique},
		{"p", desc.Site{PointerDefault: midl.PointerTypePtr}, midl.PointerTypePtr},
		// the outermost pointer of an operation parameter is [ref] unless annotated.
		{"p", desc.Site{Operation: true, PointerDefault: midl.PointerTypePtr}, midl.PointerTypeRef},
		{"f", desc.Site{Operation: true}, midl.PointerTypePtr},
	} {
		if got := tc.site.EffectivePointer(at(tc.name)); got != tc.want {
			t.Errorf("%s %+v: got %v, want %v", tc.name, tc.site, got, tc.want)
		}
	}
}
