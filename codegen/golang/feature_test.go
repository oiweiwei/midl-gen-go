package golang

import (
	"context"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/oiweiwei/midl-gen-go/codegen/golang/emit"
	"github.com/oiweiwei/midl-gen-go/midl"
)

func TestParseFeatures(t *testing.T) {
	for _, tc := range []struct {
		spec string
		want []string
	}{
		{"", []string{"client", "server"}},
		{"all", []string{"client", "server"}},
		{"client,server", []string{"client", "server"}},
		{"client", []string{"client"}},
		{"all,-server", []string{"client"}},
		{"-server", []string{"client"}}, // a leading removal starts from all.
		{" server , client ", []string{"client", "server"}},
		{"none", []string{}},
		{"client,-client", []string{}},
	} {
		got, err := ParseFeatures(tc.spec)
		if err != nil {
			t.Errorf("%q: %v", tc.spec, err)
			continue
		}
		if got == nil || !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%q: got %#v, want %#v", tc.spec, got, tc.want)
		}
	}

	for _, spec := range []string{"clinet", "-clinet", "client,wat"} {
		if _, err := ParseFeatures(spec); err == nil || !strings.Contains(err.Error(), "client, server") {
			t.Errorf("%q: want an error that lists the features, got %v", spec, err)
		}
	}
}

type fakeFeature struct{ requires []string }

func (fakeFeature) Name() string                                            { return "fake" }
func (fakeFeature) Doc() string                                             { return "" }
func (f fakeFeature) Requires() []string                                    { return f.requires }
func (fakeFeature) Phases() []Phase                                         { return nil }
func (fakeFeature) Interface(*emit.Env, *Generator, *midl.Interface, Phase) {}

func TestFeatureRequirements(t *testing.T) {
	saved := registry
	defer func() { registry = saved }()

	RegisterFeature(fakeFeature{requires: []string{"client"}})

	if _, err := selectFeatures([]string{"fake"}); err == nil || !strings.Contains(err.Error(), `requires feature "client"`) {
		t.Errorf("a missing dependency must be reported, got %v", err)
	}
	if fs, err := selectFeatures([]string{"fake", "client"}); err != nil || len(fs) != 2 {
		t.Errorf("got %v, %v", fs, err)
	}

	defer func() {
		if recover() == nil {
			t.Error("registering a feature twice must panic")
		}
	}()
	RegisterFeature(fakeFeature{})
}

func generate(t *testing.T, features []string) map[string]string {
	t.Helper()
	dir, err := filepath.Abs("../../testdata/override")
	if err != nil {
		t.Fatal(err)
	}
	g, err := New(Options{Session: midl.NewSession(midl.WithInclude(dir)), ImportsPath: "x/", Format: true, Features: features})
	if err != nil {
		t.Fatal(err)
	}
	files, err := g.Gen(context.Background(), filepath.Join(dir, "ov.idl"))
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]string{}
	for _, f := range files {
		out[filepath.Base(f.Name)] = string(f.Content)
	}
	return out
}

func TestGenerationFollowsTheSelection(t *testing.T) {
	all := generate(t, nil)
	join := func(m map[string]string) string {
		var b strings.Builder
		for _, s := range m {
			b.WriteString(s)
		}
		return b.String()
	}

	for _, tc := range []struct {
		name           string
		features       []string
		server, client bool
		serverFile     bool
	}{
		{"all", nil, true, true, true},
		{"explicit all", []string{"client", "server"}, true, true, true},
		{"client", []string{"client"}, false, true, false},
		{"server", []string{"server"}, true, false, true},
		{"none", []string{}, false, false, false},
	} {
		files := generate(t, tc.features)
		src := join(files)

		if got := strings.Contains(src, "OvServer"); got != tc.server {
			t.Errorf("%s: server code present = %v, want %v", tc.name, got, tc.server)
		}
		if got := strings.Contains(src, "OvClient"); got != tc.client {
			t.Errorf("%s: client code present = %v, want %v", tc.name, got, tc.client)
		}
		if _, got := files["server.go"]; got != tc.serverFile {
			t.Errorf("%s: server.go present = %v, want %v", tc.name, got, tc.serverFile)
		}
		// the core is always there: the types and the request and response of the operation.
		for _, core := range []string{"type Buf struct", "type StatsRequest struct", "type StatsResponse struct", "OvSyntaxUUID"} {
			if !strings.Contains(src, core) {
				t.Errorf("%s: the core must be generated, missing %q", tc.name, core)
			}
		}
	}

	// selecting everything by name is the same as the default.
	explicit := generate(t, []string{"client", "server"})
	if !reflect.DeepEqual(all, explicit) {
		t.Error("an explicit selection of all features must generate the same files as the default")
	}
}

func TestUnknownFeatureFailsNew(t *testing.T) {
	if _, err := New(Options{Features: []string{"nope"}}); err == nil {
		t.Fatal("an unknown feature must be an error")
	}
}
