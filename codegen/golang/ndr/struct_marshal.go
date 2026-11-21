package ndr

import (
	"fmt"

	"github.com/oiweiwei/midl-gen-go/codegen/golang/emit"
	"github.com/oiweiwei/midl-gen-go/desc"
	"github.com/oiweiwei/midl-gen-go/desc/ndr"
	"github.com/oiweiwei/midl-gen-go/midl"
)

func (p *TypeGenerator) GenStructNDRSizeInfo(ctx *emit.Env) {

	if !ndr.IsConformant(p.Cursor) {
		return
	}

	p.P()
	p.P("func", "(o *"+p.GoTypeName+")", "NDRSizeInfo() []uint64 {")

	last := p.Struct().LastField()
	if scopes := desc.Of(last); !scopes.Is(midl.TypeArray) {
		p.P("return", p.O(p.GoFieldName(ctx, last))+".NDRSizeInfo()")
	} else {

		var dims []string

		for i := 0; scopes.IsNotFixedArray(); scopes, i = scopes.Next(), i+1 {
			dims = append(dims, p.GenArraySizeVar(ctx, last, i, scopes, "Size", scopes.Dim().Size().Is()))
		}
		p.P("return []uint64{")
		for _, dim := range dims {
			p.P(dim, ",")
		}
		p.P("}")
	}

	p.P("}")
}

func (p *TypeGenerator) GenStructMarshalNDRSizePreamble(ctx *emit.Env, field *midl.Field) {
	if !ndr.IsConformant(p.Cursor) {
		return
	}
	p.P("sizeInfo, ok := ctx.Value(ndr.SizeInfo).([]uint64)")
	p.If("!ok", func() {
		p.P("sizeInfo", "=", p.O("NDRSizeInfo()"))
		p.Range("sz1", "sizeInfo", func() {
			p.GenWriteSize(ctx, p.IVar("sizeInfo", "sz1"))
		})
		p.P("ctx = context.WithValue(ctx, ndr.SizeInfo, sizeInfo)")
	})
}

func (p *TypeGenerator) GenStructMarshalNDR(ctx *emit.Env) {

	trailingField := p.Struct().LastFieldIndex()
	if ndr.IsConformant(p.Cursor) || ndr.IsVarying(p.Cursor) {
		trailingField--
	}

	p.Block("func", "(o *"+p.GoTypeName+")", "MarshalNDR(ctx context.Context, w ndr.Writer)", "error", func() {
		// prepare payload.
		p.CheckErr("o." + p.XXX() + "PreparePayload(ctx)")
		// write size information.
		p.GenStructMarshalNDRSizePreamble(ctx, p.StructLastField())
		// align the structure.
		p.GenDoAlignmentMarshalNDR(ctx, ndr.Alignment(p.Cursor))
		// marshal fields.
		for i, field := range p.Struct().Fields {
			// marshal field.
			p.GenFieldMarshalNDR(ctx, field, desc.Of(field))
			// add trailing gap.
			if i == trailingField {
				if f := desc.Of(field); ndr.PlainAlignment(f) != ndr.PlainAlignment(p.Cursor) {
					p.GenDoTrailingGapMarshalNDR(ctx, ndr.Alignment(p.Cursor))
				}
			}
		}
		if p.Layer().Pad != 0 {
			p.P("//", "pad", p.Layer().Pad)
			p.GenDoAlignmentMarshalNDR(ctx, int(p.Layer().Pad))
		}
		p.P("return nil")
	})

}

func (p *TypeGenerator) GenFieldMarshalNDR(ctx *emit.Env, field *midl.Field, scopes *desc.Cursor, index ...interface{}) {

	if scopes == nil || field.Attrs.Ignore {
		// end.
		return
	}

	// reserved field.
	if p.GoFieldName(ctx, field) == "_" {
		p.P("//", "reserved", field.Name)
		p.GenZeroFieldMarshalNDR(ctx.WithVarName(emit.ZeroLen), field, scopes, index...)
		return
	}

	// get variable name.
	name := p.VarName(ctx, index...)
	if name == "" {
		name = p.O(p.IVar(p.GoFieldName(ctx, field), index...))
	}

	switch {

	case scopes.Type().Is(midl.TypeHandle):

		p.P("//", "skip", "[handle]", field.Name)

	case scopes.Type().IsPrimitiveType():

		// marshal primitive type.

		if scopes.IsBool() && midl.PrimitiveTypeSize(scopes.Kind()) > 1 {
			typeName := emit.GoPrimitiveTypeName(scopes.Kind())
			p.If("!", name, func() {
				p.CheckErr(p.B("w.WriteData", p.B(typeName, "0")))
			}, p.Else(func() {
				p.CheckErr(p.B("w.WriteData", p.B(typeName, "1")))
			}))
		} else if scopes.Is(midl.TypeInt32_64) {
			p.CheckErr(p.B("w.WriteData", p.B("ndr.Int3264", name)))
		} else if scopes.Is(midl.TypeUint32_64) {
			p.CheckErr(p.B("w.WriteData", p.B("ndr.Uint3264", name)))
		} else if field.Attrs.Format.Rune {
			switch scopes.Kind() {
			case midl.TypeChar, midl.TypeUChar, midl.TypeUint8:
				p.CheckErr(p.B("w.WriteData", p.B("uint8", name)))
			case midl.TypeWChar, midl.TypeUint16, midl.TypeInt16:
				p.CheckErr(p.B("w.WriteData", p.B("uint16", name)))
			}
		} else {
			p.CheckErr(p.B("w.WriteData", name))
		}

	case scopes.Is(midl.TypeEnum):

		// marshal enum.

		p.CheckErr(p.B("w.WriteEnum", p.B(scopes.EnumType(), name)))

	case scopes.Is(midl.TypeUnion):

		// marshal union.

		switchVar := "_sw" + p.ToVar(name)
		if !scopes.Union().IsEncapsulated() {
			p.GenerateSwitchIs(ctx, field, scopes, switchVar)
		}

		p.If(name, "!=", "nil", func() {
			if !scopes.Union().IsEncapsulated() {
				p.CheckErr(p.B(name+".MarshalUnionNDR", "ctx", "w", switchVar))
			} else {
				p.CheckErr(p.B(name+".MarshalNDR", "ctx", "w"))
			}
		}, p.Else(func() {
			p.GenZeroUnionFieldMarshalNDR(ctx, field, scopes, index...)
		}))

	case scopes.Is(midl.TypeStruct), scopes.Is(midl.TypeInterface):

		// marshal structure.

		p.If(name, "!=", "nil", func() {
			p.CheckErr(p.B(name+".MarshalNDR", "ctx", "w"))
		}, p.Else(func() {
			p.GenZeroStructFieldMarshalNDR(ctx, field, scopes, index...)
		}))

	case scopes.Is(midl.TypePointer):

		// marshal pointer.

		var nullChk string

		if next := scopes.Next(); next.Type().IsPrimitiveType() {

			if field.Attrs.DefaultNull != nil {
				if len(field.Attrs.DefaultNull) > 0 {
					for _, expr := range field.Attrs.DefaultNull {
						chk := p.GenExpr(ctx, expr, p.LookupExprField(ctx, expr), "")
						if nullChk == "" {
							nullChk = fmt.Sprintf("!%s", chk)
						} else {
							nullChk = fmt.Sprintf("%s || !%s", nullChk, chk)
						}
					}
				} else {
					nullChk = fmt.Sprintf("%s != %v", name, p.GoTypeZeroValue(ctx, p.Layer(), field, scopes))
				}
			}

			if emit.HasNullMaskBit(ctx, field) {
				if nullChk != "" {
					nullChk += "&& "
				}
				nullChk += fmt.Sprintf("(%s & %s == 0 || %s != %s)", p.O(p.NullMask()), ctx.NullMask+emit.GoFieldName(ctx, field),
					name, p.GoTypeZeroValue(ctx, p.Layer(), field, scopes))
			}

			if nullChk == "" && !scopes.Layer().Pointer.IsWeak(ctx.Operation) {
				p.P("// XXX: pointer default mask was not generated as conditions")
				p.P("// for automatic generation were not met.")
				p.P("// If this behavior is not desired, use goext_default_null([cond]) attribute.")
			}
		}

		if nullChk != "" {
			p.If(nullChk, func() {
				fN := "_ptr_" + field.Name
				p.P(fN, ":=", "ndr.MarshalNDRFunc", "(", "func(ctx context.Context, w ndr.Writer) error {")
				p.GenFieldMarshalNDR(ctx, field, scopes.Next(), index...)
				p.P("return nil")
				p.P("})")
				p.CheckErr(p.B("w.WritePointer", p.Amp(name), fN))
			}, p.Else(func() {
				p.GenZeroPointerFieldMarshalNDR(ctx, field, scopes, index...)
			}))
		} else {

			sizeChk := ""
			if next := scopes.Next(); next.Is(midl.TypeArray) && !next.Dim().Size().Empty() {
				// immitate memory allocation for conformance array.
				sizeChk = fmt.Sprintf("|| %s > 0", p.GenExpr(ctx, next.Dim().Size().Is(), p.LookupExprField(ctx, next.Dim().Size().Is()), ""))
			}

			p.If(name, "!=", p.GoTypeZeroValue(ctx, p.Layer(), field, scopes), sizeChk, func() {
				fN := "_ptr_" + field.Name
				p.P(fN, ":=", "ndr.MarshalNDRFunc", "(", "func(ctx context.Context, w ndr.Writer) error {")
				p.GenFieldMarshalNDR(ctx, field, scopes.Next(), index...)
				p.P("return nil")
				p.P("})")
				p.CheckErr(p.B("w.WritePointer", p.Amp(name), fN))
			}, p.Else(func() {
				p.GenZeroPointerFieldMarshalNDR(ctx, field, scopes, index...)
			}))
		}

	case scopes.Is(midl.TypeArray):

		// marshal array.

		ctx := ctx.WithNullMask("") /* clear null mask as arrays are unsupported */

		if !p.GenMarshalSizeInfo(ctx, field, scopes, index...) {
			break
		}

		idx := p.Var("i", len(index)+1)

		if scopes.Dim().IsString {
			name = p.GenWriteStringBuffer(ctx, scopes, field, index...)
			ctx, index = ctx.WithVarName(name), nil
		}

		if scopes.Dim().NoSizeLimit {
			p.Range(idx, name, func() {
				p.P(idx, ":=", idx)
				p.GenFieldMarshalNDR(ctx, field, scopes.Next(), append(index, idx)...)
			})
			break
		}

		p.Range(idx, name, func() {
			// iterate over the array.
			p.P(idx, ":=", idx)
			if scopes.Array().IsFixed() && (!scopes.Dim().IsString || p.IsNotLastFixedArray(ctx, scopes, field)) {
				// break if index exceeds the fixed-array limit.
				p.If(p.B("uint64", idx), ">=", scopes.Array().Size(), func() {
					p.P("break")
				})
			} else {
				// break if index exceeds the conformance/varyingness limit.
				p.If(p.B("uint64", idx), ">=", p.IVar("sizeInfo", scopes.Dim().Dimension), func() {
					p.P("break")
				})
			}
			p.GenFieldMarshalNDR(ctx, field, scopes.Next(), append(index, idx)...)
		})

		p.GenZeroFieldMarshalNDR(ctx, field, scopes, index...)

	case scopes.Is(midl.TypePipe):

		p.P("//", "marshal", "pipe", field.Name)

		field := p.GenPipeField(ctx, scopes.Type())

		p.If(name, "!=", "nil", func() {
			p.Range("_chunk", name+".Pipe()", func() {
				p.GenFieldMarshalNDR(ctx.WithVarName("_chunk"), field, desc.Of(field))
			})
		})
		p.GenWriteSize(ctx, "0")

	default:
		p.P("//", "FIXME", "unknown type", field.Name)
	}
}

func (p *TypeGenerator) IsNotLastFixedArray(ctx *emit.Env, scopes *desc.Cursor, field *midl.Field) bool {
	return !ndr.IsTopLevelArray(scopes) && scopes.Array().IsFixed() && p.Struct() != nil && len(p.Struct().Fields) > 0 && field.Position != p.Struct().LastField().Position
}

func (p *TypeGenerator) GenMarshalSizeInfo(ctx *emit.Env, field *midl.Field, scopes *desc.Cursor, index ...interface{}) bool {

	// get variable name.
	name := p.VarName(ctx, index...)
	if name == "" {
		name = p.O(p.IVar(p.GoFieldName(ctx, field), index...))
	}

	if p.IsNotLastFixedArray(ctx, scopes, field) {
		return true
	}

	isConformant, isVarying := ndr.IsConformant(scopes), ndr.IsVarying(scopes)

	if field.Position == 0 || ndr.IsTopLevelArray(scopes) {

		// for non-top level array size information is handled via NDRSizeInfo.

		if dim := scopes.Dim(); dim.IsString && !dim.NoSizeLimit && dim.Size().Empty() && dim.LengthIs.IsEmpty() {

			// top-level string.

			if name == emit.ZeroLen {
				name = `""`
			}

			switch scopes.Next().Kind() {
			case midl.TypeWChar, midl.TypeUint16, midl.TypeInt16:
				if dim.IsNullTerminated {
					p.CheckErr(p.B("ndr.WriteUTF16NString", "ctx", "w", name))
				} else {
					p.CheckErr(p.B("ndr.WriteUTF16String", "ctx", "w", name))
				}
			case midl.TypeUChar, midl.TypeChar, midl.TypeUint8:
				if dim.IsNullTerminated {
					p.CheckErr(p.B("ndr.WriteCharNString", "ctx", "w", name))
				} else {
					p.CheckErr(p.B("ndr.WriteCharString", "ctx", "w", name))
				}
			}

			return false

			// else, string length should be retrieved from external parameters
			// size is, length is, and so on.
		}

		if isConformant {

			dims := []string{}
			for scopes, i := scopes, len(index); scopes.IsNotFixedArray(); scopes, i = scopes.Next(), i+1 {
				dims = append(dims, p.GenArraySizeVar(ctx, field, i, scopes, "Size", scopes.Dim().Size().Is()))
				// write max_count.
				p.GenWriteSize(ctx, dims[len(dims)-1])
			}

			p.Block("sizeInfo := []uint64", func() {
				for _, dim := range dims {
					p.P(dim, ",")
				}
			})
		}
	}

	if isVarying {

		if !isConformant {
			p.Block("sizeInfo := []uint64", func() {
				for scopes, i := scopes, len(index); scopes.IsNotFixedArray(); scopes, i = scopes.Next(), i+1 {
					p.P(0, ",")
				}
			})
		}

		for scopes, i := scopes, len(index); scopes.IsNotFixedArray(); scopes, i = scopes.Next(), i+1 {
			dim := p.GenArraySizeVar(ctx, field, i, scopes, "Length", scopes.Dim().LengthIs)
			if !isConformant {
				p.P(p.IVar("sizeInfo", scopes.Dim().Dimension), "=", dim)
			} else {
				p.If(dim, ">", p.IVar("sizeInfo", scopes.Dim().Dimension), func() {
					p.P(dim, "=", p.IVar("sizeInfo", scopes.Dim().Dimension))
				}, p.Else(func() {
					// reset the size information back for padding and etc.
					p.P(p.IVar("sizeInfo", scopes.Dim().Dimension), "=", dim)
				}))
			}
			// write offset.
			p.GenWriteSize(ctx, 0)
			// write actual_count.
			p.GenWriteSize(ctx, dim)
		}
	}

	return true
}

func (p *TypeGenerator) GenArraySizeVar(ctx *emit.Env, field *midl.Field, i int, scopes *desc.Cursor, name string, sizeExpr midl.Expr) string {

	if !scopes.IsNotFixedArray() {
		return ""
	}

	dimVar := p.Var("dim"+name, i+1)

	if !sizeExpr.IsEmpty() {
		p.P(dimVar, ":=", p.B("uint64", p.GenExpr(ctx, sizeExpr, p.LookupExprField(ctx, sizeExpr), "uint64")))
		// check minimum value borders.
		if to, val, ok := p.GetMinVarValue(ctx, sizeExpr); ok {
			p.If(p.GenExpr(ctx, to, p.LookupExprField(ctx, to), ""), "<", val, func() {
				p.P(dimVar, "=", "uint64(0)")
			})
		}
		return dimVar
	}

	if i == 0 {
		// short-path.
		// get variable name.
		name := p.VarName(ctx)
		if name == "" {
			name = p.O(emit.GoFieldName(ctx, field))
		}
		p.P(dimVar, ":=", p.DataLen(ctx, field, scopes, name))
		return dimVar
	}

	// calculate size on the fly.
	p.P(dimVar, ":=", "uint64(0)")
	p.GenRange(ctx, p.O(p.GoFieldName(ctx, field)), 0, i, func(ctx *emit.Env, n string) {
		lenVar := p.Var("len" + emit.Title(dimVar))
		p.If(lenVar, ":=", p.DataLen(ctx, field, scopes, n), ";", lenVar, ">", dimVar, func() {
			p.P(dimVar, "=", lenVar)
		})
	})

	return dimVar
}

// GenWriteStringBuffer function ...
func (p *TypeGenerator) GenWriteStringBuffer(ctx *emit.Env, scopes *desc.Cursor, field *midl.Field, index ...interface{}) string {

	name := p.O(p.IVar(p.GoFieldName(ctx, field), index...))

	// XXX: either use ZeroString as a pointer discriminator.
	// p.P(name, "=", p.B("strings.TrimPrefix", name, "ndr.ZeroString"))

	varName := p.BufVar(name)

	if field.Attrs.Format.MultiSize {
		p.P("var", varName, "[]uint16")
		p.Range("i1", name, func() {
			p.P(varName, "=", p.B("append", varName, p.B("utf16.Encode", p.B("[]rune", p.Var(name, "[i1]")))+"..."))
			p.P(varName, "=", p.B("append", varName, p.B("uint16", "0")))
		})
		p.P(varName, "=", p.B("append", varName, p.B("uint16", "0")))
		return varName
	}

	switch scopes := scopes.Next(); scopes.Kind() {
	case midl.TypeWChar, midl.TypeUint16, midl.TypeInt16:
		p.P(varName, ":=", p.B("utf16.Encode", p.B("[]rune", name)))
	case midl.TypeChar, midl.TypeUChar, midl.TypeInt8, midl.TypeUint8:
		p.P(varName, ":=", p.B("[]byte", name))
	default:
		p.P(`return errors.New("cannot handle ` + scopes.Kind().String() + ` to string type conversion")`)
	}

	if scopes.Dim().NoSizeLimit {
		// skip for unlimited size types.
		return varName
	}

	if scopes.Dim().IsNullTerminated {

		// declared with [string] attribute. size written should be one less
		// to include zero.

		if scopes.Array().IsFixed() {
			size := scopes.Array().Size()
			p.If(p.B("uint64", p.Len(varName)), ">", size, "-", "1", func() {
				p.P(varName, "=", varName+fmt.Sprintf("[:%d-1]", size))
			})
		} else {
			sizeInfo := p.IVar("sizeInfo", scopes.Dim().Dimension)
			p.If(p.B("uint64", p.Len(varName)), ">", sizeInfo, "-", "1", func() {
				p.P(varName, "=", varName+p.IVar("", ":"+sizeInfo+"-1"))
			})
		}

		zeroValue := p.B(emit.GoPrimitiveTypeName(scopes.Next().Kind()), "0")

		// append zeroValue only if string is not zero.
		p.If(name, "!=", "ndr.ZeroString", func() {
			p.P(varName, "=", p.B("append", varName, zeroValue))
		})

	} else {

		if scopes.Array().IsFixed() {
			size := scopes.Array().Size()
			p.If(p.B("uint64", p.Len(varName)), ">", size, func() {
				p.P(varName, "=", varName+fmt.Sprintf("[:%d]", size))
			})
		} else {
			sizeInfo := p.IVar("sizeInfo", scopes.Dim().Dimension)
			p.If(p.B("uint64", p.Len(varName)), ">", sizeInfo, func() {
				p.P(varName, "=", varName+p.IVar("", ":"+sizeInfo))
			})
		}
	}

	return varName
}
