package ndr

import "github.com/oiweiwei/midl-gen-go/codegen/golang/emit"

func (p *Emitter) GenDoTrailingGapMarshalNDR(ctx *emit.Env, gap int) {
	if gap > 1 {
		p.CheckErr(p.B("w.WriteTrailingGap", gap))
	}
}

func (p *Emitter) GenDoAlignmentMarshalNDR(ctx *emit.Env, alignment int) {
	if alignment > 1 {
		p.CheckErr(p.B("w.WriteAlign", alignment))
	}
}

func (p *Emitter) GenDoUnionAlignemntMarshalNDR(ctx *emit.Env, alignment int) {
	if alignment > 1 {
		p.CheckErr(p.B("w.WriteUnionAlign", alignment))
	}
}

func (p *Emitter) GenDoTrailingGapUnmarshalNDR(ctx *emit.Env, gap int) {
	if gap > 1 {
		p.CheckErr(p.B("w.ReadTrailingGap", gap))
	}
}

func (p *Emitter) GenDoAlignmentUnmarshalNDR(ctx *emit.Env, alignment int) {
	if alignment > 1 {
		p.CheckErr(p.B("w.ReadAlign", alignment))
	}
}

func (p *Emitter) GenDoUnionAlignemntUnmarshalNDR(ctx *emit.Env, alignment int) {
	if alignment > 1 {
		p.CheckErr(p.B("w.ReadUnionAlign", alignment))
	}
}

func (p *Emitter) GenWriteSize(ctx *emit.Env, sz interface{}) {
	p.CheckErr(p.B("w.WriteSize", sz))
}
