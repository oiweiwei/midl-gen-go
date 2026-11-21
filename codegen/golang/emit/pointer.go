package emit

import (
	"github.com/oiweiwei/midl-gen-go/desc"
	"github.com/oiweiwei/midl-gen-go/midl"
)

// siteOf returns the use site the generator is emitting code for.
func siteOf(ctx *Env) desc.Site {
	site := desc.Site{Operation: ctx.Operation}
	if iff := ctx.Interface; iff != nil && iff.Attrs != nil {
		site.PointerDefault = iff.Attrs.PointerDefault
	}
	return site
}

// IsIgnoredPointer reports whether the field starts with a pointer that does
// not appear on the wire, see desc.Site.IsIgnoredPointer.
func IsIgnoredPointer(ctx *Env, field *midl.Field) bool {
	return siteOf(ctx).IsIgnoredPointer(desc.Of(field))
}

// EffectivePointer resolves the kind of the pointer the scopes point to, see
// desc.Site.EffectivePointer. Every emitter that depends on the full ([ptr])
// semantics must use it.
func EffectivePointer(ctx *Env, scopes *desc.Cursor) midl.PointerType {
	return siteOf(ctx).EffectivePointer(scopes)
}

// IsFullPointer reports whether the pointer is a full ([ptr]) pointer: the only
// kind whose referent ids alias on the wire.
func IsFullPointer(ctx *Env, scopes *desc.Cursor) bool {
	return siteOf(ctx).IsFullPointer(scopes)
}
