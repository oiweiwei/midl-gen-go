package midl

// dims.go: the dimensional information of an array, derived from the size_is, max_is,
// min_is, length_is, first_is and last_is attributes of a field.

import "github.com/oiweiwei/midl-gen-go/midl/expr"

type Dim struct {
	MinIs            Expr
	MaxIs            Expr
	SizeIs           Expr
	FirstIs          Expr
	LastIs           Expr
	LengthIs         Expr
	IsString         bool
	IsNullTerminated bool
	IsMultiSize      bool
	IsUTF8           bool
	Dimension        int
	NoSizeLimit      bool
}

type Size struct {
	MinIs  Expr
	MaxIs  Expr
	SizeIs Expr
}

func (d Size) Empty() bool {
	return d.MinIs.IsEmpty() && d.MaxIs.IsEmpty() && d.SizeIs.IsEmpty()
}

// Is returns the number of elements: size_is, or max_is - min_is + 1.
func (d Size) Is() Expr {
	if max := d.MaxIs; !max.IsEmpty() {
		// an operation that fails on a malformed operand (e.g. a string) is skipped.
		if !d.MinIs.IsEmpty() {
			if n, err := expr.Binary(expr.OpSub, max, d.MinIs); err == nil {
				max = n
			}
		}
		if n, err := expr.Binary(expr.OpAdd, max, expr.Int64(1)); err == nil {
			max = n
		}
		return max
	}
	return d.SizeIs
}

func (d Dim) Size() Size {
	return Size{d.MinIs, d.MaxIs, d.SizeIs}
}

func (d Dim) Empty() bool {
	return d.MinIs.IsEmpty() && d.MaxIs.IsEmpty() && d.SizeIs.IsEmpty() &&
		d.FirstIs.IsEmpty() && d.LastIs.IsEmpty() && d.LengthIs.IsEmpty()
}
