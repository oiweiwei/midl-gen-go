package desc

import (
	"strings"

	"github.com/oiweiwei/midl-gen-go/midl"
)

// Cursor is a position in a Chain. It is immutable: Next returns a new cursor,
// and nil past the last node. Within one layer the attributes of the layer
// apply to all its nodes.
type Cursor struct {
	layerPos, nodePos int
	chain             *Chain
	prev              *midl.Type
	alias             string
}

// NewCursor returns the cursor at the first node of the chain, or nil if the
// chain is empty.
func NewCursor(c *Chain) *Cursor {
	i := &Cursor{chain: c, nodePos: -1}
	return i.Next()
}

// Of resolves the field and returns the cursor at the start of its chain.
func Of(f *midl.Field) *Cursor { return NewCursor(Resolve(f)) }

// OfType resolves the type (without any use site attributes) and returns the
// cursor at the start of its chain.
func OfType(t *midl.Type) *Cursor { return NewCursor(ResolveType(t)) }

func (i *Cursor) clone() *Cursor {
	c := *i
	return &c
}

// Next returns the cursor at the following node, or nil if there is none.
func (i *Cursor) Next() *Cursor {

	if i == nil || len(i.chain.Layers) == 0 {
		return nil
	}

	i = i.clone()

	if i.nodePos >= 0 {
		i.prev = i.Type()
	}

	if i.nodePos++; i.nodePos >= len(i.chain.Layers[i.layerPos].Nodes) {
		i.nodePos = 0
		for {
			if i.layerPos++; i.layerPos >= len(i.chain.Layers) {
				return nil
			}
			if len(i.chain.Layers[i.layerPos].Nodes) != 0 {
				break
			}
		}
	}
	return i
}

// Prev returns the type the cursor came from, or nil at the start.
func (i *Cursor) Prev() *midl.Type { return i.prev }

func (i *Cursor) node() *Node { return i.chain.Layers[i.layerPos].Nodes[i.nodePos] }

// IsFirstType reports whether the cursor is at the very first type of the chain.
func (i *Cursor) IsFirstType() bool { return i.layerPos == 0 && i.nodePos == 0 }

// Base returns the cursor at the innermost type that has no element type, i.e.
// the base type behind all pointers and arrays.
func (i *Cursor) Base() *Cursor {

	i = i.clone()

	for i != nil && i.Type().Elem != nil {
		i = i.Next()
	}

	return i
}

// Type returns the type at the cursor.
func (i *Cursor) Type() *midl.Type { return i.node().Type }

// Union returns the union at the cursor, if it is one.
func (i *Cursor) Union() *midl.Union { return i.Type().Union }

// Struct returns the struct at the cursor, if it is one.
func (i *Cursor) Struct() *midl.Struct { return i.Type().Struct }

// Array returns the array at the cursor, if it is one.
func (i *Cursor) Array() *midl.Array { return i.Type().Array }

// Enum returns the enum at the cursor, if it is one.
func (i *Cursor) Enum() *midl.Enum { return i.Type().Enum }

// ScopedType returns a copy of the type at the cursor whose attributes are
// merged with the attributes of its layer.
func (i *Cursor) ScopedType() *midl.Type {
	typ := *i.Type()
	typ.Attrs = (&midl.TypeAttr{}).Merge(typ.Attrs).Merge(i.Layer())
	return &typ
}

// Is reports whether the type at the cursor is of the kind; it is false for a
// nil cursor.
func (i *Cursor) Is(kind midl.Kind) bool {
	if i != nil {
		return i.Type().Is(kind)
	}
	return false
}

// Kind returns the kind of the type at the cursor.
func (i *Cursor) Kind() midl.Kind { return i.Type().Kind }

// Layer returns the attributes that apply at the cursor.
func (i *Cursor) Layer() *midl.TypeAttr { return i.chain.Layers[i.layerPos].Attr }

// WithAlias returns the cursor reporting the alias instead of the one of its layer.
func (i *Cursor) WithAlias(a string) *Cursor {
	i = i.clone()
	i.alias = a
	return i
}

// Alias returns the typedef name the cursor stands for, if any.
func (i *Cursor) Alias() string {
	if i.alias != "" {
		return i.alias
	}
	return i.Layer().Alias
}

// IsBool reports whether the type is a primitive named BOOL*, which is
// rendered as a boolean.
func (i *Cursor) IsBool() bool {
	return i.Type().IsPrimitiveType() && strings.HasPrefix(strings.ToUpper(i.Alias()), "BOOL")
}

// IsNotFixedArray reports whether the cursor is at an array whose length is
// only known at run time (conformant, or a string).
func (i *Cursor) IsNotFixedArray() bool {
	return i != nil && i.Array() != nil && (!i.Array().IsFixed() || i.Dim().IsString)
}

// Dim returns the dimension of the array at the cursor.
func (i *Cursor) Dim() midl.Dim { return i.node().Dim }

// PointerCount returns the number of pointers following the current pointer
// plus the current pointer itself.
func (i *Cursor) PointerCount() int { return i.node().Pointer }

// IsTopLevelPointer reports whether the cursor is at the first pointer of its
// layer; only that pointer is subject to the pointer attribute of the layer.
func (i *Cursor) IsTopLevelPointer() bool {
	nodes := i.chain.Layers[i.layerPos].Nodes
	for pos := i.nodePos - 1; pos >= 0; pos-- {
		if nodes[pos].Is(midl.TypePointer) {
			return false
		}
	}
	return true
}

// IsV1Enum reports whether any layer marks the enum as a v1 (32-bit) enum.
func (i *Cursor) IsV1Enum() bool {
	for _, layer := range i.chain.Layers {
		if layer.Attr.V1Enum {
			return true
		}
	}
	return false
}

// EnumType returns the integer type the enum is transmitted as.
func (i *Cursor) EnumType() midl.Kind {
	for _, layer := range i.chain.Layers {
		if layer.Attr.V1Enum {
			return layer.Attr.EnumType()
		}
	}
	return i.chain.Layers[0].Attr.EnumType()
}

// StructLastField returns the last field of the struct at the cursor, or nil
// if the cursor is not at a struct.
func (i *Cursor) StructLastField() *midl.Field {
	if !i.Is(midl.TypeStruct) {
		return nil
	}
	return i.Struct().LastField()
}
