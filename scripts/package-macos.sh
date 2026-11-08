#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
[ "$(uname -s)" = Darwin ] || { echo 'macOS build host required'; exit 1; }
./scripts/prepare-sqlcipher.sh
npm --prefix frontend run build
mkdir -p build/Pass.app/Contents/MacOS build/Pass.app/Contents/Resources
export MACOSX_DEPLOYMENT_TARGET=15.0
CGO_ENABLED=1 go build -trimpath -tags production -ldflags='-w' -o build/Pass.app/Contents/MacOS/Pass .
cat > build/Pass.app/Contents/Info.plist <<'PLIST'
<?xml version="1.0" encoding="UTF-8"?><!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd"><plist version="1.0"><dict>
<key>CFBundleExecutable</key><string>Pass</string><key>CFBundleIdentifier</key><string>io.ssemakov.pass-rework</string><key>CFBundleName</key><string>Pass</string><key>CFBundlePackageType</key><string>APPL</string><key>CFBundleShortVersionString</key><string>1.0.0</string><key>CFBundleVersion</key><string>1</string><key>LSMinimumSystemVersion</key><string>15.0</string><key>NSHighResolutionCapable</key><true/><key>NSRequiresAquaSystemAppearance</key><false/></dict></plist>
PLIST
# Local/CI builds are ad-hoc signed. A supplied identity enables proper signing.
# Notarization is a separate release step requiring the owner's Apple credentials.
codesign --force --deep --sign "${PASS_SIGN_IDENTITY:--}" --options runtime --timestamp=none build/Pass.app
codesign --verify --strict --deep build/Pass.app
