package golang

import (
	"fmt"
	"sort"
	"strings"

	"github.com/oiweiwei/midl-gen-go/codegen/golang/emit"
	"github.com/oiweiwei/midl-gen-go/midl"
)

// Phase is a point in the generation of an interface at which a feature emits
// code. The core (interface identifiers, constants, types, the request and
// response of each operation) is always generated; features add to it at these
// points, in the order they are registered.
type Phase int

const (
	// BeforeTypes is after the interface identifiers and before the constants and
	// types of the interface.
	BeforeTypes Phase = iota
	// AfterTypes is after the constants and types and before the operations. It is
	// not reached for an interface without a UUID, which has no operations.
	AfterTypes
	// InFile is the generation of the file of the feature, see FileNamer: after the
	// operations, in a file of its own.
	InFile
)

// Feature is a part of the generated code that can be switched on and off.
// See FEATURES.md for how to add one.
type Feature interface {
	// Name is the name the feature is selected by, e.g. "client".
	Name() string
	// Doc is a one line description for the feature listing.
	Doc() string
	// Requires lists the features that must be enabled for this one to work.
	Requires() []string
	// Phases lists the phases of an interface the feature emits code in.
	Phases() []Phase
	// Interface emits the code of the feature for the interface in the phase.
	Interface(ctx *emit.Env, p *Generator, iff *midl.Interface, phase Phase)
}

// FileNamer is implemented by a feature that has the InFile phase: the name of
// its file, e.g. "server" for server.go. The generator starts the file before
// calling the feature.
type FileNamer interface {
	FileName() string
}

// FileFeature is a feature that also emits code once per IDL file, after all of
// its interfaces; it starts its own file if it needs one.
type FileFeature interface {
	Feature
	File(ctx *emit.Env, p *Generator, f *midl.File)
}

var registry []Feature

// RegisterFeature adds the feature to the ones that can be selected. Features are
// emitted in the order they are registered. It panics on an empty or repeated
// name: that is a programming error, not an input error.
func RegisterFeature(f Feature) {
	if f.Name() == "" {
		panic("golang: feature without a name")
	}
	for _, r := range registry {
		if r.Name() == f.Name() {
			panic("golang: feature registered twice: " + f.Name())
		}
	}
	registry = append(registry, f)
}

// Features returns the registered features in the order they are emitted.
func Features() []Feature { return append([]Feature(nil), registry...) }

func init() {
	RegisterFeature(clientFeature{})
	RegisterFeature(serverFeature{})
}

// ParseFeatures turns a feature selection into the list of feature names. The
// selection is a comma separated list: "all" selects every feature, a name adds
// a feature, "-name" removes one. The empty selection is "all", and so is one that
// starts with a removal. The result is never nil, and may be empty: "none" selects
// no feature, only the core.
//
//	client,server    both
//	all,-server      everything but the server
//	-server          the same
//	client           the client only
//	none             neither
func ParseFeatures(spec string) ([]string, error) {

	chosen := map[string]bool{}

	known := map[string]bool{}
	for _, f := range registry {
		known[f.Name()] = true
	}

	toks := strings.Split(spec, ",")
	if first := strings.TrimSpace(toks[0]); first == "" || strings.HasPrefix(first, "-") {
		for n := range known {
			chosen[n] = true
		}
	}

	for _, tok := range toks {
		tok = strings.TrimSpace(tok)
		switch {
		case tok == "" || tok == "none":
		case tok == "all":
			for n := range known {
				chosen[n] = true
			}
		case strings.HasPrefix(tok, "-"):
			if !known[tok[1:]] {
				return nil, unknownFeature(tok[1:])
			}
			delete(chosen, tok[1:])
		default:
			if !known[tok] {
				return nil, unknownFeature(tok)
			}
			chosen[tok] = true
		}
	}

	names := make([]string, 0, len(chosen))
	for n := range chosen {
		names = append(names, n)
	}
	sort.Strings(names)

	return names, nil
}

func unknownFeature(name string) error {
	var names []string
	for _, f := range registry {
		names = append(names, f.Name())
	}
	return fmt.Errorf("unknown feature %q (features: %s)", name, strings.Join(names, ", "))
}

// selectFeatures returns the features with the names, in registration order; nil
// names are all features, empty names are none. It fails on an unknown name or a
// missing dependency.
func selectFeatures(names []string) ([]Feature, error) {

	if names == nil {
		return Features(), nil
	}

	want := map[string]bool{}
	for _, n := range names {
		want[n] = true
	}

	var out []Feature
	for _, f := range registry {
		if want[f.Name()] {
			out = append(out, f)
			delete(want, f.Name())
		}
	}

	for n := range want {
		return nil, unknownFeature(n)
	}

	have := map[string]bool{}
	for _, f := range out {
		have[f.Name()] = true
	}
	for _, f := range out {
		for _, r := range f.Requires() {
			if !have[r] {
				return nil, fmt.Errorf("feature %q requires feature %q", f.Name(), r)
			}
		}
	}

	return out, nil
}

// phase runs the features that take part in the phase for the interface.
func (p *Generator) phase(ctx *emit.Env, iff *midl.Interface, ph Phase) {
	for _, f := range p.features {
		for _, x := range f.Phases() {
			if x == ph {
				f.Interface(ctx, p, iff, ph)
			}
		}
	}
}

// files runs the features that have a file of their own for the interface.
func (p *Generator) files(ctx *emit.Env, iff *midl.Interface) {
	for _, f := range p.features {
		for _, x := range f.Phases() {
			if x != InFile {
				continue
			}
			name := ""
			if fn, ok := f.(FileNamer); ok {
				name = fn.FileName()
			}
			p.Reset(ctx, iff, name)
			f.Interface(ctx, p, iff, InFile)
		}
	}
}
