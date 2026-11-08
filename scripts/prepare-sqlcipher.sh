#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
revision=355d4db92d3404e71faa8088d071a71d767807f4
sha=fc5394c916beeabb1f43e2ac97d0827f36e8917f6675a00cca54537c32cb36a2
mkdir -p .deps
archive=.deps/sqlcipher.tar.gz
if [ ! -f "$archive" ]; then
  curl --fail --location --proto '=https' --tlsv1.2 "https://codeload.github.com/sqlcipher/sqlcipher/tar.gz/$revision" -o "$archive"
fi
python3 - "$archive" "$sha" <<'PY'
import hashlib,sys
assert hashlib.sha256(open(sys.argv[1],'rb').read()).hexdigest()==sys.argv[2], 'SQLCipher source hash mismatch'
PY
if [ ! -f internal/cipher/sqlite3.c ]; then
  tar -xzf "$archive" -C .deps
  ( cd ".deps/sqlcipher-$revision"; ./configure --disable-tcl --disable-readline --with-tempstore=always >configure.log; make sqlite3.c >generate.log )
  cp ".deps/sqlcipher-$revision/sqlite3.c" ".deps/sqlcipher-$revision/sqlite3.h" internal/cipher/
  cp ".deps/sqlcipher-$revision/LICENSE.md" internal/cipher/SQLCIPHER-LICENSE.md
fi
