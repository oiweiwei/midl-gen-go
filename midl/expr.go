package midl

import (
	"fmt"
	"math/big"

	"github.com/oiweiwei/midl-gen-go/midl/expr"
)

// Expr is an IDL expression, see package expr.
type Expr = expr.Expr

// Exprs is a list of expressions.
type Exprs []Expr

// Coerce converts the constant expression to the range and representation of
// the IDL type kind, e.g. for "const short X = 5". It fails if the expression
// is not constant, has the wrong type or does not fit the type.
//
// Note: the integer range checks compare the bit length of the magnitude, so
// a value is accepted if it fits the unsigned range of the same width
// (0xFFFF is a valid short). The value is then wrapped to the signed range for
// the Go constant.
func Coerce(e Expr, kind Kind) (Expr, error) {

	if !e.IsConst() {
		return Expr{}, fmt.Errorf("cannot evaluate a value")
	}

	switch kind {
	case TypeInt8, TypeInt16, TypeInt32, TypeInt64:
		v, ok := e.Int()
		if !ok {
			return Expr{}, fmt.Errorf("cannot coerce the expression to the integer type")
		}
		bits := map[Kind]int{TypeInt8: 8, TypeInt16: 16, TypeInt32: 32, TypeInt64: 64}[kind]
		if v.BitLen() > bits {
			return Expr{}, fmt.Errorf("%d-bit integer overflow", bits)
		}
		return expr.Int(big.NewInt(v.Int64())), nil
	case TypeUChar, TypeUint8, TypeUint16, TypeUint32, TypeUint64:
		v, ok := e.Int()
		if !ok {
			return Expr{}, fmt.Errorf("cannot coerce the expression to the integer type")
		}
		bits := map[Kind]int{TypeUChar: 8, TypeUint8: 8, TypeUint16: 16, TypeUint32: 32, TypeUint64: 64}[kind]
		if v.BitLen() > bits {
			return Expr{}, fmt.Errorf("%d-bit integer overflow", bits)
		}
		return expr.Int(big.NewInt(0).SetUint64(v.Uint64())), nil
	case TypeFloat32, TypeFloat64:
		v, ok := e.Int()
		if !ok {
			return Expr{}, fmt.Errorf("cannot coerce the expression to the floating point type")
		}
		return expr.Float(big.NewFloat(0).SetInt(v)), nil
	case TypeString, TypeWChar:
		v, ok := e.Str()
		if !ok {
			return Expr{}, fmt.Errorf("cannot coerce the expression to the %s type", kind)
		}
		return expr.String(v), nil
	case TypeBoolean:
		return expr.Bool(e.Truth()), nil
	case TypeChar:
		v, ok := e.Char()
		if !ok {
			return Expr{}, fmt.Errorf("cannot coerce the expression to the char type")
		}
		return expr.Char(v), nil
	case TypeVoid:
		if !e.IsNull() {
			return Expr{}, fmt.Errorf("cannot coerce the expression to the void type")
		}
		return expr.Null(), nil
	}

	return Expr{}, fmt.Errorf("unsupported type %s", kind)
}
