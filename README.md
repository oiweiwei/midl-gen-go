# midl-gen-go

MIDL parser and Go client stub generator for MS-RPC and DCOM interfaces.

Extracted from [go-msrpc](https://github.com/oiweiwei/go-msrpc). Generated stubs target the
`dcerpc` and `ndr` runtime packages from that module.

## Build

```sh
make build   # produces bin/midl-gen-go
```

The parser is generated from `midl/parse.y` via `goyacc`:

```sh
make gen     # regenerate midl/parse.go from parse.y
```

## Usage

```sh
midl-gen-go generate [flags] file.idl ...
midl-gen-go dump     [flags] file.idl
```

### `generate` flags

| Flag | Default | Description |
| ---- | ------- | ----------- |
| `-I`, `--include` | | IDL search path entry; repeatable. Format: `path` or `base=path` |
| `-o`, `--output` | `msrpc/` | Output directory root |
| `--pkg` | `github.com/oiweiwei/go-msrpc/msrpc` | Go import path base for generated packages |
| `--doc-cache` | `.cache/doc/` | Cache directory for MSDN documentation |
| `--verbose` | `false` | Enable trace output |
| `--no-format` | `false` | Skip `gofmt` on generated files |

### `dump` flags

| Flag | Description |
| ---- | ----------- |
| `-I`, `--include` | IDL search path entry (same format as `generate`) |

### IDL Search Path (`-I`)

Each `-I` value is a colon-separated list of directories (or a single entry) searched when
resolving `import` statements. An entry can carry an optional Go module base prefix:

```
-I github.com/oiweiwei/go-msrpc/msrpc=../go-msrpc/idl/
-I ./idl/
```

An entry without `=` is treated as a plain filesystem path with no Go base override.
`-I` values are prepended to the `MSIDLPATH` environment variable if it is already set.

### File arguments

IDL files can be passed as absolute paths or as paths relative to one of the `-I` directories.
`-I` is always required - the generator uses it both to resolve `import` statements and to
derive the Go import path for generated packages.

> **Note:** If your IDL uses `context_handle` or imports any MS-RPCE base types
> (`ms-dtyp.idl`, `ms-rpce.idl`, etc.), you must include the
> [go-msrpc](https://github.com/oiweiwei/go-msrpc) IDL directory in `-I`:
> ```
> -I github.com/oiweiwei/go-msrpc/msrpc=<path-to-go-msrpc>/idl/
> ```

## Demo

Generate all IDL files under `examples/demo/idl/`:

```sh
./bin/midl-gen-go generate \
    -I $(pwd)/../go-msrpc/idl \
    -I examples/demo/idl/ \
    -o examples/demo/ \
    --pkg github.com/oiweiwei/midl-gen-go/examples/demo \
    examples/demo/idl/*.idl
```

Or a single file by name relative to one of the `-I` directories:

```sh
./bin/midl-gen-go generate \
    -I $(pwd)/../go-msrpc/idl \
    -I examples/demo/idl/ \
    -o examples/demo/ \
    --pkg github.com/oiweiwei/midl-gen-go/examples/demo \
    myidl.idl
```

See [examples/demo/idl/myidl.idl](examples/demo/idl/myidl.idl) for the source IDL.

## Generated Output Structure

For an IDL file `myidl.idl` the generator creates:

```
<dir>/myidl/
    myidl.go              - shared types and struct definitions
    client/
        client.go         - DCOM client set (only for files with [object] interfaces)
    <interface_name>/
        v<major>/
            v<major>.go   - client interface + request/response types
            server.go     - server-side interface skeleton
```

- Plain RPC interfaces (no `[object]` attribute) produce only
  `<interface_name>/v<major>/v<major>.go` and `server.go`.
- DCOM interfaces (`[object]`) additionally produce `client/client.go` which aggregates
  all DCOM interfaces in the file into a single `Client`.

## Naming

See [gonames/README.md](gonames/README.md) for the full documentation on identifier naming
rules and `.midl/config.yaml` configuration.

## Docker

Pull the image:

```sh
docker pull ghcr.io/oiweiwei/midl-gen-go:latest
```

Run with local IDL and output directories mounted. Use `--user` so generated files are
owned by the current user, not root:

```sh
docker run --rm \
    --user "$(id -u):$(id -g)" \
    -v $(pwd):/work \
    -v $(pwd)/../go-msrpc/idl:/go-msrpc/idl:ro \
    ghcr.io/oiweiwei/midl-gen-go \
    generate \
    -I github.com/oiweiwei/go-msrpc/msrpc=/go-msrpc/idl/ \
    -I /work/examples/demo/idl/ \
    -o /work/examples/demo/ \
    --pkg github.com/oiweiwei/midl-gen-go/examples/demo \
    /work/examples/demo/idl/myidl.idl
```

All paths inside the container must be absolute. The working directory `/work` is
arbitrary - choose any mount point that is consistent across `-I`, `-o`, and file arguments.

## Runtime Dependency

Generated code imports `github.com/oiweiwei/go-msrpc` for the DCE/RPC transport and NDR
marshaling runtime. Add it to your module:

```sh
go get github.com/oiweiwei/go-msrpc
```
