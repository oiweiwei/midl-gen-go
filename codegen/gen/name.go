package gen

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	go_names "github.com/oiweiwei/midl-gen-go/gonames"
	"github.com/oiweiwei/midl-gen-go/midl"
)

var (
	GoName           = go_names.GoName
	GoNamePrivate    = go_names.GoNamePrivate
	GoNameNoReserved = go_names.GoNameNoReserved
	GoMergeNames     = go_names.GoMergeNames
	GoSnakeCase      = go_names.GoSnakeCase
	LexName          = go_names.GoLexName
	Escape           = go_names.Escape
	RPCName          = go_names.Unescape
	Title            = go_names.Title
)

func GoHex(n int) func(uint64) string {
	return func(v uint64) string {
		return fmt.Sprintf("0x%0*X", n*2, v)
	}
}

var defaultVersion = &midl.Version{}

func (p *Generator) WithInterfaceNamer(ctx context.Context, iff *midl.Interface) context.Context {
	for _, f := range midl.Files() {
		if _, ok := f.LookupType(iff.Name); !ok {
			continue
		}
		return go_names.WithNamer(ctx, f.Namer())
	}
	return ctx
}

func (p *Generator) WithInterfaceBaseNamer(ctx context.Context, iff *midl.Interface) context.Context {
	name := iff.BaseName
	if iff.Base != nil {
		name = iff.Base.Name
	}
	for _, f := range midl.Files() {
		if _, ok := f.LookupType(name); !ok {
			continue
		}
		return go_names.WithNamer(ctx, f.Namer())
	}
	return ctx
}

func (p *Generator) GoInterfaceTypeName(ctx context.Context, iff *midl.Interface, nhook ...func(string) string) string {

	n := iff.Name

	for _, f := range midl.Files() {

		if _, ok := f.LookupType(iff.Name); !ok {
			continue
		}

		ctx = go_names.WithNamer(ctx, f.Namer())

		n := GoName(ctx, iff.Name)
		for _, hook := range nhook {
			n = hook(n)
		}

		pkg, ver := filepath.Join(f.GoPkg, strings.ToLower(iff.Name)), iff.Attrs.Version
		if ver == nil {
			ver = defaultVersion
		}

		base := p.ImportsPath
		if f.GoPkgBase != "" {
			base = f.GoPkgBase
		}

		pkgName := lastPart(pkg)

		p.AddImport(Import{
			Name:  pkgName,
			Path:  filepath.Join(base, pkg, ver.String()),
			Guard: pkgName + "." + "GoPackage",
		})

		return pkgName + "." + n
	}

	return n
}

func (p *Generator) GoScopeTypeNameWithN(ctx context.Context, attr *midl.TypeAttr, field *midl.Field, scopes *Scopes, trim bool, n ...string) string {
	t := p.GoTypeName(ctx, attr, field, scopes, n...)
	if trim {
		return strings.TrimLeft(t, "*")
	}
	return t
}

func (p *Generator) GoScopeTypeName(ctx context.Context, attr *midl.TypeAttr, field *midl.Field, scopes *Scopes, trim ...bool) string {
	n := p.GoTypeName(ctx, attr, field, scopes)
	if len(trim) > 0 {
		return strings.TrimLeft(n, "*")
	}
	return n
}

func (p *Generator) GoFieldTypeName(ctx context.Context, attr *midl.TypeAttr, field *midl.Field, n ...string) string {
	return p.GoTypeName(ctx, attr, field, NewScopes(field.Scopes()), n...)
}

func (p *Generator) EmbeddedTypeName(ctx context.Context, attrs *midl.TypeAttr, field *midl.Field) string {
	return GoName(ctx, attrs.Alias) + "_" + GoFieldName(ctx, field)
}

func (p *Generator) GoTypeName(ctx context.Context, attrs *midl.TypeAttr, field *midl.Field, scopes *Scopes, n ...string) string {

	var ret string

go_type_name_loop:
	for scopes := scopes; scopes != nil; scopes = scopes.Next() {

		switch {
		case scopes.Is(midl.TypeArray):

			if scopes.Dim().IsString {
				// string is a terminal type.
				ret += "string"

				if field.Attrs.Format.MultiSize {
					ret = "[]" + ret
				}

				break go_type_name_loop
			}

			// array declaration. add [] if it is not a string.
			ret = "[]" + ret
			continue

		case scopes.Is(midl.TypePointer):
			// pointer declaration. noop.
			continue

		case scopes.Is(midl.TypeVoid):
			ret = "[]byte"
			break go_type_name_loop
		}

		// primitive type.
		if scopes.Type().IsPrimitiveType() {
			if scopes.IsBool() {
				ret += "bool"
				break go_type_name_loop
			}

			if field.Attrs.Format.Rune {
				ret += "rune"
				break go_type_name_loop
			}

			ret += GoPrimitiveTypeName(scopes.Kind())
			// terminal type.
			break go_type_name_loop
		}

		// constructed (exported type) type.

		if !scopes.Is(midl.TypeEnum) && !scopes.Is(midl.TypePipe) {
			// always with pointer.
			ret += "*"
		}

		// lookup type name.
		name := LookupName(ctx, scopes)
		if name == "" {
			ret += p.EmbeddedTypeName(ctx, attrs, field)
		} else {
			ret += p.GoPackageName(ctx, scopes) + GoName(p.WithNamer(ctx, scopes), name)
		}

		break go_type_name_loop
	}

	if ret == "byte" {
		ret = "uint8"
	}

	if strings.Contains(ret, "[]") {
		ret = strings.ReplaceAll(ret, "uint8", "byte")
	}

	if strings.Contains(ret, ".") {
		return strings.ReplaceAll(ret, ".", "."+strings.Join(n, ""))
	}

	return strings.Join(n, "") + ret
}

func (p *Generator) GoPackageName(ctx context.Context, scopes *Scopes) string {

	pkg, err := p.GoPackage(ctx, scopes)
	if err != nil {
		panic(fmt.Sprintf("cannot determine package for type: %v", err))
	}

	if pkg == nil {
		return ""
	}

	return lastPart(pkg.Name) + "."
}

func (p *Generator) WithNamer(ctx context.Context, scopes *Scopes) context.Context {
	pkg, _ := p.GoPackage(ctx, scopes)
	if pkg == nil || pkg.File == nil || pkg.File.Namer() == nil {
		return ctx
	}
	return go_names.WithNamer(ctx, pkg.File.Namer())
}

func (p *Generator) GoPackage(ctx context.Context, scopes *Scopes) (*Package, error) {

	n := scopes.Alias()
	if n == "" {
		n = scopes.Type().TypeName()
	}

	if n == "" {
		return nil, fmt.Errorf("cannot determine package for type")
	}

	pkg, ok := GoPackage(ctx, n)
	if !ok {
		return nil, fmt.Errorf("cannot find package for type: %s", n)
	}

	if pkg.Name == "" {
		return nil, nil
	}

	if strings.Contains(pkg.Name, "guiddef") {
		pkg.Name = "dtyp"
	}

	pkgName := lastPart(pkg.Name)

	if pkg.Base == "" {
		pkg.Base = p.ImportsPath
	}

	if pkg.Version != nil {
		p.AddImport(Import{
			Name:  pkgName,
			Path:  filepath.Join(pkg.Base, pkg.Name, pkg.Version.String()),
			Guard: pkgName + "." + "GoPackage",
		})
	} else {
		p.AddImport(Import{
			Name:  pkgName,
			Path:  filepath.Join(pkg.Base, pkg.Name),
			Guard: pkgName + "." + "GoPackage",
		})
	}

	return pkg, nil
}

func lastPart(n string) string {
	parts := strings.Split(n, "/")
	return parts[len(parts)-1]
}

type Package struct {
	Name    string
	Base    string
	Version *midl.Version
	File    *midl.File
}

func (p *Package) TypeName(n string) string {
	if p.File != nil {
		if namer := p.File.Namer(); namer != nil {
			return GoName(go_names.WithNamer(context.Background(), namer), n)
		}
	}
	return n
}

func GoPackage(ctx context.Context, n string) (*Package, bool) {

	var defaultVersion = &midl.Version{}

	if iff := Interface(ctx); iff != nil {

		// we are rendering interface.

		if _, ok := iff.Body.Export[n]; ok {
			// type is local to interface.
			return &Package{}, true
		}

		for _, iff := range File(ctx).Interfaces {
			// type is in some other interface.
			// (in the same file).
			if _, ok := iff.Body.Export[n]; ok {

				if iff.Attrs.Version == nil {
					return &Package{
						Name:    filepath.Join(File(ctx).GoPkg, strings.ToLower(iff.Name)),
						Base:    "",
						Version: defaultVersion,
					}, true
				}

				return &Package{
					Name:    filepath.Join(File(ctx).GoPkg, strings.ToLower(iff.Name)),
					Base:    "",
					Version: iff.Attrs.Version,
				}, true
			}
		}

		if File(ctx).IsLocal(n) {
			// type is file-level definition.
			return &Package{
				Name:    File(ctx).GoPkg,
				Base:    "",
				Version: nil,
			}, true
		}
	} else {
		if _, ok := File(ctx).Export[n]; ok {
			return &Package{}, true
		}
	}

	for _, f := range midl.Files() {
		if _, ok := f.Export[n]; ok {
			return &Package{
				Name: f.GoPkg,
				Base: f.GoPkgBase,
				File: f,
			}, true
		}
		for _, iff := range f.Interfaces {
			if _, ok := iff.Body.Export[n]; ok {
				// type is local to interface.
				if iff.Attrs.Version == nil {
					return &Package{
						Name:    filepath.Join(f.GoPkg, strings.ToLower(iff.Name)),
						Base:    f.GoPkgBase,
						Version: defaultVersion,
						File:    f,
					}, true
				}
				return &Package{
					Name:    filepath.Join(f.GoPkg, strings.ToLower(iff.Name)),
					Base:    f.GoPkgBase,
					Version: iff.Attrs.Version,
					File:    f,
				}, true
			}
		}
	}

	return nil, false
}

func LookupName(ctx context.Context, scopes *Scopes) string {

	if alias := scopes.Alias(); alias != "" {
		if strings.HasPrefix(alias, "PDNS") {
			// FIXME: dnsp.idl
			alias = scopes.Scope().Names[0]
		}
		return alias
	}

	// find alias based on tagged type.
	for _, alias := range File(ctx).LookupAlias(scopes.Type().TypeName()) {
		if alias != "" {
			return alias
		}
	}

	return ""
}

func GoPrimitiveTypeName(t midl.Kind) string {

	switch t {
	case midl.TypeWChar:
		return "uint16"
	case midl.TypeUChar, midl.TypeChar:
		return "byte"
	case midl.TypeError:
		return "uint32"
	case midl.TypeInt32_64:
		return "int64"
	case midl.TypeUint32_64:
		return "uint64"
	}

	return t.String()
}

func GoFieldName(ctx context.Context, field *midl.Field) string {
	return GoName(ctx, FieldName(field.Position, field.Name))
}
