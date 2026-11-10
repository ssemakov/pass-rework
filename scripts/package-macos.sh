#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
[ "$(uname -s)" = Darwin ] || { echo 'macOS build host required'; exit 1; }
./scripts/prepare-sqlcipher.sh
npm --prefix frontend run build
mkdir -p build/Pass.app/Contents/MacOS build/Pass.app/Contents/Resources
export MACOSX_DEPLOYMENT_TARGET=15.0
CGO_ENABLED=1 go build -trimpath -tags production -ldflags='-w' -o build/Pass.app/Contents/MacOS/Pass .
python3 scripts/write-plist.py build/Pass.app/Contents/Info.plist
# Local/CI builds are ad-hoc signed. Developer ID releases use a secure timestamp.
timestamp=--timestamp
if [ "${PASS_SIGN_IDENTITY:--}" = - ]; then timestamp=--timestamp=none; fi
codesign --force --deep --sign "${PASS_SIGN_IDENTITY:--}" --options runtime "$timestamp" build/Pass.app
codesign --verify --strict --deep build/Pass.app
# ZIP retains executable permissions and bundle metadata through CI downloads.
ditto -c -k --sequesterRsrc --keepParent build/Pass.app "build/Pass-macos-$(uname -m).zip"
