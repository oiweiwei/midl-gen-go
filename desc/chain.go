// Package desc answers the question "what is the type of this field?" once and
// for all. The IDL gives the answer in pieces: a typedef chain whose layers carry
// attributes ([unique], [string], ...), and attributes of the use site
// ([size_is(n)], [ref], ...). Package desc combines them into a Chain: the
// sequence of pointers, arrays and base types a value really consists of, with
// array dimensions attached to the right array, implicit conformant arrays
// inserted after pointers that have a size_is, strings turned into arrays, and
// pointers numbered by depth.
//
// The result does not depend on the target language or the wire format, so
// the code generators and a dynamic (reflection-based) engine use the same
// chain. Rules specific to NDR are in package desc/ndr.
package desc

import (
	"github.com/oiweiwei/midl-gen-go/midl"
)

// Chain is the resolved type of a field, parameter or type expression.
type Chain struct {
	// Layers group the types by the attributes that apply to them.
	Layers []*Layer
}

// Layer is a run of nodes that share one set of type attributes.
type Layer struct {
	// Attr are the attributes of the layer.
	Attr *midl.TypeAttr
	// Nodes are the types of the layer, from the outermost to the innermost.
	Nodes []*Node
}

// Node is a type of the chain together with what is known about it from its
// position in the chain.
type Node struct {
	*midl.Type
	// Dim is the dimension of an array node: the size, length and string
	// properties that apply to it.
	Dim midl.Dim
	// Pointer is, for a pointer node, the number of pointers following it in
	// the chain plus itself: **p is 2 for the first pointer and 1 for the second.
	Pointer int
}

// ResolveType builds the chain of the type alone, i.e. without the attributes
// of a use site.
func ResolveType(t *midl.Type) *Chain {
	chain := &Chain{}
	for _, layer := range t.Layers() {
		l := &Layer{Attr: layer.Attr, Nodes: make([]*Node, len(layer.Types))}
		for i, typ := range layer.Types {
			l.Nodes[i] = &Node{Type: typ}
		}
		chain.Layers = append(chain.Layers, l)
	}
	return chain
}

// Resolve builds the chain of the field: its type combined with the
// attributes of the field.
func Resolve(f *midl.Field) *Chain {

	var ok bool

	// first determine if there are any dimensional attributes.
	dims, chain := f.Attrs.Dim(), ResolveType(f.Type)
	layers := chain.Layers
	if len(layers) == 0 {
		return chain
	}

	// count the number of pointers that should point to the conformant array:
	// every dimension needs an array, the existing ones are used first.
	ptrCount := len(dims)
	for _, layer := range layers {
		for _, node := range layer.Nodes {
			if node.Kind == midl.TypeArray && !node.Array.IsFixed() {
				ptrCount--
			}
		}
	}

	// overwrite the attributes.
	// this will override the pointer at the reference site.
	layers[0].Attr = layers[0].Attr.Merge(f.Attrs.TypeAttr())

	dimCount := len(dims)

	// assign sizes to proper dimensions.
	// if needed add arrays.
dim_loop:
	for i := 0; i < len(layers) && len(dims) > 0; i++ {

		for j := 0; j < len(layers[i].Nodes); j++ {

			switch node := layers[i].Nodes[j]; {
			case node.Is(midl.TypeArray):

				if node.Array.IsFixed() {
					continue
				}
				node.Dim = dims[len(dims)-dimCount]

			case node.Is(midl.TypePointer):

				if ptrCount <= 0 {
					// ignore this pointer since there is an array
					// to attach the sizing information.
					continue
				}
				ptrCount--

				if dims[len(dims)-dimCount].Empty() {
					// skip over the empty dimension. break the switch statement.
					break
				}

				// find next type in this or next layer.
				if i, j, ok = nextNode(layers, i, j); !ok {
					break dim_loop
				}

				// https://learn.microsoft.com/en-us/windows/win32/Midl/size-is
				// insert the absent array right after the pointer.
				insertArray(layers, i, j, dims[len(dims)-dimCount])

				j++

			default:
				continue
			}

			if dimCount--; dimCount <= 0 {
				break dim_loop
			}
		}
	}

	isString := f.IsString()
	isNullTerminated := isString
	isChar := false

	// go in reverse direction and determine where to set the string.
string_loop:
	for i := len(layers) - 1; i >= 0; i-- {
		for j := len(layers[i].Nodes) - 1; j >= 0; j-- {
			switch node := layers[i].Nodes[j]; {
			case node.Is(midl.TypeChar) || node.Is(midl.TypeWChar):
				// keep track of isChar, to avoid rendering pointer to char as a uint8.
				isString, isChar = true, node.Is(midl.TypeChar)
				if f.Attrs.Format.Rune {
					isString = false
				}
			case node.Is(midl.TypePointer):

				if !f.IsString() && (layers[i].Attr.Pointer == midl.PointerTypeRef || layers[i].Attr.Pointer == midl.PointerTypeRefWeak) {
					break string_loop
				}

				if !isString {
					break string_loop
				}

				// insert string pointing to the next element.
				if i, j, ok = nextNode(layers, i, j); !ok {
					break string_loop
				}

				insertArray(layers, i, j, midl.Dim{IsString: true, IsNullTerminated: isNullTerminated})

				break string_loop

			case node.Is(midl.TypeArray):

				if !isString {
					break string_loop
				}
				// break loop if we determined the non-null terminated string of
				// char (which is byte buffer, or fixed array).
				if !f.IsString() && (isChar || node.Array.IsFixed()) {
					break string_loop
				}

				// set is_string
				node.Dim.IsString = true
				node.Dim.IsNullTerminated = isNullTerminated
				break string_loop
			}
		}
	}

	// put proper dimension number.
	dimension := 0
	for _, layer := range layers {
		for _, node := range layer.Nodes {
			if node.Kind == midl.TypeArray {
				node.Dim.Dimension = dimension
				dimension++
			} else {
				dimension = 0
			}
		}
	}

	// put proper pointer dereference number.
	// (go in reverse order).
	// so that pointer(**ptr) -> 2, pointer(*ptr) -> 1
	ptrCount = 0
	for i, pointer := len(layers)-1, 1; i >= 0; i-- {
		for j := len(layers[i].Nodes) - 1; j >= 0; j-- {
			if layers[i].Nodes[j].Is(midl.TypePointer) {
				ptrCount++
				layers[i].Nodes[j].Pointer = pointer
				pointer++
			} else {
				pointer = 1
			}
		}
	}

	if ptrCount == 0 && (f.Attrs.Pointer == midl.PointerTypeUnique || f.Attrs.Pointer == midl.PointerTypePtr) {
		layers[0].Nodes = append([]*Node{{
			Type:    &midl.Type{Kind: midl.TypePointer, Elem: layers[0].Nodes[0].Type},
			Pointer: 1,
		}}, layers[0].Nodes...)
	}

	return chain
}

// nextNode returns the position of the node following (i, j).
func nextNode(layers []*Layer, i, j int) (int, int, bool) {
	j++
	for i < len(layers) && j >= len(layers[i].Nodes) {
		i++
		j = 0
	}
	return i, j, i < len(layers)
}

// insertArray inserts a conformant array with the dimension in front of the
// node at (i, j); the array's element is that node.
func insertArray(layers []*Layer, i, j int, dim midl.Dim) {
	nodes := layers[i].Nodes
	arr := &Node{
		Type: &midl.Type{
			Kind:  midl.TypeArray,
			Array: &midl.Array{Bound: midl.ArrayBound{Lower: -1}},
			Elem:  nodes[j].Type,
		},
		Dim: dim,
	}
	layers[i].Nodes = append(nodes[:j], append([]*Node{arr}, nodes[j:]...)...)
}
