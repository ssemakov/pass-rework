# Validation record

All fixtures and passwords are synthetic/public. No real vault, user password or plaintext export was used. The implementation branch is `implementation/option-d`; `main` is owner-controlled.

## Repeatable checks

On macOS, first install the documented toolchain and generate the verified SQLCipher amalgamation:

```sh
./scripts/prepare-sqlcipher.sh
npm --prefix frontend ci --ignore-scripts --no-audit --no-fund
npm --prefix frontend run build
npm --prefix frontend test
go test -race -tags production ./...
go vet -tags production ./...
./scripts/native-smoke.sh
./scripts/package-macos.sh
./scripts/check-artifact.sh build/Pass.app/Contents/MacOS/Pass
go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 -mode=binary build/Pass.app/Contents/MacOS/Pass
```

Browser integration tests run the actual Go facade/SQLCipher backend with a constrained synthetic picker/clipboard adapter. They cover full create/edit/delete, unchanged-password editing, search/reveal/copy, backups, rotation/wrong passwords/reopen/restore, actual historical-driver migration, safe rendering, keyboard focus/Escape, responsive layout and delayed reveal/status/lock ordering:

```sh
cd frontend
npx playwright install chromium
npx vite build --mode e2e
npm run e2e
```

The `e2e` Go build tag and Vite mode expose synthetic test controls only in that test process. Release builds embed production assets and have no browser-test server/control route. The native smoke binary uses a separate `native_smoke` tag and explicit completion assertions; a clean exit by itself never counts as a pass.

## Coverage

| Gate | Evidence |
|---|---|
| Native cipher configuration | Exact version/provider/settings/temp-store readback |
| Encryption failure behavior | Wrong password, plaintext input, plain SQLite reader, tampered page, reserved raw-key password rejection/legacy conversion |
| No plaintext persistence | Vault header/sentinel and live encrypted journal inspection |
| Legacy compatibility | Actual historical driver fixture plus genuine profiles 3/4; source hash unchanged |
| Exact transfer | Every field/entry, NULL/empty/NUL/Unicode, signed IDs, large timestamps, revisions, sequence |
| Restart/transaction recovery | SIGKILL import discard/restart; SIGKILL uncommitted edit restored by SQLite |
| Ownership/format | No-overwrite publication, interrupted staging setup, malicious schema/aliases, concurrent SQLite session exclusion |
| Lifecycle | Immediate lock/idle invalidation, queued stale unlock, terminal Close, safe shutdown ordering |
| Bridge | Unauthorized methods/callers/headers/envelopes denied; Lock capacity priority |
| Frontend | Seven end-to-end scenarios plus unit fence/readiness tests |
| Packaged app | Native WebKit assertions, CommonCrypto build, signature, native symbol linkage and binary vulnerability scan |

## Results

Linux development verification on 2026-11-08: Go race tests pass for vault/service packages; frontend production build and unit tests pass; all seven Chromium integration tests pass. npm dependency audit reports zero known vulnerabilities after upgrading Vite/Vitest; Go core-package vulnerability scan reports none. These advisory scans are time-dependent and do not certify dependencies or cover C code.

Native CI is defined in [option-d.yml](../.github/workflows/option-d.yml), using native GitHub runners for macOS 15 arm64/x86_64 and macOS 26 arm64/x86_64. Run [42531608409](https://github.com/ssemakov/pass-rework/actions/runs/42531608409) at commit `96aca33f67f67cbab1f35ed085553a90b7afb819` passed all five jobs: browser plus macOS 15/26 on arm64/x86_64. Each native log contains the explicit WebKit UI-completion marker in addition to successful race tests, packaging, signature/linkage checks and binary advisory scan. A final run will additionally validate the stricter per-handle cipher-profile readback, asserted provider settings and ZIP packaging.

Developer ID signing, Apple notarization/stapling and testing on an available future beta require the owner's release environment; no such results are claimed. CI produces usable architecture-specific ad-hoc-signed ZIPs and encryption/build evidence, without publishing a release. VoiceOver/manual usability and real-user migration pilot remain owner release activities; automated keyboard/focus tests do not establish full accessibility conformance.
