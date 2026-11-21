package golang

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/oiweiwei/midl-gen-go/codegen/golang/emit"
	"github.com/oiweiwei/midl-gen-go/desc"
	"github.com/oiweiwei/midl-gen-go/logx"
	"github.com/oiweiwei/midl-gen-go/midl"
	"github.com/oiweiwei/midl-gen-go/msdn/openspecs"
)

// Gen generates the Go sources for the IDL file and its libraries.
func (p *Generator) Gen(parent context.Context, fn string) ([]emit.OutputFile, error) {

	if p.Session == nil {
		return nil, errors.New("gen: session is required")
	}

	p.Begin()

	ctx := emit.NewEnv(logx.With(parent, p.Logger()), p.Session)

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

func (p *Generator) GenFile(ctx *emit.Env, fn string, f *midl.File) error {

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
			p.P("var", emit.GoName(ctx, sym.Name)+"ClassID", "=", "&dcom.ClassID", emit.UUIDToGUID(sym.Attrs.UUID))
		}
	}

	for _, lib := range f.Libraries {
		if lib.Attrs.UUID != nil {
			p.P("//", lib.Name, "class", "identifier", lib.Attrs.UUID)
			p.P("var", emit.GoName(ctx, lib.Name)+"LibraryID", "=", "&dcom.ClassID", emit.UUIDToGUID(lib.Attrs.UUID))
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

	for _, feature := range p.features {
		if ff, ok := feature.(FileFeature); ok {
			ff.File(ctx, p, f)
		}
	}

	return nil
}

func (p *Generator) GenModule(ctx *emit.Env, module *midl.Module) {

	p.Reset(ctx, module)

	for _, sym := range module.Members {
		if sym.Const != nil {
			p.GenConst(ctx, sym.Const)
		}
	}

}

func (p *Generator) GenInterface(ctx *emit.Env, iff *midl.Interface) {

	ctx = ctx.WithInterface(iff)

	p.Reset(ctx, iff)

	p.GenInterfaceID(ctx, iff)
	p.phase(ctx, iff, BeforeTypes)

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

	p.phase(ctx, iff, AfterTypes)

	for _, op := range iff.Body.Operations {
		p.GenOperation(ctx, op)
	}

	p.files(ctx, iff)
}

func (p *Generator) GenConst(ctx *emit.Env, c *midl.Const) {
	p.P()
	p.P("//", emit.GoName(ctx, c.Name), "represents the", c.Name, "RPC constant")
	if midl.IsIntegerType(c.Type) && !c.Type.Signed() {
		p.P("const", emit.GoName(ctx, c.Name), "=", p.FormatConstExpr(ctx, c.Value, midl.PrimitiveTypeSize(c.Type)))
		return
	}
	p.P("var", emit.GoName(ctx, c.Name), "=", p.FormatConstExpr(ctx, c.Value, 0))
}
