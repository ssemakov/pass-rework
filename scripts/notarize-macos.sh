#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
: "${PASS_NOTARY_PROFILE:?Set the existing notarytool keychain profile name}"
app=build/Pass.app
archive="build/Pass-macos-$(uname -m).zip"
# This step is for an already Developer ID signed build. Never ship ad-hoc CI
# binaries as a notarized release. Credentials remain in the owner's keychain.
codesign --verify --strict --deep "$app"
codesign -dv "$app" 2>&1 | grep 'Authority=Developer ID Application:' >/dev/null
xcrun notarytool submit "$archive" --keychain-profile "$PASS_NOTARY_PROFILE" --wait
xcrun stapler staple "$app"
xcrun stapler validate "$app"
spctl --assess --type execute --verbose "$app"
ditto -c -k --sequesterRsrc --keepParent "$app" "$archive"
shasum -a 256 "$archive"
