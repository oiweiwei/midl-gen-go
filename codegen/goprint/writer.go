package goprint

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

// Writer writes Go source lines into a buffer. Indentation is not its concern:
// the result is meant to be passed through gofmt (see Assemble).
//
// Expressions are strings; the helpers Call, Quote, Amp and Deref build them
// without the caller having to remember how to join or escape.
type Writer struct {
	buf *bytes.Buffer
}

// NewWriter returns a writer appending to buf.
func NewWriter(buf *bytes.Buffer) *Writer { return &Writer{buf: buf} }

// Line writes the parts separated by a space, and a newline.
func (w *Writer) Line(parts ...string) {
	w.buf.WriteString(strings.Join(parts, " "))
	w.buf.WriteByte('\n')
}

// Blank writes an empty line.
func (w *Writer) Blank() { w.buf.WriteByte('\n') }

// Comment writes the text as a // comment, one comment line per line of text.
func (w *Writer) Comment(text string) {
	for _, line := range strings.Split(text, "\n") {
		if line == "" {
			w.Line("//")
		} else {
			w.Line("//", line)
		}
	}
}

// Block writes "header {", the body and "}". A nil body gives an empty block.
func (w *Writer) Block(header string, body func()) {
	w.Line(header, "{")
	if body != nil {
		body()
	}
	w.Line("}")
}

// Composite writes "header {", the body and "}" followed by the tail, for a
// composite literal that is part of a larger statement: Composite("return &T",
// body, ", nil") is "return &T { body }, nil".
func (w *Writer) Composite(header string, body func(), tail string) {
	w.Line(header, "{")
	run(body)
	w.Line("}" + tail)
}

// Branch is one arm of a conditional, see Cond.
type Branch struct {
	kind byte // 'i' if, 'e' else if, 'x' else
	cond string
	body func()
}

// If is the first arm of a conditional.
func If(cond string, body func()) Branch { return Branch{'i', cond, body} }

// ElseIf is an "else if" arm; it must follow an If or an ElseIf.
func ElseIf(cond string, body func()) Branch { return Branch{'e', cond, body} }

// Else is the final arm.
func Else(body func()) Branch { return Branch{'x', "", body} }

// Cond writes an if / else if / else chain. It panics on a chain that does not
// start with If or in which Else is not last, which would be invalid Go.
func (w *Writer) Cond(first Branch, rest ...Branch) {
	if first.kind != 'i' {
		panic("goprint: a conditional must start with If")
	}
	w.Line("if", first.cond, "{")
	run(first.body)
	for i, b := range rest {
		switch b.kind {
		case 'e':
			w.Line("} else if", b.cond, "{")
		case 'x':
			if i != len(rest)-1 {
				panic("goprint: Else must be the last arm")
			}
			w.Line("} else {")
		default:
			panic("goprint: only ElseIf and Else may follow If")
		}
		run(b.body)
	}
	w.Line("}")
}

// If writes a conditional without else arms.
func (w *Writer) If(cond string, body func()) { w.Cond(If(cond, body)) }

func run(f func()) {
	if f != nil {
		f()
	}
}

// For writes "for clause { body }"; an empty clause is an infinite loop.
func (w *Writer) For(clause string, body func()) {
	if clause == "" {
		w.Block("for", body)
		return
	}
	w.Block("for "+clause, body)
}

// Switch writes "switch tag { body }"; the body writes Case and Default lines
// and the statements that follow them. An empty tag is a tagless switch.
func (w *Writer) Switch(tag string, body func()) {
	if tag == "" {
		w.Block("switch", body)
		return
	}
	w.Block("switch "+tag, body)
}

// Case writes the "case a, b:" line of a switch.
func (w *Writer) Case(exprs ...string) { w.Line("case " + strings.Join(exprs, ", ") + ":") }

// Default writes the "default:" line of a switch.
func (w *Writer) Default() { w.Line("default:") }

// Return writes a return statement.
func (w *Writer) Return(values ...string) {
	w.Line(append([]string{"return"}, joinComma(values))...)
}

func joinComma(values []string) string { return strings.Join(values, ", ") }

// Param is a parameter, a result or a receiver. An empty Name is an unnamed
// parameter.
type Param struct{ Name, Type string }

// Named returns a parameter with a name.
func Named(name, typ string) Param { return Param{Name: name, Type: typ} }

// Unnamed returns a parameter without a name, e.g. a result.
func Unnamed(typ string) Param { return Param{Type: typ} }

func (p Param) String() string {
	if p.Name == "" {
		return p.Type
	}
	return p.Name + " " + p.Type
}

// Func describes a function or a method.
type Func struct {
	// Recv is the receiver of a method; nil for a function.
	Recv *Param
	// Name is the name of the function; empty for a function literal.
	Name    string
	Params  []Param
	Results []Param
}

// Signature renders the declaration up to, not including, the opening brace.
func (f Func) Signature() string {
	var b strings.Builder
	b.WriteString("func")
	if f.Recv != nil {
		b.WriteString(" (" + f.Recv.String() + ")")
	}
	if f.Name != "" {
		b.WriteString(" " + f.Name)
	}
	b.WriteString("(" + params(f.Params) + ")")
	switch {
	case len(f.Results) == 1 && f.Results[0].Name == "":
		b.WriteString(" " + f.Results[0].Type)
	case len(f.Results) > 0:
		b.WriteString(" (" + params(f.Results) + ")")
	}
	return b.String()
}

func params(ps []Param) string {
	ss := make([]string, len(ps))
	for i, p := range ps {
		ss[i] = p.String()
	}
	return strings.Join(ss, ", ")
}

// Func writes a function declaration with the body.
func (w *Writer) Func(f Func, body func()) { w.Block(f.Signature(), body) }

// Struct writes a type declaration of a struct; the body writes the fields.
func (w *Writer) Struct(name string, body func()) { w.Block("type "+name+" struct", body) }

// Interface writes a type declaration of an interface.
func (w *Writer) Interface(name string, body func()) { w.Block("type "+name+" interface", body) }

// Field writes a struct field; an empty name is an embedded field, an empty
// tag is no tag.
func (w *Writer) Field(name, typ, tag string) {
	parts := []string{}
	if name != "" {
		parts = append(parts, name)
	}
	parts = append(parts, typ)
	if tag != "" {
		parts = append(parts, "`"+tag+"`")
	}
	w.Line(parts...)
}

// Const writes a constant declaration; typ may be empty.
func (w *Writer) Const(name, typ, value string) { w.decl("const", name, typ, value) }

// Var writes a variable declaration; typ or value may be empty.
func (w *Writer) Var(name, typ, value string) { w.decl("var", name, typ, value) }

func (w *Writer) decl(kw, name, typ, value string) {
	parts := []string{kw, name}
	if typ != "" {
		parts = append(parts, typ)
	}
	if value != "" {
		parts = append(parts, "=", value)
	}
	w.Line(parts...)
}

// Tokens joins the parts with a space, formatting each as fmt.Print does: a
// line of Go written as a list of tokens, e.g. Tokens("if", n, "> 0") is
// "if 3 > 0" for n = 3.
func Tokens(parts ...any) string {
	return strings.TrimSuffix(fmt.Sprintln(parts...), "\n")
}

// Call returns the call expression fn(args...).
func Call(fn string, args ...string) string { return fn + "(" + joinComma(args) + ")" }

// Quote returns the Go string literal for s, with all special characters escaped.
func Quote(s string) string { return strconv.Quote(s) }

// Amp returns &x.
func Amp(x string) string { return "&" + x }

// Deref returns *x.
func Deref(x string) string { return "*" + x }
