// Package golang generates Go client and server stubs from IDL files.
//
// The generator is layered, each layer using only the ones below it:
//
//	golang       this package: the files of an IDL file and its interfaces, the
//	             features (client, server), the symbols listing
//	golang/ndr   the NDR marshalling of types and operations
//	golang/emit  Go source writing: printing, names, imports, output files
//
// What is generated beyond the core (types, operation requests and responses)
// is selected with Options.Features, see Feature and FEATURES.md.
package golang

import (
	"log/slog"

	"github.com/oiweiwei/midl-gen-go/codegen/golang/emit"
	"github.com/oiweiwei/midl-gen-go/codegen/golang/ndr"
	"github.com/oiweiwei/midl-gen-go/midl"
)

// Options configures the generator.
type Options struct {
	// Session is the IDL session the files are loaded in. Required.
	Session *midl.Session
	// Log receives progress and warnings; defaults to discarding.
	Log *slog.Logger
	// Features are the names of the features to generate (see ParseFeatures); nil
	// means all of them, an empty list none.
	Features []string
	// EmitTrace adds trace comments to the generated source.
	EmitTrace bool
	// Format runs gofmt on the generated files.
	Format bool
	// ImportsPath is the Go import path the generated packages are rooted at.
	ImportsPath string
	// MSDNCache is the directory the documentation pages are cached in.
	MSDNCache string
	// MSDNIndexerFile and MSDNIndexerExtraFile are the indexes of the documentation;
	// without MSDNIndexerFile the code is generated without documentation.
	MSDNIndexerFile, MSDNIndexerExtraFile string
}

// OutputFile is a generated file, not yet written to disk.
type OutputFile = emit.OutputFile

// WriteFiles writes the generated files below dir, see emit.WriteFiles.
var WriteFiles = emit.WriteFiles

// Generator generates the Go stubs of IDL files.
type Generator struct {
	*ndr.Emitter

	features []Feature
}

// New returns the generator for the options. It fails if a feature is unknown or
// is missing a feature it requires.
func New(opts Options) (*Generator, error) {

	features, err := selectFeatures(opts.Features)
	if err != nil {
		return nil, err
	}

	return &Generator{
		Emitter: ndr.New(emit.Options{
			Session:              opts.Session,
			Log:                  opts.Log,
			EmitTrace:            opts.EmitTrace,
			Format:               opts.Format,
			ImportsPath:          opts.ImportsPath,
			MSDNCache:            opts.MSDNCache,
			MSDNIndexerFile:      opts.MSDNIndexerFile,
			MSDNIndexerExtraFile: opts.MSDNIndexerExtraFile,
		}),
		features: features,
	}, nil
}
