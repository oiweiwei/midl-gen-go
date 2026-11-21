package ndr

import (
	"strings"

	"github.com/oiweiwei/midl-gen-go/codegen/golang/emit"
	"github.com/oiweiwei/midl-gen-go/desc"
	"github.com/oiweiwei/midl-gen-go/desc/ndr"
	"github.com/oiweiwei/midl-gen-go/midl"
)

type ParamGenerator struct {
	*Emitter
	*TypeGenerator
}

func (p *Emitter) NewParamGenerator(ctx *emit.Env, typ *midl.Type) *ParamGenerator {
	return &ParamGenerator{
		Emitter:       p,
		TypeGenerator: p.NewTypeGenerator(ctx, typ),
	}
}

func (p *Emitter) GenOperation(ctx *emit.Env, op *midl.Operation) {
	if p.IsUnusedOp(op.Name) {
		return
	}
	ctx = ctx.WithOperation().WithOwner(op.Name)

	mask := p.GenOperationNullMask(ctx, op)

	ctx = ctx.WithNullMask(mask)

	p.GenOperationStruct(ctx, op, emit.AnyParam)
	p.GenOperationMarshalNDR(ctx, op, emit.InParam)
	p.GenOperationUnmarshalNDR(ctx, op, emit.InParam)
	p.GenOperationMarshalNDR(ctx, op, emit.OutParam)
	p.GenOperationUnmarshalNDR(ctx, op, emit.OutParam)
	p.GenOperationStruct(ctx, op, emit.InParam)
	p.GenOperationStruct(ctx, op, emit.OutParam)
}

func (p *Emitter) GenOperationMarshalNDR(ctx *emit.Env, op *midl.Operation, dir int) {

	pN := emit.ParamName(dir)
	fN := "MarshalNDR" + pN
	opN := p.OpName(ctx, op, emit.AnyParam)

	strct := &midl.Type{
		Kind: midl.TypeAttribute,
		Attrs: &midl.TypeAttr{
			Alias: emit.Escape(p.OpName(ctx, op, dir)),
		},
		Elem: &midl.Type{
			Kind: midl.TypeStruct, Struct: &midl.Struct{}},
	}

	for _, param := range p.OperationParams(ctx, op) {
		if !p.IsDir(ctx, param.Attrs.Direction, dir) || param.IsHandle() {
			continue
		}
		strct.Elem.Struct.Fields = append(strct.Elem.Struct.Fields, &midl.Field{
			Name:  param.Name,
			Type:  param.Type,
			Attrs: param.Attrs.FieldAttr,
		})
	}

	pp := p.NewTypeGenerator(ctx, strct)

	p.P()
	p.Block("func", "(o *"+opN+")", p.XXX()+"Prepare"+pN+"Payload", "(ctx context.Context)", "error", func() {
		pp.GenStructPreparePayloadFields(ctx, strct.Elem.Struct.Fields)
		p.PrepareOperationPayloadHook(ctx, dir)
		p.P("return nil")
	})

	p.P()
	p.Block("func", "(o *"+opN+")", fN, "(ctx context.Context, w ndr.Writer)", "error", func() {

		p.CheckErr("o." + p.XXX() + "Prepare" + pN + "Payload(ctx)")

		for _, param := range p.OperationParams(ctx, op) {

			if !p.IsDir(ctx, param.Attrs.Direction, dir) || param.IsHandle() {
				continue
			}

			field := param.Field()

			scopes := desc.Of(field)
			if scopes.Is(midl.TypePointer) &&
				(scopes.Layer().Pointer == midl.PointerTypeNone || scopes.Layer().Pointer == midl.PointerTypeRef || scopes.Layer().Pointer == midl.PointerTypeRefWeak) &&
				// pointers to interfaces are always unique pointers.
				!scopes.Next().Is(midl.TypeInterface) {
				// skip ref pointer.
				scopes = scopes.Next()
			}

			p.P("//", field.Name, pp.GenParamComment(ctx, param), pp.GenStructFieldComment(ctx, field))
			p.Block("", func() {
				pp.GenFieldMarshalNDR(ctx, field, scopes)
				if ndr.IsDeferrable(scopes) {
					p.CheckErr(p.B("w.WriteDeferred"))
				}
			})

		}

		p.P("return nil")
	})

}

func (p *Emitter) PrepareOperationPayloadHook(ctx *emit.Env, dir int) {
	pN := emit.ParamName(dir)
	p.If("hook, ok := (interface{})(o).(interface { AfterPrepare"+pN+"Payload(context.Context) error }); ok", func() {
		p.CheckErr(p.B("hook.AfterPrepare"+pN+"Payload", "ctx"))
	})
}

func (p *Emitter) GenOperationUnmarshalNDR(ctx *emit.Env, op *midl.Operation, dir int) {

	fn := "UnmarshalNDR"
	if dir == emit.InParam {
		fn += "Request"
	} else if dir == emit.OutParam {
		fn += "Response"
	}

	p.P()
	p.Block("func", "(o *"+p.OpName(ctx, op, emit.AnyParam)+")", fn, "(ctx context.Context, w ndr.Reader)", "error", func() {

		strct := &midl.Type{
			Kind: midl.TypeAttribute,
			Attrs: &midl.TypeAttr{
				Alias: emit.Escape(p.OpName(ctx, op, dir)),
			},
			Elem: &midl.Type{
				Kind: midl.TypeStruct, Struct: &midl.Struct{}},
		}

		pp := p.NewTypeGenerator(ctx, strct)

		for _, param := range p.OperationParams(ctx, op) {

			if !p.IsDir(ctx, param.Attrs.Direction, dir) || param.IsHandle() {
				continue
			}

			field := param.Field()

			scopes := desc.Of(field)
			if scopes.Is(midl.TypePointer) && scopes.Layer().Pointer.IsWeak(true) &&
				// pointers to interfaces are always unique pointers.
				!scopes.Next().Is(midl.TypeInterface) {
				scopes = scopes.Next()
			}

			p.P("//", field.Name, pp.GenParamComment(ctx, param), pp.GenStructFieldComment(ctx, field, "ref"))
			p.P("{")
			pp.GenFieldUnmarshalNDR(ctx, field, scopes)
			if ndr.IsDeferrable(scopes) {
				p.CheckErr(p.B("w.ReadDeferred"))
			}
			p.P("}")

		}

		p.P("return nil")
	})

}

func (p *Emitter) GenParamComment(ctx *emit.Env, param *midl.Param) string {
	if param.Attrs == nil {
		return "{}"
	}
	attr := *param.Attrs
	attr.FieldAttr = nil
	return "{" + (&attr).String() + "}"
}

func (p *Emitter) GetOutputParamImplicitDependents(ctx *emit.Env, op *midl.Operation, dir int) []*midl.Param {

	if dir != emit.OutParam {
		if dir == emit.InParam {
			for _, param := range p.OperationParams(ctx, op) {
				if param.Attrs.Direction.Out && param.IsPipe() {
					return []*midl.Param{param}
				}
			}
		}
		return nil
	}

	deps, seen := []*midl.Param{}, make(map[string]bool)
	params := make(map[string]*midl.Param)

	for _, param := range p.OperationParams(ctx, op) {
		params[param.Name] = param
	}

	for _, param := range p.OperationParams(ctx, op) {
		if !param.Attrs.Direction.Out {
			continue
		}
		for _, expr := range [][]midl.Expr{
			param.Attrs.SizeIs,
			param.Attrs.LengthIs,
			param.Attrs.FirstIs,
			param.Attrs.LastIs,
			param.Attrs.MaxIs,
			param.Attrs.MinIs,
			[]midl.Expr{param.Attrs.SwitchIs},
		} {
			for _, ident := range p.GetExprIdents(ctx, expr...) {
				if p, ok := params[ident]; ok && p.Attrs.Direction.In && !p.Attrs.Direction.Out && !seen[p.Name] {
					deps, seen[p.Name] = append(deps, p), true
				}
			}
		}
	}

	return deps
}

func (p *Emitter) GenOperationStruct(ctx *emit.Env, op *midl.Operation, dir int) {

	p.P()
	p.P("//", p.OpName(ctx, op, dir), "structure", "represents", "the", emit.RPCName(op.Name),
		"operation", strings.ToLower(emit.ParamName(dir)))

	doc, _ := p.MSDN.GetPage(ctx, op.Name)

	mask := ctx.NullMask

	// generate go structure for the in/out/any parameters.
	p.Structure(p.OpName(ctx, op, dir), func() {

		if mask != "" {
			p.P()
			p.P("//", mask, "is used to carry information on null-valued primitive values.")
			p.P(p.NullMask(), mask)
			p.P()
		}

		if implicit := p.GetOutputParamImplicitDependents(ctx, op, dir); len(implicit) > 0 {
			for _, param := range implicit {
				p.P("//", "XXX:", param.Name, "is an implicit input depedency for output parameters")
				p.NewParamGenerator(ctx, param.Type).GenStructField(ctx, param.Field())
			}
			p.P()
		}

		for _, param := range p.OperationParams(ctx, op) {
			if !p.IsDir(ctx, param.Attrs.Direction, dir) {
				continue
			}
			if param.IsHandle() {
				// skip handle.
				continue
			}
			if dir != emit.AnyParam && emit.GoParamName(ctx, param) == "_" {
				// skip reserved.
				continue
			}
			if dir != emit.AnyParam {
				// gen docstring.
				if doc, ok := doc.GetObjectSection(param.Name); ok {
					p.GenComment(ctx, doc.Documentation)
				} else if param.Name == p.ReturnValue() {
					p.P("//", p.ReturnValue()+":", "The", op.Name, "return value.")
				}
				if param.Attrs.DocString != "" {
					p.P("//")
					p.GenComment(ctx, []string{param.Attrs.DocString})
				}

			}
			// generate structure field.
			p.NewParamGenerator(ctx, param.Type).GenStructField(ctx, param.Field())
		}

	})

	if dir == emit.AnyParam {
		p.GenOpNum(ctx, op, dir)
		p.GenOpName(ctx, op, dir)
		return
	}

	p.GenOperationToOp(ctx, op, dir, mask != "")
	p.GenOperationFromOp(ctx, op, dir, mask != "")

	dirN := emit.ParamName(dir)

	p.Block("func", "(o *"+p.OpName(ctx, op, dir)+")", "MarshalNDR", "(ctx context.Context, w ndr.Writer)", "error", func() {
		p.P("return", "o."+p.XXX()+"ToOp(ctx, nil).MarshalNDR"+dirN+"(ctx, w)")
	})

	p.Block("func", "(o *"+p.OpName(ctx, op, dir)+")", "UnmarshalNDR", "(ctx context.Context, r ndr.Reader)", "error", func() {
		p.P("_o", ":=", p.Amp(p.OpName(ctx, op, emit.AnyParam))+"{}")
		p.CheckErr("_o.UnmarshalNDR" + dirN + "(ctx, r)")
		p.P("o." + p.XXX() + "FromOp(ctx, _o)")
		p.P("return", "nil")
	})

	if dir == emit.InParam {
		p.GenMakeResponse(ctx, op, dir)
		p.GenOpNum(ctx, op, dir)
		p.GenOpName(ctx, op, dir)
	}
}

func (p *Emitter) GenOperationNullMask(ctx *emit.Env, op *midl.Operation) string {

	var nullable []*midl.Param

	for _, param := range p.OperationParams(ctx, op) {
		if emit.IsIgnoredPointer(ctx, param.Field()) {
			nullable = append(nullable, param)
		}
	}

	mask := p.MethodName(ctx, op) + p.NullMask()

	if len(nullable) == 0 {
		return ""
	}

	p.P()
	p.P("type", mask, "ndr.NullMask")

	p.P("var", "(")
	in, out := []string{"0"}, []string{"0"}
	for i, param := range nullable {
		n := mask + emit.GoFieldName(ctx, param.Field())
		p.P(n, mask, "=", "1", "<<", i /* cannot use position here, using i */)

		if param.Attrs.Direction.In {
			in = append(in, n)
		}

		if param.Attrs.Direction.Out {
			out = append(out, n)
		}
	}

	p.P()
	p.P(mask+p.DirName(ctx, emit.InParam)+"All", mask, "=", strings.Join(in, " | "))
	p.P(mask+p.DirName(ctx, emit.OutParam)+"All", mask, "=", strings.Join(out, " | "))

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

func (p *Emitter) GenMakeResponse(ctx *emit.Env, op *midl.Operation, dir int) {
	implicit := p.GetOutputParamImplicitDependents(ctx, op, emit.OutParam)
	p.P()
	p.P("//", "Make"+p.OpName(ctx, op, dir), "build a response structure from the given request structure.")
	p.Block("func", "(o *"+p.OpName(ctx, op, dir)+")", "MakeResponse()", "*"+p.OpName(ctx, op, emit.OutParam), func() {
		if len(implicit) == 0 {
			p.P("return", "&"+p.OpName(ctx, op, emit.OutParam)+"{}")
			return
		}
		p.If("o == nil", func() {
			p.P("return", "&"+p.OpName(ctx, op, emit.OutParam)+"{}")
		})
		p.Block("return", "&"+p.OpName(ctx, op, emit.OutParam), func() {
			for _, param := range implicit {
				n := emit.GoParamName(ctx, param)
				if n == "_" {
					continue
				}
				p.P(n+":", "o."+n+",")
			}
		})
	})
}

func (p *Emitter) GenOpNum(ctx *emit.Env, op *midl.Operation, dir int) {
	p.P()
	p.P("//", "OpNum", "returns", "the", "operation", "number", "of", op.Name, "operation.")
	p.P("func", "(o *"+p.OpName(ctx, op, dir)+")", "OpNum()", "int", "{", "return", op.OpNum, "}")
}

func (p *Emitter) GenOpName(ctx *emit.Env, op *midl.Operation, dir int) {
	p.P()
	p.P("//", "OpName", "returns", "the", "operation", "name", "of", op.Name, "operation.")
	p.P("func", "(o *"+p.OpName(ctx, op, dir)+")", "OpName()", "string", "{",
		"return", p.Q("/"+ctx.Interface.Name+"/"+ctx.Interface.Attrs.Version.String()+"/"+op.Name),
		"}")
}

func (p *Emitter) GenOperationToOp(ctx *emit.Env, op *midl.Operation, dir int, mask bool) {

	p.P()
	p.P("func", "(o *"+p.OpName(ctx, op, dir)+")", p.XXX()+"ToOp(ctx context.Context, op *"+p.OpName(ctx, op, emit.AnyParam), ")", "*"+p.OpName(ctx, op, emit.AnyParam), "{")
	p.If("op == nil", func() {
		p.P("op", "=", "&"+p.OpName(ctx, op, emit.AnyParam)+"{}")
	})
	p.If("o == nil", func() {
		p.P("return", "op")
	})

	if implicit := p.GetOutputParamImplicitDependents(ctx, op, dir); len(implicit) > 0 {
		p.P("//", "XXX:", "implicit input dependencies for output parameters")
		for _, param := range implicit {
			n := emit.GoParamName(ctx, param)
			if n == "_" {
				continue
			}
			f := &midl.Field{
				Name:  param.Name,
				Type:  param.Type,
				Attrs: param.Attrs.FieldAttr,
			}

			// if op.(n) is zero value, you can set it.
			p.If("op."+n, "==", p.GoTypeZeroValue(ctx, f.Type.Attrs, f, desc.Of(f)), func() {
				p.P("op."+n, "=", p.O(n))
			})
		}
		p.P()
	}

	for _, param := range p.OperationParams(ctx, op) {
		if !p.IsDir(ctx, param.Attrs.Direction, dir) || param.IsHandle() {
			continue
		}
		n := emit.GoParamName(ctx, param)
		if n == "_" {
			continue
		}
		p.P("op."+n, "=", p.O(n))

	}

	if mask {
		p.P("op."+p.NullMask(), "=", p.O(p.NullMask()))
	}

	p.P("return", "op")
	p.P("}")
}

func (p *Emitter) GenOperationFromOp(ctx *emit.Env, op *midl.Operation, dir int, mask bool) {
	p.P()
	p.P("func", "(o *"+p.OpName(ctx, op, dir)+")", p.XXX()+"FromOp(ctx context.Context, op *"+p.OpName(ctx, op, emit.AnyParam), ")", "{")
	p.If("o == nil", func() {
		p.P("return")
	})
	if implicit := p.GetOutputParamImplicitDependents(ctx, op, dir); len(implicit) > 0 {
		p.P("//", "XXX:", "implicit input dependencies for output parameters")
		for _, param := range implicit {
			n := emit.GoParamName(ctx, param)
			if n == "_" {
				continue
			}
			p.P(p.O(n), "=", "op."+n)
		}
		p.P()
	}

	for _, param := range p.OperationParams(ctx, op) {
		if !p.IsDir(ctx, param.Attrs.Direction, dir) || param.IsHandle() {
			continue
		}
		n := emit.GoParamName(ctx, param)
		if n == "_" {
			continue
		}
		p.P(p.O(n), "=", "op."+n)
	}

	if mask {
		mask := p.NullMask(p.MethodName(ctx, op))
		p.P(p.O(p.NullMask()), "=", p.B(mask, "op."+p.NullMask()), "&", mask+p.DirName(ctx, dir)+"All")
	}

	p.P("}")
}
