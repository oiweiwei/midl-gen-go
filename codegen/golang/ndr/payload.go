package ndr

import (
	"github.com/oiweiwei/midl-gen-go/codegen/golang/emit"
	"github.com/oiweiwei/midl-gen-go/desc"
	"github.com/oiweiwei/midl-gen-go/midl"
	"github.com/oiweiwei/midl-gen-go/midl/expr"
)

func (p *TypeGenerator) GenStructPreparePayloadAfterHook(ctx *emit.Env) {
	p.CheckErr(p.B("ndr.AfterPreparePayload", "ctx", "o"))
}

func (p *TypeGenerator) GenStructPreparePayloadBeforeHook(ctx *emit.Env) {
	p.CheckErr(p.B("ndr.BeforePreparePayload", "ctx", "o"))
}

func (p *TypeGenerator) GenStructPreparePayloadFields(ctx *emit.Env, fields []*midl.Field) {

	ranged := make([]*midl.Field, 0)
	defaulted := make([]*midl.Field, 0)

	for _, field := range fields {

		defaulted = append(defaulted, field)

		skip := p.GoFieldName(ctx, field) == "_"

		if field.Attrs.Range != nil && !skip {
			ranged = append(ranged, field)
		}

		if field.Attrs.SizeAttr() == nil && field.Attrs.LengthIs == nil {
			continue
		}

		scopes := desc.Of(field)
		for ; scopes != nil; scopes = scopes.Next() {
			if scopes.Is(midl.TypeArray) {
				break
			}
		}

		if scopes := scopes; field.Attrs.SizeAttr() != nil {
			for ; scopes != nil; scopes = scopes.Next() {
				if sz := scopes.Dim().Size(); !sz.Empty() {
					if !skip {
						p.GenStructPreparePayloadSizeExpr(ctx, field, scopes, sz.Is())
					}
					// check minimum value borders.
					if to, val, ok := p.GetMinVarValue(ctx, sz.Is()); ok {
						p.If(p.FormatExprO(ctx, to), "<", val, func() {
							p.P(p.FormatExprO(ctx, to), "=", val)
						})
					}
					break
				}
			}
		}

		if scopes := scopes; field.Attrs.LengthIs != nil {
			for ; scopes != nil; scopes = scopes.Next() {
				if sz := scopes.Dim().LengthIs; !sz.IsEmpty() {
					if !skip {
						p.GenStructPreparePayloadSizeExpr(ctx, field, scopes, sz)
					}
					if to, val, ok := p.GetMinVarValue(ctx, sz); ok {
						p.If(p.FormatExprO(ctx, to), "<", val, func() {
							p.P(p.FormatExprO(ctx, to), "=", val)
						})
					}
					break
				}
			}
		}
	}

ranged_loop:
	for _, field := range ranged {

		rng := field.Attrs.Range
		fL := p.O(p.GoFieldName(ctx, field))
		fT := p.GoFieldTypeName(ctx, p.Layer(), field)

		// process range statement for array.
		for scopes := desc.Of(field); scopes != nil; scopes = scopes.Next() {
			if scopes.Is(midl.TypeArray) {
				p.If(p.Len(fL), ">", p.B("int", rng.Max), func() {
					p.P(`return fmt.Errorf("` + p.GoFieldName(ctx, field) + ` is out of range")`)
				})
				continue ranged_loop
			}
		}

		// process range statement for primitive type.
		if field.Type.Base().Kind.Signed() || rng.Min != 0 {
			p.If(fL, "<", p.B(fT, rng.Min), "||", fL, ">", p.B(fT, rng.Max), func() {
				p.P(`return fmt.Errorf("` + p.GoFieldName(ctx, field) + ` is out of range")`)
			})
		} else if rng.Min == 0 {
			p.If(fL, ">", p.B(fT, rng.Max), func() {
				p.P(`return fmt.Errorf("` + p.GoFieldName(ctx, field) + ` is out of range")`)
			})
		}
	}

	for _, field := range defaulted {
		if !field.DefaultValue.IsEmpty() && !field.Attrs.Ignore {
			p.If(p.EO(ctx, field.Name), "==", p.GoTypeZeroValue(ctx, p.Layer(), field, desc.Of(field)), func() {
				expr := field.DefaultValue
				p.P(p.EO(ctx, field.Name), "=", p.B(
					p.GoFieldTypeName(ctx, p.Layer(), field),
					p.GenExpr(ctx, expr, p.LookupExprField(ctx, expr), ""),
				))
			})
		}
	}
}

func (p *TypeGenerator) GenStructPreparePayload(ctx *emit.Env) {

	p.P("")
	p.P("func", "(o *"+p.GoTypeName+")", p.XXX()+"PreparePayload(ctx context.Context)", "error", "{")

	p.GenStructPreparePayloadBeforeHook(ctx)
	p.GenStructPreparePayloadFields(ctx, p.Struct().Fields)
	p.GenStructPreparePayloadAfterHook(ctx)

	p.P("return", "nil")
	p.P("}")
}

func (p *TypeGenerator) GenStructPreparePayloadSizeExpr(ctx *emit.Env, field *midl.Field, scopes *desc.Cursor, sz midl.Expr) {

	n := p.GoFieldName(ctx, field)
	lenIdent := expr.Ident(p.DataLen(ctx, field, desc.Of(field), p.O(n), ""))

	sol, ok := sz.Solve(lenIdent)
	if !ok {
		// the expression cannot be inverted: the size is its first identifier
		// set to the length as it is.
		name, found := sz.FirstIdent()
		if !found {
			p.P("//", "cannot evaluate expression", sz.String())
			return
		}
		sol = expr.Solution{Ident: name, Value: lenIdent}
	}

	target := sol.Ident
	for _, szField := range p.Struct().Fields {
		if szField.Name != target {
			continue
		}
		szN := emit.GoMemberName(ctx, szField.Name)
		szT := p.GoFieldTypeName(ctx, p.Layer(), szField)
		// ... if o.Data != nil && o.DataLen == 0 ...
		p.If(p.O(n), "!=", p.GoTypeZeroValue(ctx, p.Layer(), field, scopes), "&&", p.O(szN), "==", "0", func() {
			// TODO: handle complex expressions.
			p.P(p.O(szN), "=", p.B(szT, p.FormatExpr(ctx, sol.Value)))
			// check variable borders.
			if to, val, ok := p.GetMinVarValue(ctx, sol.Value); ok {
				p.If(p.FormatExpr(ctx, to), "<", val, func() {
					p.P(p.O(szN), "=", p.B(szT, "0"))
				})
			}
		})
	}
}
