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

Native CI is defined in [option-d.yml](../.github/workflows/option-d.yml). Final implementation commit **`a54a9852f32adeba027046bc7a6d9b6ecd57e4bd`** passed **all five jobs** in [run 42533514896](https://github.com/ssemakov/pass-rework/actions/runs/42533514896): the seven-scenario browser suite and the four native platforms below. Every native job passed production race tests, vet, exact CommonCrypto/cipher-setting checks, reserved raw-key regressions, explicit WebKit UI assertions, packaging, signature/linkage checks and the packaged-binary advisory scan.

| Runner / architecture | Observed OS | Xcode | Result |
|---|---|---|---|
| macos-15 / arm64 | 15.8.2 | 16.4 | Passed |
| macos-15-intel / x86_64 | 15.7.9 | 16.4 | Passed |
| macos-26 / arm64 | 26.7.2 | 26.6 | Passed |
| macos-26-intel / x86_64 | 26.6.1 | 26.6 | Passed |

The [permanent evidence manifest](evidence/2026-11-08.json) records source pins, effective cipher settings and each downloaded native binary's SHA-256, platform and artifact ID. All four ZIPs were downloaded and checked: inner app executable mode `100755`, hash matches the CI manifest, Go build metadata names the tested commit, explicit native UI completion is present and each binary advisory scan says “No vulnerabilities found.” CI artifacts include the app ZIP and full platform/test/linkage/signature/build evidence. GitHub artifact retention is finite; the source checks and commands remain reproducible.

A separate adversary agent independently reviewed the final implementation and checked native logs: **13 findings resolved, no remaining code blockers**. Its exact scope and independently reproduced tests are in [IMPLEMENTATION-REVIEW.md](IMPLEMENTATION-REVIEW.md). The earlier complete run at `96aca33` predates the final profile/rotation/raw-key corrections and is retained only as development history. Subsequent validation-record edits change documentation only; the implementation source remains the tested commit above.

Developer ID signing, Apple notarization/stapling and testing on an available future beta require the owner's release environment; no such results are claimed. CI produces usable architecture-specific ad-hoc-signed ZIPs and encryption/build evidence, without publishing a release. VoiceOver/manual usability and real-user migration pilot remain owner release activities; automated keyboard/focus tests do not establish full accessibility conformance.

## Subsequent legacy UI restoration

The later UI work restores the legacy screens, 904 × 534 outer window and Command-tilde lock shortcut. Final UI source `460666df2939a507a9e8591755f86e96dc6c8ff5` passed its local production build, three unit tests, Go vet and all 11 browser integration tests, including four source-derived visual comparisons with zero pixels above the stated tolerance. The preceding UI source `5a420d3` passed the full native/browser matrix in run 42559048614; all four app artifacts were downloaded and verified. Final-source CI attempts fail before runner assignment with zero steps, so those predecessor native results are not attributed to the final source. See [UI restoration results](UI-RESTORATION.md), the separate [UI review](UI-REVIEW.md) and [UI evidence manifest](evidence/ui-2026-11-09.json). The original implementation results above remain historical evidence for their named source.
