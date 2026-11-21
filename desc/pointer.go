package desc

import "github.com/oiweiwei/midl-gen-go/midl"

// Site describes where a type is used; it decides what an unannotated pointer
// is.
type Site struct {
	// Operation is true for the parameters of an operation, where the
	// outermost pointer defaults to [ref].
	Operation bool
	// PointerDefault is the pointer_default attribute of the interface; None if
	// there is none.
	PointerDefault midl.PointerType
}

// EffectivePointer resolves the kind of the pointer at the cursor with the MIDL
// precedence: the attribute of the use site or of the typedef (the use site
// wins, the outermost pointer of the layer only), the [ref] default of an
// operation parameter, the pointer_default of the interface, [unique].
func (s Site) EffectivePointer(c *Cursor) midl.PointerType {

	if c.IsTopLevelPointer() {
		if ptr := c.Layer().Pointer; ptr != midl.PointerTypeNone {
			if ptr == midl.PointerTypeRefWeak {
				return midl.PointerTypeRef
			}
			return ptr
		}
	}

	if s.Operation && c.IsFirstType() {
		return midl.PointerTypeRef
	}

	if s.PointerDefault != midl.PointerTypeNone {
		return s.PointerDefault
	}

	return midl.PointerTypeUnique
}

// IsFullPointer reports whether the pointer at the cursor is a full ([ptr])
// pointer: the only kind whose referent ids alias on the wire.
func (s Site) IsFullPointer(c *Cursor) bool {
	return s.EffectivePointer(c) == midl.PointerTypePtr
}

// IsIgnoredPointer reports whether the chain starts with a pointer that does
// not appear on the wire: a [ref] (or operation default) pointer to a
// primitive type, which is transmitted as the primitive itself.
func (s Site) IsIgnoredPointer(c *Cursor) bool {
	for ; c != nil; c = c.Next() {
		if c.Is(midl.TypePointer) {
			if c.Layer().Pointer.IsWeak(s.Operation) {
				continue
			}
			if next := c.Next(); next.Is(midl.TypePointer) {
				continue
			} else if next != nil && next.Type().IsPrimitiveType() {
				return true
			}
			break
		}
	}
	return false
}
