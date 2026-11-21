package ndr

import (
	"fmt"
	"strings"

	"github.com/oiweiwei/midl-gen-go/codegen/golang/emit"
	"github.com/oiweiwei/midl-gen-go/desc"
	"github.com/oiweiwei/midl-gen-go/midl"
)

func (p *TypeGenerator) GenStructField(ctx *emit.Env, field *midl.Field) {
	if field.IsHandle() {
		return
	}
	if doc, ok := p.Doc.GetSection(field.Name); ok {
		p.GenComment(ctx, doc.Documentation)
	}
	if len(field.Attrs.DocString) > 0 {
		p.GenCommentLine(ctx, field.Attrs.DocString)
	}
	notes := ""
	if !field.IsString() && p.GoFieldTypeName(ctx, p.Layer(), field) == "string" {
		// notes = `// field is not declared as a string` + p.GenStructFieldComment(ctx, field)
	}
	p.P(p.GoFieldName(ctx, field), p.GoFieldTypeName(ctx, p.Layer(), field), p.GenTag(ctx, field), notes)
	for _, field := range field.Attrs.Layout {
		p.GenStructField(ctx, field)
	}
}

func (p *TypeGenerator) GenStructFieldComment(ctx *emit.Env, field *midl.Field, defaultPtr ...string) string {
	ret := ""
	for i, scope := range desc.Resolve(field).Layers {
		ret += fmt.Sprintf("(%d:", i+1)
		attr := scope.Attr.String()
		if len(defaultPtr) > 0 && len(scope.Nodes) > 0 && scope.Nodes[0].Is(midl.TypePointer) && scope.Attr.Pointer == midl.PointerTypeNone {
			if attr != "" {
				attr += ","
			}
			attr += "pointer=" + defaultPtr[0]
		}

		if attr != "" {
			ret += "{" + attr + "}"
		}
		for _, typ := range scope.Nodes {
			switch {
			case typ.Is(midl.TypePointer):
				ret += "*"
				ret += fmt.Sprintf("(%d)", typ.Pointer)
			case typ.Is(midl.TypeArray):
				ret += "["
				if typ.Array.IsFixed() {
					ret += fmt.Sprintf("%d", typ.Array.Size())
				} else {
					ret += typ.Dim.String()
				}
				ret += "]"
			default:
				ret += "(" + typ.Kind.String() + ")"
			}
		}
		ret += ")"
	}

	return ret
}

func (p *TypeGenerator) GenSubTypes(ctx *emit.Env, fields []*midl.Field) {

	for _, field := range fields {
		p.GenSubType(ctx, field)
		for _, field := range field.Attrs.Layout {
			p.GenSubType(ctx, field)
		}
	}
}

func (p *TypeGenerator) GenSubType(ctx *emit.Env, field *midl.Field) {

	switch t := field.Type.Base(); {
	case t.Is(midl.TypeStruct) || t.Is(midl.TypeUnion) || t.Is(midl.TypeCUnion):

		if emit.LookupName(ctx, desc.Of(field).Base()) != "" {
			return
		}

		n := p.EmbeddedTypeName(ctx, p.Layer(), field)

		if strings.Contains(n, p.XXX()) {
			panic("invalid type " + n)
		}

		parent := p.Layer().Parent
		if parent == "" {
			parent = p.Alias()
		}

		p.GenType(ctx, &midl.Type{
			Kind:  midl.TypeAttribute,
			Attrs: field.Attrs.TypeAttr().Merge(&midl.TypeAttr{Alias: emit.Escape(n), Parent: parent}),
			Elem:  t,
		})
	}
}

func (p *TypeGenerator) GenInterface(ctx *emit.Env) {
	tt := *p.Session.LookupType("MInterfacePointer")
	tt.Attrs = tt.Attrs.Merge(p.OrigType.Attrs)
	p.NewTypeGenerator(ctx, &tt).GenStruct(ctx)
}

func (p *TypeGenerator) GenStruct(ctx *emit.Env) {

	if p.Layer().Parent != "" {
		p.P("//", p.GoTypeName, "structure", "represents", emit.RPCName(p.Layer().Parent), "structure", "anonymous", "member.")
	} else {
		p.P("//", p.GoTypeName, "structure", "represents", emit.RPCName(p.Alias()), "RPC", "structure.")
	}
	if p.Doc != nil {
		p.P("//")
		p.GenComment(ctx, p.Doc.Documentation)
	}
	if len(p.Layer().DocString) > 0 {
		p.P("//")
		p.GenCommentLine(ctx, p.Layer().DocString)
	}

	if n := p.Layer().Names; len(n) > 0 {
		ctx = p.WithNamer(ctx, p.Cursor.WithAlias(n[0]))
		names := p.GoScopeTypeName(ctx, p.Layer(), nil, p.Cursor.WithAlias(n[0]), true)
		p.P("type", p.GoTypeName, names)
		p.P()
		p.P("func", "(o *"+p.GoTypeName+")", emit.GoName(ctx, n[0]), "()", "*"+names, "{", "return", p.B("(*"+names+")", "o"), "}")
	} else {

		mask := p.GenNullMask(ctx)
		ctx = ctx.WithNullMask(mask)

		p.Structure(p.GoTypeName, func() {
			if mask != "" {
				p.P()
				p.P("//", mask, "is used to carry information on null-valued primitive values.")
				p.P(p.NullMask(), mask)
				p.P()
			}
			for _, field := range p.Struct().Fields {
				p.GenStructField(ctx, field)
			}
		})
	}

	p.GenStructPreparePayload(ctx)
	p.GenStructNDRSizeInfo(ctx)
	p.GenLayout(ctx)
	p.GenStructMarshalNDR(ctx)
	p.GenStructUnmarshalNDR(ctx)
	p.GenSubTypes(ctx, p.Struct().Fields)

}

func (p *TypeGenerator) GenNullMask(ctx *emit.Env) string {

	nullable := []*midl.Field{}

	for _, field := range p.Struct().Fields {
		if emit.IsIgnoredPointer(ctx, field) && emit.GoFieldName(ctx, field) != "_" {
			nullable = append(nullable, field)
		}
	}

	mask := p.GoTypeName + p.NullMask()

	if len(nullable) == 0 {
		return ""
	}

	p.P("type", mask, "ndr.NullMask")
	p.P()
	p.P("var", "(")
	for i, field := range nullable {
		p.P(mask+emit.GoFieldName(ctx, field), mask, "=", "1", "<<", i)
	}
	p.P(")")
	p.P()
	p.P("func", p.B("", "o "+mask), "IsSet(v "+mask+") bool", "{ return o&v != 0 }")
	p.P()
	p.P("func", p.B("", "o "+mask), "Set(v "+mask+") "+mask, "{ return o|v }")
	p.P()
	p.P("func", p.B("", "o "+mask), "Unset(v "+mask+") "+mask, "{ return o&^v }")
	p.P()

	return mask
}

func (p *TypeGenerator) GenLayout(ctx *emit.Env) {
	if p.Layer().IsLayout {
		p.P("func", "(o *"+p.GoTypeName+")", "NDRLayout()", "{}")
	}
}
