package gen

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

func (p *Generator) GenRange(ctx *Env, n string, i, upto int, fn func(*Env, string)) {
	if i >= upto {
		fn(ctx, n)
		return
	}
	idx := fmt.Sprintf("i%d", i+1)
	p.Range(idx, n, func() {
		p.GenRange(ctx, p.IVar(n, idx), i+1, upto, fn)
	})
}

func (p *Generator) If(args ...interface{}) {
	p.Block(append([]interface{}{"if"}, args...)...)
}

var (
	CheckErr1 = func(p *Generator) func(...interface{}) {
		return func(args ...interface{}) {
			p.Block(append(append([]interface{}{"if", "err", ":="}, args...), ";", "err != nil", func() {
				p.P("return", "err")
			}))
		}
	}

	CheckErr2 = func(p *Generator) func(...interface{}) {
		return func(args ...interface{}) {
			p.P(args...)
		}
	}
)

func (p *Generator) Structure(args ...interface{}) {
	p.Block(append([]interface{}{"type", args[0], "struct"}, args[1:]...)...)
}

func (p *Generator) Range(args ...interface{}) {

	switch len(args) {
	case 0:
		panic("unknown range")
	case 1:
		p.Block("for", args[0])
	case 2:
		p.Block(append([]interface{}{"for", "range"}, args...)...)
	case 3:
		p.Block(append([]interface{}{"for", args[0], ":=", "range"}, args[1:]...)...)
	default:
		p.Block(append([]interface{}{"for", args[0], ",", args[1], ":=", "range"}, args[2:]...)...)
	}
}

func (p *Generator) ElseIf(args ...interface{}) []interface{} {
	return append([]interface{}{"else", "if"}, args...)
}

func (p *Generator) Else(args ...interface{}) []interface{} {
	return append([]interface{}{"else"}, args...)
}

func (p *Generator) Switch(args ...interface{}) {
	p.Block(append([]interface{}{"switch"}, args...)...)
}

func (p *Generator) Block(args ...interface{}) {

	var (
		printable []interface{}
	)

	for _, arg := range args {
		switch arg := arg.(type) {
		case func():
			if len(printable) > 0 {
				p.P(append(printable, "{")...)
			}
			printable = []interface{}{"}"}
			arg()
		case []interface{}:
			p.Block(append(printable, arg...)...)
			printable = nil
		default:
			printable = append(printable, arg)
		}
	}

	if len(printable) > 0 {
		p.P(printable...)
	}

}

// Var function generates the variable name.
func (p *Generator) Var(n string, idx ...interface{}) string {
	if len(idx) > 0 {
		return fmt.Sprintf("%s%v", n, idx[0])
	}
	return n
}

// Q returns the Go string literal for n; quotes, backslashes and control characters are escaped.
func (p *Generator) Q(n string) string {
	return goprint.Quote(n)
}

// W returns a typed writer appending to the file being generated. It must be
// called again after the file changes (Reset): it is not meant to be kept.
func (p *Generator) W() *goprint.Writer {
	return goprint.NewWriter(p.out.Out)
}

func (p *Generator) B(pre interface{}, s ...interface{}) string {

	ss := make([]string, len(s))
	for i := range s {
		ss[i] = fmt.Sprintf("%v", s[i])
	}

	return fmt.Sprintf("%v(%s)", pre, strings.Join(ss, ","))
}

func (p *Generator) Amp(v interface{}) string {
	return "&" + fmt.Sprintf("%v", v)
}

func (p *Generator) Ptr(ptr interface{}) string {
	return fmt.Sprintf("*%v", ptr)
}

func (p *Generator) BPtr(ptr interface{}) string {
	return fmt.Sprintf("(*%v)", ptr)
}

var (
	varReplacer = strings.NewReplacer("[", "_", "]", "", Obj+".", "")
)

func (p *Generator) ToVar(s string) string {
	return varReplacer.Replace(s)
}

func (p *Generator) IVar(n string, idxs ...interface{}) string {
	for _, idx := range idxs {
		n += fmt.Sprintf("[%v]", idx)
	}
	return n
}

// EO function generates the O with namer.
func (p *Generator) EO(ctx *Env, n string) string {
	return p.O(GoMemberName(ctx, n))
}

func (p *Generator) EOWithCtx(ctx *Env) func(string) string {
	return func(n string) string {
		return p.EO(ctx, n)
	}
}

// O function generates the "o."
func (p *Generator) O(n string) string {
	return Obj + "." + n
}

var (
	ZeroLen = "%ZERO%"
)

func (p *Generator) Len(n string) string {
	if strings.HasPrefix(n, ZeroLen) {
		return "0"
	}
	return "len(" + n + ")"
}

// R function generates the repeat with start end sequence.
func (p *Generator) R(start, s, end string, count int) string {
	return strings.Repeat(start, count) + s + strings.Repeat(end, count)
}

func (p *Generator) BufVar(name string) string {
	return "_" + p.ToVar(name) + "_buf"
}

func (p *Generator) NullMask(name ...string) string {
	return strings.Join(name, "") + "NullMask"
}

func (p *Generator) VarName(ctx *Env, index ...interface{}) string {
	if ctx.VarName != "" {
		return p.IVar(ctx.VarName, index...)
	}
	return ""
}
