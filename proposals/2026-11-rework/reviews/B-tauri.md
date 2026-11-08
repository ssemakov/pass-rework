# Adversarial review B — Tauri / Rust / SQLCipher

Reviewer: separate adversary agent `/root/adversary_tauri`, 2026-11-08.

Scope: Option B and the shared migration/evidence documents as first drafted, published at [`b7fede59b8acbc3d07bc59e53deacc210739c800`](https://github.com/ssemakov/pass-rework/commit/b7fede59b8acbc3d07bc59e53deacc210739c800). This is a design review, not an implementation audit. No app, encryption, migration or macOS runtime tests were performed. Primary Tauri documentation was checked to challenge the proposed privilege boundary.

## Verdict

**Conditionally viable; no basis yet to approve real-vault use.** SQLCipher compatibility is a plausible route and is independent of the WebView UI. B can be justified by a concrete cross-platform need. For a macOS-only manager it adds a security-sensitive serialization and command surface that A does not need. Native key ownership reduces exposure but does not make an authorized compromised frontend safe.

The proposal correctly labels evidence as outstanding, rejects plaintext fallback, preserves the old vault and excludes generic SQL/filesystem/shell capabilities. The following details need explicit resolution before implementation is considered ready for security review.

## Findings and required changes

### B-H1 — High: application commands need explicit permission registration

The current proposal says “least-privilege Tauri capabilities,” but does not state the special treatment of application commands. Tauri documents that commands registered with `invoke_handler` are accessible to all application windows/WebViews by default; `AppManifest::commands` changes this behavior. A developer can correctly restrict plugin capabilities and still accidentally expose every vault command to a secondary/help window. Overlapping capabilities also combine privileges. [Primary source: Tauri capabilities](https://v2.tauri.app/security/capabilities/).

Required gate: register all vault commands in the application manifest, define explicit permissions for the intended window, explicitly select capabilities in configuration, avoid window wildcards/remote origins, and test denial from an unprivileged secondary WebView and after navigation. Do not expose window creation to low-privilege content. Application-specific scopes must be checked in Rust; declaring a scope alone does not enforce it. [Primary source: Tauri command scopes](https://v2.tauri.app/security/scope/).

### B-H2 — High: reveal/copy/edit remain authority available to frontend compromise

The draft acknowledges malicious reveal/copy calls, but “consider native confirmation” leaves the product's defense unresolved. Omitting bulk export does not prevent a loop over `list` plus `reveal`; a malicious frontend could also overwrite/delete entries. A native unlock prompt only protects password entry, not all subsequent authorized calls.

Required gate: choose and document the threat model before implementation. Either accept that frontend compromise exposes/manipulates an unlocked vault, or require native per-operation authorization for the sensitive operations it must resist. A rate limit alone does not establish consent. Keep password changes, migration and backup destinations behind native flow; arbitrary frontend-supplied destinations must never become an export mechanism. Demonstrate the selected behavior with malicious command-call sequences. The locked-file encryption claim remains meaningful even if unlocked frontend compromise is excluded.

### B-H3 — High: locking needs backend authority and a defined operation boundary

“Clear fields and reject late responses” does not define races around native clipboard writes, saves or unlocking multiple handles. Frontend idle timers can pause or be bypassed. Closing a window is not equivalent to locking an app process. A background request may retain a DB connection/key after a visible lock if ownership is spread across tasks.

Required gate: one Rust vault owner, backend monotonic idle timing, native OS sleep/screen-lock events, bounded operation queue and per-session generation numbers. Validate the generation immediately before every secret response or clipboard side effect. Define whether an in-flight transaction finishes before the lock acknowledgment; do not claim “locked” while secret-bearing tasks remain runnable. Cancel queued operations, close all handles and clear transient editor/search state on lock. Test lock/unlock while earlier requests are pending. State explicitly that JavaScript strings and WebView copies cannot be reliably zeroized.

### B-H4 — High: exact legacy bytes can be lost at Rust/JavaScript boundaries

The migration contract promises exact integer preservation, while frontend numbers are not exact for every SQLite signed 64-bit value. The legacy UI normally generated contemporary millisecond timestamps, but the database schema does not enforce their range. Passing an ID or timestamp through JSON as a number can silently round unusual but valid values; using `null` and empty string interchangeably also violates the mapping contract.

Required gate: migrate directly between native database handles without a UI round trip. Encode IDs/revisions and arbitrary 64-bit integer representations as decimal strings on the IPC boundary, with explicit option/null handling. Test `2^53 ± 1`, signed 64-bit boundaries, out-of-range timestamp values and empty/null fields. Text and password input length limits must reject unsupported values explicitly rather than truncate them.

### B-H5 — High: shipping SQLCipher linkage remains the central encryption blocker

There is no selected binding/build configuration or shipping binary. A “bundled SQLCipher” feature can pin a different source version than the app's intended release, link another SQLite implementation through a dependency, or change crypto provider between CPUs. Rust does not protect against configuration errors or unsafe native-code bugs.

Required gate: choose and pin binding, SQLCipher source, provider and build flags; inspect each architecture's linkage; run E1–E7 using the packaged app's actual storage path. Assert encryption before the first sensitive write, not only at reopen. An unknown pragma result is a failure. Reconstruct a synthetic fixture with the historic app's actual driver/opening sequence because explicit compatibility-3 fixtures alone do not prove the released old app used that profile. Never test with a real vault in CI.

### B-M1 — Medium: macOS 13+ is a build target, not a security-support policy

The proposed rolling “latest plus two” runtime matrix and broad “13+” target are different commitments. Compatibility on an old OS does not imply that its embedded WebKit receives required security fixes. The Tauri minimum floor says little about the complete Rust/native/provider/plugin bundle.

Required gate: distinguish build floor, tested OS versions and security-supported OS versions in release notes. Confirm exact versions/CPUs with the owner before committing to old releases. Never promise future beta compatibility. Maintain physical/VM target coverage, verify minimum deployment versions of every native library, and rerun packaged storage/lifecycle tests when WebKit or native dependencies change. A Playwright test in Chromium is insufficient evidence for WKWebView behavior.

### B-M2 — Medium: single-file and atomic-file behavior need operational precision

The shared design correctly permits encrypted rollback journals, but the UI must prevent users treating an open main DB file as a complete backup. Migration's “existing destination is an error” cannot safely be implemented as an existence check followed by an overwriting rename. File selection/canonicalization checks can race with path changes. `PRAGMA synchronous=FULL` does not itself establish the complete copy/rename durability protocol.

Required gate: a native consistent-backup command or explicit closed-vault copy procedure; exclusively created private temporary files; no-follow/file-identity checks; an atomic no-replace finalization primitive; disk-full/crash tests at each boundary. Distinguish fresh-destination migration from deliberate existing-vault replacement during master-password rotation. The latter must retain a verified recoverable old vault until replacement is durable. Source and destination filesystem assumptions must be explicit.

## Residual risks after the changes

SQLCipher's page authentication does not establish freshness, prevent valid old-file replacement, or guarantee secure deletion. Master-password strength remains crucial for offline attacks. Clipboard propagation, WebView/OS caches, system crash handling and unlocked memory remain outside a simple encrypted-file claim. Supply-chain integrity and security updates are ongoing obligations. B's architecture can satisfy the stated storage and migration requirements, but only the required implementation evidence can support a release decision.
