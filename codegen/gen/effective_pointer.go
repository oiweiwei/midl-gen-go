package gen

import (
	"context"

	"github.com/oiweiwei/midl-gen-go/midl"
)

// IsFirstType reports whether the scopes point to the very first type of the field.
func (i *Scopes) IsFirstType() bool {
	return i.scopePos == 0 && i.typePos == 0
}

// EffectivePointer resolves the kind of the pointer the scopes point to, using
// the MIDL precedence: the attribute of the use site or of the typedef (the
// use site wins, the outermost pointer of the scope only), the [ref] default of
// an operation parameter, the pointer_default of the interface, [unique].
// Every emitter that depends on the full ([ptr]) semantics must use it.
func EffectivePointer(ctx context.Context, scopes *Scopes) midl.PointerType {

	if scopes.IsTopLevelPointer() {
		if ptr := scopes.Scope().Pointer; ptr != midl.PointerTypeNone {
			if ptr == midl.PointerTypeRefWeak {
				return midl.PointerTypeRef
			}
			return ptr
		}
	}

	if IsOp(ctx) && scopes.IsFirstType() {
		return midl.PointerTypeRef
	}

	if iff := Interface(ctx); iff != nil && iff.Attrs != nil {
		if ptr := iff.Attrs.PointerDefault; ptr != midl.PointerTypeNone {
			return ptr
		}
	}

	return midl.PointerTypeUnique
}

// IsFullPointer reports whether the pointer is a full ([ptr]) pointer: the only
// kind whose referent ids alias on the wire.
func IsFullPointer(ctx context.Context, scopes *Scopes) bool {
	return EffectivePointer(ctx, scopes) == midl.PointerTypePtr
}

// GenReadPointer returns the pointer read statement: the full pointers use ReadFullPointer,
// others ReadPointer; the hook (keyed literal) is used only when onNull is given.
func (p *TypeGenerator) GenReadPointer(ctx context.Context, scopes *Scopes, ptr, setter, onNull, deferred string) string {
	fn := "w.ReadPointer"
	if IsFullPointer(ctx, scopes) {
		fn = "w.ReadFullPointer"
	}
	if onNull == "" {
		return p.B(fn, ptr, setter, deferred)
	}
	return p.B(fn+"WithHook", ptr, "ndr.PointerHook{OnSet: "+setter+", OnNull: "+onNull+"}", deferred)
}
