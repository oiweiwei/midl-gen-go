package gen

import (
	"bytes"
	"context"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/oiweiwei/midl-gen-go/midl"
)

// Compile and execute the actual emitted client method with a small in-memory
// transport. Parsing the IDL exercises typedef resolution as well as generation.
func TestGeneratedClientReturnStatus(t *testing.T) {
	for _, tc := range []struct {
		name, idlType, goType string
		hresult               bool
	}{
		{"HRESULT", "HRESULT", "int32", true},
		{"alias", "RESULT_ALIAS", "int32", true},
		{"alias_chain", "RESULT_CHAIN", "int32", true},
		{"LONG", "long", "int32", false},
		{"DWORD", "DWORD", "uint32", false},
		{"WIN32_ERROR", "WIN32_ERROR", "uint32", false},
		{"RPC_STATUS", "RPC_STATUS", "uint32", false},
		{"error_status_t", "error_status_t", "uint32", false},
		// Primitive pointers are flattened to their scalar representation.
		{"pointer_to_HRESULT", "PHRESULT", "int32", false},
		{"pointer_backed_HRESULT", "HRESULT", "*int32", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			hresultType := "typedef long HRESULT;"
			if tc.name == "pointer_backed_HRESULT" {
				// wtypesbase.idl's strict HRESULT is a pointer, not a status scalar.
				hresultType = "typedef struct _HRESULT_STRUCT { unsigned long Data1; } HRESULT_STRUCT, *PHRESULT_STRUCT; typedef PHRESULT_STRUCT HRESULT;"
			}
			f, err := midl.Parse(fmt.Sprintf(`
%s
typedef HRESULT RESULT_ALIAS;
typedef RESULT_ALIAS RESULT_CHAIN;
typedef HRESULT *PHRESULT;
typedef unsigned long DWORD;
typedef DWORD WIN32_ERROR;
typedef unsigned long RPC_STATUS;
[uuid(12345678-1234-1234-1234-123456789abc), version(1.0)]
interface Status { %s Probe(); }
`, hresultType, tc.idlType))
			if err != nil {
				t.Fatal(err)
			}
			p := &Generator{out: NewFileBuffer("status", "status")}
			p.GenClient(WithInterface(WithFile(context.Background(), f), f.Interfaces[0]), f.Interfaces[0])
			fs := token.NewFileSet()
			generated, err := parser.ParseFile(fs, "client.go", "package fixture\n"+p.out.Out.String(), 0)
			if err != nil {
				t.Fatal(err)
			}
			var method *ast.FuncDecl
			for _, decl := range generated.Decls {
				if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name.Name == "Probe" {
					method = fn
				}
			}
			if method == nil {
				t.Fatal("generated client has no Probe method")
			}
			// Only substitute the unused call-option type to keep this executable
			// fixture independent of the external RPC runtime.
			method.Type.Params.List[2].Type.(*ast.Ellipsis).Elt = ast.NewIdent("any")
			var body bytes.Buffer
			if err := format.Node(&body, fs, method); err != nil {
				t.Fatal(err)
			}
			dir := t.TempDir()
			failure := "2147500037"
			if tc.goType == "int32" {
				failure = "-2147467259"
			}
			values := fmt.Sprintf("{{0, false}, {1, %t}, {2147483647, %t}, {%s, true}}", !tc.hresult, !tc.hresult, failure)
			if tc.goType == "*int32" {
				values = "{{nil, false}, {new(int32), true}}"
			}
			files := map[string]string{
				"go.mod": "module fixture\n\ngo 1.25.0\n",
				"client_test.go": fmt.Sprintf(`package fixture
import ("context"; "errors"; "fmt"; "testing")
type status = %s
type operation struct { Return status; Errors []int32 }
func (*operation) OpName() string { return "Probe" }
type ProbeRequest struct{}
func (*ProbeRequest) xxx_ToOp(context.Context, *operation) *operation { return &operation{} }
type ProbeResponse struct { Return status; Errors []int32 }
func (r *ProbeResponse) xxx_FromOp(_ context.Context, op *operation) { r.Return, r.Errors = op.Return, op.Errors }
type connection struct { value status; err error }
func (c *connection) Invoke(_ context.Context, op *operation, _ ...any) error { op.Return, op.Errors = c.value, []int32{0, -1}; return c.err }
func (*connection) Error(context.Context, any) error { return errors.New("status failure") }
type xxx_DefaultStatusClient struct { cc *connection }
%s
func TestStatus(t *testing.T) {
 for _, tc := range []struct { value status; wantErr bool }%s {
  c := &xxx_DefaultStatusClient{cc: &connection{value: tc.value}}
  out, err := c.Probe(context.Background(), &ProbeRequest{})
  if (err != nil) != tc.wantErr { t.Errorf("status %%v: error=%%v, wantErr=%%v", tc.value, err, tc.wantErr) }
  if out == nil || out.Return != tc.value || len(out.Errors) != 2 || out.Errors[1] != -1 { t.Fatalf("lost response: %%+v", out) }
 }
 transportErr := errors.New("transport")
 c := &xxx_DefaultStatusClient{cc: &connection{err: transportErr}}
 if out, err := c.Probe(context.Background(), &ProbeRequest{}); out != nil || !errors.Is(err, transportErr) { t.Fatalf("transport: %%v, %%v", out, err) }
}
`, tc.goType, body.String(), values),
			}
			for name, content := range files {
				if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0600); err != nil {
					t.Fatal(err)
				}
			}
			cmd := exec.Command("go", "test", "-count=1", ".")
			cmd.Dir = dir
			cmd.Env = append(os.Environ(), "GOWORK=off")
			if output, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("generated client behavior: %v\n%s", err, output)
			}
		})
	}
}
