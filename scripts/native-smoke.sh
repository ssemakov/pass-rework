#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
mkdir -p build
CGO_ENABLED=1 MACOSX_DEPLOYMENT_TARGET=15.0 go build -trimpath -tags production,native_smoke -o build/pass-native-smoke .
python3 - <<'PY'
import subprocess, tempfile, os, json
from pathlib import Path
Path('artifacts').mkdir(exist_ok=True)
logs=[]
with tempfile.TemporaryDirectory(prefix='pass-native-restart-') as root:
    for phase in ['first','restart']:
        env=dict(os.environ,PASS_NATIVE_SMOKE_ROOT=root,PASS_NATIVE_SMOKE_PHASE=phase)
        result=subprocess.run(['build/pass-native-smoke'],env=env,capture_output=True,text=True,timeout=80)
        logs.append(f'Native process phase: {phase}\n'+result.stdout+result.stderr)
        Path('artifacts/native-smoke.txt').write_text(''.join(logs))
        print(logs[-1],end='')
        assert result.returncode==0, f'native {phase} smoke returned failure'
        assert 'Native window frame verified: 904x534' in result.stdout, 'legacy window frame not verified'
        assert 'Native WebKit smoke passed: explicit UI assertions complete' in result.stdout, 'native smoke did not report completion'
        saved=json.loads((Path(root)/'settings.json').read_text())
        assert saved['default_storage_path']==str(Path(root)/'imported.pass'), 'imported storage path was not persisted'
        assert set(saved)=={'default_storage_path','idle_minutes'}, 'unexpected settings data'
        if phase=='restart':
            assert 'Native remembered storage restart verified' in result.stdout, 'native restart was not verified'
        else:
            assert 'Native import count and same-screen result verified; source unchanged' in result.stdout, 'native import count/source was not verified'

PY
