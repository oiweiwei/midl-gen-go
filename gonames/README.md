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
  ob: Offset    # obSize -> Size then append "Offset" -> SizeOffset
```

The key is compared to the first lexem as it is written, so `ob` matches the
Hungarian-notation `obSize` but not `ObSize`.

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
    - ptr
    - p
```

The words are compared in lower case, so write them in lower case.

### `acronyms` - words that are written in a fixed spelling

A list of spellings. A word that equals one ignoring case becomes that spelling:

```yaml
acronyms:
  - ACL        # token "acl"  -> "ACL"
  - DWORD
  - IKEv1      # token "ikev1" -> "IKEv1"
```

It is the short form of an `abbr` entry whose value is its own key in another case
(`acl: ACL`); `abbr` entries take precedence. Use `abbr` when the word is replaced by a
different word (`buf: Buffer`).

Lexems are never joined automatically. Joining `I` and `Pv6` into `IPv6` would also change
what the other rules see (a trimmed prefix `p` in `pIn` would be swallowed into the
acronym `PIN`) and the words of the snake case name (`nt_owf_password` would become
`ntowf_password`, a different JSON name). Use `lookBehind` for a join you want.

### `abbr` - expand or canonicalize a lowercased token

The key is lowercase; the value is the exact string used in the Go name.

```yaml
abbr:
  acl: ACL        # token "acl" -> "ACL"
  buf: Buffer     # token "buf" -> "Buffer"
  api: API
  attr: Attribute
```

## Overriding one name

The rules above are global. To rename one field of one struct (or one parameter of one
operation, or one element of one enum) without touching anything else, use `override`:

```yaml
override:
  SAMPR_USER_ALL_INFORMATION:     # the owner: the typedef name of the struct, union or enum
    NtOwfPassword: [Nt, Hash]     #   member (IDL name): the words of the Go name -> NtHash
  rpc__mgmt_inq_stats:            # an operation; its members are its parameters
    count: [Max, Count]
```

* The **owner** is the name the type is declared with (the typedef name, or the tag if there
  is none), or the operation name. The **member** is the IDL name of the field, union arm,
  parameter or enumeration element.
* The value is the list of words, as for `rename`; it wins over every other rule and applies to
  that member only.
* It changes the Go identifier everywhere the member is used (the declaration, the marshalling
  code, the `size_is` expressions of other members). It does not change the `json` tag or the
  `idl:"name:..."` tag: those follow the IDL name, so the wire formats stay as they were.
* Per-IDL overrides are written under `gonames.idl` in `config.yaml`, as for the other rules.

## Seeing how a name is derived

```
midl-gen-go symbols -I idl idl/epm.idl --kind field --match ept_entry
midl-gen-go symbols -I idl idl/samr.idl --match 'NtOwfPassword$' --explain
```

lists every name of the file with the Go name the generator gives it, the words, and the rules
that fired (`override`, `rename`, `trim.prefixAll`, `trim.suffix`, `lookBehind`, `split`,
`rotate`, `trim.prefix`, `trim.word`, `abbr`, `acronym`, `reserved`, ...). The OWNER column is the
key to use in `override`. `--explain` shows every step; `--json` gives the same for scripts.
