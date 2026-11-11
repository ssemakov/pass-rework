# Adversarial review A — native SwiftUI/AppKit + SQLCipher

Historical review of the original proposal. Later owner decisions supersede the former macOS 13 scope and any requirement for a commissioned external human audit in v1; tests and adversary-agent review remain. See [current discussion decisions](../DISCUSSION-DECISIONS.md) and [dispositions](../REVIEW-RESULTS.md). The original findings below are preserved.

Reviewer: dedicated adversary agent for Option A. Date: 2026-11-08.

Scope: OPTIONS, LEGACY-AND-MIGRATION and ENCRYPTION-EVIDENCE at proposal snapshot [`b7fede59b8acbc3d07bc59e53deacc210739c800`](https://github.com/ssemakov/pass-rework/commit/b7fede59b8acbc3d07bc59e53deacc210739c800). This review challenges the proposed design; it does not audit code or certify encryption. No app, vault migration or macOS execution test was run.

## Verdict

**Conditionally viable and a reasonable bespoke macOS direction, but not ready for production implementation sign-off or real-vault migration.** Reusing SQLCipher avoids designing a new storage cipher. Native UI removes the old renderer/Node boundary, but does not establish key secrecy, migration correctness or crash safety. The drafts are candid about missing evidence. The following gaps must remain visible until resolved.

## Findings, ordered by severity

### A1 — High: replacement-based password rotation needs a real multi-process exclusion protocol

The proposal recommends copying, verifying and atomically replacing a vault for password changes, while normal CRUD holds SQLite connections. “Detect external replacement” does not specify when all handles close, how another app instance is excluded, or how the check avoids a race. An existing open handle can keep referring to the old inode after a pathname replacement. This is a corruption and lost-update issue, not merely an attacker concern. SQLite explicitly documents renaming/unlinking open databases as unsafe ([SQLite corruption guidance](https://www.sqlite.org/howtocorrupt.html#_unlinking_or_renaming_a_database_file_while_in_use)).

Required change/gate: specify single owning process per vault, lifecycle-wide coordination covering all windows, and a replacement protocol that drains work and closes every app-owned handle before replacement. Reject rotation/restore when ownership cannot be established. Check identity immediately before commit/replacement and invalidate all previous sessions afterward. A lock on the old database inode alone does not solve pathname replacement. Test a second process and old open connections, not just concurrent Swift tasks. Reject hard-linked aliases as well as symlinks or define their handling explicitly.

### A2 — High: legacy compatibility is a hypothesis until the historical opening path is exercised

The historic app requests compatibility before supplying the key; the proposal correctly avoids treating that as definitive proof of a v3 file. Trying 3 and 4 with a modern library is a sensible diagnostic, but it does not establish the semantics of the shipped driver, quoted passwords, Unicode or historical raw-key conventions.

Required gate: create synthetic files through the exact historic storage/driver path and verify them with the shipped engine candidate. Record actual cipher parameters, order and driver versions. Include apostrophes, non-ASCII composed/decomposed inputs, empty-string policy and any key interpretation inherited from the legacy API. Never silently reinterpret a key or “repair” an input to force a match. A local user vault should be attempted only after these fixtures pass; retain an escape route for an unrecognized historic profile. The proposal already states the original binary was not inspected; retain that limitation in the decision summary.

### A3 — High: an actor does not make lock immediate or cancel synchronous C work

The actor can serialize reads and writes, but a long synchronous SQLCipher operation can delay a queued lock. Task cancellation is cooperative and does not automatically interrupt SQLite or erase result buffers. Native state can also receive a completed reveal after the user has seen the lock screen unless delivery is guarded separately.

Required change/gate: define a lock linearization point and maximum lock latency. Immediately invalidate the session generation and conceal UI on lock request; block new secret delivery while draining/interruption/handle closure completes. Check generation both before and after reads and at delivery. Decide whether an in-flight write completes or rolls back and report its result accurately. Test lock while KDF, integrity check, reveal and write are in progress. Avoid claiming “locked” means keys have already been wiped while cleanup remains pending.

### A4 — Medium: the macOS 13 deployment floor is not covered by a newest-plus-two test policy

An SDK deployment target allows launch eligibility; it does not test the oldest supported OS. A rolling latest/previous-two matrix can omit macOS 13 while the binary still claims 13+. Swift concurrency back-deployment, AppKit lifecycle events, SQLCipher provider integration and compiler-generated runtime assumptions all need target testing. A universal binary alone proves neither CPU architecture executes correctly.

Required change/gate: test the declared minimum OS explicitly in addition to the rolling matrix, or narrow the published supported range. Name exact OS/CPU combinations in each release's support manifest. Confirm the selected Xcode/Swift package/native library recipe can actually build those combinations before advertising them. Prerelease testing is a compatibility signal, not a forward guarantee.

### A5 — Medium: native UI persistence must be part of the plaintext-artifact threat model

The draft covers logs, temporary files and Swift string copies, but needs explicit treatment of window restoration, autosave/scene storage, crash reporting, accessibility and app-switcher/window previews. Secret editor/reveal values must not enter restoration/preferences/document-autosave mechanisms. Clipboard timeout does not retract data already consumed by another process or clipboard service.

Required change/gate: inventory every path that can persist or expose a revealed/editing secret; disable inappropriate persistence and redact diagnostic collection. Exercise screen lock/sleep and window restoration with synthetic sentinels. Document accessibility and clipboard exposure as unlocked-use tradeoffs. Keep memory erasure claims limited; SwiftUI controls and immutable strings make blanket zeroization claims untenable.

### A6 — Medium: the encryption proof remains a required work product, not an inherited property

SQLCipher's documented construction is credible, but no evidence currently ties the proposed shipping Swift binary to that construction. Runtime cipher identity checks, physical artifact checks and negative tests are essential because a build can accidentally use ordinary SQLite. Default PBKDF2 is interoperable but less resistant to parallel guessing than a well-calibrated memory-hard KDF. The correct answer is to expose that tradeoff, not invent a wrapper.

Required gate: E1–E10 remain release gates. Include artifact-hash-specific evidence for both architectures and an independent review of the actual native bridge and lifecycle. Audit scope must include C/Swift conversions, explicit key byte lengths and failed-open cleanup. Preserve the stated limitations: no automatic protection against weak passwords, valid old-vault replay, compromised unlocked endpoints or deletion from historical copies.

## Residual risks even after these changes

- SQLCipher may leave authenticated old values in encrypted free pages or retained backups. Delete means active-record removal; snapshots and other copies remain outside its control.
- One portable committed file still needs transient encrypted journal/save artifacts for safe writes. The drafts disclose this correctly. A zero-auxiliary-files interpretation would require a different storage policy and cannot be met by disabling journaling.
- SQLite atomic transactions rely on filesystem/storage behavior. Crash tests give evidence for tested conditions, not universal power-loss guarantees ([SQLite atomic-commit design](https://www.sqlite.org/atomiccommit.html)).
- Native UI lowers dependency and IPC complexity relative to Option B; the app still owns a security-sensitive update, distribution, migration and maintenance lifecycle.

## Recommendation to the owner

Keep A on the shortlist if a bespoke macOS app matters. Approve only a bounded storage/migration and native lifecycle spike next. Require A1–A4 to be designed and demonstrated before trusting migration or password rotation. A5–A6 must be included in the shipping security evidence, with no claim that this agent review substitutes for independent implementation review.
