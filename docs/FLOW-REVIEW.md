# Startup, authentication and persistence adversarial review

Reviewer: separate agent `/root/implementation_adversary`, 2026-11-09 (UTC).

Scope: the startup/path regression reported after `a701182`, its settings and facade changes, login/picker ordering, cancellation, restart and interaction with existing storage/session operations. The earlier [13-finding implementation review](IMPLEMENTATION-REVIEW.md) and [UI review](UI-REVIEW.md) remain preserved as separate history.

**Source review complete: no unresolved blocking code finding.** Final reviewed application source is `2af2b2d369d585b24465f1f73f2590957ceea626`. Local flow tests passed independently, and the final complete local browser suite passed. Verification uses the local workflow and does not depend on GitHub Actions. The new two-process native macOS restart smoke is reviewed source but has not executed in this Linux workspace; previous native passes do not validate this feature.

## Findings and dispositions

### FLOW-01 — Medium / resolved: the selected storage path was not persisted

At `a701182`, neither the service nor manager loads or saves a selected vault path. The initial locked screen therefore displays login without any selected vault. Lock scrubs the selection, and a successful import also discards its verified destination.

The pinned legacy source routes to signup when `electron-settings` has no `default_storage_path`, otherwise to login. Signup saves that setting after successful creation. Login loads the saved path and validates a key against that path without opening a file picker.

Correction reviewed: backend-owned preferences load once and remember the path only after successful creation, successful authentication of an explicitly opened vault, or verified import. Locked State supplies stable opaque grants for the remembered path and a default fresh destination. First launch offers create/import/open; subsequent launches and locks restore the remembered login. Import returns to login for its new destination. Password rotation retains the same path.

Picker cancellation, a rejected key, a missing file and a failed import do not replace the previous saved choice. If saving preferences fails after successful creation, the creation result remains successful and State carries a visible settings warning; the UI does not misleadingly invite another Create against an already-created file.

### FLOW-02 — Medium / resolved: password submission could invoke the native picker

The old frontend used `selection ?? await Choose(...)` inside authentication. Submitting a password without an existing selection therefore opened a native picker before validating that key. Existing browser helpers always invoked File → Open first, so this ordinary login path was uncovered.

Correction reviewed: authentication requires an existing opaque selection and calls only Create or Unlock. The picker is reached through explicit path/menu actions. Those storage commands also wait until initial State exists, preventing an early native menu action from racing bootstrap restoration. A rejected key keeps the login and selected vault, reports the error, and permits a correct retry without opening another dialog. The same error mapping applies to a rejected import key or current rotation password without scheduling another operation.

Repeated submission is guarded synchronously in the frontend. The service uses a nonblocking work gate for pickers, authentication, backup, preservation, import and rotation, so duplicate expensive operations are rejected rather than queued. Lock bypasses that gate, retaining its generation invalidation and priority admission.

### FLOW-03 — Low / resolved: a failed local verification rerun could retain stale success evidence

The initial local verification script wrote its result manifest only after success but left an earlier successful manifest present when a subsequent run failed. Correction reviewed: startup removes the previous manifest; only a completed successful run writes a new `status:passed` record with UTC start/completion timestamps. An independent isolated copy of the actual script, with synthetic command stubs forcing the browser gate to fail, preserved exit code 27 through the log pipeline, removed stale success, wrote no new success, and restored production assets exactly once. This failure harness tests script behavior, not crypto or native execution.

## Security and lifecycle assessment

- The separate JSON preferences contain only `default_storage_path` and `idle_minutes`. They contain no master password, derived key or entry values. The absolute path is unencrypted local metadata; the vault remains the encrypted portable storage file.
- The settings directory is owner-private and its opened root identity is checked. Settings reads reject a final symlink, nonregular/insecure file, oversized content, unknown fields, invalid values and trailing JSON. Writes use a fresh private temporary file, file sync, atomic replacement and directory sync. A save never opens and writes through the existing settings filename.
- The renderer receives basename labels and random native-owned grants, not an arbitrary-path setter. Remembered paths still pass the existing vault ownership, identity, permission, locking, schema and cipher checks at Unlock. A remembered path is not installed into the manager as an open vault.
- State caches grants per generation, so polling does not exhaust or invalidate a login token. Picker replacement prunes only the same purpose while preserving remembered/default grants; repeatedly changing an import source does not revoke its destination. Stale-generation grants remain unusable, and State rejects a changed generation before mutating its cache.
- Lock, idle/system locking and terminal Close retain manager generation admission. Plaintext entries, reveal results and credential drafts are scrubbed while nonsecret remembered-path preferences survive. The existing session fence, reveal nonce and exact lock-handshake generation checks remain in place.
- Wrong-key SQLCipher validation fails at the first authenticated schema read, before full schema/all-entry and cipher/SQLite integrity scans. Correct authentication performs those full checks. The service runs through Wails worker execution; native dialogs dispatch through AppKit. Source inspection does not establish the cause of the originally reported macOS hang. PBKDF2/full validation can take time, and initial authentication's unpublished native handle is not immediately interruptible; generation admission prevents a concurrent Lock from subsequently reopening it.
- CRUD still uses the same standard SQLite transactions and revision checks. Encrypted snapshot backup, untouched original preservation, restartable migration and password rotation retain their existing storage behavior; preferences do not add custom vault recovery or migration checkpoints. Synthetic HTTP/restart controls and native smoke routes remain excluded from the release binary.

## Flow coverage

| Flow | Reviewed behavior and retained verification |
|---|---|
| Startup/service failure | No unconfigured login while State is unavailable; visible connection error and recovery to the appropriate form. |
| First create | Private default destination without a picker; password confirmation, explicit alternate path selection and no-overwrite storage rules. |
| Login/retry | Remembered grant, one attempt while busy, wrong-key error only, preserved login/choice and correct retry. |
| Explicit Open | Picker selection alone changes no preferences; successful authentication remembers the new path; cancellation returns to the prior choice. |
| Lock/restart | Credentials and reveal state clear; reconstructing manager/facade from the same real settings file restores login and the idle preference. |
| Missing/moved vault | Error on the same login with no automatic picker; an explicit Open can select and remember its moved location. |
| Create/edit/delete/reveal/copy | Existing encrypted integration/session tests remain; cancelled edit/delete leaves stored values intact. |
| Backup/preservation | Explicit native selection; cancellation is not a success or error; consistent encrypted destination/verification behavior remains. |
| Import | Bad key leaves settings unchanged; successful full verification remembers the destination and permits login before and after restart. |
| Rotation/settings | Wrong current password leaves the session/preferences intact; rotation retains path; selected idle duration survives restart. |
| Concurrent/stale work | Duplicate heavy work rejected; Lock cancels a pending picker grant; earlier menu/status/reveal regressions remain retained. |

## Independently executed checks

The reviewer ran uncached `go test -count=1 -race ./internal/settings ./internal/service` and reran after the final grant/root/admission changes: both packages passed (latest settings **1.045 s**, service **5.592 s**). These checks cover private settings/atomic read visibility, malformed and symlink rejection, real settings reload, stable grants across polling and repeated source selection, successful-only remembering, wrong-key retries, verified import, cancellation, save failure and Lock during a pending picker.

The reviewer then built the e2e assets and ran the seven-case version of `frontend/tests/flows.spec.ts` against the actual Go/SQLCipher synthetic server with Playwright Chromium headless build 1194: **7 passed in 16.1 s**. The rejected-key test observes exactly one Unlock and no other operation apart from ordinary State polling, checks unchanged picker/settings statistics beyond a polling interval, and then authenticates correctly. Browser restart reconstructs the manager and facade while retaining encrypted files and the real settings file, rather than merely reloading the page. Production frontend assets were restored afterward. The final bootstrap-command guard and additional mismatch/existing-destination test were subsequently source-reviewed; the final complete browser/native results must identify their tested source separately.

An initial reviewer invocation accidentally used production frontend assets in the browser harness and therefore could not obtain the native Wails capability. That setup error was identified, the assets were rebuilt in e2e mode, and the complete seven-test rerun above passed. It is not presented as an application failure or native evidence.

## Final-source validation and native gap

Source `2af2b2d369d585b24465f1f73f2590957ceea626` has recorded local passes for all **19 browser tests in 40.5 s**, including the bootstrap-command guard and mismatch/existing-destination regression added after the reviewer's seven-case run. All four source-derived legacy visual comparisons recorded zero pixels above the unchanged threshold. Its complete production-tagged Go race run passed (vault **28.640 s**, service **4.463 s**, settings **1.044 s**), along with Go vet, the production frontend build and all three unit tests. These are local results; the independently executed subset and source-review scope are stated above.

The owner confirmed on 2026-11-09 that monthly free GitHub Actions minutes were exhausted and explicitly instructed, “Don't rely on CI for anything.” CI polling/retries stopped. Earlier independent inspection had found jobs failing before runner assignment with zero steps and unavailable logs; those attempts supply no application/native test evidence. The quota explanation is owner-confirmed, rather than inferred from the API response. No new CI outcome is required or claimed.

The reviewer audited `scripts/verify-local.sh`: it runs the pinned SQLCipher preparation, uncached production-tagged race suite, vet, dependency installation, production build, unit tests and the complete browser suite locally. `set -euo pipefail` preserves failed gate status through log pipelines. An EXIT trap restores production assets if browser setup/testing fails. Native WebKit restart smoke, packaging, signature/encryption linkage checks and packaged-binary vulnerability scanning run only on Darwin, and the result manifest explicitly records whether native checks executed.

The completed final Linux local-workflow logs and manifest were independently inspected (2026-11-09 **15:21:04–15:22:31 UTC**): **19 browser tests passed in 42.2 s**, all three unit tests passed, Go race packages passed (vault **28.968 s**, service **5.880 s**, settings **1.040 s**), and production assets were restored. Its manifest names application HEAD `2af2b2d`, explicitly records a dirty working tree for the subsequent verification-script/documentation edits, `status:passed`, UTC timestamps, `usesGitHubActions:false` and `nativeMacOSChecksExecuted:false`. This is local execution evidence, with no attribution of Linux browser results to macOS.

The reviewed native smoke launches two actual application processes against one synthetic settings directory. It requires first-launch creation, remembered login after process restart, wrong-key indication, correct retry, zero picker calls, persisted secret values, the 904×534 frame and physical Command-tilde behavior before reporting success. The new native restart behavior remains unvalidated on macOS until an exact-source run executes. No native success, commissioned audit, notarization or beta compatibility is inferred from local Chromium tests.

No real vaults or passwords were used.
