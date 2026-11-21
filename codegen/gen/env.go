package gen

import (
	"context"

	"github.com/oiweiwei/midl-gen-go/gonames"
	"github.com/oiweiwei/midl-gen-go/midl"
)

// Env is the state of the emission at the point the generator is in: the file,
// interface and field being generated, whether it is an operation parameter,
// the namer in effect and so on. It replaces the values the generator used to
// hide in context keys.
//
// An Env is never modified: the With methods return a changed copy. It
// implements context.Context (the embedded one carries cancellation and the
// logger, see package logx), so it can be passed to the code that takes one.
type Env struct {
	context.Context

	// Session is the IDL session the files are loaded in.
	Session *midl.Session
	// File is the IDL file being generated.
	File *midl.File
	// Interface is the interface being generated.
	Interface *midl.Interface
	// Field is the field being generated.
	Field *midl.Field
	// Namer converts IDL names to Go names; nil is the default configuration.
	Namer *gonames.Namer
	// NullMask is the name of the null mask variable of the operation (empty if none).
	NullMask string
	// VarName is the name of the variable the value is read into or written from.
	VarName string
	// Owner is the IDL name of the struct, union, enum or operation whose
	// members are being named (see gonames Config.Override).
	Owner string
	// Operation is true while the parameters of an operation are generated,
	// where the outermost pointer defaults to [ref].
	Operation bool
	// Reserved is true while a reserved (unnamed) field is generated.
	Reserved bool
}

// NewEnv returns the initial state for generating in the session.
func NewEnv(ctx context.Context, s *midl.Session) *Env {
	return &Env{Context: ctx, Session: s}
}

// Clone returns a copy of the state.
func (e *Env) Clone() *Env {
	c := *e
	return &c
}

// WithSession returns the state with the session.
func (e *Env) WithSession(s *midl.Session) *Env { c := e.Clone(); c.Session = s; return c }

// WithFile returns the state with the file.
func (e *Env) WithFile(f *midl.File) *Env { c := e.Clone(); c.File = f; return c }

// WithInterface returns the state with the interface.
func (e *Env) WithInterface(iff *midl.Interface) *Env { c := e.Clone(); c.Interface = iff; return c }

// WithField returns the state with the field.
func (e *Env) WithField(f *midl.Field) *Env { c := e.Clone(); c.Field = f; return c }

// WithNamer returns the state with the namer; a nil namer leaves the current one in effect.
func (e *Env) WithNamer(n *gonames.Namer) *Env {
	if n == nil {
		return e
	}
	c := e.Clone()
	c.Namer = n
	return c
}

// WithNullMask returns the state with the null mask variable name.
func (e *Env) WithNullMask(mask string) *Env { c := e.Clone(); c.NullMask = mask; return c }

// WithVarName returns the state with the variable name.
func (e *Env) WithVarName(n string) *Env { c := e.Clone(); c.VarName = n; return c }

// WithOwner returns the state with the owner of the members that are named.
func (e *Env) WithOwner(owner string) *Env { c := e.Clone(); c.Owner = owner; return c }

// WithOperation returns the state for the parameters of an operation.
func (e *Env) WithOperation() *Env { c := e.Clone(); c.Operation = true; return c }

// WithReserved returns the state for a reserved field.
func (e *Env) WithReserved() *Env { c := e.Clone(); c.Reserved = true; return c }
