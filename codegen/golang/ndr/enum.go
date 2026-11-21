package ndr

import (
	"strconv"

	"github.com/oiweiwei/midl-gen-go/codegen/golang/emit"
)

func (p *TypeGenerator) GenEnum(ctx *emit.Env) {

	p.P("//", p.GoTypeName, "type", "represents", emit.RPCName(p.Alias()), "RPC", "enumeration.")
	if p.Doc != nil {
		p.P("//")
		p.GenComment(ctx, p.Doc.Documentation)
	}
	if len(p.Layer().DocString) > 0 {
		p.P("//")
		p.GenCommentLine(ctx, p.Layer().DocString)
	}

	if p.Enum().Is32 {
		// accomodate all possible values.
		p.P("type", p.GoTypeName, "uint32")
	} else {
		p.P("type", p.GoTypeName, p.EnumType())
	}
	p.P()
	p.P("var", "(")
	for _, enum := range p.Enum().Elems {
		if doc, ok := p.Doc.GetSection(enum.Value); ok {
			p.GenComment(ctx, doc.Documentation)
		}
		p.P(emit.GoMergeNames(ctx, p.GoTypeName, emit.GoMemberNameNoReserved(ctx, enum.Value)), p.GoTypeName, "=", strconv.Itoa(enum.ID))
	}
	p.P(")")
	p.P()
	p.Block("func", "(o "+p.GoTypeName+")", "String()", "string", func() {
		p.Block("switch o", func() {
			for _, enum := range p.Enum().Elems {
				n := emit.GoMergeNames(ctx, p.GoTypeName, emit.GoMemberNameNoReserved(ctx, enum.Value))
				p.P("case", n, ":")
				p.P("return", p.Q(n))
			}
		})
		p.P("return", p.Q("Invalid"))
	})
}
