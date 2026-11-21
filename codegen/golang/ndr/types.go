package ndr

import (
	"github.com/oiweiwei/midl-gen-go/codegen/golang/emit"
	"github.com/oiweiwei/midl-gen-go/midl"
)

// GenType generates the Go type of the IDL type and its NDR marshalling.
func (p *Emitter) GenType(ctx *emit.Env, t *midl.Type) {

	pp := p.NewTypeGenerator(ctx, t)

	if pp.Skip() {
		return
	}

	ctx = ctx.WithOwner(pp.OwnerName())

	switch {
	case pp.Is(midl.TypeStruct):
		pp.GenStruct(ctx)
	case pp.Is(midl.TypeCUnion):
		pp.GenCUnion(ctx)
	case pp.Is(midl.TypeUnion):
		pp.GenUnion(ctx)
	case pp.Is(midl.TypeEnum):
		pp.GenEnum(ctx)
	case pp.Is(midl.TypeInterface):
		pp.GenInterface(ctx)
	case pp.Is(midl.TypePipe):
		pp.GenPipe(ctx)
	}
}
