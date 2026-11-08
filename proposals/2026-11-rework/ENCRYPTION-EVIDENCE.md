# Encryption evidence and acceptance gates

Status: required evidence plan; **no new implementation has passed these tests**. “Solid proof” means a traceable design, proven primitives, exact binary/dependency identification, reproducible positive and negative tests, and independent review. Finite tests or a vendor's algorithm choice cannot mathematically prove an entire app secure. The proposal reviews are agent design critiques, not independent cryptographic audits.

## Security claim and limits

Protect a stolen, locked vault and its temporary persistent artifacts against content disclosure and unauthorized modification without the key, assuming an adequate master password, uncompromised device/software and correctly used crypto. Password guessing remains possible offline. File size, timestamps and storage-format metadata may leak; KDBX includes a public header. Per-page authentication does not provide freshness or prevent valid page/file replay. Neither format by itself provides rollback protection without trusted state outside the file. No claim of secure deletion from APFS snapshots, swap, system backups or another device; no complete heap-erasure promise.

## A/B: why SQLCipher 4

The [upstream storage design](https://www.zetetic.net/sqlcipher/design/) specifies AES-256-CBC with fresh page IVs, separately keyed HMAC-SHA512 page authentication, and salted PBKDF2-HMAC-SHA512 (default 256,000 iterations). This is a documented construction, not an app audit. Keep the supported default cipher profile for interoperability; benchmark unlocking on the oldest target machine. Do not implement an ad-hoc Argon2 wrapper or keep legacy profile 3 for new writes. Reconsider stronger KDF requirements as a separate reviewed format decision.

Use the [documented API](https://www.zetetic.net/sqlcipher/sqlcipher-api/) to assert cipher identity/settings, key connections via the C API, and perform both page authentication and logical DB integrity checks. Ordinary SQLite accepts unknown pragmas silently; a successful `PRAGMA key` call is no evidence that encryption is active. Require a nonempty expected cipher version, supported provider/settings and active-encryption status plus actual reopen/tamper tests. Missing/unknown results are failures. The storage boundary must never fall back to plain SQLite after errors.

## C: why KDBX 4

The [format specification](https://keepass.info/help/kb/kdbx.html) defines separate header/data authentication and encrypted contents. Select AES-256 + Argon2id; bound untrusted KDF parameters before allocating memory. A public header checksum is not keyed authentication. Validate the chosen writer and reader implementation, not merely the format. Match independent KeePassXC [review/certification scope](https://keepassxc.org/audits/) to actual versions/platforms; do not claim that Windows certification extends to a new macOS release or importer.

## Required reproducible evidence

| ID | Test / evidence | Required result |
|---|---|---|
| E1 | Source tags/commit hashes, dependency lockfiles, SBOM, crypto provider, compiler/build flags, architecture, linkage and artifact hash | Identify exactly which engine each shipped artifact uses; prove plain SQLite substitution is rejected before sensitive writes |
| E2 | Synthetic sentinel vault using real shipping storage code; correct password reopen after create/edit/delete/rotate and process restart | Exact expected logical state; no test-only encryption path |
| E3 | Wrong/empty/truncated/Unicode/quote-containing passwords, missing key, engine compiled without cipher | Fail closed, original unchanged, no plaintext fallback; unusual valid passwords work byte-exactly |
| E4 | Flip a byte in known used encrypted payload/HMAC areas of first and later pages/blocks; truncate, reorder, corrupt header/salt and test structured malformed files | Open/full-integrity validation fails safely, bounded resource use; file replacement by a valid old copy is documented as undetectable |
| E5 | Examine vault, live journal, save temp files, logs, caches, backup, UI restoration/autosave, crash artifacts and watched filesystem writes for synthetic secrets; plain SQLite must not read SQLCipher schema | No plaintext sentinels in persistent app-created artifacts; scanning alone is supplementary, not proof of cryptography |
| E6 | Kill process / simulate disk-full, permission errors and interrupted save/migration/rotation at each durable-write boundary | Either prior valid vault or complete new vault; no false success, silent data loss or plaintext recovery output |
| E7 | Synthetic fixtures from historic app path and explicit SQLCipher 3/4 profiles; every migration-field comparison, duplicate/null/Unicode/NUL/control-character/long fields, >2^53 integers, timestamps, sequence behavior | No silent loss; unsupported destination strings are rejected; original remains unchanged. C additionally cross-opens in KeePassXC, checks encrypted custom metadata, and repeats native edit/save/reopen fidelity checks |
| E8 | Multiple processes/alias paths, simultaneous edits, lock during KDF/integrity check/reveal/copy/save, external file replacement, malformed vault and path/symlink inputs | No late secret response after lock, stale overwrite, unsafe path write or silent corrupt-file repair |
| E9 | Release binaries on explicitly declared minimum and rolling macOS/CPU matrix, sleep/wake, screen lock, clipboard timeout, signing/notarization and restore tests | Passing results tied to artifact hashes; clean-account file-only restore; beta failures tracked separately and support claims limited accordingly |
| E10 | Independent implementation security review; published findings and remediation evidence | Check KDF/key handling, imports, memory/log boundaries, filesystem safety, distribution/update chain; resolve blocking findings before real-vault use |

Keep only synthetic vault fixtures in git. Fixture passwords must be explicitly public and synthetic. Never ask users to upload a real vault or commit its contents. Reports may record tool versions, test IDs and artifact hashes; never persist record values or master passwords. Real-data migration validation stays local and reports only pass/fail/counts.

## Evidence not yet available

No real legacy fixture, original released binary analysis, migration prototype, packaged app, macOS runtime check, dependency audit or new cryptographic audit was performed for this proposal. These are implementation gates, not passed checks. Current work establishes source-derived feasibility and a reviewable validation contract. If the owner requires completed storage proof before choosing a UI, the first implementation milestone should be a shared SQLCipher migration/evidence spike (A/B) and a KDBX importer/evidence spike (C), on separate non-main branches.
