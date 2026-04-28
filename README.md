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
midl-gen-go msdn     [flags] index-name [object-name...]
```

### `generate` flags

| Flag | Default | Description |
| ---- | ------- | ----------- |
| `-I`, `--include` | | IDL search path entry; repeatable. Format: `path` or `base=path` |
| `-o`, `--output` | `msrpc/` | Output directory root |
| `--pkg` | `github.com/oiweiwei/go-msrpc/msrpc` | Go import path base for generated packages |
| `--msdn-openspecs-cache-dir` | `.cache/doc/` | Cache directory for MSDN documentation |
| `--msdn-openspecs-indexer-file` | `/msdn/index.yaml` | Protocol indexer file for MSDN documentation (empty = skip MSDN sync) |
| `--msdn-openspecs-indexer-extra-file` | `/msdn/extra.yaml` | Extra indexer file for MSDN documentation |
| `--verbose` | `false` | Enable trace output |
| `--no-format` | `false` | Skip `gofmt` on generated files |

Setting `--msdn-openspecs-indexer-file` to an empty string disables MSDN documentation
fetching entirely. Generated files will have no doc-comments sourced from MS Open
Specifications pages.

### `dump` flags

| Flag | Description |
| ---- | ----------- |
| `-I`, `--include` | IDL search path entry (same format as `generate`) |

### `msdn` flags

| Flag | Default | Description |
| ---- | ------- | ----------- |
| `--msdn-openspecs-cache-dir` | `msdn/.cache/` | Cache directory for fetched MSDN pages |
| `--msdn-openspecs-indexer-file` | `msdn/index.yaml` | Protocol indexer file mapping names to URLs |
| `--msdn-openspecs-indexer-extra-file` | `msdn/extra.yaml` | Extra indexer file for additional entries |
| `--list` | `false` | List all available object names in the protocol index |
| `-o`, `--output` | | Output format: `json` (default: plain-text render) |
| `--verbose` | `false` | Enable trace output |

The `msdn` command fetches and renders MS Open Specifications documentation pages.
`index-name` is a protocol short name (e.g. `ms-rpce`) looked up in the indexer files.
Optional `object-name` arguments narrow the result to a specific struct, enum, or method.

```sh
# list all documented objects for a protocol
midl-gen-go msdn ms-dtyp --list

# render a single object as text
midl-gen-go msdn ms-dtyp FILETIME

# render as JSON
midl-gen-go msdn -o json ms-dtyp FILETIME
```

### Indexer file format (`index.yaml`)

The indexer file is a YAML list of protocol entries. Each entry maps a short name to its
MS Open Specifications family and UUID. The generator uses this to locate the online
documentation page for a given IDL file.

```yaml
- family: windows_protocols   # "windows_protocols" or "exchange_server_protocols"
  name: dtyp                  # short name; "ms-" prefix is added automatically
  uuid: cd85413a-cb32-43f1-ac50-d5267cd9542a

# A protocol that lives inside another protocol's document uses "ref":
- family: windows_protocols
  name: claims
  uuid: 9b7ab76e-076a-4f91-9d9d-186fd211fd66
  ref: ms-adts/               # redirect: look up pages under ms-adts instead
```

Field reference:

| Field | Required | Description |
| ----- | -------- | ----------- |
| `family` | no | Protocol family. Defaults to `windows_protocols`. |
| `name` | yes | Short protocol name (e.g. `dtyp`, `ms-dtyp`, `mc-ccfg`). The `ms-` prefix is added automatically if absent. |
| `uuid` | no | Interface UUID used to locate the documentation page. |
| `ref` | no | Redirect to another protocol's document subtree (e.g. `ms-adts/`). |

### Extra indexer file format (`extra.yaml`)

The extra file supplies per-protocol overrides for individual type or method entries that
are not discoverable from the main page index (e.g. union arms that share a page with a
parent type).

```yaml
- dnsp:                              # protocol short name (without "ms-" prefix)
  - name: DNSSRV_RPC_UNION          # type or method name as it appears in the IDL
    uuid: b61a8727-46b1-4981-a6b3-a1d4b92b67c4  # UUID of the specific section

- rprn:
  - name: RPC_V2_UREPLY_PRINTER
    uuid: 1ccdac5b-0b2a-4bd3-8afd-d2c2589130fc

  - name: DRIVER_INFO_1
    uuid: ms-rprn/4464eaf0-f34f-40d5-b970-736437a21913  # cross-protocol ref: proto/uuid
    aliases:                         # additional IDL names that map to the same page
      - DRIVER_INFO_2
      - RPC_DRIVER_INFO_3
```

Field reference:

| Field | Required | Description |
| ----- | -------- | ----------- |
| `name` | yes | Type or method name as it appears in the IDL. |
| `uuid` | yes | UUID of the documentation section. Use `protocol/uuid` to reference a page from another protocol. |
| `aliases` | no | Additional IDL names that resolve to the same documentation page. |

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
