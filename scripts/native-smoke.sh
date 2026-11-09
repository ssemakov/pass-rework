#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
mkdir -p build
CGO_ENABLED=1 MACOSX_DEPLOYMENT_TARGET=15.0 go build -trimpath -tags production,native_smoke -o build/pass-native-smoke .
python3 - <<'PY'
import subprocess
from pathlib import Path
result=subprocess.run(['build/pass-native-smoke'],capture_output=True,text=True,timeout=80)
print(result.stdout,end='');print(result.stderr,end='')
Path('artifacts').mkdir(exist_ok=True)
Path('artifacts/native-smoke.txt').write_text(result.stdout+result.stderr)
assert result.returncode==0, 'native smoke returned failure'
assert 'Native window frame verified: 904x534' in result.stdout, 'legacy window frame not verified'
assert 'Native WebKit smoke passed: explicit UI assertions complete' in result.stdout, 'native smoke did not report completion'

PY
