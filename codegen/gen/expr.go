package gen

import (
	"github.com/oiweiwei/midl-gen-go/midl"
	"github.com/oiweiwei/midl-gen-go/midl/expr"
)

// goBitNot renders ~x the way the generated code needs it: the operand is
// always a 32-bit quantity.
func goBitNot(operand string) string { return "(^uint32(" + operand + "))" }

// goExprFormat returns the Go rendering of expressions. ident renders a
// reference to a name (nil keeps the name).
func goExprFormat(ident func(string) string) expr.Format {
	return expr.Format{Ident: ident, BitNot: goBitNot}
}

// FormatExpr renders the expression as Go; identifiers are rendered as plain
// names.
func (p *Generator) FormatExpr(ctx *Env, e midl.Expr) string {
	return e.Format(goExprFormat(nil))
}

// FormatExprO renders the expression as Go; identifiers are the fields of the
// receiver ("o.Field"), named according to the namer of the context.
func (p *Generator) FormatExprO(ctx *Env, e midl.Expr) string {
	return e.Format(goExprFormat(p.EOWithCtx(ctx)))
}

// FormatConstExpr renders the value of an IDL constant as Go. Identifiers are
// other constants; when hexSize is positive, integer literals are rendered as
// hexadecimal of that many bytes.
func (p *Generator) FormatConstExpr(ctx *Env, e midl.Expr, hexSize int) string {
	f := goExprFormat(func(n string) string { return GoName(ctx, n) })
	if hexSize > 0 {
		f.Uint = GoHex(hexSize)
	}
	return e.Format(f)
}
