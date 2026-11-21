package emit

import (
	"strings"

	"github.com/oiweiwei/midl-gen-go/midl"
)

func (p *Emitter) GenTag(ctx *Env, field *midl.Field) string {
	tag := []string{`name:` + field.Name}

	for _, size := range []struct {
		E []midl.Expr
		N string
	}{
		{field.Attrs.SizeIs, "size_is"},
		{field.Attrs.MaxIs, "max_is"},
		{field.Attrs.MinIs, "min_is"},
		{field.Attrs.LengthIs, "length_is"},
		{field.Attrs.FirstIs, "first_is"},
		{field.Attrs.LastIs, "last_is"},
	} {
		if size.E == nil {
			continue
		}

		szs := []string{}
		for _, sz := range size.E {
			szs = append(szs, p.FormatExpr(ctx, sz))
		}
		tag = append(tag, size.N+":("+strings.Join(szs, ", ")+")")
	}

	if field.Attrs.Usage.IsString {
		tag = append(tag, "string")
	}

	if !field.Attrs.SwitchIs.IsEmpty() {
		tag = append(tag, "switch_is:"+field.Attrs.SwitchIs.String())
	}

	if field.Attrs.Pointer != midl.PointerTypeNone && field.Attrs.Pointer != midl.PointerTypeRefWeak {
		tag = append(tag, "pointer:"+field.Attrs.Pointer.String())
	}

	if field.Attrs.Ignore {
		tag = append(tag, "ignore")
	}

	n := GoSnakeCase(ctx, field.Name)
	if n == "_" {
		return "`idl:\"" + strings.Join(tag, ";") + "\"`"
	}

	return "`idl:\"" + strings.Join(tag, ";") + "\" json:\"" + n + "\"`"
}
