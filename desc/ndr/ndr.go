// Package ndr holds the NDR layout rules over a resolved type chain (package
// desc): alignment, conformance, variance and deferral of pointers. They are
// properties of the wire format, not of the Go generator, so the same rules
// serve any code generator and the dynamic engine.
//
// Sources: DCE 1.1 RPC (C706) chapter 14, "Alignment of Primitive Types",
// "Alignment of Constructed Types", "Structures Containing Arrays", "Strings" and
// "Algorithm for Deferral of Referents"; MS-RPCE for NDR64. Samba's pidl
// (pidl/lib/Parse/Pidl/NDR.pm) implements the same rules independently and is
// the cross-check.
//
// Varying arrays. C706 includes the alignment of the size information (an
// unsigned long) in the alignment of a conformant or varying array. MS-RPCE
// (v20240916) has no statement that overrides this for NDR20, and for NDR64 it
// only widens the count words to 64 bits (2.2.5.3). So Alignment uses
// PointerAlign for them: that symbol is 4 in NDR20 and 8 in NDR64, which is the
// size of the count words in each syntax. Samba's pidl differs: it aligns a
// structure by its element types only and aligns the count words when it writes
// them, which gives other bytes only for a structure whose varying array follows
// a smaller field and which starts unaligned. For conformant arrays the
// question does not arise: their size information is moved in front of the
// structure and aligned on its own.
//
// Known gap: enums. MS-RPCE 2.2.5.2 makes enums 4 octets in NDR64 (2 in
// NDR20, 4 for [v1_enum] in both), so the alignment of a plain enum depends on
// the transfer syntax like the alignment of a pointer. Alignment returns a
// fixed 2, so a structure whose largest member is such an enum is aligned to 2
// under NDR64 instead of 4. Fixing it needs a symbolic enum alignment and
// support in the runtime.
//
// ms_union (MS-RPCE 2.2.4.5) changes the alignment of the arms of a union in
// NDR20; it is applied by the generator, not here.
package ndr

import (
	"github.com/oiweiwei/midl-gen-go/desc"
	"github.com/oiweiwei/midl-gen-go/midl"
)

// PointerAlign is the symbolic alignment of a pointer. It is not a size: the
// size of a pointer depends on the transfer syntax (4 bytes in NDR20, 8 in
// NDR64), so the generated code passes the symbol and the runtime resolves it.
// pidl uses the same marker (5, "translated into an alignment of 4 for NDR and
// 8 for NDR64").
const PointerAlign = 5

// Alignment returns the alignment the generated code passes to the runtime's
// Align calls for the type at the cursor.
//
// It is the plain alignment (see PlainAlignment) unless that is PointerAlign;
// then the alignment the data would have if its pointers were opaque is added
// (so 6, 7, 9, 13), which lets the runtime recover it when it handles the value
// as opaque data and not as an NDR pointer.
func Alignment(c *desc.Cursor) int {
	a := alignment(c, false)
	if a != PointerAlign {
		return a
	}
	return a + alignment(c, true)
}

// PlainAlignment returns the alignment of the type at the cursor with a
// pointer counting as PointerAlign. Two types need the same padding if their
// plain alignments are equal, which is how a trailing gap is detected.
func PlainAlignment(c *desc.Cursor) int { return alignment(c, false) }

func alignment(c *desc.Cursor, opaque bool) int {

	switch t := c.ScopedType(); {
	case t.Is(midl.TypeEnum):
		if t.Attrs.V1Enum {
			return 4
		}
		return 2
	case t.IsPrimitiveType():
		return midl.PrimitiveTypeSize(t.Kind)
	case t.Is(midl.TypePointer):
		if opaque {
			return 1
		}
		return PointerAlign
	case t.Is(midl.TypeArray):
		a := alignment(c.Next(), opaque)
		va := PointerAlign
		if opaque {
			va = 1
		}
		if IsVarying(c) {
			if a < va {
				a = va
			}
		}
		return a
	case t.Is(midl.TypeStruct):
		a := 0
		for _, f := range t.Struct.Fields {
			if f.Attrs.Ignore {
				continue
			}
			if fa := alignment(desc.Of(f), opaque); fa > a {
				a = fa
			}
		}
		return a
	case t.Is(midl.TypeUnion) || t.Is(midl.TypeCUnion):
		a := 0
		if c.Layer().SwitchType != nil {
			a = alignment(desc.OfType(c.Layer().SwitchType), opaque)
		}
		for _, cs := range t.Union.Body {
			for _, f := range cs.Arms {
				if cfa := alignment(desc.Of(f), opaque); cfa > a {
					a = cfa
				}
			}
			if t.Is(midl.TypeCUnion) {
				break
			}
		}
		return a
	}
	return 0
}

// IsConformant reports whether the type is conformant: it is (or ends with) an
// array whose size is only known at run time, so the size is transmitted in
// front of the data.
func IsConformant(c *desc.Cursor) bool {

	for ; c != nil; c = c.Next() {
		if c.Is(midl.TypeArray) {
			if !c.Type().Array.IsFixed() && !c.Dim().NoSizeLimit {
				return true
			}
		} else if c.Is(midl.TypeStruct) {
			last := c.StructLastField()
			return last != nil && IsConformant(desc.Of(last))
		} else {
			break
		}
	}
	return false
}

// IsVarying reports whether the array at the cursor is varying: it has
// length_is, first_is or last_is, or it is a null terminated string.
func IsVarying(c *desc.Cursor) bool {
	for ; c != nil; c = c.Next() {
		if c.Is(midl.TypeArray) {
			if dim := c.Dim(); (dim.IsString && dim.IsNullTerminated) || !dim.LengthIs.IsEmpty() || !dim.FirstIs.IsEmpty() || !dim.LastIs.IsEmpty() {
				return true
			}
		} else {
			break
		}
	}
	return false
}

// IsTopLevelArray reports whether the array at the cursor has its conformance
// data written with it: it is the top-level array if the data type is a
// pointer, or a fixed array. The top-level array may contain varying
// information which needs to be written.
func IsTopLevelArray(c *desc.Cursor) bool {
	prev := c.Prev()
	return prev != nil && (prev.Is(midl.TypePointer) || prev.Is(midl.TypeArray) && prev.Array.IsFixed())
}

// IsDeferrable reports whether the type has pointers whose referents are
// transmitted after the enclosing structure.
func IsDeferrable(c *desc.Cursor) bool {
	switch {
	case c.Is(midl.TypePointer):
		return true
	case c.Is(midl.TypeArray):
		return IsDeferrable(c.Next())
	case c.Is(midl.TypeStruct):
		for _, field := range c.Struct().Fields {
			if field.Attrs.Ignore {
				continue
			}
			if IsDeferrable(desc.Of(field)) {
				return true
			}
		}
	case c.Is(midl.TypeUnion) || c.Is(midl.TypeCUnion):
		for _, cs := range c.Union().Body {
			for _, arm := range cs.Arms {
				if IsDeferrable(desc.Of(arm)) {
					return true
				}
			}
		}
	}
	return false
}
