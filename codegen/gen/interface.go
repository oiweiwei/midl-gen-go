package gen

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/oiweiwei/midl-gen-go/codegen/goprint"
	"github.com/oiweiwei/midl-gen-go/desc"
	"github.com/oiweiwei/midl-gen-go/midl"
)

var (
	notUsedOpRe = regexp.MustCompile("(NotUsedOnWire)|(NotImplemented)|(LSA_DP)|(LSA_TM)|(Unused)|(Opnum\\d+Reserved)|(Reserved_Opnum\\d+)|(Reserved\\d+)|(Opnum\\d+NotUsedByProtocol)")
)

func (p *Generator) IsUnusedOp(n string) bool {
	return notUsedOpRe.MatchString(n)
}

// GenServerHandle generates the registration function and the operation
// dispatcher of the server interface.
func (p *Generator) GenServerHandle(ctx *Env, iff *midl.Interface) {

	n := GoName(ctx, iff.Name) + "Server"

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
				w.Line("op", ":=", goprint.Amp(p.OpName(ctx, op, AnyParam))+"{}")
				w.If("err := op.UnmarshalNDRRequest(ctx, r); err != nil", func() {
					w.Return("nil", "err")
				})
				w.Line("req", ":=", goprint.Amp(p.OpName(ctx, op, InParam)+"{}"))
				w.Line("req." + goprint.Call(p.XXX()+"FromOp", "ctx", "op"))
				w.Line("resp, err", ":=", goprint.Call("o."+p.MethodName(ctx, op), "ctx", "req"))
				w.Return("resp."+p.XXX()+"ToOp(ctx, op)", "err")
			}
		})

		w.Return("nil", "nil")
	})
}

func (p *Generator) GenServerInterface(ctx *Env, iff *midl.Interface) {

	n := GoName(ctx, iff.Name) + "Server"

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
				"*"+p.OpName(ctx, op, InParam),
			),
				p.B("", "*"+p.OpName(ctx, op, OutParam), "error"))
		}
	})
}

func (p *Generator) GenUnplementedServer(ctx *Env, iff *midl.Interface) {

	n := "Unimplemented" + GoName(ctx, iff.Name) + "Server"

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
		p.P("func", p.B("", n), p.B(p.MethodName(ctx, op), "context.Context", "*"+p.OpName(ctx, op, InParam)),
			p.B("", "*"+p.OpName(ctx, op, OutParam), "error"), "{ return nil, dcerpc.ErrNotImplemented }")
	}
	p.P()
	p.P("var", "_", GoName(ctx, iff.Name)+"Server", "=", p.B("", "*"+n), p.B("", "nil"))
}

func (p *Generator) GenClient(ctx *Env, iff *midl.Interface) {

	n := GoName(ctx, iff.Name) + "Client"

	dn := p.XXX() + "Default" + n

	p.P()
	p.Structure(dn, func() {
		if iff.IsObject() && iff.Base != nil {
			ctx := p.WithInterfaceBaseNamer(ctx, iff)
			p.P(p.GoInterfaceTypeName(ctx, iff.Base, func(n string) string {
				return n + "Client"
			}))
		}
		p.P("cc", "dcerpc.Conn")
		if iff.IsObject() {
			p.P("ipid", "*dcom.IPID")
		}
	})

	if iff.IsObject() && iff.Base != nil {
		ctx := p.WithInterfaceBaseNamer(ctx, iff)
		p.Block("func", "(o *"+dn+")", p.B(GoName(ctx, iff.BaseName)), p.GoInterfaceTypeName(ctx, iff.Base, func(n string) string { return n + "Client" }), func() {
			p.P("return", p.O(GoName(ctx, iff.BaseName)+"Client"))
		})
	}

	for _, op := range iff.Body.Operations {

		if p.IsUnusedOp(op.Name) {
			continue
		}

		p.P()
		p.P("func", "(o *"+dn+")",
			p.B(p.MethodName(ctx, op),
				"ctx context.Context", "in *"+p.OpName(ctx, op, InParam), "opts ...dcerpc.CallOption",
			),
			p.B("",
				"*"+p.OpName(ctx, op, OutParam), "error",
			), "{")
		p.P("op", ":=", p.B("in."+p.XXX()+"ToOp", "ctx", "nil"))
		if iff.IsObject() {
			p.If("_, ok := dcom.HasIPID(opts); !ok", func() {
				p.If("o.ipid != nil", func() {
					p.P("opts = append(opts, dcom.WithIPID(o.ipid))")
				}, p.Else(func() {
					p.P("return", "nil, fmt.Errorf(\"%s: ipid is missing\", op.OpName())")
				}))
			})
		}
		p.If("err", ":=", p.B("o.cc.Invoke", "ctx", "op", "opts..."), ";", "err != nil", func() {
			p.P("return", "nil, err")
		})
		p.P("out", ":=", p.Amp(p.OpName(ctx, op, OutParam)+"{}"))
		p.P(p.B("out."+p.XXX()+"FromOp", "ctx", "op"))

		if !op.Type.Is(midl.TypeVoid) {

			param := p.OperationReturnValue(ctx, op)

			field := &midl.Field{
				Name:  param.Name,
				Type:  param.Type,
				Attrs: param.Attrs.FieldAttr,
			}

			cmp := "!="

			if field.Type.IsSigned() {
				cmp = "<"
			}

			p.If("op."+GoName(ctx, field.Name), cmp, p.GoTypeZeroValue(ctx, nil, field, desc.Of(field)), func() {
				p.P("return", "out", ",", p.B("fmt.Errorf", `"%s: %w"`, "op.OpName()", p.B("o.cc.Error", "ctx", "op."+GoName(ctx, field.Name))))
			})
		}

		p.P("return", "out, nil")
		p.P("}")
	}

	p.P()
	p.Block("func", "(o *"+dn+")", "AlterContext(ctx context.Context, opts ...dcerpc.Option)", "error", func() {
		p.P("return o.cc.AlterContext(ctx, opts...)")
	})
	p.P()
	p.Block("func", "(o *"+dn+")", "Conn()", "dcerpc.Conn", func() {
		p.P("return o.cc")
	})
	if iff.IsObject() {
		p.P()
		p.Block("func", "(o *"+dn+")", "IPID(ctx context.Context, ipid *dcom.IPID)", n, func() {
			p.If("ipid == nil", func() {
				p.P("ipid = &dcom.IPID{}")
			})
			p.Block("return", "&", dn, func() {
				if iff.Base != nil {
					ctx := p.WithInterfaceBaseNamer(ctx, iff)
					p.P(GoName(ctx, iff.BaseName)+"Client", ":", p.O(GoName(ctx, iff.BaseName)+"Client"+"."+p.B("IPID", "ctx", "ipid")), ",")
				}
				p.P("cc", ":", "o.cc,")
				p.P("ipid", ":", "ipid,")
			})
		})
	}

	p.P()
	p.Block("func", p.B("New"+n, "ctx context.Context", "cc dcerpc.Conn", "opts ...dcerpc.Option"), p.B("", n, "error"), func() {

		if iff.IsObject() {

			p.P("var", "err", "error")

			p.If("!dcom.IsSuperclass(opts)", func() {
				p.P("cc, err", "=", p.B("cc.Bind", "ctx", p.B("append", "opts", p.B("dcerpc.WithAbstractSyntax", p.SyntaxName(ctx, iff)))+"..."), ";")
				p.If("err != nil", func() {
					p.P("return", "nil, err")
				})
			})

			if iff.Base != nil {
				ctx := p.WithInterfaceBaseNamer(ctx, iff)
				p.P("base, err", ":=", p.B(p.GoInterfaceTypeName(ctx, iff.Base, func(n string) string { return "New" + n + "Client" }),
					"ctx", "cc", p.B("append", "opts", p.B("dcom.Superclass", "cc"))+"..."))
				p.If("err != nil", func() {
					p.P("return", "nil, err")
				})
			}

			if iff.Base != nil {
				p.P("ipid, ok := dcom.HasIPID(opts)")
				p.If("ok", func() {
					p.P("base", "=", p.B("base.IPID", "ctx", "ipid"))
				})
			} else {
				p.P("ipid, _ := dcom.HasIPID(opts)")
			}

			p.Block("return", "&", dn, func() {
				if iff.Base != nil {
					ctx := p.WithInterfaceBaseNamer(ctx, iff)
					p.P(GoName(ctx, iff.BaseName)+"Client", ":", "base", ",")
				}
				p.P("cc", ":", "cc,")
				p.P("ipid", ":", "ipid,")
			}, ",", "nil")

		} else {
			p.P("cc, err", ":=", p.B("cc.Bind", "ctx", p.B("append", "opts", p.B("dcerpc.WithAbstractSyntax", p.SyntaxName(ctx, iff)))+"..."), ";")
			p.If("err != nil", func() {
				p.P("return", "nil, err")
			})
			p.P("return", p.Amp(dn+"{cc: cc}"), ",", "nil")
		}
	})
}

func (p *Generator) GenClientInterface(ctx *Env, iff *midl.Interface) {
	n := GoName(ctx, iff.Name) + "Client"
	p.P()
	p.P("//", iff.Name, "interface.")
	p.Block("type", n, "interface", func() {
		if iff.IsObject() && iff.Base != nil {
			ctx := p.WithInterfaceBaseNamer(ctx, iff)
			// p.P()
			// p.P("//", iff.BaseName, "base class.")
			// p.P(p.GoInterfaceTypeName(ctx, iff.Base, baseN))
			p.P()
			p.P("//", iff.BaseName, "retrieval method.")
			p.P(p.B(GoName(ctx, iff.BaseName)), p.GoInterfaceTypeName(ctx, iff.Base, func(n string) string { return n + "Client" }))
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
				"*"+p.OpName(ctx, op, InParam),
				"...dcerpc.CallOption",
			),
				p.B("", "*"+p.OpName(ctx, op, OutParam), "error"))
		}

		p.P()
		p.P("//", "AlterContext alters the client context.")
		p.P(p.B("AlterContext", "context.Context", "...dcerpc.Option"), "error")
		p.P()
		p.P("//", "Conn returns the client connection (unsafe)")
		p.P(p.B("Conn"), "dcerpc.Conn")

		if iff.IsObject() {
			p.P()
			p.P("//", "IPID sets the object interface identifier.")
			p.P(p.B("IPID", "context.Context", "*dcom.IPID"), p.B("", n))
		}
	})
}

func (p *Generator) SyntaxName(ctx *Env, iff *midl.Interface) string {
	n := GoName(ctx, iff.Name) + "Syntax"
	ver := iff.Attrs.Version
	if ver == nil {
		ver = &midl.Version{}
	}
	return fmt.Sprintf("%sV%d_%d", n, ver.Major, ver.Minor)
}

func (p *Generator) SyntaxNameWithImport(ctx *Env, iff *midl.Interface) string {
	n := p.GoInterfaceTypeName(ctx, iff) + "Syntax"
	ver := iff.Attrs.Version
	if ver == nil {
		ver = &midl.Version{}
	}
	return fmt.Sprintf("%sV%d_%d", n, ver.Major, ver.Minor)
}

func (p *Generator) GenInterfaceID(ctx *Env, iff *midl.Interface) {

	if iff.Attrs.UUID == nil {
		return
	}

	n := GoName(ctx, iff.Name) + "Syntax"

	p.P()
	p.P("var", "(")
	if iff.IsObject() {
		_ = p.GoPackageName(ctx, desc.OfType(p.Session.LookupType("IID")))
		p.P("//", iff.Name, "interface", "identifier", iff.Attrs.UUID)
		p.P(GoName(ctx, iff.Name)+"IID", "=", "&dcom.IID", UUIDToGUID(iff.Attrs.UUID))
	}

	p.P("//", "Syntax UUID")
	p.P(n+"UUID", "=", fmt.Sprintf("%#v", iff.Attrs.UUID))
	ver := iff.Attrs.Version
	if ver == nil {
		ver = &midl.Version{Major: 0, Minor: 0}
	}
	p.P("//", "Syntax ID")
	p.P(n+fmt.Sprintf("V%d_%d", ver.Major, ver.Minor), "=", "&dcerpc.SyntaxID{",
		"IfUUID:", n+"UUID", ",", "IfVersionMajor:", ver.Major, ",", "IfVersionMinor:", ver.Minor, "}")
	p.P(")")
}
