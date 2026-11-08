# Option D adversarial implementation review

Reviewer: separate adversary agent `/root/implementation_adversary`, 2026-11-08.

**Final source-review disposition: no unresolved blocking code findings.** The thirteen findings below were corrected and re-reviewed. This is an independent agent review with synthetic tests, not an external human audit or certification. A commissioned human audit is omitted from v1 as the owner requested.

Reviewed the current implementation based on published snapshot `c8f5ed6c65c1095079e8e6dcff13c46ac1167217`, including its subsequent IR-13 raw-key-literal correction and regression tests. Final native validation is separate from source-review completion: the designated PR rerun will be recorded here after the IR-13 correction is published and all results are observed. The earlier complete native/browser matrix is recorded below; it must not be mistaken for validation of later code changes. See [VALIDATION.md](VALIDATION.md) for the final tested commit and results.

## Scope and binding requirements

Reviewed the CGo/SQLCipher boundary, schema and CRUD helpers, filesystem ownership/publication, manager lifecycle, import/backup/rotation, Wails facade and transport gate, macOS clipboard/navigation/shutdown integration, React state delivery, regression tests, historical fixture provenance, build/signing/notarization scripts, and README/security/migration/distribution/validation documentation. Relevant pinned Wails and SQLCipher source was inspected to establish native header injection, flag delivery, clipboard/main-thread behavior and pragma result types.

The owner-approved migration contract is implemented: preserve the encrypted original backup untouched; write a fresh encrypted destination; reopen it and compare every field of every entry before reporting success; discard an owned incomplete attempt and restart from the backup after interruption. There are no checkpoints or resumable row state. Normal edits/deletes use SQLite transactions and built-in journal recovery, without a custom replay/recovery layer.

## Findings and final dispositions

All descriptions in the issue column refer to earlier implementation snapshots, not unresolved defects in the reviewed source.

| ID / severity | Issue found | Correction and retained evidence |
|---|---|---|
| IR-01 / High | `cipher_status` returns TEXT but was checked as an integer, rejecting every open | Typed text check; exact effective profile readback on every connection; real encrypted create/open/wrong-key and configuration tests |
| IR-02 / High | Hard-link publication could leave a two-link vault rejected after interruption | Descriptor-relative atomic no-replace rename (`renameat2` / `renameatx_np`), publication/no-overwrite tests and killed-import restart |
| IR-03 / Medium | SQL `_` wildcard hid unexpected schema objects; version-table constraints/orphans were not validated | Exact object enumeration, canonical current-format definitions, type/UTF-8 checks and `foreign_key_check`; malicious-trigger/table/orphan fixtures rejected |
| IR-04 / Medium | Interruption before completing the stage marker prevented restart | Safe cleanup of empty or exact-prefix partially marked private stages; unexpected files/owners/aliases still refused; setup-state regression tests |
| IR-05 / Medium | Reopening a migration candidate replaced the connection captured by deferred cleanup | A closure closes the current verification handle on all exits; statements finalize before return and candidate cleanup follows handle closure |
| IR-06 / High | Main-thread shutdown could wait on a manager operation waiting for native clipboard dispatch | Background shutdown draining through `ShouldQuit`; main-thread hook only clears native owned clipboard state; coordinator regression and real native quit path |
| IR-07 / High | Runtime fallback/request routes were broader than the intended single-window facade | Native-overwritten exact window ID, per-run capability, strict POST/envelope/method allowlist, bounded ordinary dispatch, native navigation/popup denial and CSP; unauthorized/helper requests and real native navigation denial tested |
| IR-08 / High | Late status/reveal responses could undo concealment or show A's password under B | Frontend session fences and reveal nonce, fresh generation on authentication, generation-tagged lock events, single-flight polling/busy ordering and lock acknowledgment; delayed-response browser regressions |
| IR-09 / High | Queued opens could reopen after an admitted lock; `Close` was not terminal | Generation-bound native selection grants and `UnlockAt`/`CreateAt`/`ImportAt` admission, post-open `finishOpen`, terminal close and active-generation checks; deterministic stale-open/close tests |
| IR-10 / High | Ordinary runtime slots could fill and reject Lock before invalidation | Fully authenticated/parsed allowlisted Lock bypasses ordinary capacity admission; independent reproduction and retest plus repository priority test |
| IR-11 / Medium | Idle timeout waited on the worker mutex before invalidating | Atomic open-session indicator admits idle lock before worker draining; activity/session fields initialize before publishing it; blocked-worker idle regression |
| IR-12 / High | Rotation could supersede a lock/close admitted during replacement and reopen under a new epoch | Rotation reuses `finishOpen(originalToken)` after reopening; deterministic clipboard-boundary Lock/Close tests reject the reopened session while retaining a usable new-key file |
| IR-13 / High | SQLCipher's reserved raw-key literal formats bypass password PBKDF2 despite a normal `cipher_settings` readback | Normal opens and all fresh-vault/snapshot write boundaries reject exactly the upstream raw-key forms; legacy read-only import/preservation remains compatible; six genuine raw-key forms, near misses, early rotation refusal and legacy conversion are tested |

Independent temporary overlay tests reproduced malformed-schema acceptance, queued post-lock unlock, Lock admission starvation and blocked-worker idle admission before their fixes. Corrected stale-generation/terminal-close and priority-Lock tests passed independently. A final rotation overlay blocked clipboard cleanup after replacement, admitted Lock, and confirmed `ErrLocked` plus new-key file usability. Equivalent critical regressions are retained in the repository; the reviewer did not edit implementation files.

For IR-13, the reviewer independently inspected the pinned `sqlcipher_cipher_ctx_key_derive` branches: exact case-insensitive `x'hex'` forms of total length 67, 99 or 163 bytes directly install key material, optional salt/HMAC key and skip password PBKDF2. The new validator matches those exact ASCII forms without trimming or changing ordinary passphrases. An independent overlay tested all six prefix/length combinations at create/snapshot boundaries and a near miss. It also created a genuine fixed-salt raw-key legacy source, observed its supplied salt in the file header, imported it to an ordinary passphrase, and confirmed every selected secret plus untouched source bytes. These tests passed; the repository retains broader preservation/rotation regressions.

## Final storage and bridge assessment

The new storage path compiles the pinned SQLCipher amalgamation into the application and uses `sqlite3_key` with explicit lengths. It checks engine identity, cipher status and exact expected profile settings before sensitive use. Reserved raw-key literals are rejected for normal vault passwords so settings readback cannot mask a bypass of the documented password KDF; historical raw-key inputs are accepted only through legacy import/preservation. Values use copying native parameter binding; result bytes are copied before stepping/finalization. Supported NULL/empty/NUL/Unicode/signed-int64 values remain distinct. Unsupported types/invalid UTF-8 are rejected rather than silently normalized during migration. Controlled native key/bind/clipboard buffers are wiped and freed; this does not imply complete Go/JavaScript heap erasure.

Normal sessions hold SQLite EXCLUSIVE ownership as well as the application lock, excluding other SQLite clients. Edits/deletes use standard SQLite transactions. Live backup and rotation use `writeSnapshot` from the owned connection, not a raw copy of an exclusive-mode database with a retained journal. Each snapshot writes fresh encrypted storage, closes/reopens it, checks schema/cipher/integrity, compares entries/sequence/current metadata and publishes without overwrite. The optional legacy preservation operation separately makes a byte-identical encrypted copy of a clean closed source under a SQLite read transaction and verifies it. Its source is never rekeyed or repaired.

Fresh files/stages use private permissions and reject unsafe aliases, ownership and changed identity. Publication uses native directory descriptors and atomic no-replace operations; rotation deliberately replaces only after verification and closure, retaining a verified encrypted old-password backup. Normal transaction recovery remains SQLite's responsibility. The 256 MiB file, 100,000-row and value/input limits and the trusted local-parent/same-user-process boundary are documented.

The sole registered facade exposes domain operations and generation-bound native dialog grants, rather than arbitrary paths or SQL. Grants may be reused for wrong-password retry within their exact generation; completed destinations still cannot be overwritten. Wails' native request wrapper overwrites the window ID before middleware. The authenticated strict gate rejects alternate query/chunk/helper transports before binding dispatch. The pinned native navigation category confines the privileged WebView to bundled main documents; future Wails updates must re-review that integration.

Framework logs are discarded because Wails debug messages can contain binding arguments/results. React renders vault fields as text. Backend lists omit passwords, frontend awaited results are fenced, clipboard checks/writes serialize with lock admission, and lock invalidates before native interruption/draining. Native clipboard change-count ownership prevents clearing later foreign content. The shutdown coordinator preserves the native main thread while draining. These are lifecycle controls, not an application recovery log.

## Verification and its limits

Independent local final-source checks completed successfully:

- `go test -race ./internal/vault ./internal/service`: vault **23.693 s**, service **1.025 s**.
- Frontend Vitest: **3 tests passed**.
- Independent deterministic rotation/lock overlay: passed, including reopening the final file with the new key after denying the session reopen.
- Independent raw-key boundary/legacy conversion overlay: **passed in 0.662 s**, including a genuine caller-supplied-salt source demonstrating the upstream raw-key branch.
- Static review of the seven-scenario browser suite, native assertions, negative encryption tests, real historical-driver fixture and distribution scripts. The implementer runs the full browser suite; the reviewer separately checked CI conclusions/log markers.

Using read-only GitHub tools, the reviewer independently checked [run 42531608409](https://github.com/ssemakov/pass-rework/actions/runs/42531608409) at `96aca33f67f67cbab1f35ed085553a90b7afb819`: browser plus all four macOS 15/26 Intel/Apple Silicon jobs concluded success. Logs for native jobs `128886814209`, `128886814236`, `128886814259` and `128886814393` each contain **“Native WebKit smoke passed: explicit UI assertions complete”**. The jobs also passed native race tests, package/signature/linkage checks and packaged-binary vulnerability scans. A clean exit without that assertion marker is explicitly rejected by the smoke script.

That matrix establishes real native behavior for its stated commit. The final per-handle profile enforcement, rotation race correction, raw-key restriction and latest UI/packaging changes require the superseding final run; pending evidence is not an unresolved source finding, and no later artifact is declared tested merely because its predecessor passed.

The historical fixture uses the original pinned Journeyapps/Knex path on Linux and reports SQLCipher 4.2.0/profile 4 despite the legacy compatibility-before-key call. Explicit profile-3 and profile-4 fixtures provide additional format coverage. This is stronger than a format assumption, but is not inspection of an original shipped macOS legacy binary.

No real vault was opened or migrated by this review. No commissioned audit, mathematical application proof, complete memory erasure, rollback freshness guarantee, future-beta guarantee, Developer ID/notarization result or full VoiceOver conformance is claimed. Those limits and owner-controlled release activities are accurately separated in the security/distribution/validation documents; they do not reinstate a paid-review requirement for v1.
