# PassV2 2.0.2

This batch implements the owner's current UI/error/version requests on `implementation/option-d`. The legacy theme, typography, 904 × 534 outer frame and Command-tilde shortcut remain. These changes supersede older screenshot/flow descriptions where they differ.

| Request | Result |
|---|---|
| Center import vertically | The form is centered in the content viewport, with scrolling when it exceeds the available height. |
| Preserve original automatically | The extra Preserve original backup action and facade method are removed. Migration always reads the original encrypted file without changing it and publishes a verified fresh destination. |
| Clear import feedback | Success stays on import as “Imported N secrets”, including a valid zero-entry import. Failure also stays on that screen. Success clears passwords/disables repeat import and offers Open storage for the known destination without another picker. The result clears before login. |
| Destination wording | Import shows **Import from** and **Backup to**, with complete native paths. Other selection/status/rollback path labels also remain complete plain text. |
| Friendly errors/recovery | Missing storage says “Storage file not found”. Permission/unreadable/unsupported errors offer explicit Open/Import actions. Wrong-key authentication says “Wrong password”, without an automatic picker or retry. |
| Known V1 storage | Read-only authentication/schema validation identifies V1 before a writable open. Explicit consent prepares the same source on import without retaining the password or opening a source picker. Unknown/malformed schemas are not called V1. |
| New/confirm fields | Creation/import display both fields immediately. Matching validation updates when either changes and blocks backend submission on mismatch. Rotation uses the same matching rule. |
| Delete tooltip | The close tooltip points inward and is tested to remain within the dialog bounds. |
| Visible lock | The icon is right-aligned opposite the count/FAB group and shares the lock handshake, plaintext/draft scrubbing and shortcut. |
| Name/settings/version | App/settings namespace is PassV2: `~/Library/Application Support/PassV2/settings.json`. Per the owner's follow-up, old settings are not migrated. Existing storage/old settings are not moved or deleted. About/package metadata is generated from `VERSION`, now **2.0.2**. |
| Future versions | Fix batches increment patch; features increment minor/reset patch. The helper cannot change major; repository instructions require explicit owner direction for a major change. |

## Storage guarantees retained

The encrypted schema/format and every-entry verification remain compatible with previous Option D storage. App version 2.0.2 does not change the internal format marker. V2 opens retain private-permission/password-KDF policy; historical V1 public-permission/raw-key sources can be identified read-only and imported into private destinations.

Identification uses an internally constructed, escaped read-only immutable SQLite URI. Authenticated current-format metadata permits the normal writable handle to let SQLite recover a killed V2 edit, followed by full schema/data/integrity validation. Dirty V1 input and unclassifiable damaged metadata fail closed without recovering/changing that source. No custom journal replay, checkpoints or writable guessing were added.

SQLCipher cannot always distinguish an incorrect key from damaged first-page authentication or unsupported encryption. “Wrong password” is the UI wording explicitly requested by the owner, rather than a stronger diagnostic claim.

## Local verification

Tested application source: **`fc9b4aee54395faa262427783a8afd9eb6de2cef`**. The local command ran against the working tree subsequently committed unchanged; its retained manifest correctly records the prior HEAD and `workingTreeDirty: true`. See the [evidence](evidence/ux-2.0.2.json) and [retained local logs](evidence/local-ux-2.0.2.txt).

`./scripts/verify-local.sh` passed end to end on Linux: **28 browser tests**, three frontend unit tests, production build/asset restoration, Go vet and the full production-tagged race suite (vault 41.776 s, service 8.306 s, settings 1.052 s). Four source-derived legacy comparisons report zero pixels above the unchanged threshold, masking only the explicitly added Home Lock control. Other geometry/style/pixel thresholds are unchanged.

New regressions cover centered import/full paths; unchanged-source preservation; live confirmation; consent focus/agreement; invalid/stale grants; missing/permission recovery; zero/nonzero same-screen results and no-picker Open; tooltip/lock bounds; delayed-result cancellation; both legacy profiles, historical/raw-key fixtures, unknown schemas and URI-special filenames. Source hash/mode/mtime and database/journal bytes are checked where applicable. Real killed V1 transactions remain untouched, while a real killed V2 edit is recovered by SQLite.

The generated plist was inspected on Linux: both version properties are 2.0.2 and the name is PassV2. Isolated patch/minor checks produce 2.0.3/2.1.0; major is refused. The [independent review](UX-2.0.2-REVIEW.md) records two corrected findings and no remaining source blockers.

The native local command now tests the Go/WebKit import count/result, unchanged source hash and no-picker Open, followed by a second process using the remembered imported file. Native frame/physical-shortcut/clipboard/bridge checks remain. That updated macOS smoke and the actual About panel were not executed on this Linux host. CI is not used or relied upon.
