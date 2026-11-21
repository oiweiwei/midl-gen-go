package emit

import (
	"github.com/oiweiwei/midl-gen-go/codegen/goprint"
	"github.com/oiweiwei/midl-gen-go/midl"
)

// Import is a Go import, see goprint.Import.
type Import = goprint.Import

// GenPackageImportGuard emits the exported GoPackage variable of the package.
func (p *Emitter) GenPackageImportGuard(ctx *Env) {
	p.P("")
	p.P("var (")
	p.P("//", "import guard")
	p.P("GoPackage", "=", p.Q(ctx.File.GoPkg))
	p.P(")")
}

var (
	// HexImport is the import of encoding/hex.
	HexImport = Import{Path: "encoding/hex"}

	// DefaultImports are the packages the generated code refers to without the
	// generator registering them one by one. They are candidates: a file imports
	// only those it uses.
	DefaultImports = []Import{
		{Path: "context"},
		{Path: "fmt"},
		{Path: "unicode/utf16"},
		{Path: "strings"},
		{Path: "github.com/oiweiwei/go-msrpc/ndr"},
		{Path: "github.com/oiweiwei/go-msrpc/midl/uuid"},
		{Path: "github.com/oiweiwei/go-msrpc/dcerpc"},
	}
)

func (p *Emitter) GenPackage(ctx *Env, f *midl.File) {
	p.P("package", p.out.PackageName[lastSlash(p.out.PackageName)+1:])
}

func lastSlash(s string) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '/' {
			return i
		}
	}
	return -1
}

// AddImport registers a candidate import for the file being generated and
// returns the identifier the generated code must use to refer to the package.
// The import is written only if the code refers to it.
func (p *Emitter) AddImport(imp Import) string {
	return p.out.Imports.Add(imp)
}
