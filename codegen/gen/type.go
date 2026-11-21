package gen

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/oiweiwei/midl-gen-go/desc"
	"github.com/oiweiwei/midl-gen-go/desc/ndr"
	"github.com/oiweiwei/midl-gen-go/midl"
	"github.com/oiweiwei/midl-gen-go/midl/expr"
	"github.com/oiweiwei/midl-gen-go/msdn/openspecs"
)

type TypeGenerator struct {
	*Generator
	*Scopes
	GoTypeName   string
	GoMethodName string
	Doc          *openspecs.Page
	OrigType     *midl.Type
}

func (p *Generator) NewTypeGenerator(ctx *Env, typ *midl.Type) *TypeGenerator {
	scopes := desc.OfType(typ)
	docs, ok := p.MSDN.GetPage(ctx, scopes.Alias())
	if !ok {
		docs, _ = p.MSDN.GetPage(ctx, scopes.Layer().Parent)
	}
	return &TypeGenerator{
		Generator:  p,
		Scopes:     scopes,
		GoTypeName: GoName(ctx, scopes.Alias()),
		Doc:        docs,
		OrigType:   typ,
	}
}

// OwnerName is the IDL name the members of the type are overridden by: the
// name of the typedef if there is one, else the tag.
func (p *TypeGenerator) OwnerName() string {
	if alias := p.Alias(); alias != "" {
		return alias
	}
	return p.Type().Tag
}

func (p *TypeGenerator) Skip() bool {
	return p.Scopes == nil || p.Layer().Alias == ""
}

func (p *TypeGenerator) GenPipeField(ctx *Env, typ *midl.Type) *midl.Field {
	return &midl.Field{
		Attrs: &midl.FieldAttr{NoSizeLimit: false},
		Type: &midl.Type{
			Kind:  midl.TypeArray,
			Array: &midl.Array{Bound: midl.ArrayBound{Upper: -1, Lower: 0}},
			Attrs: &midl.TypeAttr{},
			Elem:  typ.Elem,
		},
	}
}

func (p *TypeGenerator) GenPipe(ctx *Env) {

	p.P("//", p.GoTypeName, "type", "represents", RPCName(p.Alias()), "RPC", "pipe.")
	if p.Doc != nil {
		p.P("//")
		p.GenComment(ctx, p.Doc.Documentation)
	}
	if len(p.Layer().DocString) > 0 {
		p.P("//")
		p.GenCommentLine(ctx, p.Layer().DocString)
	}

	p.P("type", p.GoTypeName, "interface", "{")
	p.P("Pipe()", "<-chan", p.GoFieldTypeName(ctx, p.Layer(), p.GenPipeField(ctx, p.Type())))
	p.P("Send(v", p.GoFieldTypeName(ctx, p.Layer(), p.GenPipeField(ctx, p.Type()))+")")
	p.P("Append(v", p.GoFieldTypeName(ctx, p.Layer(), p.GenPipeField(ctx, p.Type()))+")")
	p.P("Recv()", p.B("", p.GoFieldTypeName(ctx, p.Layer(), p.GenPipeField(ctx, p.Type())), "bool"))
	p.P("Close()", "error")
	p.P("}")

	field := p.GenPipeField(ctx, p.Type())

	p.P()
	p.P("//", "Pipe", "provides", "a", "channel-based", "interface", "to", "send", "data", "to", "the", "RPC", "server.")
	p.P("//", "Note: pipe must be closed after use to avoid resource leaks.")
	p.P("type", p.XXX()+p.GoTypeName, "struct {")
	p.P("p", "chan", p.GoFieldTypeName(ctx, p.Layer(), field))
	p.P("recv", "[]", p.GoFieldTypeName(ctx, p.Layer(), field))
	p.P("}")

	p.P()
	p.P("//", "New"+p.XXX()+p.GoTypeName, "creates", "a", "new", p.GoTypeName, "instance.")
	p.P("func", "New"+p.GoTypeName+"()", p.GoTypeName, "{")
	p.P("return", "&"+p.XXX()+p.GoTypeName+"{")
	p.P("p: make(chan", p.GoFieldTypeName(ctx, p.Layer(), field), "),")
	p.P("}")
	p.P("}")

	p.P()
	p.P("func", "(o *"+p.XXX()+p.GoTypeName+")", "Pipe()", "<-chan", p.GoFieldTypeName(ctx, p.Layer(), field), "{")
	p.P("return", "o.p")
	p.P("}")

	p.P()
	p.P("//", "Send", "sends", "data", "to", "the", "RPC", "server", "via", "the", "pipe.")
	p.P("func", "(o *"+p.XXX()+p.GoTypeName+")", "Send(v", p.GoFieldTypeName(ctx, p.Layer(), field)+")", "{")
	p.P("if o != nil && o.p != nil {")
	p.P("o.p <- v")
	p.P("}")
	p.P("}")

	// append to pipe.
	p.P()
	p.P("//", "Append", "appends", "data", "to", "the", "internal", "buffer", "of", "the", "pipe.")
	p.P("func", "(o *"+p.XXX()+p.GoTypeName+")", "Append(v", p.GoFieldTypeName(ctx, p.Layer(), field)+")", "{")
	p.P("if o != nil {")
	p.P("o.recv =", "append(o.recv, v)")
	p.P("}")
	p.P("}")

	// receive first item of recv.
	p.P()
	p.P("//", "Recv", "receives", "data", "from", "the", "internal", "buffer", "of", "the", "pipe.")
	p.P("func", "(o *"+p.XXX()+p.GoTypeName+")", "Recv()", p.B("", p.GoFieldTypeName(ctx, p.Layer(), field), "bool"), "{")
	p.P("if o != nil {")
	p.P("if len(o.recv) > 0 {")
	p.P("v", ":=", "o.recv[0]")
	p.P("o.recv", "=", "o.recv[1:]")
	p.P("return", "v", ",", "true")
	p.P("}")
	p.P("}")
	p.P("return", p.GoTypeZeroValue(ctx, p.Layer(), field, desc.Of(field)), ",", "false")
	p.P("}")

	p.P("func", "(o *"+p.XXX()+p.GoTypeName+")", "Close()", "error", "{")
	p.If("o != nil && o.p != nil", func() {
		p.P("close(o.p)")
	})
	p.P("return", "nil")
	p.P("}")

}

func (p *TypeGenerator) GenEnum(ctx *Env) {

	p.P("//", p.GoTypeName, "type", "represents", RPCName(p.Alias()), "RPC", "enumeration.")
	if p.Doc != nil {
		p.P("//")
		p.GenComment(ctx, p.Doc.Documentation)
	}
	if len(p.Layer().DocString) > 0 {
		p.P("//")
		p.GenCommentLine(ctx, p.Layer().DocString)
	}

	if p.Enum().Is32 {
		// accomodate all possible values.
		p.P("type", p.GoTypeName, "uint32")
	} else {
		p.P("type", p.GoTypeName, p.EnumType())
	}
	p.P()
	p.P("var", "(")
	for _, enum := range p.Enum().Elems {
		if doc, ok := p.Doc.GetSection(enum.Value); ok {
			p.GenComment(ctx, doc.Documentation)
		}
		p.P(GoMergeNames(ctx, p.GoTypeName, GoMemberNameNoReserved(ctx, enum.Value)), p.GoTypeName, "=", strconv.Itoa(enum.ID))
	}
	p.P(")")
	p.P()
	p.Block("func", "(o "+p.GoTypeName+")", "String()", "string", func() {
		p.Block("switch o", func() {
			for _, enum := range p.Enum().Elems {
				n := GoMergeNames(ctx, p.GoTypeName, GoMemberNameNoReserved(ctx, enum.Value))
				p.P("case", n, ":")
				p.P("return", p.Q(n))
			}
		})
		p.P("return", p.Q("Invalid"))
	})
}

func (p *TypeGenerator) GenCUnion(ctx *Env) {

	p.Structure(p.GoTypeName, func() {
		for _, cases := range p.Union().Body {
			for _, field := range cases.Arms {
				p.GenStructField(ctx, field)
			}
		}
	})

	for _, cases := range p.Union().Body {
		p.GenSubTypes(ctx, cases.Arms)
	}
}

func (p *TypeGenerator) GenStructField(ctx *Env, field *midl.Field) {
	if field.IsHandle() {
		return
	}
	if doc, ok := p.Doc.GetSection(field.Name); ok {
		p.GenComment(ctx, doc.Documentation)
	}
	if len(field.Attrs.DocString) > 0 {
		p.GenCommentLine(ctx, field.Attrs.DocString)
	}
	notes := ""
	if !field.IsString() && p.GoFieldTypeName(ctx, p.Layer(), field) == "string" {
		// notes = `// field is not declared as a string` + p.GenStructFieldComment(ctx, field)
	}
	p.P(p.GoFieldName(ctx, field), p.GoFieldTypeName(ctx, p.Layer(), field), p.GenTag(ctx, field), notes)
	for _, field := range field.Attrs.Layout {
		p.GenStructField(ctx, field)
	}
}

func (p *TypeGenerator) GenStructFieldComment(ctx *Env, field *midl.Field, defaultPtr ...string) string {
	ret := ""
	for i, scope := range desc.Resolve(field).Layers {
		ret += fmt.Sprintf("(%d:", i+1)
		attr := scope.Attr.String()
		if len(defaultPtr) > 0 && len(scope.Nodes) > 0 && scope.Nodes[0].Is(midl.TypePointer) && scope.Attr.Pointer == midl.PointerTypeNone {
			if attr != "" {
				attr += ","
			}
			attr += "pointer=" + defaultPtr[0]
		}

		if attr != "" {
			ret += "{" + attr + "}"
		}
		for _, typ := range scope.Nodes {
			switch {
			case typ.Is(midl.TypePointer):
				ret += "*"
				ret += fmt.Sprintf("(%d)", typ.Pointer)
			case typ.Is(midl.TypeArray):
				ret += "["
				if typ.Array.IsFixed() {
					ret += fmt.Sprintf("%d", typ.Array.Size())
				} else {
					ret += typ.Dim.String()
				}
				ret += "]"
			default:
				ret += "(" + typ.Kind.String() + ")"
			}
		}
		ret += ")"
	}

	return ret
}

func (p *TypeGenerator) GenStructPreparePayloadAfterHook(ctx *Env) {
	p.CheckErr(p.B("ndr.AfterPreparePayload", "ctx", "o"))
}

func (p *TypeGenerator) GenStructPreparePayloadBeforeHook(ctx *Env) {
	p.CheckErr(p.B("ndr.BeforePreparePayload", "ctx", "o"))
}

func (p *TypeGenerator) GenStructPreparePayloadFields(ctx *Env, fields []*midl.Field) {

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

// LookupExprField returns the field of the current struct the expression refers to
// (its first identifier), or nil.
func (p *TypeGenerator) LookupExprField(ctx *Env, e midl.Expr) *midl.Field {

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
func (p *TypeGenerator) GenExpr(ctx *Env, e midl.Expr, field *midl.Field, typ string, n ...string) string {

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

	p.If(p.FormatExprO(ctx, cond), chk, func() {
		p.P(varName, "=", p.B(typ, p.FormatExprO(ctx, lval)))
	}, p.Else(func() {
		p.P(varName, "=", p.B(typ, p.FormatExprO(ctx, rval)))
	}))

	return varName
}

func (p *TypeGenerator) GenStructPreparePayload(ctx *Env) {

	p.P("")
	p.P("func", "(o *"+p.GoTypeName+")", p.XXX()+"PreparePayload(ctx context.Context)", "error", "{")

	p.GenStructPreparePayloadBeforeHook(ctx)
	p.GenStructPreparePayloadFields(ctx, p.Struct().Fields)
	p.GenStructPreparePayloadAfterHook(ctx)

	p.P("return", "nil")
	p.P("}")
}

func (p *TypeGenerator) GenStructPreparePayloadSizeExpr(ctx *Env, field *midl.Field, scopes *Scopes, sz midl.Expr) {

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
		szN := GoMemberName(ctx, szField.Name)
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

func (p *TypeGenerator) GenSubTypes(ctx *Env, fields []*midl.Field) {

	for _, field := range fields {
		p.GenSubType(ctx, field)
		for _, field := range field.Attrs.Layout {
			p.GenSubType(ctx, field)
		}
	}
}

func (p *TypeGenerator) GenSubType(ctx *Env, field *midl.Field) {

	switch t := field.Type.Base(); {
	case t.Is(midl.TypeStruct) || t.Is(midl.TypeUnion) || t.Is(midl.TypeCUnion):

		if LookupName(ctx, desc.Of(field).Base()) != "" {
			return
		}

		n := p.EmbeddedTypeName(ctx, p.Layer(), field)

		if strings.Contains(n, p.XXX()) {
			panic("invalid type " + n)
		}

		parent := p.Layer().Parent
		if parent == "" {
			parent = p.Alias()
		}

		p.GenType(ctx, &midl.Type{
			Kind:  midl.TypeAttribute,
			Attrs: field.Attrs.TypeAttr().Merge(&midl.TypeAttr{Alias: Escape(n), Parent: parent}),
			Elem:  t,
		})
	}
}

func (p *TypeGenerator) GenInterface(ctx *Env) {
	tt := *p.Session.LookupType("MInterfacePointer")
	tt.Attrs = tt.Attrs.Merge(p.OrigType.Attrs)
	p.NewTypeGenerator(ctx, &tt).GenStruct(ctx)
}

func (p *TypeGenerator) GenStruct(ctx *Env) {

	if p.Layer().Parent != "" {
		p.P("//", p.GoTypeName, "structure", "represents", RPCName(p.Layer().Parent), "structure", "anonymous", "member.")
	} else {
		p.P("//", p.GoTypeName, "structure", "represents", RPCName(p.Alias()), "RPC", "structure.")
	}
	if p.Doc != nil {
		p.P("//")
		p.GenComment(ctx, p.Doc.Documentation)
	}
	if len(p.Layer().DocString) > 0 {
		p.P("//")
		p.GenCommentLine(ctx, p.Layer().DocString)
	}

	if n := p.Layer().Names; len(n) > 0 {
		ctx = p.WithNamer(ctx, p.Scopes.WithAlias(n[0]))
		names := p.GoScopeTypeName(ctx, p.Layer(), nil, p.Scopes.WithAlias(n[0]), true)
		p.P("type", p.GoTypeName, names)
		p.P()
		p.P("func", "(o *"+p.GoTypeName+")", GoName(ctx, n[0]), "()", "*"+names, "{", "return", p.B("(*"+names+")", "o"), "}")
	} else {

		mask := p.GenNullMask(ctx)
		ctx = ctx.WithNullMask(mask)

		p.Structure(p.GoTypeName, func() {
			if mask != "" {
				p.P()
				p.P("//", mask, "is used to carry information on null-valued primitive values.")
				p.P(p.NullMask(), mask)
				p.P()
			}
			for _, field := range p.Struct().Fields {
				p.GenStructField(ctx, field)
			}
		})
	}

	p.GenStructPreparePayload(ctx)
	p.GenStructNDRSizeInfo(ctx)
	p.GenLayout(ctx)
	p.GenStructMarshalNDR(ctx)
	p.GenStructUnmarshalNDR(ctx)
	p.GenSubTypes(ctx, p.Struct().Fields)

}

func (p *TypeGenerator) GenNullMask(ctx *Env) string {

	nullable := []*midl.Field{}

	for _, field := range p.Struct().Fields {
		if IsIgnoredPointer(ctx, field) && GoFieldName(ctx, field) != "_" {
			nullable = append(nullable, field)
		}
	}

	mask := p.GoTypeName + p.NullMask()

	if len(nullable) == 0 {
		return ""
	}

	p.P("type", mask, "ndr.NullMask")
	p.P()
	p.P("var", "(")
	for i, field := range nullable {
		p.P(mask+GoFieldName(ctx, field), mask, "=", "1", "<<", i)
	}
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

func (p *TypeGenerator) GenLayout(ctx *Env) {
	if p.Layer().IsLayout {
		p.P("func", "(o *"+p.GoTypeName+")", "NDRLayout()", "{}")
	}
}

func (p *TypeGenerator) GenStructNDRSizeInfo(ctx *Env) {

	if !ndr.IsConformant(p.Scopes) {
		return
	}

	p.P()
	p.P("func", "(o *"+p.GoTypeName+")", "NDRSizeInfo() []uint64 {")

	last := p.Struct().LastField()
	if scopes := desc.Of(last); !scopes.Is(midl.TypeArray) {
		p.P("return", p.O(p.GoFieldName(ctx, last))+".NDRSizeInfo()")
	} else {

		var dims []string

		for i := 0; scopes.IsNotFixedArray(); scopes, i = scopes.Next(), i+1 {
			dims = append(dims, p.GenArraySizeVar(ctx, last, i, scopes, "Size", scopes.Dim().Size().Is()))
		}
		p.P("return []uint64{")
		for _, dim := range dims {
			p.P(dim, ",")
		}
		p.P("}")
	}

	p.P("}")
}

func (p *TypeGenerator) GenStructMarshalNDRSizePreamble(ctx *Env, field *midl.Field) {
	if !ndr.IsConformant(p.Scopes) {
		return
	}
	p.P("sizeInfo, ok := ctx.Value(ndr.SizeInfo).([]uint64)")
	p.If("!ok", func() {
		p.P("sizeInfo", "=", p.O("NDRSizeInfo()"))
		p.Range("sz1", "sizeInfo", func() {
			p.GenWriteSize(ctx, p.IVar("sizeInfo", "sz1"))
		})
		p.P("ctx = context.WithValue(ctx, ndr.SizeInfo, sizeInfo)")
	})
}

func (p *TypeGenerator) GenStructMarshalNDR(ctx *Env) {

	trailingField := p.Struct().LastFieldIndex()
	if ndr.IsConformant(p.Scopes) || ndr.IsVarying(p.Scopes) {
		trailingField--
	}

	p.Block("func", "(o *"+p.GoTypeName+")", "MarshalNDR(ctx context.Context, w ndr.Writer)", "error", func() {
		// prepare payload.
		p.CheckErr("o." + p.XXX() + "PreparePayload(ctx)")
		// write size information.
		p.GenStructMarshalNDRSizePreamble(ctx, p.StructLastField())
		// align the structure.
		p.GenDoAlignmentMarshalNDR(ctx, ndr.Alignment(p.Scopes))
		// marshal fields.
		for i, field := range p.Struct().Fields {
			// marshal field.
			p.GenFieldMarshalNDR(ctx, field, desc.Of(field))
			// add trailing gap.
			if i == trailingField {
				if f := desc.Of(field); ndr.PlainAlignment(f) != ndr.PlainAlignment(p.Scopes) {
					p.GenDoTrailingGapMarshalNDR(ctx, ndr.Alignment(p.Scopes))
				}
			}
		}
		if p.Layer().Pad != 0 {
			p.P("//", "pad", p.Layer().Pad)
			p.GenDoAlignmentMarshalNDR(ctx, int(p.Layer().Pad))
		}
		p.P("return nil")
	})

}

func (p *TypeGenerator) GenFieldMarshalNDR(ctx *Env, field *midl.Field, scopes *Scopes, index ...interface{}) {

	if scopes == nil || field.Attrs.Ignore {
		// end.
		return
	}

	// reserved field.
	if p.GoFieldName(ctx, field) == "_" {
		p.P("//", "reserved", field.Name)
		p.GenZeroFieldMarshalNDR(ctx.WithVarName(ZeroLen), field, scopes, index...)
		return
	}

	// get variable name.
	name := p.VarName(ctx, index...)
	if name == "" {
		name = p.O(p.IVar(p.GoFieldName(ctx, field), index...))
	}

	switch {

	case scopes.Type().Is(midl.TypeHandle):

		p.P("//", "skip", "[handle]", field.Name)

	case scopes.Type().IsPrimitiveType():

		// marshal primitive type.

		if scopes.IsBool() && midl.PrimitiveTypeSize(scopes.Kind()) > 1 {
			typeName := GoPrimitiveTypeName(scopes.Kind())
			p.If("!", name, func() {
				p.CheckErr(p.B("w.WriteData", p.B(typeName, "0")))
			}, p.Else(func() {
				p.CheckErr(p.B("w.WriteData", p.B(typeName, "1")))
			}))
		} else if scopes.Is(midl.TypeInt32_64) {
			p.CheckErr(p.B("w.WriteData", p.B("ndr.Int3264", name)))
		} else if scopes.Is(midl.TypeUint32_64) {
			p.CheckErr(p.B("w.WriteData", p.B("ndr.Uint3264", name)))
		} else if field.Attrs.Format.Rune {
			switch scopes.Kind() {
			case midl.TypeChar, midl.TypeUChar, midl.TypeUint8:
				p.CheckErr(p.B("w.WriteData", p.B("uint8", name)))
			case midl.TypeWChar, midl.TypeUint16, midl.TypeInt16:
				p.CheckErr(p.B("w.WriteData", p.B("uint16", name)))
			}
		} else {
			p.CheckErr(p.B("w.WriteData", name))
		}

	case scopes.Is(midl.TypeEnum):

		// marshal enum.

		p.CheckErr(p.B("w.WriteEnum", p.B(scopes.EnumType(), name)))

	case scopes.Is(midl.TypeUnion):

		// marshal union.

		switchVar := "_sw" + p.ToVar(name)
		if !scopes.Union().IsEncapsulated() {
			p.GenerateSwitchIs(ctx, field, scopes, switchVar)
		}

		p.If(name, "!=", "nil", func() {
			if !scopes.Union().IsEncapsulated() {
				p.CheckErr(p.B(name+".MarshalUnionNDR", "ctx", "w", switchVar))
			} else {
				p.CheckErr(p.B(name+".MarshalNDR", "ctx", "w"))
			}
		}, p.Else(func() {
			p.GenZeroUnionFieldMarshalNDR(ctx, field, scopes, index...)
		}))

	case scopes.Is(midl.TypeStruct), scopes.Is(midl.TypeInterface):

		// marshal structure.

		p.If(name, "!=", "nil", func() {
			p.CheckErr(p.B(name+".MarshalNDR", "ctx", "w"))
		}, p.Else(func() {
			p.GenZeroStructFieldMarshalNDR(ctx, field, scopes, index...)
		}))

	case scopes.Is(midl.TypePointer):

		// marshal pointer.

		var nullChk string

		if next := scopes.Next(); next.Type().IsPrimitiveType() {

			if field.Attrs.DefaultNull != nil {
				if len(field.Attrs.DefaultNull) > 0 {
					for _, expr := range field.Attrs.DefaultNull {
						chk := p.GenExpr(ctx, expr, p.LookupExprField(ctx, expr), "")
						if nullChk == "" {
							nullChk = fmt.Sprintf("!%s", chk)
						} else {
							nullChk = fmt.Sprintf("%s || !%s", nullChk, chk)
						}
					}
				} else {
					nullChk = fmt.Sprintf("%s != %v", name, p.GoTypeZeroValue(ctx, p.Layer(), field, scopes))
				}
			}

			if HasNullMaskBit(ctx, field) {
				if nullChk != "" {
					nullChk += "&& "
				}
				nullChk += fmt.Sprintf("(%s & %s == 0 || %s != %s)", p.O(p.NullMask()), ctx.NullMask+GoFieldName(ctx, field),
					name, p.GoTypeZeroValue(ctx, p.Layer(), field, scopes))
			}

			if nullChk == "" && !scopes.Layer().Pointer.IsWeak(ctx.Operation) {
				p.P("// XXX: pointer default mask was not generated as conditions")
				p.P("// for automatic generation were not met.")
				p.P("// If this behavior is not desired, use goext_default_null([cond]) attribute.")
			}
		}

		if nullChk != "" {
			p.If(nullChk, func() {
				fN := "_ptr_" + field.Name
				p.P(fN, ":=", "ndr.MarshalNDRFunc", "(", "func(ctx context.Context, w ndr.Writer) error {")
				p.GenFieldMarshalNDR(ctx, field, scopes.Next(), index...)
				p.P("return nil")
				p.P("})")
				p.CheckErr(p.B("w.WritePointer", p.Amp(name), fN))
			}, p.Else(func() {
				p.GenZeroPointerFieldMarshalNDR(ctx, field, scopes, index...)
			}))
		} else {

			sizeChk := ""
			if next := scopes.Next(); next.Is(midl.TypeArray) && !next.Dim().Size().Empty() {
				// immitate memory allocation for conformance array.
				sizeChk = fmt.Sprintf("|| %s > 0", p.GenExpr(ctx, next.Dim().Size().Is(), p.LookupExprField(ctx, next.Dim().Size().Is()), ""))
			}

			p.If(name, "!=", p.GoTypeZeroValue(ctx, p.Layer(), field, scopes), sizeChk, func() {
				fN := "_ptr_" + field.Name
				p.P(fN, ":=", "ndr.MarshalNDRFunc", "(", "func(ctx context.Context, w ndr.Writer) error {")
				p.GenFieldMarshalNDR(ctx, field, scopes.Next(), index...)
				p.P("return nil")
				p.P("})")
				p.CheckErr(p.B("w.WritePointer", p.Amp(name), fN))
			}, p.Else(func() {
				p.GenZeroPointerFieldMarshalNDR(ctx, field, scopes, index...)
			}))
		}

	case scopes.Is(midl.TypeArray):

		// marshal array.

		ctx := ctx.WithNullMask("") /* clear null mask as arrays are unsupported */

		if !p.GenMarshalSizeInfo(ctx, field, scopes, index...) {
			break
		}

		idx := p.Var("i", len(index)+1)

		if scopes.Dim().IsString {
			name = p.GenWriteStringBuffer(ctx, scopes, field, index...)
			ctx, index = ctx.WithVarName(name), nil
		}

		if scopes.Dim().NoSizeLimit {
			p.Range(idx, name, func() {
				p.P(idx, ":=", idx)
				p.GenFieldMarshalNDR(ctx, field, scopes.Next(), append(index, idx)...)
			})
			break
		}

		p.Range(idx, name, func() {
			// iterate over the array.
			p.P(idx, ":=", idx)
			if scopes.Array().IsFixed() && (!scopes.Dim().IsString || p.IsNotLastFixedArray(ctx, scopes, field)) {
				// break if index exceeds the fixed-array limit.
				p.If(p.B("uint64", idx), ">=", scopes.Array().Size(), func() {
					p.P("break")
				})
			} else {
				// break if index exceeds the conformance/varyingness limit.
				p.If(p.B("uint64", idx), ">=", p.IVar("sizeInfo", scopes.Dim().Dimension), func() {
					p.P("break")
				})
			}
			p.GenFieldMarshalNDR(ctx, field, scopes.Next(), append(index, idx)...)
		})

		p.GenZeroFieldMarshalNDR(ctx, field, scopes, index...)

	case scopes.Is(midl.TypePipe):

		p.P("//", "marshal", "pipe", field.Name)

		field := p.GenPipeField(ctx, scopes.Type())

		p.If(name, "!=", "nil", func() {
			p.Range("_chunk", name+".Pipe()", func() {
				p.GenFieldMarshalNDR(ctx.WithVarName("_chunk"), field, desc.Of(field))
			})
		})
		p.GenWriteSize(ctx, "0")

	default:
		p.P("//", "FIXME", "unknown type", field.Name)
	}
}

func (p *TypeGenerator) IsNotLastFixedArray(ctx *Env, scopes *Scopes, field *midl.Field) bool {
	return !ndr.IsTopLevelArray(scopes) && scopes.Array().IsFixed() && p.Struct() != nil && len(p.Struct().Fields) > 0 && field.Position != p.Struct().LastField().Position
}

func (p *TypeGenerator) GenMarshalSizeInfo(ctx *Env, field *midl.Field, scopes *Scopes, index ...interface{}) bool {

	// get variable name.
	name := p.VarName(ctx, index...)
	if name == "" {
		name = p.O(p.IVar(p.GoFieldName(ctx, field), index...))
	}

	if p.IsNotLastFixedArray(ctx, scopes, field) {
		return true
	}

	isConformant, isVarying := ndr.IsConformant(scopes), ndr.IsVarying(scopes)

	if field.Position == 0 || ndr.IsTopLevelArray(scopes) {

		// for non-top level array size information is handled via NDRSizeInfo.

		if dim := scopes.Dim(); dim.IsString && !dim.NoSizeLimit && dim.Size().Empty() && dim.LengthIs.IsEmpty() {

			// top-level string.

			if name == ZeroLen {
				name = `""`
			}

			switch scopes.Next().Kind() {
			case midl.TypeWChar, midl.TypeUint16, midl.TypeInt16:
				if dim.IsNullTerminated {
					p.CheckErr(p.B("ndr.WriteUTF16NString", "ctx", "w", name))
				} else {
					p.CheckErr(p.B("ndr.WriteUTF16String", "ctx", "w", name))
				}
			case midl.TypeUChar, midl.TypeChar, midl.TypeUint8:
				if dim.IsNullTerminated {
					p.CheckErr(p.B("ndr.WriteCharNString", "ctx", "w", name))
				} else {
					p.CheckErr(p.B("ndr.WriteCharString", "ctx", "w", name))
				}
			}

			return false

			// else, string length should be retrieved from external parameters
			// size is, length is, and so on.
		}

		if isConformant {

			dims := []string{}
			for scopes, i := scopes, len(index); scopes.IsNotFixedArray(); scopes, i = scopes.Next(), i+1 {
				dims = append(dims, p.GenArraySizeVar(ctx, field, i, scopes, "Size", scopes.Dim().Size().Is()))
				// write max_count.
				p.GenWriteSize(ctx, dims[len(dims)-1])
			}

			p.Block("sizeInfo := []uint64", func() {
				for _, dim := range dims {
					p.P(dim, ",")
				}
			})
		}
	}

	if isVarying {

		if !isConformant {
			p.Block("sizeInfo := []uint64", func() {
				for scopes, i := scopes, len(index); scopes.IsNotFixedArray(); scopes, i = scopes.Next(), i+1 {
					p.P(0, ",")
				}
			})
		}

		for scopes, i := scopes, len(index); scopes.IsNotFixedArray(); scopes, i = scopes.Next(), i+1 {
			dim := p.GenArraySizeVar(ctx, field, i, scopes, "Length", scopes.Dim().LengthIs)
			if !isConformant {
				p.P(p.IVar("sizeInfo", scopes.Dim().Dimension), "=", dim)
			} else {
				p.If(dim, ">", p.IVar("sizeInfo", scopes.Dim().Dimension), func() {
					p.P(dim, "=", p.IVar("sizeInfo", scopes.Dim().Dimension))
				}, p.Else(func() {
					// reset the size information back for padding and etc.
					p.P(p.IVar("sizeInfo", scopes.Dim().Dimension), "=", dim)
				}))
			}
			// write offset.
			p.GenWriteSize(ctx, 0)
			// write actual_count.
			p.GenWriteSize(ctx, dim)
		}
	}

	return true
}

// GetMinVarValue reports whether the size expression can become negative when
// its only identifier is zero, e.g. size_is(n - 1). It returns the identifier
// and the value it must be at least.
func (p *TypeGenerator) GetMinVarValue(ctx *Env, sizeExpr midl.Expr) (midl.Expr, int64, bool) {
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

func (p *TypeGenerator) GenArraySizeVar(ctx *Env, field *midl.Field, i int, scopes *Scopes, name string, sizeExpr midl.Expr) string {

	if !scopes.IsNotFixedArray() {
		return ""
	}

	dimVar := p.Var("dim"+name, i+1)

	if !sizeExpr.IsEmpty() {
		p.P(dimVar, ":=", p.B("uint64", p.GenExpr(ctx, sizeExpr, p.LookupExprField(ctx, sizeExpr), "uint64")))
		// check minimum value borders.
		if to, val, ok := p.GetMinVarValue(ctx, sizeExpr); ok {
			p.If(p.GenExpr(ctx, to, p.LookupExprField(ctx, to), ""), "<", val, func() {
				p.P(dimVar, "=", "uint64(0)")
			})
		}
		return dimVar
	}

	if i == 0 {
		// short-path.
		// get variable name.
		name := p.VarName(ctx)
		if name == "" {
			name = p.O(GoFieldName(ctx, field))
		}
		p.P(dimVar, ":=", p.DataLen(ctx, field, scopes, name))
		return dimVar
	}

	// calculate size on the fly.
	p.P(dimVar, ":=", "uint64(0)")
	p.GenRange(ctx, p.O(p.GoFieldName(ctx, field)), 0, i, func(ctx *Env, n string) {
		lenVar := p.Var("len" + Title(dimVar))
		p.If(lenVar, ":=", p.DataLen(ctx, field, scopes, n), ";", lenVar, ">", dimVar, func() {
			p.P(dimVar, "=", lenVar)
		})
	})

	return dimVar
}

// GenWriteStringBuffer function ...
func (p *TypeGenerator) GenWriteStringBuffer(ctx *Env, scopes *Scopes, field *midl.Field, index ...interface{}) string {

	name := p.O(p.IVar(p.GoFieldName(ctx, field), index...))

	// XXX: either use ZeroString as a pointer discriminator.
	// p.P(name, "=", p.B("strings.TrimPrefix", name, "ndr.ZeroString"))

	varName := p.BufVar(name)

	if field.Attrs.Format.MultiSize {
		p.P("var", varName, "[]uint16")
		p.Range("i1", name, func() {
			p.P(varName, "=", p.B("append", varName, p.B("utf16.Encode", p.B("[]rune", p.Var(name, "[i1]")))+"..."))
			p.P(varName, "=", p.B("append", varName, p.B("uint16", "0")))
		})
		p.P(varName, "=", p.B("append", varName, p.B("uint16", "0")))
		return varName
	}

	switch scopes := scopes.Next(); scopes.Kind() {
	case midl.TypeWChar, midl.TypeUint16, midl.TypeInt16:
		p.P(varName, ":=", p.B("utf16.Encode", p.B("[]rune", name)))
	case midl.TypeChar, midl.TypeUChar, midl.TypeInt8, midl.TypeUint8:
		p.P(varName, ":=", p.B("[]byte", name))
	default:
		p.P(`return errors.New("cannot handle ` + scopes.Kind().String() + ` to string type conversion")`)
	}

	if scopes.Dim().NoSizeLimit {
		// skip for unlimited size types.
		return varName
	}

	if scopes.Dim().IsNullTerminated {

		// declared with [string] attribute. size written should be one less
		// to include zero.

		if scopes.Array().IsFixed() {
			size := scopes.Array().Size()
			p.If(p.B("uint64", p.Len(varName)), ">", size, "-", "1", func() {
				p.P(varName, "=", varName+fmt.Sprintf("[:%d-1]", size))
			})
		} else {
			sizeInfo := p.IVar("sizeInfo", scopes.Dim().Dimension)
			p.If(p.B("uint64", p.Len(varName)), ">", sizeInfo, "-", "1", func() {
				p.P(varName, "=", varName+p.IVar("", ":"+sizeInfo+"-1"))
			})
		}

		zeroValue := p.B(GoPrimitiveTypeName(scopes.Next().Kind()), "0")

		// append zeroValue only if string is not zero.
		p.If(name, "!=", "ndr.ZeroString", func() {
			p.P(varName, "=", p.B("append", varName, zeroValue))
		})

	} else {

		if scopes.Array().IsFixed() {
			size := scopes.Array().Size()
			p.If(p.B("uint64", p.Len(varName)), ">", size, func() {
				p.P(varName, "=", varName+fmt.Sprintf("[:%d]", size))
			})
		} else {
			sizeInfo := p.IVar("sizeInfo", scopes.Dim().Dimension)
			p.If(p.B("uint64", p.Len(varName)), ">", sizeInfo, func() {
				p.P(varName, "=", varName+p.IVar("", ":"+sizeInfo))
			})
		}
	}

	return varName
}

func (p *TypeGenerator) GoFieldName(ctx *Env, field *midl.Field) string {

	if field.Name == "" {
		if field.Type.Base().Is(midl.TypeUnion) {
			field.Name = p.GoTypeName
		}
	}

	return GoFieldName(ctx, field)

}
