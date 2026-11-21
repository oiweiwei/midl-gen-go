package gen

import (
	"context"

	"github.com/oiweiwei/midl-gen-go/desc"
	"github.com/oiweiwei/midl-gen-go/midl"
)

// Scopes is the cursor over the resolved type chain of a field. The chain and
// the cursor live in package desc; the name is kept for the generator code.
type Scopes = desc.Cursor

// siteOf returns the use site the generator is emitting code for.
func siteOf(ctx context.Context) desc.Site {
	site := desc.Site{Operation: IsOp(ctx)}
	if iff := Interface(ctx); iff != nil && iff.Attrs != nil {
		site.PointerDefault = iff.Attrs.PointerDefault
	}
	return site
}

// IsIgnoredPointer reports whether the field starts with a pointer that does
// not appear on the wire, see desc.Site.IsIgnoredPointer.
func IsIgnoredPointer(ctx context.Context, field *midl.Field) bool {
	return siteOf(ctx).IsIgnoredPointer(desc.Of(field))
}
