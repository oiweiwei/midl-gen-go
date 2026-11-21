package midl

// parser.go contains the IDL parser implementation.

import (
	"fmt"
	"strings"
)

func init() {
	// report the expected tokens on syntax errors.
	RPCErrorVerbose = true
}

// Import loads the imported file into the parser session.
func (p *Parser) Import(name string) error {
	_, err := p.sess.Load(name)
	return err
}

// Parse parses the IDL source. Type and constant references that are not
// local to the source are resolved against the files already loaded in the
// session.
func (s *Session) Parse(src string) (*File, error) {

	var (
		p = NewParser(s, src)
	)

	if RPCParse(p); p.errLex != nil {
		return nil, p.errLex
	}

	if p.result == nil {
		return &File{}, nil
	}

	// save exportSyms to the file for further reference.
	p.result.exportSyms = p.exportSyms
	p.result.sess = s

	// resolve defered type rederences.
	for _, ref := range p.refs {
		typ, ok := p.LookupType(ref.Name)
		if !ok {
			name := strings.Replace(ref.Name, "_struct_", "_union_", 1)
			if typ, ok = p.LookupType(name); !ok {
				return nil, fmt.Errorf("unable to resolve type %s", ref.Name)
			}
		}

		*ref = *typ
	}

	return p.result, nil
}

// setResult ...
func setResult(p interface{}, f *File) {
	p.(*Parser).result = f
}

// exportSyms ...
func exportSyms(p interface{}, n string, e *Export) {
	pr := p.(*Parser)
	pr.curPos++
	e.Position = pr.curPos
	pr.exportSyms[n] = e
}

// tokName ...
func tokName(p interface{}, c int) (string, bool) {
	ret, ok := p.(*Parser).TokName(c)
	if ok {
		p.(*Parser).errLex = nil
	}
	return ret, ok
}

// pushLex ...
func pushLex(p interface{}, s string) {
	p.(*Parser).Push(s)
}

// pushRef ...
func pushRef(p interface{}, typ *Type) *Type {
	p.(*Parser).Defer(typ)
	return typ
}

// lookupType ...
func lookupType(p interface{}, n string) (*Type, bool) {
	typ, ok := p.(*Parser).LookupType(n)
	return typ, ok
}

func resolvePointers(p interface{}, pointerDefault PointerType) {
}

// lookupConst ...
func lookupConst(p interface{}, n string) (*Const, bool) {
	c, ok := p.(*Parser).LookupConst(n)
	return c, ok
}

// storeConst ...
func storeConst(p interface{}, n string, val Expr, typ ...Kind) {
	p.(*Parser).Store(n, val, typ...)
}

type Parser struct {
	// The lexer entity.
	*Lexer
	// The session used to resolve external references.
	sess *Session
	// The result.
	result *File
	// The export symbols for the resulting file.
	exportSyms map[string]*Export
	// The list of delayed references.
	refs []*Type
	// The current exported symbol position.
	curPos int
}

// NewParser function returns the parser entity.
func NewParser(sess *Session, src string) *Parser {
	return &Parser{
		sess:       sess,
		Lexer:      NewLexer(src),
		exportSyms: make(map[string]*Export),
	}
}

// Defer function defers the type resolution until the parse
// end.
func (p *Parser) Defer(ref *Type) {
	p.refs = append(p.refs, ref)
}

// LookupType ...
func (p *Parser) LookupType(n string) (*Type, bool) {

	var (
		e  *Export
		ok bool
	)

	if e, ok = p.exportSyms[n]; !ok {
		for _, f := range p.sess.Files() {
			if e, ok = f.exportSyms[n]; !ok {
				continue
			}
			break
		}
	}

	if !ok {
		return nil, false
	} else if e.Type == nil {
		p.sess.log.Warn("not a type reference", "name", n)
		return nil, false
	}

	return e.Type, true
}

// LookupConst ...
func (p *Parser) LookupConst(n string) (*Const, bool) {

	var (
		e  *Export
		f  *File
		ok bool
	)

	if e, ok = p.exportSyms[n]; !ok {
		for _, f = range p.sess.Files() {
			if e, ok = f.exportSyms[n]; !ok {
				continue
			}
			break
		}
	}
	if !ok {
		return nil, false
	} else if e.Const == nil {
		p.sess.log.Warn("not a const reference", "name", n)
		return nil, false
	}

	return e.Const, true
}

// LookupExpr ...
func (p *Parser) LookupExpr(n string) (Expr, bool) {

	c, ok := p.LookupConst(n)
	if !ok {
		return Expr{}, false
	}

	return c.Value, true
}

// Store ...
func (p *Parser) Store(n string, val Expr, typ ...Kind) {
	e := &Export{
		Name: n,
		Const: &Const{
			Name:  n,
			Value: val,
		},
	}
	if len(typ) > 0 {
		e.Const.Type = typ[0]
	}
	p.exportSyms[n] = e
}
