# Option D adversarial implementation review

Reviewer: separate adversary agent `/root/implementation_adversary`, 2026-11-08.

Status: **interim review of work in progress** on `implementation/option-d`. This document will be updated after the remaining implementation and tests are available. It is an agent code review, not an external human audit or certification.

## Scope and requirements

Reviewed the native SQLCipher boundary and the initial vault file/schema/CRUD helpers, against the proposal and recorded discussion decisions. The desktop facade, migration orchestration, rotation, frontend and tests were not yet present at this first pass. Their absence is not reported as an implementation defect at this stage.

The owner's requirements remain: preserve the encrypted original backup untouched; use a fresh encrypted migration destination; reopen it and verify every entry before success; discard interrupted candidates and restart from the backup, with no resumable checkpoint. Normal edits/deletes use SQLite transactions and its journal recovery. A commissioned external human audit is omitted from v1.

## Findings

### IR-01 — High / resolved: cipher-status type mismatch prevents every successful open

`internal/cipher/cipher.go` tests `status[0][0].Int != 1` after `PRAGMA cipher_status`. The pinned SQLCipher amalgamation implements that pragma with `sqlcipher_vdbe_return_string(..., "1", ...)`, so the bridge returns `Value{Kind: Text, Text: "1"}` and `Int` remains zero. Correctly encrypted vaults therefore fail with `ErrCipher`.

Correction observed: the bridge now checks `Kind == Text` and `Text == "1"`. `TestEncryptedCRUD` and `TestCipherStatusAndTamper` exercise the real native create/open path and passed in this environment. Additional row/column shape hardening is a minor remaining defensive improvement for native pragma results.

### IR-02 — High / resolved and tested: hard-link publication window

`internal/vault/files.go` publishes a staged candidate by creating a hard link at the final filename, then unlinking the candidate. A process interruption between those operations leaves both names pointing to the same inode (`Nlink == 2`). `regular()` rejects any vault with multiple links, while the next migration attempt immediately refuses the existing final destination. The verified vault is stranded until manual filesystem repair, contrary to the agreed interruption behavior.

Correction observed: `rename.go` now uses Linux `renameat2(..., RENAME_NOREPLACE)` or macOS `renameatx_np(..., RENAME_EXCL)` relative to already opened source/destination directory descriptors. This removes the two-hard-link state and refuses an existing destination atomically. `TestStagingInterruptedSetupAndAtomicPublication` verifies single-link publication and the import suite rejects an existing destination. A separate killed-process test exercises discard/restart during row insertion. This is migration file publication, not a custom SQLite transaction-recovery layer.

### IR-03 — Medium / resolved and tested: schema validation accepts unexpected write behavior

`schema()` filters names with `name NOT LIKE 'sqlite_%'`. In SQL `LIKE`, `_` matches any single character. A user-defined trigger named `sqliteXwipe`, for example, is consequently omitted even though it does not have SQLite's reserved `sqlite_` prefix. Such a trigger can change the effect of later writes while the schema check accepts the file.

Correction observed: enumeration now excludes only the exact expected `sqlite_sequence` object. Current-format tables must match their exact supported definitions and `foreign_key_check` must return no violations. `TestSchemaTamperingAndAliases` rejects the hidden-prefix trigger, malformed version table and orphan version fixture before normal operations. The expanded native suite passed independently.

### IR-04 — Medium / resolved and tested: staging setup interruption

`newStaging()` creates its deterministic private directory before writing its ownership marker. Interruption after `Mkdir`, before the marker is completely written, leaves an empty or partially marked stage. A subsequent attempt requires the complete marker and refuses that leftover directory, so it cannot discard the incomplete output and restart automatically.

Correction observed: an absent or exact-prefix partial marker is accepted only in an otherwise empty private stage; files other than the marker still require the complete marker. Foreign ownership, symlinks, broad directory permissions and unexpected files remain rejected. `TestStagingInterruptedSetupAndAtomicPublication` reconstructs these interrupted setup states and passes; `TestUnexpectedStageNeverDeleted` preserves unrelated contents.

### IR-05 — Medium / resolved in code: reopened migration verification handle cleanup

`Import()` defers `b.Close()` on the initial destination connection, closes that connection, then assigns a newly opened verification connection to `b`. The original deferred method captures its original receiver. Failures in the reopened connection's schema, integrity or comparison checks consequently return without closing that new native handle. Its key and native resources remain live while the stage is discarded.

Correction observed: cleanup now uses a closure over the current `b`, so it closes the reopened verification connection on error. The bridge's `Close` is nil-safe. A failure-path integration test would strengthen this check.

### IR-06 — High / code corrected, lifecycle tests pending: shutdown/clipboard deadlock

The pinned Wails `cleanup()` invokes shutdown hooks synchronously on the AppKit main thread. The configured hook calls `manager.Close()`, which waits on the manager's admission/state locks. Concurrent `Manager.Copy` holds those locks while `desktop.Clipboard.Write` calls `application.InvokeSync` and waits for the same main thread. Shutdown can therefore block the main thread waiting for a copy whose queued main-thread callback cannot run. Timer/lock clipboard clearing has the same lock-order risk.

Correction observed: `ShouldQuit` now initiates manager draining on a background goroutine and returns to the native event loop; only after completion does it call `app.Quit`. The main-thread shutdown hook performs native last-owned clipboard cleanup directly and does not wait for the manager. Test this ordering deterministically, including pending clipboard write and clear callbacks during shutdown.

### IR-07 — High / open verification gate: window context is not independent caller identity

The desktop source identity is stronger than the first pass established: pinned Wails `webViewAssetRequest.Header()` clones the request headers and overwrites `x-wails-window-id` with the native window ID (and its name for this named window). This native overwrite was independently checked in `application.go`; frontend headers cannot impersonate another native window through the desktop scheme path. Downstream Wails nevertheless has a first-window fallback when identity is absent. `SecureAssets` currently allows missing Origin/Referer, and the desktop runtime handler does not enforce POST; it accepts query-encoded calls as well. Native prevention of navigation to other content is not yet configured.

Required correction/gate: require the exact native-injected window identity before Wails' fallback, enforce the precise allowed runtime request shape and trusted content before dispatch, prohibit secondary/external content, and exercise these paths against the packaged macOS WebView. Custom-scheme Origin behavior needs runtime verification. A capability put in `app.Flags` alone would not distinguish external content: the pinned macOS navigation-finished hook unconditionally injects runtime flags into newly navigated pages. No exploit on macOS has been reproduced by this Linux-only static review; this is a concrete authorization/design gate, not a claim that ordinary remote pages already bypass the full platform.

### IR-08 — High / partially corrected, delayed-response tests pending: late frontend responses

Most `run()` and list results have a session fence, but `updateState()` applies its awaited status without one. A pre-lock unlocked response delivered after `scrub()` can restore the unlocked workspace. Polling likewise accepts a response merely because its generation differs, without establishing that it is newer/current.

A pending reveal also has no entry/interaction identity. Entry-list buttons remain selectable while the reveal is pending: select A, request its password, select B, then deliver A's response. The response sets the shared `revealed` state and displays A's secret beneath B's title. Escape/search/panel changes can also clear a reveal and then have the pending result restore it.

Correction observed: state requests now capture the session fence and reject older generations; concealment increments a reveal nonce which guards late secret delivery; picker workflows check their fence between awaits. The reveal and multi-picker paths are corrected in code.

Remaining status ordering: a poll can read `locked:true` before an unlock finishes, then arrive after the explicit state refresh applied `locked:false` for the same generation. Both responses have the same fence and generation, so the stale poll scrubs the new session. Later unlocked polls with that same generation do not update the locked UI. Invalidate older status requests at authentication completion and/or use coordinated request serials, with deliberate admission for locked-to-unlocked transitions. Add delayed-response tests for this same-generation case as well as lock, entry changes and explicit concealment. The rotation callback also sets a nonsecret notice after an additional awaited state refresh; that does not restore a secret but can produce stale feedback.

### IR-09 — High / corrected and independently retested: queued unlock after lock / nonterminal close

`Manager.Unlock` and `Create` capture the epoch only after acquiring the state mutex. An unlock that has consumed its native selection token can queue while `Lock` holds that mutex during cleanup, acquire it after lock completes, capture the new epoch and open the vault despite the admitted lock. Explicit facade lock clears selected tokens only after manager locking completes, and native/idle locks do not clear them. Frontend result fencing conceals the late result but does not close this reopened backend session.

`Manager.Close` also stops the watcher and calls `Lock` without making future opens fail. A queued unlock can reopen after shutdown draining, when no watcher remains.

Correction observed: picker capabilities are generation-bound, the facade uses `UnlockAt`/`CreateAt` with an expected generation checked inside admission, and `Close` has a terminal atomic flag. An independent synthetic overlay test queued stale-generation `UnlockAt` behind blocked clipboard cleanup and verified rejection; a second test verified a closed manager cannot reopen. Both passed after correction. The production facade's generation-bound contract is the relevant boundary; repository tests should retain equivalent coverage.

### IR-10 — High / corrected and independently retested: ordinary request admission can starve lock

The hardened runtime handler limits all requests to eight occupied slots. Frontend polling starts another `State` every second even when prior calls are still waiting for the manager's state mutex. A long backup/storage operation can therefore accumulate blocked status calls and fill the slots. A subsequent explicit Lock request receives HTTP 429 before it reaches immediate epoch invalidation or native interruption.

Correction observed: strictly authenticated, parsed and allowlisted Lock calls are exempt from ordinary slot admission; polling is single-flight. The independent overlay test first reproduced HTTP 429 behind eight blocked State calls and passed after the correction. Lock can still wait for safe native draining, but its invalidation is no longer rejected behind queued ordinary work.

### IR-11 — Medium / open: idle lock waits for the storage worker before invalidating

`watch()` acquires the manager state mutex to decide whether a vault is open before calling `Lock`. A storage/native operation which holds that mutex beyond the idle deadline prevents timeout lock admission, epoch invalidation, UI concealment and native interruption until the operation ends. This defeats the intended immediate concealment/drain separation for idle locking.

Required correction: use an atomic session-open/admission indicator or equivalent nonblocking state to initiate timeout lock, then drain under the state mutex. An independent synthetic overlay test holds the worker mutex across an expired idle tick and observes that the epoch fails to advance; it currently fails as expected.

## Positive observations from the initial pass

- Keys use `sqlite3_key` with explicit byte lengths, not interpolated SQL; statements bind values with `SQLITE_TRANSIENT`, and text results are copied before stepping/finalization.
- The native cipher version is checked and native failures are redacted. The compile flags omit loadable extensions and keep temporary storage in memory.
- New files use exclusive creation and mode `0600`; lock files reject symlinks, foreign ownership and broad permissions. File helpers reject nonregular and hard-linked files.
- Create/edit/delete operations use native SQLite transactions; optimistic revisions detect stale writes.
- The migration comparison helper compares every selected field with its native null/text/integer representation and preserves the AUTOINCREMENT sequence, rather than checking only counts.
- The sole registered service exposes domain operations and one-use native selection tokens rather than arbitrary paths or SQL. All service methods fail closed when caller authorization is absent.
- Desktop framework logs are explicitly discarded because Wails debug output can contain binding arguments/results. The macOS clipboard uses a concealed-type marker and native change-count ownership rather than comparing the copied secret.

## Validation performed

Static review of `internal/cipher/cipher.go`, the initial vault helpers and manager/transfer orchestration, relevant SQLCipher pragma implementation, the D adversary proposal review and the discussion decisions. Ran `go test ./internal/vault` successfully with the provided Go toolchain and workspace caches; the two currently present tests use public synthetic data. No real vault was opened. No desktop or macOS runtime behavior has been established by this interim review.

An independent temporary Go overlay test, `TestAdversarialMalformedVersionSchema`, replaced only a synthetic encrypted vault's `item_versions` with `CREATE TABLE item_versions(id INTEGER, revision INTEGER)`. The test expects rejection and failed because `schema(d, true)` accepted it, confirming the remaining IR-03 concern. The test source/overlay lived in `/tmp`; no implementation files were edited by this reviewer.
