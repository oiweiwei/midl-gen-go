# gonames

The `gonames` package converts IDL identifiers to idiomatic Go names.

## Configuration

Configuration lives in a `.midl/` directory next to the IDL files. The main file is
`.midl/config.yaml`:

```yaml
go:
  pkg: github.com/example/mypackage   # Go import path base for generated packages

gonames:
  # Path to a directory of gonames YAML files (loaded and merged in glob order).
  path: ./gonames
```

Alternatively, gonames rules can be written inline in `config.yaml` using `gonames.global`
(applied to all IDL files) and `gonames.idl` (per-file overrides):

```yaml
gonames:
  global:
    trim:
      prefixAll:
        - MY_
  idl:
    - myidl:
        rename:
          MY_STRUCT: [My, Struct]
```

## Naming rules

The pipeline for a single identifier is:

1. **`rename`** exact match - if the full identifier is found, return that token list immediately, skip all other steps.
2. **`trim.prefixAll`** - strip matching prefixes from the identifier string (applied before lexing).
3. **`trim.suffix`** - strip matching suffixes from the identifier string (applied before lexing).
4. **Lex** - split into tokens on `_` and CamelCase boundaries, applying `split` and `lookBehind` during lexing.
5. For each token: apply **`rotate`** (first token only), **`trim.prefix`** (first token only), **`trim.word`**, **`abbr`**, title-case fallback.

### `rename` - replace the entire identifier

Full identifier is matched as-is; value is the output token list.

```yaml
rename:
  hRpc: [Handle]               # hRpc -> Handle
  para: [Parameters]           # para -> Parameters
  SizeOfStruct: [Structure, Size]  # SizeOfStruct -> StructureSize
```

### `trim.prefixAll` - strip a prefix from the identifier string before lexing

```yaml
trim:
  prefixAll:
    - FSRM_     # FSRM_FolderUsage -> FolderUsage
    - MY_       # MY_STRUCT -> STRUCT (then lexed and named by other rules)
```

### `trim.suffix` - strip a suffix from the identifier string before lexing

```yaml
trim:
  suffix:
    - W         # CreateFileW -> CreateFile
    - Ex        # OpenEx -> Open
```

### `split` - force a compound token to be split into multiple tokens during lexing

Used when the lexer cannot split a run-together lowercase word on its own.

```yaml
split:
  addentry: [add, entry]           # addentry -> [add, entry] -> AddEntry
  autoconfigure: [auto, configure] # autoconfigure -> [auto, configure] -> AutoConfigure
  authip: [auth, ip]               # authip -> [auth, ip] -> AuthIP (ip expanded by abbr)
```

### `lookBehind` - context-sensitive token replacement during lexing

The **outer key is the previous token**; the inner map is `current-token: replacement-list`.
When `prev` followed by `cur` is seen, both are replaced by the replacement list.

```yaml
lookBehind:
  h:
    context: [context]   # H then Context -> Context  (drops the H prefix)
    rpc: [h]             # H then Rpc -> H            (keeps H, drops Rpc)
  end:
    point: [endpoint]    # End then Point -> single token "endpoint" -> Endpoint
    points: [endpoints]
  d:
    word: [dword]        # D then Word -> "dword" -> DWORD via abbr
```

An empty replacement list drops both tokens.

### `rotate` - move the first token to the end of the result

Applied only to the **first** token. The token is removed from the front and appended
after all other tokens are assembled (with special handling when the last token is `Size`
or `Length`).

```yaml
rotate:
  ob: Offset    # ObSize -> Size then append "Offset" -> SizeOffset
```

### `trim.prefix` - drop the first token if it matches

```yaml
trim:
  prefix:
    - I         # IFoo -> Foo  (COM-style interface prefix)
    - Rpc       # RpcBinding -> Binding
    - dw        # dwSize -> Size
```

### `trim.word` - drop a token at any position if it matches

```yaml
trim:
  word:
    - Ptr
    - p
```

### `abbr` - expand or canonicalize a lowercased token

The key is lowercase; the value is the exact string used in the Go name.

```yaml
abbr:
  acl: ACL        # token "acl" -> "ACL"
  buf: Buffer     # token "buf" -> "Buffer"
  api: API
  attr: Attribute
```
