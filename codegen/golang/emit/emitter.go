package emit

import (
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/oiweiwei/midl-gen-go/midl"
	"github.com/oiweiwei/midl-gen-go/midl/uuid"
	"github.com/oiweiwei/midl-gen-go/msdn/openspecs"
)

func FieldName(i int, n string) string {
	if n == "" {
		return "Field" + strconv.Itoa(i)
	}
	return n
}

// Options configures the generator.
type Options struct {
	// Session is the IDL session the files are loaded in. Required.
	Session *midl.Session
	// Log receives progress and warnings; defaults to discarding.
	Log *slog.Logger
	// EmitTrace adds trace comments (see Emitter.T) to the generated source.
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

// Emitter writes Go source: the part of the generator that does not know about
// the wire format. It holds the printing primitives, the naming of Go identifiers,
// the imports and the assembly of the output files.
type Emitter struct {
	Options

	// MSDN is the documentation of the IDL file being generated.
	MSDN *openspecs.MSDN
	// CheckErr emits the call with the check of its error.
	CheckErr func(...interface{})
	// Files are the buffers of the files being generated.
	Files []*FileBuffer

	out  *FileBuffer
	inCB bool // in comment block.
}

// Begin discards the output of a previous run, so that the emitter can be used
// for another IDL file.
func (p *Emitter) Begin() { p.out, p.Files = nil, nil }

// New returns the emitter for the options.
func New(opts Options) *Emitter {
	p := &Emitter{Options: opts}
	p.CheckErr = CheckErr1(p)
	return p
}

// Logger returns the generator logger (never nil).
func (p *Emitter) Logger() *slog.Logger {
	if p.Log == nil {
		p.Log = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	return p.Log
}

func (p *Emitter) Reset(ctx *Env, source interface{}, n ...string) {

	if p.out != nil {
		p.Files = append(p.Files, p.out)
	}

	if source == nil {
		return
	}

	switch source := source.(type) {
	case *midl.File:
		path := filepath.Join(append([]string{trimExt(source.Path)}, n...)...)
		p.out = NewFileBuffer(path, source.GoPkg)
		if len(n) == 0 && len(p.Files) == 0 {
			p.out.IsRoot = true
		}
	case *midl.Interface:
		path := filepath.Join(trimExt(ctx.File.Path), strings.ToLower(source.Name), source.Attrs.Version.String())
		p.out = NewFileBuffer(path, strings.ToLower(source.Name), n...)
	case *midl.Module:
		path := filepath.Join(trimExt(ctx.File.Path), strings.ToLower(source.Name))
		p.out = NewFileBuffer(path, strings.ToLower(source.Name), n...)
	}

	p.out.File = ctx.File
}

func (p *Emitter) P(args ...interface{}) {
	if p.inCB {
		fmt.Fprintln(p.out.Out, append([]interface{}{"//"}, args...)...)
	} else {
		fmt.Fprintln(p.out.Out, args...)
	}
}

func (p *Emitter) T(args ...interface{}) {
	if p.EmitTrace {
		fmt.Fprintln(p.out.Out, args...)
	}
}

func MSUnion(ctx *Env) bool {
	iff := ctx.Interface
	if iff != nil {
		return iff.Attrs.MSUnion
	}
	return false
}

func PointerDefault(ctx *Env) midl.PointerType {
	iff := ctx.Interface
	if iff != nil && iff.Attrs.PointerDefault != midl.PointerTypeNone {
		return iff.Attrs.PointerDefault
	}
	return midl.PointerTypeUnique
}

func (p *Emitter) XXX() string {
	return "xxx_"
}

func UUIDToGUID(u *uuid.UUID) string {

	return fmt.Sprintf("{Data1: 0x%08x, Data2: 0x%04x, Data3: 0x%04x, Data4: "+
		"[]byte{0x%02x, 0x%02x, 0x%02x, 0x%02x, 0x%02x, 0x%02x, 0x%02x, 0x%02x}}",
		u.TimeLow, u.TimeMid, u.TimeHiAndVersion,
		u.ClockSeqHiAndReserved, u.ClockSeqLow, u.Node[0], u.Node[1], u.Node[2], u.Node[3], u.Node[4], u.Node[5])

}
