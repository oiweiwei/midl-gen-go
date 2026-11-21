package golang

import (
	"strconv"

	"github.com/oiweiwei/midl-gen-go/codegen/golang/emit"
	"github.com/oiweiwei/midl-gen-go/codegen/goprint"
	"github.com/oiweiwei/midl-gen-go/midl"
)

// GenServerHandle generates the registration function and the operation
// dispatcher of the server interface.
func (p *Generator) GenServerHandle(ctx *emit.Env, iff *midl.Interface) {

	n := emit.GoName(ctx, iff.Name) + "Server"

	w := p.W()

	handleParams := []goprint.Param{goprint.Named("ctx", "context.Context"), goprint.Named("opNum", "int"), goprint.Named("r", "ndr.Reader")}
	handleResults := []goprint.Param{goprint.Unnamed("dcerpc.Operation"), goprint.Unnamed("error")}

	w.Blank()
	w.Func(goprint.Func{
		Name:   "Register" + n,
		Params: []goprint.Param{goprint.Named("conn", "dcerpc.Conn"), goprint.Named("o", n), goprint.Named("opts", "...dcerpc.Option")},
	}, func() {
		w.Line(goprint.Call("conn.RegisterServer", goprint.Call("New"+n+"Handle", "o"),
			goprint.Call("append", "opts", goprint.Call("dcerpc.WithAbstractSyntax", p.SyntaxName(ctx, iff)))+"..."))
	})

	w.Blank()
	w.Func(goprint.Func{
		Name:    "New" + n + "Handle",
		Params:  []goprint.Param{goprint.Named("o", n)},
		Results: []goprint.Param{goprint.Unnamed("dcerpc.ServerHandle")},
	}, func() {
		w.Block("return "+goprint.Func{Params: handleParams, Results: handleResults}.Signature(), func() {
			w.Return(goprint.Call(n+"Handle", "ctx", "o", "opNum", "r"))
		})
	})

	w.Blank()
	w.Func(goprint.Func{
		Name:    n + "Handle",
		Params:  append([]goprint.Param{goprint.Named("ctx", "context.Context"), goprint.Named("o", n)}, handleParams[1:]...),
		Results: handleResults,
	}, func() {

		first := 0
		if len(iff.Body.Operations) > 0 {
			first = iff.Body.Operations[0].OpNum
		}

		if first > 0 {
			w.If("opNum < "+strconv.Itoa(first), func() {
				ctx := p.WithInterfaceBaseNamer(ctx, iff)
				w.Comment(iff.BaseName + " base method.")
				w.Return(goprint.Call(p.GoInterfaceTypeName(ctx, iff.Base, func(n string) string {
					return n + "ServerHandle"
				}), "ctx", "o", "opNum", "r"))
			})
		}

		w.Switch("opNum", func() {
			for _, op := range iff.Body.Operations {
				w.Line("case "+strconv.Itoa(op.OpNum)+":", "//", op.Name)
				if p.IsUnusedOp(op.Name) {
					w.Comment(op.Name)
					w.Return("nil", "nil")
					continue
				}
				w.Line("op", ":=", goprint.Amp(p.OpName(ctx, op, emit.AnyParam))+"{}")
				w.If("err := op.UnmarshalNDRRequest(ctx, r); err != nil", func() {
					w.Return("nil", "err")
				})
				w.Line("req", ":=", goprint.Amp(p.OpName(ctx, op, emit.InParam)+"{}"))
				w.Line("req." + goprint.Call(p.XXX()+"FromOp", "ctx", "op"))
				w.Line("resp, err", ":=", goprint.Call("o."+p.MethodName(ctx, op), "ctx", "req"))
				w.Return("resp."+p.XXX()+"ToOp(ctx, op)", "err")
			}
		})

		w.Return("nil", "nil")
	})
}

func (p *Generator) GenServerInterface(ctx *emit.Env, iff *midl.Interface) {

	n := emit.GoName(ctx, iff.Name) + "Server"

	p.P()
	p.P("//", iff.Name, "server interface.")
	p.Block("type", n, "interface", func() {
		if iff.IsObject() && iff.Base != nil {
			ctx := p.WithInterfaceBaseNamer(ctx, iff)
			p.P()
			p.P("//", iff.BaseName, "base class.")
			p.P(p.GoInterfaceTypeName(ctx, iff.Base, func(n string) string {
				return n + "Server"
			}))
		}
		for _, op := range iff.Body.Operations {
			// gen operation doc.
			doc, ok := p.MSDN.GetPage(ctx, op.Name)
			if ok {
				p.P()
				p.GenComment(ctx, doc.Documentation)
			} else {
				p.P()
				p.P("//", op.Name, "operation.")
			}
			for _, field := range []string{"Return Values", "Exceptions Thrown"} {
				if doc, ok := doc.GetSection(field); ok {
					p.P("//")
					p.GenComment(ctx, doc.Documentation)
				}
			}
			if p.IsUnusedOp(op.Name) {
				p.P("//", p.MethodName(ctx, op))
				continue
			}

			p.P(p.B(p.MethodName(ctx, op),
				"context.Context",
				"*"+p.OpName(ctx, op, emit.InParam),
			),
				p.B("", "*"+p.OpName(ctx, op, emit.OutParam), "error"))
		}
	})
}

func (p *Generator) GenUnplementedServer(ctx *emit.Env, iff *midl.Interface) {

	n := "Unimplemented" + emit.GoName(ctx, iff.Name) + "Server"

	p.P()
	p.P("//", "Unimplemented", iff.Name)
	p.Block("type", n, "struct", func() {
		if iff.IsObject() && iff.Base != nil {
			ctx := p.WithInterfaceBaseNamer(ctx, iff)
			p.P()
			p.P(p.GoInterfaceTypeName(ctx, iff.Base, func(n string) string {
				return "Unimplemented" + n + "Server"
			}))
		}
	})

	p.P()
	for _, op := range iff.Body.Operations {
		if p.IsUnusedOp(op.Name) {
			continue
		}
		p.P("func", p.B("", n), p.B(p.MethodName(ctx, op), "context.Context", "*"+p.OpName(ctx, op, emit.InParam)),
			p.B("", "*"+p.OpName(ctx, op, emit.OutParam), "error"), "{ return nil, dcerpc.ErrNotImplemented }")
	}
	p.P()
	p.P("var", "_", emit.GoName(ctx, iff.Name)+"Server", "=", p.B("", "*"+n), p.B("", "nil"))
}

// serverFeature generates the server: the server interface, the operation
// dispatcher and the unimplemented server, in server.go.
type serverFeature struct{}

func (serverFeature) Name() string { return "server" }
func (serverFeature) Doc() string {
	return "the server interface, its operation dispatcher and the unimplemented server"
}
func (serverFeature) Requires() []string { return nil }
func (serverFeature) Phases() []Phase    { return []Phase{InFile} }
func (serverFeature) FileName() string   { return "server" }

func (serverFeature) Interface(ctx *emit.Env, p *Generator, iff *midl.Interface, phase Phase) {
	p.GenServerInterface(ctx, iff)
	p.GenServerHandle(ctx, iff)
	p.GenUnplementedServer(ctx, iff)
}
