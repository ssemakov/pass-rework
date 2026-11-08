#!/usr/bin/env bash
set -euo pipefail
binary=${1:?binary required}
mkdir -p artifacts
go version -m "$binary" > artifacts/go-build-info.txt
shasum -a 256 "$binary" > artifacts/binary-sha256.txt
if grep -E 'mcp|devtools|purego|debug|server|native_smoke|e2e' artifacts/go-build-info.txt; then echo 'Forbidden release build tags'; exit 1; fi
if [ "$(uname -s)" = Darwin ]; then
 otool -L "$binary" >artifacts/linkage.txt
 # WebKit/AppKit may themselves depend on system SQLite; this executable's
 # storage code must have no undefined sqlite3 symbols that bind to it.
 nm -u "$binary" >artifacts/undefined-symbols.txt
 if grep -E '_sqlite3_' artifacts/undefined-symbols.txt; then echo 'External SQLite storage symbols'; exit 1; fi
 nm "$binary" >artifacts/symbols.txt
 grep -q '_sqlite3_key' artifacts/symbols.txt || { echo 'SQLCipher key API missing'; exit 1; }
 codesign --verify --strict --deep "$(dirname "$(dirname "$(dirname "$binary")")")"
fi
