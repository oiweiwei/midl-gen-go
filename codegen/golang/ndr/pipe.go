package ndr

import (
	"github.com/oiweiwei/midl-gen-go/codegen/golang/emit"
	"github.com/oiweiwei/midl-gen-go/desc"
	"github.com/oiweiwei/midl-gen-go/midl"
)

func (p *TypeGenerator) GenPipeField(ctx *emit.Env, typ *midl.Type) *midl.Field {
	return &midl.Field{
		Attrs: &midl.FieldAttr{NoSizeLimit: false},
		Type: &midl.Type{
			Kind:  midl.TypeArray,
			Array: &midl.Array{Bound: midl.ArrayBound{Upper: -1, Lower: 0}},
			Attrs: &midl.TypeAttr{},
			Elem:  typ.Elem,
		},
	}
}

func (p *TypeGenerator) GenPipe(ctx *emit.Env) {

	p.P("//", p.GoTypeName, "type", "represents", emit.RPCName(p.Alias()), "RPC", "pipe.")
	if p.Doc != nil {
		p.P("//")
		p.GenComment(ctx, p.Doc.Documentation)
	}
	if len(p.Layer().DocString) > 0 {
		p.P("//")
		p.GenCommentLine(ctx, p.Layer().DocString)
	}

	p.P("type", p.GoTypeName, "interface", "{")
	p.P("Pipe()", "<-chan", p.GoFieldTypeName(ctx, p.Layer(), p.GenPipeField(ctx, p.Type())))
	p.P("Send(v", p.GoFieldTypeName(ctx, p.Layer(), p.GenPipeField(ctx, p.Type()))+")")
	p.P("Append(v", p.GoFieldTypeName(ctx, p.Layer(), p.GenPipeField(ctx, p.Type()))+")")
	p.P("Recv()", p.B("", p.GoFieldTypeName(ctx, p.Layer(), p.GenPipeField(ctx, p.Type())), "bool"))
	p.P("Close()", "error")
	p.P("}")

	field := p.GenPipeField(ctx, p.Type())

	p.P()
	p.P("//", "Pipe", "provides", "a", "channel-based", "interface", "to", "send", "data", "to", "the", "RPC", "server.")
	p.P("//", "Note: pipe must be closed after use to avoid resource leaks.")
	p.P("type", p.XXX()+p.GoTypeName, "struct {")
	p.P("p", "chan", p.GoFieldTypeName(ctx, p.Layer(), field))
	p.P("recv", "[]", p.GoFieldTypeName(ctx, p.Layer(), field))
	p.P("}")

	p.P()
	p.P("//", "New"+p.XXX()+p.GoTypeName, "creates", "a", "new", p.GoTypeName, "instance.")
	p.P("func", "New"+p.GoTypeName+"()", p.GoTypeName, "{")
	p.P("return", "&"+p.XXX()+p.GoTypeName+"{")
	p.P("p: make(chan", p.GoFieldTypeName(ctx, p.Layer(), field), "),")
	p.P("}")
	p.P("}")

	p.P()
	p.P("func", "(o *"+p.XXX()+p.GoTypeName+")", "Pipe()", "<-chan", p.GoFieldTypeName(ctx, p.Layer(), field), "{")
	p.P("return", "o.p")
	p.P("}")

	p.P()
	p.P("//", "Send", "sends", "data", "to", "the", "RPC", "server", "via", "the", "pipe.")
	p.P("func", "(o *"+p.XXX()+p.GoTypeName+")", "Send(v", p.GoFieldTypeName(ctx, p.Layer(), field)+")", "{")
	p.P("if o != nil && o.p != nil {")
	p.P("o.p <- v")
	p.P("}")
	p.P("}")

	// append to pipe.
	p.P()
	p.P("//", "Append", "appends", "data", "to", "the", "internal", "buffer", "of", "the", "pipe.")
	p.P("func", "(o *"+p.XXX()+p.GoTypeName+")", "Append(v", p.GoFieldTypeName(ctx, p.Layer(), field)+")", "{")
	p.P("if o != nil {")
	p.P("o.recv =", "append(o.recv, v)")
	p.P("}")
	p.P("}")

	// receive first item of recv.
	p.P()
	p.P("//", "Recv", "receives", "data", "from", "the", "internal", "buffer", "of", "the", "pipe.")
	p.P("func", "(o *"+p.XXX()+p.GoTypeName+")", "Recv()", p.B("", p.GoFieldTypeName(ctx, p.Layer(), field), "bool"), "{")
	p.P("if o != nil {")
	p.P("if len(o.recv) > 0 {")
	p.P("v", ":=", "o.recv[0]")
	p.P("o.recv", "=", "o.recv[1:]")
	p.P("return", "v", ",", "true")
	p.P("}")
	p.P("}")
	p.P("return", p.GoTypeZeroValue(ctx, p.Layer(), field, desc.Of(field)), ",", "false")
	p.P("}")

	p.P("func", "(o *"+p.XXX()+p.GoTypeName+")", "Close()", "error", "{")
	p.If("o != nil && o.p != nil", func() {
		p.P("close(o.p)")
	})
	p.P("return", "nil")
	p.P("}")

}
