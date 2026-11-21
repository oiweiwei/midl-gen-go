package ndr

import (
	"strings"

	"github.com/oiweiwei/midl-gen-go/codegen/golang/emit"
	"github.com/oiweiwei/midl-gen-go/codegen/goprint"
	"github.com/oiweiwei/midl-gen-go/desc"
	"github.com/oiweiwei/midl-gen-go/midl"
	"github.com/oiweiwei/midl-gen-go/midl/expr"
)

// LookupExprField returns the field of the current struct the expression refers to
// (its first identifier), or nil.
func (p *TypeGenerator) LookupExprField(ctx *emit.Env, e midl.Expr) *midl.Field {

	if p.Struct() == nil {
		return nil
	}

	name, ok := e.FirstIdent()
	if !ok {
		return nil
	}

	for _, field := range p.Struct().Fields {
		if name == field.Name {
			return field
		}
	}

	return nil
}

// GenExpr renders the expression as a Go expression for the field and returns
// it. A conditional on a non-reference, or any expression of a boolean field,
// needs statements: they are emitted to a temporary variable which is returned.
func (p *TypeGenerator) GenExpr(ctx *emit.Env, e midl.Expr, field *midl.Field, typ string, n ...string) string {

	isBool := false
	if field != nil {
		isBool = desc.Of(field).IsBool()
	}

	root := e.Root()

	if field == nil || root == nil || (!isBool && (root.Kind != expr.NodeCond || root.X.IsRef())) {

		out := p.FormatExprO(ctx, e)
		if out == "o._" {
			return "0"
		}
		return out
	}

	varName := "_expr" + field.Name + strings.Join(n, "")

	if typ == "" {
		typ = p.GoFieldTypeName(ctx, p.Layer(), field)
	}

	p.P(varName, ":=", typ+"(0)")

	var cond, lval, rval midl.Expr
	if isBool {
		cond, lval, rval = expr.Ident(field.Name), expr.Int64(1), expr.Int64(0)
	} else {
		cond, lval, rval = expr.FromNode(root.X), expr.FromNode(root.Y), expr.FromNode(root.Z)
	}

	chk := ""
	if !isBool && !root.X.IsBool() {
		chk = "!= 0"
	}

	p.IfElse(goprint.Tokens(p.FormatExprO(ctx, cond), chk), func() {
		p.P(varName, "=", p.B(typ, p.FormatExprO(ctx, lval)))
	}, func() {
		p.P(varName, "=", p.B(typ, p.FormatExprO(ctx, rval)))
	})

	return varName
}

// GetMinVarValue reports whether the size expression can become negative when
// its only identifier is zero, e.g. size_is(n - 1). It returns the identifier
// and the value it must be at least.
func (p *TypeGenerator) GetMinVarValue(ctx *emit.Env, sizeExpr midl.Expr) (midl.Expr, int64, bool) {
	idents := sizeExpr.Idents()
	if len(idents) != 1 {
		return midl.Expr{}, 0, false
	}
	atZero, err := sizeExpr.Eval(expr.Always(expr.Int64(0)))
	if err != nil {
		return midl.Expr{}, 0, false
	}
	val, ok := atZero.Int64()
	if !ok || val >= 0 {
		return midl.Expr{}, 0, false
	}
	return expr.Ident(idents[0]), -val, true
}
