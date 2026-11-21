# Features

The Go generator always generates the **core**: the interface identifiers, the constants
and types of the IDL, and the request and response of every operation with their NDR
marshalling. Everything else is a **feature** that can be switched on and off:

```
midl-gen-go features                              # list them
midl-gen-go generate --features all,-server ...   # everything but the server
midl-gen-go generate --features client ...        # the client only
midl-gen-go generate --features none ...          # the core only
```

| feature | generates |
|---|---|
| `client` | the client interface and its implementation; for DCOM the client set of an IDL file |
| `server` | `server.go`: the server interface, the operation dispatcher, the unimplemented server |

The core never depends on a feature. A feature may require others (`Requires`), which is
checked when the generator is created.

## Adding a feature

1. **Pick the phases** (`feature.go`). A feature emits code for an interface at one or
   more points, in the order the features are registered:

   | phase | where |
   |---|---|
   | `BeforeTypes` | after the interface identifiers, before the interface's constants and types |
   | `AfterTypes` | after the types, before the operations (not reached for an interface without a UUID) |
   | `InFile` | in a file of its own, after the operations; implement `FileNamer` to name it |

   A feature that also needs something once per IDL file implements `FileFeature` (that is
   how the client emits the DCOM client set).

2. **Write it in its own file** (`myfeature.go`), as a type implementing `Feature`:

   ```go
   type metricsFeature struct{}

   func (metricsFeature) Name() string       { return "metrics" }
   func (metricsFeature) Doc() string        { return "a call counter per operation" }
   func (metricsFeature) Requires() []string { return []string{"client"} }
   func (metricsFeature) Phases() []Phase    { return []Phase{AfterTypes} }

   func (metricsFeature) Interface(ctx *emit.Env, p *Generator, iff *midl.Interface, _ Phase) {
       w := p.W()
       w.Blank()
       w.Func(goprint.Func{Name: "Count" + emit.GoName(ctx, iff.Name)}, func() { ... })
   }
   ```

   * Write with `p.W()` (the typed `goprint.Writer`) for new code, not `p.P(...)`.
   * Name things with `emit.GoName`, `emit.GoMemberName`, `p.OpName`, `p.MethodName`: never build
     Go names by hand, so that the gonames rules and overrides apply (`midl-gen-go symbols`
     shows the result).
   * Refer to other packages by their qualifier (`ndr.`, `dcerpc.`) or `p.AddImport(...)`;
     the import is written only if the code uses it.

3. **Register it** in `init()` in `feature.go`: `RegisterFeature(metricsFeature{})`. The order of
   registration is the order of emission within a phase.

4. **Test it** (see `feature_test.go`): generate `testdata/override/ov.idl` with the feature
   selected and not, and check that the code is there and is not. Then:
   * `make golden`: the output with the default selection must be byte-identical unless the
     new feature is meant to change it (then re-record the golden output and say so in the commit).
   * compile the result: generate a few real IDLs with each selection and build them against
     go-msrpc (`go build -overlay`), so a feature that the core depends on, or the other way
     round, shows up.

5. **Document it**: the `Doc` string is what `midl-gen-go features` prints; add a row to the table above.

## Layers

```
golang        this package: files, interfaces, features, symbols
  golang/ndr    NDR marshalling of types and operations
    golang/emit   printing, names, imports, output files
```

A layer uses only the ones below it. A feature belongs to the top layer and may call any
emitter method; code that every feature needs belongs lower down, in `ndr` or `emit`.
