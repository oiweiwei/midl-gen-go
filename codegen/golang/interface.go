package golang

import (
	"fmt"

	"github.com/oiweiwei/midl-gen-go/codegen/golang/emit"
	"github.com/oiweiwei/midl-gen-go/desc"
	"github.com/oiweiwei/midl-gen-go/midl"
)

func (p *Generator) SyntaxName(ctx *emit.Env, iff *midl.Interface) string {
	n := emit.GoName(ctx, iff.Name) + "Syntax"
	ver := iff.Attrs.Version
	if ver == nil {
		ver = &midl.Version{}
	}
	return fmt.Sprintf("%sV%d_%d", n, ver.Major, ver.Minor)
}

func (p *Generator) SyntaxNameWithImport(ctx *emit.Env, iff *midl.Interface) string {
	n := p.GoInterfaceTypeName(ctx, iff) + "Syntax"
	ver := iff.Attrs.Version
	if ver == nil {
		ver = &midl.Version{}
	}
	return fmt.Sprintf("%sV%d_%d", n, ver.Major, ver.Minor)
}

func (p *Generator) GenInterfaceID(ctx *emit.Env, iff *midl.Interface) {

	if iff.Attrs.UUID == nil {
		return
	}

	n := emit.GoName(ctx, iff.Name) + "Syntax"

	p.P()
	p.P("var", "(")
	if iff.IsObject() {
		_ = p.GoPackageName(ctx, desc.OfType(p.Session.LookupType("IID")))
		p.P("//", iff.Name, "interface", "identifier", iff.Attrs.UUID)
		p.P(emit.GoName(ctx, iff.Name)+"IID", "=", "&dcom.IID", emit.UUIDToGUID(iff.Attrs.UUID))
	}

	p.P("//", "Syntax UUID")
	p.P(n+"UUID", "=", fmt.Sprintf("%#v", iff.Attrs.UUID))
	ver := iff.Attrs.Version
	if ver == nil {
		ver = &midl.Version{Major: 0, Minor: 0}
	}
	p.P("//", "Syntax ID")
	p.P(n+fmt.Sprintf("V%d_%d", ver.Major, ver.Minor), "=", "&dcerpc.SyntaxID{",
		"IfUUID:", n+"UUID", ",", "IfVersionMajor:", ver.Major, ",", "IfVersionMinor:", ver.Minor, "}")
	p.P(")")
}
