package gen

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/oiweiwei/midl-gen-go/codegen/goprint"
	"github.com/oiweiwei/midl-gen-go/desc"
	"github.com/oiweiwei/midl-gen-go/logx"
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

type GeneratorOptions struct {
	UnmarshalRequest  bool
	UnmarshalResponse bool
}

// OutputFile is a generated file, not yet written to disk.
type OutputFile struct {
	// Dir is the directory relative to the output root.
	Dir string
	// Name is the file name.
	Name string
	// Content is the (formatted, if requested) Go source.
	Content []byte
}

// Path returns the path of the file relative to the output root.
func (f OutputFile) Path() string { return filepath.Join(f.Dir, f.Name) }

type Generator struct {
	out  *FileBuffer
	inCB bool // in comment block.
	// Session is the IDL session the files are loaded in. Required.
	Session *midl.Session
	// Log receives progress and warnings; defaults to discarding.
	Log *slog.Logger
	// EmitTrace adds trace comments (via T) to the generated source.
	EmitTrace            bool
	MultiFile            bool
	Format               bool
	ImportsPath          string
	Dir                  string
	MSDN                 *openspecs.MSDN
	MSDNCache            string
	MSDNIndexerFile      string
	MSDNIndexerExtraFile string
	Opts                 GeneratorOptions
	CheckErr             func(...interface{})
	Files                []*FileBuffer
}

// Logger returns the generator logger (never nil).
func (p *Generator) Logger() *slog.Logger {
	if p.Log == nil {
		p.Log = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	return p.Log
}

func trimExt(p string) string {
	return strings.TrimSuffix(p, filepath.Ext(p))
}

func (p *Generator) Reset(ctx *Env, source interface{}, n ...string) {

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

func (p *Generator) P(args ...interface{}) {
	if p.inCB {
		fmt.Fprintln(p.out.Out, append([]interface{}{"//"}, args...)...)
	} else {
		fmt.Fprintln(p.out.Out, args...)
	}
}

func (p *Generator) T(args ...interface{}) {
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

// Gen generates the Go sources for the IDL file and its libraries.
func (p *Generator) Gen(parent context.Context, fn string) ([]OutputFile, error) {

	if p.Session == nil {
		return nil, errors.New("gen: session is required")
	}

	p.CheckErr = CheckErr1(p)
	p.out, p.Files = nil, nil

	ctx := NewEnv(logx.With(parent, p.Logger()), p.Session)

	f, err := p.Session.Load(fn)
	if err != nil {
		return nil, err
	}

	if err := p.GenFile(ctx, fn, f); err != nil {
		return nil, err
	}

	for _, lib := range f.Libraries {
		if err := p.GenFile(ctx, fn, lib.Body); err != nil {
			return nil, err
		}
	}

	return p.Finalize(ctx)
}

func (p *Generator) GenFile(ctx *Env, fn string, f *midl.File) error {

	ctx = ctx.WithNamer(f.Namer())

	if f.IsEmpty() {
		return nil
	}

	if p.MSDNIndexerFile != "" {
		indexer, err := openspecs.NewProtocolIndexerFromFile(p.MSDNIndexerFile)
		if err != nil {
			return fmt.Errorf("msdn: openspecs: load indexer file: %w", err)
		}

		if p.MSDNIndexerExtraFile != "" {
			if err := indexer.ReadExtraFromFile(p.MSDNIndexerExtraFile); err != nil {
				return fmt.Errorf("msdn: openspecs: read extra indexer file: %w", err)
			}
		}

		p.MSDN = &openspecs.MSDN{CacheFS: p.MSDNCache, Indexer: indexer}

		if err := p.MSDN.SyncForFile(ctx, filepath.Base(fn)); err != nil {
			// A private / non-MS IDL (e.g. a reverse-engineered interface) has no OpenSpecs page; generate
			// it without doc comments instead of failing. Any other sync error is still fatal.
			if !errors.Is(err, openspecs.ErrProtocolNotFound) {
				return fmt.Errorf("msdn: openspecs: sync: %w", err)
			}
			p.Logger().Warn("msdn: generating without documentation", "file", fn, "err", err)
			p.MSDN = &openspecs.MSDN{}
		}
	} else {
		p.MSDN = &openspecs.MSDN{}
	}

	ctx = ctx.WithFile(f)

	p.Reset(ctx, f)

	for _, sym := range f.Exports() {
		if sym.Const != nil {
			p.GenConst(ctx, sym.Const)
		}
	}

	for _, sym := range f.Exports() {
		if sym.Type != nil {
			p.GenType(ctx, sym.Type)
		}
	}

	if len(f.ComClasses) > 0 || len(f.Libraries) > 0 {
		_ = p.GoPackageName(ctx, desc.OfType(p.Session.LookupType("CLSID")))
	}

	for _, sym := range f.ComClasses {
		if sym.Attrs.UUID != nil {
			p.P("//", sym.Name, "class", "identifier", sym.Attrs.UUID)
			p.P("var", GoName(ctx, sym.Name)+"ClassID", "=", "&dcom.ClassID", UUIDToGUID(sym.Attrs.UUID))
		}
	}

	for _, lib := range f.Libraries {
		if lib.Attrs.UUID != nil {
			p.P("//", lib.Name, "class", "identifier", lib.Attrs.UUID)
			p.P("var", GoName(ctx, lib.Name)+"LibraryID", "=", "&dcom.ClassID", UUIDToGUID(lib.Attrs.UUID))
		}
	}

	for _, iff := range f.Interfaces {
		if !iff.ForwardDeclarator {
			p.GenInterface(ctx, iff)
		}
	}

	for _, module := range f.Modules {
		if module.HasConstants() {
			p.GenModule(ctx, module)
		}
	}

	if f.IsDCOM() {
		p.Reset(ctx, f, "client")
		p.GenClientSet(ctx, f)
	}

	return nil
}

// Finalize renders the accumulated buffers into output files.
func (p *Generator) Finalize(ctx *Env) ([]OutputFile, error) {

	var (
		err error
		out []OutputFile
	)

	p.Reset(ctx, nil)

	for _, file := range p.Files {

		b := file.Reset()
		p.out = file

		ctx := ctx.WithFile(p.out.File)

		if p.out.IsRoot {
			p.P("//", "The", p.out.PackageName, "package", "implements", "the", strings.ToUpper(p.out.PackageName), "client", "protocol.")
			if intro, ok := p.MSDN.GetIntroduction(ctx); ok {
				p.P("//")
				p.P("//", "# Introduction")
				p.P("//")
				p.GenComment(ctx, intro.Documentation)
			}
			if overview, ok := p.MSDN.GetOverview(ctx); ok {
				p.P("//")
				p.P("//", "# Overview")
				p.P("//")
				p.GenComment(ctx, overview.Documentation)
			}
		}

		p.GenPackage(ctx, p.out.File)
		header := append([]byte("// Code generated by github.com/oiweiwei/midl-gen-go. DO NOT EDIT.\n\n"), file.Reset()...)

		if file.FileName == "" {
			p.GenPackageImportGuard(ctx)
		}

		b = append(file.Reset(), b...)

		for _, c := range file.Imports.Conflicts() {
			p.Logger().Warn("import conflict", "file", file.Path, "conflict", c.String())
		}

		if b, err = goprint.Assemble(header, b, file.Imports, p.Format); err != nil {
			return nil, fmt.Errorf("%s: %w", file.Path, err)
		}

		dir := strings.ReplaceAll(file.Path, "-", "_")

		base := file.FileName
		if base == "" {
			base = filepath.Base(dir)
		}

		out = append(out, OutputFile{Dir: dir, Name: base + ".go", Content: b})
	}

	return out, nil
}

// WriteFiles writes the generated files below dir, creating directories as needed.
func WriteFiles(log *slog.Logger, dir string, files []OutputFile) error {
	if log == nil {
		log = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	for _, f := range files {
		d := filepath.Join(dir, f.Dir)
		if err := os.MkdirAll(d, 0o777); err != nil {
			return err
		}
		path := filepath.Join(dir, f.Path())
		log.Debug("write file", "path", path, "size", len(f.Content))
		if err := os.WriteFile(path, f.Content, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func (p *Generator) XXX() string {
	return "xxx_"
}

func (p *Generator) GenModule(ctx *Env, module *midl.Module) {

	p.Reset(ctx, module)

	for _, sym := range module.Members {
		if sym.Const != nil {
			p.GenConst(ctx, sym.Const)
		}
	}

}

func (p *Generator) GenInterface(ctx *Env, iff *midl.Interface) {

	ctx = ctx.WithInterface(iff)

	p.Reset(ctx, iff)

	p.GenInterfaceID(ctx, iff)
	p.GenClientInterface(ctx, iff)

	for _, sym := range iff.Exports() {
		if sym.Const != nil {
			p.GenConst(ctx, sym.Const)
		}
	}

	for _, sym := range iff.Exports() {
		if sym.Type != nil {
			p.GenType(ctx, sym.Type)
		}
	}

	if iff.Attrs.UUID == nil {
		p.P("//", "XXX: no declaration")
		return
	}

	p.GenClient(ctx, iff)

	for _, op := range iff.Body.Operations {
		p.GenOperation(ctx, op)
	}

	p.Reset(ctx, iff, "server")

	p.GenServerInterface(ctx, iff)
	p.GenServerHandle(ctx, iff)
	p.GenUnplementedServer(ctx, iff)
}

func (p *Generator) GenConst(ctx *Env, c *midl.Const) {
	p.P()
	p.P("//", GoName(ctx, c.Name), "represents the", c.Name, "RPC constant")
	if midl.IsIntegerType(c.Type) && !c.Type.Signed() {
		p.P("const", GoName(ctx, c.Name), "=", p.FormatConstExpr(ctx, c.Value, midl.PrimitiveTypeSize(c.Type)))
		return
	}
	p.P("var", GoName(ctx, c.Name), "=", p.FormatConstExpr(ctx, c.Value, 0))
}

func (p *Generator) GenType(ctx *Env, t *midl.Type) {

	pp := p.NewTypeGenerator(ctx, t)

	if pp.Skip() {
		return
	}

	switch {
	case pp.Is(midl.TypeStruct):
		pp.GenStruct(ctx)
	case pp.Is(midl.TypeCUnion):
		pp.GenCUnion(ctx)
	case pp.Is(midl.TypeUnion):
		pp.GenUnion(ctx)
	case pp.Is(midl.TypeEnum):
		pp.GenEnum(ctx)
	case pp.Is(midl.TypeInterface):
		pp.GenInterface(ctx)
	case pp.Is(midl.TypePipe):
		pp.GenPipe(ctx)
	}
}

func (p *Generator) GenDoTrailingGapMarshalNDR(ctx *Env, gap int) {
	if gap > 1 {
		p.CheckErr(p.B("w.WriteTrailingGap", gap))
	}
}

func (p *Generator) GenDoAlignmentMarshalNDR(ctx *Env, alignment int) {
	if alignment > 1 {
		p.CheckErr(p.B("w.WriteAlign", alignment))
	}
}

func (p *Generator) GenDoUnionAlignemntMarshalNDR(ctx *Env, alignment int) {
	if alignment > 1 {
		p.CheckErr(p.B("w.WriteUnionAlign", alignment))
	}
}

func (p *Generator) GenDoTrailingGapUnmarshalNDR(ctx *Env, gap int) {
	if gap > 1 {
		p.CheckErr(p.B("w.ReadTrailingGap", gap))
	}
}

func (p *Generator) GenDoAlignmentUnmarshalNDR(ctx *Env, alignment int) {
	if alignment > 1 {
		p.CheckErr(p.B("w.ReadAlign", alignment))
	}
}

func (p *Generator) GenDoUnionAlignemntUnmarshalNDR(ctx *Env, alignment int) {
	if alignment > 1 {
		p.CheckErr(p.B("w.ReadUnionAlign", alignment))
	}
}

/*
When no pointer attribute is associated with a pointer that is a member
of a structure or union, the MIDL compiler assigns pointer attributes
using the following priority rules (1 is highest):

1. Attributes explicitly applied to the pointer parameter or member
2. Attributes explicitly applied to the pointer type (typedef)
3. (?) The pointer_default attribute in the IDL file that defines the type
4. The pointer_default attribute in the IDL file that imports the type
5. Unique (Microsoft RPC default mode)
*/
// Deprecated: unused, see EffectivePointer.
func (p *Generator) GenPointerType(ctx *Env, scopes *Scopes) midl.PointerType {

	var ptr midl.PointerType

	if scopes.IsTopLevelPointer() {
		if ptr = scopes.Layer().Pointer; ptr != midl.PointerTypeNone {
			return ptr
		}
	}

	if iff := ctx.Interface; iff != nil {
		if ptr = iff.Attrs.PointerDefault; ptr != midl.PointerTypeNone {
			return ptr
		}
	}

	return midl.PointerTypeUnique
}

func (p *Generator) GenWriteSize(ctx *Env, sz interface{}) {
	p.CheckErr(p.B("w.WriteSize", sz))
}

func (p *Generator) CB(f func()) {
	p.inCB = true
	f()
	p.inCB = false
}

func (p *Generator) GenCommentLine(ctx *Env, line string) {
	p.GenComment(ctx, []string{line})
}

func (p *Generator) GenComment(ctx *Env, docs []string) {
	p.CB(func() {
		for _, doc := range docs {
			for _, ddoc := range strings.Split(doc, "\n") {
				ddoc = strings.ReplaceAll(ddoc, "… ", "... ")
				if strings.Contains(ddoc, "|") || strings.Contains(ddoc, "+---") {
					p.P("\t", ddoc)
					continue
				}
				line := ""
				for _, word := range strings.Split(ddoc, " ") {
					line += word + " "
					if len(line) > 80 {
						p.P(line)
						line = ""
					}
				}
				if len(line) > 0 {
					p.P(line)
				}
			}
			p.P()
		}
	})
}

func (p *Generator) GenTag(ctx *Env, field *midl.Field) string {
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

func UUIDToGUID(u *uuid.UUID) string {

	return fmt.Sprintf("{Data1: 0x%08x, Data2: 0x%04x, Data3: 0x%04x, Data4: "+
		"[]byte{0x%02x, 0x%02x, 0x%02x, 0x%02x, 0x%02x, 0x%02x, 0x%02x, 0x%02x}}",
		u.TimeLow, u.TimeMid, u.TimeHiAndVersion,
		u.ClockSeqHiAndReserved, u.ClockSeqLow, u.Node[0], u.Node[1], u.Node[2], u.Node[3], u.Node[4], u.Node[5])

}
