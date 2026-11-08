# Rework options — 2026-11-08

Status: draft for adversarial review. No implementation selected.

## Requirements and assumptions

- Create, search, reveal/copy, edit, delete, change master password, manual lock, automatic lock, and encrypted backup/restore.
- One portable encrypted vault file; no account, network service, or mandatory Keychain dependency to decrypt it.
- Existing DB compatibility or a verified, non-destructive migration. No plaintext export on disk.
- Interpret “new and prep versions” to cover both previous macOS versions and prereleases. Provisional custom-app deployment target: macOS 13+, Intel x86_64 and Apple Silicon arm64. This is a target, not a tested support claim. Required older versions can change this choice.
- Release support policy: test newest stable macOS and two preceding major releases, with next beta/RC as a separate exploratory lane. Test both CPU architectures where Apple supports them. Build SDK versions and deployment targets are separate; dependency updates must not silently raise the latter. Future Apple releases cannot be guaranteed in advance.
- No cloud sync or browser extension in a custom-app v1. Local concurrent opens and externally replaced vaults must fail safely. Encryption cannot prevent malware reading an unlocked vault, offline password guessing, or replacement with a valid older vault.

## Comparison

| Option | Stack and storage | Legacy path | Best fit | Main cost/risk |
|---|---|---|---|---|
| A: native Pass | Swift 6 language/toolchain, SwiftUI with AppKit integration, direct SQLCipher C API; stable SQLCipher 4 file | Read legacy profiles; migrate to a new encrypted file | macOS is the product, bespoke UI matters | Own the full app and security lifecycle; Swift UI copies of secrets |
| B: portable Pass | Tauri 2, Rust vault service, React + TypeScript + Vite, bundled SQLCipher through a reviewed Rust binding | Same migration contract as A | Future Linux/Windows support and existing React skills | WebView/IPC attack surface plus Rust/C packaging |
| C: adopt KeePassXC | Maintained upstream C++/Qt desktop app; KDBX 4 + Argon2id; small migration utility in this repo | One-way SQLCipher-to-KDBX conversion | Lowest ongoing ownership and mature CRUD | New UX and format; importer still security-sensitive |

Provisional preference: **A for a bespoke macOS replacement; C if maintaining a bespoke manager is optional. B if cross-platform is a real requirement.** A and B share storage deliberately: choosing a UI should not force inventing a new cryptographic format. C provides a materially different storage and maintenance strategy.

## A — SwiftUI/AppKit + SQLCipher

Keep the relational data model. Replace Electron, Flow, Redux, Knex and the Node native module with a small Swift package for vault operations and a native app. Use an isolated vault actor to serialize access; C API calls use explicit error checks and bound SQL values. Pass key bytes via `sqlite3_key` rather than SQL interpolation. Link only the chosen SQLCipher SQLite implementation, and verify the packaged binary does not resolve storage calls to system SQLite.

List responses omit passwords. Reveal only on demand; copy via native clipboard code. An actor is concurrency isolation, not a security sandbox. A compromised app can access unlocked secrets. Do not promise complete Swift `String` erasure; minimize lifetimes/copies and clear editor state at lock. Close DB handles, cancel pending work and invalidate session tokens on idle, screen lock, sleep and manual lock. A lock event must stop late async responses from redisplaying secrets. Clipboard clearing must only clear content this app still owns; users can disable copying if Universal Clipboard propagation is unacceptable.

Use a pinned SQLCipher 4 stable build; the current upstream stable release observed is [4.20.0](https://github.com/sqlcipher/sqlcipher/releases/tag/v4.20.0). Confirm the Swift package/build recipe resolves that exact reviewed source and provider; do not accept a wrapper's ordinary SQLite fallback. On Apple, prefer the supported CommonCrypto build path. SQLCipher 5 is currently beta and changes the storage format; do not base v1 on it ([upstream announcement](https://www.zetetic.net/blog/)).

The new vault keeps the legacy columns and IDs, with a transactional schema version for future evolution. Legacy files open in a bounded import/read-only mode first; normal editing uses the verified new file. Direct legacy read/write mode is possible but not recommended because it perpetuates weaker KDF settings and increases the test matrix. Modernized files will not open in the old app.

Build with a current supported Xcode on macOS. Use SDK availability checks for APIs newer than the deployment target. Ship signed, hardened-runtime, notarized and stapled universal binaries (or separately verified architecture packages); verify every native dependency's architecture and minimum OS. Distribution signing proves publisher identity, not storage encryption. Disable network access in app design; defer updater code until its signing/update path is designed. Existing macOS CI images do not prove older OS behavior; use VMs or physical test machines for the matrix.

Effort: medium-to-high initial work, moderate continuing maintenance. First spike: packaged SQLCipher linkage, legacy fixtures, CRUD, crash recovery and lock lifecycle on both CPU architectures before investing in UI polish. Native app independently audited before real-vault adoption.

## B — Tauri 2 + Rust + TypeScript + SQLCipher

Reuse concepts from the React UI, not the insecure renderer/storage wiring. Put all keys, SQL, vault lifecycle and copying in Rust. Bundle a pinned SQLCipher build; inspect the Rust binding's actual SQLCipher source/provider version and avoid accidental dual linkage to SQLite. No generic frontend SQL, shell, filesystem or HTTP capability. Expose narrowly typed commands: unlock, list metadata, create, edit, delete, reveal, copy and lock. Restrict paths to a user-chosen vault handle; reject arbitrary path/SQL inputs. Validate commands in Rust even if frontend validation exists.

Frontend runs bundled local assets only, with a restrictive CSP, no remote navigation, no HTML rendering from vault values, and least-privilege Tauri capabilities. List/search never returns passwords. A native secure unlock dialog is preferred if feasible; otherwise password entry transiently exists in JS, which must be explicitly included in the threat model. Even with native unlock, a compromised authorized WebView could invoke permitted reveal/copy commands: capability rules alone cannot prevent this. Consider native user confirmation for sensitive export/reveal operations; keep v1 without bulk export APIs. Clear transient fields and reject late responses on lock.

Use the same SQLCipher file policy, migration, crash safety and evidence gates as A. Tauri's [documented default macOS floor](https://v2.tauri.app/distribute/macos-application-bundle/) is 10.13; that is not this app's support promise. Our proposed 13+ floor depends on tested Rust, SQLCipher/provider, plugins and WebKit behavior. Test on real target OS versions, not only a recent WebView or headless browser. Same signing/notarization and native-library checks as A. WebKit security fixes follow the OS update lifecycle.

Effort: high initial integration work and more dependencies than A, offset if multiple desktop platforms are needed. First spike: packaged SQLCipher compatibility, adversarial IPC/path validation, stale-response rejection and WebKit testing. Rust reduces some memory mistakes; it does not establish confidentiality against an authorized malicious frontend or unsafe C code.

## C — migrate to KeePassXC / KDBX 4

Use upstream KeePassXC rather than fork a password manager. This repo owns a small offline migration utility, tests, release evidence and instructions. Existing upstream GUI supplies edit/delete, generator and lock behavior; configure clipboard timeout, auto-lock, history and recycle-bin retention explicitly. If bespoke branding/UI is mandatory, this option does not meet that preference.

Current [official downloads](https://keepassxc.org/download/) advertise KeePassXC 2.8.0 for macOS 12+ with Intel and Apple Silicon packages. Verify the selected release, signature and actual target OS before rollout. Test latest/prior macOS and next beta as above; upstream support and Apple changes remain dependencies. Do not freeze an obsolete client to claim indefinite compatibility.

Convert in memory from read-only legacy SQLCipher to **KDBX 4 with AES-256 and Argon2id**, using an existing KDBX implementation or reviewed upstream serialization code, never a new crypto implementation. Candidate implementation/library and exact release must be selected and independently evaluated in the first spike; this is an unresolved gate, not a claim that any Rust KDBX crate is audited. Prefer a small isolated utility that can reuse upstream code, subject to license review. Reusing GPL code can require GPL distribution of the utility; do not silently add it under the repository's MIT license. No CSV/XML plaintext file workaround.

Map `name/login/password/url` to Title/UserName/Password/URL. Preserve integer ID, timestamps with millisecond precision, null-vs-empty information and any unmappable original representation in encrypted custom data; preserve native timestamps as closely as the destination supports. Test duplicate titles, arbitrary Unicode, long values, nulls and unusual timestamp values. Reopen with the actual KeePassXC release and compare all fields through an independent read path before declaring success. Migration is one-way; the original encrypted vault remains a rollback artifact.

The [KDBX format specification](https://keepass.info/help/kb/kdbx.html) documents authenticated headers and ciphertext blocks, key derivation and public metadata. KDF parameters and file structure remain visible; vault contents are encrypted. Calibrate Argon2id on the slowest supported machine (initial candidate: 64 MiB, 3 iterations, parallelism 1; measured target roughly 0.5–1 second, then adjust and record). A malicious file's KDF/resource parameters require bounds before allocation. No key file is required for the single-file portability goal.

KeePassXC has [version-scoped security reviews](https://keepassxc.org/audits/): the 2023 review covers 2.7.4 source and the ANSSI certification listed there covers 2.7.9 on Windows 10. These do not certify the current macOS build or our importer. Match report scope to the chosen release and inspect changes. C has the strongest available independent review trail of the three; importer and target build still need validation.

Effort: lowest ongoing application work; importer complexity is substantial enough to require its own spike and security review. First spike: select licensed KDBX writer, validate exact field round trips and cross-open with KeePassXC, inspect temporary files and atomic saves. If safe conversion requires plaintext disk staging, stop and choose a different route.

## Decision and delivery sequence

1. Owner selects direction and confirms the oldest required macOS release; no product implementation before that decision.
2. Build storage/migration spike with synthetic fixtures and reproducible evidence from the selected shipping dependency versions. Do not handle a real vault in CI.
3. Implement custom CRUD/lifecycle for A/B or conversion UX for C. Define delete as removal from the active vault, not assured erasure from backups/APFS snapshots/history.
4. Run the encryption, migration, crash and macOS matrix gates; commission an independent security review of the chosen implementation. Treat findings as blockers by severity.
5. Pilot with a local backup and independent comparison. Owner decides merge and release.

See [migration details](LEGACY-AND-MIGRATION.md), [evidence gates](ENCRYPTION-EVIDENCE.md), and the individual adversarial reviews in `reviews/`.
