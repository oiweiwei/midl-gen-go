package gonames

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func namer(t *testing.T, yaml string) *Namer {
	t.Helper()
	c, err := ParseConfig([]byte(yaml))
	if err != nil {
		t.Fatal(err)
	}
	return NewNamer(c)
}

func TestSplit(t *testing.T) {
	for in, want := range map[string][]string{
		"NtlmSsp":    {"Ntlm", "Ssp"},
		"NTLMSSP":    {"NTLMSSP"}, // a run of capitals is one lexem.
		"HTTPServer": {"HTTP", "Server"},
		"rpc_bind":   {"rpc", "bind"},
		"a__b_":      {"a", "b"},
		"Ipv6Addr":   {"Ipv6", "Addr"},
		"obSize":     {"ob", "Size"},
		"ID":         {"ID"},
		"":           nil,
	} {
		if got := Split(in); !reflect.DeepEqual(got, want) {
			t.Errorf("Split(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestDefaultNamer(t *testing.T) {
	// without rules every lexem is capitalized.
	if got := GoName(nil, "rpc_bind"); got != "RpcBind" {
		t.Errorf("got %s", got)
	}
	if got := GoNamePrivate(nil, "RpcBind"); got != "rpcBind" {
		t.Errorf("got %s", got)
	}
	if got := GoNamePrivate(nil, "Import"); got != "import1" {
		t.Errorf("a Go keyword must be avoided, got %s", got)
	}
	if got := GoSnakeCase(nil, "RpcBind"); got != "rpc_bind" {
		t.Errorf("got %s", got)
	}
	if got := GoName(nil, ""); got != "" {
		t.Errorf("got %q", got)
	}
}

// The cases below are the rules of README.md.
func TestRules(t *testing.T) {
	for _, tc := range []struct {
		name, yaml, in, want string
	}{
		{"rename", "rename: {hRpc: [Handle], SizeOfStruct: [Structure, Size]}", "SizeOfStruct", "StructureSize"},
		{"rename beats everything", "rename: {hRpc: [Handle]}\ntrim: {prefixAll: [h]}", "hRpc", "Handle"},
		{"prefixAll", "trim: {prefixAll: [FSRM_]}", "FSRM_FolderUsage", "FolderUsage"},
		{"suffix", "trim: {suffix: [W, Ex]}", "CreateFileW", "CreateFile"},
		{"suffix 2", "trim: {suffix: [W, Ex]}", "OpenEx", "Open"},
		{"split", "split: {addentry: [add, entry]}", "addentry", "AddEntry"},
		{"lookBehind keeps the replacement", "lookBehind: {h: {context: [context], rpc: [h]}}", "HContext", "Context"},
		{"lookBehind join", "lookBehind: {end: {point: [endpoint]}}", "EndPoint", "Endpoint"},
		{"lookBehind and abbr", "lookBehind: {d: {word: [dword]}}\nabbr: {dword: DWORD}", "DWord", "DWORD"},
		{"lookBehind drops both", "lookBehind: {a: {b: []}}", "X_A_B", "X"},
		{"rotate", "rotate: {ob: Offset}", "obSize", "SizeOffset"},
		{"rotate replaces size", "rotate: {cch: Length}", "cchSize", "Length"},
		{"rotate is case sensitive", "rotate: {ob: Offset}", "ObSize", "ObSize"},
		{"trim prefix", "trim: {prefix: [I, dw]}", "IFoo", "Foo"},
		{"trim prefix 2", "trim: {prefix: [I, dw]}", "dwSize", "Size"},
		{"trim prefix only first", "trim: {prefix: [I]}", "FooI", "FooI"},
		{"trim word", "trim: {word: [struct]}", "Foo_struct", "Foo"},
		{"abbr", "abbr: {acl: ACL, buf: Buffer}", "AclBuf", "ACLBuffer"},
		{"everything dropped", "trim: {prefix: [I]}", "I", "I"},
		{"context handle", "", "ContextHandleFoo", "Foo"},
		{"context handle in two words", "", "ContextHandle", "ContextHandle"},
		{"escaped", "abbr: {foo: FOO}", "%foo", "foo"},
	} {
		if got := GoName(namer(t, tc.yaml), tc.in); got != tc.want {
			t.Errorf("%s: GoName(%q) = %q, want %q", tc.name, tc.in, got, tc.want)
		}
	}
}

func TestAcronyms(t *testing.T) {
	for _, tc := range []struct {
		name, yaml, in, want string
	}{
		{"spelling", "acronyms: [ACL, DWORD]", "AclEntry", "ACLEntry"},
		{"mixed case spelling", "acronyms: [IPv6, IKEv1]", "ipv6_addr", "IPv6Addr"},
		// lexems are never joined: the words of a name stay what the lexer found.
		{"not joined", "acronyms: [DWORD]", "DWordValue", "DWordValue"},
		{"a trimmed prefix is not swallowed by an acronym", "acronyms: [PIN]\ntrim: {prefix: [p]}", "pIn", "In"},
		{"words stay apart for snake case", "acronyms: [NT, OWF, NTOWF]", "NtOwfPassword", "NTOWFPassword"},
		{"abbr wins over acronyms", "abbr: {acl: Access}\nacronyms: [ACL]", "AclEntry", "AccessEntry"},
		{"a plain word is not an acronym", "acronyms: [ACL]", "Acls", "Acls"},
		{"join is case insensitive", "acronyms: [DWORD]", "DWORDValue", "DWORDValue"},
	} {
		if got := GoName(namer(t, tc.yaml), tc.in); got != tc.want {
			t.Errorf("%s: GoName(%q) = %q, want %q", tc.name, tc.in, got, tc.want)
		}
	}
}

func TestReservedFields(t *testing.T) {
	for _, in := range []string{"Reserved", "Reserved1", "Unused", "dummy", "WordReserved"} {
		if got := GoName(nil, in); got != "_" {
			t.Errorf("GoName(%q) = %q, want _", in, got)
		}
	}
	if got := GoNameNoReserved(nil, "Reserved"); got != "Reserved" {
		t.Errorf("got %s", got)
	}
}

func TestMergeNames(t *testing.T) {
	if got := GoSnakeCase(namer(t, "acronyms: [NT, OWF, NTOWF]"), "NtOwfPassword"); got != "nt_owf_password" {
		t.Errorf("snake case keeps the words apart, got %s", got)
	}
	if got := GoMergeNames(nil, "Foo", "Bar"); got != "FooBar" {
		t.Errorf("got %s", got)
	}
}

func TestMerge(t *testing.T) {
	a := &Config{Abbr: map[string]string{"x": "X", "y": "Y"}, Trim: Trim{Words: []string{"a"}}}
	b := &Config{Abbr: map[string]string{"y": "YY"}, Trim: Trim{Words: []string{"b"}}}
	m := a.Merge(b)
	if m.Abbr["x"] != "X" || m.Abbr["y"] != "YY" {
		t.Errorf("maps: later entries replace earlier ones: %v", m.Abbr)
	}
	if am := (&Config{Acronyms: []string{"A"}}).Merge(&Config{Acronyms: []string{"B"}}); !reflect.DeepEqual(am.Acronyms, []string{"A", "B"}) {
		t.Errorf("acronyms are concatenated: %v", am.Acronyms)
	}
	if !reflect.DeepEqual(m.Trim.Words, []string{"a", "b"}) {
		t.Errorf("lists are concatenated: %v", m.Trim.Words)
	}
	if a.Abbr["y"] != "Y" || len(a.Trim.Words) != 1 {
		t.Error("Merge must not modify its receiver")
	}
	var nilCfg *Config
	if nilCfg.Merge(nil) == nil {
		t.Error("merging nothing is an empty config")
	}
}

func TestLoadConfigFromPath(t *testing.T) {
	dir := t.TempDir()
	write := func(name, content string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// two files define the same list: both must be in effect (a later file used to
	// replace the earlier one's list).
	write("a.yaml", "trim: {word: [foo]}\nabbr: {x: X}")
	write("b.yaml", "trim: {word: [bar]}\nabbr: {y: Y}")
	write("c.yaml", "# an empty document is not an error\n")
	write("ignored.txt", "not: [yaml")

	nr, err := LoadNamerFromPath(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := GoName(nr, "Keep_foo_bar_X_y"); got != "KeepXY" {
		t.Errorf("both trim.word lists must apply, got %s", got)
	}

	// a directory without rules (or without files) has no rules.
	if nr, err := LoadNamerFromPath(filepath.Join(dir, "missing")); err != nil || GoName(nr, "foo_bar") != "FooBar" {
		t.Errorf("missing directory: %v", err)
	}

	// broken YAML is reported, not panicked on.
	write("d.yaml", "trim: [unclosed")
	if _, err := LoadNamerFromPath(dir); err == nil {
		t.Error("invalid YAML must be an error")
	}
}
