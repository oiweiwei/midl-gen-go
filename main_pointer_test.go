package main

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/oiweiwei/midl-gen-go/codegen/gen"
	"github.com/oiweiwei/midl-gen-go/midl"
)

var readRe = regexp.MustCompile(`w\.Read(Full)?Pointer(?:WithHook)?\(&o\.([A-Za-z0-9]+)`)

// F marks a pointer read with ReadFullPointer, U the plain ReadPointer, in the order of the generated text
// (the pointee read precedes the pointer it belongs to). The expectations follow the MIDL oracle of
// harness/lab/kb/pointer-audit (fx_*.idl: 0 mismatches): a use-site attribute wins over the typedef
// attribute, pointer_default(ptr) makes unannotated embedded pointers and pointers below the first
// level of an operation parameter full, the first level of a parameter is [ref].
var pointerGolden = map[string]string{
	"fx_ptr": "F:A U:B U:C F:D F:E U:F F:G U:H U:I F:J F:Pp F:Pp F:Ppl F:Ppl U:U F:P F:Pl U:Pu U:R F:S2 F:Pp F:Pq F:Pq F:Pr2 U:Pr2",
	"fx_ref": "U:A U:B U:C F:D F:E U:F U:G U:H U:I F:J U:Pp U:Pp F:Ppl U:Ppl U:U F:P F:Pl U:Pu U:R F:S2 U:Pp U:Pq F:Pq F:Pr2 U:Pr2",
	"fx_uni": "U:A U:B U:C F:D F:E U:F U:G U:H U:I F:J U:Pp U:Pp F:Ppl U:Ppl U:U F:P F:Pl U:Pu U:R F:S2 U:Pp U:Pq F:Pq F:Pr2 U:Pr2",
	// embedded pointer, [unique]/[ref] overrides, typedef chain, typedef [unique], array of pointers,
	// pointer to pointer, union arms (default, [unique], struct pointer), parameters.
	"fx_mix": "F:Emb U:Uni U:Rf F:Chain U:Tu F:Arr F:Pp F:Pp F:Value U:Value F:Uc F:C F:Pc F:C",
}

func TestFullPointerReads(t *testing.T) {
	dir, err := filepath.Abs("testdata/pointer")
	if err != nil {
		t.Fatal(err)
	}
	sess := midl.NewSession(midl.WithInclude(dir))

	out := t.TempDir()
	for name, want := range pointerGolden {
		p := &gen.Generator{Session: sess, ImportsPath: "x/", Format: false, Dir: out}
		files, err := p.Gen(context.Background(), filepath.Join(dir, name+".idl"))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if err := gen.WriteFiles(nil, out, files); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		b, err := os.ReadFile(filepath.Join(out, name, name, "v1", "v1.go"))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(b), "ndr.PointerHook{_") || !strings.Contains(string(b), "ndr.PointerHook{OnSet: ") {
			t.Errorf("%s: hook literals must be keyed", name)
		}
		var got []string
		for _, m := range readRe.FindAllStringSubmatch(string(b), -1) {
			k := "U"
			if m[1] != "" {
				k = "F"
			}
			got = append(got, k+":"+m[2])
		}
		if g := strings.Join(got, " "); g != want {
			t.Errorf("%s:\n got %s\nwant %s", name, g, want)
		}
	}
}
