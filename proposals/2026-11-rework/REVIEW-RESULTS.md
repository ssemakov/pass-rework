# Adversarial review results and dispositions

Three separate adversary agents reviewed the options before presentation to the owner. Original draft snapshot: [`b7fede59`](https://github.com/ssemakov/pass-rework/commit/b7fede59b8acbc3d07bc59e53deacc210739c800). Original critiques are preserved; this document records the author's revisions. This is agent architecture review, not independent security certification. No implementation tests were run.

| Option | Reviewer / report | Verdict |
|---|---|---|
| A | `adversary_native` — [native review](reviews/A-native.md) | Conditionally viable; good bespoke macOS candidate; storage/lifecycle proof outstanding |
| B | `adversary_tauri` — [Tauri review](reviews/B-tauri.md) | Conditionally viable; justify cross-platform complexity and accept explicit frontend-compromise limits |
| C | `adversary_keepassxc` — [KeePassXC review](reviews/C-keepassxc.md) | Conditionally viable; reduces ownership; safe lossless importer not yet established |

## A dispositions

- **A1, high:** Added process-wide vault ownership and replacement protocol to LEGACY-AND-MIGRATION. Drain work and close all handles before replacing; reject aliases and disable replacement if ownership cannot be established. The coordination primitive and tests remain a blocking spike.
- **A2, high:** Retained historical driver/open-sequence fixtures as a migration gate. Actual shipped legacy profile and unusual key semantics remain unverified; source intent alone is insufficient.
- **A3, high:** Defined immediate session-generation invalidation outside a busy actor, concealed UI, delivery checks and final lock acknowledgment after handle/work cleanup. C task interruption and lock-latency budget remain implementation gates.
- **A4, medium:** Required explicit minimum-OS testing alongside the rolling matrix, and separated deployment, tested and security-supported OS claims.
- **A5, medium:** Added restoration/autosave/crash/accessibility/window-preview inventory and artifact checks. Complete memory erasure is still excluded.
- **A6, medium:** Kept packaged-binary encryption evidence and independent implementation review mandatory. PBKDF2 versus memory-hard KDF tradeoff is explicit; no homemade crypto wrapper.

## B dispositions

- **B-H1, high:** Added application-command manifest registration, explicit per-window permissions, Rust scope checks and denied-call tests. Capability JSON alone is not enough.
- **B-H2, high:** Explicitly excluded an authorized compromised frontend from unlocked-vault confidentiality/integrity guarantees. No bulk export does not prevent repeated reveal. Migration/rotation/backup paths require native flows; no arbitrary frontend output paths.
- **B-H3, high:** Added one backend owner, monotonic idle enforcement, native OS events, bounded queue and generation checks before responses/clipboard side effects. Full lock waits for backend cleanup; JS copies cannot be reliably zeroized.
- **B-H4, high:** Native-only migration; decimal strings for arbitrary 64-bit IPC integers, explicit nulls, >2^53 and boundary fixtures. No numeric rounding/truncation.
- **B-H5, high:** Binding/source/provider selection and shipping linkage remain blocking gates. Cipher identity must be asserted before sensitive writes; no fallback to ordinary SQLite.
- **B-M1, medium:** Separated runnable floor, tested releases and security-supported releases; require minimum-OS WKWebView checks.
- **B-M2, medium:** Added atomic no-replace migration finalization and distinguished it from deliberate rotation replacement. Filesystem races, consistent backup and interruption tests remain required.

## C dispositions

- **C-H1, high:** Concrete writer/no-plaintext integration and license choice remain unresolved blocking spike items. Stock XML-path CLI import is not claimed as a solution. No importer code/library is presented as already audited.
- **C-H2, high:** Exact ID/timestamp/null metadata now requires versioned typed/base64 encoding. Native strings must survive byte-exactly; reject unsupported values including NUL/XML-disallowed characters instead of sanitizing them. Require open/save/reopen verification.
- **C-H3, high:** Prohibited full-row archival secret copies. Metadata retains only ID/timestamp/null representations. Upstream history/recycle-bin/backup retention must be configured and tested; secure historical deletion remains excluded.
- **C-M1, medium:** Preserved exact version/platform limitations of existing reviews. Converter and target macOS client hashes need their own tests/review.
- **C-M2, medium:** Require both converter and client to pass oldest/current/prior OS tests, with prerelease results reported separately. Upstream minimum alone is insufficient.
- **C-M3, medium:** Added clean-account file-plus-password restore and disabled optional browser/network integrations for the pilot. Inspect encrypted save staging/backup artifacts; no mandatory Keychain/key file dependency.

## Recommendation after review

**Choose A for a bespoke macOS app; choose C if the goal is primarily a durable offline password manager with less software to maintain. Choose B if Linux/Windows support justifies owning the WebView/IPC boundary.** All satisfy the proposed portable-single-file model at design level, allowing transient encrypted save/journal artifacts. None currently has completed implementation-level storage proof. Do not migrate real data until the chosen path's gates pass.
