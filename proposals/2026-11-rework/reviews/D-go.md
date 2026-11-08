# Adversarial review D — Wails / Go / CGo / SQLCipher

Reviewer: separate adversary agent `/root/adversary_go`, 2026-11-08.

Scope: Option D initial local draft based on proposal branch head `40abeca045b236de98e6d0061de2a1b13dd6ebd9`, with D and the UI/discussion documents added locally on 2026-11-08. Reviewed OPTIONS, LEGACY-AND-MIGRATION, ENCRYPTION-EVIDENCE, UI-COMPARISON and DISCUSSION-DECISIONS. Primary Wails documentation, selected v3.1.0 source and Go CGo rules were checked. No app, encryption, migration or macOS runtime tests were run.

The owner's v1 decisions are binding: no commissioned external human audit; migration restarts from an untouched encrypted backup into a fresh encrypted destination, verifies every entry and discards interrupted/incomplete output; no checkpoints or resumable migration; normal edits/deletes use SQLCipher/SQLite transactions and built-in recovery. This review does not reinstate an external review requirement or propose a custom recovery layer.

## Verdict

**Conditionally viable as the Go alternative to B.** D can use essentially the same React UI and the same SQLCipher format as A/B. This does not establish a working Go/SQLCipher bridge, correct Wails authorization or safe locking. Initial work should prove the packaged storage path and bridge/lifecycle behavior before UI polish. The existing test gates and adversary-agent review are the v1 verification route; no implementation or certification is claimed.

## Findings and required changes

### D-H1 — High: the proposed CGo bridge needs an explicit ownership and byte-preservation contract

“Minimal reviewed bridge,” explicit key length and native parameter binding are necessary, but insufficient to prevent retained Go pointers, freed C buffers or accidental truncation. The [CGo rules](https://pkg.go.dev/cmd/cgo) constrain which Go memory C may retain; immutable Go strings and temporary conversions are not lifetime management. Native SQLite bindings may also retain buffers until reset/finalization, depending on destructor arguments.

Required change/gate: document allocation, ownership, lifetime and cleanup for keys, bound values and read results. A small C shim can use SQLite's copying binding convention (`SQLITE_TRANSIENT`) where appropriate, explicit byte lengths and deliberate cleanup; do not retain transient Go pointers in C. Copy returned column bytes before stepping/resetting/finalizing. Check length conversion/overflow and errors. Do not infer string lengths with `strlen`, trim/normalize passwords, or silently coerce null to empty. Include embedded NUL, quotes, long values, UTF-8, exact int64 boundaries and failed-open/statement-finalization cases in bridge tests. Migration must stay native and preserve original representations or reject unsupported values explicitly. Ordinary JSON/JS round trips must not become a data-normalization stage.

### D-H2 — High: a facade limits method exposure, but additional runtime surfaces must be excluded explicitly

The draft correctly says to inspect actual callable methods, validate in Go and reject unexpected callers. Wails' [service system](https://wails.io/features/bindings/services/) exposes exported service methods; TypeScript omission alone is not authorization. The selected [v3.1.0 service source](https://github.com/wailsapp/wails/blob/v3.1.0/v3/pkg/application/services.go) also supports mounting a service implementing `http.Handler`, and framework error handling can serialize/log errors.

Required gate: expose one narrow facade, avoid embedded/exported helpers and `http.Handler` implementations, and use redacted service errors that cannot include SQL parameters, keys or records. Demonstrate actual method allowlists and locked-session/caller checks from unauthorized windows, frames and navigation. If the pinned desktop bridge cannot reliably establish the necessary caller identity, keep one privileged window, prohibit secondary web content and fail the spike rather than trusting a frontend-supplied window ID. Treat all authorized frontend compromise as excluded from the unlocked-vault guarantee, as the draft already does.

Also explicitly exclude the optional Wails **MCP control build tag** and development/control listeners from shipping artifacts. Wails [documents](https://wails.io/guides/mcp-service/) that this optional feature can evaluate JS and invoke bound Go methods; its production exclusion is a concrete packaging requirement, not a generic security warning. The draft already excludes server mode and raw message handlers. Verify build tags and runtime listener absence in the artifact evidence; never load a real vault during development automation.

### D-H3 — High: backend generation checks alone cannot stop a previously queued frontend response

The draft has a good backend owner and immediate generation invalidation, but “check at reveal/copy delivery” needs two boundaries. A Go method can pass its check and return before lock, while its Promise result is delivered/rendered afterward. A separate check followed by clipboard write also races with lock unless their ordering is coordinated. Go context cancellation does not stop a running native KDF/SQLite call.

Required change/gate: capture a session generation in each frontend request and reject its result after every `await` if that generation is no longer current; clear outstanding editor/reveal state on lock. Serialize the backend generation check and clipboard side effect with lock admission. Keep handle lifetime valid during `sqlite3_interrupt` and drain statements before close; never free a handle still used by C. Define measured concealment/full-lock latency and transaction finish/rollback semantics. Test scheduled delivery before/after lock, lock during unlock/KDF/integrity scan/save, and clipboard writes at the lock boundary. This is lifecycle coordination, not custom database recovery.

### D-M1 — Medium: stable Wails version status does not select a secure complete build

The observed [v3.1.0 release](https://github.com/wailsapp/wails/releases/tag/v3.1.0) and recent stable-v3 transition support feasibility. They do not mean every optional platform, dev server or bridge route has identical security properties. The proposal correctly excludes experimental CGo-free GUI work and notes it cannot remove SQLCipher's CGo dependency.

Required gate: record the exact Go patch, Wails module/CLI/runtime, SQLCipher/provider, CGo flags and relevant advisory dispositions for the desktop configuration. Test the packaged build, not just `wails3 dev`. Do not infer macOS vulnerability or safety from a report about a different platform/mode. If B/D frontend code is shared experimentally, keep framework-specific adapters and tests separate. No runtime/storage benchmark or bundle-size advantage has been demonstrated.

### D-M2 — Medium: macOS UI parity and deployment remain testable hypotheses

The UI comparison is appropriately explicit that B and D can present equivalent web controls. It also correctly distinguishes native window/WebView hosting from native AppKit controls. Neither React reuse nor generated Go bindings guarantees focus, VoiceOver, clipboard ownership or screen-lock notification behavior. Native secure password entry remains a possible additional implementation, not a supplied capability of a file dialog.

Required gate: run the stated 15/26 comparison matrix with the actual release artifact on supported Intel/Apple Silicon combinations. Verify the minimum deployment version of the Go runtime and every C dependency, native lifecycle hooks and safe main-thread clipboard/dialog integration. Test keyboard/VoiceOver behavior and secret concealment in both versions. Retain A's explicitly agreed scope and label the same B/D scope as the comparison assumption; do not silently broaden v1 to older OS releases.

### D-M3 — Medium: the bridge and migration remain the unproven work, not a reason for custom recovery

The shared restart-from-backup algorithm is compatible with D. It still requires a real historical-driver fixture and field-by-field comparison from the newly opened destination before success. A partial file or item count alone is insufficient. A Go panic or process interruption must leave the source/backup untouched, and cleanup must identify only the attempt's own incomplete encrypted output and associated artifacts.

Required gate: E1–E9 plus adversary-agent implementation review (E10), tied to artifact hashes. Exercise interruption and fresh restart from the same backup, exact null/empty/int64/string values and the direct native comparison path. Let SQLite perform normal transaction rollback/journal recovery; do not introduce a migration checkpoint database or an edit/delete replay log. File replacement for password rotation still requires the shared handle-closure/ownership protocol.

## Residual risks

Go/JS garbage collection and serialization can retain copies of secrets; the draft is right to avoid an erasure guarantee. A compromised authorized frontend may read/manipulate unlocked data, regardless of a native unlock prompt. Clipboard replication, old encrypted backups, valid-old-file rollback, offline guessing and compromised unlocked processes remain outside the locked-file encryption claim. Choosing Go instead of Rust does not change the SQLCipher construction or supply proof of storage encryption.

Keep D on the shortlist if Go ownership and a customizable React interface are preferred. Resolve D-H1–D-H3 in the design and demonstrate them in the storage/bridge spike before trusting real-vault migration. The user-approved omission of a paid external review is respected; evidence is accurately described as tests and agent review.
