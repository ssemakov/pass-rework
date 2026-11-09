# Startup, settings and complete flow trace

Reference: legacy Pass at `7cd8c6c0bc395fa3676c151a84c1657cda8cb481`. `AuthNavigation.js` checks electron-settings `default_storage_path`: absent → signup; present → login. `Login.jsx` loads that path and validates the submitted password without opening a picker. `actions/storage.js` writes the path only after successful creation. These transitions are part of the reference alongside its screenshots.

The earlier rework preserved the screen styling but omitted this persistence. Its password-submit handler called `Choose` when no selection existed, and locks/import discarded that selection. Earlier tests explicitly invoked File → Open and reset the backend/files between tests, missing remembered login and process restart. Those gaps are now retained as regressions.

## Preferences

The macOS settings file is `~/Library/Application Support/Pass Rework/settings.json`; the initial vault destination is `storage.pass` in that directory. Directory/file permissions are `0700`/`0600`. Only `default_storage_path` and `idle_minutes` are saved. Passwords, keys and entries are never written there; the vault remains a separate encrypted portable file. Legacy settings and encrypted originals are untouched.

The backend loads settings once per instance and exposes opaque generation-scoped selection tokens, without a frontend method accepting arbitrary paths. Remembered/default grants are stable under status polling, renewed after lock, and retained when repeatedly choosing other files for import. A successfully created/opened vault or completely verified import becomes the default; choosing/cancelling/rejecting a key/failing an import does not change it.

Settings use bounded no-follow parsing and an atomic synced replacement. Ordinary vault edit/delete recovery remains SQLite's responsibility. A settings-save failure leaves an actually created/opened vault usable and clearly says to select it again after restarting. Unreadable/malformed settings show an error and explicit Open/Create/Import recovery. Existing vaults are never silently overwritten.

## Flow inventory

| Trigger | Result / retained checks |
|---|---|
| No saved path | Create Storage with the original two-step confirmation and folder control, plus Import/Open alternatives. No automatic picker or database read. |
| Matching creation password | Create a fresh encrypted destination, verify/open it, save its path. Mismatch only says “Passwords don't match”; existing destination is refused. |
| Restart with saved path | Preselected login asking only for the master password. Passwords and plaintext views are not restored. |
| Rejected login key | One SQLCipher authentication attempt and “That didn't work”. Same login/selection supports a correct retry. No Choose/Create/Import/retry or full-vault scan follows rejection. |
| Missing/moved/unavailable file | Error on the same login; retain the saved choice. Explicit File → Open can locate it. Remember the new path only after successful authentication. |
| File → Open | Clear old password drafts; lock/acknowledge any active session; open picker. Cancel returns to remembered login, or first-run create if no path exists. Never submit the old draft to the chosen file. |
| File → New | Lock the old session; show creation form. Destination must be fresh. Escape returns to remembered login; prior setting remains until successful creation. |
| Import / restore | Explicit encrypted source and fresh destination; validate passwords; migrate, reopen and compare every entry. Save verified destination and return to its login. Failure/cancel preserves previous setting and source. Interrupted staging is discarded/restarted as before. |
| Preserve original backup | Explicit source/fresh backup pickers and password/byte verification. No change to default vault; wrong-key verification has no subsequent automatic action. |
| Home/search | Total count, initially empty results, name/URL matches after three characters. Metadata listing does not prefetch passwords. |
| Add | Original four fields and Save/Cancel. Transactional save; Cancel/Escape discards draft. Starting Add resets search. |
| Reveal/hide/copy | Explicit read/copy only. Search/lock/transitions invalidate late reveal delivery; clipboard clearing remains ownership-checked. |
| Edit | Metadata prefill; retain password unless explicitly replaced. Cancel discards draft. Preserve transaction/revision checks, including empty replacement. |
| Delete | Explicit confirmation and transaction/revision check. Cancel keeps entry. |
| Encrypted backup | Explicit fresh destination; verified encrypted snapshot. Cancel has no error or settings change. |
| Change master password | Wrong current password only says “That didn't work”; no backup/rotation starts. Success keeps default path and retains verified old-password encrypted backup. |
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

Results and tested commit are recorded in the final validation update; earlier evidence remains historical for its named sources.
