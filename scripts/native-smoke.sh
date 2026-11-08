#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
mkdir -p build
CGO_ENABLED=1 MACOSX_DEPLOYMENT_TARGET=15.0 go build -trimpath -tags production,native_smoke -o build/pass-native-smoke .
python3 - <<'PY'
import subprocess
subprocess.run(['build/pass-native-smoke'],check=True,timeout=80)
PY
