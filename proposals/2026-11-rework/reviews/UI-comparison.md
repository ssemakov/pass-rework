# Adversarial review — UI comparison and discussed scope

Reviewer: separate adversary agent `/root/adversary_ui`, 2026-11-08.

Scope: expanded local proposal draft adding Option D and UI-COMPARISON, based on branch head `40abeca045b236de98e6d0061de2a1b13dd6ebd9`. Reviewed UI-COMPARISON, OPTIONS, DISCUSSION-DECISIONS and ENCRYPTION-EVIDENCE. This is a document/design review. No released legacy binary, proposed UI, migration or macOS runtime test was executed; no accessibility or performance measurement is claimed.

## Verdict

**Suitable for comparing the four directions, with the following clarifications before implementing their workflow contracts.** The comparison correctly separates storage compatibility from UI fidelity, makes B/D's possible React parity explicit, and treats native feel/accessibility as assessment and acceptance work rather than proven outcomes. It does not pretend that a Go backend makes native HTML controls or that adopting KeePassXC preserves Pass's layout.

The user decisions are carried into the current documents: A targets macOS 15 and 26; B/C/D use that matrix as a comparison assumption; external paid human review is omitted from v1; encryption/migration tests and agent review remain. No additional approval or external audit is required by this review.

## Findings and dispositions required

### UI-M1 — Medium: distinguish imported metadata from live field state in C

The migration stores original millisecond timestamps and null flags in custom metadata, while the UI contract expects native KeePassXC edit/save/reopen behavior. An ordinary KeePassXC edit does not know the semantics of project-specific legacy metadata. After a native field changes, an original null flag or original `updated_at` value can be stale even when it remains useful as provenance. This is different from secret duplication, which the current proposal already prohibits.

Required clarification: label the typed metadata as an immutable migration snapshot/provenance record, not the current semantic state of every later native field. Native KeePassXC fields/timestamps govern subsequent edits. Verification must distinguish exact migration fidelity from deliberate later mutation. If bidirectional export ever uses this metadata, it must detect divergence or define a new mapping rather than restoring an old null/timestamp over a later edit. No reverse-export feature is needed for v1.

### UI-M2 — Medium: define unchanged, blank, null and masked-password edits

“Same four fields” plus masking does not define a safe edit payload. A form that sends empty strings for fields not fetched, or sends `*****` as a replacement password, can overwrite real data while every SQL transaction succeeds. Null-vs-empty preservation also needs a value/dirty-state model rather than just a visible text box.

Required clarification: use explicit per-field edit intent. An untouched password stays unchanged; a display mask is never a stored value. Fetch an existing password only for the selected entry when editing/revealing requires it, or offer a separate explicit replace-password action. Clearing a value must be deliberate, cancel must discard the draft, and untouched nullable fields must retain their original null/empty distinction. Add fixtures/tasks for each case. This is UI/API correctness, not a custom database recovery mechanism.

### UI-M3 — Medium: complete task parity for password change and backup/restore

The prose lists master-password change and encrypted backup/import, but the workflow table and task evaluation do not explicitly cover those full paths. A migration importer is not the same user operation as restoring a current-format backup. C's existing GUI can reduce implementation work, but its actual behavior/settings still need evaluation.

Required clarification: add change-password and backup/restore tasks for A/B/D and equivalent C checks. Include cancel/failure behavior, correct final-file reopen, rejected old password on the new rotated file, preserved old backups, copying only a closed/consistent vault, and clean-account restore with only file plus master password. Reuse the storage contract rather than inventing UI-only recovery steps.

### UI-L1 — Low: keep qualitative native/UI effort ranking visibly conditional

“Strongest native result” and relative effort are reasonable architectural judgments, not measured outcomes. The introduction already makes this limitation clear. Keep the conditional framing if summaries are copied into PRs/chat: A provides the best starting point for native controls, while B/D can preserve a web design more directly. A poorly implemented native UI can still be less usable or accessible than a well-implemented WebView UI. No numerical score or performance claim is justified yet.

### UI-L2 — Low: historical reports must not silently reintroduce superseded gates

Original A/B reports refer to older OS floors and A's E1–E10 gate includes an external review recommendation. Preserve those reports as historical assessments, but place a visible supersession pointer in the review index or report preamble. The current user decision and current E10 govern v1; their omission is not a failed acceptance check.

## Acceptance work retained

- One shared synthetic vault and task script across A/B/C/D, with exact pass/fail observations tied to client/build versions.
- Keyboard-only and VoiceOver completion of unlock, search, reveal/copy, create/edit/cancel/delete, password change and backup/restore; masked secrets must not be inadvertently announced or persisted.
- Dark mode, text scaling, resized windows, long/Unicode values, explicit null/empty/unchanged edits, and no accidental password replacement.
- Lock during reveal/edit/save/copy, stale-response rejection and no false successful save/migration message. The distinction between immediate visual concealment and completed backend lock remains visible in the contract.
- On B/D, use packaged WKWebView on macOS 15/26 rather than browser-dev-server tests alone; no claim that sharing React automatically proves equivalent native menus or lifecycle behavior.

These are implementation checks, not results already obtained. They preserve the agreed migration restart behavior and reliance on SQLite's standard transaction/journal mechanism for normal edits/deletes.
