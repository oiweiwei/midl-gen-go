package ndr

import (
	"github.com/oiweiwei/midl-gen-go/codegen/golang/emit"
	"github.com/oiweiwei/midl-gen-go/codegen/goprint"
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

	p.Block(goprint.Tokens("type", p.GoTypeName, "interface"), func() {
		p.P("Pipe()", "<-chan", p.GoFieldTypeName(ctx, p.Layer(), p.GenPipeField(ctx, p.Type())))
		p.P("Send(v", p.GoFieldTypeName(ctx, p.Layer(), p.GenPipeField(ctx, p.Type()))+")")
		p.P("Append(v", p.GoFieldTypeName(ctx, p.Layer(), p.GenPipeField(ctx, p.Type()))+")")
		p.P("Recv()", p.B("", p.GoFieldTypeName(ctx, p.Layer(), p.GenPipeField(ctx, p.Type())), "bool"))
		p.P("Close()", "error")
	})

	field := p.GenPipeField(ctx, p.Type())

	p.P()
	p.P("//", "Pipe", "provides", "a", "channel-based", "interface", "to", "send", "data", "to", "the", "RPC", "server.")
	p.P("//", "Note: pipe must be closed after use to avoid resource leaks.")
	p.Block(goprint.Tokens("type", p.XXX()+p.GoTypeName, "struct"), func() {
		p.P("p", "chan", p.GoFieldTypeName(ctx, p.Layer(), field))
		p.P("recv", "[]", p.GoFieldTypeName(ctx, p.Layer(), field))
	})

	p.P()
	p.P("//", "New"+p.XXX()+p.GoTypeName, "creates", "a", "new", p.GoTypeName, "instance.")
	p.Function(goprint.Tokens("New"+p.GoTypeName+"()", p.GoTypeName), func() {
		p.Composite("return &"+p.XXX()+p.GoTypeName, func() {
			p.P("p: make(chan", p.GoFieldTypeName(ctx, p.Layer(), field), "),")
		}, "")
	})

	p.P()
	p.Method("o *"+p.XXX()+p.GoTypeName, goprint.Tokens("Pipe() <-chan", p.GoFieldTypeName(ctx, p.Layer(), field)), func() {
		p.P("return", "o.p")
	})

	p.P()
	p.P("//", "Send", "sends", "data", "to", "the", "RPC", "server", "via", "the", "pipe.")
	p.Method("o *"+p.XXX()+p.GoTypeName, goprint.Tokens("Send(v", p.GoFieldTypeName(ctx, p.Layer(), field)+")"), func() {
		p.If("o != nil && o.p != nil", func() {
			p.P("o.p <- v")
		})
	})

	// append to pipe.
	p.P()
	p.P("//", "Append", "appends", "data", "to", "the", "internal", "buffer", "of", "the", "pipe.")
	p.Method("o *"+p.XXX()+p.GoTypeName, goprint.Tokens("Append(v", p.GoFieldTypeName(ctx, p.Layer(), field)+")"), func() {
		p.If("o != nil", func() {
			p.P("o.recv =", "append(o.recv, v)")
		})
	})

	// receive first item of recv.
	p.P()
	p.P("//", "Recv", "receives", "data", "from", "the", "internal", "buffer", "of", "the", "pipe.")
	p.Method("o *"+p.XXX()+p.GoTypeName, goprint.Tokens("Recv()", p.B("", p.GoFieldTypeName(ctx, p.Layer(), field), "bool")), func() {
		p.If("o != nil", func() {
			p.If("len(o.recv) > 0", func() {
				p.P("v", ":=", "o.recv[0]")
				p.P("o.recv", "=", "o.recv[1:]")
				p.P("return", "v", ",", "true")
			})
		})
		p.P("return", p.GoTypeZeroValue(ctx, p.Layer(), field, desc.Of(field)), ",", "false")
	})

	p.Method("o *"+p.XXX()+p.GoTypeName, "Close() error", func() {
		p.If("o != nil && o.p != nil", func() {
			p.P("close(o.p)")
		})
		p.P("return", "nil")
	})

}
