#!/bin/bash
# Golden regression check for the generator.
#
#   scripts/golden.sh record <idl-root> <golden-dir>   generate with the current build into <golden-dir>
#   scripts/golden.sh check  <idl-root> <golden-dir>   generate into a temp dir and diff against <golden-dir>
#
# <idl-root> must contain *.idl at its top level and an h/ include dir, plus msdn/ (index.yaml, extra.yaml,
# .cache/) next to it, e.g. a copy of the go-msrpc checkout's idl/ and msdn/. Each IDL is generated in its own
# process, because a single failing IDL (e.g. an MSDN page that is not cached) aborts a batch run. IDLs that fail
# are listed on stderr; the set of failures is part of the comparison.
set -u
mode=${1:?record|check}; root=$(cd "${2:?idl-root}" && pwd); golden=${3:?golden-dir}
here=$(cd "$(dirname "$0")/.." && pwd)
tmp=$(mktemp -d); trap 'rm -rf "$tmp"' EXIT
(cd "$here" && go build -o "$tmp/midl-gen-go" .) || exit 1

out=$tmp/out; mkdir -p "$out"; : > "$tmp/fail"
cd "$root/.." || exit 1
base=$(basename "$root")
for f in "$base"/*.idl; do
  "$tmp/midl-gen-go" generate --pkg github.com/oiweiwei/go-msrpc/msrpc/ -I "./$base" -I "./$base/h" \
    --output "$out/" --msdn-openspecs-indexer-file ./msdn/index.yaml \
    --msdn-openspecs-indexer-extra-file ./msdn/extra.yaml --msdn-openspecs-cache-dir ./msdn/.cache/ \
    "$f" >"$tmp/log" 2>&1 || echo "$f" >> "$tmp/fail"
done
sort -o "$tmp/fail" "$tmp/fail"; cp "$tmp/fail" "$out/.failures"

case $mode in
record) rm -rf "$golden"; mkdir -p "$golden"; cp -r "$out/." "$golden/"; echo "recorded $(find "$golden" -name '*.go' | wc -l) files" ;;
check)  if diff -r "$golden" "$out"; then echo "golden: identical ($(find "$out" -name '*.go' | wc -l) files)"; else echo "golden: DIFFERENT" >&2; exit 1; fi ;;
*) echo "unknown mode $mode" >&2; exit 2 ;;
esac
cat "$tmp/fail" | sed 's/^/failed: /' >&2
