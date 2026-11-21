package golang

import (
	"context"

	"github.com/oiweiwei/midl-gen-go/codegen/golang/emit"
	"github.com/oiweiwei/midl-gen-go/gonames"
	"github.com/oiweiwei/midl-gen-go/midl"
	"github.com/oiweiwei/midl-gen-go/msdn/openspecs"
)

// Symbol is a name of the IDL and the Go name the generator gives it.
type Symbol struct {
	// Kind is what the name is: "const", "type", "field", "arm", "element",
	// "interface", "operation" or "param".
	Kind string `json:"kind"`
	// Owner is the IDL name of the struct, union, enum or operation the name is a
	// member of; empty for a name that is not a member. It is the key of the
	// override section of the gonames rules.
	Owner string `json:"owner,omitempty"`
	// IDL is the name in the IDL file.
	IDL string `json:"idl"`
	// Go is the name as it appears in the generated code.
	Go string `json:"go"`
	// Trace says how the words of the name were derived and which rules fired.
	Trace *gonames.Trace `json:"trace"`
}

// Symbols lists the names of the IDL file with the Go names the generator
// gives them, in the order the generator meets them. It uses the same naming
// functions as the generation, so it shows what the generated code will have.
func (p *Generator) Symbols(parent context.Context, f *midl.File) []Symbol {

	p.MSDN = &openspecs.MSDN{}

	ctx := emit.NewEnv(parent, p.Session).WithFile(f).WithNamer(f.Namer())

	var out []Symbol

	for _, sym := range f.Exports() {
		if sym.Const != nil {
			out = append(out, p.symbol(ctx, "const", "", sym.Const.Name, true, emit.GoName(ctx, sym.Const.Name)))
		}
		if sym.Type != nil {
			out = append(out, p.typeSymbols(ctx, sym.Type)...)
		}
	}

	for _, iff := range f.Interfaces {
		if !iff.ForwardDeclarator {
			out = append(out, p.interfaceSymbols(ctx.WithInterface(iff), iff)...)
		}
	}

	for _, module := range f.Modules {
		for _, m := range module.Members {
			if m.Const != nil {
				out = append(out, p.symbol(ctx, "const", module.Name, m.Const.Name, true, emit.GoName(ctx, m.Const.Name)))
			}
		}
	}

	for _, lib := range f.Libraries {
		out = append(out, p.Symbols(parent, lib.Body)...)
	}

	return out
}

// symbol builds a Symbol; reserved is the flag the name is derived with.
func (p *Generator) symbol(ctx *emit.Env, kind, owner, idl string, reserved bool, goName string) Symbol {
	return Symbol{
		Kind:  kind,
		Owner: owner,
		IDL:   idl,
		Go:    goName,
		Trace: gonames.Resolve(ctx.Namer).ExplainMember(owner, idl, reserved),
	}
}

func (p *Generator) interfaceSymbols(ctx *emit.Env, iff *midl.Interface) []Symbol {

	out := []Symbol{p.symbol(ctx, "interface", "", iff.Name, true, emit.GoName(ctx, iff.Name))}

	for _, sym := range iff.Exports() {
		if sym.Const != nil {
			out = append(out, p.symbol(ctx, "const", iff.Name, sym.Const.Name, true, emit.GoName(ctx, sym.Const.Name)))
		}
		if sym.Type != nil {
			out = append(out, p.typeSymbols(ctx, sym.Type)...)
		}
	}

	for _, op := range iff.Body.Operations {
		out = append(out, p.symbol(ctx, "operation", iff.Name, op.Name, true, p.MethodName(ctx, op)))
		octx := ctx.WithOwner(op.Name)
		for _, param := range op.Params {
			out = append(out, p.symbol(octx, "param", op.Name, param.Name, true, emit.GoParamName(octx, param)))
		}
	}

	return out
}

func (p *Generator) typeSymbols(ctx *emit.Env, t *midl.Type) []Symbol {

	pp := p.NewTypeGenerator(ctx, t)

	if pp.Skip() {
		return nil
	}

	owner := pp.OwnerName()
	ctx = ctx.WithOwner(owner)

	out := []Symbol{p.symbol(ctx, "type", "", owner, true, pp.GoTypeName)}

	field := func(kind string, f *midl.Field) {
		name := emit.FieldName(f.Position, f.Name)
		out = append(out, p.symbol(ctx, kind, owner, name, true, emit.GoFieldName(ctx, f)))
		for _, nested := range f.Attrs.Layout {
			out = append(out, p.symbol(ctx, kind, owner, emit.FieldName(nested.Position, nested.Name), true, emit.GoFieldName(ctx, nested)))
		}
	}

	switch {
	case pp.Is(midl.TypeStruct):
		for _, f := range pp.Struct().Fields {
			field("field", f)
		}
	case pp.Is(midl.TypeUnion) || pp.Is(midl.TypeCUnion):
		for _, c := range pp.Union().Body {
			for _, arm := range c.Arms {
				field("arm", arm)
			}
		}
	case pp.Is(midl.TypeEnum):
		for _, e := range pp.Enum().Elems {
			out = append(out, p.symbol(ctx, "element", owner, e.Value, false,
				emit.GoMergeNames(ctx, pp.GoTypeName, emit.GoMemberNameNoReserved(ctx, e.Value))))
		}
	}

	return out
}
