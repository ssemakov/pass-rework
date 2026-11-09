#!/usr/bin/env bash
# Local validation uses synthetic data only and never depends on GitHub Actions.
set -euo pipefail
cd "$(dirname "$0")/.."
mkdir -p artifacts
# A failed rerun must never leave a previous successful manifest as its result.
rm -f artifacts/local-validation.json
verification_started_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
verification_e2e_assets=0
verification_cleanup() {
  verification_result=$?
  trap - EXIT
  if [ "$verification_e2e_assets" = 1 ]; then
    # Never leave test-server bindings in assets used for a later release build.
    if ! npm --prefix frontend run build > artifacts/local-asset-restore.txt 2>&1; then
      cat artifacts/local-asset-restore.txt
      exit 1
    fi
  fi
  exit "$verification_result"
}
trap verification_cleanup EXIT

./scripts/prepare-sqlcipher.sh
go test -count=1 -race -tags production ./... 2>&1 | tee artifacts/local-go-tests.txt
go vet -tags production ./... 2>&1 | tee artifacts/local-go-vet.txt
npm --prefix frontend ci --ignore-scripts --no-audit --no-fund
npm --prefix frontend run build 2>&1 | tee artifacts/local-frontend-build.txt
npm --prefix frontend test 2>&1 | tee artifacts/local-frontend-unit.txt
verification_e2e_assets=1
(
  cd frontend
  if [ -z "${PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH:-}" ]; then
    npx playwright install chromium --only-shell
  fi
  npx vite build --mode e2e
  npm run e2e
) 2>&1 | tee artifacts/local-browser-tests.txt
npm --prefix frontend run build 2>&1 | tee artifacts/local-production-assets.txt
verification_e2e_assets=0

verification_native=false
if [ "$(uname -s)" = Darwin ]; then
  ./scripts/native-smoke.sh
  ./scripts/package-macos.sh
  ./scripts/check-artifact.sh build/Pass.app/Contents/MacOS/Pass
  go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 -mode=binary build/Pass.app/Contents/MacOS/Pass > artifacts/go-vulnerability-scan.txt
  verification_native=true
fi
python3 - "$verification_native" "$verification_started_at" <<'PY'
import datetime, json, pathlib, platform, subprocess, sys
native = sys.argv[1] == 'true'
def git(*args):
    return subprocess.check_output(['git', *args], text=True).strip()
record = {'status':'passed', 'startedAtUTC':sys.argv[2],
          'completedAtUTC':datetime.datetime.now(datetime.timezone.utc).isoformat(),
          'sourceCommit':git('rev-parse','HEAD'),
          'workingTreeDirty':bool(git('status','--porcelain')),
          'platform':platform.platform(), 'architecture':platform.machine(),
          'raceTests':'passed', 'vet':'passed', 'productionBuild':'passed',
          'unitTests':'passed', 'browserTests':'passed',
          'nativeMacOSChecksExecuted':native, 'usesGitHubActions':False}
pathlib.Path('artifacts/local-validation.json').write_text(json.dumps(record,indent=2)+'\n')
print('Local checks passed.' if native else 'Local checks passed; native macOS checks were not run on this platform.')
print('Evidence: artifacts/local-validation.json and local-*.txt')
PY
