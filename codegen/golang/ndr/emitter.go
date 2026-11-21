package ndr

import "github.com/oiweiwei/midl-gen-go/codegen/golang/emit"

// Emitter generates the Go code that marshals IDL types and operations with the
// NDR transfer syntax: structs, unions, enumerations, pipes, the request and
// response of each operation.
type Emitter struct {
	*emit.Emitter
}

// New returns the NDR emitter for the options.
func New(opts emit.Options) *Emitter {
	return &Emitter{Emitter: emit.New(opts)}
}
