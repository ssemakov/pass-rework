# Adversarial review results and dispositions

Separate adversary agents reviewed the original A/B/C options before presentation to the owner. The expansion adds a D-specific review and a comparative UI review. Original draft snapshot: [`b7fede59`](https://github.com/ssemakov/pass-rework/commit/b7fede59b8acbc3d07bc59e53deacc210739c800). Original critiques are preserved as historical assessments; the owner’s later macOS and v1-review decisions supersede conflicting recommendations. this document records the author's revisions. This is agent architecture review, not independent security certification. No implementation tests were run.

| Option | Reviewer / report | Verdict |
|---|---|---|
| A | `adversary_native` — [native review](reviews/A-native.md) | Conditionally viable; good bespoke macOS candidate; storage/lifecycle proof outstanding |
| B | `adversary_tauri` — [Tauri review](reviews/B-tauri.md) | Conditionally viable; justify cross-platform complexity and accept explicit frontend-compromise limits |
| C | `adversary_keepassxc` — [KeePassXC review](reviews/C-keepassxc.md) | Conditionally viable; reduces ownership; safe lossless importer not yet established |
| D | `adversary_go` — [Go/Wails review](reviews/D-go.md) | Conditionally viable; prove CGo ownership, actual bridge surface and lock ordering |
| UI comparison | `adversary_ui` — [UI review](reviews/UI-comparison.md) | Suitable for comparing directions; clarify edit intent, imported provenance and full task parity |

## A dispositions

- **A1, high:** Added process-wide vault ownership and replacement protocol to LEGACY-AND-MIGRATION. Drain work and close all handles before replacing; reject aliases and disable replacement if ownership cannot be established. The coordination primitive and tests remain a blocking spike.
- **A2, high:** Retained historical driver/open-sequence fixtures as a migration gate. Actual shipped legacy profile and unusual key semantics remain unverified; source intent alone is insufficient.
- **A3, high:** Defined immediate session-generation invalidation outside a busy actor, concealed UI, delivery checks and final lock acknowledgment after handle/work cleanup. C task interruption and lock-latency budget remain implementation gates.
- **A4, medium:** Required explicit minimum-OS testing on macOS 15 and 26, and separated deployment, tested and security-supported OS claims.
- **A5, medium:** Added restoration/autosave/crash/accessibility/window-preview inventory and artifact checks. Complete memory erasure is still excluded.
- **A6, medium:** Kept packaged-binary encryption evidence mandatory. The original external-review recommendation is superseded: v1 uses adversary-agent implementation review, with no commissioned human review. PBKDF2 versus memory-hard KDF tradeoff is explicit; no homemade crypto wrapper.

## B dispositions

- **B-H1, high:** Added application-command manifest registration, explicit per-window permissions, Rust scope checks and denied-call tests. Capability JSON alone is not enough.
- **B-H2, high:** Explicitly excluded an authorized compromised frontend from unlocked-vault confidentiality/integrity guarantees. No bulk export does not prevent repeated reveal. Migration/rotation/backup paths require native flows; no arbitrary frontend output paths.
- **B-H3, high:** Added one backend owner, monotonic idle enforcement, native OS events, bounded queue and generation checks before responses/clipboard side effects. Full lock waits for backend cleanup; JS copies cannot be reliably zeroized.
- **B-H4, high:** Native-only migration; decimal strings for arbitrary 64-bit IPC integers, explicit nulls, >2^53 and boundary fixtures. No numeric rounding/truncation.
- **B-H5, high:** Binding/source/provider selection and shipping linkage remain blocking gates. Cipher identity must be asserted before sensitive writes; no fallback to ordinary SQLite.
- **B-M1, medium:** Separated runnable floor, tested releases and security-supported releases; require 15/26 WKWebView checks for the comparison baseline.
- **B-M2, medium:** Added atomic no-replace migration finalization and distinguished it from deliberate rotation replacement. Filesystem races, consistent backup and interruption tests remain required.

## C dispositions

- **C-H1, high:** Concrete writer/no-plaintext integration and license choice remain unresolved blocking spike items. Stock XML-path CLI import is not claimed as a solution. No importer code/library is presented as already audited.
- **C-H2, high:** Exact ID/timestamp/null metadata now requires versioned typed/base64 encoding. Native strings must survive byte-exactly; reject unsupported values including NUL/XML-disallowed characters instead of sanitizing them. Require open/save/reopen verification.
- **C-H3, high:** Prohibited full-row archival secret copies. Metadata retains only ID/timestamp/null representations. Upstream history/recycle-bin/backup retention must be configured and tested; secure historical deletion remains excluded.
- **C-M1, medium:** Preserved exact version/platform limitations of existing reviews. Converter and target macOS client hashes need their own tests/review.
- **C-M2, medium:** Require both converter and client to pass oldest/current/prior OS tests, with prerelease results reported separately. Upstream minimum alone is insufficient.
- **C-M3, medium:** Added clean-account file-plus-password restore and disabled optional browser/network integrations for the pilot. Inspect encrypted save staging/backup artifacts; no mandatory Keychain/key file dependency.

## D dispositions

- **D-H1, high:** Added explicit CGo buffer ownership/lifetimes, C-owned memory where required, copying binds with explicit lengths, copying read results before invalidation, overflow/error handling and NUL/int64/failure tests. No retained transient Go pointers or C-string truncation. The concrete bridge remains a blocking implementation spike.
- **D-H2, high:** Added a narrow non-HTTP facade, redacted errors, actual callable-surface tests and native caller evidence. V1 has one privileged WebView and no secondary web content; the required caller boundary must be proved. Excluded MCP/control/dev/server/raw surfaces and require artifact/listener evidence. No Tauri-style permission model is assumed for Wails.
- **D-H3, high:** Added frontend generation checks after awaits, serialized clipboard check/write versus lock, and valid C-handle lifetime through interruption/draining. Tests must establish the two delivery boundaries and distinguish visual concealment from completed lock.
- **D-M1, medium:** Kept exact Wails/Go/runtime/SQLCipher/provider/build-tag and advisory validation as artifact evidence; no unmeasured performance or bundle-size benefit is claimed.
- **D-M2, medium:** Preserved 15/26 as D’s comparison assumption, with native dependency, lifecycle, keyboard and accessibility tests on supported CPU/OS combinations.
- **D-M3, medium:** Retained historic fixtures/every-entry comparison, untouched-backup restart and built-in SQLite transaction recovery. No checkpoints or custom edit/delete replay logic.

## UI review dispositions

- **UI-M1:** C’s exact legacy ID/timestamp/null metadata is immutable import provenance. Native fields are authoritative after edits; never replay old metadata over a newer value. No bidirectional export in v1.
- **UI-M2:** Added explicit unchanged / blank / null edit intent. Never write a masked placeholder or an unrequested blank password; untouched fields remain unchanged. Test cancel and deliberate replacement.
- **UI-M3:** Added password-change and encrypted backup/restore parity rows, plus cancellation/failure and clean-account reopen checks for custom apps and C.
- **UI-L1:** Kept UI/native-feel/effort rankings qualitative; no UI, accessibility, size or speed measurements are claimed.
- **UI-L2:** Added visible supersession notes to preserved A/B/C reports. Owner decisions and current E10 govern v1; a paid human audit is not required.

## Recommendation after review

**Choose A for a bespoke macOS app; choose C if the goal is primarily a durable offline password manager with less software to maintain. Choose B for a Rust backend or D for a Go backend when a custom shared React UI justifies owning the WebView/bridge boundary.** All satisfy the proposed portable-single-file model at design level, allowing transient encrypted save/journal artifacts. None currently has completed implementation-level storage proof. Do not migrate real data until the chosen path's gates pass.

## Owner decision incorporated after review

The owner agreed that migration restarts from the preserved, untouched encrypted backup. Every attempt creates a fresh encrypted destination and verifies every entry before success. Interrupted/incomplete destinations are discarded; no checkpoints or resume support. Normal edit/delete operations use SQLCipher/SQLite transactions and built-in recovery without custom recovery logic. The migration contract and E6 gate now reflect this simplification; it does not resolve the outstanding implementation evidence or select a product option.


## Later chat decisions incorporated

- A’s minimum is macOS 15; test 15 and 26. No 13/14 support requirement. B/C/D use 15/26 as a comparison assumption.
- External paid human review is omitted from v1. Encryption/migration tests and adversary-agent implementation review remain required. Original review statements requiring an external audit are historical, not current gates.
- D is a Go/Wails alternative with SQLCipher/CGo and the same restartable migration contract; its UI can match B’s React implementation.
- Added [UI comparison](UI-COMPARISON.md) and [discussion decisions](DISCUSSION-DECISIONS.md), including the system-SQLite linkage risk and KDF explanation.
