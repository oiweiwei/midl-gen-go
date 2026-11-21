package ndr

import (
	"github.com/oiweiwei/midl-gen-go/codegen/golang/emit"
	"github.com/oiweiwei/midl-gen-go/desc"
)

// GenReadPointer returns the pointer read statement: the full pointers use ReadFullPointer,
// others ReadPointer; the hook (keyed literal) is used only when onNull is given.
func (p *TypeGenerator) GenReadPointer(ctx *emit.Env, scopes *desc.Cursor, ptr, setter, onNull, deferred string) string {
	fn := "w.ReadPointer"
	if emit.IsFullPointer(ctx, scopes) {
		fn = "w.ReadFullPointer"
	}
	if onNull == "" {
		return p.B(fn, ptr, setter, deferred)
	}
	return p.B(fn+"WithHook", ptr, "ndr.PointerHook{OnSet: "+setter+", OnNull: "+onNull+"}", deferred)
}
