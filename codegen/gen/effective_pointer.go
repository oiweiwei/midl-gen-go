package gen

import (
	"context"

	"github.com/oiweiwei/midl-gen-go/midl"
)

// EffectivePointer resolves the kind of the pointer the scopes point to, see
// desc.Site.EffectivePointer. Every emitter that depends on the full ([ptr])
// semantics must use it.
func EffectivePointer(ctx context.Context, scopes *Scopes) midl.PointerType {
	return siteOf(ctx).EffectivePointer(scopes)
}

// IsFullPointer reports whether the pointer is a full ([ptr]) pointer: the only
// kind whose referent ids alias on the wire.
func IsFullPointer(ctx context.Context, scopes *Scopes) bool {
	return siteOf(ctx).IsFullPointer(scopes)
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
