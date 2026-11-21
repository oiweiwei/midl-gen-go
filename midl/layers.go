package midl

// Layer is a run of types that share the attributes of one typedef or use
// site, e.g. for "typedef [unique] long *PLONG; [ref] PLONG p;" the chain
// p -> PLONG -> long has two layers: the one of the use site ([ref]) and the
// one of the typedef ([unique]).
type Layer struct {
	// Attr are the attributes of the layer; the attributes of nested
	// TypeAttribute wrappers are merged, the outer one winning.
	Attr *TypeAttr
	// Types are the types of the layer, from the outermost to the innermost.
	Types []*Type
}

// Layers groups the chain of the type (t, t.Elem, t.Elem.Elem, ...) by the
// attributes that apply. The result is purely syntactic: pointer defaults,
// array dimensions and the like are not applied here (see package desc).
func (t *Type) Layers() []Layer {

	layers := []Layer{}

	var cur *Layer

	for t := t; t != nil; t = t.Elem {
		if t.Is(TypeAttribute) {
			attr := t.Attrs.Clone()
			for t = t.Elem; t.Is(TypeAttribute); t = t.Elem {
				attr = t.Attrs.Merge(attr)
			}
			if cur != nil {
				layers = append(layers, *cur)
			}
			cur = &Layer{Attr: attr}
		}

		if cur == nil {
			cur = &Layer{Attr: &TypeAttr{}}
		}

		cur.Types = append(cur.Types, t)
	}

	if cur != nil {
		layers = append(layers, *cur)
	}

	return layers
}
