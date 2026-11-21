package golang

import (
	"path/filepath"

	"github.com/oiweiwei/midl-gen-go/codegen/golang/emit"
	"github.com/oiweiwei/midl-gen-go/codegen/goprint"
	"github.com/oiweiwei/midl-gen-go/desc"
	"github.com/oiweiwei/midl-gen-go/midl"
)

func (p *Generator) GenClient(ctx *emit.Env, iff *midl.Interface) {

	n := emit.GoName(ctx, iff.Name) + "Client"

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
		p.Method("o *"+dn, goprint.Tokens(p.B(emit.GoName(ctx, iff.BaseName)), p.GoInterfaceTypeName(ctx, iff.Base, func(n string) string { return n + "Client" })), func() {
			p.P("return", p.O(emit.GoName(ctx, iff.BaseName)+"Client"))
		})
	}

	for _, op := range iff.Body.Operations {

		if p.IsUnusedOp(op.Name) {
			continue
		}

		p.P()
		p.Method("o *"+dn, goprint.Tokens(p.B(p.MethodName(ctx, op),
			"ctx context.Context", "in *"+p.OpName(ctx, op, emit.InParam), "opts ...dcerpc.CallOption",
		), p.B("",
			"*"+p.OpName(ctx, op, emit.OutParam), "error",
		)), func() {
			p.P("op", ":=", p.B("in."+p.XXX()+"ToOp", "ctx", "nil"))
			if iff.IsObject() {
				p.If("_, ok := dcom.HasIPID(opts); !ok", func() {
					p.IfElse("o.ipid != nil", func() {
						p.P("opts = append(opts, dcom.WithIPID(o.ipid))")
					}, func() {
						p.P("return", "nil, fmt.Errorf(\"%s: ipid is missing\", op.OpName())")
					})
				})
			}
			p.If(goprint.Tokens("err :=", p.B("o.cc.Invoke", "ctx", "op", "opts..."), "; err != nil"), func() {
				p.P("return", "nil, err")
			})
			p.P("out", ":=", p.Amp(p.OpName(ctx, op, emit.OutParam)+"{}"))
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

				p.If(goprint.Tokens("op."+emit.GoParamName(ctx.WithOwner(op.Name), param), cmp, p.GoTypeZeroValue(ctx, nil, field, desc.Of(field))), func() {
					p.P("return", "out", ",", p.B("fmt.Errorf", `"%s: %w"`, "op.OpName()", p.B("o.cc.Error", "ctx", "op."+emit.GoParamName(ctx.WithOwner(op.Name), param))))
				})
			}

			p.P("return", "out, nil")
		})
	}

	p.P()
	p.Method("o *"+dn, "AlterContext(ctx context.Context, opts ...dcerpc.Option) error", func() {
		p.P("return o.cc.AlterContext(ctx, opts...)")
	})
	p.P()
	p.Method("o *"+dn, "Conn() dcerpc.Conn", func() {
		p.P("return o.cc")
	})
	if iff.IsObject() {
		p.P()
		p.Method("o *"+dn, goprint.Tokens("IPID(ctx context.Context, ipid *dcom.IPID)", n), func() {
			p.If("ipid == nil", func() {
				p.P("ipid = &dcom.IPID{}")
			})
			p.Block(goprint.Tokens("return &", dn), func() {
				if iff.Base != nil {
					ctx := p.WithInterfaceBaseNamer(ctx, iff)
					p.P(emit.GoName(ctx, iff.BaseName)+"Client", ":", p.O(emit.GoName(ctx, iff.BaseName)+"Client"+"."+p.B("IPID", "ctx", "ipid")), ",")
				}
				p.P("cc", ":", "o.cc,")
				p.P("ipid", ":", "ipid,")
			})
		})
	}

	p.P()
	p.Function(goprint.Tokens(p.B("New"+n, "ctx context.Context", "cc dcerpc.Conn", "opts ...dcerpc.Option"), p.B("", n, "error")), func() {

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

			p.Composite(goprint.Tokens("return", "&", dn), func() {
				if iff.Base != nil {
					ctx := p.WithInterfaceBaseNamer(ctx, iff)
					p.P(emit.GoName(ctx, iff.BaseName)+"Client", ":", "base", ",")
				}
				p.P("cc", ":", "cc,")
				p.P("ipid", ":", "ipid,")
			}, ", nil")

		} else {
			p.P("cc, err", ":=", p.B("cc.Bind", "ctx", p.B("append", "opts", p.B("dcerpc.WithAbstractSyntax", p.SyntaxName(ctx, iff)))+"..."), ";")
			p.If("err != nil", func() {
				p.P("return", "nil, err")
			})
			p.P("return", p.Amp(dn+"{cc: cc}"), ",", "nil")
		}
	})
}

func (p *Generator) GenClientInterface(ctx *emit.Env, iff *midl.Interface) {
	n := emit.GoName(ctx, iff.Name) + "Client"
	p.P()
	p.P("//", iff.Name, "interface.")
	p.Interface(n, func() {
		if iff.IsObject() && iff.Base != nil {
			ctx := p.WithInterfaceBaseNamer(ctx, iff)
			// p.P()
			// p.P("//", iff.BaseName, "base class.")
			// p.P(p.GoInterfaceTypeName(ctx, iff.Base, baseN))
			p.P()
			p.P("//", iff.BaseName, "retrieval method.")
			p.P(p.B(emit.GoName(ctx, iff.BaseName)), p.GoInterfaceTypeName(ctx, iff.Base, func(n string) string { return n + "Client" }))
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
				"...dcerpc.CallOption",
			),
				p.B("", "*"+p.OpName(ctx, op, emit.OutParam), "error"))
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

func (p *Generator) GenClientSet(ctx *emit.Env, f *midl.File) {

	n := "Client"

	hook := func(n string) string {
		return n + "Client"
	}

	var dcom *midl.File

	for _, f := range p.Session.Files() {
		if f.PkgIs("dcom") {
			dcom = f
			break
		}
	}

	base := dcom.GoPkgBase
	if base == "" {
		base = p.ImportsPath
	}

	if !f.PkgIs("dcom") {
		p.AddImport(emit.Import{
			Name: "dcom_client",
			Path: filepath.Join(dcom.GoPkgBase, "dcom", "client"),
		})
	}
	p.AddImport(emit.Import{
		Name: "dcom",
		Path: filepath.Join(dcom.GoPkgBase, "dcom"),
	})

	p.P("//", f.GoPkg, "client set.")
	p.Interface(n, func() {
		if !f.PkgIs("dcom") {
			p.P()
			p.P("//", "DCOM common interfaces")
			for _, iff := range dcom.Interfaces {
				ctx := p.WithInterfaceNamer(ctx, iff)
				if iff.Body.Operations != nil && iff.Base != nil {
					p.P(emit.GoName(ctx, iff.Name), "()", p.GoInterfaceTypeName(ctx, iff, hook))
				}
			}
		}
		p.P()
		p.P("//", "Package specific interfaces")
		for _, iff := range f.Interfaces {
			if iff.Body.Operations != nil && iff.Base != nil {
				p.P(emit.GoName(ctx, iff.Name), "()", p.GoInterfaceTypeName(ctx, iff, hook))
			}
		}
		p.P("//", "AlterContext alters the client context.")
		p.P(p.B("AlterContext", "context.Context", "...dcerpc.Option"), "error")
		p.P()
		p.P("//", "Conn returns the client connection (unsafe)")
		p.P(p.B("Conn"), "dcerpc.Conn")
		p.P()
		p.P("//", "IPID sets the object interface identifier.")
		p.P(p.B("IPID", "context.Context", "*dcom.IPID"), p.B("", n))
	})

	dn := p.XXX() + "Default" + n
	p.Block(goprint.Tokens("type", dn, "struct"), func() {
		p.P("cc", "dcerpc.Conn")
		p.P()
		if !f.PkgIs("dcom") {
			p.P("dcomClient", "dcom_client.Client")
			p.P()
		}
		for _, iff := range f.Interfaces {
			if iff.Body.Operations != nil && iff.Base != nil {
				p.P(emit.GoNamePrivate(ctx, iff.Name), p.GoInterfaceTypeName(ctx, iff, hook))
			}
		}
	})

	if !f.PkgIs("dcom") {
		for _, iff := range dcom.Interfaces {
			ctx := p.WithInterfaceNamer(ctx, iff)
			if iff.Body.Operations != nil && iff.Base != nil {
				p.P()
				p.Method("o *"+dn, goprint.Tokens(emit.GoName(ctx, iff.Name), "()", p.GoInterfaceTypeName(ctx, iff, hook)), func() {
					p.P("return", p.O("dcomClient"+"."+emit.GoName(ctx, iff.Name)), "()")
				})
			}
		}
	}

	for _, iff := range f.Interfaces {
		if iff.Body.Operations != nil && iff.Base != nil {
			p.P()
			p.Method("o *"+dn, goprint.Tokens(emit.GoName(ctx, iff.Name), "()", p.GoInterfaceTypeName(ctx, iff, hook)), func() {
				p.P("return", p.O(emit.GoNamePrivate(ctx, iff.Name)))
			})
		}
	}

	p.P()

	p.Block(goprint.Tokens("func NewClient (ctx context.Context, cc dcerpc.Conn, opts ...dcerpc.Option)", p.B("", n, "error")), func() {

		p.P()
		p.P("opts", "=", "append", "(", "opts", ",")
		for _, iff := range f.Interfaces {
			if iff.Body.Operations != nil && iff.Base != nil {
				p.P(p.B("dcerpc.WithAbstractSyntax", p.SyntaxNameWithImport(ctx, iff)), ",")
			}
		}
		if !f.PkgIs("dcom") {
			for _, iff := range dcom.Interfaces {
				ctx := p.WithInterfaceNamer(ctx, iff)
				if iff.Body.Operations != nil && iff.Base != nil {
					p.P(p.B("dcerpc.WithAbstractSyntax", p.SyntaxNameWithImport(ctx, iff)), ",")
				}
			}
		}
		p.P(")")
		p.P()
		p.P("cc, err", ":=", p.B("cc.Bind", "ctx", "opts..."), ";")
		p.If("err != nil", func() {
			p.P("return", "nil, err")
		})
		p.P()
		p.P("o", ":=", p.Amp(dn+"{cc: cc}"))
		p.P()
		if !f.PkgIs("dcom") {
			p.P("dcomClient", ",", "err", ":=", p.B("dcom_client.NewClient", "ctx", "cc", p.B("append", "opts", p.B("dcerpc.WithNoBind", "cc"))+"..."))
			p.If("err != nil", func() {
				p.P("return", "nil, err")
			})
			p.P("o.dcomClient", "=", "dcomClient")
		}
		p.P()
		p.P("sub, ok := cc.(dcerpc.SubConn)")
		p.If("!ok", func() {
			p.P("return", "nil, fmt.Errorf(\"sub-conn is not supported\")")
		})
		p.P()
		for _, iff := range f.Interfaces {
			if iff.Body.Operations != nil && iff.Base != nil {
				p.P()
				iffName := emit.GoNamePrivate(ctx, iff.Name)
				p.P(iffName+"SubConn, err", ":=", p.B("sub.SubConn", "ctx", p.SyntaxNameWithImport(ctx, iff)))
				p.If("err != nil", func() {
					p.P("//", "XXX: use main subconnection as a last resort")
					p.P("//", "it was noticed that we can reuse the main connection for dcom interfaces")
					p.P(iffName+"SubConn", "=", "sub")
				})
				p.P()
				hook := func(n string) string { return "New" + n + "Client" }
				p.P("o."+emit.GoNamePrivate(ctx, iff.Name), ",", "err", "=",
					p.B(p.GoInterfaceTypeName(ctx, iff, hook),
						"ctx", iffName+"SubConn",
						p.B("append", "opts", p.B("dcerpc.WithNoBind", iffName+"SubConn"))+"..."))
			}
		}

		p.P("return", "o", ",", "nil")
	})

	p.P()
	p.Method("o *"+dn, "AlterContext(ctx context.Context, opts ...dcerpc.Option) error", func() {
		p.P("return o.cc.AlterContext(ctx, opts...)")
	})
	p.P()
	p.Method("o *"+dn, "Conn() dcerpc.Conn", func() {
		p.P("return o.cc")
	})
	p.P()
	p.Method("o *"+dn, goprint.Tokens("IPID(ctx context.Context, ipid *dcom.IPID)", n), func() {
		p.If("ipid == nil", func() {
			p.P("ipid = &dcom.IPID{}")
		})
		p.Block(goprint.Tokens("return &", dn), func() {
			if !f.PkgIs("dcom") {
				p.P("dcomClient", ":", p.O("dcomClient."+p.B("IPID", "ctx", "ipid")), ",")
			}
			for _, iff := range f.Interfaces {
				if iff.Body.Operations != nil && iff.Base != nil {
					p.P(emit.GoNamePrivate(ctx, iff.Name), ":", p.O(emit.GoNamePrivate(ctx, iff.Name)+"."+p.B("IPID", "ctx", "ipid")), ",")
				}
			}
			p.P("cc", ":", "o.cc,")
		})
	})

}

// clientFeature generates the client: the client interface, its implementation
// and, for DCOM, the client set of an IDL file.
type clientFeature struct{}

func (clientFeature) Name() string { return "client" }
func (clientFeature) Doc() string {
	return "the client interface and its implementation, the DCOM client set"
}
func (clientFeature) Requires() []string { return nil }
func (clientFeature) Phases() []Phase    { return []Phase{BeforeTypes, AfterTypes} }

func (clientFeature) Interface(ctx *emit.Env, p *Generator, iff *midl.Interface, phase Phase) {
	switch phase {
	case BeforeTypes:
		p.GenClientInterface(ctx, iff)
	case AfterTypes:
		p.GenClient(ctx, iff)
	}
}

func (clientFeature) File(ctx *emit.Env, p *Generator, f *midl.File) {
	if f.IsDCOM() {
		p.Reset(ctx, f, "client")
		p.GenClientSet(ctx, f)
	}
}
