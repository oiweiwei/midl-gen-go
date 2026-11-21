// Package golang generates Go client and server stubs from IDL files.
//
// The generator is layered, each layer using only the ones below it:
//
//	golang       this package: the files of an IDL file and its interfaces, the
//	             client and the server, the symbols listing
//	golang/ndr   the NDR marshalling of types and operations
//	golang/emit  Go source writing: printing, names, imports, output files
package golang

import (
	"github.com/oiweiwei/midl-gen-go/codegen/golang/emit"
	"github.com/oiweiwei/midl-gen-go/codegen/golang/ndr"
)

// Options configures the generator, see emit.Options.
type Options = emit.Options

// OutputFile is a generated file, not yet written to disk.
type OutputFile = emit.OutputFile

// WriteFiles writes the generated files below dir, see emit.WriteFiles.
var WriteFiles = emit.WriteFiles

// Generator generates the Go stubs of IDL files.
type Generator struct {
	*ndr.Emitter
}

// New returns the generator for the options.
func New(opts Options) *Generator {
	return &Generator{Emitter: ndr.New(opts)}
}
