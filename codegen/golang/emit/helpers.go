package emit

import (
	"fmt"
	"strings"

	"github.com/oiweiwei/midl-gen-go/codegen/goprint"
	"github.com/oiweiwei/midl-gen-go/midl"
)

// HasNullMaskBit reports whether the field has a null-mask bit; must match GenNullMask ([ref] pointers have none).
func HasNullMaskBit(ctx *Env, field *midl.Field) bool {
	return ctx.NullMask != "" && IsIgnoredPointer(ctx, field) && GoFieldName(ctx, field) != "_"
}

var (
	Obj = "o"
)

func (p *Emitter) GenRange(ctx *Env, n string, i, upto int, fn func(*Env, string)) {
	if i >= upto {
		fn(ctx, n)
		return
	}
	idx := fmt.Sprintf("i%d", i+1)
	p.For(goprint.Tokens(idx, ":= range", n), func() {
		p.GenRange(ctx, p.IVar(n, idx), i+1, upto, fn)
	})
}

// CheckErr1 returns the function that emits a call with the check of its error:
// CheckErr(call) writes "if err := call; err != nil { return err }".
var CheckErr1 = func(p *Emitter) func(...interface{}) {
	return func(args ...interface{}) {
		p.If("err := "+goprint.Tokens(args...)+"; err != nil", func() {
			p.P("return", "err")
		})
	}
}

// CheckErr2 is CheckErr without the check: it writes the call as it is.
var CheckErr2 = func(p *Emitter) func(...interface{}) {
	return func(args ...interface{}) {
		p.P(args...)
	}
}

// Block writes "header {", the body and "}".
func (p *Emitter) Block(header string, body func()) { p.W().Block(header, body) }

// If writes "if cond { body }".
func (p *Emitter) If(cond string, body func()) { p.W().If(cond, body) }

// IfElse writes "if cond { then } else { otherwise }".
func (p *Emitter) IfElse(cond string, then, otherwise func()) {
	p.W().Cond(goprint.If(cond, then), goprint.Else(otherwise))
}

// For writes "for clause { body }"; an empty clause is an infinite loop.
func (p *Emitter) For(clause string, body func()) { p.W().For(clause, body) }

// Switch writes "switch tag { body }"; the body writes the case lines.
func (p *Emitter) Switch(tag string, body func()) { p.W().Switch(tag, body) }

// Method writes the declaration of a method: "func (recv) decl { body }", where recv
// is the receiver as written ("o *T") and decl the name, parameters and results
// ("MarshalNDR(ctx context.Context, w ndr.Writer) error").
func (p *Emitter) Method(recv, decl string, body func()) {
	p.Block("func ("+recv+") "+decl, body)
}

// Function writes the declaration of a function: "func decl { body }".
func (p *Emitter) Function(decl string, body func()) { p.Block("func "+decl, body) }

// Structure writes the declaration of a struct type; the body writes the fields.
func (p *Emitter) Structure(name string, body func()) { p.Block("type "+name+" struct", body) }

// Interface writes the declaration of an interface type; the body writes the methods.
func (p *Emitter) Interface(name string, body func()) { p.Block("type "+name+" interface", body) }

// Composite writes "header {", the body and "}" followed by the tail, for a
// composite literal that is part of a larger statement.
func (p *Emitter) Composite(header string, body func(), tail string) {
	p.W().Composite(header, body, tail)
}

// Var function generates the variable name.
func (p *Emitter) Var(n string, idx ...interface{}) string {
	if len(idx) > 0 {
		return fmt.Sprintf("%s%v", n, idx[0])
	}
	return n
}

// Q returns the Go string literal for n; quotes, backslashes and control characters are escaped.
func (p *Emitter) Q(n string) string {
	return goprint.Quote(n)
}

// W returns a typed writer appending to the file being generated. It must be
// called again after the file changes (Reset): it is not meant to be kept.
func (p *Emitter) W() *goprint.Writer {
	return goprint.NewWriter(p.out.Out)
}

func (p *Emitter) B(pre interface{}, s ...interface{}) string {

	ss := make([]string, len(s))
	for i := range s {
		ss[i] = fmt.Sprintf("%v", s[i])
	}

	return fmt.Sprintf("%v(%s)", pre, strings.Join(ss, ","))
}

func (p *Emitter) Amp(v interface{}) string {
	return "&" + fmt.Sprintf("%v", v)
}

func (p *Emitter) Ptr(ptr interface{}) string {
	return fmt.Sprintf("*%v", ptr)
}

func (p *Emitter) BPtr(ptr interface{}) string {
	return fmt.Sprintf("(*%v)", ptr)
}

var (
	varReplacer = strings.NewReplacer("[", "_", "]", "", Obj+".", "")
)

func (p *Emitter) ToVar(s string) string {
	return varReplacer.Replace(s)
}

func (p *Emitter) IVar(n string, idxs ...interface{}) string {
	for _, idx := range idxs {
		n += fmt.Sprintf("[%v]", idx)
	}
	return n
}

// EO function generates the O with namer.
func (p *Emitter) EO(ctx *Env, n string) string {
	return p.O(GoMemberName(ctx, n))
}

func (p *Emitter) EOWithCtx(ctx *Env) func(string) string {
	return func(n string) string {
		return p.EO(ctx, n)
	}
}

// O function generates the "o."
func (p *Emitter) O(n string) string {
	return Obj + "." + n
}

var (
	ZeroLen = "%ZERO%"
)

func (p *Emitter) Len(n string) string {
	if strings.HasPrefix(n, ZeroLen) {
		return "0"
	}
	return "len(" + n + ")"
}

// R function generates the repeat with start end sequence.
func (p *Emitter) R(start, s, end string, count int) string {
	return strings.Repeat(start, count) + s + strings.Repeat(end, count)
}

func (p *Emitter) BufVar(name string) string {
	return "_" + p.ToVar(name) + "_buf"
}

func (p *Emitter) NullMask(name ...string) string {
	return strings.Join(name, "") + "NullMask"
}

func (p *Emitter) VarName(ctx *Env, index ...interface{}) string {
	if ctx.VarName != "" {
		return p.IVar(ctx.VarName, index...)
	}
	return ""
}
