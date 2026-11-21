package ndr

import (
	"github.com/oiweiwei/midl-gen-go/codegen/golang/emit"
	"github.com/oiweiwei/midl-gen-go/desc"
	"github.com/oiweiwei/midl-gen-go/midl"
	"github.com/oiweiwei/midl-gen-go/msdn/openspecs"
)

type TypeGenerator struct {
	*Emitter
	*desc.Cursor
	GoTypeName   string
	GoMethodName string
	Doc          *openspecs.Page
	OrigType     *midl.Type
}

func (p *Emitter) NewTypeGenerator(ctx *emit.Env, typ *midl.Type) *TypeGenerator {
	scopes := desc.OfType(typ)
	docs, ok := p.MSDN.GetPage(ctx, scopes.Alias())
	if !ok {
		docs, _ = p.MSDN.GetPage(ctx, scopes.Layer().Parent)
	}
	return &TypeGenerator{
		Emitter:    p,
		Cursor:     scopes,
		GoTypeName: emit.GoName(ctx, scopes.Alias()),
		Doc:        docs,
		OrigType:   typ,
	}
}

// OwnerName is the IDL name the members of the type are overridden by: the
// name of the typedef if there is one, else the tag.
func (p *TypeGenerator) OwnerName() string {
	if alias := p.Alias(); alias != "" {
		return alias
	}
	return p.Type().Tag
}

func (p *TypeGenerator) Skip() bool {
	return p.Cursor == nil || p.Layer().Alias == ""
}

func (p *TypeGenerator) GoFieldName(ctx *emit.Env, field *midl.Field) string {

	if field.Name == "" {
		if field.Type.Base().Is(midl.TypeUnion) {
			field.Name = p.GoTypeName
		}
	}

	return emit.GoFieldName(ctx, field)

}
