package expr

import (
	"fmt"
	"math/big"
)

// ValueKind is the dynamic type of a Value.
type ValueKind uint8

// Value kinds.
const (
	ValueNull ValueKind = iota
	ValueInt
	ValueString
	ValueChar
	ValueBool
	ValueFloat
)

// Value is a literal: NULL, an integer of arbitrary size, a string, a
// character, a boolean or a floating point number. The zero Value is NULL.
type Value struct {
	kind ValueKind
	i    *big.Int
	f    *big.Float
	s    string
	r    rune
	b    bool
}

// Kind returns the dynamic type of the value.
func (v Value) Kind() ValueKind { return v.kind }

// IsNull reports whether the value is NULL.
func (v Value) IsNull() bool { return v.kind == ValueNull }

// Int returns the integer, if the value is one.
func (v Value) Int() (*big.Int, bool) { return v.i, v.kind == ValueInt && v.i != nil }

// Str returns the string, if the value is one.
func (v Value) Str() (string, bool) { return v.s, v.kind == ValueString }

// Char returns the character, if the value is one.
func (v Value) Char() (rune, bool) { return v.r, v.kind == ValueChar }

// Float returns the floating point number, if the value is one.
func (v Value) Float() (*big.Float, bool) { return v.f, v.kind == ValueFloat && v.f != nil }

// Truth returns the C truth value: non-zero numbers, non-empty strings and
// true are true; NULL is false.
func (v Value) Truth() bool {
	switch v.kind {
	case ValueInt:
		return v.i != nil && v.i.Sign() != 0
	case ValueFloat:
		return v.f != nil && v.f.Sign() != 0
	case ValueString:
		return v.s != ""
	case ValueChar:
		return v.r != 0
	case ValueBool:
		return v.b
	}
	return false
}

// Any returns the underlying Go value: nil, *big.Int, *big.Float, string,
// rune or bool.
func (v Value) Any() any {
	switch v.kind {
	case ValueInt:
		return v.i
	case ValueFloat:
		return v.f
	case ValueString:
		return v.s
	case ValueChar:
		return v.r
	case ValueBool:
		return v.b
	}
	return nil
}

// String returns the literal as it is spelled in the generated code: strings are
// quoted, a character is its numeric code.
func (v Value) String() string {
	switch v.kind {
	case ValueNull:
		return "NULL"
	case ValueString:
		return fmt.Sprintf("%q", v.s)
	}
	// a character is its code, as in C.
	return fmt.Sprint(v.Any())
}
