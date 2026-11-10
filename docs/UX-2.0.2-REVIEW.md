# PassV2 2.0.2 adversarial review

Reviewer: separate agent `/root/implementation_adversary`, 2026-11-09 (America/New_York).

Scope: V1 detection and explicit migration consent, import results and source preservation, friendly errors and path labels, confirmation validation, visible Lock and tooltip behavior, the fresh PassV2 settings namespace, and application/version packaging. The earlier [implementation](IMPLEMENTATION-REVIEW.md), [UI](UI-REVIEW.md) and [flow](FLOW-REVIEW.md) reviews remain separate history.

**Source review complete: no unresolved blocking code finding.** The two findings below are corrected in the reviewed working tree following `eceddcbba2ff14a839fc7025e1c3e2dd69a540b2`. The complete local workflow passed. Native macOS execution is not established by this Linux review, and verification does not rely on CI.

## Findings and dispositions

### UX202-01 — Medium / resolved: V2-only password/permission checks prevented V1 detection

The first implementation checked normal-vault private permissions and reserved raw-key password syntax before classifying the file. A legitimate legacy file created with mode `0644` therefore reported a permission failure instead of offering migration. A historical raw-key source was likewise rejected by the new-password policy before read-only legacy identification.

Correction reviewed: input length/path/ownership checks and the authenticated read-only classification precede V2-only private-mode and passphrase-policy checks. Validated legacy files return the migration-required result without a writable open. Normal V2 sessions and every fresh destination still enforce private permissions and reject SQLCipher's reserved raw-key forms. Regression tests cover both legacy cipher profiles, public source permissions, URI-special/Unicode filenames, the actual historical-driver fixture, all supported raw-key forms, unchanged source bytes/mode/mtime and private migrated destinations.

### UX202-02 — Medium / resolved: login autofocus could escape the migration-consent dialog

The existing login autofocus effect ran whenever busy state cleared while locked. The new V1 error mounts a consent dialog at that same transition, allowing the effect to refocus the underlying password input. That bypassed the dialog's keyboard focus loop and could let Enter submit another Unlock behind the dialog.

Correction reviewed: login autofocus excludes the active legacy offer and tracks its presence. The retained browser regression requires initial focus inside the V1 dialog, uses Enter to close its focused control, verifies no extra Unlock, and then explicitly accepts migration. Migration preparation clears the login password before presenting import fields.

## Storage and classification assessment

- `OpenImmutable` constructs an escaped native `file:` URI internally with read-only/immutable flags. Renderer-supplied filenames or URI parameters are not accepted. SQLCipher version/status/effective profile checks remain on every handle.
- V1 is identified by its validated legacy schema and `user_version=0`, independently of SQLCipher compatibility profile. Both profiles 4 and 3 are tried read-only; the real historical-driver fixture is profile 4. A wrong key or unknown/malformed schema is not classified as known V1.
- Before opening an original file writable, V2 classification requires authenticated profile-4 metadata, `user_version=1`, the exact three canonical table definitions and the valid format/UUID row. It deliberately does not validate potentially uncommitted data rows through the immutable handle. Ordinary SQLite then performs recovery, followed by the existing complete schema, row and integrity checks.
- This preserves SQLite-owned recovery for the real killed-edit regression without repairing an unrecognized legacy source. Dirty V1 input is refused. Tests compare both original database and real hot-journal bytes after V1 detection. If damaged main-file metadata cannot positively identify V2 even though a journal might repair it, the operation fails closed and requires recovery guidance/a clean backup; no writable fallback guesses the format. No custom journal replay, checkpoints or resumable migration were added.
- Import still reads the source, writes a fresh encrypted candidate, reopens it and compares every entry/field before publication and success. Removing the manual preservation action does not remove source preservation: the original remains unchanged. The removed facade method is no longer in the native allowlist; explicit encrypted backup remains available for normal sessions.
- `PrepareImport` converts only an already-authorized, current-generation opaque Open selection into a source grant after the user's consent. It accepts no arbitrary path or password and opens no picker. Invalid/stale tokens are rejected. Full path labels are intentional unencrypted metadata, rendered as React text rather than HTML or executable file/URL links.

## UI and version assessment

- Import stays on the same centered screen after verified completion, including zero-entry success. Password fields clear, repeat submission/path changes are disabled, and “Imported N secrets” remains until an explicit action. Open storage uses the known destination grant to show its login; it neither opens a picker nor authenticates automatically.
- Import/result and consent preparation retain session fences and generation checks. Lock scrubs offers, results, passwords, entry drafts and revealed values. Delayed successful imports and delayed V1 errors cannot restore their old UI after Lock. The visible Home Lock control uses the same lock handshake and remains available during ordinary work.
- Confirmation fields are immediately visible and matching validation updates when either value changes. Mismatch prevents Create/Import/Rotate backend submission. Missing files and permission failures show friendly text and explicit recovery actions; an error never automatically launches a recovery picker or migration.
- The owner explicitly requested literal **“Wrong password”**. The frontend retains that wording. SQLCipher's first-page authentication failure also covers an unsupported profile or damaged first page; this label is not cryptographic proof that the password itself is wrong.
- The settings directory is now `PassV2`. The owner's subsequent instruction explicitly removed compatibility with the previous settings namespace from scope. There is no settings fallback/migration, relocation of an old vault or overwrite of old preferences; the established private atomic settings store remains unchanged apart from the namespace.
- `VERSION` is `2.0.2`, with synchronized frontend package/lock versions. The plist generator supplies that value to both native version fields; local verification checks metadata generation, and packaging uses that generator. The pinned macOS About role calls `orderFrontStandardAboutPanel:`, which reads packaged bundle metadata. This is source/metadata verification, not an observed native About-panel pass.
- Repository instructions and the isolated bump utility encode the owner's future patch/minor policy without changing major automatically. The explicit `2.0.2` request governs this batch.
- The new Home Lock control is an intentional user-requested addition to the historical UI. The legacy comparison masks only that control using the existing background color; it preserves the rest of the geometry/style assertions and unchanged pixel tolerance. Delete-dialog close tooltips are placed inward and have a retained bounds assertion.

## Verification

The reviewer independently ran an uncached Go race subset in **11.085 s**, covering both-profile V1 detection, public permissions/special filenames, the historical fixture, malformed/dirty/damaged classification, current raw-key refusal, six legacy raw-key conversions, and real killed-edit V2 recovery. Additional uncached race checks passed for the real killed legacy journal (**1.821 s**) and invalid/stale/full-path migration preparation grants (**1.021 s**).

The final end-to-end `verify-local.sh` logs and result manifest were independently inspected. The Linux run completed on 2026-11-10 **02:45:03–02:47:10 UTC**: all **28 browser tests passed** (Playwright reports **1.1 min**), all three unit tests passed, production-tagged race packages passed (vault **41.776 s**, service **8.306 s**, settings **1.052 s**), vet passed and production assets were restored. Four visual metrics were zero above the unchanged threshold, with only the explicitly added Home Lock control masked. Generated plist metadata reports `2.0.2` in both version fields and `PassV2` for the application/display name.

The manifest records parent HEAD `eceddcbba2ff14a839fc7025e1c3e2dd69a540b2` with `workingTreeDirty:true`, accurately identifying that this run exercised the uncommitted application/version changes under review. It also records `status:passed`, `usesGitHubActions:false` and `nativeMacOSChecksExecuted:false`. The implementation agent's earlier isolated nine-case browser run passed in **18.3 s**. Those new regressions cover centered import/full paths, immediate validation, consent focus and source hash, friendly recovery controls, zero/nonzero import results, known-destination Open, visible lock/tooltip bounds and delayed result/consent rejection. No running browser/build was disturbed by this review.

The native smoke source was reviewed: its first process creates/edits/locks, imports through exactly two explicitly requested pickers, checks the same-screen imported count, verifies the source hash and opens the imported destination. The second process uses the remembered imported path and key, rejects a wrong password, retries successfully and requires zero pickers. Both retain explicit 904×534 frame, physical key, bridge and navigation assertions. Positive completion markers and persisted settings are required by the shell harness; a clean exit alone does not establish a pass. This updated native smoke has not executed in the Linux workspace.

No real vaults or passwords were used. No CI result, commissioned human audit, notarization or future-beta compatibility is claimed.
