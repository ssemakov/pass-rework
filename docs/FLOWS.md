> Updated for the owner's 2.0.2 requests: full paths, friendly recovery, V1 consent, same-screen import results and header Lock. Earlier validation results below remain tied to their named commits; see [2.0.2 verification](UX-2.0.2.md).

# Startup, settings and complete flow trace

Reference: legacy Pass at `7cd8c6c0bc395fa3676c151a84c1657cda8cb481`. `AuthNavigation.js` checks electron-settings `default_storage_path`: absent → signup; present → login. `Login.jsx` loads that path and validates the submitted password without opening a picker. `actions/storage.js` writes the path only after successful creation. These transitions are part of the reference alongside its screenshots.

The earlier rework preserved the screen styling but omitted this persistence. Its password-submit handler called `Choose` when no selection existed, and locks/import discarded that selection. Earlier tests explicitly invoked File → Open and reset the backend/files between tests, missing remembered login and process restart. Those gaps are now retained as regressions.

## Preferences

The macOS settings file is `~/Library/Application Support/PassV2/settings.json`; the initial vault destination is `storage.pass` in that directory. Directory/file permissions are `0700`/`0600`. Only `default_storage_path` and `idle_minutes` are saved. Passwords, keys and entries are never written there; the vault remains a separate encrypted portable file. Legacy settings and encrypted originals are untouched.

The backend loads settings once per instance and exposes opaque generation-scoped selection tokens, without a frontend method accepting arbitrary paths. Remembered/default grants are stable under status polling, renewed after lock, and retained when repeatedly choosing other files for import. A successfully created/opened vault or completely verified import becomes the default; choosing/cancelling/rejecting a key/failing an import does not change it.

Settings use bounded no-follow parsing and an atomic synced replacement. Ordinary vault edit/delete recovery remains SQLite's responsibility. A settings-save failure leaves an actually created/opened vault usable and clearly says to select it again after restarting. Unreadable/malformed settings show an error and explicit Open/Create/Import recovery. Existing vaults are never silently overwritten.

## Flow inventory

| Trigger | Result / retained checks |
|---|---|
| No saved path | Create Storage with the new/confirmation fields with live matching validation and folder control, plus Import/Open alternatives. No automatic picker or database read. |
| Matching creation password | Create a fresh encrypted destination, verify/open it, save its path. Mismatch only says “Passwords don't match”; existing destination is refused. |
| Restart with saved path | Preselected login asking only for the master password. Passwords and plaintext views are not restored. |
| Rejected login key | One submitted operation with bounded read-only format detection and “Wrong password”. Same login/selection supports a correct retry. No Choose/Create/Import/retry or full-vault scan follows rejection. |
| Missing/moved/unavailable file | Error on the same login; retain the saved choice. Explicit File → Open can locate it. Remember the new path only after successful authentication. |
| File → Open | Clear old password drafts; lock/acknowledge any active session; open picker. Cancel returns to remembered login, or first-run create if no path exists. Never submit the old draft to the chosen file. |
| File → New | Lock the old session; show creation form. Destination must be fresh. Escape returns to remembered login; prior setting remains until successful creation. |
| Import / restore | Explicit encrypted source and fresh destination; validate passwords; migrate, reopen and compare every entry. Save verified destination; keep “Imported N secrets” on the import screen. Explicit Open storage then shows its preselected login without a picker. Failure/cancel preserves previous setting and source. Interrupted staging is discarded/restarted as before. |
| Original source preservation | Automatic: import opens the original encrypted source read-only and leaves it unchanged. No extra preservation dialog. |
| Home/search | Total count, initially empty results, name/URL matches after three characters. Metadata listing does not prefetch passwords. |
| Add | Original four fields and Save/Cancel. Transactional save; Cancel/Escape discards draft. Starting Add resets search. |
| Reveal/hide/copy | Explicit read/copy only. Search/lock/transitions invalidate late reveal delivery; clipboard clearing remains ownership-checked. |
| Edit | Metadata prefill; retain password unless explicitly replaced. Cancel discards draft. Preserve transaction/revision checks, including empty replacement. |
| Delete | Explicit confirmation and transaction/revision check. Cancel keeps entry. |
| Encrypted backup | Explicit fresh destination; verified encrypted snapshot. Cancel has no error or settings change. |
| Change master password | Wrong current password only says “Wrong password”; no backup/rotation starts. Success keeps default path and retains verified old-password encrypted backup. |
| Inactivity preference | Validate/save 1/5/15 minutes and restore for next unlocked session. |
| Manual/idle/system lock or sleep | Invalidate generation, scrub plaintext/drafts, close handle, clear owned clipboard. Keep non-secret remembered path; relogin needs no picker. |
| Lock/repeated submission during work | One UI operation and one backend expensive/picker operation at a time. Reject duplicates rather than queue. Generations/nonces discard stale completions/grants; lock admission remains independent. |
| Service startup failure / early menu | Show connection error; lightweight status recovery restores correct first screen. Do not launch pickers before initial native state is known. |
| Quit/relaunch | Async native shutdown closes storage without waiting for AppKit cleanup on its main thread. Relaunch reads settings; no password/plaintext restoration. |

## Verification

`frontend/tests/flows.spec.ts` tests fresh creation without a picker, actual backend reconstruction preserving files/settings, ordinary login, wrong-key duplicates/correct retry, moved files, cancellations, verified import/restart, rejected rotation, startup recovery and existing-destination safety. Original CRUD/transfer/ordering tests and all four source-derived visual comparisons remain.

`internal/service/flows_test.go` reconstructs actual Manager/Facade instances and checks unchanged settings after wrong keys, stable grants, repeated source choice, settings-save failure and nonqueued work. `internal/settings/settings_test.go` checks private on-disk format, malformed/oversized/symlink rejection, atomic reads and failed-write preservation.

The native smoke runs two separate processes sharing only a temporary synthetic vault/settings directory. It requires first-run create/import UI, zero unexpected pickers, second-process remembered login, wrong-key error with a responsive form, correct retry and persisted entry/password. Both retain actual 904 × 534 frame, AppKit shortcut, WebKit, bridge and navigation assertions. New native evidence must come from this implementation; predecessor results do not establish restart behavior.

The trace establishes the erroneous picker fallback and missing persistence. It does not establish the cause of an entire macOS freeze on the owner's machine. Wrong-key authentication already precedes schema/integrity scans; the fix removes unexpected picker/password handoff and bounds overlapping work. Platform tests establish observed behavior within their deadlines, not an explanation of an uncollected operating-system hang.

Run `./scripts/verify-local.sh` for the complete locally executable checks, with no dependency on GitHub Actions. The script restores production assets after browser testing (also on failure), and on macOS executes the native two-process smoke, packaging, artifact checks and binary scan. Linux validation explicitly records that native macOS checks were not run. Evidence is saved under `artifacts/`.

Application source **`2af2b2d369d585b24465f1f73f2590957ceea626`** passed all **19 browser tests in 40.5 seconds**, all four legacy comparisons with zero pixels above their tolerance, three unit tests, the production frontend build, Go vet and the complete production-tagged Go race suite. The separate [flow review](FLOW-REVIEW.md) has no remaining source blockers and records its independent subset of checks. Subsequent local-runner/documentation edits do not change this application source.

The final local-command run also passed end to end: 19 browser tests in **42.2 seconds**, three unit tests, production race/vet/build and production-asset restoration. Its timestamped [manifest](evidence/flows-2026-11-09.json) and [retained local logs](evidence/local-validation-2026-11-09.txt) record the Linux platform, unchanged application source and explicit absence of native macOS execution. The script invalidates old success manifests at startup; negative checks verified that failed reruns preserve failure status and restore production assets instead of leaving stale success evidence.

The owner confirmed on 2026-11-09 that GitHub Actions free minutes are exhausted and instructed that no validation should rely on CI. CI retries/polling have stopped; no new CI outcome is claimed. This workspace is Linux, so the new native macOS restart test is available in the local command but has not executed here. The reported whole-system macOS hang has not been independently reproduced or explained. Earlier native evidence remains historical for its named sources.
